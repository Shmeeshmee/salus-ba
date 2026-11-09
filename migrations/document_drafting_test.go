package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A template has an original language, a status (a draft until the clinic has reviewed its wording) and the
// LOINC document type of what it writes; a draft keeps the language, title and type it was drafted with.
func TestDocumentDrafting_TemplatesAndDraftsHaveTheirDefaults(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	var tpl, draft string
	if err := owner.QueryRow(t.Context(), `SELECT status || '/' || language || '/' || type_code FROM document_template WHERE id = $1`,
		c.templates[clinicA]).Scan(&tpl); err != nil || tpl != "active/bs/11488-4" {
		t.Errorf("template defaults = %q, %v; want active/bs/11488-4", tpl, err)
	}
	if err := owner.QueryRow(t.Context(), `SELECT language || '/' || title || '/' || type_code || '/' || (signed_at IS NULL)::text
		FROM document_draft WHERE encounter_id = $1`, c.encounters[c.a.aPast]).Scan(&draft); err != nil || draft != "bs//11488-4/true" {
		t.Errorf("draft defaults = %q, %v; want bs//11488-4/true", draft, err)
	}
}

// The checks: a known template status, a LOINC-shaped type code, a translation's texts as an object, and a
// signature complete or absent.
func TestDocumentDrafting_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	tpl, e := c.templates[clinicA], c.encounters[c.a.aPast]
	for _, tt := range []struct {
		name string
		stmt string
		args []any
	}{
		{"an unknown template status", `UPDATE document_template SET status = 'retired' WHERE id = $1`, []any{tpl}},
		{"a type code no LOINC code", `UPDATE document_template SET type_code = 'referral' WHERE id = $1`, []any{tpl}},
		{"translation texts no object", `INSERT INTO document_template_translation (template_id, language, texts) VALUES ($1, 'en', '[]')`, []any{tpl}},
		{"a signature without its document", `UPDATE document_draft SET signed_at = now() WHERE encounter_id = $1`, []any{e}},
		// The open-draft index counts a preliminary draft as open: a signed one must not stay preliminary.
		{"a signed preliminary draft", `UPDATE document_draft SET signed_at = now(), pdf_size = 1, document_reference_id = gen_random_uuid(),
			provenance_id = gen_random_uuid(), signer_role_id = (SELECT practitioner_role_id FROM encounter WHERE id = $1),
			pdf_kek_id = 'k1', pdf_sha1 = decode(repeat('00', 20), 'hex'), pdf_wrapped_key = '\x00'
			WHERE encounter_id = $1 AND status = 'preliminary'`, []any{e}},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, tt.args...); sqlState(err) != "23514" {
			t.Errorf("%s: %v, want check_violation (23514)", tt.name, err)
		}
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO document_template_translation (template_id, language, texts) VALUES ($1, 'xx', '{}')`, tpl); sqlState(err) != "22P02" {
		t.Errorf("a translation in no language of the enum: %v, want invalid_text_representation (22P02)", err)
	}
}

// An encounter has at most one open (preliminary) draft per template: a regeneration replaces its fields. A
// signed one does not count.
func TestDocumentDraft_OneOpenDraftPerEncounterAndTemplate(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	e, tpl := c.encounters[c.a.aPast], c.templates[clinicA]
	insert := `INSERT INTO document_draft (encounter_id, template_id, fields) VALUES ($1, $2, '[{"key":"a","missing":true}]')`
	if _, err := owner.Exec(t.Context(), insert, e, tpl); sqlState(err) != "23505" {
		t.Errorf("a second open draft: %v, want unique_violation (23505)", err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE document_draft SET status = 'final' WHERE encounter_id = $1`, e); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), insert, e, tpl); err != nil {
		t.Errorf("an open draft beside a signed one: %v, want success", err)
	}
}

// The app role signs a draft (the signature's columns), and edits a template's status, language and type and
// its translations, in the clinic's scope; a translation is seen only with its template.
func TestDocumentDrafting_TheAppRoleSignsAndTranslatesInTheClinicsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	var role string
	if err := owner.QueryRow(t.Context(), `SELECT practitioner_role_id::text FROM encounter WHERE id = $1`, c.encounters[c.a.aPast]).Scan(&role); err != nil {
		t.Fatal(err)
	}
	scopeA := map[string]string{"app.organization_id": clinicA}
	for _, stmt := range []string{
		`UPDATE document_draft SET status = 'final', language = 'hr', title = 'Placeholder title', type_code = '57133-1',
			signed_at = now(), document_reference_id = gen_random_uuid(), provenance_id = gen_random_uuid(),
			signer_role_id = '` + role + `', pdf_size = 10, pdf_kek_id = 'k1', pdf_sha1 = decode(repeat('00', 20), 'hex'), pdf_wrapped_key = '\x00'
			WHERE encounter_id = '` + c.encounters[c.a.aPast] + `'`,
		`UPDATE document_template SET status = 'draft', language = 'en', type_code = '57133-1'`,
		`INSERT INTO document_template_translation (template_id, language, texts) VALUES ('` + c.templates[clinicA] + `', 'en', '{}')`,
		`UPDATE document_template_translation SET texts = '{"title":"x"}'`,
		`DELETE FROM document_template_translation`,
	} {
		if err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		}); err != nil {
			t.Errorf("%s: %v, want success", stmt, err)
		}
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO document_template_translation (template_id, language, texts) VALUES ($1, 'de', '{}')`,
		c.templates[clinicA]); err != nil {
		t.Fatal(err)
	}
	for scope, want := range map[string]int{clinicA: 1, clinicB: 0} {
		var n int
		err := asApp(t, app, map[string]string{"app.organization_id": scope}, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, `SELECT count(*) FROM document_template_translation`).Scan(&n)
		})
		if err != nil || n != want {
			t.Errorf("translations seen by %s = %d, %v; want %d", scope, n, err, want)
		}
	}
	err := asApp(t, app, map[string]string{"app.organization_id": clinicB}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO document_template_translation (template_id, language, texts) VALUES ($1, 'it', '{}')`, c.templates[clinicA])
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("another clinic translating the template: %v, want insufficient_privilege (42501)", err)
	}
}
