package datarequest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// The warning's sender reads the open request in the deadline job's scope with its answerer's time zone: the
// clinic's for a clinic target, the operator's for the account; audited as the warning's.
func TestWarning_TheOpenRequestAndItsAnswerersTimeZone(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(warnedAt)
	f.repo.owner = false // the notifier's connection: the deadline scope only
	for id, zone := range map[string]string{toAccountOpen: datarequest.OperatorTimeZone, toClinic: "Europe/Sarajevo"} {
		r, loc, err := f.svc.Warning(t.Context(), datarequest.WarningArgs{RequestID: id, Days: datarequest.WarnDays})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if r.ID != id || loc.String() != zone || !f.repo.deadlines {
			t.Errorf("%s: %+v in %v (deadline scope %v), want it in %s", id, r, loc, f.repo.deadlines, zone)
		}
	}
	if len(f.audit.events) != 2 {
		t.Fatalf("%d audit events, want 2", len(f.audit.events))
	}
	e := f.audit.events[1]
	if e.AgentName != datarequest.AgentWarning || e.AgentAccountID != "" || e.Action != auditevent.ActionRead ||
		e.EntityReference != "Patient/"+profileA || e.OrganizationID != clinicA {
		t.Errorf("audit event %+v, want the warning's read", e)
	}
}

// A request answered or withdrawn since the warning was queued is not seen: nothing to warn of.
func TestWarning_AClosedRequestIsNotFound(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(warnedAt)
	f.repo.owner = false
	if _, _, err := f.svc.Warning(t.Context(), datarequest.WarningArgs{RequestID: toAccountClosed, Days: 1}); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

// A stale warning sends nothing: its deadline passed while the send was retried, or the request was extended since.
func TestWarning_AStaleWarningIsNotFound(t *testing.T) {
	t.Parallel()
	late := newOperatorFixture(time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC))
	late.repo.owner = false
	if _, _, err := late.svc.Warning(t.Context(), datarequest.WarningArgs{RequestID: toAccountOpen, Days: 1}); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("past the deadline: error = %v, want ErrNotFound", err)
	}
	moved := newOperatorFixture(warnedAt)
	r := moved.repo.requests[toAccountOpen]
	r.ExtendedUntil, r.ExtensionReason = r.DueAt.AddDate(0, 2, 0), datarequest.ExtensionComplex
	moved.repo.requests[toAccountOpen] = r
	moved.repo.owner = false
	if _, _, err := moved.svc.Warning(t.Context(), datarequest.WarningArgs{RequestID: toAccountOpen, Days: 7}); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("extended since: error = %v, want ErrNotFound", err)
	}
}

// warnedAt is when the 7-day warning of the operator's fixture requests is queued.
var warnedAt = time.Date(2026, 12, 17, 23, 0, 0, 0, time.UTC)
