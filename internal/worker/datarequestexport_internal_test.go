package worker

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The data_request_export_build job against the database and the object store (docs/clinic-cms/WAVE5.md D2, S4): it
// reads the patient's own record as the patient, through the services they read in the app (their own row-level
// scopes), stores the archive encrypted, fulfils the request, and the patient downloads it; the reads are audited
// with the patient's account as the agent.
func TestDataRequestExportWorker_BuildsThePatientsArchiveAndFulfilsTheRequest(t *testing.T) {
	t.Parallel()
	endpoint, region, bucket := os.Getenv("TEST_S3_ENDPOINT"), os.Getenv("TEST_S3_REGION"), os.Getenv("TEST_S3_BUCKET")
	keyID, secret := os.Getenv("TEST_S3_ACCESS_KEY_ID"), os.Getenv("TEST_S3_SECRET_ACCESS_KEY")
	if endpoint == "" || region == "" || bucket == "" || keyID == "" || secret == "" {
		t.Fatal("TEST_S3_* are not set; run the tests with `make test`, which points them at the compose object store")
	}
	kek, err := envelope.NewKEK("test-1", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	store := binary.NewS3Store(&http.Client{Timeout: time.Minute}, endpoint, region, bucket, keyID, secret, 1000)
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var account, patient, request string
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role)
		VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO patient (account_id, given_name, family_name, birth_date)
		VALUES ($1, 'Synthetic', 'Patient', '1980-02-03') RETURNING id`, account).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	org, appt, enc, answer := seedExportVisit(t, owner, patient)
	if err := owner.QueryRow(ctx, `INSERT INTO data_request (account_id, patient_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $1, 'portability', now(), now() + interval '1 month') RETURNING id`, account, patient).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO data_request_export (data_request_id, account_id, requested_at) VALUES ($1, $2, now())`,
		request, account); err != nil {
		t.Fatal(err)
	}
	clk := clock.Fixed{T: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)}
	svc := newDataRequestExports(app, Files{Store: store, KEK: kek}, newRevocations(app, nil, clk), newReceipts(app, clk), clk)
	w := &dataRequestExportWorker{exports: svc, log: slog.New(slog.DiscardHandler)}
	args := datarequest.ExportArgs{RequestID: request, AccountID: account}
	if err := w.Work(ctx, &river.Job[datarequest.ExportArgs]{JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 3}, Args: args}); err != nil {
		t.Fatalf("Work() = %v", err)
	}
	var key string
	if err := owner.QueryRow(ctx, `SELECT object_key FROM data_request_export WHERE data_request_id = $1`, request).Scan(&key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), key) })

	var status, code string
	if err := owner.QueryRow(ctx, `SELECT status, answer_code FROM data_request WHERE id = $1`, request).Scan(&status, &code); err != nil ||
		status != "fulfilled" || code != "done" {
		t.Fatalf("request %s %s (%v), want fulfilled done", status, code, err)
	}
	body, size, err := svc.ExportArchive(ctx, datarequest.Patient{AccountID: account}, request)
	if err != nil {
		t.Fatalf("ExportArchive() = %v", err)
	}
	b, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || int64(len(b)) != size {
		t.Fatalf("read %d of %d bytes: %v", len(b), size, err)
	}
	names := unzip(t, b)
	rec := names["clinics/"+org+"/record.json"]
	for _, id := range []string{patient, appt, enc, answer} {
		if !strings.Contains(rec, "urn:uuid:"+id) {
			t.Errorf("the clinic's record lacks %s:\n%s", id, rec)
		}
	}
	if !strings.Contains(names["patient.json"], "urn:uuid:"+patient) || !strings.Contains(names["account.json"], account) ||
		!strings.Contains(names["manifest.txt"], "clinics/"+org+"/") {
		t.Errorf("archive %v, want the patient's profile, account and manifest", names)
	}
	var named int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_account_id = $1 AND request_id = $2`, account,
		datarequest.AuditRequestID(request)).Scan(&named); err != nil || named < 8 {
		t.Errorf("%d audit events as the patient name the request (%v), want every read of the build", named, err)
	}
	var reads int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_account_id = $1`, account).Scan(&reads); err != nil || reads < 4 {
		t.Errorf("%d audit events as the patient (%v), want the profile, consents, visits, receipts and account reads", reads, err)
	}
}

// seedExportVisit seeds, as the owner, a visit of the patient at a new clinic with their consent there, its encounter
// and a submitted questionnaire answer: the export reads them through the encounter, composition, questionnaire
// response and questionnaire services (D7).
func seedExportVisit(t *testing.T, owner *pgxpool.Pool, patient string) (org, appt, enc, answer string) {
	t.Helper()
	ctx := t.Context()
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id),
		s AS (INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) SELECT organization_id, id, 30 FROM r RETURNING id, organization_id, practitioner_role_id),
		sl AS (INSERT INTO slot (schedule_id, organization_id, period, status) SELECT id, organization_id,
			tstzrange(date_trunc('minute', now()) - interval '1 day', date_trunc('minute', now()) - interval '1 day' + interval '30 minutes'), 'busy' FROM s
			RETURNING id, organization_id, period),
		a AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at)
			SELECT $1, s.practitioner_role_id, sl.organization_id, sl.id, sl.period, now() - interval '2 days' FROM s, sl RETURNING id, organization_id)
		SELECT organization_id::text, id::text FROM a`, patient).Scan(&org, &appt); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now() - interval '3 days')`,
		patient, org); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
		SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM appointment WHERE id = $1 RETURNING id::text`, appt).Scan(&enc); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `WITH q AS (INSERT INTO questionnaire (organization_id, series_id, version, status, title, items, published_at)
			VALUES ($1, gen_random_uuid(), 1, 'active', 'Placeholder questionnaire', '[{"linkId": "q1", "text": "Placeholder question", "type": "boolean"}]', now())
			RETURNING id, series_id)
		INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
		SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', 'in-progress', lower(a.period)
		FROM appointment a, q WHERE a.id = $2 RETURNING id::text`, org, appt).Scan(&answer); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE questionnaire_response SET status = 'completed', authored_at = now() WHERE id = $1`, answer); err != nil {
		t.Fatal(err)
	}
	return org, appt, enc, answer
}

// unzip is the names and contents of the zip b.
func unzip(t *testing.T, b []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		content, _ := io.ReadAll(rc)
		_ = rc.Close()
		names[f.Name] = string(content)
	}
	return names
}

// A build attempt may run longer than River's default minute: it reads every signed PDF and uploaded file (D7).
func TestDataRequestExportWorker_HasItsOwnTimeout(t *testing.T) {
	t.Parallel()
	w := &dataRequestExportWorker{}
	if got := w.Timeout(&river.Job[datarequest.ExportArgs]{}); got != exportTimeout || got <= time.Minute {
		t.Errorf("Timeout() = %v, want %v", got, exportTimeout)
	}
}
