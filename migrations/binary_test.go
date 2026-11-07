package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// insertFile is a synthetic file row of a response, inserted with the response's clinic, patient and
// appointment, as binary's service copies them from the holder's Target.
const insertFile = `INSERT INTO "binary" (organization_id, patient_id, appointment_id, questionnaire_response_id, link_id,
		content_type, size, title, sha1, kek_id, wrapped_key, segment_size)
	SELECT r.organization_id, r.patient_id, r.appointment_id, r.id, 'f1', 'image/jpeg', 1024, 'placeholder.jpg',
		digest(r.id::text, 'sha1'), 'local-1', gen_random_bytes(60), 65536
	FROM questionnaire_response r WHERE r.id = $1
	RETURNING id`

// seedFiles creates, as the owner, the rows of seedResponses and one quarantined file per response, and returns
// the file of each response.
func seedFiles(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, appointmentRows, responseRows, map[string]string) {
	t.Helper()
	p, a, r := seedResponses(t, owner)
	files := map[string]string{}
	for _, ids := range []map[string]string{r.completed, r.open} {
		for _, response := range ids {
			var id string
			if err := owner.QueryRow(t.Context(), insertFile, response).Scan(&id); err != nil {
				t.Fatal(err)
			}
			files[response] = id
		}
	}
	return p, a, r, files
}

// filesOf is the files of the responses, sorted.
func filesOf(files map[string]string, responses ...string) []string {
	var out []string
	for _, r := range responses {
		out = append(out, files[r])
	}
	return sorted(out...)
}

