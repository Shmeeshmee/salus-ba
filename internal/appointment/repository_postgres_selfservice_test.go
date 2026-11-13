package appointment_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// bookInDB books patient A's appointment at at in the patient's scope.
func bookInDB(t *testing.T, app *pgxpool.Pool, s seeded, at time.Time) appointment.Appointment {
	t.Helper()
	var a appointment.Appointment
	if err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		a, err = r.Create(ctx, newAppt(s, s.patientA, at))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

// inSelf runs fn with the self-service repository in a transaction scoped by scope.
func inSelf(t *testing.T, app *pgxpool.Pool, scope func(ctx context.Context, r appointment.Repository) error, fn func(ctx context.Context, r appointment.SelfServiceRepository) error) error {
	t.Helper()
	self := appointment.NewPostgresSelfServiceRepository(app)
	return inScope(t, app, scope, func(ctx context.Context, _ appointment.Repository) error { return fn(ctx, self) })
}

func TestConfirm_StoresThePatientsConfirmationOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	a := bookInDB(t, app, s, start)
	err := inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		got, err := r.Confirm(ctx, a.ID, appointment.ViaLink, now)
		if err != nil || !got.ConfirmedAt.Equal(now) || got.ConfirmedVia != appointment.ViaLink || got.VersionID != a.VersionID+1 {
			t.Errorf("Confirm() = %+v, %v; want confirmed via the link at the next version", got, err)
		}
		if _, err := r.Confirm(ctx, a.ID, appointment.ViaApp, now.Add(time.Minute)); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("Confirm() again = %v, want ErrClosed", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The clinic's scope may not confirm for the patient (the guard trigger), nor may another patient's.
	b := bookInDB(t, app, s, start.Add(time.Hour))
	for name, scope := range map[string]func(ctx context.Context, r appointment.Repository) error{"clinic": asClinic, "patient B": asPatient(s.patientB)} {
		err := inSelf(t, app, scope, func(ctx context.Context, r appointment.SelfServiceRepository) error {
			_, err := r.Confirm(ctx, b.ID, appointment.ViaApp, now)
			return err
		})
		if err == nil {
			t.Errorf("Confirm() in the %s scope succeeded", name)
		}
	}
}

func TestReschedule_MovesTheAppointmentOnceAndTheExclusionConstraintGuards(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	a := bookInDB(t, app, s, start)
	other := bookInDB(t, app, s, start.Add(3*time.Hour))
	to := appointment.Move{Start: start.Add(time.Hour), End: start.Add(90 * time.Minute), SlotID: slotSoon, From: 0}
	err := inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		got, err := r.Reschedule(ctx, a.ID, to, now)
		if err != nil || !got.Start.Equal(to.Start) || !got.End.Equal(to.End) || got.SlotID != slotSoon || got.RescheduleCount != 1 ||
			got.VersionID != a.VersionID+1 || !got.ConfirmedAt.Equal(now) || got.ConfirmedVia != appointment.ViaApp {
			t.Errorf("Reschedule() = %+v, %v; want moved, counted, confirmed in the app", got, err)
		}
		// A reschedule decided on the old count changes nothing.
		if _, err := r.Reschedule(ctx, a.ID, to, now); !errors.Is(err, appointment.ErrClosed) {
			t.Errorf("Reschedule() from a stale count = %v, want ErrClosed", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Onto the other appointment's time: the exclusion constraint refuses it.
	err = inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		_, err := r.Reschedule(ctx, a.ID, appointment.Move{Start: other.Start, End: other.End, SlotID: slotPast, From: 1}, now)
		return err
	})
	if !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Errorf("Reschedule() onto a booked time = %v, want ErrSlotUnavailable", err)
	}
	// The clinic's scope may not move it.
	err = inSelf(t, app, asClinic, func(ctx context.Context, r appointment.SelfServiceRepository) error {
		_, err := r.Reschedule(ctx, a.ID, appointment.Move{Start: start.Add(5 * time.Hour), End: start.Add(330 * time.Minute), SlotID: slotU, From: 1}, now)
		return err
	})
	if err == nil {
		t.Error("Reschedule() in the clinic's scope succeeded")
	}
}

func TestByIDForUpdate_MakesAConcurrentRescheduleWait(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	a := bookInDB(t, app, s, start)
	// Two reschedules of one appointment at once, each reading under the lock and then moving: the second waits for the
	// first and reads its count, so both move it, one after the other (without the lock both would read 0 and the
	// second would change nothing).
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, slotID := range []string{slotSoon, slotPast} {
		wg.Go(func() {
			errs[i] = inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
				cur, err := r.ByIDForUpdate(ctx, a.ID)
				if err != nil {
					return err
				}
				at := start.Add(time.Duration(i+1) * time.Hour)
				_, err = r.Reschedule(ctx, a.ID, appointment.Move{Start: at, End: at.Add(30 * time.Minute), SlotID: slotID, From: cur.RescheduleCount}, now)
				return err
			})
		})
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("reschedules = %v, %v; want both to run under the lock", errs[0], errs[1])
	}
	var count int
	if err := owner.QueryRow(t.Context(), "SELECT reschedule_count FROM appointment WHERE id = $1", a.ID).Scan(&count); err != nil || count != 2 {
		t.Errorf("reschedule_count = %d, %v; want 2 (one after the other)", count, err)
	}
}

