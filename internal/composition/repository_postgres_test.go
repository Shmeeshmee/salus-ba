package composition_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed is a clinic's encounter, as the owner: a clinic with a practitioner and role, a patient with an active
// consent, an appointment and its encounter in progress. Synthetic.
type seed struct{ clinic, role, encounter string }

func seedEncounter(t testing.TB, owner *pgxpool.Pool) seed {
	t.Helper()
	var s seed
	err := owner.QueryRow(context.Background(), `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id),
		a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		pa AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT pa.id, r.organization_id, 'active', now() - interval '30 days' FROM pa, r RETURNING patient_id),
		ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT c.patient_id, r.id, r.organization_id, gen_random_uuid(), tstzrange(now(), now() + interval '30 minutes') FROM c, r
			RETURNING id, patient_id, practitioner_role_id, organization_id)
		INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', now() FROM ap
		RETURNING organization_id::text, practitioner_role_id::text, id::text`).Scan(&s.clinic, &s.role, &s.encounter)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// inClinic runs fn on the repository in a transaction of its own, scoped to the clinic.
func inClinic(t testing.TB, app *pgxpool.Pool, clinic string, fn func(ctx context.Context, r composition.Repository) error) error {
	t.Helper()
	r := composition.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(context.Background(), func(ctx context.Context) error {
		if err := r.SetClinic(ctx, clinic); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

var at = time.Date(2030, 1, 7, 9, 30, 0, 0, time.UTC)

// A template is stored with its translations and read back as it was written, in its clinic only, by title.
func TestTemplates_StoresAndReadsATemplateWithItsTranslations(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s, other := seedEncounter(t, owner), seedEncounter(t, owner)
	var tpl composition.Template
	err := inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		var err error
		tpl, err = r.InsertTemplate(ctx, s.clinic, content(), at)
		if err != nil {
			return err
		}
		second := content()
		second.Title, second.Translations = "A placeholder referral", nil
		_, err = r.InsertTemplate(ctx, s.clinic, second, at)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := composition.Template{
		ID: tpl.ID, OrganizationID: s.clinic, LastUpdated: at, VersionID: 1, Title: content().Title, Language: "bs",
		Status: composition.TemplateDraft, TypeCode: "11488-4", Fields: content().Fields, Translations: content().Translations,
	}
	if !sameTemplate(tpl, want) {
		t.Errorf("InsertTemplate() = %+v,\nwant %+v", tpl, want)
	}
	var list []composition.Template
	var read composition.Template
	err = inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		var err error
		if list, err = r.Templates(ctx, s.clinic); err != nil {
			return err
		}
		read, err = r.Template(ctx, tpl.ID, true)
		return err
	})
	if err != nil || len(list) != 2 || list[0].Title != "A placeholder referral" || !sameTemplate(list[1], want) || !sameTemplate(read, want) {
		t.Errorf("Templates() = %+v, Template() = %+v, %v", list, read, err)
	}
	for _, id := range []string{tpl.ID, unknownID, "x"} {
		err = inClinic(t, app, other.clinic, func(ctx context.Context, r composition.Repository) error {
			_, err := r.Template(ctx, id, false)
			return err
		})
		if !errors.Is(err, composition.ErrTemplateNotFound) {
			t.Errorf("Template(%s) in another clinic = %v, want ErrTemplateNotFound", id, err)
		}
	}
}

// SAL-16: the owner (salusctl) is past row-level security, so Templates reads the clinic's templates by its id,
// never another clinic's.
func TestTemplates_ReadsOnlyTheClinicsTemplatesAsTheOwner(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	a, b := seedEncounter(t, owner), seedEncounter(t, owner)
	var list []composition.Template
	err := inClinic(t, owner, a.clinic, func(ctx context.Context, r composition.Repository) error {
		if _, err := r.InsertTemplate(ctx, a.clinic, content(), at); err != nil {
			return err
		}
		if _, err := r.InsertTemplate(ctx, b.clinic, content(), at); err != nil {
			return err
		}
		var err error
		list, err = r.Templates(ctx, b.clinic)
		return err
	})
	if err != nil || len(list) != 1 || list[0].OrganizationID != b.clinic || len(list[0].Translations) != len(content().Translations) {
		t.Errorf("Templates(other clinic) as the owner = %+v, %v; want its one template", list, err)
	}
	err = inClinic(t, owner, a.clinic, func(ctx context.Context, r composition.Repository) error {
		var err error
		list, err = r.Templates(ctx, "x")
		return err
	})
	if err != nil || len(list) != 0 {
		t.Errorf("Templates(x) = %+v, %v; want none", list, err)
	}
}

func sameTemplate(a, b composition.Template) bool {
	if len(a.Translations) != len(b.Translations) {
		return false
	}
	for i := range a.Translations {
		ta, tb := a.Translations[i], b.Translations[i]
		if ta.Language != tb.Language || ta.Title != tb.Title || len(ta.Fields) != len(tb.Fields) {
			return false
		}
		for k, v := range ta.Fields {
			if tb.Fields[k] != v {
				return false
			}
		}
	}
	return a.ID == b.ID && a.OrganizationID == b.OrganizationID && a.LastUpdated.Equal(b.LastUpdated) && a.VersionID == b.VersionID &&
		a.Title == b.Title && a.Language == b.Language && a.Status == b.Status && a.TypeCode == b.TypeCode && slices.Equal(a.Fields, b.Fields)
}

// An update replaces the content and the translations as the next version.
func TestUpdateTemplate_ReplacesTheContentAndTranslations(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedEncounter(t, owner)
	var got composition.Template
	err := inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		tpl, err := r.InsertTemplate(ctx, s.clinic, content(), at)
		if err != nil {
			return err
		}
		c := content()
		c.Language, c.Status, c.TypeCode = "en", composition.TemplateActive, "57133-1"
		c.Translations[0].Language = "hr"
		got, err = r.UpdateTemplate(ctx, tpl.ID, c, at.Add(time.Hour))
		return err
	})
	if err != nil || got.VersionID != 2 || got.Language != "en" || got.Status != composition.TemplateActive || got.TypeCode != "57133-1" ||
		len(got.Translations) != 1 || got.Translations[0].Language != "hr" || !got.LastUpdated.Equal(at.Add(time.Hour)) {
		t.Errorf("UpdateTemplate() = %+v, %v", got, err)
	}
}

