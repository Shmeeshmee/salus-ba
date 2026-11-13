package worker_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
	"github.com/Shmeeshmee/salus/pkg/sms/smstest"
)

// movingClock is a clock the test moves.
type movingClock struct {
	t  time.Time
	mu sync.Mutex
}

func (c *movingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *movingClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

// capturingSender keeps the e-mails it is given.
type capturingSender struct {
	sent []notification.Message
	mu   sync.Mutex
}

func (s *capturingSender) Send(_ context.Context, m notification.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, m)
	return nil
}

// noSlots is the slots port of a cancellation in these tests: the appointments have no slot rows to free.
type noSlots struct{}

func (noSlots) Bookable(context.Context, string) (slot.Bookable, error) {
	return slot.Bookable{}, slot.ErrUnavailable
}
func (noSlots) Hold(context.Context, string, string) (slot.Slot, error) {
	return slot.Slot{}, slot.ErrUnavailable
}
func (noSlots) Release(context.Context, string, string) (bool, error) { return false, nil }

// noQuestionnaires is the questionnaire check of these tests: none outstanding (questionnaireresponse's own tests
// cover it).
type noQuestionnaires struct{}

func (noQuestionnaires) Outstanding(context.Context, string, string, string) (bool, error) {
	return false, nil
}

// failOnce fails the first enqueue of a delivery, as a crash before the dispatch's commit would.
type failOnce struct {
	notification.DeliveryJobs
	failed bool
}

func (f *failOnce) EnqueueDelivery(ctx context.Context, ch notification.Channel, id string, at time.Time) error {
	if !f.failed {
		f.failed = true
		return errors.New("the notifier stopped")
	}
	return f.DeliveryJobs.EnqueueDelivery(ctx, ch, id, at)
}

// reminderStack is the reminders as the api, the worker and the notifier run them, on salus_app, at a clock the test
// moves; the senders are fakes.
type reminderStack struct {
	owner, app   *pgxpool.Pool
	clock        *movingClock
	appointments *appointment.Service
	mail         *notification.Service
	jobs         *failOnce
	sender       *capturingSender
	pusher       *pushtest.Sender
	sms          *smstest.Sender
}

func newReminderStack(t testing.TB) *reminderStack {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	inserter, err := worker.NewInserter(app)
	if err != nil {
		t.Fatal(err)
	}
	s := &reminderStack{owner: owner, app: app, clock: &movingClock{}, sender: &capturingSender{}, pusher: &pushtest.Sender{}, sms: &smstest.Sender{}}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, s.clock)
	patients := patient.NewService(patient.NewPostgresRepository(app), audit, tx, s.clock, nil, nil)
	consents := consent.NewService(consent.NewPostgresRepository(app), audit, tx, s.clock, nil)
	rules := organization.NewAppointments(organization.AppointmentsPorts{Repo: organization.NewPostgresAppointmentsRepository(app), Audit: audit, Tx: tx, Clock: s.clock})
	s.appointments = appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: s.clock, Consents: consents, Slots: noSlots{},
		Patients: patients, Jobs: appointment.NewRiverJobs(inserter), Stats: clinicstats.NewRiverQueue(inserter),
		Reminders: appointment.NewPostgresReminderRepository(app), Rules: rules,
		Tokens: appointment.NewPostgresTokenRepository(app), LinkURL: "https://api.example.test",
	})
	accounts := auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: s.clock})
	s.jobs = &failOnce{DeliveryJobs: notification.NewRiverDeliveryJobs(inserter), failed: true}
	s.mail = notification.NewService(notification.Ports{
		Accounts: accounts, Clinics: organization.NewService(organization.NewPostgresRepository(app), audit, tx, s.clock, nil),
		Questionnaires: noQuestionnaires{},
		Sender:         s.sender, PatientAppURL: "https://patient.example.test",
		Deliveries: notification.NewPostgresDeliveryRepository(app), DeliveryJobs: s.jobs, Devices: notification.NewPostgresRepository(app),
		Reminders: s.appointments, Rules: rules, Languages: accounts, Phones: patients, Pusher: s.pusher, SMS: s.sms,
		Audit: audit, Tx: tx, Clock: s.clock,
	})
	return s
}

// visit is an appointment of the seeded patient and its reminders, as a booking schedules them.
type visit struct {
	reminders map[int]string // by offset
	id        string
}

