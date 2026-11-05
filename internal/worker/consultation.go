package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// windowCloseWorker ends a consultation when its join window has closed (consultation_window_close,
// enqueued with the session), or snoozes while the room is occupied before the hard cap. It logs the
// session's id and the snooze, nothing else: never the room or an identity.
type windowCloseWorker struct {
	river.WorkerDefaults[videosession.WindowCloseArgs]
	sessions *videosession.Service
	log      *slog.Logger
}

func (w *windowCloseWorker) Work(ctx context.Context, job *river.Job[videosession.WindowCloseArgs]) error {
	snooze, err := w.sessions.CloseWindow(ctx, job.Args.SessionID, job.Args.OrganizationID)
	if err != nil {
		return err
	}
	if snooze > 0 {
		w.log.InfoContext(ctx, "consultation room still occupied", "session_id", job.Args.SessionID, "snooze_seconds", snooze.Seconds())
		return river.JobSnooze(snooze)
	}
	w.log.InfoContext(ctx, "consultation window closed", "session_id", job.Args.SessionID)
	return nil
}

// accessEndWorker ends the patient's access to the consultation of a cancelled appointment
// (consultation_access_end, enqueued in the cancellation's transaction): it removes the patient from the room
// and ends the session. It logs the appointment's id, nothing else.
type accessEndWorker struct {
	river.WorkerDefaults[appointment.AccessEndArgs]
	sessions *videosession.Service
	log      *slog.Logger
}

func (w *accessEndWorker) Work(ctx context.Context, job *river.Job[appointment.AccessEndArgs]) error {
	if err := w.sessions.EndAccess(ctx, job.Args.AppointmentID, job.Args.OrganizationID); err != nil {
		return err
	}
	w.log.InfoContext(ctx, "consultation access ended", "appointment_id", job.Args.AppointmentID)
	return nil
}

// newSessions builds the videosession service on pool (salus_app) at clk for CloseWindow: the sessions, the
// video provider, and the encounter, which records the appointment's outcome; and for EndAccess and
// EndPatientAccess, which read the appointment through Lookup.
func newSessions(pool *pgxpool.Pool, provider videosession.Provider, clk clock.Clock) *videosession.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	// Only Lookup and RecordConsultationOutcome are asked of the appointment service.
	appointments := appointment.NewService(appointment.Ports{Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	encounters := encounter.NewService(encounter.Ports{
		Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Appointments: appointments,
	})
	return videosession.NewService(videosession.Ports{
		Repo: videosession.NewPostgresRepository(pool), Provider: provider, Audit: audit, Tx: tx, Clock: clk, Encounters: encounters,
		Appointments: appointments,
	})
}
