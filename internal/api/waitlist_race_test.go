package api_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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
	"github.com/Shmeeshmee/salus/pkg/payments/paymentstest"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// waitlistAt is the appointment service with the waitlist as internal/api and the worker compose it, on the salus_app
// login, at clk.
func waitlistAt(t *testing.T, app *pgxpool.Pool, clk clock.Clock, change ...func(p *appointment.Ports)) *appointment.Service {
	t.Helper()
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clk)
	jobs, err := worker.NewInserter(app)
	if err != nil {
		t.Fatal(err)
	}
	clinics := organization.NewService(organization.NewPostgresRepository(app), audit, tx, clk, nil)
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk, Clinics: clinics,
		People: practitioner.NewService(practitioner.NewPostgresRepository(app), audit, tx),
	})
	schedules := schedule.NewService(schedule.Ports{Repo: schedule.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk})
	slots := slot.NewService(slot.NewPostgresRepository(app), tx, clk, schedules, clinics)
	consents := consent.NewService(consent.NewPostgresRepository(app), audit, tx, clk, consent.NewRiverJobs(jobs))
	repo := appointment.NewPostgresRepository(app)
	p := appointment.Ports{
		Repo: repo, Audit: audit, Tx: tx, Clock: clk, Consents: consents, Slots: slots, Clinics: clinics, Roles: roles,
		Patients: patient.NewService(patient.NewPostgresRepository(app), audit, tx, clk, consents, appointment.NewFacts(repo)),
		Jobs:     appointment.NewRiverJobs(jobs), Stats: clinicstats.NewRiverQueue(jobs),
		Waitlist: appointment.NewPostgresWaitlistRepository(app), Offered: slots,
		Settings: organization.NewAppointments(organization.AppointmentsPorts{
			Repo: organization.NewPostgresAppointmentsRepository(app), Audit: audit, Tx: tx, Clock: clk,
		}),
		OfferJobs: appointment.NewRiverOfferJobs(jobs), Notices: appointment.NewRiverOfferNotices(jobs),
	}
	for _, c := range change {
		c(&p)
	}
	return appointment.NewService(p)
}

// waitlistClinic is seedClinic's clinic with its waitlist on, n patients each waiting for its role over the next
// ten days (in the order of the patients), and a free slot at each offset.
func waitlistClinic(t *testing.T, owner *pgxpool.Pool, n int, offsets ...time.Duration) clinic {
	t.Helper()
	c := seedClinic(t, owner, n, offsets...)
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization_appointment_settings (organization_id, waitlist_enabled) VALUES ($1, true)`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	for i, p := range c.patients {
		if _, err := owner.Exec(t.Context(), `INSERT INTO waitlist_entry (created_at, patient_id, organization_id, practitioner_role_id, earliest, latest, ends_at)
			VALUES (now() - interval '1 day' + $4 * interval '1 second', $1, $2, $3, current_date - 1, current_date + 10, current_date + 11)`,
			p, inviteClinic, c.role, i); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// checkWaitlistInStep fails unless, in the clinic, a slot is busy exactly when an appointment that is not cancelled
// holds it, and busy-tentative exactly when an open offer holds it; no slot has two of either.
func checkWaitlistInStep(t *testing.T, owner *pgxpool.Pool, round int) {
	t.Helper()
	var broken, doubled int
	if err := owner.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM slot s WHERE s.organization_id = $1 AND (
			(s.status = 'busy') <> EXISTS (SELECT 1 FROM appointment a WHERE a.slot_id = s.id AND a.status <> 'cancelled')
			OR (s.status = 'busy-tentative') <> EXISTS (SELECT 1 FROM waitlist_offer o WHERE o.slot_id = s.id AND o.state = 'open'))),
		(SELECT count(*) FROM (SELECT slot_id FROM appointment WHERE status <> 'cancelled' GROUP BY slot_id HAVING count(*) > 1) d)
		+ (SELECT count(*) FROM (SELECT slot_id FROM waitlist_offer WHERE state = 'open' GROUP BY slot_id HAVING count(*) > 1) o)
		+ (SELECT count(*) FROM waitlist_offer o JOIN waitlist_entry e ON e.id = o.entry_id WHERE o.state = 'open' AND e.status <> 'active')`,
		inviteClinic).Scan(&broken, &doubled); err != nil {
		t.Fatal(err)
	}
	if broken != 0 || doubled != 0 {
		t.Errorf("round %d: %d slots out of step with their appointments and offers, %d held twice or by a closed entry", round, broken, doubled)
	}
}

