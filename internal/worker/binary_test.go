package worker_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// fileTitle is the synthetic file name, which no log line may hold.
const fileTitle = "Synthetic placeholder document.pdf"

// prefixStore is the compose object store seen through a key prefix unique to the test: the bucket is shared
// with every test running against the stack (and with the dev stack), so a test's jobs and its purger see and
// delete only the objects under its own prefix.
type prefixStore struct {
	binary.Store
	prefix string
}

func (s prefixStore) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	return s.Store.Put(ctx, s.prefix+key, r, size)
}

func (s prefixStore) GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error) {
	return s.Store.GetRange(ctx, s.prefix+key, from, to)
}

func (s prefixStore) Delete(ctx context.Context, key string) error {
	return s.Store.Delete(ctx, s.prefix+key)
}

func (s prefixStore) ListOlderThan(ctx context.Context, before time.Time, fn func(keys []string) error) error {
	return s.Store.ListOlderThan(ctx, before, func(keys []string) error {
		var own []string
		for _, k := range keys {
			if rest, ok := strings.CutPrefix(k, s.prefix); ok {
				own = append(own, rest)
			}
		}
		if len(own) == 0 {
			return nil
		}
		return fn(own)
	})
}

// testFiles is the binary jobs' dependencies against the compose object store (under a prefix of the test's own)
// and clamd, with a synthetic key-encryption key. The tests fail without the stack; they never skip.
func testFiles(t *testing.T) worker.Files {
	t.Helper()
	endpoint, region, bucket := os.Getenv("TEST_S3_ENDPOINT"), os.Getenv("TEST_S3_REGION"), os.Getenv("TEST_S3_BUCKET")
	keyID, secret, addr := os.Getenv("TEST_S3_ACCESS_KEY_ID"), os.Getenv("TEST_S3_SECRET_ACCESS_KEY"), os.Getenv("TEST_CLAMD_ADDR")
	if endpoint == "" || region == "" || bucket == "" || keyID == "" || secret == "" || addr == "" {
		t.Fatal("TEST_S3_* and TEST_CLAMD_ADDR are not set; run the tests with `make test`, which points them at the compose services")
	}
	kek, err := envelope.NewKEK("test-1", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	store := prefixStore{
		Store:  binary.NewS3Store(&http.Client{Timeout: time.Minute}, endpoint, region, bucket, keyID, secret, 1000),
		prefix: "test/" + strings.ReplaceAll(t.Name(), "/", "_") + "-" + strings.ToLower(rand.Text()) + "/",
	}
	return worker.Files{Store: store, KEK: kek, Scanner: clamd.New(addr)}
}

// eicar assembles the EICAR anti-malware test string, so the repository never holds it as a file.
func eicar() []byte {
	return []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}` + `$EICAR-STANDARD-ANTIVIRUS` + `-TEST-FILE!$H+H*`)
}

// pdfWith is a generated PDF whose one content stream is content.
func pdfWith(content []byte) []byte {
	return []byte("%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj\n" +
		"4 0 obj << /Length " + strconv.Itoa(len(content)) + " >> stream\n" + string(content) + "\nendstream endobj\n" +
		"trailer << /Root 1 0 R >>\n%%EOF\n")
}

// attachedFile is a file seeded on the in-progress response of the booking's future appointment.
type attachedFile struct {
	id, response, patient string
}

// seedFile creates, as the owner, an in-progress response to the booking's future appointment (once per
// booking) and a quarantined file of it holding body, encrypted and stored as an upload stores it, attached
// pending.
func seedFile(t *testing.T, db logins, b booking, files worker.Files, body []byte) attachedFile {
	t.Helper()
	ctx := t.Context()
	var response string
	err := db.owner.QueryRow(ctx, `SELECT id::text FROM questionnaire_response WHERE appointment_id = $1`, b.future).Scan(&response)
	if err != nil {
		version := seedQuestionnaire(t, db, b)
		err = db.owner.QueryRow(ctx, `INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id,
				practitioner_role_id, questionnaire_id, series_id, source, status, due_at)
			SELECT a.organization_id, a.id, a.patient_id, a.practitioner_role_id, q.id, q.series_id, 'added', 'in-progress', lower(a.period)
			FROM appointment a, questionnaire q WHERE a.id = $1 AND q.id = $2 RETURNING id::text`, b.future, version).Scan(&response)
	}
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err := db.owner.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	dek, err := envelope.NewDataKey()
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := files.KEK.Wrap(dek, id)
	if err != nil {
		t.Fatal(err)
	}
	var sealed bytes.Buffer
	w, err := envelope.NewWriter(&sealed, dek, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil || w.Close() != nil {
		t.Fatal("sealing the fixture failed")
	}
	t.Cleanup(func() { _ = files.Store.Delete(context.Background(), id) })
	if err := files.Store.Put(ctx, id, bytes.NewReader(sealed.Bytes()), int64(sealed.Len())); err != nil {
		t.Fatal(err)
	}
	_, err = db.owner.Exec(ctx, `WITH f AS (INSERT INTO "binary" (id, organization_id, patient_id, appointment_id, questionnaire_response_id,
				link_id, content_type, size, title, sha1, kek_id, wrapped_key, segment_size)
			SELECT $1::text::uuid, organization_id, patient_id, appointment_id, id, 'files', 'application/pdf', $3::bigint, $4::text,
				digest($5::bytea, 'sha1'), 'test-1', $6, $7 FROM questionnaire_response WHERE id = $2 RETURNING id)
		UPDATE questionnaire_response SET attachments = attachments || jsonb_build_array(jsonb_build_object(
			'linkId', 'files', 'binaryId', $1::text, 'contentType', 'application/pdf', 'size', $3::bigint, 'title', $4::text,
			'hash', '', 'pending', true))
		WHERE id = $2 AND EXISTS (SELECT 1 FROM f)`, id, response, len(body), fileTitle, body, wrapped, envelope.SegmentSize)
	if err != nil {
		t.Fatal(err)
	}
	return attachedFile{id: id, response: response, patient: b.patient}
}

// runFilesJob starts the worker on db with files, enqueues args with opts, and returns how the job's first
// attempt ended and the worker's log.
func runFilesJob(t *testing.T, db logins, files worker.Files, args river.JobArgs, opts *river.InsertOpts) (river.EventKind, string) {
	t.Helper()
	return runJobOn(t, db, liveKit(t), files, args, opts)
}

// fileState is a file's status, whether it still has its key, and its attachment as the response lists it
// ("pending", "cleared" or "none").
func fileState(t *testing.T, db logins, f attachedFile) (string, bool, string) {
	t.Helper()
	var status, attachment string
	var keyed bool
	err := db.owner.QueryRow(t.Context(), `SELECT coalesce((SELECT status FROM "binary" WHERE id = $1), 'gone'),
			coalesce((SELECT wrapped_key IS NOT NULL FROM "binary" WHERE id = $1), false),
			coalesce((SELECT CASE WHEN (a->>'pending')::boolean THEN 'pending' ELSE 'cleared' END
			          FROM questionnaire_response, jsonb_array_elements(attachments) a WHERE id = $2 AND a->>'binaryId' = $1::text), 'none')`,
		f.id, f.response).Scan(&status, &keyed, &attachment)
	if err != nil {
		t.Fatal(err)
	}
	return status, keyed, attachment
}

// checkFilesLog fails when the job's log holds the file's title, its patient's id or the e-mail, or lacks want.
func checkFilesLog(t *testing.T, logs string, b booking, want string) {
	t.Helper()
	for _, v := range []string{fileTitle, b.patient, b.email, "Ana"} {
		if strings.Contains(logs, v) {
			t.Errorf("the log holds %q:\n%s", v, logs)
		}
	}
	if !strings.Contains(logs, want) {
		t.Errorf("the log lacks %s:\n%s", want, logs)
	}
}

// auditedBy counts the audit events of the job agent on the file and on its response.
func auditedBy(t *testing.T, db logins, agent string, f attachedFile, action string) int {
	t.Helper()
	var n int
	err := db.owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name = $1 AND action = $2
		AND purpose_code = 'HOPERAT' AND entity_reference IN ('Binary/' || $3, 'QuestionnaireResponse/' || $4)`, agent, action, f.id, f.response).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

