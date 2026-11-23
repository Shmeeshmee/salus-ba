package consent_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// patientC is a patient clinic A registered (docs/clinic-cms F2), without an account.
const patientC = "d3000000-0000-4000-8000-0000000000d3"

func TestRecordInPerson_StoresTheCareConsentAsGivenInPersonRecordedByStaffAndAuditsIt(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.svc.RecordInPerson(t.Context(), staff, patientC)
	if err != nil || got.ID != newID || got.Kind != consent.KindCare || got.Status != consent.StatusActive || !got.GivenInPerson ||
		got.RecordedBy != staffA || got.PatientID != patientC || got.OrganizationID != clinicA || !got.DateTime.Equal(now) {
		t.Fatalf("RecordInPerson = %+v, %v; want clinic A's active care consent given in person, recorded by staff", got, err)
	}
	if want := []string{"SetClinic", "GrantInPerson"}; !slices.Equal(f.repo.calls, want) || f.repo.clinic != clinicA || f.repo.patient != "" {
		t.Errorf("calls %v in clinic %q patient %q, want %v in clinic A's scope", f.repo.calls, f.repo.clinic, f.repo.patient, want)
	}
	checkEvents(t, "RecordInPerson", f.audit, event(staff, auditevent.CodeCreate, newID, auditevent.OutcomeSuccess))
}

func TestRecordInPerson_RefusesAPatientAndAConsentActiveAlreadyAndAuditsIt(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		caller  consent.Caller
		patient string
	}{
		"a patient":      {caller: me, patient: patientC, want: consent.ErrForbidden},
		"active already": {caller: staff, patient: patientA, want: consent.ErrActive},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			if _, err := f.svc.RecordInPerson(t.Context(), tc.caller, tc.patient); !errors.Is(err, tc.want) {
				t.Fatalf("RecordInPerson = %v, want %v", err, tc.want)
			}
			checkEvents(t, name, f.audit, event(tc.caller, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
		})
	}
}
