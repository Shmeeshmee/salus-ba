package patient_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

func (r *fakeRepository) ObjectToAnalytics(ctx context.Context, id string, at time.Time) error {
	r.seen(ctx, "ObjectToAnalytics "+id+" "+at.Format(time.RFC3339))
	return r.err
}

// An objection to the analytics (docs/clinic-cms/WAVE5.md S9) marks the profile from the request on, in the caller's
// transaction and scope (the data request's filing), audited as an update of the Patient.
func TestObjectToAnalytics_MarksTheProfileInTheCallersTransaction(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	if err := newService(repo, audit).ObjectToAnalytics(t.Context(), patientA); err != nil {
		t.Fatal(err)
	}
	if want := "ObjectToAnalytics " + patientA + " " + now.Format(time.RFC3339); len(repo.calls) != 1 || repo.calls[0] != want {
		t.Errorf("calls %v, want %q in a transaction", repo.calls, want)
	}
	if len(audit.events) != 1 || audit.events[0].Action != auditevent.ActionUpdate || audit.events[0].EntityReference != "Patient/"+patientA ||
		audit.events[0].Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("audit %+v, want the Patient's update at the patient's request", audit.events)
	}
}
