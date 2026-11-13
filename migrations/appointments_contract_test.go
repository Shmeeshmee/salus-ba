package migrations_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

var errNoRow = errors.New("no row changed")

// insertChange writes, as the owner, a confirmation of the appointment id and returns its id.
func insertChange(t *testing.T, owner *pgxpool.Pool, id string) string {
	t.Helper()
	var change string
	err := owner.QueryRow(t.Context(), `INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, kind, via, agent)
		SELECT now(), id, organization_id, patient_id, 'confirmed', 'system', 'test' FROM appointment WHERE id = $1 RETURNING id`, id).Scan(&change)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

// An appointment's history is seen exactly where the appointment is: the patient's own, the clinic's with the consent
// (after a revocation, the past ones), a job's one appointment; nothing without a scope.
func TestAppointmentChange_IsSeenWhereItsAppointmentIs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	c := map[string]string{}
	for _, id := range []string{a.aPast, a.aFuture, a.bPast, a.bFuture, a.other} {
		c[id] = insertChange(t, owner, id)
	}
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(c[a.aPast], c[a.aFuture], c[a.bPast], c[a.bFuture])},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(c[a.other])},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(c[a.aPast], c[a.aFuture])},
		{name: "clinic B after the revocation", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(c[a.bPast])},
		{name: "a job's one appointment", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture}, want: sorted(c[a.bPast], c[a.bFuture])},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM appointment_change"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// The app writes a change only for an appointment it sees, with the appointment's clinic and patient, and never
