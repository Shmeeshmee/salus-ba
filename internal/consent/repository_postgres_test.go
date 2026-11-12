package consent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// Synthetic ids only.
const (
	accountA  = "c1000000-0000-4000-8000-0000000000c1"
	accountB  = "c2000000-0000-4000-8000-0000000000c2"
	patientA  = "d1000000-0000-4000-8000-0000000000d1"
	patientB  = "d2000000-0000-4000-8000-0000000000d2"
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	consentAA = "f1000000-0000-4000-8000-0000000000f1" // patient A with clinic A, active
	consentAB = "f2000000-0000-4000-8000-0000000000f2" // patient A with clinic B, revoked
)

var granted = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

// seed creates, as the owner, clinics A and B and patients A and B, with patient A's consent with clinic A
// active (consentAA) and with clinic B revoked (consentAB), granted at granted and revoked a day later.
func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	stmts := []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')", []any{clinicA, clinicB}},
		{"INSERT INTO account (id, email, password_hash, role) VALUES ($1, 'pat.a@example.test', 'h', 'patient'), ($2, 'pat.b@example.test', 'h', 'patient')", []any{accountA, accountB}},
		{"INSERT INTO patient (id, account_id, given_name, family_name, birth_date) VALUES ($1, $2, 'Ada', 'Example', '1990-01-01'), ($3, $4, 'Bo', 'Example', '1991-02-02')", []any{patientA, accountA, patientB, accountB}},
		{`INSERT INTO consent (id, patient_id, organization_id, status, date_time, revoked_at, updated_at, version_id) VALUES
			($1, $3, $4, 'active', $6, NULL, $6, 1), ($2, $3, $5, 'inactive', $6, $7, $7, 2)`,
			[]any{consentAA, consentAB, patientA, clinicA, clinicB, granted, granted.Add(24 * time.Hour)}},
	}
	for _, s := range stmts {
		if _, err := owner.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// in runs fn in a transaction on app scoped by set.
func in(t *testing.T, app *pgxpool.Pool, set func(ctx context.Context) error, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := set(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

var (
	activeAA  = consent.Consent{ID: consentAA, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindCare, Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1}
	revokedAB = consent.Consent{ID: consentAB, PatientID: patientA, OrganizationID: clinicB, Kind: consent.KindCare, Status: consent.StatusInactive, DateTime: granted,
		RevokedAt: granted.Add(24 * time.Hour), LastUpdated: granted.Add(24 * time.Hour), VersionID: 2}
)

func same(a, b consent.Consent) bool {
	return a.ID == b.ID && a.Kind == b.Kind && a.PatientID == b.PatientID && a.OrganizationID == b.OrganizationID && a.Status == b.Status &&
		a.DateTime.Equal(b.DateTime) && a.RevokedAt.Equal(b.RevokedAt) && a.LastUpdated.Equal(b.LastUpdated) && a.VersionID == b.VersionID
}

func TestByIDAndByPatient_SeeOnlyTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)

	var byID consent.Consent
	var mine []consent.Consent
	err := in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }, func(ctx context.Context) error {
		var err error
		if byID, err = repo.ByID(ctx, consentAB); err != nil {
			return err
		}
		mine, err = repo.ByPatient(ctx, patientA)
		return err
	})
	if err != nil || !same(byID, revokedAB) || len(mine) != 2 || !same(mine[0], activeAA) && !same(mine[0], revokedAB) {
		t.Fatalf("in patient A: ByID = %+v, ByPatient = %+v, %v", byID, mine, err)
	}
	err = in(t, app, func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, func(ctx context.Context) error {
		var err error
		byID, err = repo.ByID(ctx, consentAA)
		return err
	})
	if err != nil || !same(byID, activeAA) {
		t.Errorf("in clinic A: ByID = %+v, %v; want consent A-A", byID, err)
	}
	for name, set := range map[string]func(ctx context.Context) error{
		"patient B": func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) },
		"clinic B":  func(ctx context.Context) error { return repo.SetClinic(ctx, clinicB) },
	} {
		err := in(t, app, set, func(ctx context.Context) error { _, err := repo.ByID(ctx, consentAA); return err })
		if !errors.Is(err, consent.ErrNotFound) {
			t.Errorf("consent A-A in %s: %v, want ErrNotFound", name, err)
		}
		err = in(t, app, set, func(ctx context.Context) error { _, err := repo.ByID(ctx, "not-a-uuid"); return err })
		if !errors.Is(err, consent.ErrNotFound) {
			t.Errorf("an id that is not a UUID in %s: %v, want ErrNotFound", name, err)
		}
	}
	var none []consent.Consent
	err = in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) }, func(ctx context.Context) error {
		var err error
		none, err = repo.ByPatient(ctx, patientA)
		return err
	})
	if err != nil || len(none) != 0 {
		t.Errorf("patient A's consents in patient B's scope: %+v, %v; want none", none, err)
	}
}

