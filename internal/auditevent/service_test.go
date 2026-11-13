package auditevent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

type fakeRepository struct {
	err     error
	inserts []auditevent.AuditEvent
	batches [][]auditevent.AuditEvent
	inTx    []bool
	counted []time.Time // the cut-offs of CountRecordedBefore
	count   int64
}

func (r *fakeRepository) CountRecordedBefore(_ context.Context, before time.Time) (int64, error) {
	r.counted = append(r.counted, before)
	return r.count, r.err
}

type txKey struct{}

func (r *fakeRepository) Insert(ctx context.Context, e auditevent.AuditEvent) error {
	r.inserts = append(r.inserts, e)
	r.inTx = append(r.inTx, ctx.Value(txKey{}) != nil)
	return r.err
}

func (r *fakeRepository) InsertAll(ctx context.Context, es []auditevent.AuditEvent) error {
	r.inTx = append(r.inTx, ctx.Value(txKey{}) != nil)
	if r.err != nil {
		return r.err
	}
	r.batches = append(r.batches, es)
	return nil
}

// fakeTransactor marks the context it hands fn, as postgres.Transactor carries its transaction.
type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

func newService() (*auditevent.Service, *fakeRepository) {
	repo := &fakeRepository{}
	return auditevent.NewService(repo, fakeTransactor{}, clock.Fixed{T: now}), repo
}

// login is a valid event: a successful login of a synthetic account.
func login() auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeUserAuthentication},
		Subtype:         auditevent.Coding{System: auditevent.SystemDCM, Code: auditevent.CodeLogin},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		AgentAccountID:  "0b6c8a52-5d0e-4a8e-9a43-3f2f5f0f0a01",
		EntityReference: "Account/0b6c8a52-5d0e-4a8e-9a43-3f2f5f0f0a01",
		Action:          auditevent.ActionExecute,
		Outcome:         auditevent.OutcomeSuccess,
	}
}

func TestRecord_StoresTheEventInATransactionStampedWithTheClock(t *testing.T) {
	svc, repo := newService()
	e := login()
	e.Recorded = now.Add(-time.Hour) // the caller's value is ignored

	if err := svc.Record(t.Context(), e); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if len(repo.inserts) != 1 {
		t.Fatalf("inserts = %d, want 1", len(repo.inserts))
	}
	want := login()
	want.Recorded = now
	if repo.inserts[0] != want {
		t.Errorf("stored %+v,\nwant %+v", repo.inserts[0], want)
	}
	if !repo.inTx[0] {
		t.Error("the insert ran outside WithTx")
	}
}

func TestRecord_TakesTheRequestIDFromTheContext(t *testing.T) {
	svc, repo := newService()
	ctx := middleware.WithRequestID(t.Context(), "req-123")

	if err := svc.Record(ctx, login()); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if got := repo.inserts[0].RequestID; got != "req-123" {
		t.Errorf("RequestID = %q, want req-123", got)
	}
}

func TestRecord_RejectsAnInvalidEvent(t *testing.T) {
	tests := []struct {
		change func(*auditevent.AuditEvent)
		name   string
	}{
		{name: "unknown action", change: func(e *auditevent.AuditEvent) { e.Action = "X" }},
		{name: "no action", change: func(e *auditevent.AuditEvent) { e.Action = "" }},
		{name: "unknown outcome", change: func(e *auditevent.AuditEvent) { e.Outcome = 3 }},
		{name: "no type code", change: func(e *auditevent.AuditEvent) { e.Type.Code = "" }},
		{name: "no type system", change: func(e *auditevent.AuditEvent) { e.Type.System = "" }},
		{name: "no subtype", change: func(e *auditevent.AuditEvent) { e.Subtype = auditevent.Coding{} }},
		{name: "no purpose", change: func(e *auditevent.AuditEvent) { e.Purpose = auditevent.Coding{} }},
		{name: "a row count without a table", change: func(e *auditevent.AuditEvent) { e.EntityCount = 3 }},
		{name: "a negative row count", change: func(e *auditevent.AuditEvent) { e.EntityName, e.EntityCount = "session", -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newService()
			e := login()
			tt.change(&e)

			if err := svc.Record(t.Context(), e); !errors.Is(err, auditevent.ErrInvalid) {
				t.Errorf("Record() error = %v, want ErrInvalid", err)
			}
			if len(repo.inserts) != 0 {
				t.Error("an invalid event was stored")
			}
		})
	}
}

func TestRecord_AcceptsEveryActionAndOutcome(t *testing.T) {
	actions := []auditevent.Action{auditevent.ActionCreate, auditevent.ActionRead, auditevent.ActionUpdate, auditevent.ActionDelete, auditevent.ActionExecute}
	outcomes := []auditevent.Outcome{auditevent.OutcomeSuccess, auditevent.OutcomeMinorFailure, auditevent.OutcomeSeriousFailure, auditevent.OutcomeMajorFailure}
	for _, a := range actions {
		for _, o := range outcomes {
			svc, _ := newService()
			e := login()
			e.Action, e.Outcome = a, o
			if err := svc.Record(t.Context(), e); err != nil {
				t.Errorf("Record(action %s, outcome %d) error = %v", a, o, err)
			}
		}
	}
}

func TestRecord_ReturnsTheRepositoryError(t *testing.T) {
	svc, repo := newService()
	repo.err = errors.New("insert failed")

	if err := svc.Record(t.Context(), login()); !errors.Is(err, repo.err) {
		t.Errorf("Record() error = %v, want the repository's error", err)
	}
}

func TestCountRecordedBefore_CountsThroughTheRepository(t *testing.T) {
	svc, repo := newService()
	repo.count = 2
	before := now.Add(-3653 * 24 * time.Hour)

	n, err := svc.CountRecordedBefore(t.Context(), before)

	if err != nil || n != 2 {
		t.Errorf("CountRecordedBefore() = %d, %v; want 2, nil", n, err)
	}
	if len(repo.counted) != 1 || !repo.counted[0].Equal(before) {
		t.Errorf("repository asked for %v, want [%v]", repo.counted, before)
	}
	repo.err = errors.New("synthetic failure")
	if _, err := svc.CountRecordedBefore(t.Context(), before); err == nil {
		t.Error("CountRecordedBefore() error = nil when the repository fails")
	}
}
