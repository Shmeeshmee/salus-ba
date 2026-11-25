package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The clinic_export_build job against the database while the object store refuses the archive (M2's deferral, BG4):
// an attempt before the last returns the error for River's retry and leaves the export queued; the last marks it
// failed with the job's audit event, returns nil (River does not retry a recorded failure), and logs the id only.
func TestClinicExportWorker_MarksTheExportFailedOnItsLastAttempt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	org, admin := seedReceiptClinic(t, owner)
	var id string
	if err := owner.QueryRow(ctx, `INSERT INTO clinic_export (requested_at, expires_at, organization_id, requested_by_account_id)
		VALUES (now(), now() + interval '7 days', $1, $2) RETURNING id::text`, org, admin).Scan(&id); err != nil {
		t.Fatal(err)
	}
	kek, err := envelope.NewKEK("test-1", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	clk := clock.Fixed{T: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clk)
	appointments := appointment.NewService(appointment.Ports{Repo: appointment.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk})
	var logs bytes.Buffer
	w := &clinicExportWorker{
		exports: newClinicExports(app, Files{Store: refusedArchive{}, KEK: kek}, appointments, newReceipts(app, clk), clk),
		log:     slog.New(slog.NewJSONHandler(&logs, nil)),
	}
	job := func(attempt int) *river.Job[clinicexport.BuildArgs] {
		return &river.Job[clinicexport.BuildArgs]{
			JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: clinicexport.BuildAttempts},
			Args:   clinicexport.BuildArgs{ExportID: id, OrganizationID: org},
		}
	}
	state := func() (status string, failures int) {
		t.Helper()
		if err := owner.QueryRow(ctx, `SELECT status FROM clinic_export WHERE id = $1`, id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_name = $1 AND outcome = $2`,
			clinicexport.AgentBuild, auditevent.OutcomeSeriousFailure).Scan(&failures); err != nil {
			t.Fatal(err)
		}
		return status, failures
	}

	if err := w.Work(ctx, job(clinicexport.BuildAttempts-1)); err == nil || !strings.Contains(err.Error(), "synthetic store failure") {
		t.Fatalf("Work() before the last attempt = %v, want the store's failure for a retry", err)
	}
	if status, failures := state(); status != "queued" || failures != 0 || logs.Len() != 0 {
		t.Errorf("before the last attempt: %s, %d failures audited, log %q; want queued, none, nothing", status, failures, logs.String())
	}
	if err := w.Work(ctx, job(clinicexport.BuildAttempts)); err != nil {
		t.Fatalf("the last Work() = %v, want nil: the failure is recorded", err)
	}
	if status, failures := state(); status != "failed" || failures != 1 {
		t.Errorf("after the last attempt: %s, %d failures audited; want failed, one", status, failures)
	}
	if line := logs.String(); !strings.Contains(line, `"msg":"clinic export failed","export_id":"`+id+`"`) || strings.Contains(line, org) {
		t.Errorf("log %q, want the export's id only", line)
	}
}

// refusedArchive is the object store refusing every put: the export's archive cannot be stored.
type refusedArchive struct{ binary.Store }

func (refusedArchive) Put(context.Context, string, io.Reader, int64) error {
	return errors.New("synthetic store failure")
}