func TestByPatient_ListsTheLatestGrantedFirst(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	later := granted.Add(48 * time.Hour)

	var got []consent.Consent
	err := in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }, func(ctx context.Context) error {
		if _, err := repo.Grant(ctx, consent.KindCare, patientA, clinicB, later); err != nil {
			return err
		}
		var err error
		got, err = repo.ByPatient(ctx, patientA)
		return err
	})
	if err != nil || len(got) != 3 || !got[0].DateTime.Equal(later) || got[0].OrganizationID != clinicB {
		t.Errorf("ByPatient() = %+v, %v; want the consent granted later first", got, err)
	}
}

func TestGrant_StoresOneActiveConsentPerClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)
	asB := func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) }

	var got consent.Consent
	err := in(t, app, asB, func(ctx context.Context) error {
		var err error
		got, err = repo.Grant(ctx, consent.KindCare, patientB, clinicA, at)
		return err
	})
	want := consent.Consent{ID: got.ID, PatientID: patientB, OrganizationID: clinicA, Kind: consent.KindCare, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
	if err != nil || got.ID == "" || !same(got, want) {
		t.Fatalf("Grant() = %+v, %v;\nwant %+v", got, err, want)
	}
	// A second grant gives ErrActive and leaves the transaction usable.
	err = in(t, app, asB, func(ctx context.Context) error {
		if _, err := repo.Grant(ctx, consent.KindCare, patientB, clinicA, at); !errors.Is(err, consent.ErrActive) {
			t.Errorf("second Grant() = %v, want ErrActive", err)
		}
		_, err := repo.ByID(ctx, got.ID)
		return err
	})
	if err != nil {
		t.Errorf("a read after the refusal: %v", err)
	}
	for name, clinic := range map[string]string{"an unknown clinic": "f0000000-0000-4000-8000-00000000000f", "a clinic id that is not a UUID": "x"} {
		err := in(t, app, asB, func(ctx context.Context) error {
			_, err := repo.Grant(ctx, consent.KindCare, patientB, clinic, at)
			return err
		})
		if !errors.Is(err, consent.ErrClinicNotFound) {
			t.Errorf("Grant() to %s = %v, want ErrClinicNotFound", name, err)
		}
	}
	// Another patient's consent is refused by row-level security.
	err = in(t, app, asB, func(ctx context.Context) error {
		_, err := repo.Grant(ctx, consent.KindCare, patientA, clinicB, at)
		return err
	})
	if err == nil || errors.Is(err, consent.ErrActive) {
		t.Errorf("Grant() for patient A in patient B's scope = %v, want the row-level security refusal", err)
	}
	err = in(t, app, asB, func(ctx context.Context) error {
		_, err := repo.Grant(ctx, consent.KindCare, "x", clinicB, at)
		return err
	})
	if err == nil {
		t.Error("Grant() with a patient id that is not a UUID succeeded")
	}
}

func TestRevoke_EndsOnlyAnActiveConsentOfTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }

	for name, tt := range map[string]struct {
		set func(ctx context.Context) error
		id  string
	}{
		"patient B's scope":   {id: consentAA, set: func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) }},
		"clinic A's scope":    {id: consentAA, set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }},
		"an inactive consent": {id: consentAB, set: asA},
		"an id not a UUID":    {id: "x", set: asA},
	} {
		err := in(t, app, tt.set, func(ctx context.Context) error { _, err := repo.Revoke(ctx, consent.KindCare, tt.id, at); return err })
		if !errors.Is(err, consent.ErrInactive) {
			t.Errorf("Revoke() in %s = %v, want ErrInactive", name, err)
		}
	}
	var got consent.Consent
	err := in(t, app, asA, func(ctx context.Context) error {
		var err error
		got, err = repo.Revoke(ctx, consent.KindCare, consentAA, at)
		return err
	})
	want := activeAA
	want.Status, want.RevokedAt, want.LastUpdated, want.VersionID = consent.StatusInactive, at, at, 2
	if err != nil || !same(got, want) {
		t.Errorf("Revoke() = %+v, %v;\nwant %+v", got, err, want)
	}
}

