package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The tables of the consultation's transcript, suggestions and drafts (docs/plans/consult-intelligence.md),
// each row belonging to one encounter.
var consultTables = []string{"transcript_segment", "suggestion", "document_draft"}

// consultRows is the seeded consultations: an encounter per seeded appointment, and a document template per
// clinic.
type consultRows struct {
	encounters map[string]string // by appointment
	templates  map[string]string // by clinic
	patients   [2]patientRow
	a          appointmentRows
}

// seedConsult creates, as the owner, the appointments and their encounters, a template per clinic, and for
// every encounter one transcript segment, one suggestion citing it and one draft from its clinic's template.
func seedConsult(t *testing.T, owner *pgxpool.Pool) consultRows {
	t.Helper()
	p, _, a := seedAppointments(t, owner)
	c := consultRows{patients: p, a: a, encounters: seedEncounters(t, owner, a), templates: map[string]string{}}
	for _, clinic := range []string{clinicA, clinicB} {
		var id string
		if err := owner.QueryRow(t.Context(), `INSERT INTO document_template (organization_id, title, fields)
			VALUES ($1, 'Placeholder template', '[{"key":"complaint","label":"Placeholder label"}]') RETURNING id`,
			clinic).Scan(&id); err != nil {
			t.Fatal(err)
		}
		c.templates[clinic] = id
	}
	for _, e := range c.encounters {
		_, err := owner.Exec(t.Context(), `WITH s AS (
				INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, translation, ts_start, ts_end)
				VALUES (gen_random_uuid(), $1, 'patient', 'bs', 'Placeholder original', '[de] Placeholder original', now(), now()) RETURNING id),
			g AS (INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
				SELECT $1, 'question', 'Placeholder question', 0.5, ARRAY[id] FROM s)
			INSERT INTO document_draft (encounter_id, template_id, fields)
			SELECT e.id, t.id, '[{"key":"a","missing":true}]' FROM encounter e JOIN document_template t ON t.organization_id = e.organization_id
			WHERE e.id = $1`, e)
		if err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// insertConsultRow returns an insert of one row of table for the encounter, the draft from template.
func insertConsultRow(table, encounter, template string) func(ctx context.Context, db postgres.Querier) error {
	return func(ctx context.Context, db postgres.Querier) error {
		var err error
		switch table {
		case "transcript_segment":
			_, err = db.Exec(ctx, `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
				VALUES (gen_random_uuid(), $1, 'doctor', 'de', 'Placeholder original', now(), now())`, encounter)
		case "suggestion":
			_, err = db.Exec(ctx, `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
				VALUES ($1, 'term', 'Placeholder term', 0.9, ARRAY[gen_random_uuid()])`, encounter)
		case "document_draft":
			_, err = db.Exec(ctx, `INSERT INTO document_draft (encounter_id, template_id, fields)
				VALUES ($1, $2, '[{"key":"a","missing":true}]')`, encounter, template)
		}
		return err
	}
}

// Every row follows its encounter in the clinic's scope only: a clinic sees the rows of the encounters it sees
// (an active consent, or after a revocation those that started before it). The patient's and an account's
// scope see none (the patient's copy comes with the export feature), and neither does a video room's, which
// shows an encounter after a revocation.
func TestConsultTables_OnlyTheClinicScopeSeesTheRowsOfTheEncountersItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	seedVideoSessions(t, owner, c.a)
	e, a := c.encounters, c.a

	tests := []struct {
		scope map[string]string
		name  string
		want  []string // encounter ids
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": c.patients[0].patient}},
		{name: "patient A with clinic A set", scope: map[string]string{"app.patient_id": c.patients[0].patient, "app.organization_id": clinicA}},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": c.patients[0].account, "app.organization_id": clinicA}},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA},
			want: sorted(e[a.aPast], e[a.aFuture])},
		{name: "clinic B after the revocation: the past appointment's only", scope: map[string]string{"app.organization_id": clinicB},
			want: sorted(e[a.bPast])},
		{name: "a video room alone", scope: map[string]string{"app.video_room": "r_" + a.bFuture}},
		{name: "a video room with clinic B set", scope: map[string]string{"app.organization_id": clinicB, "app.video_room": "r_" + a.bFuture}},
	}
	for _, table := range consultTables {
		for _, tt := range tests {
			if got := idsIn(t, app, tt.scope, "SELECT encounter_id::text FROM "+table); !slices.Equal(got, tt.want) {
				t.Errorf("%s, %s: sees %v, want %v", table, tt.name, got, tt.want)
			}
		}
	}
}

