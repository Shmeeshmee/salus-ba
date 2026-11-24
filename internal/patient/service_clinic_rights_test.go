package patient_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

const (
	patientR = "d4000000-0000-4000-8000-0000000000d4" // registered by clinic A
	patientS = "d5000000-0000-4000-8000-0000000000d5" // registered by clinic B
	consentR = "f4000000-0000-4000-8000-0000000000f4"
)

func (f *fakeConsents) WithdrawInPerson(ctx context.Context, c consent.Caller, patientID string) (consent.Consent, error) {
	f.withdraw = []any{c, patientID, ctx.Value(txKey{}) != nil}
	return consent.Consent{ID: consentR, PatientID: patientID, Status: consent.StatusInactive}, f.err
}

func (f *fakeConsents) OfClinicPatient(_ context.Context, _ consent.Caller, _ string) ([]consent.Consent, error) {
	return f.of, f.err
}

func (r *fakeRepository) Correct(ctx context.Context, p patient.Patient, org string, at time.Time) (patient.Patient, error) {
	r.seen(ctx, "Correct")
	if r.err != nil {
		return patient.Patient{}, r.err
	}
	for k, cur := range r.profiles {
		if cur.ID == p.ID && cur.RegisteredBy == org && org == r.clinic && cur.VersionID == p.VersionID {
			p.RegisteredBy, p.VersionID, p.LastUpdated = org, cur.VersionID+1, at
			r.profiles[k] = p
			return p, nil
		}
	}
	return patient.Patient{}, patient.ErrConflict
}

// withRegistered is the fake repository with patient R, registered by clinic A, and patient S, registered by clinic B
// (which clinic A's scope would not show; the fake shows every profile, so the service's own check is tested).
func withRegistered() *fakeRepository {
	repo := newFakeRepository()
	repo.profiles["r"] = patient.Patient{ID: patientR, RegisteredBy: clinicA, GivenName: "Ra", FamilyName: "Example", BirthDate: date(1970, 5, 6), VersionID: 3}
	repo.profiles["s"] = patient.Patient{ID: patientS, RegisteredBy: clinicB, GivenName: "Sa", FamilyName: "Example", BirthDate: date(1971, 5, 6), VersionID: 1}
	return repo
}

func TestCorrect_UpdatesTheRegisteredProfileInTheClinicsScopeVersionedAndAudited(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]patient.Caller{"admin": admin, "receptionist": reception} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit := withRegistered(), &fakeAuditor{}
			in := patient.Patient{ID: patientR, VersionID: 3, GivenName: " Rada ", FamilyName: "Example", BirthDate: date(1970, 6, 5), Phone: "+38761000004"}
			got, err := newService(repo, audit).Correct(t.Context(), c, in)
			if err != nil || got.GivenName != "Rada" || got.VersionID != 4 || !got.BirthDate.Equal(date(1970, 6, 5)) || got.Phone != "+38761000004" || !got.LastUpdated.Equal(now) {
				t.Fatalf("Correct = %+v, %v; want the trimmed profile at version 4", got, err)
			}
			if want := []string{"SetClinic", "ByID", "Correct"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicA {
				t.Errorf("calls %v in clinic %q, want %v in clinic A", repo.calls, repo.clinic, want)
			}
			checkEvents(t, audit, event(c, auditevent.CodeUpdate, patientR, auditevent.OutcomeSuccess))
		})
	}
}

func TestCorrect_RefusesAnotherClinicsOrAnAppPatientsProfileAPractitionerAStaleVersionAndABadField(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		caller patient.Caller
		in     patient.Patient
		audit  bool
	}{
		"another clinic's patient": {caller: admin, in: patient.Patient{ID: patientS, VersionID: 1}, want: patient.ErrNotFound, audit: true},
		"an app patient":           {caller: admin, in: patient.Patient{ID: patientA, VersionID: 2}, want: patient.ErrNotFound, audit: true},
		"a practitioner":           {caller: practitioner, in: patient.Patient{ID: patientR, VersionID: 3}, want: patient.ErrForbidden, audit: true},
		"a patient":                {caller: me, in: patient.Patient{ID: patientR, VersionID: 3}, want: patient.ErrForbidden, audit: true},
		"a stale version":          {caller: admin, in: patient.Patient{ID: patientR, VersionID: 2}, want: patient.ErrConflict, audit: true},
		"a blank name":             {caller: admin, in: patient.Patient{ID: patientR, VersionID: 3, GivenName: " "}, want: patient.ErrInvalidName},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo, audit := withRegistered(), &fakeAuditor{}
			in := tc.in
			if in.GivenName == "" {
				in.GivenName, in.FamilyName, in.BirthDate = "Changed", "Example", date(1980, 1, 1)
			} else {
				in.FamilyName, in.BirthDate = "Example", date(1980, 1, 1)
			}
			if _, err := newService(repo, audit).Correct(t.Context(), tc.caller, in); !errors.Is(err, tc.want) {
				t.Fatalf("Correct = %v, want %v", err, tc.want)
			}
			for _, p := range repo.profiles {
				if p.GivenName == "Changed" {
					t.Fatalf("profile %s changed", p.ID)
				}
			}
			if tc.audit {
				checkEvents(t, audit, event(tc.caller, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure))
			}
		})
	}
}

