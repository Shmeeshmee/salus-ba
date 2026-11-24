package patient_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// erasureRepo records the erasure's restrictions on the fake repository.
type erasureRepo struct {
	*fakeRepository
	restricted []string
}

func (r *erasureRepo) RestrictForErasure(ctx context.Context, patientID, org string, at time.Time) error {
	r.seen(ctx, "RestrictForErasure")
	r.restricted = append(r.restricted, patientID+"@"+org+"@"+at.Format(time.RFC3339))
	return nil
}

// RestrictForErasure makes the fake a Repository; erasureRepo records the calls.
func (r *fakeRepository) RestrictForErasure(ctx context.Context, _, _ string, _ time.Time) error {
	r.seen(ctx, "RestrictForErasure")
	return r.err
}

// The erasure job restricts the record a clinic keeps of an erased account's profile, in that clinic's scope, audited
// as the job's update of the profile there (docs/clinic-cms/WAVE5.md S6, S7).
func TestRestrictForErasure_RestrictsInTheClinicsScopeAuditedAsTheJob(t *testing.T) {
	repo := &erasureRepo{fakeRepository: newFakeRepository()}
	audit := &fakeAuditor{}
	svc := patient.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now}, &fakeConsents{}, &fakeAppointments{})

	if err := svc.RestrictForErasure(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("RestrictForErasure() error = %v", err)
	}
	if want := []string{patientA + "@" + clinicA + "@" + now.Format(time.RFC3339)}; !slices.Equal(repo.restricted, want) {
		t.Errorf("restricted %v, want %v", repo.restricted, want)
	}
	if repo.clinic != clinicA || !slices.Equal(repo.calls, []string{"SetClinic", "RestrictForErasure"}) {
		t.Errorf("scope %q, calls %v; want clinic A's, in the transaction", repo.clinic, repo.calls)
	}
	if len(audit.events) != 1 {
		t.Fatalf("%d audit events, want 1", len(audit.events))
	}
	e := audit.events[0]
	if e.AgentName != patient.ErasureAgent || e.AgentAccountID != "" || e.EntityReference != "Patient/"+patientA ||
		e.OrganizationID != clinicA || e.EntityName != "" {
		t.Errorf("audit event %+v, want the job's update of the profile at clinic A", e)
	}
}
