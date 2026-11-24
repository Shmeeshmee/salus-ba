package consent_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// RevokeInPerson revokes the patient's active care consent with the clinic given in person, in the clinic's scope, as
// the database's policy for the clinic allows it.
func (r *fakeRepository) RevokeInPerson(ctx context.Context, patientID, organizationID string, at time.Time) (consent.Consent, error) {
	r.seen(ctx, "RevokeInPerson")
	if r.err != nil {
		return consent.Consent{}, r.err
	}
	for i, c := range r.consents {
		if c.PatientID == patientID && c.OrganizationID == organizationID && r.clinic == organizationID && r.patient == "" &&
			c.Kind == consent.KindCare && c.GivenInPerson && c.Status == consent.StatusActive {
			c.Status, c.RevokedAt, c.LastUpdated, c.VersionID = consent.StatusInactive, at, at, c.VersionID+1
			r.consents[i] = c
			return c, nil
		}
	}
	return consent.Consent{}, consent.ErrInactive
}

const consentC = "f3000000-0000-4000-8000-0000000000f3" // patient C's consent to clinic A, given in person

// withInPerson is the fixture with patient C's in-person care consent to clinic A.
func withInPerson() fixture {
	f := newFixture()
	f.repo.consents = append(f.repo.consents, consent.Consent{ID: consentC, PatientID: patientC, OrganizationID: clinicA, Kind: consent.KindCare,
		Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1, GivenInPerson: true, RecordedBy: staffA})
	return f
}

func TestWithdrawInPerson_RevokesTheConsentInTheClinicsScopeEnqueuesTheRevocationAndAuditsTheStaff(t *testing.T) {
	t.Parallel()
	f := withInPerson()
	got, err := f.svc.WithdrawInPerson(t.Context(), staff, patientC)
	if err != nil || got.ID != consentC || got.Status != consent.StatusInactive || !got.RevokedAt.Equal(now) {
		t.Fatalf("WithdrawInPerson = %+v, %v; want consent C revoked now", got, err)
	}
	if f.repo.calls[0] != "SetClinic" || f.repo.clinic != clinicA || f.repo.patient != "" || !slices.Contains(f.repo.calls, "RevokeInPerson") {
		t.Errorf("calls %v in clinic %q patient %q, want the revocation in clinic A's scope", f.repo.calls, f.repo.clinic, f.repo.patient)
	}
	// What an app patient's revocation does: the job cancels the visits ahead and withdraws the waitlist entries.
	if want := []string{patientC + " " + clinicA}; !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v, want %v", f.jobs.enqueued, want)
	}
	checkEvents(t, "WithdrawInPerson", f.audit, event(staff, auditevent.CodeUpdate, consentC, auditevent.OutcomeSuccess))
}

func TestWithdrawInPerson_RefusesAPatientAnotherClinicAndAConsentNotActiveAndAuditsIt(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		caller  consent.Caller
		patient string
	}{
		"a patient":                 {caller: me, patient: patientC, want: consent.ErrForbidden},
		"another clinic":            {caller: staffB, patient: patientC, want: consent.ErrInactive},
		"a consent given in an app": {caller: staff, patient: patientA, want: consent.ErrInactive},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := withInPerson()
			if _, err := f.svc.WithdrawInPerson(t.Context(), tc.caller, tc.patient); !errors.Is(err, tc.want) {
				t.Fatalf("WithdrawInPerson = %v, want %v", err, tc.want)
			}
			if len(f.jobs.enqueued) != 0 {
				t.Errorf("enqueued %v, want nothing", f.jobs.enqueued)
			}
			checkEvents(t, name, f.audit, event(tc.caller, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
		})
	}
}

func TestOfClinicPatient_GivesTheClinicsConsentsOfThePatientAndAuditsEachRead(t *testing.T) {
	t.Parallel()
	f := withInPerson()
	got, err := f.svc.OfClinicPatient(t.Context(), staff, patientC)
	if err != nil || len(got) != 1 || got[0].ID != consentC {
		t.Fatalf("OfClinicPatient = %+v, %v; want consent C", got, err)
	}
	if f.repo.clinic != clinicA || f.repo.patient != "" {
		t.Errorf("read in clinic %q patient %q, want clinic A's scope", f.repo.clinic, f.repo.patient)
	}
	checkEvents(t, "OfClinicPatient", f.audit, event(staff, auditevent.CodeRead, consentC, auditevent.OutcomeSuccess))
	if _, err := f.svc.OfClinicPatient(t.Context(), me, patientC); !errors.Is(err, consent.ErrForbidden) {
		t.Errorf("OfClinicPatient by a patient = %v, want ErrForbidden", err)
	}
}

func TestWithdrawInPerson_FailsOnARepositoryError(t *testing.T) {
	t.Parallel()
	f := withInPerson()
	f.repo.err = errors.New("boom")
	if _, err := f.svc.WithdrawInPerson(t.Context(), staff, patientC); err == nil {
		t.Fatal("WithdrawInPerson = nil, want the error")
	}
	if _, err := f.svc.OfClinicPatient(t.Context(), staff, patientC); err == nil {
		t.Fatal("OfClinicPatient = nil, want the error")
	}
}
