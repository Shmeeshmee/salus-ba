package api_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// countingRepository counts the bookings the exclusion constraint refused.
type countingRepository struct {
	appointment.Repository
	excluded atomic.Int64
}

func (r *countingRepository) Create(ctx context.Context, a appointment.Appointment) (appointment.Appointment, error) {
	out, err := r.Repository.Create(ctx, a)
	if errors.Is(err, appointment.ErrSlotUnavailable) {
		r.excluded.Add(1)
	}
	return out, err
}

// booking is the services as internal/api composes them, on the salus_app login (row-level security on).
type booking struct {
	consents     *consent.Service
	patients     *patient.Service
	appointments *appointment.Service
	repo         *countingRepository
}

func newBooking(t *testing.T, app *pgxpool.Pool) booking {
	t.Helper()
	clk := clock.System{}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clk)
	jobs, err := worker.NewInserter(app)
	if err != nil {
		t.Fatal(err)
	}
	clinics := organization.NewService(organization.NewPostgresRepository(app), audit, tx, clk, nil)
	practitioners := practitioner.NewService(practitioner.NewPostgresRepository(app), audit, tx)
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk, Clinics: clinics, People: practitioners,
	})
	schedules := schedule.NewService(schedule.Ports{Repo: schedule.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk})
	b := booking{repo: &countingRepository{Repository: appointment.NewPostgresRepository(app)}}
	b.consents = consent.NewService(consent.NewPostgresRepository(app), audit, tx, clk, consent.NewRiverJobs(jobs))
	b.patients = patient.NewService(patient.NewPostgresRepository(app), audit, tx, clk, b.consents, appointment.NewFacts(b.repo))
	b.appointments = appointment.NewService(appointment.Ports{
		Repo: b.repo, Audit: audit, Tx: tx, Clock: clk, Consents: b.consents,
		Slots: slot.NewService(slot.NewPostgresRepository(app), tx, clk, schedules, clinics), Patients: b.patients,
		Clinics: clinics, Roles: roles, Jobs: appointment.NewRiverJobs(jobs), Stats: clinicstats.NewRiverQueue(jobs),
		Reminders: appointment.NewPostgresReminderRepository(app), Rules: organization.NewAppointments(organization.AppointmentsPorts{Repo: organization.NewPostgresAppointmentsRepository(app), Audit: audit, Tx: tx, Clock: clk}),
		SelfService: appointment.NewPostgresSelfServiceRepository(app), // the patient's reschedule (SAL-34)
	})
	return b
}

// clinic is what seedClinic created.
type clinic struct {
	role, practitioner, practitionerAccount string
	patients, accounts                      []string
	slots                                   []string
}

// seedClinic creates, as the owner, a listed clinic with a practitioner (and their account), a schedule with
// a free half-hour slot at each offset from now, and n patients with an active consent with it.
func seedClinic(t *testing.T, owner *pgxpool.Pool, n int, offsets ...time.Duration) clinic {
	t.Helper()
	ctx := t.Context()
	var c clinic
	var sched string
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id, practitioner_id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, practitioner_id) SELECT 'cy@example.test', 'h', 'practitioner', organization_id, practitioner_id FROM r RETURNING id),
		s AS (INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) SELECT organization_id, id, 30 FROM r RETURNING id)
		SELECT (SELECT id::text FROM r), (SELECT practitioner_id::text FROM r), (SELECT id::text FROM a), (SELECT id::text FROM s)`, inviteClinic).
		Scan(&c.role, &c.practitioner, &c.practitionerAccount, &sched)
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		var account, id string
		if err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES ($1, 'h', 'patient') RETURNING id),
			p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Zorana', 'Syntheticova', '1977-07-07' FROM a RETURNING id, account_id)
			INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $2, 'active', now() - interval '1 day' FROM p
			RETURNING (SELECT account_id::text FROM p), patient_id::text`, fmt.Sprintf("patient-%d@example.test", i), inviteClinic).Scan(&account, &id); err != nil {
			t.Fatal(err)
		}
		c.accounts, c.patients = append(c.accounts, account), append(c.patients, id)
	}
	for _, off := range offsets {
		var id string
		if err := owner.QueryRow(ctx, `INSERT INTO slot (schedule_id, organization_id, period)
			VALUES ($1, $2, tstzrange(date_trunc('minute', now()) + $3::interval, date_trunc('minute', now()) + $3::interval + interval '30 minutes'))
			RETURNING id`, sched, inviteClinic, fmt.Sprintf("%d seconds", int(off.Seconds()))).Scan(&id); err != nil {
			t.Fatal(err)
		}
		c.slots = append(c.slots, id)
	}
	return c
}

