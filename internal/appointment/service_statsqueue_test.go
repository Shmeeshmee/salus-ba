package appointment_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
)

// refresh is the clinic-day refresh of an appointment of clinicA starting at at.
func refresh(at time.Time) string { return clinicA + " " + at.UTC().Format(time.RFC3339) }

// Every appointment write enqueues, in its transaction, the refresh of the clinic's statistics for the day of the
// appointment's start (docs/plans/analytics.md §4): a booking, a cancellation, a status set by the practitioner or
// by the consultation's end, a cancellation after a revocation.
func TestStatsRefresh_EveryAppointmentWriteEnqueuesItsClinicDay(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		fixture func() *fixture
		write   func(ctx context.Context, f *fixture) error
		want    []string
	}{
		"a booking": {func() *fixture { return newFixture() }, func(ctx context.Context, f *fixture) error {
			_, err := f.svc.Book(ctx, me, slotFree)
			return err
		}, []string{refresh(start)}},
		"a cancellation": {func() *fixture { return cancelFixture(start) }, func(ctx context.Context, f *fixture) error {
			_, err := f.svc.Cancel(ctx, pract, apptA)
			return err
		}, []string{refresh(start)}},
		"a status": {func() *fixture { return cancelFixture(now) }, func(ctx context.Context, f *fixture) error {
			_, err := f.svc.SetStatus(ctx, pract, apptA, appointment.StatusFulfilled)
			return err
		}, []string{refresh(now)}},
		"the consultation's outcome": {func() *fixture { return cancelFixture(now.Add(-time.Hour)) }, func(ctx context.Context, f *fixture) error {
			return inClinic(t, f, clinicA, func(ctx context.Context) error {
				return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA, true)
			})
		}, []string{refresh(now.Add(-time.Hour))}},
		"a revocation": {func() *fixture {
			f := newFixture(booked(apptA, patientA, roleA1, clinicA, start), booked(apptA2, patientA, roleA2, clinicA, start.Add(2*time.Hour)))
			f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Minute)}
			return f
		}, func(ctx context.Context, f *fixture) error {
			_, err := f.svc.CancelForRevokedConsent(ctx, patientA, clinicA)
			return err
		}, []string{refresh(start), refresh(start.Add(2 * time.Hour))}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := tt.fixture()
			if err := tt.write(t.Context(), f); err != nil {
				t.Fatal(err)
			}
			if got := slices.Sorted(slices.Values(f.stats.refreshes)); !slices.Equal(got, tt.want) {
				t.Errorf("refreshes %v, want %v", got, tt.want)
			}
		})
	}
}

// A refused write and an outcome that changes nothing enqueue no refresh.
func TestStatsRefresh_NothingWrittenNothingEnqueued(t *testing.T) {
	t.Parallel()
	f := cancelFixture(now.Add(time.Hour))
	if _, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusFulfilled); err == nil {
		t.Fatal("SetStatus() before the start succeeded")
	}
	if _, err := f.svc.Cancel(t.Context(), pract2, apptA); err == nil {
		t.Fatal("another practitioner cancelled")
	}
	if _, err := f.svc.Cancel(t.Context(), pract, apptA); err != nil {
		t.Fatal(err)
	}
	f.stats.refreshes = nil
	err := inClinic(t, f, clinicA, func(ctx context.Context) error {
		return f.svc.RecordConsultationOutcome(ctx, "consultation_window_close", apptA, true)
	})
	if err != nil || len(f.stats.refreshes) != 0 {
		t.Errorf("refreshes %v, %v; want none", f.stats.refreshes, err)
	}
}
