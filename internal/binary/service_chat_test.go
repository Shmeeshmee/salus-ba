package binary_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// Synthetic ids of the chat (SAL-30): conversation C of patient A in clinic A, its patient's participant row and its
// practitioner's.
const (
	convC     = "c0000000-0000-4000-8000-0000000000c1"
	convD     = "c0000000-0000-4000-8000-0000000000d1"
	partPat   = "f0000000-0000-4000-8000-0000000000f1"
	partPract = "f0000000-0000-4000-8000-0000000000f2"
	messageM  = "e0000000-0000-4000-8000-0000000000e1"
)

// chatUpload is a file for conversation C from the patient's participant.
func chatUpload(body []byte) binary.ChatUpload {
	return binary.ChatUpload{
		Body: bytes.NewReader(body), ConversationID: convC, ParticipantID: partPat, OrganizationID: clinicA, PatientID: patientA,
		Title: title, Size: int64(len(body)),
	}
}

// enterConversation is the chat's part of an upload's transaction, as communication does it: the conversation set
// (row-level security), and the calls it saw.
type enterConversation struct {
	err   error
	repo  *fakeRepository
	files []binary.Binary
}

func (e *enterConversation) commit(ctx context.Context, b binary.Binary) error {
	if e.err != nil {
		return e.err
	}
	e.repo.conversation = convC
	e.files = append(e.files, b)
	return nil
}

// The chat's upload (docs/plans/chat.md, SAL-30): the type from the bytes, a photo or a PDF only, sealed into the
// store outside any transaction, then in one transaction the chat's own checks and audit (commit), the row
// quarantined in the conversation and its scan job, which names the conversation.
func TestUploadChat_StoresAPhotoOrAPDFAndEnqueuesItsScanInTheConversation(t *testing.T) {
	t.Parallel()
	for name, body := range map[string][]byte{"png": pngBytes(t), "jpeg": jpegBytes(t), "pdf": pdf()} {
		f := newFixture(t)
		e := &enterConversation{repo: f.repo}
		got, err := f.svc.UploadChat(t.Context(), chatUpload(body), e.commit)
		if err != nil {
			t.Fatalf("%s: UploadChat() = %v", name, err)
		}
		if got.ConversationID != convC || got.ParticipantID != partPat || got.Status != binary.StatusQuarantined || got.ResponseID != "" ||
			got.Title != title || got.Size != int64(len(body)) || got.KEKID != "test-1" {
			t.Errorf("%s: UploadChat() = %+v", name, got)
		}
		if len(e.files) != 1 || e.files[0].ID != got.ID {
			t.Errorf("%s: commit saw %v, want the file once", name, e.files)
		}
		if want := []binary.ScanArgs{{BinaryID: got.ID, ConversationID: convC, OrganizationID: clinicA}}; !slices.Equal(f.jobs.scans, want) {
			t.Errorf("%s: scans = %v, want %v", name, f.jobs.scans, want)
		}
		if len(f.audit.events) != 0 || f.holder.uploadTargets != 0 || len(f.holder.attached) != 0 {
			t.Errorf("%s: binary audited %d events and asked the questionnaire holder %d times: the chat decides and audits", name,
				len(f.audit.events), f.holder.uploadTargets)
		}
		if plain := f.plaintext(t, got); !bytes.Equal(plain, body) {
			t.Errorf("%s: the stored object does not open to the upload", name)
		}
	}
}

