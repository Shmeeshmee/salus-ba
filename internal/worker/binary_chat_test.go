package worker_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// chatFile is a file of a conversation of the booking's patient, quarantined (SAL-30).
type chatFile struct{ id, conversation string }

// seedChatFile creates, as the owner, a conversation of the booking's patient with the clinic's practitioner role, its
// patient participant and a quarantined file of it holding body, of type contentType, encrypted and stored as an
// upload stores it.
func seedChatFile(t *testing.T, db logins, b booking, files worker.Files, body []byte, contentType string) chatFile {
	t.Helper()
	ctx := t.Context()
	var f chatFile
	var part string
	if err := db.owner.QueryRow(ctx, `WITH c AS (INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, created_by)
			SELECT $1, p.id, r.id, p.account_id FROM patient p, practitioner_role r WHERE p.id = $2 AND r.organization_id = $1 RETURNING id, patient_id, created_by)
		INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by) SELECT id, patient_id, 'patient', created_by FROM c
		RETURNING conversation_id::text, id::text`, clinicID, b.patient).Scan(&f.conversation, &part); err != nil {
		t.Fatal(err)
	}
	if err := db.owner.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&f.id); err != nil {
		t.Fatal(err)
	}
	dek, err := envelope.NewDataKey()
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := files.KEK.Wrap(dek, f.id)
	if err != nil {
		t.Fatal(err)
	}
	var sealed bytes.Buffer
	w, err := envelope.NewWriter(&sealed, dek, f.id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil || w.Close() != nil {
		t.Fatal("sealing the fixture failed")
	}
	t.Cleanup(func() { _ = files.Store.Delete(context.Background(), f.id) })
	if err := files.Store.Put(ctx, f.id, bytes.NewReader(sealed.Bytes()), int64(sealed.Len())); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(ctx, `INSERT INTO "binary" (id, organization_id, patient_id, conversation_id, conversation_participant_id,
			content_type, size, title, sha1, kek_id, wrapped_key, segment_size)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, digest($9::bytea, 'sha1'), 'test-1', $10, $11)`,
		f.id, clinicID, b.patient, f.conversation, part, contentType, len(body), fileTitle, body, wrapped, envelope.SegmentSize); err != nil {
		t.Fatal(err)
	}
	return f
}

func chatScanState(t *testing.T, db logins, f chatFile) (status string, thumbnail bool) {
	t.Helper()
	if err := db.owner.QueryRow(t.Context(), `SELECT coalesce((SELECT status FROM "binary" WHERE id = $1), 'gone'),
		coalesce((SELECT thumbnail_id IS NOT NULL FROM "binary" WHERE id = $1), false)`, f.id).Scan(&status, &thumbnail); err != nil {
		t.Fatal(err)
	}
	return status, thumbnail
}

// A chat image's scan runs in its conversation's job scope and leaves it clean with its thumbnail.
func TestBinaryScan_MarksAChatImageCleanWithItsThumbnail(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewGray(image.Rect(0, 0, 640, 480))); err != nil {
		t.Fatal(err)
	}
	f := seedChatFile(t, db, b, files, img.Bytes(), "image/png")

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ConversationID: f.conversation, OrganizationID: clinicID}, scanQueue)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, thumbnail := chatScanState(t, db, f); status != "clean" || !thumbnail {
		t.Errorf("the file is %s (thumbnail %v), want clean with its thumbnail", status, thumbnail)
	}
	checkFilesLog(t, logs, b, `"msg":"binary scan done","binary_id":"`+f.id+`","result":"clean"`)
}

// refusingPuts is the store with its puts refused: a thumbnail cannot be stored.
type refusingPuts struct{ binary.Store }

func (refusingPuts) Put(context.Context, string, io.Reader, int64) error {
	return errors.New("synthetic store failure")
}

// A chat image whose thumbnail cannot be stored is clean all the same, without one; the log says the thumbnail
// failed (SAL-30 review, finding 5).
func TestBinaryScan_MarksAChatImageCleanWhenItsThumbnailFails(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewGray(image.Rect(0, 0, 640, 480))); err != nil {
		t.Fatal(err)
	}
	f := seedChatFile(t, db, b, files, img.Bytes(), "image/png")
	files.Store = refusingPuts{files.Store}

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ConversationID: f.conversation, OrganizationID: clinicID},
		&river.InsertOpts{Queue: binary.QueueScan, MaxAttempts: 1})

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, thumbnail := chatScanState(t, db, f); status != "clean" || thumbnail {
		t.Errorf("the file is %s (thumbnail %v), want clean without one", status, thumbnail)
	}
	checkFilesLog(t, logs, b, `"result":"clean","thumbnail_error":"binary: scan: synthetic store failure"`)
}

// A chat file's scan that fails on its last attempt rejects it (unscannable), so its message never shows it as
// scanning for ever; the job completes.
func TestBinaryScan_RejectsAChatFileItCannotScanOnTheLastAttempt(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	files.Scanner = clamd.New("127.0.0.1:1")
	f := seedChatFile(t, db, b, files, pdfWith([]byte("BT (Placeholder) Tj ET")), "application/pdf")

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ConversationID: f.conversation, OrganizationID: clinicID},
		&river.InsertOpts{Queue: binary.QueueScan, MaxAttempts: 1})

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, _ := chatScanState(t, db, f); status != "rejected" {
		t.Errorf("the file is %s, want rejected", status)
	}
	checkFilesLog(t, logs, b, `"msg":"binary scan done","binary_id":"`+f.id+`","result":"unscannable"`)
}

// An infected chat file (the EICAR test string in a PDF) is rejected by clamd and its object deleted: its message
// never opens it (SAL-30, acceptance).
func TestBinaryScan_RejectsAnInfectedChatFile(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	files := testFiles(t)
	f := seedChatFile(t, db, b, files, pdfWith(eicar()), "application/pdf")

	ended, logs := runFilesJob(t, db, files, binary.ScanArgs{BinaryID: f.id, ConversationID: f.conversation, OrganizationID: clinicID}, scanQueue)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("the job ended %s, want completed:\n%s", ended, logs)
	}
	if status, thumbnail := chatScanState(t, db, f); status != "rejected" || thumbnail {
		t.Errorf("the file is %s (thumbnail %v), want rejected without one", status, thumbnail)
	}
	if _, err := files.Store.GetRange(t.Context(), f.id, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Errorf("the rejected file's object: %v, want ErrObjectNotFound", err)
	}
	checkFilesLog(t, logs, b, `"result":"rejected","signature":"Eicar-Signature"`)
}