// openOffer is the open offer of the slot, as the owner reads it ("" for none), and its patient.
func openOffer(t *testing.T, owner *pgxpool.Pool, slotID string) (id, patientID string, expires time.Time) {
	t.Helper()
	err := owner.QueryRow(t.Context(), `SELECT id::text, patient_id::text, expires_at FROM waitlist_offer WHERE slot_id = $1 AND state = 'open'`, slotID).
		Scan(&id, &patientID, &expires)
	if err != nil {
		return "", "", time.Time{}
	}
	return id, patientID, expires
}

func callerOf(c clinic, patientID string) appointment.Caller {
	for i, p := range c.patients {
		if p == patientID {
			return appointment.Caller{AccountID: c.accounts[i], PatientID: p}
		}
	}
	return appointment.Caller{}
}

// atOnce runs each of fns at once and returns their errors.
func atOnce(fns ...func() error) []error {
	errs := make([]error, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		wg.Go(func() { errs[i] = fn() })
	}
	wg.Wait()
	return errs
}

// The patient accepts just before the hold ends while the expiry runs just after: the offer's row lock serialises
// them, so the slot is either booked by the acceptance or handed to the next entry, never both.
func TestWaitlist_AnAcceptanceRacingTheExpiryBooksOrHandsOnNeverBoth(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 8 {
		offsets = append(offsets, 72*time.Hour+time.Duration(i)*time.Hour)
	}
	c := waitlistClinic(t, owner, 20, offsets...)
	jobs := waitlistAt(t, app, clock.System{})
	var accepted, expired int
	for round, s := range c.slots {
		if err := jobs.HandOn(t.Context(), inviteClinic, s); err != nil {
			t.Fatal(err)
		}
		offer, p, expires := openOffer(t, owner, s)
		if offer == "" {
			t.Fatalf("round %d: no offer", round)
		}
		early, late := waitlistAt(t, app, clock.Fixed{T: expires.Add(-time.Millisecond)}), waitlistAt(t, app, clock.Fixed{T: expires.Add(time.Millisecond)})
		// The expiry starts later each round, from at once to well after the acceptance: the rounds between meet on
		// the offer's lock.
		delay := []time.Duration{0, time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond, 8 * time.Millisecond,
			16 * time.Millisecond, 32 * time.Millisecond, 250 * time.Millisecond}[round]
		errs := atOnce(
			func() error { _, err := early.Accept(t.Context(), callerOf(c, p), offer); return err },
			func() error { time.Sleep(delay); return late.ExpireOffer(t.Context(), inviteClinic, offer) },
		)
		switch {
		case errs[1] != nil:
			t.Fatalf("round %d: ExpireOffer() = %v", round, errs[1])
		case errs[0] == nil:
			accepted++
		case errors.Is(errs[0], appointment.ErrOfferExpired):
			expired++
		default:
			t.Fatalf("round %d: Accept() = %v", round, errs[0])
		}
		checkWaitlistInStep(t, owner, round)
	}
	t.Logf("%d accepted, %d expired first", accepted, expired)
	if accepted+expired != len(c.slots) || accepted == 0 || expired == 0 {
		t.Errorf("%d accepted and %d expired over %d rounds, want both outcomes", accepted, expired, len(c.slots))
	}
}

