package appointment_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const generalPractice = "394814009"

// The waitlist rows of a test: clinic A's role A1 (general practice), a second role of another specialty, and
// patients with clinic A's care consent unless said otherwise.
type waitlistDB struct {
	owner, app *pgxpool.Pool
	s          seeded
	roleOther  string
}

func newWaitlistDB(t *testing.T) waitlistDB {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	w := waitlistDB{owner: owner, app: app, s: seedDB(t, owner)}
	ownerExec(t, owner, `UPDATE practitioner_role SET specialty_code = $1 WHERE id = $2`, generalPractice, w.s.roleA1)
	if err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Di', 'Example') RETURNING id)
		INSERT INTO practitioner_role (organization_id, practitioner_id, specialty_code) SELECT $1, id, '394579002' FROM p RETURNING id`, clinicA).Scan(&w.roleOther); err != nil {
		t.Fatal(err)
	}
	return w
}

func ownerExec(t *testing.T, owner *pgxpool.Pool, stmt string, args ...any) {
	t.Helper()
	if _, err := owner.Exec(t.Context(), stmt, args...); err != nil {
		t.Fatal(err)
	}
}

// patient adds a patient, with clinic A's care consent when consented, and returns its id.
func (w waitlistDB) patient(t *testing.T, consented bool) string {
	t.Helper()
	var id string
	if err := w.owner.QueryRow(t.Context(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id)
		INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if consented {
		ownerExec(t, w.owner, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`, id, clinicA)
	}
	return id
}

// entry is an entry's columns for the owner's insert; zero values take the defaults of add.
type entry struct {
	created   time.Time
	endsAt    time.Time
	times     *appointment.Times
	patient   string
	role      string
	specialty string
	earliest  string
	latest    string
	status    string
	priority  int
}

// add inserts e as the owner and returns its id: by default role A1, from the 1st to the 31st of January 2030, active,
// ending far ahead.
func (w waitlistDB) add(t *testing.T, e entry) string {
	t.Helper()
	if e.role == "" && e.specialty == "" {
		e.role = w.s.roleA1
	}
	if e.earliest == "" {
		e.earliest, e.latest = "2030-01-01", "2030-01-31"
	}
	if e.created.IsZero() {
		e.created = now
	}
	if e.endsAt.IsZero() {
		e.endsAt = time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	}
	if e.status == "" {
		e.status = "active"
	}
	var start, end any
	if e.times != nil {
		start, end = e.times.Start, e.times.End
	}
	var id string
	err := w.owner.QueryRow(t.Context(), `INSERT INTO waitlist_entry (created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, specialty,
			earliest, latest, times_start_minute, times_end_minute, priority, status)
		VALUES ($1, $2, CASE WHEN $12 = 'active' THEN NULL ELSE $1::timestamptz END, $3, $4, NULLIF($5, '')::uuid, NULLIF($6, ''), $7, $8, $9, $10, $11, $12) RETURNING id`,
		e.created, e.endsAt, e.patient, clinicA, e.role, e.specialty, e.earliest, e.latest, start, end, e.priority, e.status).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// slotMatch is a freed slot of role A1 on Monday 7 January 2030, 10:00 to 10:30 in Sarajevo.
func (w waitlistDB) slotMatch() appointment.Match {
	return appointment.Match{
		Start: start, End: start.Add(30 * time.Minute), Day: time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC), Now: now,
		OrganizationID: clinicA, SlotID: slotFree, PractitionerRoleID: w.s.roleA1, Specialty: generalPractice,
		Times: appointment.Times{Start: 600, End: 630},
	}
}