// idsWith returns the ids query gives in the scope with its arguments, sorted.
func idsWith(t *testing.T, pool *pgxpool.Pool, scope map[string]string, query string, args ...any) []string {
	t.Helper()
	var ids []string
	err := asApp(t, pool, scope, func(ctx context.Context, db postgres.Querier) error {
		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return sorted(ids...)
}

// tryAsOwner runs stmt as the owner in a transaction it rolls back, so each statement sees the seeded rows.
func tryAsOwner(t *testing.T, owner *pgxpool.Pool, stmt string) error {
	t.Helper()
	tx, err := owner.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = tx.Exec(t.Context(), stmt)
	return err
}

// jobScope is the scan and discard jobs' scope: one response of one clinic.
func jobScope(org, response string) map[string]string {
	return map[string]string{"app.organization_id": org, "app.questionnaire_response_id": response}
}

// Row-level security (docs/attachments/SPEC.md §2 "Tables"): the patient their own files; a clinic the files of
// its clinic whose response it can see (consent and D9 through questionnaire_response's own policies); the job
// scope the files of its one response, whatever the consent; nothing else.
func TestBinary_EachScopeSeesOnlyItsFiles(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r, f := seedFiles(t, owner)
	all := func(appts ...string) []string { return filesOf(f, both(r, appts...)...) }

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: all(a.aPast, a.aFuture, a.bPast, a.bFuture)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: all(a.other)},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: all(a.aPast, a.aFuture)},
		{name: "clinic B after the revocation: the past appointment's completed response's file only",
			scope: map[string]string{"app.organization_id": clinicB}, want: filesOf(f, r.completed[a.bPast])},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
		{name: "job: a response of clinic A", scope: jobScope(clinicA, r.open[a.aFuture]), want: filesOf(f, r.open[a.aFuture])},
		{name: "job: a response the clinic no longer sees after the revocation", scope: jobScope(clinicB, r.open[a.bFuture]),
			want: filesOf(f, r.open[a.bFuture])},
		{name: "job: a response named with another clinic", scope: jobScope(clinicB, r.open[a.aFuture])},
		{name: "job: a response alone", scope: map[string]string{"app.questionnaire_response_id": r.open[a.aFuture]}},
		{name: "job with an account set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA,
			"app.questionnaire_response_id": r.open[a.aFuture]}},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM "binary"`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// The patient inserts files only for themselves, on their own response, with that response's clinic and
// appointment; nobody else inserts one.
func TestBinary_OnlyThePatientInsertsAndOnlyOnTheirOwnResponse(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	insert := `INSERT INTO "binary" (organization_id, patient_id, appointment_id, questionnaire_response_id, link_id,
			content_type, size, title, sha1, kek_id, wrapped_key, segment_size)
		SELECT $2, $3, $4, $1, 'f1', 'application/pdf', 2048, 'placeholder.pdf', digest('x', 'sha1'), 'local-1', '\x00', 65536`
	own := r.open[a.aFuture]
	patientA := map[string]string{"app.patient_id": p[0].patient}
	tests := []struct {
		scope                map[string]string
		name                 string
		response, org        string
		patient, appointment string
		want                 string
	}{
		{name: "patient A, their own response", scope: patientA, response: own, org: clinicA, patient: p[0].patient, appointment: a.aFuture},
		{name: "patient A, as patient B", scope: patientA, response: own, org: clinicA, patient: p[1].patient, appointment: a.aFuture, want: "42501"},
		{name: "patient A, on patient B's response", scope: patientA, response: r.open[a.other], org: clinicA, patient: p[0].patient,
			appointment: a.other, want: "42501"},
		{name: "patient A, their response with another clinic", scope: patientA, response: own, org: clinicB, patient: p[0].patient,
			appointment: a.aFuture, want: "42501"},
		{name: "patient A, their response with another appointment", scope: patientA, response: own, org: clinicA, patient: p[0].patient,
			appointment: a.aPast, want: "42501"},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, response: own, org: clinicA, patient: p[0].patient,
			appointment: a.aFuture, want: "42501"},
		{name: "the job scope", scope: jobScope(clinicA, own), response: own, org: clinicA, patient: p[0].patient, appointment: a.aFuture, want: "42501"},
	}
	for _, tt := range tests {
		_, err := exec(t, app, tt.scope, insert, tt.response, tt.org, tt.patient, tt.appointment)
		if sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %q", tt.name, err, tt.want)
		}
	}
}

// Deletes: the patient their own files, the job scope its response's; updates: the job scope only. The clinic
// does neither.
func TestBinary_EachScopeUpdatesAndDeletesOnlyWhatItMay(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r, f := seedFiles(t, owner)
	update := `UPDATE "binary" SET status = 'clean', scanned_at = now(), updated_at = now() WHERE id = ANY($1::uuid[])`
	remove := `DELETE FROM "binary" WHERE id = ANY($1::uuid[])`
	everyFile := filesOf(f, both(r, a.aPast, a.aFuture, a.bPast, a.bFuture, a.other)...)
	tests := []struct {
		scope map[string]string
		name  string
		stmt  string
		ids   []string
		want  int64
	}{
		{name: "clinic A updates", scope: map[string]string{"app.organization_id": clinicA}, stmt: update, ids: everyFile},
		{name: "patient A updates", scope: map[string]string{"app.patient_id": p[0].patient}, stmt: update, ids: everyFile},
		{name: "the job scope updates", scope: jobScope(clinicB, r.open[a.bFuture]), stmt: update, ids: everyFile, want: 1},
		{name: "clinic A deletes", scope: map[string]string{"app.organization_id": clinicA}, stmt: remove, ids: everyFile},
		{name: "patient B deletes patient A's", scope: map[string]string{"app.patient_id": p[1].patient}, stmt: remove,
			ids: filesOf(f, both(r, a.aPast, a.aFuture)...)},
		{name: "the job scope deletes", scope: jobScope(clinicA, r.open[a.aFuture]), stmt: remove, ids: everyFile, want: 1},
		{name: "patient A deletes their own", scope: map[string]string{"app.patient_id": p[0].patient}, stmt: remove, ids: everyFile, want: 7},
	}
	for _, tt := range tests {
		if n, err := exec(t, app, tt.scope, tt.stmt, tt.ids); err != nil || n != tt.want {
			t.Errorf("%s: %d rows, %v; want %d", tt.name, n, err, tt.want)
		}
	}
}

// The app role updates only the scan's columns: what a file is, whose and where it is never changes.
func TestBinary_TheAppRoleUpdatesOnlyTheScansColumns(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, a, r, _ := seedFiles(t, owner)
	scope := jobScope(clinicA, r.open[a.aFuture])
	for _, stmt := range []string{
		`TRUNCATE "binary"`,
		`UPDATE "binary" SET id = id`, `UPDATE "binary" SET organization_id = organization_id`,
		`UPDATE "binary" SET patient_id = patient_id`, `UPDATE "binary" SET appointment_id = appointment_id`,
		`UPDATE "binary" SET questionnaire_response_id = questionnaire_response_id`, `UPDATE "binary" SET link_id = link_id`,
		`UPDATE "binary" SET content_type = content_type`, `UPDATE "binary" SET size = size`, `UPDATE "binary" SET title = title`,
		`UPDATE "binary" SET sha1 = sha1`, `UPDATE "binary" SET kek_id = kek_id`, `UPDATE "binary" SET segment_size = segment_size`,
		`UPDATE "binary" SET created_at = now()`,
	} {
		if _, err := exec(t, app, scope, stmt); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	rejected := `UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL, updated_at = now()`
	if n, err := exec(t, app, scope, rejected); err != nil || n != 1 {
		t.Errorf("a rejection: %d rows, %v; want 1", n, err)
	}
}

// The checks of SPEC.md §2 "Tables": the four content types, the three statuses, a positive size, a title of at
// most 200 characters, a 20-byte SHA-1, a scan time exactly once scanned, a wrapped key exactly while not
// rejected, and a response that exists.
func TestBinary_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, a, r, _ := seedFiles(t, owner)
	ctx := t.Context()
	for stmt, want := range map[string]string{
		`UPDATE "binary" SET content_type = 'image/heic'`:                          "23514",
		`UPDATE "binary" SET content_type = 'video/quicktime'`:                     "23514",
		`UPDATE "binary" SET status = 'infected'`:                                  "23514",
		`UPDATE "binary" SET size = 0`:                                             "23514",
		`UPDATE "binary" SET title = repeat('x', 201)`:                             "23514",
		`UPDATE "binary" SET sha1 = '\x00'`:                                        "23514",
		`UPDATE "binary" SET segment_size = 0`:                                     "23514",
		`UPDATE "binary" SET status = 'clean'`:                                     "23514", // no scan time
		`UPDATE "binary" SET scanned_at = now()`:                                   "23514", // still quarantined
		`UPDATE "binary" SET status = 'rejected', scanned_at = now()`:              "23514", // the key kept
		`UPDATE "binary" SET wrapped_key = NULL`:                                   "23514", // not rejected
		`UPDATE "binary" SET questionnaire_response_id = gen_random_uuid()`:        "23503",
		`UPDATE "binary" SET title = repeat('é', 200), content_type = 'video/mp4'`: "",
		`UPDATE "binary" SET status = 'clean', scanned_at = now()`:                 "",
	} {
		if err := tryAsOwner(t, owner, stmt); sqlState(err) != want {
			t.Errorf("%s: %v, want SQLSTATE %q", stmt, err, want)
		}
	}
	var n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM "binary" WHERE questionnaire_response_id = $1 AND link_id = 'f1'`,
		r.open[a.aFuture]).Scan(&n); err != nil || n != 1 {
		t.Errorf("the response's files on the item: %d, %v; want 1", n, err)
	}
}

