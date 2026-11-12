package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// historyRows is the seeded history of seedHistory: the rows of seedResponses, an encounter per appointment (the
// past ones finished), a signed and an open document per encounter, a clean file per completed response and a
// quarantined one per open response, and patient A's history consent with clinic A (active, sharing nothing).
type historyRows struct {
	encounters map[string]string // by appointment
	final      map[string]string // the signed document of each encounter
	clean      map[string]string // the clean file of each completed response
	r          responseRows
	a          appointmentRows
	consent    string // patient A's history consent with clinic A
	patients   [2]patientRow
}

// seedHistory creates, as the owner, the rows of historyRows.
func seedHistory(t *testing.T, owner *pgxpool.Pool) historyRows {
	t.Helper()
	ctx := t.Context()
	p, a, r := seedResponses(t, owner)
	h := historyRows{patients: p, a: a, r: r, encounters: seedEncounters(t, owner, a), final: map[string]string{}, clean: map[string]string{}}
	if _, err := owner.Exec(ctx, `UPDATE encounter SET status = 'finished', period_end = period_start + interval '20 minutes'
		WHERE appointment_id = ANY($1)`, []string{a.aPast, a.bPast}); err != nil {
		t.Fatal(err)
	}
	for _, clinic := range []string{clinicA, clinicB} {
		if _, err := owner.Exec(ctx, `INSERT INTO document_template (organization_id, title, fields)
			VALUES ($1, 'Placeholder template', '[{"key":"complaint","label":"Placeholder label"}]')`, clinic); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range h.encounters {
		var id string
		if err := owner.QueryRow(ctx, `WITH open AS (
				INSERT INTO document_draft (encounter_id, template_id, fields)
				SELECT e.id, t.id, '[{"key":"a","missing":true}]' FROM encounter e JOIN document_template t ON t.organization_id = e.organization_id
				WHERE e.id = $1)
			INSERT INTO document_draft (encounter_id, template_id, status, fields)
			SELECT e.id, t.id, 'final', '[{"key":"a","missing":true}]' FROM encounter e JOIN document_template t ON t.organization_id = e.organization_id
			WHERE e.id = $1 RETURNING id`, e).Scan(&id); err != nil {
			t.Fatal(err)
		}
		h.final[e] = id
	}
	for appt, response := range r.completed {
		var id string
		if err := owner.QueryRow(ctx, insertFile, response).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := owner.Exec(ctx, `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		h.clean[appt] = id
	}
	for _, response := range r.open {
		if _, err := owner.Exec(ctx, insertFile, response); err != nil {
			t.Fatal(err)
		}
	}
	if err := owner.QueryRow(ctx, `INSERT INTO history_consent (patient_id, organization_id, status, date_time)
		VALUES ($1, $2, 'active', '2026-11-01T08:00:00Z') RETURNING id`, p[0].patient, clinicA).Scan(&h.consent); err != nil {
		t.Fatal(err)
	}
	return h
}

// share adds, as the owner, the encounters to the history consent's selection.
func share(t *testing.T, owner *pgxpool.Pool, consent string, encounters ...string) {
	t.Helper()
	for _, e := range encounters {
		if _, err := owner.Exec(t.Context(), `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at)
			VALUES ($1, $2, now())`, consent, e); err != nil {
			t.Fatal(err)
		}
	}
}

// ownerExec runs stmt as the owner.
func ownerExec(t *testing.T, owner *pgxpool.Pool, stmt string, args ...any) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), stmt, args...); err != nil {
		t.Fatal(err)
	}
}

// sharedQueries are what clinic A's scope may see of another clinic's visit once it is shared: the encounter, its
// signed document, the completed response, its clean file, the visit's clinic and practitioner role. Each query
// gives only rows of clinic B, so clinic A's own rows (unchanged by sharing) are left out.
var sharedQueries = map[string]string{
	"encounter":              `SELECT id::text FROM encounter WHERE organization_id = '` + clinicB + `'`,
	"document_draft":         `SELECT d.id::text FROM document_draft d WHERE d.encounter_id IN (SELECT id FROM encounter WHERE organization_id = '` + clinicB + `')`,
	"questionnaire_response": `SELECT id::text FROM questionnaire_response WHERE organization_id = '` + clinicB + `'`,
	"binary":                 `SELECT id::text FROM "binary" WHERE organization_id = '` + clinicB + `'`,
	"organization":           `SELECT id::text FROM organization WHERE id = '` + clinicB + `'`,
	"practitioner_role":      `SELECT id::text FROM practitioner_role WHERE organization_id = '` + clinicB + `'`,
}

// sharedWant is what clinic A sees of clinic B's visit bPast when it is shared.
func sharedWant(h historyRows) map[string][]string {
	e := h.encounters[h.a.bPast]
	return map[string][]string{
		"encounter":              {e},
		"document_draft":         {h.final[e]},
		"questionnaire_response": {h.r.completed[h.a.bPast]},
		"binary":                 {h.clean[h.a.bPast]},
		"organization":           {clinicB},
		"practitioner_role":      {h.r.roles[clinicB]},
	}
}

// checkShared compares what clinic A's scope sees of clinic B's rows with want (nothing where want has no entry).
func checkShared(t *testing.T, app *pgxpool.Pool, state string, want map[string][]string) {
	t.Helper()
	for table, query := range sharedQueries {
		got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, query)
		if w := sorted(want[table]...); !slices.Equal(got, w) {
			t.Errorf("%s: clinic A sees %s %v of clinic B, want %v", state, table, got, w)
		}
	}
}

// ADR 0013: another clinic's visit is seen by a clinic only while the patient's history consent with it is active,
// lists the visit, and their care consent with it is active; a removal or a revocation ends it when it commits.
func TestHistorySharing_AClinicSeesAnotherClinicsVisitOnlyWhileItIsShared(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	e := h.encounters[h.a.bPast]

	checkShared(t, app, "nothing shared", nil)

	share(t, owner, h.consent, e)
	checkShared(t, app, "shared", sharedWant(h))

	ownerExec(t, owner, `UPDATE history_consent_encounter SET removed_at = now() WHERE consent_id = $1`, h.consent)
	checkShared(t, app, "removed", nil)

	share(t, owner, h.consent, e)
	ownerExec(t, owner, `UPDATE history_consent SET status = 'inactive', revoked_at = now() WHERE id = $1`, h.consent)
	checkShared(t, app, "history consent revoked", nil)

	ownerExec(t, owner, `UPDATE history_consent SET status = 'active', revoked_at = NULL WHERE id = $1`, h.consent)
	checkShared(t, app, "history consent active again", sharedWant(h))
	ownerExec(t, owner, `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1 AND organization_id = $2`,
		h.patients[0].patient, clinicA)
	checkShared(t, app, "care consent with clinic A revoked", nil)
}

// Only a finished visit is shared, and of it only what is final: a signed document (never a draft), a completed
// response (never one in progress) and a clean file (never one in quarantine). Another clinic's in-progress visit
// can't be listed at all (TestHistorySharing_ThePatientSharesOnlyTheirOwnFinishedVisitsAtOtherClinics).
func TestHistorySharing_OnlyWhatIsFinalOfAVisitIsShared(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	share(t, owner, h.consent, h.encounters[h.a.bPast])
	open := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, `SELECT d.id::text FROM document_draft d
		JOIN encounter e ON e.id = d.encounter_id WHERE e.organization_id = '`+clinicB+`' AND d.status <> 'final'`)
	if len(open) != 0 {
		t.Errorf("clinic A sees the open drafts %v of the shared visit, want none", open)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA},
		`SELECT id::text FROM questionnaire_response WHERE organization_id = '`+clinicB+`' AND status <> 'completed'`); len(got) != 0 {
		t.Errorf("clinic A sees the open responses %v of the shared visit, want none", got)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA},
		`SELECT id::text FROM "binary" WHERE organization_id = '`+clinicB+`' AND status <> 'clean'`); len(got) != 0 {
		t.Errorf("clinic A sees the unscanned files %v of the shared visit, want none", got)
	}
	// Clinic B's view of its own visit is unchanged, and patient B's scope sees nothing of patient A.
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, `SELECT id::text FROM encounter`); !slices.Equal(got, sorted(h.encounters[h.a.bPast])) {
		t.Errorf("clinic B sees the encounters %v, want its past one only", got)
	}
	if got := idsIn(t, app, map[string]string{"app.patient_id": h.patients[1].patient},
		`SELECT id::text FROM encounter WHERE patient_id = '`+h.patients[0].patient+`'`); len(got) != 0 {
		t.Errorf("patient B sees patient A's encounters %v", got)
	}
}

// The patient's own scope sees, grants, changes and revokes their history consents and their selections; a clinic
// reads those given to it; no one else sees any.
func TestHistoryConsent_EachScopeSeesOnlyItsConsentsAndSelections(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	share(t, owner, h.consent, h.encounters[h.a.bPast])
	for name, tt := range map[string]struct {
		scope map[string]string
		want  []string
	}{
		"no scope":   {nil, nil},
		"patient A":  {map[string]string{"app.patient_id": h.patients[0].patient}, sorted(h.consent)},
		"patient B":  {map[string]string{"app.patient_id": h.patients[1].patient}, nil},
		"clinic A":   {map[string]string{"app.organization_id": clinicA}, sorted(h.consent)},
		"clinic B":   {map[string]string{"app.organization_id": clinicB}, nil},
		"an account": {map[string]string{"app.account_id": h.patients[0].account, "app.organization_id": clinicA}, nil},
	} {
		if got := idsIn(t, app, tt.scope, `SELECT id::text FROM history_consent`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees the history consents %v, want %v", name, got, tt.want)
		}
		if got := idsIn(t, app, tt.scope, `SELECT consent_id::text FROM history_consent_encounter`); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees the selections of %v, want %v", name, got, tt.want)
		}
	}
	clinic := map[string]string{"app.organization_id": clinicA}
	for name, stmt := range map[string]string{
		"grant":  `INSERT INTO history_consent (patient_id, organization_id, status, date_time) VALUES ('` + h.patients[1].patient + `', '` + clinicA + `', 'active', now())`,
		"share":  `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at) VALUES ('` + h.consent + `', '` + h.encounters[h.a.aPast] + `', now())`,
		"remove": `UPDATE history_consent_encounter SET removed_at = now()`,
		"revoke": `UPDATE history_consent SET status = 'inactive', revoked_at = now()`,
	} {
		// A refusal is an error (row-level security on an insert) or no row changed (on an update).
		_ = asApp(t, app, clinic, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			if err == nil && tag.RowsAffected() != 0 {
				t.Errorf("clinic A could %s (%d rows)", name, tag.RowsAffected())
			}
			return err
		})
	}
}

// The patient lists only their own finished visits at other clinics than the consent's: not one at the clinic
// itself, not one in progress, not another patient's. Row-level security refuses the others, so the service's check
// is not the only one.
func TestHistorySharing_ThePatientSharesOnlyTheirOwnFinishedVisitsAtOtherClinics(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	patientA := map[string]string{"app.patient_id": h.patients[0].patient}
	var other string // patient B's history consent with clinic B, for another patient's attempt
	if err := owner.QueryRow(t.Context(), `INSERT INTO history_consent (patient_id, organization_id, status, date_time)
		VALUES ($1, $2, 'active', now()) RETURNING id`, h.patients[1].patient, clinicB).Scan(&other); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, consent, encounter string
		ok                       bool
	}{
		{"a finished visit at clinic B", h.consent, h.encounters[h.a.bPast], true},
		{"a visit at the consent's own clinic", h.consent, h.encounters[h.a.aPast], false},
		{"a visit in progress", h.consent, h.encounters[h.a.bFuture], false},
		{"another patient's visit", h.consent, h.encounters[h.a.other], false},
		{"into another patient's consent", other, h.encounters[h.a.bPast], false},
	}
	for _, tt := range tests {
		err := asApp(t, app, patientA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at) VALUES ($1, $2, now())`,
				tt.consent, tt.encounter)
			return err
		})
		if (err == nil) != tt.ok {
			t.Errorf("%s: err %v, want ok %v", tt.name, err, tt.ok)
		}
	}
	// The patient removes their selection and revokes their consent in their own scope.
	err := asApp(t, app, patientA, func(ctx context.Context, db postgres.Querier) error {
		if _, err := db.Exec(ctx, `UPDATE history_consent_encounter SET removed_at = now() WHERE consent_id = $1`, h.consent); err != nil {
			return err
		}
		tag, err := db.Exec(ctx, `UPDATE history_consent SET status = 'inactive', revoked_at = now(), version_id = version_id + 1 WHERE id = $1`, h.consent)
		if err == nil && tag.RowsAffected() != 1 {
			t.Errorf("revoked %d consents, want 1", tag.RowsAffected())
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// One open selection of a visit per consent.
	share(t, owner, h.consent, h.encounters[h.a.bPast])
	if _, err := owner.Exec(t.Context(), `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at) VALUES ($1, $2, now())`,
		h.consent, h.encounters[h.a.bPast]); sqlState(err) != "23505" {
		t.Errorf("a second open selection of one visit: %v, want unique_violation", err)
	}
}

// entryRows is one patient-history entry table's seeded rows: patient A's own entry, the entries clinics A and B
// recorded in patient A's past visits with them, and patient B's own entry.
type entryRows struct {
	own, atA, atB, otherPatient string
}

// entryTables are the two profile tables with an insert of one synthetic row each: $1 the patient, $2 the clinic,
// $3 the encounter, $4 the practitioner role (all but $1 null for the patient's own entry), $5 in the profile.
var entryTables = map[string]string{
	"allergy_intolerance": `INSERT INTO allergy_intolerance (patient_id, organization_id, encounter_id, recorder_role_id, in_profile,
			code_text, verification_status, recorded_date, confirmed_at, confirmed_by_role_id, confirmed_organization_id)
		VALUES ($1, $2, $3, $4, $5, 'Placeholder substance', CASE WHEN $2::uuid IS NULL THEN 'unconfirmed' ELSE 'confirmed' END, now(),
			CASE WHEN $2::uuid IS NOT NULL THEN now() END, $4, $2) RETURNING id`,
	"medication_statement": `INSERT INTO medication_statement (patient_id, organization_id, encounter_id, recorder_role_id, in_profile,
			medication_text, date_asserted, confirmed_at, confirmed_by_role_id, confirmed_organization_id)
		VALUES ($1, $2, $3, $4, $5, 'Placeholder medication', now(), CASE WHEN $2::uuid IS NOT NULL THEN now() END, $4, $2) RETURNING id`,
}

func seedEntries(t *testing.T, owner *pgxpool.Pool, h historyRows, insert string) entryRows {
	t.Helper()
	var r entryRows
	for _, row := range []struct {
		id                *string
		clinic, enc, role any
		patient           string
		inProfile         bool
	}{
		{&r.own, nil, nil, nil, h.patients[0].patient, true},
		{&r.atA, clinicA, h.encounters[h.a.aPast], h.r.roles[clinicA], h.patients[0].patient, false},
		{&r.atB, clinicB, h.encounters[h.a.bPast], h.r.roles[clinicB], h.patients[0].patient, false},
		{&r.otherPatient, nil, nil, nil, h.patients[1].patient, true},
	} {
		if err := owner.QueryRow(t.Context(), insert, row.patient, row.clinic, row.enc, row.role, row.inProfile).Scan(row.id); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// The visit-scoped rule (ADR 0013): the patient sees all their entries; a clinic sees a patient's profile entries
// while their care consent with it is active, and an entry recorded in a visit wherever it sees the visit: its own
// clinic's, and another clinic's only while it is shared. Who of the clinic may see which is the service's rule.
func TestProfileEntries_EachScopeSeesTheEntriesOfItsVisitsAndTheProfile(t *testing.T) {
	t.Parallel()
	for table, insert := range entryTables {
		t.Run(table, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, insert)
			query := `SELECT id::text FROM ` + table
			for name, tt := range map[string]struct {
				scope map[string]string
				want  []string
			}{
				"no scope":   {nil, nil},
				"patient A":  {map[string]string{"app.patient_id": h.patients[0].patient}, sorted(r.own, r.atA, r.atB)},
				"patient B":  {map[string]string{"app.patient_id": h.patients[1].patient}, sorted(r.otherPatient)},
				"clinic A":   {map[string]string{"app.organization_id": clinicA}, sorted(r.own, r.atA)},
				"clinic B":   {map[string]string{"app.organization_id": clinicB}, sorted(r.atB)}, // revoked: its past visit only
				"an account": {map[string]string{"app.account_id": h.patients[0].account}, nil},
			} {
				if got := idsIn(t, app, tt.scope, query); !slices.Equal(got, tt.want) {
					t.Errorf("%s: sees %v, want %v", name, got, tt.want)
				}
			}
			share(t, owner, h.consent, h.encounters[h.a.bPast])
			if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, query); !slices.Equal(got, sorted(r.own, r.atA, r.atB)) {
				t.Errorf("clinic A with the visit at B shared: sees %v, want the profile, its own and the shared visit's", got)
			}
			// Adding an entry of the shared visit to the profile does not open it to a clinic without an active consent.
			ownerExec(t, owner, `UPDATE `+table+` SET in_profile = true WHERE id = $1`, r.atA)
			if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, query); !slices.Equal(got, sorted(r.atB)) {
				t.Errorf("clinic B (revoked) sees %v, want its visit's entry only", got)
			}
		})
	}
}

