package patient_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

func TestRecordConsent_RecordsTheInPersonConsentAsTheStaffInOneTransactionAfterReadingTheProfile(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]patient.Caller{"admin": admin, "receptionist": reception} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit, consents := withRegistered(), &fakeAuditor{}, &fakeConsents{}
			if _, err := newClinicService(repo, audit, consents).RecordConsent(t.Context(), c, patientR); err != nil {
				t.Fatalf("RecordConsent = %v", err)
			}
			want := []any{consent.Caller{AccountID: c.AccountID, OrganizationID: clinicA}, patientR, true}
			if !slices.Equal(consents.inPerson, want) {
				t.Errorf("consent %v, want %v inside the transaction", consents.inPerson, want)
			}
			if repo.clinic != clinicA {
				t.Errorf("scope %q, want clinic A", repo.clinic)
			}
			checkEvents(t, audit, event(c, auditevent.CodeRead, patientR, auditevent.OutcomeSuccess))
		})
	}
}

func TestRecordConsent_RefusesAnotherClinicsPatientAPractitionerAndAnActiveConsent(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		err     error
		patient string
		caller  patient.Caller
		audit   auditevent.AuditEvent
	}{
		"another clinic's patient": {caller: admin, patient: patientS, want: patient.ErrNotFound,
			audit: event(admin, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)},
		"a practitioner": {caller: practitioner, patient: patientR, want: patient.ErrForbidden,
			audit: event(practitioner, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)},
		// The consent's guard finds the profile gone in the same instant (docs/clinic-cms/WAVE6.md BG1): a 404, not a 500.
		"a profile gone as the consent is recorded": {caller: admin, patient: patientR, err: consent.ErrNotFound, want: patient.ErrNotFound,
			audit: event(admin, auditevent.CodeRead, patientR, auditevent.OutcomeSuccess)},
		"an active consent": {caller: reception, patient: patientR, err: consent.ErrActive, want: consent.ErrActive,
			audit: event(reception, auditevent.CodeRead, patientR, auditevent.OutcomeSuccess)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			audit, consents := &fakeAuditor{}, &fakeConsents{err: tc.err}
			_, err := newClinicService(withRegistered(), audit, consents).RecordConsent(t.Context(), tc.caller, tc.patient)
			if !errors.Is(err, tc.want) {
				t.Fatalf("RecordConsent = %v, want %v", err, tc.want)
			}
			if tc.err == nil && consents.inPerson != nil {
				t.Error("recorded anyway")
			}
			checkEvents(t, audit, tc.audit)
		})
	}
}

func TestSearchClinicAndCorrect_SayWhetherThePatientHasAnActiveConsentWithTheClinic(t *testing.T) {
	t.Parallel()
	for name, active := range map[string]bool{"active": true, "none": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			consents := &fakeConsents{state: access.Consent{Active: active}}
			repo := withRegistered()
			repo.found = []patient.Patient{repo.profiles["r"]}
			svc := newClinicService(repo, &fakeAuditor{}, consents)
			found, err := svc.SearchClinic(t.Context(), admin, patient.ClinicSearch{Name: "Ra"})
			if err != nil || len(found) == 0 {
				t.Fatalf("SearchClinic = %v, %v", found, err)
			}
			for _, p := range found {
				if p.ConsentActive != active {
					t.Errorf("search: %s ConsentActive = %v, want %v", p.ID, p.ConsentActive, active)
				}
			}
			got, err := svc.Correct(t.Context(), reception, patient.Patient{ID: patientR, VersionID: 3, GivenName: "Ra", FamilyName: "Example", BirthDate: date(1970, 5, 6)})
			if err != nil || got.ConsentActive != active {
				t.Errorf("Correct = %+v, %v; want ConsentActive %v", got, err, active)
			}
			if !slices.Contains(consents.args, clinicA) || slices.Contains(consents.args, clinicB) {
				t.Errorf("consent states asked %v, want clinic A's only", consents.args)
			}
		})
	}
}

// The desk's registration and correction read a phone as the import does (walk wave 4, finding 2): 061 000 399 is
// +38761000399.
func TestRegisterAndCorrect_ReadALocalPhoneAsTheImportDoes(t *testing.T) {
	t.Parallel()
	svc := newClinicService(withRegistered(), &fakeAuditor{}, &fakeConsents{})
	got, err := svc.Correct(t.Context(), reception, patient.Patient{ID: patientR, VersionID: 3, GivenName: "Ra", FamilyName: "Example", BirthDate: date(1970, 5, 6), Phone: "061 000 399"})
	if err != nil || got.Phone != "+38761000399" {
		t.Errorf("Correct = %+v, %v; want the phone +38761000399", got, err)
	}
	reg, err := svc.Register(t.Context(), reception, patient.Patient{GivenName: "Ana", FamilyName: "Example", BirthDate: date(1980, 1, 2), Phone: "00387 61 000 400"})
	if err != nil || reg.Phone != "+38761000400" {
		t.Errorf("Register = %+v, %v; want the phone +38761000400", reg, err)
	}
}
