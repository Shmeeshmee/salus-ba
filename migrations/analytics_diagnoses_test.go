package migrations_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// analyticsDiagnoses is the version of 00041 (SAL-19, docs/plans/analytics.md §3).
const analyticsDiagnoses = 41

// signWithCodes signs the encounter's seeded draft as the owner, with a coded field per code and one text field,
// as composition's Sign stores them.
func signWithCodes(t *testing.T, owner *pgxpool.Pool, encounter string, codes ...string) string {
	t.Helper()
	fields := `[{"type":"text","key":"complaint","value":"Placeholder complaint","missing":false,"required":false}`
	for i, c := range codes {
		fields += `,{"type":"coded","key":"diagnosis` + string(rune('a'+i)) + `","value":"` + c + `","missing":false,"required":false}`
	}
	fields += `,{"type":"coded","key":"empty","value":"","missing":true,"required":false}]`
	var id string
	err := owner.QueryRow(t.Context(), `UPDATE document_draft d SET status = 'final', fields = $2, signed_at = '2026-11-02T09:30:00Z',
			pdf_size = 1, signer_role_id = e.practitioner_role_id, document_reference_id = gen_random_uuid(),
			provenance_id = gen_random_uuid(), pdf_kek_id = 'kek', pdf_sha1 = '\x0000000000000000000000000000000000000000',
			pdf_wrapped_key = '\x00'
		FROM encounter e WHERE e.id = d.encounter_id AND d.encounter_id = $1 RETURNING d.id`, encounter, fields).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// The backfill turns every coded value of a document signed before 00041 into an unconfirmed Condition of the
// document's patient and clinic, recorded when and by whom it was signed; open drafts and empty values make none.
func TestAnalyticsDiagnoses_TheBackfillMakesUnconfirmedConditionsOfSignedDocuments(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewEmptyDatabaseURL(t)
	provider := gooseProvider(t, dbURL)
	if _, err := provider.UpTo(t.Context(), analyticsDiagnoses-1); err != nil {
		t.Fatal(err)
	}
	owner := open(t, dbURL)
	c := seedConsult(t, owner)
	draft := signWithCodes(t, owner, c.encounters[c.a.bPast], "J06.9", "I10")

	if _, err := provider.UpTo(t.Context(), analyticsDiagnoses); err != nil {
		t.Fatal(err)
	}
	rows, err := owner.Query(t.Context(), `SELECT c.field_key || ' ' || c.code || ' ' || c.verification_status || ' ' ||
			(c.patient_id = e.patient_id AND c.organization_id = e.organization_id AND c.practitioner_role_id = d.signer_role_id
			 AND c.recorded_at = d.signed_at AND c.encounter_id = e.id)::text
		FROM condition c JOIN document_draft d ON d.id = c.document_draft_id JOIN encounter e ON e.id = d.encounter_id
		WHERE c.document_draft_id = $1 ORDER BY c.field_key`, draft)
	if err != nil {
		t.Fatal(err)
	}
	got, err := collectStrings(rows)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"diagnosisa J06.9 unconfirmed true", "diagnosisb I10 unconfirmed true"}; !slices.Equal(got, want) {
		t.Errorf("backfilled %v, want %v", got, want)
	}
	var all int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM condition`).Scan(&all); err != nil || all != 2 {
		t.Errorf("%d Conditions in all, %v; want only the signed document's two", all, err)
	}
}

// A clinic sees its own Conditions and inserts them; the patient sees their own; nobody else sees any, and the
// application never changes or deletes one.
func TestAnalyticsDiagnoses_ConditionScopes(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedConsult(t, owner)
	enc := c.encounters[c.a.bPast] // patient A in clinic B
	draft := signWithCodes(t, owner, enc)
	insert := func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id,
				document_draft_id, practitioner_role_id, field_key, code, verification_status)
			SELECT now(), e.organization_id, e.patient_id, e.id, $2, e.practitioner_role_id, 'diagnosis', 'J00', 'confirmed'
			FROM encounter e WHERE e.id = $1`, enc, draft)
		return err
	}
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicB}, insert); err != nil {
		t.Fatalf("clinic B inserts its Condition: %v", err)
	}
	// A Condition of another clinic's or patient's encounter, or of another encounter's document, is refused.
	for name, stmt := range map[string]string{
		"another encounter's document": `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
				practitioner_role_id, field_key, code, verification_status)
			SELECT now(), e.organization_id, e.patient_id, e.id, (SELECT id FROM document_draft WHERE encounter_id <> $1 LIMIT 1),
				e.practitioner_role_id, 'other', 'J00', 'confirmed' FROM encounter e WHERE e.id = $1 AND $2::uuid IS NOT NULL`,
		"another patient": `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
				practitioner_role_id, field_key, code, verification_status)
			SELECT now(), e.organization_id, $2::uuid, e.id, d.id, e.practitioner_role_id, 'other', 'J00', 'confirmed'
			FROM encounter e JOIN document_draft d ON d.encounter_id = e.id WHERE e.id = $1`,
	} {
		_, err := owner.Exec(t.Context(), stmt, enc, c.patients[1].patient)
		if sqlState(err) != "23503" {
			t.Errorf("%s: %v, want a foreign key violation", name, err)
		}
	}
	for name, tt := range map[string]struct {
		scope map[string]string
		want  int
	}{
		"clinic B":            {map[string]string{"app.organization_id": clinicB}, 1},
		"clinic A":            {map[string]string{"app.organization_id": clinicA}, 0},
		"patient A":           {map[string]string{"app.patient_id": c.patients[0].patient}, 1},
		"patient B":           {map[string]string{"app.patient_id": c.patients[1].patient}, 0},
		"patient A in a room": {map[string]string{"app.patient_id": c.patients[0].patient, "app.video_room": "r_x"}, 0},
		"an account":          {map[string]string{"app.account_id": c.patients[0].account}, 0},
		"no scope":            {map[string]string{}, 0},
	} {
		if got := len(idsIn(t, app, tt.scope, "SELECT id::text FROM condition")); got != tt.want {
			t.Errorf("%s sees %d Conditions, want %d", name, got, tt.want)
		}
	}
	for _, stmt := range []string{`UPDATE condition SET code = 'J01'`, `DELETE FROM condition`} {
		err := asApp(t, app, map[string]string{"app.organization_id": clinicB}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s = %v, want permission denied", stmt, err)
		}
	}
}

