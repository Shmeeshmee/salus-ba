package migrations_test

import (
	"context"
	"crypto/sha256"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

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

// A token is issued by a job for its one appointment, read and used by its patient only, and found by its hash
// from any scope through the function, which returns only its ids.
func TestAppointmentActionToken_IssuedByTheJobUsedByThePatient(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	hash := sha256.Sum256([]byte("synthetic token"))
	issue := func(scope map[string]string) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO appointment_action_token (created_at, expires_at, appointment_id, organization_id, patient_id,
				account_id, token_hash, channel, language) VALUES (now(), '2030-01-07T09:00:00Z', $1, $2, $3, $4, $5, 'email', 'bs')`,
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
