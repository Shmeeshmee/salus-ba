package auditevent_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// RecordAll stores several events in one insert, each stamped like Record; none is no insert, and one invalid
// event stores none of them.
func TestRecordAll_StoresTheEventsInOneInsert(t *testing.T) {
	svc, repo := newService()
	first, second := login(), login()
	second.EntityReference = ""
	ctx := middleware.WithRequestID(t.Context(), "req-9")
	if err := svc.RecordAll(ctx, []auditevent.AuditEvent{first, second}); err != nil {
		t.Fatalf("RecordAll() = %v", err)
	}
	if len(repo.batches) != 1 || len(repo.batches[0]) != 2 || !repo.inTx[0] {
		t.Fatalf("batches %+v, want one of two in a transaction", repo.batches)
	}
	for i, e := range repo.batches[0] {
		if !e.Recorded.Equal(now) || e.RequestID != "req-9" {
			t.Errorf("event %d = %+v, want it stamped with the clock and the request id", i, e)
		}
	}
	if err := svc.RecordAll(t.Context(), nil); err != nil || len(repo.batches) != 1 {
		t.Errorf("RecordAll(none) = %v with %d batches, want no insert", err, len(repo.batches))
	}
	bad := login()
	bad.Action = ""
	if err := svc.RecordAll(t.Context(), []auditevent.AuditEvent{first, bad}); !errors.Is(err, auditevent.ErrInvalid) || len(repo.batches) != 1 {
		t.Errorf("RecordAll(an invalid one) = %v, want ErrInvalid and nothing stored", err)
	}
	repo.err = errors.New("the database is down")
	if err := svc.RecordAll(t.Context(), []auditevent.AuditEvent{first}); !errors.Is(err, repo.err) {
		t.Errorf("RecordAll() = %v, want the repository's error", err)
	}
}
