package organization_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
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

// A patient reads a clinic they are served by (its directory form is the handler's), in its scope and audited as a
// read by them, so that the app names the clinic of a visit the directory no longer shows; any other clinic is not
// found, as for anyone outside it. Staff read their own as before, Served not asked.
func TestReadFor_GivesAServedPatientTheirClinicAuditedAndNoOneElse(t *testing.T) {
	served := organization.Caller{AccountID: patient.AccountID, PatientID: patientOfB}
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).ReadFor(t.Context(), served, clinicB, servedAs(clinicB))
	if err != nil || got.ID != clinicB || got.Name != "Synthetic clinic B" || repo.clinic != clinicB || slices.Contains(repo.calls, "outside:ByID") {
		t.Errorf("ReadFor(B) as its patient = %+v, %v (scope %q, calls %v); want B in its scope", got, err, repo.clinic, repo.calls)
	}
	if len(audit.events) != 1 || audit.events[0].Subtype.Code != auditevent.CodeRead || audit.events[0].AgentAccountID != patient.AccountID ||
		audit.events[0].EntityReference != "Organization/"+clinicB || audit.events[0].OrganizationID != clinicB || audit.events[0].Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit events %+v, want one read of clinic B by the patient", audit.events)
	}
	for name, sv := range map[string]organization.Served{"A's patient": servedAs(clinicA), "no one served": servedAs(), "no Served": nil} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		if _, err := newService(repo, audit).ReadFor(t.Context(), served, clinicB, sv); !errors.Is(err, organization.ErrNotFound) || len(repo.calls) != 0 {
			t.Errorf("ReadFor(B) as %s = %v after %v, want ErrNotFound without reading", name, err, repo.calls)
		}
	}
	broken := func(context.Context) ([]string, error) { return nil, errors.New("connection reset") }
	if _, err := newService(newFakeRepository(), &fakeAuditor{}).ReadFor(t.Context(), served, clinicB, broken); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ReadFor() with a failing Served = %v, want its error", err)
	}
	repo = newFakeRepository()
	if got, err := newService(repo, &fakeAuditor{}).ReadFor(t.Context(), admin, clinicA, broken); err != nil || got != repo.orgs[clinicA] {
		t.Errorf("ReadFor(A) as its admin = %+v, %v; want A without asking Served", got, err)
	}
	if _, err := newService(newFakeRepository(), &fakeAuditor{}).ReadFor(t.Context(), otherAdmin, clinicA, servedAs(clinicA)); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ReadFor(A) as B's admin = %v, want ErrNotFound", err)
	}
}