// Deleting a response deletes its files' rows in the same statement, by every path: the discard job's scope
// for a cancelled appointment, and salus_retention without any privilege of its own on "binary": the referential
// action runs as the table's owner. 00028 grants the role SELECT and DELETE there for binary's purger, so this
// test revokes them in its own database first.
func TestBinary_DeletingAResponseDeletesItsFiles(t *testing.T) {
	t.Parallel()
	t.Run("the discard scope", func(t *testing.T) {
		t.Parallel()
		owner, app := pgtest.NewAppDatabase(t)
		_, a, r, f := seedFiles(t, owner)
		discard := map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture}
		if n, err := exec(t, app, discard, "DELETE FROM questionnaire_response"); err != nil || n != 2 {
			t.Fatalf("discard: %d, %v; want 2", n, err)
		}
		if got := idsWith(t, owner, nil, `SELECT id::text FROM "binary" WHERE questionnaire_response_id = ANY($1::uuid[])`, both(r, a.bFuture)); len(got) != 0 {
			t.Errorf("the cancelled appointment's files are left: %v", got)
		}
		if got := idsWith(t, owner, nil, `SELECT id::text FROM "binary"`); len(got) != len(f)-2 {
			t.Errorf("%d files left, want %d", len(got), len(f)-2)
		}
	})
	t.Run("salus_retention", func(t *testing.T) {
		t.Parallel()
		owner, ret := pgtest.NewRetentionDatabase(t)
		_, a, r, _ := seedFiles(t, owner)
		if _, err := owner.Exec(t.Context(), `REVOKE SELECT, DELETE ON "binary" FROM `+pgtest.RetentionRole); err != nil {
			t.Fatal(err)
		}
		if _, err := ret.Exec(t.Context(), `SELECT count(*) FROM "binary"`); sqlState(err) != "42501" {
			t.Errorf("%s reads binary: %v, want insufficient_privilege", pgtest.RetentionRole, err)
		}
		tag, err := ret.Exec(t.Context(), "DELETE FROM questionnaire_response WHERE status <> 'completed' AND due_at < $1", "2027-01-01T00:00:00Z")
		if err != nil || tag.RowsAffected() != 2 {
			t.Fatalf("purge: %d rows, %v; want 2", tag.RowsAffected(), err)
		}
		if got := idsWith(t, owner, nil, `SELECT questionnaire_response_id::text FROM "binary" WHERE questionnaire_response_id = ANY($1::uuid[])`,
			[]string{r.open[a.aPast], r.open[a.bPast]}); len(got) != 0 {
			t.Errorf("the purged responses' files are left: %v", got)
		}
	})
}

