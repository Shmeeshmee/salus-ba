package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// questionnaireRequestWorker requests the default questionnaire of a booked appointment's role
// (questionnaire_default_request, enqueued in the booking's transaction). It logs the appointment's id,
// nothing else: never a title or an answer.
type questionnaireRequestWorker struct {
	river.WorkerDefaults[appointment.QuestionnaireRequestArgs]
	responses *questionnaireresponse.Service
	log       *slog.Logger
}

func (w *questionnaireRequestWorker) Work(ctx context.Context, job *river.Job[appointment.QuestionnaireRequestArgs]) error {
	if err := w.responses.RequestDefault(ctx, job.Args.AppointmentID, job.Args.OrganizationID); err != nil {
		return err
	}
	w.log.InfoContext(ctx, "questionnaire default request done", "appointment_id", job.Args.AppointmentID)
	return nil
}

// questionnaireDiscardWorker deletes the questionnaire responses of a cancelled appointment
// (questionnaire_responses_discard, enqueued in the cancellation's transaction). It logs the appointment's id,
// nothing else.
type questionnaireDiscardWorker struct {
	river.WorkerDefaults[appointment.QuestionnaireDiscardArgs]
	responses *questionnaireresponse.Service
	log       *slog.Logger
}

func (w *questionnaireDiscardWorker) Work(ctx context.Context, job *river.Job[appointment.QuestionnaireDiscardArgs]) error {
	if err := w.responses.Discard(ctx, job.Args.AppointmentID, job.Args.OrganizationID); err != nil {
		return err
	}
	w.log.InfoContext(ctx, "questionnaire responses discard done", "appointment_id", job.Args.AppointmentID)
	return nil
}

// newQuestionnaireResponses builds the questionnaireresponse service on pool (salus_app) at clk for
// RequestDefault and Discard: the appointment, read through Lookup, and the role's default questionnaire,
// through ActiveDefault.
func newQuestionnaireResponses(pool *pgxpool.Pool, clk clock.Clock) *questionnaireresponse.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	// Only Lookup is asked of the appointment service, and only ActiveDefault of the questionnaire service.
	appointments := appointment.NewService(appointment.Ports{Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	questionnaires := questionnaire.NewService(questionnaire.Ports{Repo: questionnaire.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	return questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk,
		Appointments: appointments, Questionnaires: questionnaires,
	})
}