func TestState_GivesTheActiveConsentAndTheLatestRevocationOfTheScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	revoked := granted.Add(24 * time.Hour)

	tests := []struct {
		set     func(ctx context.Context) error
		name    string
		patient string
		clinic  string
		want    access.Consent
	}{
		{name: "active, in clinic A", set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, patient: patientA, clinic: clinicA, want: access.Consent{Active: true}},
		{name: "revoked, in clinic B", set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicB) }, patient: patientA, clinic: clinicB, want: access.Consent{RevokedAt: revoked}},
		{name: "revoked, in patient A", set: func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }, patient: patientA, clinic: clinicB, want: access.Consent{RevokedAt: revoked}},
		{name: "never, in clinic A", set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, patient: patientB, clinic: clinicA},
		{name: "clinic A's consent in clinic B", set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicB) }, patient: patientA, clinic: clinicA},
		{name: "ids that are not UUIDs", set: func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, patient: "x", clinic: "y"},
		{name: "no scope", set: func(context.Context) error { return nil }, patient: patientA, clinic: clinicA},
	}
	for _, tt := range tests {
		var got access.Consent
		err := in(t, app, tt.set, func(ctx context.Context) error {
			var err error
			got, err = repo.State(ctx, tt.patient, tt.clinic)
			return err
		})
		if err != nil || got.Active != tt.want.Active || !got.RevokedAt.Equal(tt.want.RevokedAt) {
			t.Errorf("%s: State() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

func TestTranscription_IsAConsentOfItsOwnWithTheSameScopes(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := granted.Add(48 * time.Hour)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }
	inClinic := func(id string) func(ctx context.Context) error {
		return func(ctx context.Context) error { return repo.SetClinic(ctx, id) }
	}

	var got consent.Consent
	err := in(t, app, asA, func(ctx context.Context) error {
		var err error
		got, err = repo.Grant(ctx, consent.KindTranscription, patientA, clinicA, at)
		return err
	})
	want := consent.Consent{ID: got.ID, PatientID: patientA, OrganizationID: clinicA, Kind: consent.KindTranscription, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
	if err != nil || got.ID == "" || got.ID == consentAA || !same(got, want) || got.Kind != want.Kind {
		t.Fatalf("Grant(transcription) = %+v, %v;\nwant %+v", got, err, want)
	}
	err = in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.Grant(ctx, consent.KindTranscription, patientA, clinicA, at)
		return err
	})
	if !errors.Is(err, consent.ErrActive) {
		t.Errorf("a second Grant(transcription) = %v, want ErrActive", err)
	}

	// Read by id and listed with its kind, in the patient's scope and its clinic's, not in another clinic's.
	var byID, care consent.Consent
	var mine []consent.Consent
	err = in(t, app, asA, func(ctx context.Context) error {
		var err error
		if byID, err = repo.ByID(ctx, got.ID); err != nil {
			return err
		}
		if care, err = repo.ByID(ctx, consentAA); err != nil {
			return err
		}
		mine, err = repo.ByPatient(ctx, patientA)
		return err
	})
	if err != nil || byID.Kind != consent.KindTranscription || !same(byID, want) || care.Kind != consent.KindCare {
		t.Errorf("ByID() = %+v and %+v, %v; want the transcription consent and the care one", byID, care, err)
	}
	if len(mine) != 3 || mine[0].ID != got.ID || mine[0].Kind != consent.KindTranscription || mine[1].Kind != consent.KindCare {
		t.Errorf("ByPatient() = %+v; want the transcription consent first, then both care consents", mine)
	}
	err = in(t, app, inClinic(clinicA), func(ctx context.Context) error { _, err := repo.ByID(ctx, got.ID); return err })
	if err != nil {
		t.Errorf("ByID() in clinic A = %v", err)
	}
	err = in(t, app, inClinic(clinicB), func(ctx context.Context) error { _, err := repo.ByID(ctx, got.ID); return err })
	if !errors.Is(err, consent.ErrNotFound) {
		t.Errorf("ByID() in clinic B = %v, want ErrNotFound", err)
	}

}

// The state shows both kinds; each kind's revocation finds only its own: a care revocation of the transcription
// consent's id revokes nothing.
func TestTranscription_HasItsStateAndRevocationOfItsOwn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := granted.Add(48 * time.Hour)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }
	inClinic := func(id string) func(ctx context.Context) error {
		return func(ctx context.Context) error { return repo.SetClinic(ctx, id) }
	}
	var got consent.Consent
	if err := in(t, app, asA, func(ctx context.Context) error {
		var err error
		got, err = repo.Grant(ctx, consent.KindTranscription, patientA, clinicA, at)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	state := func() access.Consent {
		var st access.Consent
		if err := in(t, app, inClinic(clinicA), func(ctx context.Context) error {
			var err error
			st, err = repo.State(ctx, patientA, clinicA)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return st
	}
	if st := state(); !st.Active || !st.Transcription {
		t.Errorf("State() = %+v, want both active", st)
	}
	err := in(t, app, asA, func(ctx context.Context) error { _, err := repo.Revoke(ctx, consent.KindCare, got.ID, at); return err })
	if !errors.Is(err, consent.ErrInactive) {
		t.Errorf("Revoke(care) of the transcription consent = %v, want ErrInactive", err)
	}
	var revoked consent.Consent
	err = in(t, app, asA, func(ctx context.Context) error {
		var err error
		revoked, err = repo.Revoke(ctx, consent.KindTranscription, got.ID, at.Add(time.Hour))
		return err
	})
	if err != nil || revoked.Status != consent.StatusInactive || revoked.Kind != consent.KindTranscription || revoked.VersionID != 2 {
		t.Errorf("Revoke(transcription) = %+v, %v", revoked, err)
	}
	if st := state(); !st.Active || st.Transcription || !st.RevokedAt.IsZero() {
		t.Errorf("State() after the withdrawal = %+v, want the care consent active only, no care revocation", st)
	}
}

func TestRevokeTranscription_EndsThePatientsActiveTranscriptionConsentWithTheClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := granted.Add(48 * time.Hour)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }

	var given, got consent.Consent
	err := in(t, app, asA, func(ctx context.Context) error {
		var err error
		if given, err = repo.Grant(ctx, consent.KindTranscription, patientA, clinicA, at); err != nil {
			return err
		}
		got, err = repo.RevokeTranscription(ctx, patientA, clinicA, at.Add(time.Hour))
		return err
	})
	if err != nil || got.ID != given.ID || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("RevokeTranscription() = %+v, %v; want %s revoked", got, err, given.ID)
	}
	for name, set := range map[string]func(ctx context.Context) error{
		"again":             asA,
		"patient B's scope": func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) },
	} {
		err := in(t, app, set, func(ctx context.Context) error {
			_, err := repo.RevokeTranscription(ctx, patientA, clinicA, at)
			return err
		})
		if !errors.Is(err, consent.ErrInactive) {
			t.Errorf("RevokeTranscription() %s = %v, want ErrInactive", name, err)
		}
	}
	err = in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.RevokeTranscription(ctx, "x", clinicA, at)
		return err
	})
	if !errors.Is(err, consent.ErrInactive) {
		t.Errorf("RevokeTranscription() with ids that are not UUIDs = %v, want ErrInactive", err)
	}
}

