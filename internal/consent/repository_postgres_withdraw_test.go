package consent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestRevokeInPerson_RevokesOnlyTheClinicsOwnInPersonConsentInItsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner) // patient A's app consent to clinic A is active
	const staffA = "a1000000-0000-4000-8000-0000000000a1"
	if _, err := owner.Exec(t.Context(), `INSERT INTO patient (id, registered_by_organization_id, given_name, family_name, birth_date)
		VALUES ($1, $2, 'Cy', 'Example', '1980-03-04')`, patientC, clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO account (id, email, password_hash, role, organization_id)
		VALUES ($1, 'desk.a@example.test', 'h', 'admin', $2)`, staffA, clinicA); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO consent (patient_id, organization_id, status, date_time, given_in_person, recorded_by_account_id)
		VALUES ($1, $2, 'active', $3, true, $4)`, patientC, clinicA, granted, staffA); err != nil {
		t.Fatal(err)
	}
	repo := consent.NewPostgresRepository(app)
	at := granted.Add(48 * time.Hour)
	revoke := func(clinic, patientID string) (consent.Consent, error) {
		var c consent.Consent
		err := in(t, app, func(ctx context.Context) error { return repo.SetClinic(ctx, clinic) }, func(ctx context.Context) error {
			var err error
			c, err = repo.RevokeInPerson(ctx, patientID, clinic, at)
			return err
		})
		return c, err
	}
	if _, err := revoke(clinicB, patientC); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("clinic B: %v, want ErrInactive (not its consent)", err)
	}
	if _, err := revoke(clinicA, patientA); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("an app patient's consent: %v, want ErrInactive (the patient's to revoke)", err)
	}
	var status string
	if err := owner.QueryRow(t.Context(), "SELECT status FROM consent WHERE id = $1", consentAA).Scan(&status); err != nil || status != "active" {
		t.Errorf("patient A's consent is %q (%v), want active", status, err)
	}
	got, err := revoke(clinicA, patientC)
	if err != nil || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(at) || got.VersionID != 2 || got.PatientID != patientC {
		t.Fatalf("RevokeInPerson = %+v, %v; want it revoked at version 2", got, err)
	}
	if _, err := revoke(clinicA, patientC); !errors.Is(err, consent.ErrInactive) {
		t.Errorf("a second time: %v, want ErrInactive", err)
	}
	// The clinic's scope never sets a consent back to active: the revoked one is not its to change.
	var n int64
	err = in(t, app, func(ctx context.Context) error { return repo.SetClinic(ctx, clinicA) }, func(ctx context.Context) error {
		tag, err := postgres.Conn(ctx, app).Exec(ctx, "UPDATE consent SET status = 'active', revoked_at = NULL WHERE patient_id = $1", patientC)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 0 {
		t.Errorf("reactivating in the clinic's scope: %d rows, %v; want none", n, err)
	}
}
