package consent_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

// seedEncounters creates, as the owner, a visit of patient A for each status at clinic (a practitioner, a role, the
// appointment and its encounter), and returns their encounters.
func seedEncounters(t *testing.T, owner *pgxpool.Pool, clinic string, statuses ...string) []string {
	t.Helper()
	ctx := t.Context()
	var role string
	if err := owner.QueryRow(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			VALUES ($1, 'Ada', 'Example') RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id`,
		clinic).Scan(&role); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(statuses))
	for i, status := range statuses {
		start := granted.Add(-time.Duration(i+1) * 24 * time.Hour)
		end := pgtype.Timestamptz{Time: start.Add(20 * time.Minute), Valid: status == "finished"}
		if err := owner.QueryRow(ctx, `WITH a AS (
				INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), 'fulfilled')
				RETURNING id, patient_id, practitioner_role_id, organization_id, lower(period) AS start)
			INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
			SELECT id, organization_id, patient_id, practitioner_role_id, $5, start, $6 FROM a
			RETURNING id`, patientA, role, clinic, start, status, end).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// GrantHistory, ChangeHistory and RevokeHistory as patient A, and what clinic A's scope sees after each: a removal
// and a revocation end the clinic's view of the visit when they commit.
func TestHistory_GrantChangeAndRevokeTakeEffectAtOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	visits := seedEncounters(t, owner, clinicB, "finished", "finished", "finished")
	repo := consent.NewPostgresRepository(app)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }
	inA := func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }
	clinicSees := func() []string {
		t.Helper()
		var st access.Consent
		if err := in(t, app, inA, func(ctx context.Context) (err error) { st, err = repo.State(ctx, patientA, clinicA); return }); err != nil {
			t.Fatal(err)
		}
		return st.Shared
	}

	var h consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) (err error) {
		h, err = repo.GrantHistory(ctx, patientA, clinicA, []string{visits[0], visits[1]}, granted)
		return
	}); err != nil {
		t.Fatal(err)
	}
	if h.Kind != consent.KindHistory || h.Status != consent.StatusActive || h.VersionID != 1 || !h.DateTime.Equal(granted) || len(h.Shared) != 2 {
		t.Fatalf("GrantHistory() = %+v, want an active history consent at version 1 listing two visits", h)
	}
	if got := clinicSees(); len(got) != 2 {
		t.Errorf("after the grant clinic A sees %v, want two visits", got)
	}
	if err := in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.GrantHistory(ctx, patientA, clinicA, []string{visits[2]}, granted)
		return err
	}); !errors.Is(err, consent.ErrActive) {
		t.Errorf("a second GrantHistory() = %v, want ErrActive", err)
	}

	later := granted.Add(time.Hour)
	if err := in(t, app, asA, func(ctx context.Context) (err error) {
		h, err = repo.ChangeHistory(ctx, h.ID, 1, []string{visits[2]}, []string{visits[0]}, later)
		return
	}); err != nil {
		t.Fatal(err)
	}
	if want := []string{visits[1], visits[2]}; h.VersionID != 2 || !h.LastUpdated.Equal(later) || !slices.Equal(ids(h), want) || !h.Shared[1].AddedAt.Equal(later) {
		t.Errorf("ChangeHistory() = %+v, want version 2 listing %v, visit 3 added later", h, want)
	}
	if got, want := clinicSees(), []string{visits[1], visits[2]}; !slices.Equal(got, want) {
		t.Errorf("after the change clinic A sees %v, want %v", got, want)
	}
	if err := in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.ChangeHistory(ctx, h.ID, 1, []string{visits[0]}, nil, later)
		return err
	}); !errors.Is(err, consent.ErrVersionConflict) {
		t.Errorf("ChangeHistory() at a stale version = %v, want ErrVersionConflict", err)
	}

	var active consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) (err error) { active, err = repo.ActiveHistory(ctx, patientA, clinicA); return }); err != nil ||
		active.ID != h.ID || !slices.Equal(ids(active), ids(h)) {
		t.Errorf("ActiveHistory() = %+v, %v; want %+v", active, err, h)
	}
	var revoked consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) (err error) { revoked, err = repo.RevokeHistory(ctx, patientA, clinicA, later); return }); err != nil ||
		revoked.Status != consent.StatusInactive || !revoked.RevokedAt.Equal(later) {
		t.Fatalf("RevokeHistory() = %+v, %v; want it inactive", revoked, err)
	}
	if got := clinicSees(); len(got) != 0 {
		t.Errorf("after the revocation clinic A sees %v, want nothing", got)
	}
	if err := in(t, app, asA, func(ctx context.Context) error { _, err := repo.ActiveHistory(ctx, patientA, clinicA); return err }); !errors.Is(err, consent.ErrNotFound) {
		t.Errorf("ActiveHistory() after the revocation = %v, want ErrNotFound", err)
	}
	if err := in(t, app, asA, func(ctx context.Context) error { _, err := repo.RevokeHistory(ctx, patientA, clinicA, later); return err }); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("a second RevokeHistory() = %v, want ErrInactive", err)
	}
	if err := in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.ChangeHistory(ctx, h.ID, revoked.VersionID, []string{visits[0]}, nil, later)
		return err
	}); !errors.Is(err, consent.ErrVersionConflict) {
		t.Errorf("ChangeHistory() after the revocation = %v, want ErrVersionConflict", err)
	}
}

// Row-level security refuses a visit at the receiving clinic itself, one not finished, and another patient's.
func TestGrantHistory_RefusesAVisitThatIsNotAFinishedOneAtAnotherClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	atA := seedEncounters(t, owner, clinicA, "finished")
	atB := seedEncounters(t, owner, clinicB, "in-progress")
	repo := consent.NewPostgresRepository(app)
	for name, tt := range map[string]struct {
		patient string
		visit   string
	}{
		"at the receiving clinic": {patientA, atA[0]},
		"not finished":            {patientA, atB[0]},
		"another patient's":       {patientB, "e9000000-0000-4000-8000-0000000000e9"},
	} {
		err := in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, tt.patient) }, func(ctx context.Context) error {
			_, err := repo.GrantHistory(ctx, tt.patient, clinicA, []string{tt.visit}, granted)
			return err
		})
		if !errors.Is(err, consent.ErrInvalidVisit) {
			t.Errorf("%s: GrantHistory() = %v, want ErrInvalidVisit", name, err)
		}
	}
	err := in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }, func(ctx context.Context) error {
		_, err := repo.GrantHistory(ctx, patientA, "f0000000-0000-4000-8000-00000000000f", []string{atB[0]}, granted)
		return err
	})
	if !errors.Is(err, consent.ErrClinicNotFound) {
		t.Errorf("GrantHistory() to an unknown clinic = %v, want ErrClinicNotFound", err)
	}
}

// ByID, ByPatient and Revoke know the history kind: the patient sees theirs with the visits it lists; the receiving
// clinic reads the active one.
func TestHistory_IsAConsentOfTheThirdKind(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	historyConsent, sharedVisit, _ := seedVisits(t, owner)
	repo := consent.NewPostgresRepository(app)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }
	for name, set := range map[string]func(ctx context.Context) error{
		"patient A": asA, "clinic A": func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) },
	} {
		var got consent.Consent
		if err := in(t, app, set, func(ctx context.Context) (err error) { got, err = repo.ByID(ctx, historyConsent); return }); err != nil ||
			got.Kind != consent.KindHistory || !slices.Equal(ids(got), []string{sharedVisit}) || !got.Shared[0].AddedAt.Equal(granted) {
			t.Errorf("%s: ByID() = %+v, %v; want the history consent listing %s", name, got, err, sharedVisit)
		}
	}
	var all []consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) (err error) { all, err = repo.ByPatient(ctx, patientA); return }); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(all, func(c consent.Consent) bool { return c.ID == historyConsent })
	if len(all) != 3 || i < 0 || !slices.Equal(ids(all[i]), []string{sharedVisit}) {
		t.Errorf("ByPatient() = %+v, want the care consents and the history consent with its visit", all)
	}
	var revoked consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) (err error) { revoked, err = repo.Revoke(ctx, consent.KindHistory, historyConsent, granted); return }); err != nil ||
		revoked.Kind != consent.KindHistory || revoked.Status != consent.StatusInactive || revoked.Shared != nil {
		t.Errorf("Revoke(history) = %+v, %v; want it inactive, listing nothing", revoked, err)
	}
	if err := in(t, app, func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, func(ctx context.Context) error {
		_, err := repo.ByID(ctx, historyConsent)
		return err
	}); !errors.Is(err, consent.ErrNotFound) {
		t.Errorf("clinic A: ByID() of the revoked history consent = %v, want ErrNotFound", err)
	}
}

func ids(c consent.Consent) []string {
	var out []string
	for _, v := range c.Shared {
		out = append(out, v.EncounterID)
	}
	return out
}
