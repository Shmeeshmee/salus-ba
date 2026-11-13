package appointment_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// The seeded rows of the repository tests: clinic A with role A1, patients A and B (A with an active consent
// with clinic A, B with none), and their accounts.
type seeded struct {
	patientA, patientB, accountA, roleA1 string
}

func seedDB(t *testing.T, owner *pgxpool.Pool) seeded {
	t.Helper()
	var s seeded
	err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id)
		SELECT id::text FROM r`, clinicA).Scan(&s.roleA1)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct{ account, patient *string }{
		{&s.accountA, &s.patientA}, {new(string), &s.patientB},
	} {
		if err := owner.QueryRow(t.Context(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id)
			INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a
			RETURNING account_id::text, id::text`).Scan(x.account, x.patient); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := owner.Exec(t.Context(), "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", s.patientA, clinicA); err != nil {
		t.Fatal(err)
	}
	return s
}

// inScope runs fn in a transaction on the app pool with the repository's scope set by scope.
func inScope(t *testing.T, app *pgxpool.Pool, scope func(ctx context.Context, r appointment.Repository) error, fn func(ctx context.Context, r appointment.Repository) error) error {
	t.Helper()
	r := appointment.NewPostgresRepository(app)
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := scope(ctx, r); err != nil {
			return err
		}
		return fn(ctx, r)
	})
}

func asPatient(id string) func(ctx context.Context, r appointment.Repository) error {
	return func(ctx context.Context, r appointment.Repository) error {
		if err := r.SetPatient(ctx, id); err != nil {
			return err
		}
		return r.SetClinic(ctx, clinicA)
	}
}

func asClinic(ctx context.Context, r appointment.Repository) error { return r.SetClinic(ctx, clinicA) }

func newAppt(s seeded, patientID string, at time.Time) appointment.Appointment {
	return appointment.Appointment{
		PatientID: patientID, PractitionerRoleID: s.roleA1, OrganizationID: clinicA, SlotID: slotFree,
		Start: at, End: at.Add(30 * time.Minute), Created: now,
	}
}

func TestCreateAndByID_StoreABookedAppointmentInThePatientsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var created, read appointment.Appointment
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if created, err = r.Create(ctx, newAppt(s, s.patientA, start)); err != nil {
			return err
		}
		read, err = r.ByID(ctx, created.ID)
		return err
	})
	want := newAppt(s, s.patientA, start)
	want.ID, want.Status, want.VersionID, want.LastUpdated, want.BookedVia = created.ID, appointment.StatusBooked, 1, now, appointment.BookedDirect
	if err != nil || !same(created, want) || !same(read, want) {
		t.Fatalf("Create() = %+v, ByID() = %+v, %v;\nwant %+v", created, read, err, want)
	}
	for name, id := range map[string]string{"not a UUID": "x", "unknown": slotFree} {
		err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
			_, err := r.ByID(ctx, id)
			return err
		})
		if !errors.Is(err, appointment.ErrNotFound) {
			t.Errorf("ByID(%s) = %v, want ErrNotFound", name, err)
		}
	}
	err = inScope(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.Repository) error {
		_, err := r.ByID(ctx, created.ID)
		return err
	})
	if !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("ByID() of patient A's in patient B's scope = %v, want ErrNotFound", err)
	}
}

// same compares appointments with their times as instants.
func same(a, b appointment.Appointment) bool {
	if !a.Start.Equal(b.Start) || !a.End.Equal(b.End) || !a.Created.Equal(b.Created) || !a.CancelledAt.Equal(b.CancelledAt) || !a.LastUpdated.Equal(b.LastUpdated) {
		return false
	}
	a.Start, a.End, a.Created, a.CancelledAt, a.LastUpdated = b.Start, b.End, b.Created, b.CancelledAt, b.LastUpdated
	return a == b
}

// Two bookings at overlapping times with one role, at once: the exclusion constraint lets one through.
func TestCreate_TwoConcurrentOverlappingBookingsOfOneRoleGiveOne(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	if _, err := owner.Exec(t.Context(), "INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())", s.patientB, clinicA); err != nil {
		t.Fatal(err)
	}
	for round := range 10 {
		at := start.Add(time.Duration(round) * time.Hour)
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, p := range []string{s.patientA, s.patientB} {
			wg.Go(func() {
				errs[i] = inScope(t, app, asPatient(p), func(ctx context.Context, r appointment.Repository) error {
					_, err := r.Create(ctx, newAppt(s, p, at.Add(time.Duration(i)*15*time.Minute)))
					return err
				})
			})
		}
		wg.Wait()
		var ok, taken int
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, appointment.ErrSlotUnavailable):
				taken++
			default:
				t.Fatalf("round %d: Create() error = %v", round, err)
			}
		}
		if ok != 1 || taken != 1 {
			t.Errorf("round %d: %d booked and %d refused, want one of each", round, ok, taken)
		}
	}
}