var scanQueue = &river.InsertOpts{Queue: binary.QueueScan}

func TestBinaryScan_MarksACleanFileCleanOnTheScanQueue(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	f := seedFile(t, db, b, files, pdfWith([]byte("BT (Placeholder) Tj ET")))

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ResponseID: f.response, OrganizationID: clinicID}, scanQueue)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, keyed, attachment := fileState(t, db, f); status != "clean" || !keyed || attachment != "cleared" {
		t.Errorf("the file is %s (key %v), its attachment %s; want clean with its key, no longer pending", status, keyed, attachment)
	}
	if n := auditedBy(t, db, "binary_scan", f, "U"); n != 2 {
		t.Errorf("%d update events by binary_scan, want 2 (the file and its response)", n)
	}
	checkFilesLog(t, logs, b, `"msg":"binary scan done","binary_id":"`+f.id+`","result":"clean"`)
}

func TestBinaryScan_RejectsAnEICARFileAndLogsItsSignatureWithoutThePatient(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	f := seedFile(t, db, b, files, pdfWith(eicar()))

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ResponseID: f.response, OrganizationID: clinicID}, scanQueue)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, keyed, attachment := fileState(t, db, f); status != "rejected" || keyed || attachment != "none" {
		t.Errorf("the file is %s (key %v), its attachment %s; want rejected without its key, detached", status, keyed, attachment)
	}
	if n := auditedBy(t, db, "binary_scan", f, "D"); n != 2 {
		t.Errorf("%d delete events by binary_scan, want 2", n)
	}
	if _, err := files.Store.GetRange(t.Context(), f.id, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Errorf("the rejected file's object: %v, want ErrObjectNotFound", err)
	}
	checkFilesLog(t, logs, b, `"msg":"binary scan done","binary_id":"`+f.id+`","result":"rejected","signature":"Eicar-Signature"`)
}

