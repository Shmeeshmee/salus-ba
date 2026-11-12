package worker_test

import (
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The job creates this month's message partition and the two after it, as the app role (ADR 0012).
func TestWorker_CreatesTheChatPartitionsAhead(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	if kind, _ := runJobOn(t, db, liveKit(t), testFiles(t), communication.PartitionsArgs{}, nil); kind != river.EventKindJobCompleted {
		t.Fatalf("job ended %s", kind)
	}
	month := clock.System{}.Now().UTC()
	for i := range communication.PartitionsAhead {
		name := "communication_" + time.Date(month.Year(), month.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC).Format("y2006m01")
		var exists bool
		if err := db.owner.QueryRow(t.Context(), "SELECT to_regclass($1) IS NOT NULL", name).Scan(&exists); err != nil || !exists {
			t.Errorf("%s: exists %v, %v", name, exists, err)
		}
	}
}