func TestSearch_FiltersWhatTheScopeShowsByStart(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var first, second, other appointment.Appointment
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if second, err = r.Create(ctx, newAppt(s, s.patientA, start.Add(time.Hour))); err != nil {
			return err
		}
		first, err = r.Create(ctx, newAppt(s, s.patientA, start))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := inScope(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.Repository) error {
		var err error
		other, err = r.Create(ctx, newAppt(s, s.patientB, start.Add(2*time.Hour)))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		scope  func(ctx context.Context, r appointment.Repository) error
		name   string
		filter appointment.Filter
		want   []string
	}{
		{name: "patient A, all", scope: asPatient(s.patientA), want: []string{first.ID, second.ID}},
		{name: "clinic A: patient B has no consent", scope: asClinic, filter: appointment.Filter{OrganizationID: clinicA}, want: []string{first.ID, second.ID}},
		{name: "a range", scope: asClinic, filter: appointment.Filter{From: start.Add(time.Minute), To: start.Add(2 * time.Hour)}, want: []string{second.ID}},
		{name: "a status", scope: asClinic, filter: appointment.Filter{Status: appointment.StatusCancelled}},
		{name: "a role", scope: asClinic, filter: appointment.Filter{PractitionerRoleID: s.roleA1, PatientID: s.patientA}, want: []string{first.ID, second.ID}},
		{name: "another patient", scope: asClinic, filter: appointment.Filter{PatientID: s.patientB}},
		{name: "ids that are not UUIDs", scope: asClinic, filter: appointment.Filter{PatientID: "x", PractitionerRoleID: "y", OrganizationID: "z"}},
	}
	for _, tt := range tests {
		var got []string
		err := inScope(t, app, tt.scope, func(ctx context.Context, r appointment.Repository) error {
			all, err := r.Search(ctx, tt.filter)
			for _, a := range all {
				got = append(got, a.ID)
			}
			return err
		})
		if err != nil || len(got) != len(tt.want) || (len(got) > 0 && (got[0] != tt.want[0] || got[len(got)-1] != tt.want[len(tt.want)-1])) {
			t.Errorf("%s: Search() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	_ = other
}

func TestCancelAndSetStatus_ChangeOnlyABookedAppointment(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a, b appointment.Appointment
	at := now.Add(time.Hour)
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if a, err = r.Create(ctx, newAppt(s, s.patientA, start)); err != nil {
			return err
		}
		if b, err = r.Create(ctx, newAppt(s, s.patientA, start.Add(time.Hour))); err != nil {
			return err
		}
		a, err = r.Cancel(ctx, a.ID, appointment.ReasonPatient, s.accountA, at)
		return err
	})
	if err != nil || a.Status != appointment.StatusCancelled || a.Reason != appointment.ReasonPatient || a.CancelledBy != s.accountA ||
		!a.CancelledAt.Equal(at) || !a.LastUpdated.Equal(at) || a.VersionID != 2 {
		t.Fatalf("Cancel() = %+v, %v; want cancelled by the patient at version 2", a, err)
	}
	err = inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		var err error
		if b, err = r.SetStatus(ctx, b.ID, appointment.StatusNoShow, at); err != nil {
			return err
		}
		if _, err := r.SetStatus(ctx, b.ID, appointment.StatusFulfilled, at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("SetStatus() of a no-show = %v, want ErrClosed", err)
		}
		if _, err := r.Cancel(ctx, a.ID, appointment.ReasonProvider, "", at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("Cancel() of a cancelled one = %v, want ErrClosed", err)
		}
		if _, err := r.Cancel(ctx, "x", appointment.ReasonProvider, "", at); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("Cancel() of an id that is not a UUID = %v, want ErrClosed", err)
		}
		_, err = r.SetStatus(ctx, "x", appointment.StatusNoShow, at)
		if !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("SetStatus() of an id that is not a UUID = %v, want ErrClosed", err)
		}
		return nil
	})
	if err != nil || b.Status != appointment.StatusNoShow || b.VersionID != 2 {
		t.Errorf("SetStatus() = %+v, %v; want a no-show at version 2", b, err)
	}
	// A job's cancellation names no account.
	var c appointment.Appointment
	err = inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if c, err = r.Create(ctx, newAppt(s, s.patientA, start.Add(3*time.Hour))); err != nil {
			return err
		}
		c, err = r.Cancel(ctx, c.ID, appointment.ReasonPatient, "", at)
		return err
	})
	if err != nil || c.CancelledBy != "" || c.Status != appointment.StatusCancelled {
		t.Errorf("Cancel() by a job = %+v, %v; want cancelled by no account", c, err)
	}
}

