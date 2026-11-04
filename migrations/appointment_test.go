package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The revocation of patient A's consent with clinic B, and the times around it.
const (
	revokedB     = "2026-11-02T08:00:00Z"
	beforeRevoke = "2026-10-30T10:00:00Z" // a past appointment: it started before the revocation
	afterRevoke  = "2030-01-07T09:00:00Z" // a future one
)

// appointmentRows is the seeded appointments. Patient A: aPast (clinic A, past) and aFuture (clinic A,
// future), with an active consent; bPast and bFuture (clinic B, cancelled by the revocation) after the
// revocation of their consent with clinic B. Patient B: other (clinic A, future), without any consent.
type appointmentRows struct {
	aPast, aFuture, bPast, bFuture, other string
}

// seedAppointments creates, as the owner, the directory, patients A and B, patient A's consents and the
// appointments.
func seedAppointments(t *testing.T, owner *pgxpool.Pool) ([2]patientRow, map[string]directory, appointmentRows) {
	t.Helper()
	ctx := t.Context()
	dir := seedDirectory(t, owner)
	p := seedPatients(t, owner)
	if _, err := owner.Exec(ctx, `INSERT INTO consent (patient_id, organization_id, status, date_time, revoked_at)
		VALUES ($1, $2, 'active', '2026-10-01T08:00:00Z', NULL), ($1, $3, 'inactive', '2026-10-01T08:00:00Z', $4)`,
		p[0].patient, clinicA, clinicB, revokedB); err != nil {
		t.Fatal(err)
	}
	var a appointmentRows
	for _, row := range []struct {
		id                     *string
		patient, clinic, start string
		cancelled              bool
	}{
		{&a.aPast, p[0].patient, clinicA, "2026-10-29T10:00:00Z", false},
		{&a.aFuture, p[0].patient, clinicA, afterRevoke, false},
		{&a.bPast, p[0].patient, clinicB, beforeRevoke, false},
		{&a.bFuture, p[0].patient, clinicB, afterRevoke, true},
		{&a.other, p[1].patient, clinicA, "2030-01-08T09:00:00Z", false},
	} {
		err := owner.QueryRow(ctx, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status,
				cancelled_at, cancellation_reason)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'),
				CASE WHEN $5 THEN 'cancelled' ELSE 'booked' END, CASE WHEN $5 THEN $6::timestamptz END, CASE WHEN $5 THEN 'pat' END)
			RETURNING id`, row.patient, dir[row.clinic].role, row.clinic, row.start, row.cancelled, revokedB).Scan(row.id)
		if err != nil {
			t.Fatal(err)
		}
	}
	return p, dir, a
}

func TestAppointment_EachScopeSeesOnlyItsAppointments(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(a.aPast, a.aFuture, a.bPast, a.bFuture)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(a.other)},
		{name: "patient A with clinic B set", scope: map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicB},
			want: sorted(a.aPast, a.aFuture, a.bPast, a.bFuture)},
		{name: "clinic A: patient A consents, patient B never did", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(a.aPast, a.aFuture)},
		{name: "clinic B after the revocation: the past appointment only", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(a.bPast)},
		{name: "account A with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
		{name: "patient B with clinic A set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicA}, want: sorted(a.other)},
		{name: "the notifier: clinic B and its cancelled appointment", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture},
			want: sorted(a.bPast, a.bFuture)},
		{name: "the notifier: another clinic's appointment", scope: map[string]string{"app.organization_id": clinicA, "app.appointment_id": a.bFuture},
			want: sorted(a.aPast, a.aFuture)},
		{name: "an appointment without a clinic", scope: map[string]string{"app.appointment_id": a.bFuture}},
		{name: "the notifier's setting in a patient's scope", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicB, "app.appointment_id": a.bFuture},
			want: sorted(a.other)},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM appointment"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// After a revocation the clinic keeps the patient through a past appointment (name, times and status are
// the service's business); with neither a consent nor a past appointment it sees nothing. The notifier sees
// the patient of the one appointment it was given.
func TestAppointment_AClinicSeesThePatientOfAPastAppointmentAfterARevocation(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	const patients = "SELECT id::text FROM patient"

	tests := []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "clinic A, with patient A's consent", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(p[0].patient)},
		{name: "clinic B, after the revocation, through the past appointment", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(p[0].patient)},
		{name: "the notifier, through patient B's appointment", scope: map[string]string{"app.organization_id": clinicA, "app.appointment_id": a.other}, want: sorted(p[0].patient, p[1].patient)},
		{name: "the notifier, another clinic's appointment", scope: map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.other}, want: sorted(p[0].patient)},
		{name: "patient B with clinic B set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicB}, want: sorted(p[1].patient)},
	}
	for _, tt := range tests {
		if got := idsIn(t, app, tt.scope, patients); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}

	// A past appointment of clinic B, but before it any revocation: the past branch needs a revocation.
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange('2026-10-01T10:00:00Z', '2026-10-01T10:30:00Z'))`, p[1].patient, dir[clinicB].role, clinicB); err != nil {
		t.Fatal(err)
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicB}, patients); !slices.Equal(got, sorted(p[0].patient)) {
		t.Errorf("clinic B with patient B's past appointment and no consent ever: sees %v, want only patient A", got)
	}
}