func TestTokens_FoundByHashReadInThePatientsScopeUsedOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	a := bookInDB(t, app, s, start)
	hash := sha256.Sum256([]byte("synthetic link"))
	tokens := appointment.NewPostgresTokenRepository(app)
	if err := inScope(t, app, func(ctx context.Context, r appointment.Repository) error {
		if err := r.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return r.SetNotice(ctx, a.ID)
	}, func(ctx context.Context, _ appointment.Repository) error {
		return tokens.InsertToken(ctx, appointment.Token{
			CreatedAt: now, StartAt: a.Start, AppointmentID: a.ID, OrganizationID: clinicA, PatientID: s.patientA, AccountID: s.accountA,
			Language: "tr", Channel: appointment.ChannelSMS, Hash: hash,
		})
	}); err != nil {
		t.Fatal(err)
	}
	self := appointment.NewPostgresSelfServiceRepository(app)
	var ids appointment.Token
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		var err error
		ids, err = self.FindToken(ctx, hash) // no scope at all
		if _, err := self.FindToken(ctx, sha256.Sum256([]byte("another"))); !errors.Is(err, appointment.ErrLinkInvalid) {
			t.Errorf("FindToken(unknown) = %v, want ErrLinkInvalid", err)
		}
		return err
	}); err != nil || ids.ID == "" || ids.AppointmentID != a.ID || ids.PatientID != s.patientA || ids.AccountID != s.accountA || ids.OrganizationID != clinicA {
		t.Fatalf("FindToken() = %+v, %v; want the link's ids", ids, err)
	}
	// Patient B's scope does not see or use it.
	if err := inSelf(t, app, asPatient(s.patientB), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		if _, err := r.TokenByID(ctx, ids.ID); !errors.Is(err, appointment.ErrLinkInvalid) {
			t.Errorf("TokenByID() as patient B = %v, want ErrLinkInvalid", err)
		}
		if err := r.UseToken(ctx, ids.ID, appointment.LinkCancel, now); !errors.Is(err, appointment.ErrLinkInvalid) {
			t.Errorf("UseToken() as patient B = %v, want ErrLinkInvalid", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
		tok, err := r.TokenByID(ctx, ids.ID)
		if err != nil || tok.Language != "tr" || !tok.StartAt.Equal(a.Start) || !tok.UsedAt.IsZero() || tok.Hash != hash || tok.Channel != appointment.ChannelSMS {
			t.Errorf("TokenByID() = %+v, %v; want the unused link", tok, err)
		}
		if err := r.UseToken(ctx, ids.ID, appointment.LinkConfirm, now); err != nil {
			t.Errorf("UseToken() = %v", err)
		}
		if err := r.UseToken(ctx, ids.ID, appointment.LinkCancel, now); !errors.Is(err, appointment.ErrLinkInvalid) {
			t.Errorf("UseToken() again = %v, want ErrLinkInvalid", err)
		}
		tok, err = r.TokenByID(ctx, ids.ID)
		if err != nil || !tok.UsedAt.Equal(now) || tok.UsedAction != appointment.LinkConfirm {
			t.Errorf("TokenByID() after use = %+v, %v; want used for confirm", tok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Two uses of one link at once: exactly one wins, the other finds it used.
func TestUseToken_TwoAtOnceUseTheLinkOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	a := bookInDB(t, app, s, start)
	self := appointment.NewPostgresSelfServiceRepository(app)
	hash2 := sha256.Sum256([]byte("second link"))
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment_action_token (created_at, expires_at, start_at, appointment_id, organization_id, patient_id, account_id, token_hash, channel, language)
		VALUES ($1, $2, $2, $3, $4, $5, $6, $7, 'email', 'bs')`, now, a.Start, a.ID, clinicA, s.patientA, s.accountA, hash2[:]); err != nil {
		t.Fatal(err)
	}
	var second appointment.Token
	_ = postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		second, _ = self.FindToken(ctx, hash2)
		return nil
	})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() {
			errs[i] = inSelf(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.SelfServiceRepository) error {
				return r.UseToken(ctx, second.ID, appointment.LinkCancel, now)
			})
		})
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) || (!errors.Is(errs[0], appointment.ErrLinkInvalid) && !errors.Is(errs[1], appointment.ErrLinkInvalid)) {
		t.Errorf("concurrent uses = %v, %v; want exactly one, the other ErrLinkInvalid", errs[0], errs[1])
	}
}
