package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedVideoSessions creates, as the owner, a video session for each seeded appointment and returns their ids
// by appointment, with the room of each (r_<appointment id>: synthetic, the application's are random).
func seedVideoSessions(t *testing.T, owner *pgxpool.Pool, a appointmentRows) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for _, appt := range []string{a.aPast, a.aFuture, a.bPast, a.bFuture, a.other} {
		var id string
		err := owner.QueryRow(t.Context(), `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity,
				practitioner_identity, closes_at)
			SELECT id, organization_id, 'r_' || id, 'p_pat_' || id, 'p_pra_' || id, upper(period) + interval '30 minutes'
			FROM appointment WHERE id = $1
			RETURNING id`, appt).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids[appt] = id
	}
	return ids
}

// Row-level security as appointment's (docs/consultation/SPEC.md §2 "Tables"): the patient their own; a
// clinic those of its appointments with an active consent, and after a revocation those of the appointments
// that started before it; the webhook (app.video_room) the one session of its room.
func TestVideoSession_EachScopeSeesOnlyItsSessions(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	s := seedVideoSessions(t, owner, a)

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(s[a.aPast], s[a.aFuture], s[a.bPast], s[a.bFuture])},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(s[a.other])},
		{name: "patient B with clinic A set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicA}, want: sorted(s[a.other])},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(s[a.aPast], s[a.aFuture])},
		{name: "clinic B after the revocation: the past appointment's only", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(s[a.bPast])},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
		{name: "the webhook: one room", scope: map[string]string{"app.video_room": "r_" + a.other}, want: sorted(s[a.other])},
		{name: "the webhook: the room of a cancelled appointment", scope: map[string]string{"app.video_room": "r_" + a.bFuture}, want: sorted(s[a.bFuture])},
		{name: "the webhook: an unknown room", scope: map[string]string{"app.video_room": "r_unknown"}},
		{name: "the webhook's room in patient A's scope", scope: map[string]string{"app.patient_id": p[0].patient, "app.video_room": "r_" + a.other},
			want: sorted(s[a.aPast], s[a.aFuture], s[a.bPast], s[a.bFuture])},
		{name: "the webhook's room with clinic B set", scope: map[string]string{"app.organization_id": clinicB, "app.video_room": "r_" + a.other},
			want: sorted(s[a.bPast], s[a.other])},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM video_session"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// The first token request creates the session in the caller's scope: the patient for their own appointment
// only, the clinic for one with an active consent only. A second insert for the appointment does nothing.
func TestVideoSession_AScopeCreatesSessionsOnlyForAppointmentsItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	insert := func(appt string) func(ctx context.Context, db postgres.Querier) error {
		return func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity, practitioner_identity, closes_at)
				VALUES ($1, (SELECT organization_id FROM appointment WHERE id = $1), 'r_' || $1, 'p_a' || $1, 'p_b' || $1, now())
				ON CONFLICT (appointment_id) DO NOTHING`, appt)
			return err
		}
	}
	tests := []struct {
		scope map[string]string
		name  string
		appt  string
		want  string // the SQLSTATE, "" for success
	}{
		{name: "patient A, their appointment", scope: map[string]string{"app.patient_id": p[0].patient}, appt: a.aFuture},
		{name: "patient A, again: nothing", scope: map[string]string{"app.patient_id": p[0].patient}, appt: a.aFuture},
		{name: "patient A, patient B's appointment", scope: map[string]string{"app.patient_id": p[0].patient}, appt: a.other, want: "42501"},
		{name: "clinic A, an appointment with an active consent", scope: map[string]string{"app.organization_id": clinicA}, appt: a.aPast},
		{name: "clinic A, without a consent", scope: map[string]string{"app.organization_id": clinicA}, appt: a.other, want: "42501"},
		{name: "clinic B, a past appointment after the revocation", scope: map[string]string{"app.organization_id": clinicB}, appt: a.bPast},
		{name: "the webhook's scope", scope: map[string]string{"app.video_room": "r_" + a.bFuture}, appt: a.bFuture, want: "42501"},
	}
	for _, tt := range tests {
		if err := asApp(t, app, tt.scope, insert(tt.appt)); sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %q", tt.name, err, tt.want)
		}
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM video_session").Scan(&n); err != nil || n != 3 {
		t.Errorf("%d sessions, %v; want 3", n, err)
	}
}

// salus_app records who joined and the end; it never deletes a session or changes what it is for.
func TestVideoSession_TheAppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	seedVideoSessions(t, owner, a)
	scope := map[string]string{"app.patient_id": p[0].patient}
	for _, stmt := range []string{
		"DELETE FROM video_session", "TRUNCATE video_session",
		"UPDATE video_session SET appointment_id = appointment_id", "UPDATE video_session SET organization_id = organization_id",
		"UPDATE video_session SET room = room", "UPDATE video_session SET patient_identity = patient_identity",
		"UPDATE video_session SET practitioner_identity = practitioner_identity", "UPDATE video_session SET closes_at = closes_at",
		"UPDATE video_session SET created_at = now()",
	} {
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	err := asApp(t, app, map[string]string{"app.video_room": "r_" + a.aFuture}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE video_session SET patient_joined_at = now(), practitioner_joined_at = now(), ended_at = now(),
			end_reason = 'practitioner', updated_at = now()`)
		return err
	})
	if err != nil {
		t.Errorf("the webhook records the joins and the end: %v", err)
	}
}

