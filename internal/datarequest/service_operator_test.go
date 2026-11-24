package datarequest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// The requests of the operator's tests: two to the account (one open, one answered) and one to clinic A.
const (
	toAccountOpen   = "00000000-0000-4000-8000-0000000000f1"
	toAccountClosed = "00000000-0000-4000-8000-0000000000f2"
	toClinic        = "00000000-0000-4000-8000-0000000000f3"
)

// fakeReopener is the Closure's reopening: it withdraws the account's open erasure request.
type fakeReopener struct {
	err      error
	accounts []string
}

func (r *fakeReopener) ReopenAccount(_ context.Context, accountID string) (datarequest.Request, error) {
	r.accounts = append(r.accounts, accountID)
	if r.err != nil {
		return datarequest.Request{}, r.err
	}
	return datarequest.Request{ID: toAccountOpen, AccountID: accountID, PatientID: profileA, Kind: datarequest.KindErasure, Status: datarequest.StatusWithdrawn}, nil
}

type operatorFixture struct {
	fixture
	op     *datarequest.Operator
	reopen *fakeReopener
}

// newOperatorFixture is the Operator at now on the owner's connection, with the three requests received at filedAt.
func newOperatorFixture(now time.Time) operatorFixture {
	repo := newFakeRepo()
	repo.owner = true
	due := time.Date(2026, 12, 24, 22, 59, 59, 0, time.UTC)
	repo.put(datarequest.Request{ID: toAccountOpen, AccountID: accountA, PatientID: profileA, FiledBy: accountA, Kind: datarequest.KindAccess,
		Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: due, Version: 1})
	repo.put(datarequest.Request{ID: toAccountClosed, AccountID: accountA, FiledBy: accountA, Kind: datarequest.KindObjection,
		Status: datarequest.StatusFulfilled, AnswerCode: datarequest.AnswerDone, ReceivedAt: filedAt, DueAt: due, AnsweredAt: filedAt, Version: 2})
	repo.put(datarequest.Request{ID: toClinic, AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, FiledBy: accountA,
		Kind: datarequest.KindErasure, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: due, Version: 1})
	f := operatorFixture{fixture: newFixtureOn(repo, now), reopen: &fakeReopener{}}
	f.op = datarequest.NewOperator(f.svc, f.reopen)
	return f
}

// wantOperatorEvent checks that e is the operator's (salusctl, no account) of the action on count requests.
func wantOperatorEvent(t *testing.T, e auditevent.AuditEvent, action auditevent.Action, ref string, count int64) {
	t.Helper()
	if e.AgentName != auditevent.AgentOperator || e.AgentAccountID != "" || e.Action != action || e.EntityName != "data_request" ||
		e.EntityReference != ref || e.EntityCount != count || e.OrganizationID != "" || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit event %+v, want salusctl's %s of %d (%q)", e, action, count, ref)
	}
}

// The operator lists the requests to the account only, the open ones by deadline, audited as one search.
func TestOperatorRequests_TheAccountsOnly(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(filedAt)
	got, err := f.op.Requests(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].OrganizationID != "" || got[1].OrganizationID != "" {
		t.Errorf("requests %+v, want the two to the account", got)
	}
	if len(f.audit.events) != 1 {
		t.Fatalf("%d audit events, want 1", len(f.audit.events))
	}
	wantOperatorEvent(t, f.audit.events[0], auditevent.ActionRead, "", 2)
}

// The operator reads one request to the account with its history, audited; a clinic's is not theirs.
func TestOperatorRequest_WithItsHistory(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(filedAt)
	r, history, err := f.op.Request(t.Context(), toAccountOpen)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != toAccountOpen || len(history) != 1 || history[0].Status != datarequest.StatusReceived {
		t.Errorf("request %+v history %+v", r, history)
	}
	wantOperatorEvent(t, f.audit.events[0], auditevent.ActionRead, "Patient/"+profileA, 1)
	if _, _, err := f.op.Request(t.Context(), toClinic); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("a clinic's request: error = %v, want ErrNotFound", err)
	}
	if _, _, err := f.op.Request(t.Context(), "00000000-0000-4000-8000-000000000999"); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("an unknown request: error = %v, want ErrNotFound", err)
	}
}

