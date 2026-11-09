package suggestion

import (
	"context"
	"sync"
	"time"
)

// Run follows the consultations' streams until ctx ends, as the member owner of the worker: every Tick it looks
// for streams, and follows each one, in a goroutine of its own, as long as the stream is there. Several members
// share the streams: each encounter is followed by the one member holding its lease (Claim), renewed every Tick;
// a member that dies loses it after leaseTTL and another takes the encounter over, from its stream's first
// entry. Errors are logged, never returned.
func (a *Assistant) Run(ctx context.Context, owner string) {
	var mu sync.Mutex
	following := map[string]bool{}
	active := func(id string) bool {
		mu.Lock()
		defer mu.Unlock()
		return following[id]
	}
	var wg sync.WaitGroup
	defer wg.Wait()
	for ctx.Err() == nil {
		ids, err := a.Live.Encounters(ctx)
		if err != nil && ctx.Err() == nil {
			a.Log.ErrorContext(ctx, "suggestions: the streams could not be listed", "error", err)
		}
		if err == nil {
			mu.Lock()
			now := make(map[string]bool, len(ids))
			for _, id := range ids {
				now[id] = true
				if !following[id] {
					wg.Go(func() { a.follow(ctx, id, owner, active) })
				}
			}
			following = now
			mu.Unlock()
		}
		wait(ctx, a.Tick)
	}
}

// follow steps the encounter's session every Tick while its stream is active and owner holds its lease. Without
// the lease it keeps no session: whoever holds it has the state.
func (a *Assistant) follow(ctx context.Context, encounterID, owner string, active func(string) bool) {
	var s *Session
	for ctx.Err() == nil && active(encounterID) {
		held, err := a.Live.Claim(ctx, encounterID, owner, leaseTTL)
		switch {
		case err != nil:
			if ctx.Err() == nil {
				a.Log.ErrorContext(ctx, "suggestions: the lease failed", "encounter_id", encounterID, "error", err)
			}
		case !held:
			s = nil
		default:
			if s == nil {
				s = a.NewSession(encounterID)
			}
			if err := a.Step(ctx, s); err != nil && ctx.Err() == nil {
				a.Log.ErrorContext(ctx, "suggestions: a step failed", "encounter_id", encounterID, "error", err)
			}
		}
		wait(ctx, a.Tick)
	}
}

// wait waits d, or until ctx ends.
func wait(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