// One patient accepts on two devices at once: one appointment, which both devices get; with a price, one charge and
// no refund (review of PR #93, findings 1 and 7).
func TestWaitlist_TwoAcceptancesBookAndChargeOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := waitlistClinic(t, owner, 2, 72*time.Hour, 73*time.Hour, 74*time.Hour, 75*time.Hour, 76*time.Hour)
	provider := &paymentstest.Provider{}
	svc := waitlistAt(t, app, clock.System{}, func(p *appointment.Ports) { p.Prices, p.Payments = flatPrice{}, provider })
	for round, s := range c.slots {
		if err := svc.HandOn(t.Context(), inviteClinic, s); err != nil {
			t.Fatal(err)
		}
		offer, p, _ := openOffer(t, owner, s)
		ids := make([]string, 2)
		accept := func(i int) func() error {
			return func() error { a, err := svc.Accept(t.Context(), callerOf(c, p), offer); ids[i] = a.ID; return err }
		}
		if errs := atOnce(accept(0), accept(1)); errs[0] != nil || errs[1] != nil || ids[0] != ids[1] {
			t.Fatalf("round %d: Accept() twice = %v (%s), %v (%s); want one appointment for both", round, errs[0], ids[0], errs[1], ids[1])
		}
		var n int
		if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM appointment WHERE slot_id = $1 AND booked_via = 'waitlist'`, s).Scan(&n); err != nil || n != 1 {
			t.Errorf("round %d: %d appointments of the slot, want one", round, n)
		}
		checkWaitlistInStep(t, owner, round)
		if _, err := owner.Exec(t.Context(), `UPDATE waitlist_entry SET status = 'active', closed_at = NULL`); err != nil {
			t.Fatal(err)
		}
	}
	charged := provider.Charged()
	if len(charged) != len(c.slots) {
		t.Errorf("%d receipts over %d rounds, want one each", len(charged), len(c.slots))
	}
	for _, r := range charged {
		if provider.Refunded(r.ID) != 0 {
			t.Errorf("receipt %s refunded: a booked appointment left unpaid", r.Key)
		}
	}
}

// flatPrice prices every slot at 30 BAM.
type flatPrice struct{}

func (flatPrice) Price(context.Context, slot.Bookable) (int64, string, error) {
	return 3000, "BAM", nil
}

// The patient accepts while they (or staff) withdraw the entry: the acceptance locks the entry before the offer, as the
// withdrawal does, so one of them wins and neither deadlocks (review of PR #93, finding 2).
func TestWaitlist_AnAcceptanceRacingAWithdrawalNeverDeadlocks(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 8 {
		offsets = append(offsets, 72*time.Hour+time.Duration(i)*time.Hour)
	}
	c := waitlistClinic(t, owner, 2, offsets...)
	svc := waitlistAt(t, app, clock.System{})
	for round, s := range c.slots {
		if err := svc.HandOn(t.Context(), inviteClinic, s); err != nil {
			t.Fatal(err)
		}
		offer, p, _ := openOffer(t, owner, s)
		var entry string
		if err := owner.QueryRow(t.Context(), `SELECT entry_id::text FROM waitlist_offer WHERE id = $1`, offer).Scan(&entry); err != nil {
			t.Fatal(err)
		}
		errs := atOnce(
			func() error { _, err := svc.Accept(t.Context(), callerOf(c, p), offer); return err },
			func() error { return svc.Withdraw(t.Context(), callerOf(c, p), entry) },
		)
		accepted, withdrawn := errs[0] == nil, errs[1] == nil
		if accepted == withdrawn || (!accepted && !errors.Is(errs[0], appointment.ErrOfferExpired)) || (!withdrawn && !errors.Is(errs[1], appointment.ErrNotFound)) {
			t.Fatalf("round %d: Accept() = %v, Withdraw() = %v; want one of them, the other refused", round, errs[0], errs[1])
		}
		if withdrawn { // the withdrawal's waitlist_slot_freed
			if err := svc.HandOn(t.Context(), inviteClinic, s); err != nil {
				t.Fatal(err)
			}
		}
		checkWaitlistInStep(t, owner, round)
		if _, err := owner.Exec(t.Context(), `UPDATE waitlist_offer SET state = 'expired', closed_at = now() WHERE state = 'open';
			UPDATE waitlist_entry SET status = 'active', closed_at = NULL`); err != nil {
			t.Fatal(err)
		}
		if _, err := owner.Exec(t.Context(), `UPDATE slot SET status = 'free' WHERE status = 'busy-tentative'`); err != nil {
			t.Fatal(err)
		}
	}
}

// Joins of one patient with one clinic at once keep the limit of five active entries (review of PR #93, finding 3).
func TestWaitlist_ConcurrentJoinsKeepTheLimitOfFive(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := waitlistClinic(t, owner, 1)
	if _, err := owner.Exec(t.Context(), `INSERT INTO waitlist_entry (patient_id, organization_id, practitioner_role_id, earliest, latest, ends_at)
		SELECT patient_id, organization_id, practitioner_role_id, earliest, latest, ends_at FROM waitlist_entry, generate_series(1, 3)`); err != nil {
		t.Fatal(err)
	}
	svc := waitlistAt(t, app, clock.System{})
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	today := clock.System{}.Now().In(loc)
	in := appointment.NewEntry{
		OrganizationID: inviteClinic, PractitionerRoleID: c.role,
		Earliest: time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC),
		Latest:   time.Date(today.Year(), today.Month(), today.Day()+5, 0, 0, 0, 0, time.UTC),
	}
	join := func() error { _, err := svc.Join(t.Context(), callerOf(c, c.patients[0]), in); return err }
	var joined, refused int
	for _, err := range atOnce(join, join, join, join, join, join) {
		switch {
		case err == nil:
			joined++
		case errors.Is(err, appointment.ErrTooManyEntries):
			refused++
		default:
			t.Fatalf("Join() = %v", err)
		}
	}
	if joined != 1 || refused != 5 {
		t.Errorf("%d joined and %d refused at once, want 1 and 5", joined, refused)
	}
}

// Two slots freed at once, one patient waiting: the first job locks the entry, the second passes it over, so the
// entry gets one offer and neither job fails on the unique index (review of PR #93, finding 4).
func TestWaitlist_TwoSlotsFreedAtOnceOfferTheOneEntryOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 12 {
		offsets = append(offsets, 72*time.Hour+time.Duration(i)*time.Hour)
	}
	c := waitlistClinic(t, owner, 1, offsets...)
	svc := waitlistAt(t, app, clock.System{})
	for round := range len(c.slots) / 2 {
		a, b := c.slots[2*round], c.slots[2*round+1]
		if err := errors.Join(atOnce(
			func() error { return svc.HandOn(t.Context(), inviteClinic, a) },
			func() error { return svc.HandOn(t.Context(), inviteClinic, b) },
		)...); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		var open int
		if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM waitlist_offer WHERE state = 'open'`).Scan(&open); err != nil || open != 1 {
			t.Errorf("round %d: %d open offers, want one", round, open)
		}
		checkWaitlistInStep(t, owner, round)
		if _, err := owner.Exec(t.Context(), `UPDATE waitlist_offer SET state = 'declined', closed_at = now() WHERE state = 'open';
			UPDATE slot SET status = 'free' WHERE status = 'busy-tentative'`); err != nil {
			t.Fatal(err)
		}
	}
}