// seedVisit stores, as the owner, an appointment of the patient at start and its reminders 24 hours and 1 hour
// before, as a booking at now does (the booking's scheduling itself is appointment's TestBook_*).
func (s *reminderStack) seedVisit(t *testing.T, role, patientID string, start time.Time) visit {
	t.Helper()
	v := visit{reminders: map[int]string{}}
	if err := s.owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), $4::timestamptz - interval '3 days') RETURNING id::text`,
		patientID, role, clinicID, start).Scan(&v.id); err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{1440, 60} {
		var id string
		if err := s.owner.QueryRow(t.Context(), `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
			VALUES ($1, $2, $3, $4::integer, $5::timestamptz, $5::timestamptz - make_interval(mins => $4::integer)) RETURNING id::text`, v.id, clinicID, patientID, offset, start).Scan(&id); err != nil {
			t.Fatal(err)
		}
		v.reminders[offset] = id
	}
	return v
}

// dispatch runs the reminder's dispatch at its due time, as its job does.
func (s *reminderStack) dispatch(t *testing.T, v visit, offset int, due time.Time) error {
	t.Helper()
	s.clock.set(due)
	return s.mail.DispatchReminder(t.Context(), appointment.ReminderArgs{ReminderID: v.reminders[offset], OrganizationID: clinicID})
}

// sends are the reminder's send jobs: "<kind> at <time>", and the deliveries' channels and states.
func (s *reminderStack) sends(t *testing.T, reminderID string) (jobs, deliveries []string) {
	t.Helper()
	rows, err := s.owner.Query(t.Context(), `SELECT j.kind || ' at ' || to_char(j.scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI"Z"'), d.channel || ' ' || d.state
		FROM notification_delivery d JOIN river_job j ON j.args->>'delivery_id' = d.id::text
		WHERE d.source_id = $1 ORDER BY d.channel`, reminderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var j, d string
		if err := rows.Scan(&j, &d); err != nil {
			t.Fatal(err)
		}
		jobs, deliveries = append(jobs, j), append(deliveries, d)
	}
	return jobs, deliveries
}

func (s *reminderStack) state(t *testing.T, reminderID string) string {
	t.Helper()
	var st string
	if err := s.owner.QueryRow(t.Context(), `SELECT state FROM appointment_reminder WHERE id = $1`, reminderID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	return st
}

// sendAll runs every pending delivery's send of the reminder at at, twice (a job run again).
func (s *reminderStack) sendAll(t *testing.T, reminderID string, at time.Time) {
	t.Helper()
	s.clock.set(at)
	rows, err := s.owner.Query(t.Context(), `SELECT id::text FROM notification_delivery WHERE source_id = $1`, reminderID)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for range 2 {
		for _, id := range ids {
			if err := s.mail.SendDelivery(t.Context(), id, false); err != nil {
				t.Fatalf("send %s: %v", id, err)
			}
		}
	}
}

// A patient in Asia/Dubai of a clinic in Europe/Sarajevo, around the October change there (25 October 2026, 03:00 to
// 02:00 local): each reminder goes at its absolute due time, its push held by the quiet hours of the zone they are in
// (the clinic's once they report none) until they end, or dropped when that is too close to the start; the e-mail is
// never held. Every job runs twice and the first dispatch is rolled back: each message goes once. A reminder of an
// appointment cancelled or moved after it was scheduled sends nothing.
func TestReminder_SentOnceAtTheRightInstant(t *testing.T) {
	t.Parallel()
	s := newReminderStack(t)
	b := seedBooking(t, s.owner)
	ctx := t.Context()
	var role, account string
	if err := s.owner.QueryRow(ctx, `SELECT practitioner_role_id::text, (SELECT account_id::text FROM patient WHERE id = $1) FROM appointment WHERE id = $2`,
		b.patient, b.future).Scan(&role, &account); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO push_device (account_id, platform, token) VALUES ($1, 'android', 'synthetic-token-dubai')`,
		`INSERT INTO notification_preference (account_id, push, email, sms, quiet_start, quiet_end, time_zone) VALUES ($1, true, true, false, '22:00', '08:00', 'Asia/Dubai')`,
	} {
		if _, err := s.owner.Exec(ctx, q, account); err != nil {
			t.Fatal(err)
		}
	}

	early := checkAbroad(t, s, b, role)
	checkSentOnce(t, s, b, early)
	checkAtHome(t, s, b, role, account)
	checkCancelledAndMoved(t, s, b, role)
}

