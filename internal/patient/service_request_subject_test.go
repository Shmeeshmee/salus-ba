package patient_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

func (r *fakeRepository) SetDataRequests(ctx context.Context, on bool) error {
	r.seen(ctx, map[bool]string{true: "SetDataRequests(on)", false: "SetDataRequests(off)"}[on])
	return nil
}

func (r *fakeRepository) Restriction(ctx context.Context, patientID, org string) (patient.Restriction, error) {
	r.seen(ctx, "Restriction")
	if org != r.clinic {
		return patient.Restriction{}, errors.New("read outside the clinic's scope")
	}
	by, ok := r.restricted[patientID+"/"+org]
	if !ok {
		return patient.Restriction{}, r.err
	}
	return patient.Restriction{At: now.Add(-time.Hour), ByErasure: by == ""}, r.err
}

// The clinic's admin reads the patient a request targeting the clinic names (wave 5 walk, finding 3): the names, the
// birth date and whether the clinic registered them, nothing else of the profile; in the clinic's scope widened to the
// patients its requests name, closed again after the read; audited as a read of the Patient at the patient's request.
func TestRequestSubject_TheAdminReadsThePatientARequestNames(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	p := repo.profiles[accountA]
	p.Phone = "+38761000111"
	repo.profiles[accountA] = p
	s := restrictionService(repo, audit, access.Consent{}, nil)
	got, err := s.RequestSubject(t.Context(), admin, patientA)
	if err != nil {
		t.Fatalf("RequestSubject: %v", err)
	}
	want := patient.Patient{ID: patientA, GivenName: "Ada", FamilyName: "Example", BirthDate: date(1990, 1, 1)}
	if got != want {
		t.Errorf("RequestSubject = %+v, want %+v (no phone, account or version)", got, want)
	}
	if want := []string{"SetClinic", "SetDataRequests(on)", "ByID", "SetDataRequests(off)"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls %v, want %v, all in a transaction", repo.calls, want)
	}
	read := event(admin, auditevent.CodeRead, patientA, auditevent.OutcomeSuccess)
	read.Purpose.Code = auditevent.CodePATRQT
	checkEvents(t, audit, read)
}

// A patient the clinic registered reads as registered by it.
func TestRequestSubject_ARegisteredPatient(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.profiles["registered"] = patient.Patient{ID: registeredID, RegisteredBy: clinicA, GivenName: "Rina", FamilyName: "Example", Phone: "+38761000222"}
	s := restrictionService(repo, audit, access.Consent{}, nil)
	got, err := s.RequestSubject(t.Context(), admin, registeredID)
	if err != nil || got.RegisteredBy != clinicA || got.Phone != "" {
		t.Errorf("RequestSubject = %+v, %v; want registered by the clinic, without the phone", got, err)
	}
}

// Only the clinic's admin reads it: a practitioner, a receptionist and a patient are refused, audited without the id,
// and nothing is read; an id the scope does not show is not found, audited without the id.
func TestRequestSubject_RefusesAllButTheAdmin(t *testing.T) {
	reception := patient.Caller{AccountID: staffA, OrganizationID: clinicA, Reception: true}
	for _, c := range []patient.Caller{practitioner, reception, me} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		s := restrictionService(repo, audit, access.Consent{}, nil)
		if got, err := s.RequestSubject(t.Context(), c, patientA); !errors.Is(err, patient.ErrForbidden) || got != (patient.Patient{}) {
			t.Errorf("RequestSubject(%+v) = %+v, %v; want ErrForbidden and nothing", c, got, err)
		}
		if slices.Contains(repo.calls, "ByID") || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure || audit.events[0].EntityReference != "" {
			t.Errorf("calls %v, audit %+v; want no read and one refusal without the id", repo.calls, audit.events)
		}
	}
	repo, audit := newFakeRepository(), &fakeAuditor{}
	s := restrictionService(repo, audit, access.Consent{}, nil)
	if _, err := s.RequestSubject(t.Context(), admin, "d7000000-0000-4000-8000-0000000000d7"); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("RequestSubject(unknown) = %v, want ErrNotFound", err)
	}
	missing := event(admin, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
	missing.Purpose.Code = auditevent.CodePATRQT
	checkEvents(t, audit, missing)
	if repo.calls[len(repo.calls)-1] != "SetDataRequests(off)" {
		t.Errorf("calls %v, want the scope closed after a miss too", repo.calls)
	}
}

// The admin reads whether the record of one of the clinic's patients is restricted there, since when and by whom (the
// clinic, or the erasure, which the clinic cannot lift): audited as a read of the Patient at the patient's request.
func TestRestriction_TheAdminReadsItsState(t *testing.T) {
	cases := map[string]struct {
		restricted map[string]string
		want       patient.Restriction
	}{
		"none":        {want: patient.Restriction{}},
		"the clinic":  {restricted: map[string]string{patientA + "/" + clinicA: staffA}, want: patient.Restriction{At: now.Add(-time.Hour)}},
		"the erasure": {restricted: map[string]string{patientA + "/" + clinicA: ""}, want: patient.Restriction{At: now.Add(-time.Hour), ByErasure: true}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo, audit := newFakeRepository(), &fakeAuditor{}
			repo.restricted = tc.restricted
			s := restrictionService(repo, audit, access.Consent{RevokedAt: now.Add(-time.Hour)}, nil)
			got, err := s.Restriction(t.Context(), admin, patientA)
			if err != nil || got != tc.want {
				t.Errorf("Restriction = %+v, %v; want %+v", got, err, tc.want)
			}
			if got.Restricted() != !tc.want.At.IsZero() {
				t.Errorf("Restricted() = %v", got.Restricted())
			}
			read := event(admin, auditevent.CodeRead, patientA, auditevent.OutcomeSuccess)
			read.Purpose.Code = auditevent.CodePATRQT
			checkEvents(t, audit, read)
		})
	}
}

// Only the admin reads it, and only for the clinic's patient.
func TestRestriction_TheStateIsTheAdminsOnly(t *testing.T) {
	for _, c := range []patient.Caller{practitioner, me, {AccountID: staffA, OrganizationID: clinicA, Reception: true}} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		s := restrictionService(repo, audit, access.Consent{Active: true}, nil)
		if _, err := s.Restriction(t.Context(), c, patientA); !errors.Is(err, patient.ErrForbidden) {
			t.Errorf("Restriction(%+v) = %v, want ErrForbidden", c, err)
		}
		if slices.Contains(repo.calls, "Restriction") || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("calls %v, audit %+v; want no read and one refusal", repo.calls, audit.events)
		}
	}
	repo, audit := newFakeRepository(), &fakeAuditor{}
	s := restrictionService(repo, audit, access.Consent{}, nil)
	if _, err := s.Restriction(t.Context(), admin, patientA); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("Restriction(not the clinic's) = %v, want ErrNotFound", err)
	}
	if len(audit.events) != 1 || audit.events[0].EntityReference != "" {
		t.Errorf("audit %+v, want one refusal without the id", audit.events)
	}
}