// clamd unreachable: the attempt fails, River retries with its backoff, and the file stays quarantined.
func TestBinaryScan_FailsAndLeavesTheFileQuarantinedWhenClamdIsUnreachable(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	files.Scanner = clamd.New("127.0.0.1:1")
	f := seedFile(t, db, b, files, pdfWith([]byte("BT (Placeholder) Tj ET")))

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ResponseID: f.response, OrganizationID: clinicID}, scanQueue)

	if ended != river.EventKindJobFailed {
		t.Fatalf("the job ended %s, want failed (retried):\n%s", ended, logs)
	}
	if status, _, attachment := fileState(t, db, f); status != "quarantined" || attachment != "pending" {
		t.Errorf("the file is %s, its attachment %s; want quarantined and pending", status, attachment)
	}
	checkFilesLog(t, logs, b, "binary_scan")
}

// After a withdrawal the response lists no file: the discard job deletes each, row then object.
func TestBinaryDiscard_DeletesTheFilesTheResponseNoLongerLists(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	kept := seedFile(t, db, b, files, pdfWith([]byte("BT (Kept) Tj ET")))
	gone := seedFile(t, db, b, files, pdfWith([]byte("BT (Gone) Tj ET")))
	if _, err := db.owner.Exec(t.Context(), `UPDATE questionnaire_response SET attachments = (SELECT jsonb_agg(a) FROM jsonb_array_elements(attachments) a
		WHERE a->>'binaryId' = $1) WHERE id = $2`, kept.id, kept.response); err != nil {
		t.Fatal(err)
	}

	ended, logs := runFilesJob(t, db, files, binary.DiscardArgs{ResponseID: gone.response, OrganizationID: clinicID}, nil)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, _, _ := fileState(t, db, gone); status != "gone" {
		t.Errorf("the discarded file is %s, want gone", status)
	}
	if status, _, _ := fileState(t, db, kept); status != "quarantined" {
		t.Errorf("the kept file is %s, want quarantined", status)
	}
	if _, err := files.Store.GetRange(t.Context(), gone.id, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Errorf("the discarded file's object: %v, want ErrObjectNotFound", err)
	}
	if n := auditedBy(t, db, "binary_discard", gone, "D"); n != 2 {
		t.Errorf("%d delete events by binary_discard, want 2", n)
	}
	checkFilesLog(t, logs, b, `"msg":"binary discard done","questionnaire_response_id":"`+gone.response+`","deleted":1`)
}

func TestNew_RefusesToStartWithoutTheFilesDependencies(t *testing.T) {
	db := newLogins(t)
	full := testFiles(t)
	for name, files := range map[string]worker.Files{
		"no store":   {KEK: full.KEK, Scanner: full.Scanner},
		"no key":     {Store: full.Store, Scanner: full.Scanner},
		"no scanner": {Store: full.Store, KEK: full.KEK},
	} {
		if _, err := worker.New(db.app, db.retention, liveKit(t), files, slog.New(slog.DiscardHandler)); err == nil {
			t.Errorf("New() with %s: error = nil", name)
		}
	}
}