// A direct booking of the offered slot never sees it (the directory shows free slots only, the hold takes a free one),
// while the offer's patient books it.
func TestWaitlist_ADirectBookingNeverTakesAnOfferedSlot(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := waitlistClinic(t, owner, 2, 72*time.Hour, 73*time.Hour, 74*time.Hour, 75*time.Hour)
	svc := waitlistAt(t, app, clock.System{})
	if _, err := owner.Exec(t.Context(), `DELETE FROM waitlist_entry WHERE patient_id = $1`, c.patients[1]); err != nil {
		t.Fatal(err)
	}
	direct := appointment.Caller{AccountID: c.accounts[1], PatientID: c.patients[1]}
	for round, s := range c.slots {
		if err := svc.HandOn(t.Context(), inviteClinic, s); err != nil {
			t.Fatal(err)
		}
		offer, p, _ := openOffer(t, owner, s)
		errs := atOnce(
			func() error { _, err := svc.Accept(t.Context(), callerOf(c, p), offer); return err },
			func() error { _, err := svc.Book(t.Context(), direct, s); return err },
		)
		if errs[0] != nil || !errors.Is(errs[1], appointment.ErrSlotUnavailable) {
			t.Fatalf("round %d: Accept() = %v, Book() = %v; want the acceptance, and the direct booking refused", round, errs[0], errs[1])
		}
		checkWaitlistInStep(t, owner, round)
		if _, err := owner.Exec(t.Context(), `UPDATE waitlist_entry SET status = 'active', closed_at = NULL WHERE patient_id = $1`, c.patients[0]); err != nil {
			t.Fatal(err)
		}
	}
}

// A patient cancels while a freed-slot job of their slot runs, and the cancellation's own job then runs: whichever
// order the two take, the slot ends up held for one waiting patient, never for the one who freed it.
func TestWaitlist_ACancellationRacingAnOfferKeepsTheSlotInStep(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 6 {
		offsets = append(offsets, 72*time.Hour+time.Duration(i)*time.Hour)
	}
	c := waitlistClinic(t, owner, 3, offsets...)
	svc := waitlistAt(t, app, clock.System{})
	holder := appointment.Caller{AccountID: c.accounts[2], PatientID: c.patients[2]}
	for round, s := range c.slots {
		a, err := svc.Book(t.Context(), holder, s)
		if err != nil {
			t.Fatalf("round %d: Book() = %v", round, err)
		}
		errs := atOnce(
			func() error { _, err := svc.Cancel(t.Context(), holder, a.ID); return err },
			func() error { return svc.HandOn(t.Context(), inviteClinic, s) },
		)
		if err := errors.Join(errs...); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if err := svc.HandOn(t.Context(), inviteClinic, s); err != nil { // the cancellation's waitlist_slot_freed
			t.Fatal(err)
		}
		offer, p, _ := openOffer(t, owner, s)
		if offer == "" || p == holder.PatientID {
			t.Errorf("round %d: offer %q to %s; want one to a waiting patient other than the one who cancelled", round, offer, p)
		}
		checkWaitlistInStep(t, owner, round)
		if _, err := svc.Accept(t.Context(), callerOf(c, p), offer); err != nil { // the next round's slot needs the entry again
			t.Fatal(err)
		}
		if _, err := owner.Exec(t.Context(), `UPDATE waitlist_entry SET status = 'active', closed_at = NULL`); err != nil {
			t.Fatal(err)
		}
	}
}

