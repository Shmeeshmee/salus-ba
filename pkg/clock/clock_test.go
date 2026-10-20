package clock_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

func TestFixed_ReturnsItsTime(t *testing.T) {
	want := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	var c clock.Clock = clock.Fixed{T: want}

	if got := c.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestSystem_ReturnsCurrentTime(t *testing.T) {
	var c clock.Clock = clock.System{}

	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, want between %v and %v", got, before, after)
	}
}