func TestWithdrawConsent_RecordsTheWithdrawalAsTheStaffInOneTransactionAfterReadingTheProfile(t *testing.T) {
	t.Parallel()
	repo, audit, consents := withRegistered(), &fakeAuditor{}, &fakeConsents{}
	got, err := newClinicService(repo, audit, consents).WithdrawConsent(t.Context(), reception, patientR)
	if err != nil || got.ID != consentR || got.Status != consent.StatusInactive {
		t.Fatalf("WithdrawConsent = %+v, %v; want the revoked consent", got, err)
	}
	want := []any{consent.Caller{AccountID: staffA, OrganizationID: clinicA}, patientR, true}
	if !slices.Equal(consents.withdraw, want) {
		t.Errorf("withdrawal %v, want %v inside the transaction", consents.withdraw, want)
	}
	checkEvents(t, audit, event(reception, auditevent.CodeRead, patientR, auditevent.OutcomeSuccess))
}

func TestWithdrawConsent_RefusesAnotherClinicsPatientAndAPractitionerAndPassesTheConsentsRefusal(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		err     error
		patient string
		caller  patient.Caller
	}{
		"another clinic's patient": {caller: admin, patient: patientS, want: patient.ErrNotFound},
		"a practitioner":           {caller: practitioner, patient: patientR, want: patient.ErrForbidden},
		"no consent in person":     {caller: admin, patient: patientR, err: consent.ErrInactive, want: consent.ErrInactive},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			consents := &fakeConsents{err: tc.err}
			_, err := newClinicService(withRegistered(), &fakeAuditor{}, consents).WithdrawConsent(t.Context(), tc.caller, tc.patient)
			if !errors.Is(err, tc.want) {
				t.Fatalf("WithdrawConsent = %v, want %v", err, tc.want)
			}
			if tc.err == nil && consents.withdraw != nil {
				t.Error("withdrawn anyway")
			}
		})
	}
}

func TestClinicRecord_GivesTheAdminTheProfileAndTheClinicsConsentsAuditedAsThePatientsRequest(t *testing.T) {
	t.Parallel()
	repo, audit := withRegistered(), &fakeAuditor{}
	consents := &fakeConsents{of: []consent.Consent{{ID: consentR, PatientID: patientR, OrganizationID: clinicA}}}
	got, err := newClinicService(repo, audit, consents).ClinicRecord(t.Context(), admin, patientR)
	if err != nil || got.Patient.ID != patientR || len(got.Consents) != 1 || got.Consents[0].ID != consentR {
		t.Fatalf("ClinicRecord = %+v, %v; want patient R with their consent", got, err)
	}
	e := event(admin, auditevent.CodeRead, patientR, auditevent.OutcomeSuccess)
	e.Purpose.Code = auditevent.CodePATRQT // handed over at the patient's request
	checkEvents(t, audit, e)
}

func TestClinicRecord_IsTheAdminsOnlyAndOnlyForTheClinicsRegisteredPatient(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		patient string
		caller  patient.Caller
	}{
		"a receptionist":           {caller: reception, patient: patientR, want: patient.ErrForbidden},
		"a practitioner":           {caller: practitioner, patient: patientR, want: patient.ErrForbidden},
		"another clinic's patient": {caller: admin, patient: patientS, want: patient.ErrNotFound},
		"an app patient":           {caller: admin, patient: patientA, want: patient.ErrNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			audit := &fakeAuditor{}
			if _, err := newService(withRegistered(), audit).ClinicRecord(t.Context(), tc.caller, tc.patient); !errors.Is(err, tc.want) {
				t.Fatalf("ClinicRecord = %v, want %v", err, tc.want)
			}
			e := event(tc.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
			e.Purpose.Code = auditevent.CodePATRQT
			checkEvents(t, audit, e)
		})
	}
}

func TestClinicRights_FailOnARepositoryError(t *testing.T) {
	t.Parallel()
	repo := withRegistered()
	repo.err = errors.New("boom")
	svc := newService(repo, &fakeAuditor{})
	if _, err := svc.Correct(t.Context(), admin, patient.Patient{ID: patientR, VersionID: 3, GivenName: "A", FamilyName: "B", BirthDate: date(1980, 1, 1)}); err == nil {
		t.Error("Correct = nil, want the error")
	}
	if _, err := svc.WithdrawConsent(t.Context(), admin, patientR); err == nil {
		t.Error("WithdrawConsent = nil, want the error")
	}
	if _, err := svc.ClinicRecord(t.Context(), admin, patientR); err == nil {
		t.Error("ClinicRecord = nil, want the error")
	}
}