// draftOf is a draft of the template for the encounter: the findings cited, the rest missing.
func draftOf(encounterID, templateID string) composition.Draft {
	return composition.Draft{
		EncounterID: encounterID, TemplateID: templateID, Language: "bs", Title: "Placeholder findings report", TypeCode: "11488-4",
		Fields: []composition.DraftField{
			{Field: composition.Field{Key: "findings", Value: "Placeholder findings", SourceSegmentIDs: []string{seg1}}, Type: composition.FieldText, Label: "Placeholder label", Instruction: "Placeholder instruction", Required: true},
			{Field: composition.Field{Key: "tests", Missing: true}, Type: composition.FieldList, Label: "Placeholder label"},
		},
	}
}

// The open draft of a template is created once and then replaced as its next version; it reads back as stored,
// and is signed once, from the version read, with its signature found by its DocumentReference and Provenance.
func TestDrafts_SaveReplaceAndSign(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s, other := seedEncounter(t, owner), seedEncounter(t, owner)
	var tplID string
	var first, second composition.Draft
	var created [2]bool
	err := inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		tpl, err := r.InsertTemplate(ctx, s.clinic, content(), at)
		if err != nil {
			return err
		}
		tplID = tpl.ID
		if first, created[0], err = r.SaveDraft(ctx, draftOf(s.encounter, tpl.ID), at); err != nil {
			return err
		}
		again := draftOf(s.encounter, tpl.ID)
		again.Language, again.Fields[1].Field = "en", composition.Field{Key: "tests", Value: "one", SourceSegmentIDs: []string{seg2}}
		second, created[1], err = r.SaveDraft(ctx, again, at.Add(time.Minute))
		return err
	})
	if err != nil || created != [2]bool{true, false} || second.ID != first.ID || first.VersionID != 1 || second.VersionID != 2 ||
		second.Language != "en" || second.Fields[1].Value != "one" || second.Status != composition.StatusPreliminary || !second.UpdatedAt.Equal(at.Add(time.Minute)) {
		t.Fatalf("SaveDraft() = %+v, %+v, created %v, %v", first, second, created, err)
	}
	if first.Fields[0].Label != "Placeholder label" || first.Fields[0].Instruction != "Placeholder instruction" || !first.Fields[0].Required ||
		!slices.Equal(first.Fields[0].SourceSegmentIDs, []string{seg1}) || first.Fields[0].Type != composition.FieldText {
		t.Errorf("stored field = %+v", first.Fields[0])
	}

	signed := second
	signed.Status, signed.SignedAt, signed.SignerRoleID = composition.StatusFinal, at.Add(time.Hour), s.role
	signed.DocumentReferenceID, signed.ProvenanceID = "d0c00000-0000-4000-8000-000000000001", "9e000000-0000-4000-8000-000000000001"
	signed.PDF = composition.PDF{KEKID: "k1", SHA1: bytes.Repeat([]byte{1}, 20), WrappedKey: []byte{2, 3}, Size: 1234}
	var got, byRef, byProv composition.Draft
	var list []composition.Draft
	err = inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		var err error
		if got, err = r.Sign(ctx, signed, at.Add(time.Hour)); err != nil {
			return err
		}
		if byRef, err = r.ByDocumentReference(ctx, signed.DocumentReferenceID); err != nil {
			return err
		}
		if byProv, err = r.ByProvenance(ctx, signed.ProvenanceID); err != nil {
			return err
		}
		list, err = r.Drafts(ctx, s.encounter)
		return err
	})
	if err != nil || got.VersionID != 3 || got.Status != composition.StatusFinal || !got.SignedAt.Equal(at.Add(time.Hour)) || got.SignerRoleID != s.role ||
		got.PDF.KEKID != "k1" || got.PDF.Size != 1234 || !bytes.Equal(got.PDF.WrappedKey, []byte{2, 3}) || byRef.ID != got.ID || byProv.ID != got.ID ||
		len(list) != 1 || list[0].ID != got.ID {
		t.Fatalf("Sign() = %+v, by reference %s, by provenance %s, drafts %d, %v", got, byRef.ID, byProv.ID, len(list), err)
	}
	err = inClinic(t, app, s.clinic, func(ctx context.Context, r composition.Repository) error {
		_, err := r.Sign(ctx, signed, at)
		return err
	})
	if !errors.Is(err, composition.ErrVersionMismatch) {
		t.Errorf("Sign() again = %v, want ErrVersionMismatch", err)
	}
	for name, read := range map[string]func(context.Context, composition.Repository) error{
		"draft": func(ctx context.Context, r composition.Repository) error {
			_, err := r.Draft(ctx, got.ID, true)
			return err
		},
		"reference": func(ctx context.Context, r composition.Repository) error {
			_, err := r.ByDocumentReference(ctx, "x")
			return err
		},
		"provenance": func(ctx context.Context, r composition.Repository) error {
			_, err := r.ByProvenance(ctx, signed.ProvenanceID)
			return err
		},
	} {
		if err := inClinic(t, app, other.clinic, read); !errors.Is(err, composition.ErrNotFound) {
			t.Errorf("%s in another clinic: %v, want ErrNotFound", name, err)
		}
	}
	_ = tplID
}