// The operator answers an open request to the account with a code that fits; history by no account; audited.
func TestOperatorAnswer(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(filedAt.Add(time.Hour))
	ctx := t.Context()
	got, err := f.op.Answer(ctx, toAccountOpen, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != datarequest.StatusFulfilled || got.AnswerCode != datarequest.AnswerDone || !got.AnsweredAt.Equal(filedAt.Add(time.Hour)) || got.Version != 2 {
		t.Errorf("answered %+v", got)
	}
	last := f.repo.changes[len(f.repo.changes)-1]
	if last.RequestID != toAccountOpen || last.Status != datarequest.StatusFulfilled || last.ChangedBy != "" {
		t.Errorf("history entry %+v, want fulfilled by no account", last)
	}
	wantOperatorEvent(t, f.audit.events[0], auditevent.ActionUpdate, "Patient/"+profileA, 1)
	for name, tc := range map[string]struct {
		want error
		id   string
		a    datarequest.Answer
	}{
		"closed":           {datarequest.ErrClosed, toAccountClosed, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 2}},
		"a clinic's":       {datarequest.ErrNotFound, toClinic, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 1}},
		"a clinic's stale": {datarequest.ErrNotFound, toClinic, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 9}},
		"code misfits":     {datarequest.ErrInvalid, toAccountOpen, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerNoData, Version: 2}},
		"answered already": {datarequest.ErrClosed, toAccountOpen, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 1}},
	} {
		if _, err := f.op.Answer(ctx, tc.id, tc.a); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tc.want)
		}
	}
}

// The operator extends an open request to the account once, by two months in the operator's time zone, while its due
// date has not passed (S12).
func TestOperatorExtend(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(filedAt.Add(time.Hour))
	ctx := t.Context()
	got, err := f.op.Extend(ctx, toAccountOpen, datarequest.ExtensionComplex, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2027, 2, 24, 23, 59, 59, 0, sarajevo(t))
	if !got.ExtendedUntil.Equal(want) || got.ExtensionReason != datarequest.ExtensionComplex || got.Version != 2 {
		t.Errorf("extended %+v, want until %v", got, want)
	}
	wantOperatorEvent(t, f.audit.events[0], auditevent.ActionUpdate, "Patient/"+profileA, 1)
	if _, err := f.op.Extend(ctx, toAccountOpen, datarequest.ExtensionComplex, 2); !errors.Is(err, datarequest.ErrAlreadyExtended) {
		t.Errorf("a second extension: error = %v, want ErrAlreadyExtended", err)
	}
	if _, err := f.op.Extend(ctx, toAccountOpen, "bogus", 2); !errors.Is(err, datarequest.ErrInvalid) {
		t.Errorf("an unknown reason: error = %v, want ErrInvalid", err)
	}
	if _, err := f.op.Extend(ctx, toClinic, datarequest.ExtensionComplex, 1); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("a clinic's: error = %v, want ErrNotFound", err)
	}
}

func TestOperatorExtend_RefusedOnceTheDueDateHasPassed(t *testing.T) {
	t.Parallel()
	due := time.Date(2026, 12, 24, 22, 59, 59, 0, time.UTC)
	for name, tc := range map[string]struct {
		now  time.Time
		want error
	}{
		"at the due date": {due, nil},
		"a second after":  {due.Add(time.Second), datarequest.ErrOverdue},
	} {
		f := newOperatorFixture(tc.now)
		if _, err := f.op.Extend(t.Context(), toAccountOpen, datarequest.ExtensionManyRequests, 1); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tc.want)
		}
	}
}

// The operator reopens a closed account through the Closure, audited as the withdrawal of its erasure request.
func TestOperatorReopenAccount(t *testing.T) {
	t.Parallel()
	f := newOperatorFixture(filedAt)
	got, err := f.op.ReopenAccount(t.Context(), accountA)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != datarequest.StatusWithdrawn || len(f.reopen.accounts) != 1 || f.reopen.accounts[0] != accountA {
		t.Errorf("reopened %+v for %v", got, f.reopen.accounts)
	}
	wantOperatorEvent(t, f.audit.events[0], auditevent.ActionUpdate, "Patient/"+profileA, 1)

	f.reopen.err = datarequest.ErrNotFound
	if _, err := f.op.ReopenAccount(t.Context(), accountA); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("past its erasure date: error = %v, want ErrNotFound", err)
	}
	if len(f.audit.events) != 1 {
		t.Errorf("%d audit events, want the one of the reopening", len(f.audit.events))
	}
}