// salus_stats reads the source columns of every clinic, past row-level security, and nothing else: no name, no
// document field, no write.
func TestAnalyticsDiagnoses_TheStatsRoleReadsOnlyTheSourceColumns(t *testing.T) {
	t.Parallel()
	owner, stats := pgtest.NewStatsDatabase(t)
	c := seedConsult(t, owner)
	signWithCodes(t, owner, c.encounters[c.a.bPast])
	signWithCodes(t, owner, c.encounters[c.a.aPast])
	if _, err := owner.Exec(t.Context(), `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id,
			document_draft_id, practitioner_role_id, field_key, code, verification_status)
		SELECT now(), e.organization_id, e.patient_id, e.id, d.id, e.practitioner_role_id, 'diagnosis', 'J00', 'confirmed'
		FROM document_draft d JOIN encounter e ON e.id = d.encounter_id WHERE d.status = 'final'`); err != nil {
		t.Fatal(err)
	}
	var clinics int
	if err := stats.QueryRow(t.Context(), `SELECT count(DISTINCT organization_id) FROM condition`).Scan(&clinics); err != nil || clinics != 2 {
		t.Errorf("salus_stats sees the Conditions of %d clinics, %v; want both", clinics, err)
	}
	for _, q := range []string{
		`SELECT organization_id, patient_id, practitioner_role_id, recorded_at, code, verification_status FROM condition`,
		`SELECT organization_id, patient_id, practitioner_role_id, period, status FROM appointment`,
		`SELECT id, organization_id, specialty_code FROM practitioner_role`,
		`SELECT id, time_zone FROM organization`,
		`SELECT * FROM diagnosis_group_range`,
		`SELECT * FROM place`,
	} {
		if _, err := stats.Exec(t.Context(), q); err != nil {
			t.Errorf("%s = %v, want it granted", q, err)
		}
	}
	for _, q := range []string{
		`SELECT field_key FROM condition`,
		`SELECT fields FROM document_draft`,
		`SELECT name FROM organization`,
		`SELECT family_name FROM practitioner`,
		`SELECT id FROM patient`,
		`SELECT original FROM transcript_segment`,
		`INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
			practitioner_role_id, field_key, code, verification_status)
			SELECT recorded_at, organization_id, patient_id, encounter_id, document_draft_id, practitioner_role_id, 'x', code, 'confirmed' FROM condition`,
	} {
		if _, err := stats.Exec(t.Context(), q); sqlState(err) != "42501" {
			t.Errorf("%s = %v, want permission denied", q, err)
		}
	}
}

// The seeded reference data: 34 diagnosis groups (27 and 7 subgroups) with a name in each of the ten languages,
// all unreviewed; the places: the Federation and its ten cantons, Republika Srpska and Brcko District, and 143
// municipalities and cities, each in a leaf region.
func TestAnalyticsDiagnoses_SeedsTheGroupsAndThePlaces(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	for q, want := range map[string]int{
		`SELECT count(*) FROM diagnosis_group`:                                                                        34,
		`SELECT count(*) FROM diagnosis_group WHERE parent_code IS NOT NULL`:                                          7,
		`SELECT count(*) FROM diagnosis_group_name`:                                                                   340,
		`SELECT count(*) FROM diagnosis_group_name WHERE review_status <> 'unreviewed'`:                               0,
		`SELECT count(DISTINCT group_code) FROM diagnosis_group_name`:                                                 34,
		`SELECT count(*) FROM place WHERE kind = 'entity'`:                                                            1,
		`SELECT count(*) FROM place WHERE kind = 'region'`:                                                            12,
		`SELECT count(*) FROM place WHERE kind = 'city'`:                                                              143,
		`SELECT count(*) FROM place c JOIN place r ON r.id = c.parent_id WHERE c.kind = 'city' AND r.kind = 'region'`: 143,
		`SELECT count(*) FROM place WHERE kind = 'region' AND parent_id IS NULL`:                                      2,
	} {
		var got int
		if err := db.QueryRow(t.Context(), q).Scan(&got); err != nil || got != want {
			t.Errorf("%s = %d, %v; want %d", q, got, err, want)
		}
	}
}

// gooseProvider returns a goose provider of the embedded migrations on dbURL.
func gooseProvider(t *testing.T, dbURL string) *goose.Provider {
	t.Helper()
	db := stdlib.OpenDBFromPool(open(t, dbURL))
	t.Cleanup(func() { _ = db.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// open returns a pool on dbURL, closed when the test ends.
func open(t *testing.T, dbURL string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(t.Context(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// collectStrings reads every row's one text column.
func collectStrings(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]string, error) {
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s.String)
	}
	return out, rows.Err()
}
