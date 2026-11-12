package consent_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedVisits creates, as the owner, two finished visits of patient A at clinic B (a practitioner, a role, the
// appointments and their encounters) and patient A's active history consent with clinic A listing the first and,
// removed, the second. It returns the consent and the two encounters.
func seedVisits(t *testing.T, owner *pgxpool.Pool) (historyConsent, shared, removed string) {
	t.Helper()
	ctx := t.Context()
	var role string
	if err := owner.QueryRow(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			VALUES ($1, 'Ada', 'Example') RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id`,
		clinicB).Scan(&role); err != nil {
		t.Fatal(err)
	}
	var encounters [2]string
	for i, start := range []string{"2026-10-20T09:00:00Z", "2026-10-21T09:00:00Z"} {
		if err := owner.QueryRow(ctx, `WITH a AS (
				INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), 'fulfilled')
				RETURNING id, patient_id, practitioner_role_id, organization_id, lower(period) AS start)
			INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', start, start + interval '20 minutes' FROM a
			RETURNING id`, patientA, role, clinicB, start).Scan(&encounters[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := owner.QueryRow(ctx, `INSERT INTO history_consent (patient_id, organization_id, status, date_time)
		VALUES ($1, $2, 'active', $3) RETURNING id`, patientA, clinicA, granted).Scan(&historyConsent); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at, removed_at)
		VALUES ($1, $2, $4, NULL), ($1, $3, $4, $4)`, historyConsent, encounters[0], encounters[1], granted); err != nil {
		t.Fatal(err)
	}
	return historyConsent, encounters[0], encounters[1]
}

// State gives the history consent and the visits it lists now (not the removed ones), to the receiving clinic and to
// the patient; another clinic's scope sees none of it.
func TestState_GivesTheHistoryConsentAndItsOpenSelection(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	historyConsent, shared, _ := seedVisits(t, owner)
	repo := consent.NewPostgresRepository(app)
	state := func(set func(ctx context.Context) error, clinic string) access.Consent {
		t.Helper()
		var st access.Consent
		if err := in(t, app, set, func(ctx context.Context) error {
			var err error
			st, err = repo.State(ctx, patientA, clinic)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return st
	}
	inClinic := func(id string) func(ctx context.Context) error {
		return func(ctx context.Context) error { return repo.SetClinic(ctx, id) }
	}
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }

	for name, st := range map[string]access.Consent{"clinic A": state(inClinic(clinicA), clinicA), "patient A": state(asA, clinicA)} {
		if !st.Active || !st.History || !slices.Equal(st.Shared, []string{shared}) {
			t.Errorf("%s: State() = %+v, want the care and history consents active, sharing %s only", name, st, shared)
		}
	}
	if st := state(inClinic(clinicB), clinicA); st.History || len(st.Shared) != 0 {
		t.Errorf("clinic B's scope: State(clinic A) = %+v, want no history consent", st)
	}
	if st := state(inClinic(clinicB), clinicB); st.History || len(st.Shared) != 0 {
		t.Errorf("clinic B: State() = %+v, want no history consent (none was given to it)", st)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE history_consent SET status = 'inactive', revoked_at = now() WHERE id = $1`, historyConsent); err != nil {
		t.Fatal(err)
	}
	if st := state(inClinic(clinicA), clinicA); st.History || len(st.Shared) != 0 || !st.Active {
		t.Errorf("after the revocation: State() = %+v, want the care consent only", st)
	}
}