func (w waitlistDB) candidates(t *testing.T, m appointment.Match) []string {
	t.Helper()
	var ids []string
	err := postgres.NewTransactor(w.app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := appointment.NewPostgresRepository(w.app).SetClinic(ctx, clinicA); err != nil {
			return err
		}
		got, err := appointment.NewPostgresWaitlistRepository(w.app).Candidates(ctx, m, 50)
		for _, e := range got {
			ids = append(ids, e.ID)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// The freed slot's candidates are the active entries of its clinic for its role or the role's specialty, its day and
// times, in offer order (priority, then who joined first), of patients the clinic sees with the consent, never
// offered this slot nor holding another offer, who never held the slot and have nothing booked at its time.
func TestCandidates_MatchTheFreedSlotInOfferOrder(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	p := func() string { return w.patient(t, true) }
	morning := &appointment.Times{Start: 480, End: 720}
	early := &appointment.Times{Start: 480, End: 615}
	first := w.add(t, entry{patient: p(), priority: 5})
	second := w.add(t, entry{patient: p(), created: now.Add(-time.Hour)})
	third := w.add(t, entry{patient: p(), specialty: generalPractice, times: morning})
	excluded := map[string]string{
		"another specialty":          w.add(t, entry{patient: p(), specialty: "394579002"}),
		"another role":               w.add(t, entry{patient: p(), role: w.roleOther}),
		"another day":                w.add(t, entry{patient: p(), earliest: "2030-01-08", latest: "2030-01-09"}),
		"times the slot ends after":  w.add(t, entry{patient: p(), times: early}),
		"past its last day":          w.add(t, entry{patient: p(), endsAt: now}),
		"withdrawn":                  w.add(t, entry{patient: p(), status: "withdrawn"}),
		"no consent":                 w.add(t, entry{patient: w.patient(t, false)}),
		"offered this slot before":   w.add(t, entry{patient: p()}),
		"holding another offer":      w.add(t, entry{patient: p()}),
		"the slot's former patient":  w.add(t, entry{patient: p()}),
		"booked at the slot's time":  w.add(t, entry{patient: p()}),
		"rescheduled off the slot":   w.add(t, entry{patient: p()}),
		"of a patient seen nowhere ": w.add(t, entry{patient: w.patient(t, false), priority: 100}),
	}
	pid := func(entryID string) string {
		var id string
		if err := w.owner.QueryRow(t.Context(), `SELECT patient_id FROM waitlist_entry WHERE id = $1`, entryID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	offer := func(entryID, slotID, state string) {
		ownerExec(t, w.owner, `INSERT INTO waitlist_offer (entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period, expires_at, created_at, state, closed_at)
			VALUES ($1, $2, $3, $4, $5, tstzrange($6, $7), $8, $9, $10, CASE WHEN $10 = 'open' THEN NULL ELSE $9::timestamptz END)`,
			entryID, slotID, clinicA, pid(entryID), w.s.roleA1, start, start.Add(30*time.Minute), now.Add(time.Minute), now, state)
	}
	offer(excluded["offered this slot before"], slotFree, "declined")
	offer(excluded["holding another offer"], slotSoon, "open")
	appt := func(entryID, slotID, status string, at time.Time) string {
		var id string
		if err := w.owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status,
				cancelled_at, cancellation_reason)
			VALUES ($1, $2, $3, $4, tstzrange($5::timestamptz, $5::timestamptz + interval '30 minutes'), $6, CASE WHEN $6 = 'cancelled' THEN now() END, CASE WHEN $6 = 'cancelled' THEN 'pat' END)
			RETURNING id`, pid(entryID), w.roleOther, clinicA, slotID, at, status).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	appt(excluded["the slot's former patient"], slotFree, "cancelled", start)
	appt(excluded["booked at the slot's time"], slotSoon, "booked", start.Add(15*time.Minute))
	moved := appt(excluded["rescheduled off the slot"], slotSoon, "booked", start.Add(48*time.Hour))
	ownerExec(t, w.owner, `INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, kind, via, agent, from_slot_id, to_slot_id, from_period, to_period)
		SELECT now(), id, organization_id, patient_id, 'rescheduled', 'system', 'test', $2, slot_id, tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes'), period
		FROM appointment WHERE id = $1`, moved, slotFree, start)

	got := w.candidates(t, w.slotMatch())
	if want := []string{first, second, third}; !slices.Equal(got, want) {
		names := map[string]string{first: "first", second: "second", third: "third"}
		for name, id := range excluded {
			names[id] = name
		}
		var named []string
		for _, id := range got {
			named = append(named, names[id])
		}
		t.Errorf("candidates %v, want first, second, third", named)
	}
	// Without the specialty (a role with none), only the role's entries.
	m := w.slotMatch()
	m.Specialty = ""
	if got := w.candidates(t, m); !slices.Equal(got, []string{first, second}) {
		t.Errorf("candidates without a specialty: %v, want the role's", got)
	}
}

// inClinicA and asPatientA run fn on the waitlist repository in clinic A's scope, or patient A's with it.
func (w waitlistDB) in(t *testing.T, patient string, fn func(ctx context.Context, r appointment.WaitlistRepository) error) error {
	t.Helper()
	return postgres.NewTransactor(w.app).WithTx(t.Context(), func(ctx context.Context) error {
		repo := appointment.NewPostgresRepository(w.app)
		if patient != "" {
			if err := repo.SetPatient(ctx, patient); err != nil {
				return err
			}
		}
		if err := repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return fn(ctx, appointment.NewPostgresWaitlistRepository(w.app))
	})
}

// The patient joins in their scope; the clinic offers a slot to the entry, which then lists with it; the patient sees
// their own newest first, the clinic its own in offer order (with a role, its specialty's too); a closed entry or
// offer does not change again.
func TestWaitlistRepository_EntriesAndOffersRoundTrip(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	a := w.s.patientA
	var joined, bySpecialty appointment.WaitlistEntry
	err := w.in(t, a, func(ctx context.Context, r appointment.WaitlistRepository) error {
		var err error
		in := appointment.WaitlistEntry{
			Created: now, EndsAt: start, Earliest: day(now), Latest: day(start), PatientID: a, OrganizationID: clinicA,
			PractitionerRoleID: w.s.roleA1, Times: appointment.Times{Start: 480, End: 1440},
		}
		if joined, err = r.InsertEntry(ctx, in); err != nil {
			return err
		}
		in.PractitionerRoleID, in.Specialty, in.Times, in.Created = "", generalPractice, appointment.Times{}, now.Add(time.Minute)
		bySpecialty, err = r.InsertEntry(ctx, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if joined.Status != appointment.WaitlistActive || !joined.Latest.Equal(day(start)) || joined.Times != (appointment.Times{Start: 480, End: 1440}) ||
		joined.PractitionerRoleID != w.s.roleA1 || bySpecialty.Specialty != generalPractice || !bySpecialty.Times.Any() {
		t.Fatalf("InsertEntry() = %+v and %+v", joined, bySpecialty)
	}
	var offer appointment.Offer
	err = w.in(t, "", func(ctx context.Context, r appointment.WaitlistRepository) error {
		var err error
		offer, err = r.InsertOffer(ctx, appointment.Offer{
			Created: now, ExpiresAt: now.Add(15 * time.Minute), Start: start, End: start.Add(30 * time.Minute), EntryID: joined.ID,
			SlotID: slotFree, OrganizationID: clinicA, PatientID: a, PractitionerRoleID: w.s.roleA1,
		})
		return err
	})
	if err != nil || offer.State != appointment.OfferOpen || !offer.Start.Equal(start) {
		t.Fatalf("InsertOffer() = %+v, %v", offer, err)
	}
	if err := w.in(t, a, func(ctx context.Context, r appointment.WaitlistRepository) error {
		return patientSeesTheirOwn(ctx, r, a, bySpecialty.ID, joined.ID, offer)
	}); err != nil {
		t.Fatal(err)
	}
	if err := w.in(t, "", func(ctx context.Context, r appointment.WaitlistRepository) error {
		return clinicOrdersAndCloses(ctx, r, w, bySpecialty.ID, joined.ID, offer.ID)
	}); err != nil {
		t.Fatal(err)
	}
}

// patientSeesTheirOwn checks, in the patient's scope, their entries newest first with the open offer, the count of
// active ones, and the open offer of the entry.
func patientSeesTheirOwn(ctx context.Context, r appointment.WaitlistRepository, a, newest, offered string, offer appointment.Offer) error {
	mine, err := r.PatientEntries(ctx, a)
	if err != nil {
		return err
	}
	if len(mine) != 2 || mine[0].ID != newest || mine[1].Offer.ID != offer.ID || !mine[1].Offer.ExpiresAt.Equal(offer.ExpiresAt) {
		return fmt.Errorf("PatientEntries() = %+v", mine)
	}
	if n, err := r.CountActiveEntries(ctx, a, clinicA, now); err != nil || n != 2 {
		return fmt.Errorf("CountActiveEntries() = %d, %v", n, err)
	}
	if n, err := r.CountActiveEntries(ctx, a, clinicA, start); err != nil || n != 0 {
		return fmt.Errorf("CountActiveEntries(after they end) = %d, %v", n, err)
	}
	o, open, err := r.OpenOffer(ctx, "", offered)
	if err != nil || !open || o.ID != offer.ID {
		return fmt.Errorf("OpenOffer(entry) = %+v, %v, %v", o, open, err)
	}
	_, err = r.OfferForUpdate(ctx, offer.ID)
	return err
}

// clinicOrdersAndCloses checks, in the clinic's scope, the offer order after a move, the role's entries with its
// specialty's, the slot's open offer, and that a closed offer or entry does not change again.
func clinicOrdersAndCloses(ctx context.Context, r appointment.WaitlistRepository, w waitlistDB, moved, offered, offer string) error {
	if _, err := r.SetPriority(ctx, moved, 9, now); err != nil {
		return err
	}
	all, err := r.ClinicEntries(ctx, clinicPage("", "", appointment.WaitlistActive))
	if err != nil || len(all) != 2 || all[0].ID != moved || all[0].Priority != 9 {
		return fmt.Errorf("ClinicEntries() = %+v, %v; want the moved one first", all, err)
	}
	if other, err := r.ClinicEntries(ctx, clinicPage(w.roleOther, "394579002", appointment.WaitlistActive)); err != nil || len(other) != 0 {
		return fmt.Errorf("ClinicEntries(another role) = %+v, %v", other, err)
	}
	if role, err := r.ClinicEntries(ctx, clinicPage(w.s.roleA1, generalPractice, appointment.WaitlistActive)); err != nil || len(role) != 2 {
		return fmt.Errorf("ClinicEntries(role A1 with its specialty) = %+v, %v", role, err)
	}
	if o, open, err := r.OpenOffer(ctx, slotFree, ""); err != nil || !open || o.ID != offer {
		return fmt.Errorf("OpenOffer(slot) = %+v, %v, %v", o, open, err)
	}
	if closed, err := r.CloseOffer(ctx, offer, appointment.OfferExpired, "", now); err != nil || closed.State != appointment.OfferExpired {
		return fmt.Errorf("CloseOffer() = %+v, %v", closed, err)
	}
	if _, err := r.CloseOffer(ctx, offer, appointment.OfferDeclined, "", now); err == nil {
		return fmt.Errorf("a closed offer closed again")
	}
	if _, err := r.CloseEntry(ctx, offered, appointment.WaitlistWithdrawn, now); err != nil {
		return err
	}
	if _, err := r.SetPriority(ctx, offered, 3, now); err == nil {
		return fmt.Errorf("a closed entry was moved")
	}
	active, err := r.ActiveEntries(ctx, w.s.patientA, clinicA)
	if err != nil || len(active) != 1 || active[0].ID != moved {
		return fmt.Errorf("ActiveEntries() = %+v, %v", active, err)
	}
	return nil
}

// Retention deletes, on the retention role's pool, the entries closed or ended before the cut-off with their offers.
func TestWaitlistPurger_DeletesEntriesThirtyDaysAfterTheyClose(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	w := waitlistDB{owner: owner, s: seedDB(t, owner)}
	old := w.add(t, entry{patient: w.s.patientA, status: "withdrawn", created: now.Add(-40 * 24 * time.Hour)})
	ended := w.add(t, entry{patient: w.s.patientA, endsAt: now.Add(-31 * 24 * time.Hour)})
	kept := w.add(t, entry{patient: w.s.patientA})
	ownerExec(t, owner, `INSERT INTO waitlist_offer (entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period, expires_at, created_at, state, closed_at)
		VALUES ($1, $2, $3, $4, $5, tstzrange($6, $7), $8, $9, 'expired', $8)`,
		old, slotFree, clinicA, w.s.patientA, w.s.roleA1, start, start.Add(30*time.Minute), now.Add(-39*24*time.Hour), now.Add(-40*24*time.Hour))
	repo := appointment.NewPostgresWaitlistRepository(retention)
	cutoff := now.Add(-30 * 24 * time.Hour)
	if n, err := appointment.NewWaitlistOfferPurger(repo).Purge(t.Context(), cutoff); err != nil || n != 1 {
		t.Errorf("offers purged %d, %v; want 1", n, err)
	}
	if n, err := appointment.NewWaitlistEntryPurger(repo).Purge(t.Context(), cutoff); err != nil || n != 2 {
		t.Errorf("entries purged %d, %v; want 2", n, err)
	}
	var left []string
	rows, err := owner.Query(t.Context(), `SELECT id::text FROM waitlist_entry`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		left = append(left, id)
	}
	if !slices.Equal(left, []string{kept}) || slices.Contains(left, ended) {
		t.Errorf("left %v, want only %s", left, kept)
	}
}

// The matching query with 10 000 entries in the clinic (2 000 patients with the care consent, five entries each, most
// for other days, times or roles), in the clinic's scope as the freed-slot job runs it: measured for
// docs/plans/appointments.PROGRESS.md, and bounded far above what it takes so a regression to a scan per entry fails.
func TestCandidates_TenThousandEntriesInAClinic(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	ownerExec(t, w.owner, `
		INSERT INTO account (email, password_hash, role) SELECT 'wl' || i || '@example.test', 'h', 'patient' FROM generate_series(1, 2000) i;
		INSERT INTO patient (account_id, given_name, family_name, birth_date)
			SELECT id, 'Ada', 'Example', '1990-01-01' FROM account WHERE email LIKE 'wl%';
		INSERT INTO consent (patient_id, organization_id, status, date_time)
			SELECT p.id, '`+clinicA+`', 'active', now() FROM patient p JOIN account a ON a.id = p.account_id WHERE a.email LIKE 'wl%';
		INSERT INTO waitlist_entry (created_at, ends_at, patient_id, organization_id, practitioner_role_id, specialty, earliest, latest,
		                            times_start_minute, times_end_minute, priority)
			SELECT now() - (n * 5 + k) * interval '1 minute', '2030-03-01', p.id, '`+clinicA+`',
			       CASE WHEN k % 2 = 0 THEN '`+w.s.roleA1+`'::uuid ELSE NULL END, CASE WHEN k % 2 = 0 THEN NULL ELSE '394579002' END,
			       '2030-01-01'::date + ((n + k) % 50)::integer, '2030-01-01'::date + ((n + k) % 50)::integer + 3,
			       CASE WHEN k = 4 THEN 840 END, CASE WHEN k = 4 THEN 960 END, (n * 7 + k) % 4
			FROM (SELECT id, row_number() OVER (ORDER BY id) AS n FROM patient) p, generate_series(0, 4) k;
		ANALYZE;`)
	var n int
	if err := w.owner.QueryRow(t.Context(), `SELECT count(*) FROM waitlist_entry WHERE organization_id = $1`, clinicA).Scan(&n); err != nil || n < 10000 {
		t.Fatalf("%d entries (%v), want 10 000", n, err)
	}
	m := w.slotMatch()
	var took []time.Duration
	var got []appointment.WaitlistEntry
	for range 5 {
		err := postgres.NewTransactor(w.app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := appointment.NewPostgresRepository(w.app).SetClinic(ctx, clinicA); err != nil {
				return err
			}
			began := clock.System{}.Now()
			var err error
			got, err = appointment.NewPostgresWaitlistRepository(w.app).Candidates(ctx, m, 10)
			took = append(took, clock.System{}.Now().Sub(began))
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	slices.Sort(took)
	t.Logf("10 candidates of %d entries: %v (median %v)", n, took, took[len(took)/2])
	if len(got) != 10 {
		t.Errorf("%d candidates, want 10", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Priority < got[i].Priority {
			t.Errorf("candidates out of offer order: %d before %d", got[i-1].Priority, got[i].Priority)
		}
	}
	if took[len(took)/2] > 500*time.Millisecond {
		t.Errorf("the matching query takes %v, want it well under a second", took[len(took)/2])
	}
}

// clinicPage is clinic A's first page of 50 of the role's (or every) entry in the status, at now.
func clinicPage(role, specialty string, status appointment.WaitlistStatus) appointment.ClinicQuery {
	return appointment.ClinicQuery{Now: now, OrganizationID: clinicA, PractitionerRoleID: role, Specialty: specialty, Status: status, Limit: 50}
}

// The clinic's page filters the status in SQL: active ones not past their last day, expired ones past it, booked and
// withdrawn by their status; a page and an offset (review of PR #93, finding 5).
func TestClinicEntries_FiltersTheStatusAndPages(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	p := w.s.patientA
	active := []string{w.add(t, entry{patient: p, priority: 2}), w.add(t, entry{patient: p, priority: 1}), w.add(t, entry{patient: p})}
	ended := w.add(t, entry{patient: p, endsAt: now})
	booked := w.add(t, entry{patient: p, status: "booked"})
	withdrawn := w.add(t, entry{patient: p, status: "withdrawn"})
	for _, tt := range []struct {
		status appointment.WaitlistStatus
		want   []string
		limit  int
		offset int
	}{
		{status: appointment.WaitlistActive, limit: 50, want: active},
		{status: appointment.WaitlistActive, limit: 1, offset: 1, want: active[1:2]},
		{status: appointment.WaitlistExpired, limit: 50, want: []string{ended}},
		{status: appointment.WaitlistBooked, limit: 50, want: []string{booked}},
		{status: appointment.WaitlistWithdrawn, limit: 50, want: []string{withdrawn}},
	} {
		var got []string
		err := w.in(t, "", func(ctx context.Context, r appointment.WaitlistRepository) error {
			q := clinicPage("", "", tt.status)
			q.Limit, q.Offset = tt.limit, tt.offset
			entries, err := r.ClinicEntries(ctx, q)
			for _, e := range entries {
				got = append(got, e.ID)
			}
			return err
		})
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("%s (%d from %d): %v, %v; want %v", tt.status, tt.limit, tt.offset, got, err, tt.want)
		}
	}
}

// The staff's list never shows an offer of a patient whose care consent is revoked: the clinic's jobs see every offer
// of the clinic (decision c), but the list reaches an offer only through its entry, which the clinic no longer sees
// (review of PR #93, finding 6).
func TestClinicEntries_NeverShowAnOfferOfARevokedPatient(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	revoked := w.patient(t, true)
	id := w.add(t, entry{patient: revoked})
	ownerExec(t, w.owner, `INSERT INTO waitlist_offer (entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, tstzrange($6, $7), $8, $9)`, id, slotFree, clinicA, revoked, w.s.roleA1, start, start.Add(30*time.Minute), now.Add(15*time.Minute), now)
	ownerExec(t, w.owner, `UPDATE consent SET status = 'inactive', revoked_at = $2 WHERE patient_id = $1`, revoked, now)
	err := w.in(t, "", func(ctx context.Context, r appointment.WaitlistRepository) error {
		for _, status := range []appointment.WaitlistStatus{appointment.WaitlistActive, appointment.WaitlistExpired, appointment.WaitlistBooked, appointment.WaitlistWithdrawn} {
			entries, err := r.ClinicEntries(ctx, clinicPage("", "", status))
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.ID == id || e.Offer.PatientID == revoked || e.PatientID == revoked {
					return fmt.Errorf("the list shows the revoked patient's entry or offer: %+v", e)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// An entry another transaction holds (a withdrawal, another slot's job) is passed over by the matching query, which
// locks the ones it gives (review of PR #93, finding 4).
func TestCandidates_PassOverAnEntryAnotherTransactionHolds(t *testing.T) {
	t.Parallel()
	w := newWaitlistDB(t)
	held := w.add(t, entry{patient: w.s.patientA, priority: 9})
	free := w.add(t, entry{patient: w.patient(t, true)})
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- w.in(t, w.s.patientA, func(ctx context.Context, r appointment.WaitlistRepository) error {
			_, err := r.EntryForUpdate(ctx, held)
			close(locked)
			<-release
			return err
		})
	}()
	<-locked
	got := w.candidates(t, w.slotMatch())
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{free}) {
		t.Errorf("candidates %v, want only the entry nobody holds %s", got, free)
	}
	if got := w.candidates(t, w.slotMatch()); !slices.Equal(got, []string{held, free}) {
		t.Errorf("candidates once released %v, want both in order", got)
	}
}
