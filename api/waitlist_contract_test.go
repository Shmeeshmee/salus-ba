package api_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// The waitlist's statuses in api/openapi.yaml are the Go constants (SAL-35).
func TestWaitlistContract_StatusesMatchTheGoConstants(t *testing.T) {
	t.Parallel()
	want := []string{string(appointment.WaitlistActive), string(appointment.WaitlistBooked), string(appointment.WaitlistWithdrawn), string(appointment.WaitlistExpired)}
	if got := schemas(t)["WaitlistStatus"].Enum; !slices.Equal(got, want) {
		t.Errorf("WaitlistStatus: enum %v, want %v", got, want)
	}
}
