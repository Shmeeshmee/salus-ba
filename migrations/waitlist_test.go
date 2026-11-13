package migrations_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// waitlistRows is the seeded waitlist: patient A's entries with clinic A (active consent) and clinic B (revoked), and
// patient B's with clinic A (no consent); an open offer of A's entry with clinic A for slot.
type waitlistRows struct {
	aA, aB, bA, offer, slot, schedule string
}

// seedWaitlist creates, as the owner, seedAppointments' rows, a schedule of clinic A's role with a free slot, the
// entries and the offer.
func seedWaitlist(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, map[string]directory, waitlistRows) {
	t.Helper()
	ctx := t.Context()
	p, dir, _ := seedAppointments(t, owner)
	var w waitlistRows
	if err := owner.QueryRow(ctx, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, 30) RETURNING id`,
		clinicA, dir[clinicA].role).Scan(&w.schedule); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO slot (schedule_id, organization_id, period, status)
		VALUES ($1, $2, tstzrange('2030-02-01T09:00:00Z', '2030-02-01T09:30:00Z'), 'busy-tentative') RETURNING id`,
		w.schedule, clinicA).Scan(&w.slot); err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct {
		id               *string
		patient, clinic  string
		role, specialty  any
		earliest, latest string
	}{
		{&w.aA, p[0].patient, clinicA, dir[clinicA].role, nil, "2030-01-01", "2030-03-01"},
		{&w.aB, p[0].patient, clinicB, nil, "394814009", "2030-01-01", "2030-03-01"},
		{&w.bA, p[1].patient, clinicA, dir[clinicA].role, nil, "2030-01-01", "2030-03-01"},
	} {
		if err := owner.QueryRow(ctx, `INSERT INTO waitlist_entry (patient_id, organization_id, practitioner_role_id, specialty, earliest, latest, ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6::date + 1) RETURNING id`, e.patient, e.clinic, e.role, e.specialty, e.earliest, e.latest).Scan(e.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := owner.QueryRow(ctx, `INSERT INTO waitlist_offer (entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period, expires_at)
		VALUES ($1, $2, $3, $4, $5, tstzrange('2030-02-01T09:00:00Z', '2030-02-01T09:30:00Z'), now() + interval '15 minutes') RETURNING id`,
		w.aA, w.slot, clinicA, p[0].patient, dir[clinicA].role).Scan(&w.offer); err != nil {
		t.Fatal(err)
	}
	return p, dir, w
}

// A waitlist entry is seen by its patient, and by its clinic while the patient's care consent there is active.
func TestWaitlistEntry_IsSeenByItsPatientAndByTheClinicWithTheConsent(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, w := seedWaitlist(t, owner)
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(w.aA, w.aB)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(w.bA)},
		{name: "patient A with clinic A set", scope: map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicA}, want: sorted(w.aA, w.aB)},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(w.aA)},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB}},
		{name: "an account in clinic A", scope: map[string]string{"app.organization_id": clinicA, "app.account_id": p[0].account}},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM waitlist_entry"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// The patient adds only their own entries, of one role or one specialty, with ordered dates and times; the clinic adds
