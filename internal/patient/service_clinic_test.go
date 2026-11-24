package patient_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

var reception = patient.Caller{AccountID: staffA, OrganizationID: clinicA, Reception: true}

// newClinicService is the service with consents whose RecordInPerson the test reads.
func newClinicService(repo *fakeRepository, audit *fakeAuditor, consents *fakeConsents) *patient.Service {
	return patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, consents, &fakeAppointments{})
}

func searchEvent(c patient.Caller, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(c, auditevent.CodeRead, id, outcome)
	e.Subtype.Code, e.Action = auditevent.CodeSearchType, auditevent.ActionExecute
	return e
}

func TestSearchClinic_GivesTheClinicsMatchesInItsScopeAndAuditsEachPatient(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		role   string
		caller patient.Caller
	}{
		"admin":        {caller: admin, role: ""},
		"receptionist": {caller: reception, role: ""},
		"practitioner": {caller: practitioner, role: roleA}, // their own patients only
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit := newFakeRepository(), &fakeAuditor{}
			repo.found = []patient.Patient{repo.profiles[accountA], repo.profiles[accountB]}
			got, err := newService(repo, audit).SearchClinic(t.Context(), tc.caller, patient.ClinicSearch{Name: "  Ex ", BirthDate: date(1990, 1, 1)})
			if err != nil || len(got) != 2 {
				t.Fatalf("SearchClinic = %v, %v; want the two", got, err)
			}
			if want := []string{"SetClinic", "SearchClinic"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicA {
				t.Errorf("calls %v in clinic %q, want %v in clinic A", repo.calls, repo.clinic, want)
			}
			if repo.search != (patient.ClinicSearch{Name: "Ex", BirthDate: date(1990, 1, 1)}) || repo.searchRole != tc.role {
				t.Errorf("searched %+v as role %q, want the trimmed name as %q", repo.search, repo.searchRole, tc.role)
			}
			checkEvents(t, audit, searchEvent(tc.caller, patientA, auditevent.OutcomeSuccess), searchEvent(tc.caller, patientB, auditevent.OutcomeSuccess))
		})
	}
}

func TestSearchClinic_AuditsASearchThatFindsNobody(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).SearchClinic(t.Context(), admin, patient.ClinicSearch{Name: "Nobody"})
	if err != nil || len(got) != 0 {
		t.Fatalf("SearchClinic = %v, %v; want none", got, err)
	}
	checkEvents(t, audit, searchEvent(admin, "", auditevent.OutcomeSuccess))
}

func TestSearchClinic_RefusesAPatientOtherStaffAndAShortName(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		query  string
		caller patient.Caller
	}{
		"a patient":            {caller: me, query: "Example", want: patient.ErrForbidden},
		"staff without a role": {caller: staff, query: "Example", want: patient.ErrForbidden},
		"one letter":           {caller: admin, query: " E ", want: patient.ErrInvalidSearch},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit := newFakeRepository(), &fakeAuditor{}
			if _, err := newService(repo, audit).SearchClinic(t.Context(), tc.caller, patient.ClinicSearch{Name: tc.query}); !errors.Is(err, tc.want) {
				t.Fatalf("SearchClinic = %v, want %v", err, tc.want)
			}
			if slices.Contains(repo.calls, "SearchClinic") {
				t.Error("searched anyway")
			}
			if tc.want == patient.ErrForbidden {
				checkEvents(t, audit, searchEvent(tc.caller, "", auditevent.OutcomeMinorFailure))
			}
		})
	}
}

func TestRegister_StoresTheClinicsProfileRecordsTheInPersonConsentAndAuditsBoth(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]patient.Caller{"admin": admin, "receptionist": reception} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit, consents := newFakeRepository(), &fakeAuditor{}, &fakeConsents{}
			in := patient.Patient{GivenName: " Cy ", FamilyName: "Example", BirthDate: date(1980, 3, 4), Phone: "+38761000002"}
			got, err := newClinicService(repo, audit, consents).Register(t.Context(), c, in)
			if err != nil || got.ID != registeredID || got.RegisteredBy != clinicA || got.GivenName != "Cy" {
				t.Fatalf("Register = %+v, %v; want the clinic's profile", got, err)
			}
			if want := []string{"SetClinic", "Register"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicA {
				t.Errorf("calls %v in clinic %q, want %v in clinic A", repo.calls, repo.clinic, want)
			}
			want := []any{consent.Caller{AccountID: staffA, OrganizationID: clinicA}, registeredID, true}
			if !slices.Equal(consents.inPerson, want) {
				t.Errorf("in-person consent %v, want %v inside the transaction", consents.inPerson, want)
			}
			checkEvents(t, audit, event(c, auditevent.CodeCreate, registeredID, auditevent.OutcomeSuccess))
		})
	}
}

func TestRegister_RefusesAPractitionerAPatientAndABadField(t *testing.T) {
	t.Parallel()
	valid := patient.Patient{GivenName: "Cy", FamilyName: "Example", BirthDate: date(1980, 3, 4)}
	bad := valid
	bad.Phone = "061 ABC" // still no phone once read as +387 (a local 061 … is now accepted, wave 4)
	for name, tc := range map[string]struct {
		want   error
		in     patient.Patient
		caller patient.Caller
	}{
		"a practitioner": {caller: practitioner, in: valid, want: patient.ErrForbidden},
		"a patient":      {caller: me, in: valid, want: patient.ErrForbidden},
		"a bad phone":    {caller: admin, in: bad, want: patient.ErrInvalidPhone},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit, consents := newFakeRepository(), &fakeAuditor{}, &fakeConsents{}
			if _, err := newClinicService(repo, audit, consents).Register(t.Context(), tc.caller, tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("Register = %v, want %v", err, tc.want)
			}
			if slices.Contains(repo.calls, "Register") || consents.inPerson != nil {
				t.Error("registered anyway")
			}
			if tc.want == patient.ErrForbidden {
				checkEvents(t, audit, event(tc.caller, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure))
			}
		})
	}
}

func TestRegister_FailsWithTheConsentAndWritesNoSuccess(t *testing.T) {
	t.Parallel()
	repo, audit, consents := newFakeRepository(), &fakeAuditor{}, &fakeConsents{err: errors.New("boom")}
	in := patient.Patient{GivenName: "Cy", FamilyName: "Example", BirthDate: date(1980, 3, 4)}
	if _, err := newClinicService(repo, audit, consents).Register(t.Context(), admin, in); err == nil {
		t.Fatal("Register succeeded without its consent")
	}
	checkEvents(t, audit)
}

// recordInPerson is consent.Service.RecordInPerson on fakeConsents: the call, and whether it ran in a transaction.
func (f *fakeConsents) RecordInPerson(ctx context.Context, c consent.Caller, patientID string) (consent.Consent, error) {
	f.inPerson = []any{c, patientID, ctx.Value(txKey{}) != nil}
	return consent.Consent{ID: "f1000000-0000-4000-8000-0000000000f1", PatientID: patientID}, f.err
}