// Two freed-slot jobs of one slot at once: the slot's lock makes the second find it held, so it is offered once.
func TestWaitlist_TwoFreedSlotJobsOfferTheSlotOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var offsets []time.Duration
	for i := range 10 {
		offsets = append(offsets, 72*time.Hour+time.Duration(i)*time.Hour)
	}
	c := waitlistClinic(t, owner, 12, offsets...)
	svc := waitlistAt(t, app, clock.System{})
	for round, s := range c.slots {
		hand := func() error { return svc.HandOn(t.Context(), inviteClinic, s) }
		if err := errors.Join(atOnce(hand, hand)...); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		var n int
		if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM waitlist_offer WHERE slot_id = $1`, s).Scan(&n); err != nil || n != 1 {
			t.Errorf("round %d: %d offers of the slot, want one", round, n)
		}
		checkWaitlistInStep(t, owner, round)
	}
	// The ten offers went to ten entries in order: one open offer per entry.
	var entries int
	if err := owner.QueryRow(t.Context(), `SELECT count(DISTINCT entry_id) FROM waitlist_offer`).Scan(&entries); err != nil || entries != len(c.slots) {
		t.Errorf("%d entries offered over %d slots, want one each", entries, len(c.slots))
	}
}

// A page of the clinic's list with 10 000 entries (2 000 patients with the care consent, five entries each), as the
// admin and as the practitioner read it: measured for docs/plans/appointments.PROGRESS.md, and bounded far above what
// it takes so a regression to reading every entry fails (review of PR #93, finding 5).
func TestWaitlist_APageOfTheClinicsListAtTenThousandEntries(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	c := waitlistClinic(t, owner, 0)
	if _, err := owner.Exec(t.Context(), `
		INSERT INTO account (email, password_hash, role) SELECT 'wl' || i || '@example.test', 'h', 'patient' FROM generate_series(1, 2000) i;
		INSERT INTO patient (account_id, given_name, family_name, birth_date)
			SELECT id, 'Ada', 'Example', '1990-01-01' FROM account WHERE email LIKE 'wl%';
		INSERT INTO consent (patient_id, organization_id, status, date_time)
			SELECT p.id, '`+inviteClinic+`', 'active', now() - interval '1 day' FROM patient p JOIN account a ON a.id = p.account_id WHERE a.email LIKE 'wl%';
		INSERT INTO waitlist_entry (created_at, ends_at, patient_id, organization_id, practitioner_role_id, earliest, latest, priority)
			SELECT now() - (n * 5 + k) * interval '1 minute', current_date + 30, p.id, '`+inviteClinic+`', '`+c.role+`',
			       current_date, current_date + 29, (n * 7 + k) % 4
			FROM (SELECT id, row_number() OVER (ORDER BY id) AS n FROM patient) p, generate_series(0, 4) k;
		ANALYZE;`); err != nil {
		t.Fatal(err)
	}
	svc := waitlistAt(t, app, clock.System{})
	for name, who := range map[string]appointment.Caller{
		"admin":        {AccountID: c.practitionerAccount, OrganizationID: inviteClinic, Admin: true},
		"practitioner": {AccountID: c.practitionerAccount, OrganizationID: inviteClinic, PractitionerID: c.practitioner},
	} {
		var took []time.Duration
		for range 5 {
			began := clock.System{}.Now()
			page, err := svc.ClinicWaitlist(t.Context(), who, appointment.WaitlistFilter{Offset: 5000})
			took = append(took, clock.System{}.Now().Sub(began))
			if err != nil || len(page) != 100 {
				t.Fatalf("%s: a page = %d entries, %v; want 100", name, len(page), err)
			}
		}
		slices.Sort(took)
		t.Logf("%s: a page of 100 of 10 000 entries: %v (median %v)", name, took, took[2])
		if took[2] > 2*time.Second {
			t.Errorf("%s: a page takes %v, want well under two seconds", name, took[2])
		}
	}
}
