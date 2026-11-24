package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

type fakeEraser struct {
	err  error
	runs int
}

func (f *fakeEraser) Run(context.Context) ([]datarequest.Erased, error) {
	f.runs++
	return []datarequest.Erased{{AccountID: "00000000-0000-4000-8000-0000000000a1", RequestID: "00000000-0000-4000-8000-0000000000d1"}}, f.err
}

// The daily account_erasure job runs the erasure once, and fails (so River retries and alerts) when it does; it is
// cancelled, not retried, while the erasure is not built in this binary.
func TestAccountErasureWorker_RunsTheErasure(t *testing.T) {
	t.Parallel()
	f := &fakeEraser{}
	w := &accountErasureWorker{erasure: f, log: slog.New(slog.DiscardHandler)}
	if err := w.Work(t.Context(), &river.Job[AccountErasureArgs]{}); err != nil || f.runs != 1 {
		t.Errorf("Work() = %v after %d runs, want nil after 1", err, f.runs)
	}
	f.err = errors.New("synthetic failure")
	if err := w.Work(t.Context(), &river.Job[AccountErasureArgs]{}); !errors.Is(err, f.err) {
		t.Errorf("Work() = %v, want the run's error", err)
	}
	var cancel *river.JobCancelError
	if err := (&accountErasureWorker{}).Work(t.Context(), &river.Job[AccountErasureArgs]{}); !errors.As(err, &cancel) {
		t.Errorf("Work() without the erasure = %v, want a cancellation", err)
	}
	if (AccountErasureArgs{}).Kind() != "account_erasure" {
		t.Errorf("kind %q", AccountErasureArgs{}.Kind())
	}
}