// questionnaire_response.attachments: an array, empty once withdrawn, updated by the app role; the job scope
// reads and updates its one response, whatever the consent, and no response of another clinic.
func TestQuestionnaireResponse_TheJobScopeReadsAndUpdatesItsResponsesAttachments(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	target := r.open[a.bFuture] // clinic B no longer sees it after the revocation
	if got := idsWith(t, app, jobScope(clinicB, target), "SELECT id::text FROM questionnaire_response WHERE id = $1", target); !slices.Equal(got, []string{target}) {
		t.Errorf("the job scope sees %v, want its response", got)
	}
	update := `UPDATE questionnaire_response SET attachments = '[{"linkId": "f1", "binaryId": "00000000-0000-4000-8000-000000000001", "pending": true}]',
		updated_at = now() WHERE id = $1`
	tests := []struct {
		scope map[string]string
		name  string
		want  int64
	}{
		{name: "the job scope", scope: jobScope(clinicB, target), want: 1},
		{name: "the job scope named with another clinic", scope: jobScope(clinicA, target)},
		{name: "the job scope with a patient set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicB,
			"app.questionnaire_response_id": target}},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB}},
		{name: "patient A, their own", scope: map[string]string{"app.patient_id": p[0].patient}, want: 1},
	}
	for _, tt := range tests {
		if n, err := exec(t, app, tt.scope, update, target); err != nil || n != tt.want {
			t.Errorf("%s: %d rows, %v; want %d", tt.name, n, err, tt.want)
		}
	}
	for stmt, want := range map[string]string{
		"UPDATE questionnaire_response SET attachments = '{}'":                                             "23514",
		"UPDATE questionnaire_response SET status = 'stopped', answers = '[]' WHERE id = '" + target + "'": "23514",
	} {
		if err := tryAsOwner(t, owner, stmt); sqlState(err) != want {
			t.Errorf("%s: %v, want SQLSTATE %s", stmt, err, want)
		}
	}
	var attachments string
	if err := owner.QueryRow(t.Context(), "SELECT attachments::text FROM questionnaire_response WHERE id = $1", r.open[a.aFuture]).Scan(&attachments); err != nil ||
		attachments != "[]" {
		t.Errorf("a response's attachments default to %q, %v; want []", attachments, err)
	}
}
