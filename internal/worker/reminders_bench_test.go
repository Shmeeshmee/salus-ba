package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// seedReminders stores, as the owner, n appointments of the seeded patient of step minutes each, one after the other
// from start, each with its reminder offset minutes before, scheduled; returns the reminders' ids.
func seedReminders(b *testing.B, s *reminderStack, role, patientID string, start time.Time, n, step, offset int) []string {
	b.Helper()
	rows, err := s.owner.Query(b.Context(), `WITH a AS (
			INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
			SELECT $1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz + i * make_interval(mins => $7::integer), $4::timestamptz + (i + 1) * make_interval(mins => $7::integer)), $4::timestamptz - interval '30 days'
			FROM generate_series(0, $5::integer - 1) i RETURNING id, organization_id, patient_id, lower(period) AS start_at)
		INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
		SELECT id, organization_id, patient_id, $6::integer, start_at, start_at - make_interval(mins => $6::integer) FROM a RETURNING id::text`,
		patientID, role, clinicID, start, n, offset, step)
	if err != nil {
		b.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			b.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// The hourly sweep over 100 000 scheduled reminders, 1 000 of them overdue (their job lost): it finds and enqueues
// again the 1 000 (docs/plans/appointments.PROGRESS.md, SAL-33).
func BenchmarkRemindersSweep(b *testing.B) {
	s := newReminderStack(b)
	bk := seedBooking(b, s.owner)
	var role string
	if err := s.owner.QueryRow(b.Context(), `SELECT practitioner_role_id::text FROM appointment WHERE id = $1`, bk.future).Scan(&role); err != nil {
		b.Fatal(err)
	}
	now := time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC)
	// 1 000 overdue: due a week before their start in the next 17 hours; 99 000 due later.
	seedReminders(b, s, role, bk.patient, now.Add(time.Minute), 1000, 1, 10080)
	seedReminders(b, s, role, bk.patient, now.Add(30*24*time.Hour), 99000, 30, 60)
	if _, err := s.owner.Exec(b.Context(), "ANALYZE appointment_reminder"); err != nil {
		b.Fatal(err)
	}
	s.clock.set(now)
	for b.Loop() { // the 100 000 rows are seeded once
		n, err := s.appointments.SweepReminders(b.Context(), 1000)
		if err != nil || n != 1000 {
			b.Fatalf("SweepReminders() = %d, %v; want the 1000 overdue", n, err)
		}
	}
}

// 1 000 reminders due in the same minute: their dispatch (an e-mail and a push each) and then their 2 000 sends, by
// the notifier's 10 workers at once, on the fake senders (docs/plans/appointments.PROGRESS.md, SAL-33).
func BenchmarkRemindersDispatchAndSend(b *testing.B) {
	for range b.N {
		b.StopTimer()
		s := newReminderStack(b)
		bk := seedBooking(b, s.owner)
		var role, account string
		if err := s.owner.QueryRow(b.Context(), `SELECT practitioner_role_id::text, (SELECT account_id::text FROM patient WHERE id = $1) FROM appointment WHERE id = $2`,
			bk.patient, bk.future).Scan(&role, &account); err != nil {
			b.Fatal(err)
		}
		if _, err := s.owner.Exec(b.Context(), `INSERT INTO push_device (account_id, platform, token) VALUES ($1, 'android', 'synthetic-token-bench')`, account); err != nil {
			b.Fatal(err)
		}
		due := time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC) // 09:00 in Sarajevo, outside the quiet hours
		ids := seedReminders(b, s, role, bk.patient, due.Add(time.Hour), 1000, 30, 60)
		s.clock.set(due.Add(30 * time.Second))
		b.StartTimer()
		start := clock.System{}.Now()
		parallel(b, ids, func(ctx context.Context, id string) error {
			return s.mail.DispatchReminder(ctx, appointment.ReminderArgs{ReminderID: id, OrganizationID: clinicID})
		})
		dispatched := clock.System{}.Now().Sub(start)
		var deliveries []string
		rows, err := s.owner.Query(b.Context(), `SELECT id::text FROM notification_delivery`)
		if err != nil {
			b.Fatal(err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				b.Fatal(err)
			}
			deliveries = append(deliveries, id)
		}
		rows.Close()
		start = clock.System{}.Now()
		parallel(b, deliveries, func(ctx context.Context, id string) error { return s.mail.SendDelivery(ctx, id, false) })
		sent := clock.System{}.Now().Sub(start)
		if len(deliveries) != 2000 || len(s.sender.sent) != 1000 || len(s.pusher.Sent()) != 1000 {
			b.Fatalf("%d deliveries, %d e-mails, %d pushes; want 2000, 1000, 1000", len(deliveries), len(s.sender.sent), len(s.pusher.Sent()))
		}
		b.ReportMetric(float64(dispatched.Milliseconds()), "dispatch-ms")
		b.ReportMetric(float64(sent.Milliseconds()), "send-ms")
	}
}

// parallel runs fn over ids with the notifier's 10 workers.
func parallel(b *testing.B, ids []string, fn func(ctx context.Context, id string) error) {
	b.Helper()
	work := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	for range 10 {
		wg.Go(func() {
			for id := range work {
				if err := fn(b.Context(), id); err != nil {
					mu.Lock()
					first = err
					mu.Unlock()
				}
			}
		})
	}
	for _, id := range ids {
		work <- id
	}
	close(work)
	wg.Wait()
	if first != nil {
		b.Fatal(first)
	}
}