func TestSetPatientAndSetClinic_NeedATransaction(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := consent.NewPostgresRepository(app)
	if err := repo.SetPatient(t.Context(), patientA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetPatient() outside a transaction = %v, want ErrNoTx", err)
	}
	if err := repo.SetClinic(t.Context(), clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetClinic() outside a transaction = %v, want ErrNoTx", err)
	}
}

// Synthetic ids of the chat (SAL-28): patient A's conversation with clinic A and clinic B's practitioner role.
const (
	conversationA = "e1000000-0000-4000-8000-0000000000e1"
	roleA         = "a2000000-0000-4000-8000-0000000000a2"
	roleB         = "b2000000-0000-4000-8000-0000000000b2"
)

// seedChat is seed with a role in each clinic and patient A's conversation with clinic A's role.
func seedChat(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	seed(t, owner)
	for _, s := range []struct {
		sql  string
		args []any
	}{
		{`WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example') RETURNING id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $2, $1, id FROM p`, []any{clinicA, roleA}},
		{`WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Di', 'Example') RETURNING id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $2, $1, id FROM p`, []any{clinicB, roleB}},
		{`INSERT INTO conversation (id, organization_id, patient_id, practitioner_role_id, created_by) VALUES ($1, $2, $3, $4, $5)`,
			[]any{conversationA, clinicA, patientA, roleA, accountA}},
	} {
		if _, err := owner.Exec(t.Context(), s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// A chat consent (SAL-28) is the patient's, for one conversation and one role: granted and revoked in the patient's
// scope, at most one active, read by the conversation's clinic and by the guest (their role in their clinic), and by
// no one else.
func TestChat_GrantsRevokesAndShowsTheConsentForTheConversationAndRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedChat(t, owner)
	repo := consent.NewPostgresRepository(app)
	at := granted.Add(48 * time.Hour)
	asA := func(ctx context.Context) error { return repo.SetPatient(ctx, patientA) }
	g := consent.ChatGrant{ConversationID: conversationA, OrganizationID: clinicA, PractitionerRoleID: roleB}

	var got consent.Consent
	err := in(t, app, asA, func(ctx context.Context) error {
		var err error
		got, err = repo.GrantChat(ctx, patientA, g, at)
		return err
	})
	want := consent.Consent{ID: got.ID, PatientID: patientA, OrganizationID: clinicA, ConversationID: conversationA, PractitionerRoleID: roleB,
		Kind: consent.KindChat, Status: consent.StatusActive, DateTime: at, LastUpdated: at, VersionID: 1}
	if err != nil || got.ID == "" || !same(got, want) || got.ConversationID != conversationA || got.PractitionerRoleID != roleB {
		t.Fatalf("GrantChat() = %+v, %v;\nwant %+v", got, err, want)
	}
	err = in(t, app, asA, func(ctx context.Context) error { _, err := repo.GrantChat(ctx, patientA, g, at); return err })
	if err == nil {
		t.Error("a second active chat consent for the conversation and role: accepted")
	}

	granted := func(name string, set func(ctx context.Context) error, role string, want bool) {
		t.Helper()
		var ok bool
		err := in(t, app, set, func(ctx context.Context) error {
			var err error
			ok, err = repo.ChatGranted(ctx, conversationA, role)
			return err
		})
		if err != nil || ok != want {
			t.Errorf("%s: ChatGranted() = %v, %v; want %v", name, ok, err, want)
		}
	}
	guest := func(org, role string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			if err := repo.SetClinic(ctx, org); err != nil {
				return err
			}
			return postgres.SetLocal(ctx, "app.practitioner_role_id", role)
		}
	}
	granted("the patient", asA, roleB, true)
	granted("the patient, another role", asA, roleA, false)
	granted("clinic A", func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, roleB, true)
	granted("the guest", guest(clinicB, roleB), roleB, true)
	granted("the guest's role with another clinic set", guest("f0000000-0000-4000-8000-00000000000f", roleB), roleB, false)
	granted("clinic B without the role", func(ctx context.Context) error { return repo.SetClinic(ctx, clinicB) }, roleB, false)
	granted("patient B", func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) }, roleB, false)

	later := at.Add(time.Hour)
	var revokedChat consent.Consent
	err = in(t, app, asA, func(ctx context.Context) error {
		var err error
		revokedChat, err = repo.RevokeChat(ctx, conversationA, roleB, later)
		return err
	})
	if err != nil || revokedChat.ID != got.ID || revokedChat.Status != consent.StatusInactive || !revokedChat.RevokedAt.Equal(later) || revokedChat.VersionID != 2 {
		t.Errorf("RevokeChat() = %+v, %v; want the consent inactive, revoked later, version 2", revokedChat, err)
	}
	granted("the guest after the revocation", guest(clinicB, roleB), roleB, false)
	err = in(t, app, asA, func(ctx context.Context) error {
		_, err := repo.RevokeChat(ctx, conversationA, roleB, later)
		return err
	})
	if !errors.Is(err, consent.ErrInactive) {
		t.Errorf("a second RevokeChat() = %v, want ErrInactive", err)
	}
	err = in(t, app, func(ctx context.Context) error { return repo.SetPatient(ctx, patientB) }, func(ctx context.Context) error {
		_, err := repo.GrantChat(ctx, patientB, g, at)
		return err
	})
	if err == nil {
		t.Error("patient B grants for patient A's conversation: accepted")
	}
}