// Book and cancel run at once over a set of slots for many rounds: after each round a slot is busy exactly
// when an appointment that is not cancelled holds it, no slot is held twice, and the exclusion constraint
// never had to refuse a booking (the slot's hold, in the same transaction, refused every taken one first).
func TestBooking_ConcurrentBookAndCancelKeepSlotsAndAppointmentsInStep(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 6 {
		offsets = append(offsets, 48*time.Hour+time.Duration(i)*time.Hour)
	}
	c := seedClinic(t, owner, 2, offsets...)
	b := newBooking(t, app)
	patients := []appointment.Caller{
		{AccountID: c.accounts[0], PatientID: c.patients[0]}, {AccountID: c.accounts[1], PatientID: c.patients[1]},
	}
	holder := map[string]appointment.Appointment{} // the booked appointment of each slot

	for round := range 25 {
		bookAndCancel(t, b, patients, c.slots, holder, round)
		forgetCancelled(t, owner, holder)
		checkInStep(t, owner, round)
	}
	if n := b.repo.excluded.Load(); n != 0 {
		t.Errorf("the exclusion constraint refused %d bookings of a free slot, want none", n)
	}
	var booked int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM appointment").Scan(&booked); err != nil || booked < len(c.slots) {
		t.Errorf("%d appointments over the rounds (%v), want the rounds to have booked", booked, err)
	}
}

