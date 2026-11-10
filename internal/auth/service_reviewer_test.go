package auth_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
)

// The reviewer permission of the term list (SAL-14): the operator grants and revokes it on a staff account, audited
// with salusctl as the agent; the account then carries it, and so does its clinic membership.
func TestSetTermReviewer_GrantsAndRevokesAStaffAccountAndAuditsTheOperator(t *testing.T) {
	f := newFixture(staff())
	for _, reviewer := range []bool{true, false} {
		f.audit.events = nil
		if err := f.svc.SetTermReviewer(t.Context(), accountID, reviewer); err != nil {
			t.Fatalf("SetTermReviewer(%v) error = %v", reviewer, err)
		}
		if len(f.repo.outside) != 0 {
			t.Errorf("SetTermReviewer(%v): calls outside a transaction: %v", reviewer, f.repo.outside)
		}
		if got := f.repo.accounts[accountID].TermReviewer; got != reviewer {
			t.Errorf("after SetTermReviewer(%v) the account's TermReviewer = %v", reviewer, got)
		}
		m, err := f.svc.ClinicMember(t.Context(), adminSession()) // read outside a transaction, as the API does
		f.repo.outside = nil
		if err != nil || m.TermReviewer != reviewer || !m.Admin {
			t.Errorf("ClinicMember() after SetTermReviewer(%v) = %+v, %v; want the admin with TermReviewer %v", reviewer, m, err, reviewer)
		}
		want := auditevent.AuditEvent{
			Type: rest, Subtype: update, Purpose: hoperat, Action: auditevent.ActionUpdate, Outcome: auditevent.OutcomeSuccess,
			AgentName: auditevent.AgentOperator, EntityReference: "Account/" + accountID, OrganizationID: clinicID,
		}
		if e := f.onlyEvent(t); e != want {
			t.Errorf("SetTermReviewer(%v): audit event %+v,\nwant %+v", reviewer, e, want)
		}
	}
}

func TestSetTermReviewer_RefusesAPatientAndAnUnknownAccount(t *testing.T) {
	f := newFixture(patientWithoutMFA())
	if err := f.svc.SetTermReviewer(t.Context(), accountID, true); !errors.Is(err, auth.ErrNotStaff) || f.repo.accounts[accountID].TermReviewer {
		t.Errorf("SetTermReviewer(patient) = %v; want ErrNotStaff and the account unchanged", err)
	}
	want := auditevent.AuditEvent{
		Type: rest, Subtype: update, Purpose: hoperat, Action: auditevent.ActionUpdate, Outcome: auditevent.OutcomeMinorFailure,
		AgentName: auditevent.AgentOperator, EntityReference: "Account/" + accountID,
	}
	if e := f.onlyEvent(t); e != want {
		t.Errorf("audit event %+v,\nwant %+v", e, want)
	}

	f = newFixture()
	if err := f.svc.SetTermReviewer(t.Context(), accountID, true); !errors.Is(err, auth.ErrNotFound) || len(f.audit.events) != 0 {
		t.Errorf("SetTermReviewer(unknown) = %v with %d events; want ErrNotFound, not audited", err, len(f.audit.events))
	}

	f = newFixture(staff())
	f.repo.err = errors.New("connection reset")
	if err := f.svc.SetTermReviewer(t.Context(), accountID, true); err == nil || errors.Is(err, auth.ErrNotFound) {
		t.Errorf("SetTermReviewer() with a failing repository = %v, want its error", err)
	}
}