// none, and changes only the status and priority of the entries it sees.
func TestWaitlistEntry_ThePatientJoinsAndTheClinicOrders(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, w := seedWaitlist(t, owner)
	join := func(scope map[string]string, patient string, role, specialty any, earliest, latest string, times ...int) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			var start, end any
			if len(times) == 2 {
				start, end = times[0], times[1]
			}
			_, err := db.Exec(ctx, `INSERT INTO waitlist_entry (patient_id, organization_id, practitioner_role_id, specialty, earliest, latest, ends_at,
				times_start_minute, times_end_minute) VALUES ($1, $2, $3, $4, $5, $6, $6::date + 1, $7, $8)`,
				patient, clinicA, role, specialty, earliest, latest, start, end)
			return err
		})
	}
	own := map[string]string{"app.patient_id": p[0].patient}
	role := dir[clinicA].role
	if err := join(own, p[0].patient, role, nil, "2030-01-01", "2030-01-02", 480, 1440); err != nil {
		t.Errorf("patient A joins for a role: %v", err)
	}
	if err := join(own, p[0].patient, nil, "394814009", "2030-01-01", "2030-01-01"); err != nil {
		t.Errorf("patient A joins for a specialty: %v", err)
	}
	for name, err := range map[string]error{
		"for patient B":               join(own, p[1].patient, role, nil, "2030-01-01", "2030-01-02"),
		"in the clinic's scope":       join(map[string]string{"app.organization_id": clinicA}, p[0].patient, role, nil, "2030-01-01", "2030-01-02"),
		"with a role and a specialty": join(own, p[0].patient, role, "394814009", "2030-01-01", "2030-01-02"),
		"with neither":                join(own, p[0].patient, nil, nil, "2030-01-01", "2030-01-02"),
		"with dates out of order":     join(own, p[0].patient, role, nil, "2030-01-02", "2030-01-01"),
		"with times out of order":     join(own, p[0].patient, role, nil, "2030-01-01", "2030-01-02", 600, 540),
		"with one time":               join(own, p[0].patient, role, nil, "2030-01-01", "2030-01-02", 600, -1),
	} {
		if err == nil {
			t.Errorf("an entry %s was stored", name)
		}
	}
	clinic := map[string]string{"app.organization_id": clinicA}
	set := func(scope map[string]string, query, id string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, query, id)
			if err == nil && tag.RowsAffected() != 1 {
				return errNoRow
			}
			return err
		})
	}
	if err := set(clinic, `UPDATE waitlist_entry SET priority = 10, updated_at = now() WHERE id = $1`, w.aA); err != nil {
		t.Errorf("clinic A moves patient A's entry up: %v", err)
	}
	if err := set(clinic, `UPDATE waitlist_entry SET priority = 10 WHERE id = $1`, w.bA); err == nil {
		t.Error("clinic A moved the entry of a patient without its consent")
	}
	if err := set(clinic, `UPDATE waitlist_entry SET earliest = '2030-01-05' WHERE id = $1`, w.aA); err == nil {
		t.Error("clinic A changed an entry's dates")
	}
	if err := set(clinic, `UPDATE waitlist_entry SET priority = 101 WHERE id = $1`, w.aA); err == nil {
		t.Error("a priority above 100 was stored")
	}
	if err := set(clinic, `UPDATE waitlist_entry SET status = 'withdrawn', closed_at = now() WHERE id = $1`, w.aA); err != nil {
		t.Errorf("clinic A removes patient A's entry: %v", err)
	}
	if err := set(own, `UPDATE waitlist_entry SET status = 'booked' WHERE id = $1`, w.aB); err == nil {
		t.Error("an entry was closed without when")
	}
}

// An offer is seen by its patient and by its clinic; the clinic offers only an entry it sees, one open offer per slot
// and per entry, and an accepted offer names its appointment.
func TestWaitlistOffer_OneOpenOfferPerSlotAndPerEntry(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, w := seedWaitlist(t, owner)
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(w.offer)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(w.offer)},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM waitlist_offer"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
	clinic := map[string]string{"app.organization_id": clinicA}
	offer := func(scope map[string]string, entry, slotID, patient string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO waitlist_offer (entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period, expires_at)
				VALUES ($1, $2, $3, $4, $5, tstzrange('2030-02-02T09:00:00Z', '2030-02-02T09:30:00Z'), now() + interval '15 minutes')`,
				entry, slotID, clinicA, patient, dir[clinicA].role)
			return err
		})
	}
	other := "d0000000-0000-4000-8000-00000000000d"
	for name, err := range map[string]error{
		"a second open offer of the slot":       offer(clinic, w.bA, w.slot, p[1].patient),
		"a second open offer of the entry":      offer(clinic, w.aA, other, p[0].patient),
		"of an entry without the consent":       offer(clinic, w.bA, other, p[1].patient),
		"by the patient":                        offer(map[string]string{"app.patient_id": p[0].patient}, w.aA, other, p[0].patient),
		"of an entry with another patient's id": offer(clinic, w.aA, other, p[1].patient),
	} {
		if err == nil {
			t.Errorf("%s was stored", name)
		}
	}
	accepted := func(scope map[string]string, query string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, query, w.offer)
			if err == nil && tag.RowsAffected() != 1 {
				return errNoRow
			}
			return err
		})
	}
	if err := accepted(clinic, `UPDATE waitlist_offer SET state = 'accepted', closed_at = now() WHERE id = $1`); err == nil {
		t.Error("an offer was accepted without its appointment")
	}
	if err := accepted(clinic, `UPDATE waitlist_offer SET state = 'expired', closed_at = now() WHERE id = $1`); err != nil {
		t.Errorf("clinic A expires the offer: %v", err)
	}
	if err := offer(clinic, w.aA, w.slot, p[0].patient); err != nil {
		t.Errorf("a new offer of the slot once the first closed: %v", err)
	}
}

// A slot may be held for an offer (busy-tentative); the public directory does not show it, and generation and
// retention, which delete free slots only, leave it.
func TestSlot_AnOfferedSlotIsHiddenFromTheDirectory(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, w := seedWaitlist(t, owner)
	ownerExec(t, owner, `UPDATE organization SET listed = true`)
	var n int
	err := inDirectory(t, app, func(ctx context.Context, db postgres.Querier) error {
		return db.QueryRow(ctx, `SELECT count(*) FROM slot WHERE id = $1`, w.slot).Scan(&n)
	})
	if err != nil || n != 0 {
		t.Errorf("the directory sees %d offered slots, %v; want none", n, err)
	}
	err = inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, `DELETE FROM slot WHERE id = $1`, w.slot)
		if err == nil && tag.RowsAffected() != 0 {
			return errNoRow
		}
		return err
	})
	if err != nil {
		t.Errorf("the app deleted an offered slot: %v", err)
	}
	if err := tryAsOwner(t, owner, `UPDATE slot SET status = 'held' WHERE id = '`+w.slot+`'`); err == nil {
		t.Error("a slot status outside FHIR's was stored")
	}
}

// Retention deletes an entry with its offers.
func TestWaitlist_TheRetentionRoleDeletesAnEntryWithItsOffers(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	_, _, w := seedWaitlist(t, owner)
	if _, err := retention.Exec(t.Context(), `DELETE FROM waitlist_entry WHERE id = $1`, w.aA); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM waitlist_offer WHERE id = $1`, w.offer).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d offers left with the deleted entry, %v", n, err)
	}
}

