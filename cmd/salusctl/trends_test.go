package main

import (
	"bytes"
	"strings"
	"testing"
)

// salusctl trends build builds a missed period and freezes it, audited as salusctl; it prints counts only. It
// refuses while ANALYTICS_PLATFORM_ENABLED is off (docs/plans/analytics.md §2a), an open period, and one built.
func TestTrends_BuildsAMissedPeriodOnlyWithThePlatformViewOn(t *testing.T) {
	_, count := ownerDB(t)
	t.Setenv("ANALYTICS_PLATFORM_ENABLED", "")
	if err := run(t.Context(), []string{"trends", "build", "2026-09"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "ANALYTICS_PLATFORM_ENABLED") {
		t.Errorf("off: %v, want a refusal naming the flag", err)
	}
	if n := count(`SELECT count(*) FROM trends_period`); n != 0 {
		t.Errorf("off: %d periods, want none", n)
	}
	t.Setenv("ANALYTICS_PLATFORM_ENABLED", "true")
	var out bytes.Buffer
	if err := run(t.Context(), []string{"trends", "build", "2026-09"}, &out); err != nil {
		t.Fatalf("on: %v", err)
	}
	if !strings.Contains(out.String(), "2026-09 published") {
		t.Errorf("output %q, want the period and its status", out.String())
	}
	if n := count(`SELECT count(*) FROM trends_period WHERE period = '2026-09' AND status = 'published'`); n != 1 {
		t.Errorf("%d periods, want the month", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND entity_name = 'condition'`); n != 1 {
		t.Errorf("%d build audit events, want one", n)
	}
	for _, args := range [][]string{{"trends", "build", "2026-09"}, {"trends", "build", "2999-01"}, {"trends", "build"}, {"trends", "list"}} {
		if err := run(t.Context(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%q): error = nil", args)
		}
	}
}
