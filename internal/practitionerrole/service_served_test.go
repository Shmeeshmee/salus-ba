package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// served is an organization.Served that gives ids.
func served(ids ...string) organization.Served {
	return func(context.Context) ([]string, error) { return ids, nil }
}

// A patient reads a role of a clinic they are served by (a care consent with it, active or revoked: organization
// .Service.Served), in that clinic's scope and audited, so that the app names the clinic of a visit the directory no
// longer shows (readiness: suspended clinics); a role of any other clinic is not found, audited as now.
func TestRead_GivesAServedPatientARoleOfTheirClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).ReadFor(t.Context(), patient, roleB, served(clinicA, clinicB))
	if err != nil || got != repo.roles[1] {
		t.Fatalf("Read(B's role) as a patient of A and B = %+v, %v; want role B", got, err)
	}
	if slices.Contains(repo.calls, "outside:ByID") || repo.clinic != clinicB {
		t.Errorf("calls %v with clinic %q, want each read in a transaction, the last in B's scope", repo.calls, repo.clinic)
	}
	if want := event(patient, auditevent.CodeRead, roleB, auditevent.OutcomeSuccess); len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v, want %+v", audit.events, want)
	}
	for name, sv := range map[string]organization.Served{"A's patient": served(clinicA), "no one served": served(), "no Served": nil} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		if _, err := newService(repo, audit).ReadFor(t.Context(), patient, roleB, sv); !errors.Is(err, practitionerrole.ErrNotFound) {
			t.Errorf("Read(B's role) as %s = %v, want ErrNotFound", name, err)
		}
		if want := event(patient, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("%s: audit events %+v, want %+v", name, audit.events, want)
		}
	}
	broken := func(context.Context) ([]string, error) { return nil, errors.New("connection reset") }
	if _, err := newService(newFakeRepository(), &fakeAuditor{}).ReadFor(t.Context(), patient, roleB, broken); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("Read() with a failing Served = %v, want its error", err)
	}
	// Staff read their own clinic's roles as before; Served is not asked.
	if _, err := newService(newFakeRepository(), &fakeAuditor{}).ReadFor(t.Context(), staffA, roleB, broken); !errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("Read(B's role) as A's staff = %v, want ErrNotFound without asking Served", err)
	}
}