// changes or removes one; the owner cannot either.
func TestAppointmentChange_IsAppendOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	insert := func(scope map[string]string, id, org, patient string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, kind, via, by_account_id)
				VALUES (now(), $1, $2, $3, 'confirmed', 'app', $4)`, id, org, patient, p[0].account)
			return err
		})
	}
	own := map[string]string{"app.patient_id": p[0].patient}
	if err := insert(own, a.aFuture, clinicA, p[0].patient); err != nil {
		t.Errorf("patient A confirms their appointment: %v", err)
	}
	if err := insert(own, a.other, clinicA, p[1].patient); err == nil {
		t.Error("patient A wrote a change of patient B's appointment")
	}
	if err := insert(map[string]string{"app.organization_id": clinicA}, a.aFuture, clinicB, p[0].patient); err == nil {
		t.Error("a change with another clinic than its appointment's was stored")
	}
	// Who wrote it is what the scope is (review, finding 10).
	write := func(scope map[string]string, via, account, agent string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, kind, via, by_account_id, agent)
				VALUES (now(), $1, $2, $3, 'confirmed', $4, NULLIF($5, '')::uuid, NULLIF($6, ''))`, a.aFuture, clinicA, p[0].patient, via, account, agent)
			return err
		})
	}
	clinic := map[string]string{"app.organization_id": clinicA}
	for _, tt := range []struct {
		scope                map[string]string
		name, via, by, agent string
		ok                   bool
	}{
		{name: "the patient's app", scope: own, via: "app", by: p[0].account, ok: true},
		{name: "the patient's link", scope: own, via: "link", by: p[0].account, ok: true},
		{name: "a job in the patient's scope", scope: own, via: "system", agent: "consent_revoked", ok: true},
		{name: "the patient's scope claiming staff", scope: own, via: "staff", by: p[0].account},
		{name: "the patient's scope with another account", scope: own, via: "app", by: p[1].account},
		{name: "the clinic's staff", scope: clinic, via: "staff", by: p[1].account, ok: true},
		{name: "a job in the clinic's scope", scope: clinic, via: "system", agent: "consultation_window_close", ok: true},
		{name: "the clinic's scope claiming the patient's app", scope: clinic, via: "app", by: p[0].account},
		{name: "the clinic's scope claiming the link", scope: clinic, via: "link", by: p[0].account},
	} {
		if err := write(tt.scope, tt.via, tt.by, tt.agent); (err == nil) != tt.ok {
			t.Errorf("%s: %v, want stored %t", tt.name, err, tt.ok)
		}
	}
	change := insertChange(t, owner, a.aFuture)
	for _, q := range []string{"UPDATE appointment_change SET via = 'app' WHERE id = $1", "DELETE FROM appointment_change WHERE id = $1"} {
		err := asApp(t, app, own, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, q, change)
			return err
		})
		if err == nil {
			t.Errorf("the app ran %q", q)
		}
		if _, err := owner.Exec(t.Context(), q, change); err == nil {
			t.Errorf("the owner ran %q", q)
		}
	}
	if _, err := owner.Exec(t.Context(), "TRUNCATE appointment_change"); err == nil {
		t.Error("the owner truncated appointment_change")
	}
	// A reschedule names both periods and slots; a status change both statuses.
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, kind, via, agent)
		SELECT now(), id, organization_id, patient_id, 'rescheduled', 'system', 'test' FROM appointment WHERE id = $1`, a.aFuture); err == nil {
		t.Error("a reschedule without its periods and slots was stored")
	}
}

// A reschedule moves the appointment's period and slot in the patient's scope; the exclusion constraint still refuses
// an overlap with another booking of the role.
func TestAppointment_ARescheduleMovesThePeriodAndTheSlot(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	move := func(id, start string) error {
		return asApp(t, app, map[string]string{"app.patient_id": p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `UPDATE appointment SET period = tstzrange($2::timestamptz, $2::timestamptz + interval '30 minutes'),
				slot_id = gen_random_uuid(), reschedule_count = reschedule_count + 1, confirmed_at = now(), confirmed_via = 'app',
				version_id = version_id + 1 WHERE id = $1`, id, start)
			return err
		})
	}
	if err := move(a.aFuture, "2030-01-07T11:00:00Z"); err != nil {
		t.Fatalf("patient A reschedules: %v", err)
	}
	if err := move(a.aFuture, "2030-01-08T09:15:00Z"); err == nil { // patient B's appointment with the same role
		t.Error("a reschedule onto another booking of the role was stored")
	}
}

// The trigger says who may change the period, the slot, the count and the confirmation (review, finding 1), under
// every UPDATE policy: only the patient moves their booked appointment, counting once; the clinic never moves one nor
// confirms; a confirmation is never cleared.
func TestAppointment_OnlyThePatientMovesOrConfirmsTheirBookedAppointment(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	own := map[string]string{"app.patient_id": p[0].patient}
	run := func(scope map[string]string, q, id string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, q, id)
			if err == nil && tag.RowsAffected() != 1 {
				return errNoRow
			}
			return err
		})
	}
	const (
		moveOnce   = "UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'), slot_id = gen_random_uuid(), reschedule_count = reschedule_count + 1 WHERE id = $1"
		moveFree   = "UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'), slot_id = gen_random_uuid() WHERE id = $1"
		moveTwice  = "UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'), slot_id = gen_random_uuid(), reschedule_count = reschedule_count + 2 WHERE id = $1"
		slotOnly   = "UPDATE appointment SET slot_id = gen_random_uuid() WHERE id = $1"
		resetCount = "UPDATE appointment SET reschedule_count = 0 WHERE id = $1"
		confirm    = "UPDATE appointment SET confirmed_at = now(), confirmed_via = 'app' WHERE id = $1"
		clear      = "UPDATE appointment SET confirmed_at = NULL, confirmed_via = NULL WHERE id = $1"
	)
	for _, tt := range []struct {
		scope   map[string]string
		name, q string
		id      string
		ok      bool
	}{
		{name: "clinic A moves its consenting patient's", scope: map[string]string{"app.organization_id": clinicA}, q: moveOnce, id: a.aFuture},
		{name: "clinic B moves a past one after the revocation", scope: map[string]string{"app.organization_id": clinicB}, q: moveOnce, id: a.bPast},
		{name: "the patient moves without counting", scope: own, q: moveFree, id: a.aFuture},
		{name: "the patient counts twice", scope: own, q: moveTwice, id: a.aFuture},
		{name: "the patient changes the slot alone", scope: own, q: slotOnly, id: a.aFuture},
		{name: "the patient moves a cancelled one", scope: own, q: moveOnce, id: a.bFuture},
		{name: "clinic A confirms", scope: map[string]string{"app.organization_id": clinicA}, q: confirm, id: a.aFuture},
		{name: "the patient moves their booked one", scope: own, q: moveOnce, id: a.aFuture, ok: true},
		{name: "the patient resets the count", scope: own, q: resetCount, id: a.aFuture},
		{name: "the patient confirms", scope: own, q: confirm, id: a.aFuture, ok: true},
		{name: "the patient clears the confirmation", scope: own, q: clear, id: a.aFuture},
	} {
		if err := run(tt.scope, tt.q, tt.id); (err == nil) != tt.ok {
			t.Errorf("%s: %v, want done %t", tt.name, err, tt.ok)
		}
	}
	// The schema owner is not guarded (migrations, the operator, fixtures), as row-level security does not hold it.
	if _, err := owner.Exec(t.Context(), moveFree, a.aFuture); err != nil {
		t.Errorf("the owner moves an appointment: %v", err)
	}
}

// A token is issued by a job for its one appointment, read and used by its patient only, and found by its hash
// from any scope through the function, which returns only its ids.
func TestAppointmentActionToken_IssuedByTheJobUsedByThePatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	hash := sha256.Sum256([]byte("synthetic token"))
	issue := func(scope map[string]string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO appointment_action_token (created_at, expires_at, start_at, appointment_id, organization_id, patient_id,
				account_id, token_hash, channel, language) VALUES (now(), '2030-01-07T09:00:00Z', '2030-01-07T09:00:00Z', $1, $2, $3, $4, $5, 'email', 'bs')`,
				a.aFuture, clinicA, p[0].patient, p[0].account, hash[:])
			return err
		})
	}
	if err := issue(map[string]string{"app.organization_id": clinicA}); err == nil {
		t.Error("a clinic without the job's appointment issued a token")
	}
	if err := issue(map[string]string{"app.patient_id": p[0].patient}); err == nil {
		t.Error("a patient issued a token")
	}
	notice := map[string]string{"app.organization_id": clinicA, "app.appointment_id": a.aFuture}
	if err := issue(notice); err != nil {
		t.Fatalf("the job issues the token: %v", err)
	}
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  int
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: 1},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}},
		{name: "the job", scope: notice, want: 1},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM appointment_action_token"); len(got) != tt.want {
			t.Errorf("%s: sees %d tokens, want %d", tt.name, len(got), tt.want)
		}
	}
	var patient string
	err := asApp(t, app, nil, func(ctx context.Context, db postgres.Querier) error {
		return db.QueryRow(ctx, "SELECT patient_id::text FROM appointment_action_token_find($1)", hash[:]).Scan(&patient)
	})
	if err != nil || patient != p[0].patient {
		t.Errorf("found by hash: %q, %v; want patient A", patient, err)
	}
	use := func(scope map[string]string) (int64, error) {
		var n int64
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, "UPDATE appointment_action_token SET used_at = now(), used_action = 'confirm' WHERE token_hash = $1", hash[:])
			n = tag.RowsAffected()
			return err
		})
		return n, err
	}
	if n, err := use(map[string]string{"app.patient_id": p[1].patient}); err != nil || n != 0 {
		t.Errorf("patient B used patient A's token: %d rows, %v", n, err)
	}
	if n, err := use(map[string]string{"app.patient_id": p[0].patient}); err != nil || n != 1 {
		t.Errorf("patient A uses their token: %d rows, %v; want 1", n, err)
	}
	// Single use (review, finding 5): a used link never changes again, for the patient or the owner.
	if n, err := use(map[string]string{"app.patient_id": p[0].patient}); err == nil {
		t.Errorf("patient A used their token twice: %d rows", n)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE appointment_action_token SET used_at = NULL, used_action = NULL WHERE token_hash = $1", hash[:]); err == nil {
		t.Error("the owner reset a used token")
	}
	// The link expires at the start it was issued for (review, finding 2).
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment_action_token (created_at, expires_at, start_at, appointment_id, organization_id, patient_id,
		account_id, token_hash, channel, language) VALUES (now(), '2030-01-08T09:00:00Z', '2030-01-07T09:00:00Z', $1, $2, $3, $4, $5, 'email', 'bs')`,
		a.aFuture, clinicA, p[0].patient, p[0].account, make([]byte, 32)); err == nil {
		t.Error("a link expiring after the start it was issued for was stored")
	}
}

// A clinic sees and writes only its own appointment settings, within their ranges.
func TestOrganizationAppointmentSettings_TheClinicsOwnWithinTheirRanges(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	put := func(org, target, offsets string, hold int) error {
		return inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO organization_appointment_settings (organization_id, reminder_offsets_minutes, waitlist_hold_minutes)
				VALUES ($1, $2::integer[], $3)`, target, offsets, hold)
			return err
		})
	}
	if err := put(clinicA, clinicA, "{1440,60}", 15); err != nil {
		t.Fatalf("clinic A stores its settings: %v", err)
	}
	if err := put(clinicA, clinicB, "{1440,60}", 15); err == nil {
		t.Error("clinic A stored clinic B's settings")
	}
	for _, bad := range []struct {
		offsets string
		hold    int
	}{{"{1440,60,30,15}", 15}, {"{10}", 15}, {"{20000}", 15}, {"{60,NULL}", 15}, {"{60}", 4}} {
		if err := put(clinicB, clinicB, bad.offsets, bad.hold); err == nil {
			t.Errorf("settings %v stored", bad)
		}
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, "SELECT organization_id::text FROM organization_appointment_settings"); len(got) != 0 {
		t.Errorf("clinic B sees %v", got)
	}
}
