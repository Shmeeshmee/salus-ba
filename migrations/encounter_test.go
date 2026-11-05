package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedEncounters creates, as the owner, an in-progress encounter for each seeded appointment and returns their
// ids by appointment.
func seedEncounters(t *testing.T, owner *pgxpool.Pool, a appointmentRows) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for _, appt := range []string{a.aPast, a.aFuture, a.bPast, a.bFuture, a.other} {
		var id string
		err := owner.QueryRow(t.Context(), `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id,
				status, period_start)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM appointment WHERE id = $1
			RETURNING id`, appt).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids[appt] = id
	}
	return ids
}

// insertEncounter is an insert of the appointment's encounter, as the lifecycle makes it, with the
// appointment's values read by the owner (a scope that hides the appointment must still be refused).
func insertEncounter(t *testing.T, owner *pgxpool.Pool, appt string) func(ctx context.Context, db postgres.Querier) error {
	t.Helper()
	var org, patient, role string
	if err := owner.QueryRow(t.Context(), `SELECT organization_id::text, patient_id::text, practitioner_role_id::text FROM appointment WHERE id = $1`,
		appt).Scan(&org, &patient, &role); err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			VALUES ($1, $2, $3, $4, 'in-progress', now())
			ON CONFLICT (appointment_id) DO NOTHING`, appt, org, patient, role)
		return err
	}
}

// Row-level security as video_session's (docs/consultation/SPEC.md §2 "Tables"): the patient their own; a
// clinic those of its appointments with an active consent, and after a revocation those of the appointments
// that started before it. A room alone (app.video_room) shows nothing without a session in it
// (TestEncounter_TheRoomScopeSeesAndFinishesOnlyItsRoomsEncounter).
func TestEncounter_EachScopeSeesOnlyItsEncounters(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	e := seedEncounters(t, owner, a)

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(e[a.aPast], e[a.aFuture], e[a.bPast], e[a.bFuture])},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(e[a.other])},
		{name: "patient B with clinic A set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicA}, want: sorted(e[a.other])},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(e[a.aPast], e[a.aFuture])},
		{name: "clinic B after the revocation: the past appointment's only", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(e[a.bPast])},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
		{name: "a video room alone", scope: map[string]string{"app.video_room": "r_" + a.other}},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM encounter"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// Only a clinic scope writes an encounter (the lifecycle: the webhook and the consultation's jobs set it), for
// an appointment it sees; the patient never writes one.
func TestEncounter_OnlyAClinicScopeWritesTheEncountersItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	seedVideoSessions(t, owner, a)
	tests := []struct {
		scope map[string]string
		name  string
		appt  string
		want  string // the SQLSTATE, "" for success
	}{
		{name: "patient A, their appointment", scope: map[string]string{"app.patient_id": p[0].patient}, appt: a.aFuture, want: "42501"},
		{name: "clinic A, an appointment with an active consent", scope: map[string]string{"app.organization_id": clinicA}, appt: a.aFuture},
		{name: "clinic A, again: nothing", scope: map[string]string{"app.organization_id": clinicA}, appt: a.aFuture},
		{name: "clinic A, without a consent", scope: map[string]string{"app.organization_id": clinicA}, appt: a.other, want: "42501"},
		{name: "clinic B, a past appointment after the revocation", scope: map[string]string{"app.organization_id": clinicB}, appt: a.bPast},
		{name: "clinic B, a later appointment after the revocation", scope: map[string]string{"app.organization_id": clinicB}, appt: a.bFuture, want: "42501"},
		{name: "a video room, its session's appointment", scope: map[string]string{"app.video_room": "r_" + a.aPast}, appt: a.aPast, want: "42501"},
		{name: "a video room, an appointment the clinic no longer sees", scope: map[string]string{"app.video_room": "r_" + a.bFuture}, appt: a.bFuture, want: "42501"},
	}
	for _, tt := range tests {
		if err := asApp(t, app, tt.scope, insertEncounter(t, owner, tt.appt)); sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %q", tt.name, err, tt.want)
		}
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM encounter").Scan(&n); err != nil || n != 2 {
		t.Errorf("%d encounters, %v; want 2", n, err)
	}
	finish := func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE encounter SET status = 'finished', period_end = period_start + interval '20 minutes',
			updated_at = now(), version_id = version_id + 1`)
		return err
	}
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, finish); err != nil {
		t.Errorf("clinic A finishes its encounter: %v", err)
	}
	var finished int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM encounter WHERE status = 'finished'").Scan(&finished); err != nil || finished != 1 {
		t.Errorf("%d finished, %v; want clinic A's one", finished, err)
	}
}

