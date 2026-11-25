package datarequest_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// TestPatientRoutes_RefuseAndAuditAStaffAccount: a staff account on /me/data-requests is refused by the service and the
// refusal is audited in its clinic (docs/clinic-cms/WAVE6.md BG1); before, the handler answered 403 unaudited.
func TestPatientRoutes_RefuseAndAuditAStaffAccount(t *testing.T) {
	t.Parallel()
	staff := datarequest.Patient{AccountID: accountA, OrganizationID: clinicA}
	for name, call := range map[string]func(f fixture) error{
		"file": func(f fixture) error {
			_, err := f.svc.File(t.Context(), staff, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount})
			return err
		},
		"mine":     func(f fixture) error { _, err := f.svc.Mine(t.Context(), staff); return err },
		"withdraw": func(f fixture) error { _, err := f.svc.Withdraw(t.Context(), staff, "r1", 1); return err },
	} {
		f := newFixture(filedAt)
		if err := call(f); !errors.Is(err, datarequest.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", name, err)
		}
		if len(f.repo.requests) != 0 {
			t.Errorf("%s: a request was stored", name)
		}
		if len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure ||
			f.audit.events[0].AgentAccountID != accountA || f.audit.events[0].OrganizationID != clinicA {
			t.Errorf("%s: audit %+v, want one refusal by the staff account in its clinic", name, f.audit.events)
		}
	}
}