// Right after the revocation commits, before its job runs, the clinic no longer sees the profile or the
// future appointment, and the practitioner still sees the past one with the name, times and status only.
// The job then cancels the future one.
func TestBooking_ARevocationEndsTheClinicsAccessAtOnceAndItsJobCancelsTheFuture(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := seedClinic(t, owner, 1, 7*24*time.Hour)
	b := newBooking(t, app)
	me := appointment.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]}
	pract := appointment.Caller{AccountID: c.practitionerAccount, OrganizationID: inviteClinic, PractitionerID: c.practitioner}
	practForProfile := patient.Caller{AccountID: c.practitionerAccount, OrganizationID: inviteClinic, PractitionerRoleID: c.role}

	future, err := b.appointments.Book(t.Context(), me, c.slots[0])
	if err != nil {
		t.Fatalf("Book() = %v", err)
	}
	var past string
	if err := owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange(now() - interval '2 hours', now() - interval '90 minutes'), now() - interval '1 day')
		RETURNING id`, c.patients[0], c.role, inviteClinic).Scan(&past); err != nil {
		t.Fatal(err)
	}
	if _, err := b.patients.Read(t.Context(), practForProfile, c.patients[0]); err != nil {
		t.Fatalf("the practitioner's read of the profile with the consent = %v, want it", err)
	}

	var consentID string
	if err := owner.QueryRow(t.Context(), "SELECT id FROM consent WHERE patient_id = $1", c.patients[0]).Scan(&consentID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.consents.Revoke(t.Context(), consent.Caller{AccountID: c.accounts[0], PatientID: c.patients[0]}, consentID); err != nil {
		t.Fatalf("Revoke() = %v", err)
	}
	var jobArgs string
	if err := owner.QueryRow(t.Context(), "SELECT args::text FROM river_job WHERE kind = 'consent_revoked'").Scan(&jobArgs); err != nil ||
		jobArgs != `{"patient_id": "`+c.patients[0]+`", "organization_id": "`+inviteClinic+`"}` {
		t.Errorf("the consent_revoked job's arguments = %s (%v), want the two ids only", jobArgs, err)
	}

	if _, err := b.patients.Read(t.Context(), practForProfile, c.patients[0]); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("the practitioner's read of the profile after the revocation = %v, want ErrNotFound", err)
	}
	if _, err := b.appointments.Read(t.Context(), pract, future.ID); !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("the practitioner's read of the future appointment after the revocation = %v, want ErrNotFound", err)
	}
	seen, err := b.appointments.Read(t.Context(), pract, past)
	if err != nil || seen.Level != access.Name || seen.Patient.Family != "Syntheticova" || seen.Appointment.SlotID != "" || seen.Appointment.Status != appointment.StatusBooked {
		t.Errorf("the practitioner's read of the past appointment = %+v, %v; want the name, times and status only", seen, err)
	}

	n, err := b.appointments.CancelForRevokedConsent(t.Context(), c.patients[0], inviteClinic)
	if err != nil || n != 1 {
		t.Fatalf("CancelForRevokedConsent() = %d, %v; want the future appointment", n, err)
	}
	var status, slotStatus string
	if err := owner.QueryRow(t.Context(), `SELECT a.status, s.status FROM appointment a JOIN slot s ON s.id = a.slot_id WHERE a.id = $1`, future.ID).
		Scan(&status, &slotStatus); err != nil || status != "cancelled" || slotStatus != "free" {
		t.Errorf("the future appointment is %s with its slot %s (%v), want cancelled and free", status, slotStatus, err)
	}
	if n, err := b.appointments.CancelForRevokedConsent(t.Context(), c.patients[0], inviteClinic); err != nil || n != 0 {
		t.Errorf("a retry = %d, %v; want nothing", n, err)
	}
	var mails int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM river_job WHERE kind = 'appointment_cancelled_email'").Scan(&mails); err != nil || mails != 1 {
		t.Errorf("%d cancellation e-mails (%v), want one", mails, err)
	}
	if strings.Contains(jobArgs, "Zorana") {
		t.Error("a job argument holds the name")
	}
}

// bookAndCancel runs one round: for each slot at once, the cancellation of the appointment holding it (if
// any) and a booking of it, by one of the patients in turn. A booking that wins is recorded in holder.
func bookAndCancel(t *testing.T, b booking, patients []appointment.Caller, slots []string, holder map[string]appointment.Appointment, round int) {
	t.Helper()
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, sl := range slots {
		if a, ok := holder[sl]; ok {
			who := patients[0]
			if a.PatientID == patients[1].PatientID {
				who = patients[1]
			}
			wg.Go(func() {
				if _, err := b.appointments.Cancel(t.Context(), who, a.ID); err != nil {
					t.Errorf("round %d: Cancel() = %v", round, err)
				}
			})
		}
		wg.Go(func() {
			a, err := b.appointments.Book(t.Context(), patients[(i+round)%2], sl)
			switch {
			case errors.Is(err, appointment.ErrSlotUnavailable):
			case err != nil:
				t.Errorf("round %d: Book() = %v", round, err)
			default:
				mu.Lock()
				holder[sl] = a
				mu.Unlock()
			}
		})
	}
	wg.Wait()
}

// forgetCancelled drops from holder the appointments cancelled in the last round.
func forgetCancelled(t *testing.T, owner *pgxpool.Pool, holder map[string]appointment.Appointment) {
	t.Helper()
	for sl, a := range holder {
		var status string
		if err := owner.QueryRow(t.Context(), "SELECT status FROM appointment WHERE id = $1", a.ID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status == "cancelled" {
			delete(holder, sl)
		}
	}
}

// checkInStep fails unless every slot is busy exactly when an appointment that is not cancelled holds it, and
// no slot is held by two.
func checkInStep(t *testing.T, owner *pgxpool.Pool, round int) {
	t.Helper()
	var disagree, doubly int
	if err := owner.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM slot s WHERE (s.status = 'busy') <> EXISTS (SELECT 1 FROM appointment a WHERE a.slot_id = s.id AND a.status <> 'cancelled')),
		(SELECT count(*) FROM (SELECT slot_id FROM appointment WHERE status <> 'cancelled' GROUP BY slot_id HAVING count(*) > 1) AS d)`).
		Scan(&disagree, &doubly); err != nil {
		t.Fatal(err)
	}
	if disagree != 0 || doubly != 0 {
		t.Fatalf("round %d: %d slots disagree with their appointments and %d are held twice, want none", round, disagree, doubly)
	}
}
