package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// revocationWorker cancels a patient's future appointments with a clinic after they revoked their consent
// there (consent_revoked, enqueued in the revocation's transaction), then removes the patient from their
// consultations there. It logs the count, nothing else.
type revocationWorker struct {
	river.WorkerDefaults[consent.RevokedArgs]
	appointments *appointment.Service
	sessions     *videosession.Service
	log          *slog.Logger
}

func (w *revocationWorker) Work(ctx context.Context, job *river.Job[consent.RevokedArgs]) error {
	n, err := w.appointments.CancelForRevokedConsent(ctx, job.Args.PatientID, job.Args.OrganizationID)
	w.log.InfoContext(ctx, "appointments cancelled after a revocation", "cancelled", n)
	if err != nil {
		return err
	}
	n, err = w.appointments.WithdrawForRevokedConsent(ctx, job.Args.PatientID, job.Args.OrganizationID)
	w.log.InfoContext(ctx, "waitlist entries withdrawn after a revocation", "withdrawn", n)
	if err != nil {
		return err
	}
	return w.sessions.EndPatientAccess(ctx, job.Args.PatientID, job.Args.OrganizationID)
}

// transcriptionRevocationWorker takes the transcription agent out of a patient's consultations with a clinic
// after they withdrew the transcription consent there (transcription_consent_revoked, enqueued in the
// withdrawal's transaction, SAL-12); the call goes on. It logs nothing about the patient.
type transcriptionRevocationWorker struct {
	river.WorkerDefaults[consent.TranscriptionRevokedArgs]
	sessions *videosession.Service
}

func (w *transcriptionRevocationWorker) Work(ctx context.Context, job *river.Job[consent.TranscriptionRevokedArgs]) error {
	return w.sessions.EndAgentAccess(ctx, job.Args.PatientID, job.Args.OrganizationID)
}

// newRevocations builds the appointment service on pool (salus_app) at clk for the default queue's appointment jobs:
// CancelForRevokedConsent and WithdrawForRevokedConsent (the consent's state, the slots to free, the cancellation
// e-mails and the statistics' refreshes), and the waitlist's HandOn and ExpireOffer (the clinic's settings, zone and
// roles, the offers' expiries and notices), enqueuing on jobs (an insert-only client).
func newRevocations(pool *pgxpool.Pool, jobs *river.Client[pgx.Tx], clk clock.Clock) *appointment.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	// Only State is asked of the consent service: it needs no jobs.
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil)
	slots := newSlots(pool, clk)
	return appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk,
		Consents: consents, Slots: slots, Jobs: appointment.NewRiverJobs(jobs),
		Stats:    clinicstats.NewRiverQueue(jobs), // a cancellation enqueues its clinic-day's refresh
		Clinics:  organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clk, nil),
		Roles:    practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Waitlist: appointment.NewPostgresWaitlistRepository(pool), Offered: slots,
		Settings:  organization.NewAppointments(organization.AppointmentsPorts{Repo: organization.NewPostgresAppointmentsRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		OfferJobs: appointment.NewRiverOfferJobs(jobs), Notices: appointment.NewRiverOfferNotices(jobs),
	})
}