// The free-slot search (slot's SearchFreeSlots) stays fast with a third of the searched schedule's slots held for
// waitlist offers (busy-tentative), in the directory's scopes: under a fifth of jit_optimize_above_cost, and the
// directory sees none of the held slots (SAL-35; main's 82 s to 4 ms fix, docs/plans/patient-history.PROGRESS.md).
func TestWaitlist_TheFreeSlotSearchStaysFastWithOfferedSlots(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	schedule := seedSlotSearch(t, owner)
	ownerExec(t, owner, `UPDATE slot SET status = 'busy-tentative'
		WHERE schedule_id = $1 AND (extract(epoch FROM lower(period))::bigint / 600) % 3 = 0`, schedule)
	ownerExec(t, owner, `ANALYZE slot`)
	const search = `SELECT id, updated_at, schedule_id, organization_id, version_id, status, lower(period)::timestamptz AS start_at,
		upper(period)::timestamptz AS end_at
		FROM slot
		WHERE schedule_id = $1 AND status = 'free'
		  AND period && tstzrange(now(), now() + interval '14 days')
		  AND lower(period) >= now() AND lower(period) < now() + interval '14 days'
		ORDER BY lower(period)`
	var held int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM slot WHERE schedule_id = $1 AND status = 'busy-tentative'`, schedule).Scan(&held); err != nil || held < 400 {
		t.Fatalf("%d held slots (%v), want a third of the schedule's", held, err)
	}
	var patient string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM patient ORDER BY id LIMIT 1`).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	for name, scope := range map[string]map[string]string{
		"directory":           {"app.directory": "public"},
		"directory + patient": {"app.directory": "public", "app.patient_id": patient},
	} {
		var cost, limit float64
		var rows int
		var took time.Duration
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			var plan []struct {
				Plan struct {
					TotalCost float64 `json:"Total Cost"`
				}
			}
			if err := db.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+search, schedule).Scan(&plan); err != nil {
				return err
			}
			cost = plan[0].Plan.TotalCost
			began := clock.System{}.Now()
			r, err := db.Query(ctx, search, schedule)
			if err != nil {
				return err
			}
			for r.Next() {
				var id, sched, org, status string
				var updated, start, end time.Time
				var version int32
				if err := r.Scan(&id, &updated, &sched, &org, &version, &status, &start, &end); err != nil {
					return err
				}
				if status != "free" {
					return errNoRow
				}
				rows++
			}
			r.Close()
			took = clock.System{}.Now().Sub(began)
			if err := r.Err(); err != nil {
				return err
			}
			return db.QueryRow(ctx, "SELECT current_setting('jit_optimize_above_cost')::float8").Scan(&limit)
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: cost %.0f, %d rows in %v", name, cost, rows, took)
		if cost >= limit/5 || rows == 0 {
			t.Errorf("%s: the free-slot search costs %.0f (%d rows), want under a fifth of jit_optimize_above_cost %.0f", name, cost, rows, limit)
		}
	}
}