// Only the clinic's scope writes a row, for an encounter it sees; a draft only from its own clinic's template.
func TestConsultTables_OnlyTheClinicScopeWritesRowsOfTheEncountersItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	seedVideoSessions(t, owner, c.a)
	e, a := c.encounters, c.a

	tests := []struct {
		scope     map[string]string
		name      string
		encounter string
		template  string
		want      string // the SQLSTATE, "" for success
	}{
		{name: "clinic A, its encounter", scope: map[string]string{"app.organization_id": clinicA}, encounter: e[a.aFuture], template: c.templates[clinicA]},
		{name: "clinic A, a patient without a consent", scope: map[string]string{"app.organization_id": clinicA}, encounter: e[a.other],
			template: c.templates[clinicA], want: "42501"},
		{name: "clinic A, clinic B's encounter", scope: map[string]string{"app.organization_id": clinicA}, encounter: e[a.bPast],
			template: c.templates[clinicA], want: "42501"},
		{name: "clinic B, after the revocation, a later appointment", scope: map[string]string{"app.organization_id": clinicB},
			encounter: e[a.bFuture], template: c.templates[clinicB], want: "42501"},
		{name: "patient A, their encounter", scope: map[string]string{"app.patient_id": c.patients[0].patient, "app.organization_id": clinicA},
			encounter: e[a.aFuture], template: c.templates[clinicA], want: "42501"},
		{name: "a video room with clinic B set", scope: map[string]string{"app.organization_id": clinicB, "app.video_room": "r_" + a.bFuture},
			encounter: e[a.bFuture], template: c.templates[clinicB], want: "42501"},
	}
	for _, table := range consultTables {
		for _, tt := range tests {
			if err := asApp(t, app, tt.scope, insertConsultRow(table, tt.encounter, tt.template)); sqlState(err) != tt.want {
				t.Errorf("%s, %s: %v, want SQLSTATE %q", table, tt.name, err, tt.want)
			}
		}
	}
	otherTemplate := insertConsultRow("document_draft", e[a.aFuture], c.templates[clinicB])
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, otherTemplate); sqlState(err) != "42501" {
		t.Errorf("clinic A drafts from clinic B's template: %v, want SQLSTATE 42501", err)
	}
}