// The room's scope (app.video_room, as the webhook's on video_session; SPEC.md §2 "Tables", changed after
// package 5) shows the one encounter of the appointment whose session has that room, whatever the consent,
// and finishes it: a revocation that cancels an appointment after the practitioner joined must not leave its
// encounter in progress. It shows nothing more, with a patient or an account set, and without a room the
// clinic's scope still hides that encounter after the revocation.
func TestEncounter_TheRoomScopeSeesAndFinishesOnlyItsRoomsEncounter(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	seedVideoSessions(t, owner, a)
	e := seedEncounters(t, owner, a)
	room := func(appt string) string { return "r_" + appt }

	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "the room of a cancelled appointment", scope: map[string]string{"app.video_room": room(a.bFuture)}, want: sorted(e[a.bFuture])},
		{name: "another room", scope: map[string]string{"app.video_room": room(a.other)}, want: sorted(e[a.other])},
		{name: "an unknown room", scope: map[string]string{"app.video_room": "r_unknown"}},
		{name: "the room with clinic B set", scope: map[string]string{"app.organization_id": clinicB, "app.video_room": room(a.bFuture)},
			want: sorted(e[a.bPast], e[a.bFuture])},
		{name: "clinic B without the room", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(e[a.bPast])},
		{name: "the room in patient B's scope", scope: map[string]string{"app.patient_id": p[1].patient, "app.video_room": room(a.bFuture)},
			want: sorted(e[a.other])},
		{name: "the room with an account set", scope: map[string]string{"app.account_id": p[0].account, "app.video_room": room(a.bFuture)}},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM encounter"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}

	finish := func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE encounter SET status = 'finished', period_end = period_start + interval '20 minutes',
			updated_at = now(), version_id = version_id + 1`)
		return err
	}
	if err := asApp(t, app, map[string]string{"app.organization_id": clinicB, "app.video_room": room(a.bFuture)}, finish); err != nil {
		t.Fatalf("the room's scope with its clinic finishes the encounter: %v", err)
	}
	finished := idsIn(t, owner, nil, "SELECT id::text FROM encounter WHERE status = 'finished' AND period_end IS NOT NULL")
	if want := sorted(e[a.bPast], e[a.bFuture]); !slices.Equal(finished, want) {
		t.Errorf("finished %v, want the room's and clinic B's own %v", finished, want)
	}
}

// salus_app finishes an encounter; it never deletes one or changes whose or which appointment's it is, or
// when it started.
func TestEncounter_TheAppRoleWritesOnlyWhatTheLifecycleNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	seedEncounters(t, owner, a)
	scope := map[string]string{"app.organization_id": clinicA}
	for _, stmt := range []string{
		"DELETE FROM encounter", "TRUNCATE encounter",
		"UPDATE encounter SET appointment_id = appointment_id", "UPDATE encounter SET organization_id = organization_id",
		"UPDATE encounter SET patient_id = patient_id", "UPDATE encounter SET practitioner_role_id = practitioner_role_id",
		"UPDATE encounter SET period_start = period_start", "UPDATE encounter SET created_at = now()",
	} {
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
}

// One encounter per appointment, of its patient, role and clinic; the two statuses; an end exactly when
// finished, not before the start.
func TestEncounter_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	_, _, a := seedAppointments(t, owner)
	seedEncounters(t, owner, a)
	for _, tt := range []struct{ name, stmt, want string }{
		{"a second encounter for the appointment", `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
			SELECT appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start FROM encounter WHERE appointment_id = $1`, "23505"},
		{"an unknown status", `UPDATE encounter SET status = 'planned' WHERE appointment_id = $1`, "23514"},
		{"finished without an end", `UPDATE encounter SET status = 'finished' WHERE appointment_id = $1`, "23514"},
		{"an end while in progress", `UPDATE encounter SET period_end = period_start WHERE appointment_id = $1`, "23514"},
		{"an end before the start", `UPDATE encounter SET status = 'finished', period_end = period_start - interval '1 second' WHERE appointment_id = $1`, "23514"},
		{"an appointment that does not exist", `UPDATE encounter SET appointment_id = gen_random_uuid() WHERE appointment_id = $1`, "23503"},
		{"a patient that does not exist", `UPDATE encounter SET patient_id = gen_random_uuid() WHERE appointment_id = $1`, "23503"},
		{"a role of another clinic", `UPDATE encounter SET organization_id = '` + clinicB + `' WHERE appointment_id = $1`, "23503"},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, a.aFuture); sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %s", tt.name, err, tt.want)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE encounter SET status = 'finished', period_end = period_start WHERE appointment_id = $1`, a.aFuture); err != nil {
		t.Errorf("finished at its start: %v", err)
	}
}