// A refusal of the bytes (no length, past 20 MB, a video or anything but a photo or a PDF, a short body) leaves
// nothing behind and never reaches the chat's commit; a refusal of the commit leaves no row and no object.
func TestUploadChat_RefusesWhatTheChatDoesNotTakeAndLeavesNothing(t *testing.T) {
	t.Parallel()
	errChat := errors.New("the chat refused")
	short := chatUpload(pdf())
	short.Size += 10
	none := chatUpload(pdf())
	none.Size = -1
	huge := chatUpload(pdf())
	huge.Size = binary.MaxPhotoSize + 1
	for _, tt := range []struct {
		name      string
		commitErr error
		want      error
		in        binary.ChatUpload
	}{
		{name: "no length", in: none, want: binary.ErrLengthRequired},
		{name: "too large", in: huge, want: binary.ErrTooLarge},
		{name: "a video", in: chatUpload(mp4(100)), want: binary.ErrUnsupportedType},
		{name: "text", in: chatUpload([]byte("plain text, no file type")), want: binary.ErrUnsupportedType},
		{name: "a short body", in: short, want: binary.ErrShortBody},
		{name: "the chat's refusal", in: chatUpload(pdf()), commitErr: errChat, want: errChat},
	} {
		f := newFixture(t)
		e := &enterConversation{repo: f.repo, err: tt.commitErr}
		if _, err := f.svc.UploadChat(t.Context(), tt.in, e.commit); !errors.Is(err, tt.want) {
			t.Errorf("%s: UploadChat() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.repo.files) != 0 || len(f.jobs.scans) != 0 {
			t.Errorf("%s: left %d rows and %d scans", tt.name, len(f.repo.files), len(f.jobs.scans))
		}
		if tt.commitErr == nil && len(e.files) != 0 {
			t.Errorf("%s: the commit ran", tt.name)
		}
		for _, key := range f.store.puts {
			if !slices.Contains(f.store.deleted, key) {
				t.Errorf("%s: the object %s was left in the store", tt.name, key)
			}
		}
	}
}

// chatFile uploads body to conversation C as the patient's participant, with status, through the service.
func (f *fixture) chatFile(t *testing.T, body []byte, status binary.Status) binary.Binary {
	t.Helper()
	e := &enterConversation{repo: f.repo}
	b, err := f.svc.UploadChat(t.Context(), chatUpload(body), e.commit)
	if err != nil {
		t.Fatal(err)
	}
	b.Status = status
	if status != binary.StatusQuarantined {
		b.ScannedAt = now
	}
	f.repo.files[b.ID] = b
	return b
}

// inChat runs fn in a transaction scoped as communication scopes one: the clinic with conversation C set.
func (f *fixture) inChat(t *testing.T, fn func(ctx context.Context) error) error {
	t.Helper()
	return fakeTransactor{f: f}.WithTx(t.Context(), func(ctx context.Context) error {
		f.repo.clinic, f.repo.conversation = clinicA, convC
		return fn(ctx)
	})
}

// A send attaches the sender's own free files of the conversation; any other id refuses the whole send.
func TestAttachChat_AttachesExactlyTheSendersFreeFiles(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a, b := f.chatFile(t, pdf(), binary.StatusQuarantined), f.chatFile(t, pngBytes(t), binary.StatusClean)
	m := binary.MessageFiles{ConversationID: convC, ParticipantID: partPat, MessageID: messageM, SentAt: now, IDs: []string{a.ID, b.ID}}
	var got []binary.Binary
	err := f.inChat(t, func(ctx context.Context) error {
		var err error
		got, err = f.svc.AttachChat(ctx, m)
		return err
	})
	if err != nil || len(got) != 2 || got[0].MessageID != messageM || got[1].MessageID != messageM {
		t.Fatalf("AttachChat() = %+v, %v; want both attached", got, err)
	}
	for name, ids := range map[string][]string{
		"attached already": {a.ID}, "unknown": {unknownID}, "a duplicate": {f.chatFile(t, pdf(), binary.StatusClean).ID, unknownID},
	} {
		m.IDs = ids
		err := f.inChat(t, func(ctx context.Context) error {
			_, err := f.svc.AttachChat(ctx, m)
			return err
		})
		if !errors.Is(err, binary.ErrNotFound) {
			t.Errorf("%s: AttachChat() = %v, want ErrNotFound", name, err)
		}
	}
	other := f.chatFile(t, pdf(), binary.StatusClean)
	m.IDs, m.ParticipantID = []string{other.ID}, partPract
	if err := f.inChat(t, func(ctx context.Context) error { _, err := f.svc.AttachChat(ctx, m); return err }); !errors.Is(err, binary.ErrNotFound) {
		t.Errorf("another participant's file: AttachChat() = %v, want ErrNotFound", err)
	}
}

// A chat file opens for the chat once clean, from its own conversation only; its thumbnail when it has one.
func TestOpenChat_OpensACleanFileOfTheConversationOrItsThumbnail(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	body := pdf()
	clean, scanning, rejected := f.chatFile(t, body, binary.StatusClean), f.chatFile(t, pdf(), binary.StatusQuarantined), f.chatFile(t, pdf(), binary.StatusRejected)
	err := f.inChat(t, func(ctx context.Context) error {
		for _, tt := range []struct {
			want      error
			id, conv  string
			thumbnail bool
		}{
			{id: scanning.ID, conv: convC, want: binary.ErrScanning},
			{id: rejected.ID, conv: convC, want: binary.ErrNotFound},
			{id: clean.ID, conv: convD, want: binary.ErrNotFound},
			{id: clean.ID, conv: convC, thumbnail: true, want: binary.ErrNotFound}, // a PDF has none
			{id: unknownID, conv: convC, want: binary.ErrNotFound},
		} {
			b, err := f.svc.ChatFile(ctx, tt.conv, tt.id)
			if err == nil {
				_, err = f.svc.OpenChat(ctx, b, tt.thumbnail)
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("%s in %s (thumbnail %v): %v, want %v", tt.id, tt.conv, tt.thumbnail, err, tt.want)
			}
		}
		b, err := f.svc.ChatFile(ctx, convC, clean.ID)
		if err != nil {
			return err
		}
		file, err := f.svc.OpenChat(ctx, b, false)
		if err != nil {
			return err
		}
		r, err := f.svc.Content(ctx, file, 0, file.Size-1)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, body) {
			t.Errorf("Content() = %d bytes, %v; want the file", len(got), err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fakeScanner answers every scan with result or err.
type fakeScanner struct {
	err    error
	result clamd.Result
}

func (s fakeScanner) Scan(_ context.Context, r io.Reader) (clamd.Result, error) {
	_, _ = io.Copy(io.Discard, r)
	return s.result, s.err
}

func chatScanArgs(b binary.Binary) binary.ScanArgs {
	return binary.ScanArgs{BinaryID: b.ID, ConversationID: b.ConversationID, OrganizationID: b.OrganizationID}
}

// chatJobEvent is the scan job's one event on a chat file: the chat has no questionnaire response to name.
func chatJobEvent(interaction, id string) []auditevent.AuditEvent {
	e := event(binary.Caller{}, interaction, auditevent.CodeHOPERAT, "Binary/"+id, auditevent.OutcomeSuccess)
	e.AgentName = "binary_scan"
	if interaction == auditevent.CodeUpdate {
		e.Action = auditevent.ActionUpdate
	}
	return []auditevent.AuditEvent{e}
}

// The scan of a chat file runs in the conversation's job scope, never asks the questionnaire holder, and gives a
// clean image a JPEG thumbnail of at most ThumbnailEdge pixels a side, made from the clean bytes and sealed under its
// own id; a clean PDF gets none; a flagged file is rejected and its object deleted.
func TestScan_ChatFilesAreScannedInTheirConversationAndCleanImagesGetAThumbnail(t *testing.T) {
	t.Parallel()
	big := image.NewRGBA(image.Rect(0, 0, 900, 600))
	var b bytes.Buffer
	if err := jpeg.Encode(&b, big, nil); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name      string
		result    clamd.Result
		want      binary.Status
		body      []byte
		thumbnail image.Point
	}{
		{name: "a wide JPEG", body: b.Bytes(), want: binary.StatusClean, thumbnail: image.Pt(binary.ThumbnailEdge, binary.ThumbnailEdge*2/3)},
		{name: "a small PNG", body: pngBytes(t), want: binary.StatusClean, thumbnail: image.Pt(64, 64)},
		{name: "a PDF", body: pdf(), want: binary.StatusClean},
		{name: "a flagged PNG", body: pngBytes(t), result: clamd.Result{Found: true, Signature: "Synthetic.Test"}, want: binary.StatusRejected},
	} {
		f := newFixture(t)
		file := f.chatFile(t, tt.body, binary.StatusQuarantined)
		got, err := f.scanning(fakeScanner{result: tt.result}).Scan(t.Context(), chatScanArgs(file))
		if err != nil || got.Status != tt.want {
			t.Fatalf("%s: Scan() = %+v, %v; want %s", tt.name, got, err, tt.want)
		}
		stored := f.repo.files[file.ID]
		interaction := auditevent.CodeUpdate
		if tt.want == binary.StatusRejected {
			interaction = auditevent.CodeDelete
		}
		if want := chatJobEvent(interaction, file.ID); !slices.EqualFunc(f.audit.events, want, func(a, b auditevent.AuditEvent) bool { return a == b }) {
			t.Errorf("%s: events = %+v, want %+v", tt.name, f.audit.events, want)
		}
		if len(f.holder.cleared)+len(f.holder.detached) != 0 {
			t.Errorf("%s: the questionnaire holder was asked about a chat file", tt.name)
		}
		if tt.thumbnail == (image.Point{}) {
			if stored.Thumbnail.ID != "" {
				t.Errorf("%s: a thumbnail %+v, want none", tt.name, stored.Thumbnail)
			}
			continue
		}
		thumb, err := f.svc.OpenChat(f.ctxInChat(), stored, true)
		if err != nil {
			t.Fatalf("%s: OpenChat(thumbnail) = %v", tt.name, err)
		}
		img, err := jpeg.Decode(bytes.NewReader(f.contentOf(t, thumb)))
		if err != nil || img.Bounds().Size() != tt.thumbnail || thumb.ContentType != "image/jpeg" || thumb.ID == file.ID {
			t.Errorf("%s: thumbnail %v (%s, %s), %v; want a JPEG of %v under its own id", tt.name, img.Bounds().Size(), thumb.ContentType, thumb.ID, err, tt.thumbnail)
		}
	}
}

// A chat file that cannot be scanned is rejected, never left quarantined: clamd's stream limit at once, any other
// failure when the job gives up (Unscannable, its last attempt). A questionnaire file stays quarantined, as before.
func TestScan_AChatFileThatCannotBeScannedIsRejected(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	file := f.chatFile(t, pdf(), binary.StatusQuarantined)
	got, err := f.scanning(fakeScanner{err: clamd.ErrStreamLimit}).Scan(t.Context(), chatScanArgs(file))
	if err != nil || got.Status != binary.StatusRejected || f.repo.files[file.ID].Status != binary.StatusRejected {
		t.Errorf("Scan() past clamd's limit = %+v, %v; want rejected", got, err)
	}

	f = newFixture(t)
	file = f.chatFile(t, pdf(), binary.StatusQuarantined)
	if _, err := f.scanning(fakeScanner{err: errBackend}).Scan(t.Context(), chatScanArgs(file)); err == nil {
		t.Fatal("Scan() with clamd failing = nil, want the error (River retries)")
	}
	if f.repo.files[file.ID].Status != binary.StatusQuarantined {
		t.Fatal("a failed scan changed the file")
	}
	ok, err := f.svc.Unscannable(t.Context(), chatScanArgs(file))
	if err != nil || !ok || f.repo.files[file.ID].Status != binary.StatusRejected || !slices.Contains(f.store.deleted, file.ID) {
		t.Errorf("Unscannable() = %v, %v; want the file rejected and its object deleted", ok, err)
	}
	if want := chatJobEvent(auditevent.CodeDelete, file.ID); !slices.Equal(f.audit.events, want) {
		t.Errorf("events = %+v, want %+v", f.audit.events, want)
	}
	if ok, err := f.svc.Unscannable(t.Context(), chatScanArgs(file)); err != nil || ok {
		t.Errorf("Unscannable() again = %v, %v; want nothing done", ok, err)
	}

	q := newFixture(t)
	qb := q.seed(t, pdf(), binary.StatusQuarantined)
	if ok, err := q.svc.Unscannable(t.Context(), scanArgs(qb)); err != nil || ok || q.repo.files[qb.ID].Status != binary.StatusQuarantined {
		t.Errorf("Unscannable() of a questionnaire file = %v, %v; want it left quarantined", ok, err)
	}
	if _, err := q.scanning(fakeScanner{err: clamd.ErrStreamLimit}).Scan(t.Context(), scanArgs(qb)); err == nil {
		t.Error("Scan() of a questionnaire file past clamd's limit = nil, want the error, as before")
	}
}

// ctxInChat is a context in a transaction of the fixture, for calls that need one outside of inChat.
func (f *fixture) ctxInChat() context.Context {
	f.repo.clinic, f.repo.conversation = clinicA, convC
	return context.WithValue(context.Background(), txKey{}, true)
}

// contentOf is the whole plaintext of an opened file.
func (f *fixture) contentOf(t *testing.T, file binary.File) []byte {
	t.Helper()
	r, err := f.svc.Content(t.Context(), file, 0, file.Size-1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// plaintext is the stored object of b opened with its data key.
func (f *fixture) plaintext(t *testing.T, b binary.Binary) []byte {
	t.Helper()
	dek, err := f.kek.Unwrap(b.WrappedKey, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := f.store.GetRange(t.Context(), b.ID, 0, envelope.CiphertextSize(b.Size)-1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = body.Close() }()
	plain, err := io.ReadAll(must(envelope.Open(body, dek, b.ID)))
	if err != nil {
		t.Fatal(err)
	}
	return plain
}

func must(r io.Reader, err error) io.Reader {
	if err != nil {
		return errReader{err}
	}
	return r
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