// checkAbroad dispatches the reminders of a visit while the patient is in Dubai, the first dispatch rolled back.
func checkAbroad(t *testing.T, s *reminderStack, b booking, role string) visit {
	t.Helper()
	// 04:30 UTC on Sunday 25 October: 05:30 in Sarajevo (CET, after the change), 08:30 in Dubai.
	early := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 25, 4, 30, 0, 0, time.UTC))
	// The day before, 04:30 UTC is 08:30 in Dubai (awake) though 06:30 in Sarajevo (CEST, quiet): the patient's zone
	// decides, the push goes at once. The first dispatch is rolled back; its retry and a second run deliver once.
	s.jobs.failed = false
	dayBefore := time.Date(2026, 10, 24, 4, 30, 0, 0, time.UTC)
	if err := s.dispatch(t, early, 1440, dayBefore); err == nil {
		t.Fatal("the first dispatch did not fail")
	}
	if st := s.state(t, early.reminders[1440]); st != "scheduled" {
		t.Fatalf("after the rolled-back dispatch: %s, want scheduled", st)
	}
	for range 2 {
		if err := s.dispatch(t, early, 1440, dayBefore); err != nil {
			t.Fatal(err)
		}
	}
	jobs, _ := s.sends(t, early.reminders[1440])
	if want := []string{"reminder_email at 2026-10-24T04:30Z", "reminder_push at 2026-10-24T04:30Z"}; !slices.Equal(jobs, want) {
		t.Errorf("the day before: %v, want %v", jobs, want)
	}
	// An hour before, 03:30 UTC is 07:30 in Dubai: quiet until 08:00 there, 04:00 UTC, just 30 minutes before the start.
	if err := s.dispatch(t, early, 60, time.Date(2026, 10, 25, 3, 30, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	jobs, _ = s.sends(t, early.reminders[60])
	if want := []string{"reminder_email at 2026-10-25T03:30Z", "reminder_push at 2026-10-25T04:00Z"}; !slices.Equal(jobs, want) {
		t.Errorf("an hour before: %v, want %v", jobs, want)
	}

	return early
}

// checkSentOnce runs the day-before reminder's sends twice: one e-mail with both times, one push, one link.
func checkSentOnce(t *testing.T, s *reminderStack, b booking, early visit) {
	t.Helper()
	ctx := t.Context()
	dayBefore := time.Date(2026, 10, 24, 4, 30, 0, 0, time.UTC)
	// Each send run twice goes once: one e-mail with both times while abroad, one push.
	s.sendAll(t, early.reminders[1440], dayBefore)
	_, deliveries := s.sends(t, early.reminders[1440])
	if want := []string{"email sent", "push sent"}; !slices.Equal(deliveries, want) {
		t.Errorf("deliveries %v, want %v", deliveries, want)
	}
	if len(s.sender.sent) != 1 || len(s.pusher.Sent()) != 1 {
		t.Fatalf("sent %d e-mails and %d pushes, want one each", len(s.sender.sent), len(s.pusher.Sent()))
	}
	body := s.sender.sent[0].Body
	for _, want := range []string{"Sunday 25 October 2026, 05:30 (Europe/Sarajevo)", "Sunday 25 October 2026, 08:30 (Asia/Dubai)", "https://api.example.test/a/"} {
		if !strings.Contains(body, want) {
			t.Errorf("the e-mail lacks %q:\n%s", want, body)
		}
	}
	for _, never := range []string{"Cy", "Ana", b.email[:8], early.id} {
		if strings.Contains(body+s.sender.sent[0].Subject, never) {
			t.Errorf("the e-mail holds %q", never)
		}
	}
	var tokens int
	if err := s.owner.QueryRow(ctx, `SELECT count(*) FROM appointment_action_token WHERE reminder_id = $1 AND channel = 'email' AND language = 'en'`,
		early.reminders[1440]).Scan(&tokens); err != nil || tokens != 1 {
		t.Errorf("%d one-time links of the e-mail (%v), want 1", tokens, err)
	}

}

// checkAtHome dispatches the reminders of a visit once the patient reports no zone: the clinic's quiet hours.
func checkAtHome(t *testing.T, s *reminderStack, b booking, role, account string) {
	t.Helper()
	ctx := t.Context()
	// The patient is back home and reports no zone: the clinic's. 21:00 UTC on 25 October is 22:00 CET in Sarajevo.
	if _, err := s.owner.Exec(ctx, `UPDATE notification_preference SET time_zone = NULL WHERE account_id = $1`, account); err != nil {
		t.Fatal(err)
	}
	late := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 25, 21, 0, 0, 0, time.UTC))
	// The day before, 21:00 UTC is 23:00 CEST: quiet until 08:00 the next morning, which after the change is 07:00 UTC.
	if err := s.dispatch(t, late, 1440, time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	jobs, _ := s.sends(t, late.reminders[1440])
	if want := []string{"reminder_email at 2026-10-24T21:00Z", "reminder_push at 2026-10-25T07:00Z"}; !slices.Equal(jobs, want) {
		t.Errorf("the day before, at home: %v, want %v", jobs, want)
	}
	// An hour before, 20:00 UTC is 21:00 CET: before the quiet hours, at once.
	if err := s.dispatch(t, late, 60, time.Date(2026, 10, 25, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	jobs, _ = s.sends(t, late.reminders[60])
	if want := []string{"reminder_email at 2026-10-25T20:00Z", "reminder_push at 2026-10-25T20:00Z"}; !slices.Equal(jobs, want) {
		t.Errorf("an hour before, at home: %v, want %v", jobs, want)
	}

}

// checkCancelledAndMoved: a visit cancelled and one rescheduled between scheduling and dispatch send nothing more.
func checkCancelledAndMoved(t *testing.T, s *reminderStack, b booking, role string) {
	t.Helper()
	ctx := t.Context()
	// Cancelled by the clinic between scheduling and dispatch: its reminders are cancelled in the cancellation's
	// transaction, and a dispatch finds nothing to do. Moved (a reschedule) after its first reminder went: the next is
	// skipped, and the pending push of the earlier is cancelled when its job runs.
	cancelled := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC))
	s.clock.set(time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	var admin string
	if err := s.owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id) VALUES ('admin-r@example.test', 'h', 'admin', $1) RETURNING id::text`,
		clinicID).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	if _, err := s.appointments.Cancel(ctx, appointment.Caller{AccountID: admin, OrganizationID: clinicID, Admin: true}, cancelled.id); err != nil {
		t.Fatal(err)
	}
	if st := s.state(t, cancelled.reminders[1440]); st != "cancelled" {
		t.Errorf("a cancelled appointment's reminder: %s, want cancelled", st)
	}
	if err := s.dispatch(t, cancelled, 1440, time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := s.sends(t, cancelled.reminders[1440]); len(jobs) != 0 {
		t.Errorf("a cancelled appointment's reminder sends %v", jobs)
	}

	moved := s.seedVisit(t, role, b.patient, time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)) // 04:00 CET: quiet
	if err := s.dispatch(t, moved, 1440, time.Date(2026, 10, 27, 3, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.owner.Exec(ctx, `UPDATE appointment SET period = tstzrange(lower(period) + interval '1 day', upper(period) + interval '1 day'),
		slot_id = gen_random_uuid(), reschedule_count = reschedule_count + 1 WHERE id = $1`, moved.id); err != nil {
		t.Fatal(err)
	}
	if err := s.dispatch(t, moved, 60, time.Date(2026, 10, 28, 2, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if st := s.state(t, moved.reminders[60]); st != "skipped" {
		t.Errorf("a moved appointment's next reminder: %s, want skipped", st)
	}
	s.sendAll(t, moved.reminders[1440], time.Date(2026, 10, 27, 7, 0, 0, 0, time.UTC))
	if _, deliveries := s.sends(t, moved.reminders[1440]); !slices.Equal(deliveries, []string{"email cancelled", "push cancelled"}) {
		t.Errorf("a moved appointment's earlier reminder: %v, want its sends cancelled", deliveries)
	}
	if len(s.sender.sent) != 1 || len(s.pusher.Sent()) != 1 {
		t.Errorf("sent %d e-mails and %d pushes in all, want the first reminder's only", len(s.sender.sent), len(s.pusher.Sent()))
	}
}
