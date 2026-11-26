package organization_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// fakePatients gives each patient's clinics of a care consent, by patient id.
type fakePatients map[string][]string

func (f fakePatients) CareClinics(_ context.Context, patientID string) ([]string, error) {
	if patientID == "broken" {
		return nil, errors.New("connection reset")
	}
	return f[patientID], nil
}

// servedAs is a Served that gives ids.
func servedAs(ids ...string) organization.Served {
	return func(context.Context) ([]string, error) { return ids, nil }
}

const patientOfB = "c2000000-0000-4000-8000-0000000000c2"

// A clinic whose membership stops keeps the directory's reads for its own staff and for the patients with a care
// consent with it, active or revoked (readiness: suspended clinics): Served gives those clinics, listed ones only.
func TestServed_GivesStaffTheirListedClinicAndAPatientTheListedClinicsOfTheirCareConsents(t *testing.T) {
	repo := newFakeRepository() // A is not listed, B is
	svc := newService(repo, &fakeAuditor{}).WithPatients(fakePatients{patientOfB: {clinicA, clinicB, "c0000000-0000-4000-8000-00000000000c"}})
	for _, tt := range []struct {
		name   string
		caller organization.Caller
		want   []string
	}{
		{"B's admin", otherAdmin, []string{clinicB}},
		{"A's admin, A not listed", admin, nil},
		{"a patient of A, B and an unknown clinic", organization.Caller{AccountID: patient.AccountID, PatientID: patientOfB}, []string{clinicB}},
		{"an account without a patient profile", patient, nil},
		{"no one signed in", organization.Caller{}, nil},
	} {
		got, err := svc.Served(t.Context(), tt.caller)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("Served(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	if slices.Contains(repo.calls, "outside:ByID") {
		t.Errorf("calls %v, want each clinic read in a transaction", repo.calls)
	}
	if got, err := newService(newFakeRepository(), &fakeAuditor{}).Served(t.Context(), organization.Caller{PatientID: patientOfB}); err != nil || got != nil {
		t.Errorf("Served(a patient) without the consents = %v, %v; want none", got, err)
	}
	if _, err := svc.Served(t.Context(), organization.Caller{PatientID: "broken"}); err == nil {
		t.Error("Served() with a failing consent read = nil, want the error")
	}
	repo.err = errors.New("connection reset")
	if _, err := svc.Served(t.Context(), otherAdmin); err == nil {
		t.Error("Served() with a failing clinic read = nil, want the error")
	}
}

// The directory hides a clinic whose membership stopped; its profile is still read, in its own scope, by those it
// serves, and by no one else. The directory's read asks nothing of the caller.
func TestProfile_AStoppedMembershipKeepsTheProfileForThoseTheClinicServes(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles = map[string]organization.Profile{clinicB: profileA}
	repo.stopped = []string{clinicB}
	svc := newService(repo, &fakeAuditor{})
	want := profileA
	want.VersionID = 1
	if got, err := svc.Profile(t.Context(), clinicB, servedAs(clinicB)); err != nil || !reflect.DeepEqual(got, want) || repo.clinic != clinicB {
		t.Errorf("Profile(stopped B) to whom it serves = %+v, %v (scope %q); want %+v in B's scope", got, err, repo.clinic, want)
	}
	for name, served := range map[string]organization.Served{"no one": nil, "A's": servedAs(clinicA), "none": servedAs()} {
		repo.clinic, repo.directory = "", false // a new transaction, without the last one's scope
		if _, err := svc.Profile(t.Context(), clinicB, served); !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("Profile(stopped B) to %s = %v, want ErrNotFound", name, err)
		}
	}
	broken := func(context.Context) ([]string, error) { return nil, errors.New("connection reset") }
	repo.clinic, repo.directory = "", false
	if _, err := svc.Profile(t.Context(), clinicB, broken); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile() with a failing Served = %v, want its error", err)
	}
	// A listed clinic with a running membership is the directory's: Served is not asked.
	repo.stopped = nil
	if _, err := svc.Profile(t.Context(), clinicB, broken); err != nil {
		t.Errorf("Profile(listed B) = %v, want its profile without asking Served", err)
	}
	// Not listed: no one gets it, served or not.
	if _, err := svc.Profile(t.Context(), clinicA, servedAs(clinicA)); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile(unlisted A) to whom it serves = %v, want ErrNotFound", err)
	}
}