func TestSetNotice_ShowsTheJobOneAppointmentOfItsClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var b appointment.Appointment
	if err := inScope(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.Repository) error {
		var err error
		b, err = r.Create(ctx, newAppt(s, s.patientB, start))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	notice := func(ctx context.Context, r appointment.Repository) error {
		if err := r.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return r.SetNotice(ctx, b.ID)
	}
	for name, scope := range map[string]func(ctx context.Context, r appointment.Repository) error{"the clinic": asClinic, "the job": notice} {
		err := inScope(t, app, scope, func(ctx context.Context, r appointment.Repository) error {
			_, err := r.ByID(ctx, b.ID)
			return err
		})
		if want := map[string]error{"the clinic": appointment.ErrNotFound, "the job": nil}[name]; !errors.Is(err, want) {
			t.Errorf("%s reads patient B's appointment without a consent: %v, want %v", name, err, want)
		}
	}
}

func TestSetScope_NeedsATransaction(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	r := appointment.NewPostgresRepository(app)
	for name, set := range map[string]func(ctx context.Context, id string) error{"SetPatient": r.SetPatient, "SetClinic": r.SetClinic, "SetNotice": r.SetNotice} {
		if err := set(t.Context(), clinicA); !errors.Is(err, postgres.ErrNoTx) {
			t.Errorf("%s outside a transaction = %v, want postgres.ErrNoTx", name, err)
		}
	}
}

// ByIDForShare reads the appointment as ByID does and locks it against a change until the transaction ends.
// Row-level security lets the patient's own scope take that lock, and the clinic's (a request, the default
// request job) while the patient's consent with it is active; not the clinic without one.
func TestByIDForShare_LocksTheAppointmentInThePatientsScopeAndTheClinicsWithAConsent(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a, b appointment.Appointment
	if err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		a, err = r.Create(ctx, newAppt(s, s.patientA, start))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := inScope(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.Repository) error {
		var err error
		b, err = r.Create(ctx, newAppt(s, s.patientB, start.Add(time.Hour)))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	patientOnly := func(id string) func(ctx context.Context, r appointment.Repository) error {
		return func(ctx context.Context, r appointment.Repository) error { return r.SetPatient(ctx, id) }
	}
	tests := []struct {
		scope func(ctx context.Context, r appointment.Repository) error
		want  error
		name  string
		id    string
	}{
		{name: "the patient's own", scope: patientOnly(s.patientA), id: a.ID},
		{name: "the clinic with the patient's consent", scope: asClinic, id: a.ID},
		{name: "the clinic without a consent", scope: asClinic, id: b.ID, want: appointment.ErrNotFound},
		{name: "another patient", scope: patientOnly(s.patientB), id: a.ID, want: appointment.ErrNotFound},
		{name: "an id that is not a UUID", scope: asClinic, id: "x", want: appointment.ErrNotFound},
	}
	for _, tt := range tests {
		var got appointment.Appointment
		err := inScope(t, app, tt.scope, func(ctx context.Context, r appointment.Repository) error {
			var err error
			got, err = r.ByIDForShare(ctx, tt.id)
			return err
		})
		if !errors.Is(err, tt.want) || (tt.want == nil && !same(got, a)) {
			t.Errorf("%s: ByIDForShare() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// A cancellation waits while another transaction holds the appointment's share lock, and commits once it ends;
// the next ByIDForShare sees it cancelled.
func TestByIDForShare_MakesACancellationWaitForTheTransaction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	var a appointment.Appointment
	if err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		a, err = r.Create(ctx, newAppt(s, s.patientA, start))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	locked, release, cancelled := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	held := make(chan error, 1)
	go func() {
		held <- inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
			if _, err := r.ByIDForShare(ctx, a.ID); err != nil {
				close(locked)
				return err
			}
			close(locked)
			select { // until released, or the test ends
			case <-release:
			case <-ctx.Done():
			}
			return nil
		})
	}()
	<-locked
	go func() {
		cancelled <- inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
			_, err := r.Cancel(ctx, a.ID, appointment.ReasonProvider, "", now)
			return err
		})
	}()
	pgtest.WaitForLockWait(t, owner)
	select {
	case err := <-cancelled:
		t.Fatalf("Cancel() returned %v while the lock was held, want it to wait", err)
	default:
	}
	close(release)
	if err := <-held; err != nil {
		t.Fatalf("the locking transaction = %v", err)
	}
	if err := <-cancelled; err != nil {
		t.Fatalf("Cancel() after the lock was released = %v", err)
	}
	err := inScope(t, app, asClinic, func(ctx context.Context, r appointment.Repository) error {
		got, err := r.ByIDForShare(ctx, a.ID)
		if err == nil && got.Status != appointment.StatusCancelled {
			t.Errorf("ByIDForShare() after the cancellation = %s, want cancelled", got.Status)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
