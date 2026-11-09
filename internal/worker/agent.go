package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/stt"
)

// Video is the transcription agent's video server (SAL-4): the provider for its token's room, the listener that
// joins it, and the LiveKit URL the agent reaches it at (not the clients' public one).
type Video struct {
	Provider videosession.Provider
	Listener videosession.Listener
	URL      string
}

// NewAgent returns the River client cmd/agent runs, which works only the transcription queue, and the
// transcription it hands the transcription_join jobs to, on pool (salus_app). The agent hears the calls through
// video, recognises them with recognizer, translates each final segment with translator and the terminology's
// glossary (SAL-5), and puts the events on rdb's streams. cmd/agent stops the client, then
// the transcription.
func NewAgent(pool *pgxpool.Pool, rdb *goredis.Client, video Video, recognizer stt.Recognizer, translator llm.Translator, log *slog.Logger) (*river.Client[pgx.Tx], *transcript.Live, error) {
	if pool == nil || rdb == nil || video.Provider == nil || video.Listener == nil || video.URL == "" || recognizer == nil || translator == nil || log == nil {
		return nil, nil, errors.New("worker: the agent needs the pool, Redis, the video provider, its listener and URL, a recognizer, a translator and a logger")
	}
	clk := clock.System{}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil) // State only
	patients := patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk, nil, nil)
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Patients: patients, // Lookup, Notice
	})
	encounters := encounter.NewService(encounter.Ports{
		Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Appointments: appointments,
	})
	sessions := videosession.NewService(videosession.Ports{
		Repo: videosession.NewPostgresRepository(pool), Provider: video.Provider, Listener: video.Listener, Audit: audit, Tx: tx,
		Clock: clk, Encounters: encounters, Appointments: appointments, Consents: consents, PublicURL: video.URL,
	})
	live := transcript.NewLive(transcript.LivePorts{
		Calls: sessions, Appointments: appointments, Clinics: organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk),
		Accounts:  auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Publisher: transcript.NewRedisPublisher(rdb), Recognizer: recognizer, Clock: clk, Log: log,
		Translator: translator, Glossary: terminology.NewService(terminology.NewPostgresRepository(pool), nil), // Glossary only
	})
	workers := river.NewWorkers()
	addAgentWorkers(workers, live, log)
	client, err := river.NewClient(riverpgxv5.New(pool), workConfig(log, videosession.QueueTranscription, workers))
	if err != nil {
		return nil, nil, err
	}
	return client, live, nil
}

// addAgentWorkers registers the transcription queue's job; the scheduler registers it with no transcription,
// like addDefaultWorkers.
func addAgentWorkers(workers *river.Workers, live *transcript.Live, log *slog.Logger) {
	river.AddWorker(workers, &transcriptionJoinWorker{live: live, log: log})
}

// transcriptionJoinWorker puts the agent in a consultation's room (transcription_join, enqueued with the joins
// of its people): it returns once the agent is in, which transcribes in the background. A consultation the agent
// may not be in (consent withdrawn, ended, not started, not found) cancels the job: a retry would be refused too,
// and the next join asks again. It logs the session's id and the outcome, nothing else.
type transcriptionJoinWorker struct {
	river.WorkerDefaults[videosession.AgentJoinArgs]
	live *transcript.Live
	log  *slog.Logger
}

func (w *transcriptionJoinWorker) Work(ctx context.Context, job *river.Job[videosession.AgentJoinArgs]) error {
	err := w.live.Transcribe(ctx, job.Args.SessionID, job.Args.OrganizationID)
	switch {
	case err == nil:
		w.log.InfoContext(ctx, "transcription agent in the consultation", "session_id", job.Args.SessionID)
		return nil
	case errors.Is(err, videosession.ErrConsentRequired), errors.Is(err, videosession.ErrConsultationEnded),
		errors.Is(err, videosession.ErrNotStarted), errors.Is(err, videosession.ErrNotFound):
		w.log.InfoContext(ctx, "transcription agent refused", "session_id", job.Args.SessionID, "error", err.Error())
		return river.JobCancel(err)
	}
	return err
}
