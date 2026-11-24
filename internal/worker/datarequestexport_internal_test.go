package worker

import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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
	if err := owner.QueryRow(ctx, `INSERT INTO data_request (account_id, patient_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $1, 'portability', now(), now() + interval '1 month') RETURNING id`, account, patient).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO data_request_export (data_request_id, account_id, requested_at) VALUES ($1, $2, now())`,
		request, account); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(t.Context(), datarequest.ExportObjectKey(request)) })
	clk := clock.Fixed{T: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)}
	svc := newDataRequestExports(app, Files{Store: store, KEK: kek}, newRevocations(app, nil, clk), newReceipts(app, clk), clk)
	w := &dataRequestExportWorker{exports: svc, log: slog.New(slog.DiscardHandler)}
	args := datarequest.ExportArgs{RequestID: request, AccountID: account}
	if err := w.Work(ctx, &river.Job[datarequest.ExportArgs]{JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 3}, Args: args}); err != nil {
		t.Fatalf("Work() = %v", err)
	}

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
	zr, err := zip.NewReader(bytes.NewReader(b), size)
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
	if !strings.Contains(names["record.json"], "urn:uuid:"+patient) || !strings.Contains(names["account.json"], account) {
		t.Errorf("archive %v, want the patient's record and account", names)
	}
	var reads int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_event WHERE agent_account_id = $1`, account).Scan(&reads); err != nil || reads < 4 {
		t.Errorf("%d audit events as the patient (%v), want the profile, consents, visits, receipts and account reads", reads, err)
	}
}