// The patient writes their own profile entries and removes the unconfirmed ones; a clinic records entries only in
// its own visits, never in the profile directly, and confirms the entries it sees.
func TestProfileEntries_EachScopeWritesOnlyWhatItMay(t *testing.T) {
	t.Parallel()
	for table, insert := range entryTables {
		t.Run(table, func(t *testing.T) {
			t.Parallel()
			owner, app := pgtest.NewAppDatabase(t)
			h := seedHistory(t, owner)
			r := seedEntries(t, owner, h, insert)
			patientA := map[string]string{"app.patient_id": h.patients[0].patient}
			clinic := map[string]string{"app.organization_id": clinicA}
			tests := []struct {
				scope map[string]string
				name  string
				args  []any
				ok    bool
			}{
				{patientA, "the patient's own entry", []any{h.patients[0].patient, nil, nil, nil, true}, true},
				{patientA, "the patient as a clinic", []any{h.patients[0].patient, clinicA, h.encounters[h.a.aPast], h.r.roles[clinicA], false}, false},
				{patientA, "another patient's entry", []any{h.patients[1].patient, nil, nil, nil, true}, false},
				{clinic, "in the clinic's own visit", []any{h.patients[0].patient, clinicA, h.encounters[h.a.aPast], h.r.roles[clinicA], false}, true},
				{clinic, "in another clinic's visit", []any{h.patients[0].patient, clinicA, h.encounters[h.a.bPast], h.r.roles[clinicA], false}, false},
				{clinic, "straight into the profile", []any{h.patients[0].patient, clinicA, h.encounters[h.a.aPast], h.r.roles[clinicA], true}, false},
				{clinic, "as the patient", []any{h.patients[0].patient, nil, nil, nil, true}, false},
			}
			for _, tt := range tests {
				err := asApp(t, app, tt.scope, func(ctx context.Context, db postgres.Querier) error {
					_, err := db.Exec(ctx, insert, tt.args...)
					return err
				})
				if (err == nil) != tt.ok {
					t.Errorf("%s: err %v, want ok %v", tt.name, err, tt.ok)
				}
			}
			// The clinic confirms the patient's own entry; the patient can no longer delete it, nor a clinic's.
			err := asApp(t, app, clinic, func(ctx context.Context, db postgres.Querier) error {
				tag, err := db.Exec(ctx, `UPDATE `+table+` SET confirmed_at = now(), confirmed_by_role_id = $2, confirmed_organization_id = $3,
					version_id = version_id + 1 WHERE id = $1`, r.own, h.r.roles[clinicA], clinicA)
				if err == nil && tag.RowsAffected() != 1 {
					t.Errorf("the clinic confirmed %d entries, want 1", tag.RowsAffected())
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			err = asApp(t, app, patientA, func(ctx context.Context, db postgres.Querier) error {
				tag, err := db.Exec(ctx, `DELETE FROM `+table+` WHERE id = ANY($1)`, []string{r.own, r.atA})
				if err == nil && tag.RowsAffected() != 0 {
					t.Errorf("the patient deleted %d confirmed or recorded entries, want none", tag.RowsAffected())
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The tables hold a consistent state: an entry is the patient's own or a clinic's from a visit, never half of
// each; a confirmation is whole or absent; codes and texts are from their value sets.
func TestPatientHistory_TablesHoldAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	pa, enc := h.patients[0].patient, h.encounters[h.a.aPast]
	for _, tt := range []struct {
		name, stmt string
		args       []any
	}{
		{"an allergy with a clinic and no visit", `INSERT INTO allergy_intolerance (patient_id, organization_id, in_profile, code_text, verification_status, recorded_date)
			VALUES ($1, $2, false, 'x', 'confirmed', now())`, []any{pa, clinicA}},
		{"a clinic's allergy without its recorder", `INSERT INTO allergy_intolerance (patient_id, organization_id, encounter_id, in_profile, code_text, verification_status, recorded_date)
			VALUES ($1, $2, $3, false, 'x', 'confirmed', now())`, []any{pa, clinicA, enc}},
		{"the patient's allergy out of the profile", `INSERT INTO allergy_intolerance (patient_id, in_profile, code_text, verification_status, recorded_date)
			VALUES ($1, false, 'x', 'unconfirmed', now())`, []any{pa}},
		{"a blank substance", `INSERT INTO allergy_intolerance (patient_id, code_text, verification_status, recorded_date) VALUES ($1, ' ', 'unconfirmed', now())`, []any{pa}},
		{"an unknown severity", `INSERT INTO allergy_intolerance (patient_id, code_text, reaction_text, severity, verification_status, recorded_date)
			VALUES ($1, 'x', 'Rash', 'fatal', 'unconfirmed', now())`, []any{pa}},
		{"a severity without a reaction", `INSERT INTO allergy_intolerance (patient_id, code_text, severity, verification_status, recorded_date)
			VALUES ($1, 'x', 'mild', 'unconfirmed', now())`, []any{pa}},
		{"an unknown category", `INSERT INTO allergy_intolerance (patient_id, code_text, category, verification_status, recorded_date)
			VALUES ($1, 'x', 'pollen', 'unconfirmed', now())`, []any{pa}},
		{"refuted (not in the value set we use)", `INSERT INTO allergy_intolerance (patient_id, code_text, verification_status, recorded_date)
			VALUES ($1, 'x', 'refuted', now())`, []any{pa}},
		{"confirmed without a confirmation", `INSERT INTO allergy_intolerance (patient_id, code_text, verification_status, recorded_date)
			VALUES ($1, 'x', 'confirmed', now())`, []any{pa}},
		{"half a confirmation", `INSERT INTO medication_statement (patient_id, medication_text, date_asserted, confirmed_at) VALUES ($1, 'x', now(), now())`, []any{pa}},
		{"a dose without a unit", `INSERT INTO medication_statement (patient_id, medication_text, date_asserted, dose_value) VALUES ($1, 'x', now(), 500)`, []any{pa}},
		{"a unit outside UCUM's list", `INSERT INTO medication_statement (patient_id, medication_text, date_asserted, dose_value, dose_unit)
			VALUES ($1, 'x', now(), 1, 'spoon')`, []any{pa}},
		{"a frequency without its period", `INSERT INTO medication_statement (patient_id, medication_text, date_asserted, frequency) VALUES ($1, 'x', now(), 2)`, []any{pa}},
		{"an unknown status", `INSERT INTO medication_statement (patient_id, medication_text, date_asserted, status) VALUES ($1, 'x', now(), 'paused')`, []any{pa}},
		{"a clinic's medication without its recorder", `INSERT INTO medication_statement (patient_id, organization_id, encounter_id, in_profile, medication_text, date_asserted)
			VALUES ($1, $2, $3, false, 'x', now())`, []any{pa, clinicA, enc}},
		{"a selection removed before it was added", `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at, removed_at)
			VALUES ($1, $2, now(), now() - interval '1 day')`, []any{h.consent, h.encounters[h.a.bPast]}},
		{"a second active history consent", `INSERT INTO history_consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
			[]any{pa, clinicA}},
		{"an inactive history consent without its revocation", `INSERT INTO history_consent (patient_id, organization_id, status, date_time)
			VALUES ($1, $2, 'inactive', now())`, []any{pa, clinicB}},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, tt.args...); err == nil {
			t.Errorf("%s: accepted", tt.name)
		}
	}
}
