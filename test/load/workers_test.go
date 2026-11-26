//go:build loadtest

package load_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
	"github.com/Shmeeshmee/salus/pkg/sms/smstest"
)

// slotService is the slot service as the worker's slots_generate job builds it (internal/worker/slots.go newSlots):
// generation on app (salus_app), the hourly run's list of clinics on stats (salus_stats).
func slotService(app, stats *pgxpool.Pool) *slot.Service {
	clk := clock.System{}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clk)
	roles := practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk})
	schedules := schedule.NewService(schedule.Ports{
		Repo: schedule.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk, Roles: roles,
		Clinics: organization.NewStats(organization.NewPostgresStatsRepository(stats)),
	})
	clinics := organization.NewService(organization.NewPostgresRepository(app), audit, tx, clk, nil)
	return slot.NewService(slot.NewPostgresRepository(app), tx, clk, schedules, clinics)
}

// measureSlots times the hourly slots_generate run over every schedule of every clinic, runs times one after the
// other (the scheduler enqueues one an hour).
func measureSlots(ctx context.Context, t target, runs int) (reading, error) {
	svc := slotService(t.app, t.stats)
	var failed error
	r := measure(ctx, "slot generation: the hourly run", runs, 1, func(ctx context.Context, _ int) (int, bool) {
		if _, err := svc.GenerateAll(ctx); err != nil {
			failed = err
			return 0, false
		}
		return 0, true
	})
	return r, failed
}

// countingSender is the notifier's SMTP port, counting what it is given.
type countingSender struct{ n atomic.Int64 }

func (s *countingSender) Send(context.Context, notification.Message) error {
	s.n.Add(1)
	return nil
}

// dueReminders makes n reminders overdue, as a lost dispatch would leave them: one more reminder on each of n booked
// visits of the next week, due a minute ago. It returns their ids.
func dueReminders(ctx context.Context, owner *pgxpool.Pool, n int) ([]string, error) {
	rows, err := owner.Query(ctx, `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
		SELECT a.id, a.organization_id, a.patient_id, m.offset, lower(a.period), lower(a.period) - make_interval(mins => m.offset)
		FROM (SELECT * FROM appointment WHERE status = 'booked' AND lower(period) BETWEEN now() + interval '30 minutes' AND now() + interval '6 days'
			ORDER BY hashtext(id::text) LIMIT $1) a,
			LATERAL (SELECT ceil(extract(epoch FROM lower(a.period) - now()) / 60)::int + 1 AS offset) m
		ON CONFLICT DO NOTHING RETURNING id::text`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// reminderReadings is the reminders' sweep and dispatch: n overdue reminders found by the hourly sweep, which enqueues
// their dispatch; the notifier's River client (worker.NewNotifier, every port real but the senders) dispatches each
// and sends its deliveries. It reports the sweep's time, the time until every reminder left scheduled and every job
// due by then finished (also returned), and each job kind's run time from River's own record.
func reminderReadings(ctx context.Context, t target, n int) ([]string, time.Duration, error) {
	ids, err := dueReminders(ctx, t.owner, n)
	if err != nil {
		return nil, 0, fmt.Errorf("due reminders: %w", err)
	}
	inserter, err := worker.NewInserter(t.app)
	if err != nil {
		return nil, 0, err
	}
	tx := postgres.NewTransactor(t.app)
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(t.app), Tx: tx, Clock: clock.System{},
		Audit:     auditevent.NewService(auditevent.NewPostgresRepository(t.app), tx, clock.System{}),
		Reminders: appointment.NewPostgresReminderRepository(t.app), Jobs: appointment.NewRiverJobs(inserter),
	})
	mail := &countingSender{}
	client, err := worker.NewNotifier(t.app, slog.New(slog.DiscardHandler), worker.Notifier{
		Sender: mail, Pusher: &pushtest.Sender{}, SMS: &smstest.Sender{},
		WebBaseURL: "https://web.example.test", PatientAppURL: "https://patient.example.test", LinkURL: "https://api.example.test",
	})
	if err != nil {
		return nil, 0, err
	}
	began := wall.Now()
	found, err := appointments.SweepReminders(ctx, 1000)
	if err != nil {
		return nil, 0, err
	}
	swept := wall.Now().Sub(began)
	// The scheduler is River's leader (cmd/scheduler): it moves a dispatch due by now from scheduled to available, every
	// few seconds, as in production.
	leader, err := worker.NewScheduler(t.app, false, slog.New(slog.DiscardHandler))
	if err != nil {
		return nil, 0, err
	}
	for _, c := range []interface {
		Start(context.Context) error
		Stop(context.Context) error
	}{leader, client} {
		if err := c.Start(ctx); err != nil {
			return nil, 0, err
		}
		defer func() { _ = c.Stop(context.WithoutCancel(ctx)) }()
	}
	done, err := waitDispatched(ctx, t.owner, ids, began, 10*time.Minute)
	if err != nil {
		return nil, 0, err
	}
	out := []string{fmt.Sprintf("reminders: %d made overdue, the sweep found %d in %s; dispatched and every job due by then done in %s (%.0f reminders/s); %d e-mails sent",
		len(ids), found, swept.Round(time.Millisecond), done.Round(time.Millisecond), float64(len(ids))/done.Seconds(), mail.n.Load())}
	rows, err := t.owner.Query(ctx, `SELECT kind, count(*),
			percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM finalized_at - attempted_at)) * 1000,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY extract(epoch FROM finalized_at - attempted_at)) * 1000,
			percentile_cont(0.99) WITHIN GROUP (ORDER BY extract(epoch FROM finalized_at - attempted_at)) * 1000
		FROM river_job WHERE created_at >= $1 AND state = 'completed' GROUP BY kind ORDER BY kind`, began)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var count int
		var p50, p95, p99 float64
		if err := rows.Scan(&kind, &count, &p50, &p95, &p99); err != nil {
			return nil, 0, err
		}
		out = append(out, fmt.Sprintf("| job %s | %d | %.1f | %.1f | %.1f | | |", kind, count, p50, p95, p99))
	}
	return out, done, rows.Err()
}

// waitDispatched waits, at most limit, until none of the reminders ids is scheduled and no job of the notifier's
// queues created since began and due by now is left to run; it returns the time since began.
func waitDispatched(ctx context.Context, owner *pgxpool.Pool, ids []string, began time.Time, limit time.Duration) (time.Duration, error) {
	deadline := wall.Now().Add(limit)
	for {
		var left int
		err := owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM appointment_reminder WHERE id = ANY($1::uuid[]) AND state = 'scheduled')
			+ (SELECT count(*) FROM river_job WHERE created_at >= $2 AND state IN ('available', 'running', 'retryable') AND scheduled_at <= now()
				AND queue IN ('email', 'push', 'sms'))`, ids, began).Scan(&left)
		if err != nil {
			return 0, err
		}
		if left == 0 {
			return wall.Now().Sub(began), nil
		}
		if wall.Now().After(deadline) {
			return 0, fmt.Errorf("reminders: %d left after %s", left, limit)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}
