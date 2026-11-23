package invoice_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// A receipt whose report failed after the job gave up is reported again when the clinic's admin retries it: the same
// job is enqueued again, and the receipt becomes fiscalized when it runs.
func TestRetry_EnqueuesTheReportOfAFailedReceiptAgain(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	if err := f.svc.GiveUp(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Retry(t.Context(), adminC, inv.ID)
	if err != nil || got.ID != inv.ID || got.FiscalStatus != invoice.FiscalFailed {
		t.Fatalf("Retry() = %+v, %v; want the failed receipt", got, err)
	}
	if len(f.jobs.enqueued) != 2 || f.jobs.enqueued[1] != inv.ID {
		t.Errorf("enqueued = %v, want the receipt's report again", f.jobs.enqueued)
	}
	e := f.audit.last()
	if e.Subtype.Code != auditevent.CodeUpdate || e.EntityReference != "Invoice/"+inv.ID || e.AgentAccountID != admin ||
		e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v", e)
	}
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	if done, _ := f.svc.Read(t.Context(), adminC, inv.ID); done.FiscalStatus != invoice.FiscalFiscalized {
		t.Errorf("status = %s after the retried job, want fiscalized", done.FiscalStatus)
	}
}

// A rejected receipt is failed too, and may be retried (after the clinic fixed what was wrong).
func TestRetry_ARejectedReceiptMayBeRetried(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	f.reporter.Err = fiscal.ErrRejected
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Retry(t.Context(), adminC, inv.ID); err != nil {
		t.Errorf("Retry() = %v", err)
	}
}

// Only a failed receipt is retried, only by the clinic's admin; every refusal is audited.
func TestRetry_Refuses(t *testing.T) {
	f := newFixture(theSeller)
	pending, _ := f.svc.Issue(t.Context(), adminC, draft())
	fiscalized, _ := f.svc.Issue(t.Context(), adminC, draft())
	if err := f.svc.Fiscalize(t.Context(), org, fiscalized.ID); err != nil {
		t.Fatal(err)
	}
	failed, _ := f.svc.Issue(t.Context(), adminC, draft())
	if err := f.svc.GiveUp(t.Context(), org, failed.ID); err != nil {
		t.Fatal(err)
	}
	otherClinic := invoice.Caller{AccountID: "acct-other", OrganizationID: "org-2", Admin: true}
	for name, tt := range map[string]struct {
		want   error
		id     string
		caller invoice.Caller
	}{
		"a pending receipt":    {invoice.ErrNotFailed, pending.ID, adminC},
		"a fiscalized receipt": {invoice.ErrNotFailed, fiscalized.ID, adminC},
		"the practitioner":     {invoice.ErrForbidden, failed.ID, doctorC},
		"a patient":            {invoice.ErrForbidden, failed.ID, patientC},
		"another clinic":       {invoice.ErrNotFound, failed.ID, otherClinic},
		"no such receipt":      {invoice.ErrNotFound, "inv-none", adminC},
	} {
		before, events := len(f.jobs.enqueued), len(f.audit.events)
		if _, err := f.svc.Retry(t.Context(), tt.caller, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: Retry() = %v, want %v", name, err, tt.want)
		}
		if len(f.jobs.enqueued) != before {
			t.Errorf("%s: a report was enqueued", name)
		}
		if len(f.audit.events) != events+1 || f.audit.last().Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: the refusal was not audited: %+v", name, f.audit.last())
		}
	}
}