// A template is its clinic's: only that clinic's scope sees and writes it.
func TestDocumentTemplate_OnlyItsClinicsScopeSeesAndWritesIt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)

	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(c.templates[clinicA])},
		{name: "patient A with clinic A set", scope: map[string]string{"app.patient_id": c.patients[0].patient, "app.organization_id": clinicA}},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": c.patients[0].account, "app.organization_id": clinicA}},
		{name: "a video room with clinic A set", scope: map[string]string{"app.organization_id": clinicA, "app.video_room": "r_" + c.a.aFuture}},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM document_template"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
	insert := func(org string) func(ctx context.Context, db postgres.Querier) error {
		return func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO document_template (organization_id, title, fields)
				VALUES ($1, 'Placeholder template', '[{"key":"plan","label":"Placeholder label"}]')`, org)
			return err
		}
	}
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, insert(clinicA)); err != nil {
		t.Errorf("clinic A adds a template: %v", err)
	}
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, insert(clinicB)); sqlState(err) != "42501" {
		t.Errorf("clinic A adds clinic B's template: %v, want SQLSTATE 42501", err)
	}
	room := map[string]string{"app.organization_id": clinicA, "app.video_room": "r_" + c.a.aFuture}
	if err := asApp(t, app, room, insert(clinicA)); sqlState(err) != "42501" {
		t.Errorf("a video room with clinic A set adds a template: %v, want SQLSTATE 42501", err)
	}
}

// salus_app writes only what the pipeline needs: a segment never changes; a suggestion's status, a draft's
// status and fields and a template's title and fields do. Nothing is deleted, and no row changes encounter,
// template or clinic.
func TestConsultTables_TheAppRoleWritesOnlyWhatThePipelineNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedConsult(t, owner)
	scope := map[string]string{"app.organization_id": clinicA}
	exec := func(stmt string) func(ctx context.Context, db postgres.Querier) error {
		return func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		}
	}
	for _, stmt := range []string{
		"UPDATE suggestion SET status = 'pinned', updated_at = now(), version_id = version_id + 1",
		"UPDATE document_draft SET status = 'final', fields = '[{\"key\":\"a\",\"value\":\"b\"}]', updated_at = now(), version_id = version_id + 1",
		"UPDATE document_template SET title = 'Placeholder title', fields = '[{\"key\":\"a\",\"label\":\"b\"}]', updated_at = now(), version_id = version_id + 1",
	} {
		if err := asApp(t, app, scope, exec(stmt)); err != nil {
			t.Errorf("%s: %v, want success", stmt, err)
		}
	}
	for _, stmt := range []string{
		"UPDATE transcript_segment SET original = original", "UPDATE transcript_segment SET translation = translation",
		"DELETE FROM transcript_segment", "TRUNCATE transcript_segment",
		"UPDATE suggestion SET text = text", "UPDATE suggestion SET encounter_id = encounter_id",
		"UPDATE suggestion SET source_segment_ids = source_segment_ids", "DELETE FROM suggestion",
		"UPDATE document_draft SET encounter_id = encounter_id", "UPDATE document_draft SET template_id = template_id",
		"DELETE FROM document_draft",
		"UPDATE document_template SET organization_id = organization_id", "DELETE FROM document_template",
	} {
		if err := asApp(t, app, scope, exec(stmt)); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
}

// The checks keep each row in the shape of its event (internal/transcript, suggestion, composition).
func TestConsultTables_HoldAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	e := c.encounters[c.a.aPast]
	tpl := c.templates[clinicA]

	for _, tt := range []struct {
		name string
		stmt string
		args []any
	}{
		{"an unknown speaker", `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
			VALUES (gen_random_uuid(), $1, 'practitioner', 'bs', 'x', now(), now())`, []any{e}},
		{"no lang", `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
			VALUES (gen_random_uuid(), $1, 'doctor', '', 'x', now(), now())`, []any{e}},
		{"a blank original", `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
			VALUES (gen_random_uuid(), $1, 'doctor', 'bs', ' ', now(), now())`, []any{e}},
		{"an end before the start", `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
			VALUES (gen_random_uuid(), $1, 'doctor', 'bs', 'x', now(), now() - interval '1 second')`, []any{e}},
		{"an unknown suggestion type", `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
			VALUES ($1, 'treatment', 'x', 0.5, ARRAY[gen_random_uuid()])`, []any{e}},
		{"a confidence over 1", `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
			VALUES ($1, 'term', 'x', 1.5, ARRAY[gen_random_uuid()])`, []any{e}},
		{"a suggestion from no segment", `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
			VALUES ($1, 'term', 'x', 0.5, '{}')`, []any{e}},
		{"a blank suggestion", `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids)
			VALUES ($1, 'term', '', 0.5, ARRAY[gen_random_uuid()])`, []any{e}},
		{"an unknown suggestion status", `INSERT INTO suggestion (encounter_id, type, text, confidence, source_segment_ids, status)
			VALUES ($1, 'term', 'x', 0.5, ARRAY[gen_random_uuid()], 'accepted')`, []any{e}},
		{"an unknown draft status", `INSERT INTO document_draft (encounter_id, template_id, status, fields)
			VALUES ($1, $2, 'draft', '[{"key":"a","missing":true}]')`, []any{e, tpl}},
		{"a draft with no field", `INSERT INTO document_draft (encounter_id, template_id, fields) VALUES ($1, $2, '[]')`, []any{e, tpl}},
		{"draft fields no array", `INSERT INTO document_draft (encounter_id, template_id, fields) VALUES ($1, $2, '{}')`, []any{e, tpl}},
		{"a template without fields", `INSERT INTO document_template (organization_id, title, fields) VALUES ($1, 'x', '[]')`, []any{clinicA}},
		{"a template fields no array", `INSERT INTO document_template (organization_id, title, fields) VALUES ($1, 'x', '{}')`, []any{clinicA}},
		{"a blank template title", `INSERT INTO document_template (organization_id, title, fields)
			VALUES ($1, ' ', '[{"key":"a","label":"b"}]')`, []any{clinicA}},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, tt.args...); sqlState(err) != "23514" {
			t.Errorf("%s: %v, want check_violation (23514)", tt.name, err)
		}
	}
	var status string
	if err := owner.QueryRow(t.Context(), `SELECT d.status || '/' || s.status FROM document_draft d, suggestion s
		WHERE d.encounter_id = $1 AND s.encounter_id = $1`, e).Scan(&status); err != nil || status != "preliminary/shown" {
		t.Errorf("defaults = %q, %v; want a preliminary draft and a shown suggestion", status, err)
	}
}

// A stored segment keeps the segment_id of its live events, which suggestions cite: the id comes from the
// event, never from the database.
func TestTranscriptSegment_TakesItsIDFromTheEvent(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	e := c.encounters[c.a.aPast]

	_, err := owner.Exec(t.Context(), `INSERT INTO transcript_segment (encounter_id, speaker, lang, original, ts_start, ts_end)
		VALUES ($1, 'doctor', 'bs', 'x', now(), now())`, e)
	if sqlState(err) != "23502" {
		t.Errorf("a segment without its segment_id: %v, want not_null_violation (23502)", err)
	}
	const id = "6f1d2c3b-4a5e-4f60-8a7b-9c0d1e2f3a4b"
	var got string
	if err := owner.QueryRow(t.Context(), `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, ts_start, ts_end)
		VALUES ($1, $2, 'doctor', 'bs', 'x', now(), now()) RETURNING id::text`, id, e).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Errorf("stored id = %s, want the event's %s", got, id)
	}
}