// Right after the revocation commits, before any job has run, the clinic loses the future appointment and
// keeps the past one and, through it, the patient's row.
func TestAppointment_TheRevocationEndsTheClinicsViewOfTheFutureWhenItCommits(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	err := asApp(t, app, map[string]string{"app.patient_id": p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE consent SET status = 'inactive', revoked_at = now(), version_id = version_id + 1, updated_at = now()
			WHERE organization_id = $1 AND status = 'active'`, clinicA)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	scopeA := map[string]string{"app.organization_id": clinicA}
	if got := idsIn(t, app, scopeA, "SELECT id::text FROM appointment"); !slices.Equal(got, []string{a.aPast}) {
		t.Errorf("clinic A after the revocation: sees %v, want only the past appointment %s", got, a.aPast)
	}
	if got := idsIn(t, app, scopeA, "SELECT id::text FROM patient"); !slices.Equal(got, []string{p[0].patient}) {
		t.Errorf("clinic A after the revocation: sees patients %v, want patient A through the past appointment", got)
	}
}

func TestAppointment_AClinicChangesOnlyTheAppointmentsOfAnActiveConsentAndBooksNone(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	update := func(scope map[string]string, id string) (int64, error) {
		var n int64
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, "UPDATE appointment SET status = 'noshow', version_id = version_id + 1, updated_at = now() WHERE id = $1", id)
			n = tag.RowsAffected()
			return err
		})
		return n, err
	}
	if n, err := update(map[string]string{"app.organization_id": clinicA}, a.aPast); err != nil || n != 1 {
		t.Errorf("clinic A marks its consenting patient's appointment: %d rows, %v; want 1", n, err)
	}
	if n, err := update(map[string]string{"app.organization_id": clinicB}, a.bPast); err != nil || n != 0 {
		t.Errorf("clinic B marks a past appointment after the revocation: %d rows, %v; want 0", n, err)
	}
	if n, err := update(map[string]string{"app.organization_id": clinicB, "app.appointment_id": a.bFuture}, a.bFuture); err != nil || n != 0 {
		t.Errorf("the notifier's scope changes an appointment: %d rows, %v; want 0", n, err)
	}
	err := asApp(t, app, map[string]string{"app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			VALUES ($1, $2, $3, gen_random_uuid(), tstzrange('2030-02-01T10:00:00Z', '2030-02-01T10:30:00Z'))`, p[0].patient, dir[clinicA].role, clinicA)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("a booking in a clinic's scope: %v, want an RLS violation (42501)", err)
	}
}

func TestAppointment_APatientBooksOnlyForThemselvesAndNeverTwiceAtOnceWithOneRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	book := func(patient, start string) error {
		return asApp(t, app, map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
				VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'))`,
				patient, dir[clinicA].role, clinicA, start)
			return err
		})
	}
	if err := book(p[1].patient, "2030-03-01T10:00:00Z"); sqlState(err) != "42501" {
		t.Errorf("booking for another patient: %v, want an RLS violation (42501)", err)
	}
	if err := book(p[0].patient, "2030-03-01T10:00:00Z"); err != nil {
		t.Fatalf("booking: %v", err)
	}
	if err := book(p[0].patient, "2030-03-01T10:15:00Z"); sqlState(err) != "23P01" {
		t.Errorf("an overlapping booking with the same role: %v, want an exclusion violation (23P01)", err)
	}
	if err := book(p[0].patient, "2030-03-01T10:30:00Z"); err != nil {
		t.Errorf("the next half hour: %v, want it booked", err)
	}
	// bFuture is cancelled: its time is free again for its role.
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'))`,
		p[1].patient, dir[clinicB].role, clinicB, afterRevoke); err != nil {
		t.Errorf("booking the time of a cancelled appointment: %v, want it booked", err)
	}
	_ = a
}

func TestAppointment_TheAppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	scope := map[string]string{"app.patient_id": p[0].patient}
	for _, stmt := range []string{
		"DELETE FROM appointment", "TRUNCATE appointment",
		"UPDATE appointment SET patient_id = patient_id", "UPDATE appointment SET organization_id = organization_id",
		"UPDATE appointment SET practitioner_role_id = practitioner_role_id", "UPDATE appointment SET period = period",
		"UPDATE appointment SET slot_id = slot_id", "UPDATE appointment SET created_at = now()",
	} {
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege (42501)", stmt, err)
		}
	}
	err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'pat',
			cancelled_by_account_id = $2, updated_at = now(), version_id = version_id + 1 WHERE id = $1`, a.aFuture, p[0].account)
		return err
	})
	if err != nil {
		t.Errorf("the patient cancels their appointment: %v", err)
	}
}

func TestAppointment_HoldsAConsistentState(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	for name, stmt := range map[string]string{
		"an unknown status":                "UPDATE appointment SET status = 'pending' WHERE id = $1",
		"cancelled without a time":         "UPDATE appointment SET status = 'cancelled', cancellation_reason = 'pat' WHERE id = $1",
		"cancelled without a reason":       "UPDATE appointment SET status = 'cancelled', cancelled_at = now() WHERE id = $1",
		"a reason that is no code":         "UPDATE appointment SET status = 'cancelled', cancelled_at = now(), cancellation_reason = 'other' WHERE id = $1",
		"a cancellation time while booked": "UPDATE appointment SET cancelled_at = now() WHERE id = $1",
		"an open period":                   "UPDATE appointment SET period = tstzrange(lower(period), NULL) WHERE id = $1",
		"an empty period":                  "UPDATE appointment SET period = tstzrange(lower(period), lower(period)) WHERE id = $1",
		"a role of another clinic":         "UPDATE appointment SET organization_id = '" + clinicB + "' WHERE id = $1",
		"a canceller while not cancelled":  "UPDATE appointment SET cancelled_by_account_id = (SELECT id FROM account LIMIT 1) WHERE id = $1",
		"an unknown patient":               "UPDATE appointment SET patient_id = gen_random_uuid() WHERE id = $1",
	} {
		if _, err := owner.Exec(t.Context(), stmt, a.aFuture); sqlState(err) == "" {
			t.Errorf("%s: accepted, want a violation", name)
		}
	}
}