// One session per appointment, random names unique, an end with its reason or neither, and only the five
// reasons.
func TestVideoSession_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	_, _, a := seedAppointments(t, owner)
	seedVideoSessions(t, owner, a)
	for _, tt := range []struct{ name, stmt, want string }{
		{"a second session for the appointment", `INSERT INTO video_session (appointment_id, organization_id, room, patient_identity, practitioner_identity, closes_at)
			SELECT appointment_id, organization_id, 'r_new', 'p_new1', 'p_new2', closes_at FROM video_session WHERE appointment_id = $1`, "23505"},
		{"a room twice", `UPDATE video_session SET room = 'r_' || $1 WHERE appointment_id <> $1`, "23505"},
		{"a patient identity twice", `UPDATE video_session SET patient_identity = 'p_pat_' || $1 WHERE appointment_id <> $1`, "23505"},
		{"a practitioner identity twice", `UPDATE video_session SET practitioner_identity = 'p_pra_' || $1 WHERE appointment_id <> $1`, "23505"},
		{"one identity for both roles", `UPDATE video_session SET practitioner_identity = patient_identity WHERE appointment_id = $1`, "23514"},
		{"an end without a reason", `UPDATE video_session SET ended_at = now() WHERE appointment_id = $1`, "23514"},
		{"a reason without an end", `UPDATE video_session SET end_reason = 'practitioner' WHERE appointment_id = $1`, "23514"},
		{"an unknown reason", `UPDATE video_session SET ended_at = now(), end_reason = 'bored' WHERE appointment_id = $1`, "23514"},
		{"an appointment that does not exist", `UPDATE video_session SET appointment_id = gen_random_uuid() WHERE appointment_id = $1`, "23503"},
	} {
		if _, err := owner.Exec(t.Context(), tt.stmt, a.aFuture); sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %s", tt.name, err, tt.want)
		}
	}
	for _, reason := range []string{"practitioner", "window_closed", "practitioner_absent", "time_limit", "cancelled"} {
		if _, err := owner.Exec(t.Context(), `UPDATE video_session SET ended_at = now(), end_reason = $2 WHERE appointment_id = $1`, a.aFuture, reason); err != nil {
			t.Errorf("end_reason %s: %v", reason, err)
		}
	}
}

// The practitioner's read of a consultation they were in (docs/consultation/SPEC.md §2 "Consultation status",
// changed after package 6): a clinic's scope narrowed to one appointment (app.appointment_id, as the notice
// scope of appointment) sees that appointment's session once its practitioner joined, whatever the consent,
// so a revocation that cancelled it before its start does not hide why it ended. Before the practitioner
// joined, for another clinic, without the clinic, or with a patient or an account set, it sees nothing more.
func TestVideoSession_OneAppointmentsScopeSeesItsSessionOnceThePractitionerJoined(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	s := seedVideoSessions(t, owner, a)
	one := func(extra map[string]string) map[string]string {
		scope := map[string]string{"app.appointment_id": a.bFuture}
		for k, v := range extra {
			scope[k] = v
		}
		return scope
	}
	query := "SELECT id::text FROM video_session"
	if got, want := idsIn(t, app, one(map[string]string{"app.organization_id": clinicB}), query), sorted(s[a.bPast]); !slices.Equal(got, want) {
		t.Errorf("before the practitioner joined: sees %v, want %v", got, want)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE video_session SET practitioner_joined_at = now() WHERE appointment_id IN ($1, $2)", a.bFuture, a.aFuture); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "clinic B with the cancelled appointment", scope: one(map[string]string{"app.organization_id": clinicB}), want: sorted(s[a.bPast], s[a.bFuture])},
		{name: "clinic A with clinic B's appointment", scope: one(map[string]string{"app.organization_id": clinicA}), want: sorted(s[a.aPast], s[a.aFuture])},
		{name: "the appointment without a clinic", scope: one(nil)},
		{name: "clinic B without the appointment", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(s[a.bPast])},
		{name: "with patient B set", scope: one(map[string]string{"app.organization_id": clinicB, "app.patient_id": p[1].patient}), want: sorted(s[a.other])},
		{name: "with an account set", scope: one(map[string]string{"app.organization_id": clinicB, "app.account_id": p[0].account})},
	} {
		if got := idsIn(t, app, tt.scope, query); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
	err := asApp(t, app, one(map[string]string{"app.organization_id": clinicB}), func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "UPDATE video_session SET ended_at = now(), end_reason = 'practitioner' WHERE appointment_id = $1", a.bFuture)
		if err == nil && tag.RowsAffected() != 0 {
			t.Errorf("the scope of one appointment ended its session; want a read only")
		}
		return err
	})
	if err != nil {
		t.Errorf("the update: %v", err)
	}
}

// salus_app records why the patient was removed, once, in the room's scope; only the two reasons of a
// removal are stored.
func TestVideoSession_RecordsWhyThePatientWasRemoved(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	seedVideoSessions(t, owner, a)
	err := asApp(t, app, map[string]string{"app.video_room": "r_" + a.bFuture}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, "UPDATE video_session SET removal_reason = 'consent_withdrawn', updated_at = now()")
		return err
	})
	if err != nil {
		t.Errorf("the room's scope records the removal: %v", err)
	}
	for _, reason := range []string{"cancelled", "consent_withdrawn"} {
		if _, err := owner.Exec(t.Context(), "UPDATE video_session SET removal_reason = $2 WHERE appointment_id = $1", a.aFuture, reason); err != nil {
			t.Errorf("removal_reason %s: %v", reason, err)
		}
	}
	for _, reason := range []string{"left", "ended", "bored"} {
		if _, err := owner.Exec(t.Context(), "UPDATE video_session SET removal_reason = $2 WHERE appointment_id = $1", a.aFuture, reason); sqlState(err) != "23514" {
			t.Errorf("removal_reason %s: %v, want check_violation (23514)", reason, err)
		}
	}
}
