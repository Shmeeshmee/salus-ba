package binary_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1; it is no security control (SPEC.md §3 O8)
	"errors"
	"io"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

var lowerUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// stored reads the object of b whole and opens it with b's wrapped key.
func stored(t *testing.T, f *fixture, b binary.Binary) []byte {
	t.Helper()
	rc, err := f.store.GetRange(t.Context(), b.ID, 0, envelope.CiphertextSize(b.Size)-1)
	if err != nil {
		t.Fatalf("GetRange(%s) = %v", b.ID, err)
	}
	defer func() { _ = rc.Close() }()
	dek, err := f.kek.Unwrap(b.WrappedKey, b.ID)
	if err != nil {
		t.Fatalf("Unwrap() = %v", err)
	}
	r, err := envelope.Open(rc, dek, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("the stored object does not open: %v", err)
	}
	return got
}

// gone checks that no object is stored under key.
func gone(t *testing.T, f *fixture, key string) {
	t.Helper()
	if _, err := f.store.GetRange(t.Context(), key, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Errorf("object %s: GetRange = %v, want ErrObjectNotFound", key, err)
	}
}

func TestUpload_StoresTheFileEncryptedQuarantinedAndAttachedPending(t *testing.T) {
	for name, body := range map[string][]byte{
		"image/jpeg": jpegBytes(t), "image/png": pngBytes(t), "application/pdf": pdf(),
		"video/mp4": mp4(3*envelope.SegmentSize + 17), // several segments
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			b, created, err := f.upload(t, body)
			if err != nil || !created {
				t.Fatalf("Upload() = %v, %v; want created", created, err)
			}
			sum := sha1.Sum(body) //nolint:gosec // see the import
			want := binary.Binary{
				CreatedAt: now, UpdatedAt: now, ID: b.ID, OrganizationID: clinicA, PatientID: patientA, AppointmentID: apptA,
				ResponseID: responseA, LinkID: "files", ContentType: name, Title: title, KEKID: "test-1",
				Status: binary.StatusQuarantined, SHA1: sum[:], WrappedKey: b.WrappedKey, Size: int64(len(body)),
				SegmentSize: envelope.SegmentSize,
			}
			if !reflect.DeepEqual(b, want) || !lowerUUID.MatchString(b.ID) || len(b.WrappedKey) != 60 {
				t.Errorf("Upload() = %+v,\nwant %+v with a lowercase UUID and a 60-byte wrapped key", b, want)
			}
			if got := f.repo.files[b.ID]; !reflect.DeepEqual(got, want) {
				t.Errorf("stored row %+v, want %+v", got, want)
			}
			if got := stored(t, f, b); !bytes.Equal(got, body) {
				t.Errorf("the stored object opens to %d bytes, want the %d uploaded", len(got), len(body))
			}
			attached := []binary.Attachment{{LinkID: "files", BinaryID: b.ID, ContentType: name, Title: title, Hash: sum[:], Size: int64(len(body)), Pending: true}}
			if !reflect.DeepEqual(f.holder.attached, attached) {
				t.Errorf("attached %+v, want %+v", f.holder.attached, attached)
			}
			if scans := []binary.ScanArgs{{BinaryID: b.ID, ResponseID: responseA, OrganizationID: clinicA}}; !reflect.DeepEqual(f.jobs.scans, scans) {
				t.Errorf("enqueued %+v, want %+v", f.jobs.scans, scans)
			}
			if !slices.Equal(f.repo.locked, []string{responseA}) {
				t.Errorf("locked %v, want the response once", f.repo.locked)
			}
			if want := events(me, auditevent.CodeCreate, auditevent.CodePATRQT, b.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
				t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
			}
		})
	}
}

// unread is a body the upload must not read.
type unread struct{ t *testing.T }

func (u unread) Read([]byte) (int, error) {
	u.t.Error("the body was read")
	return 0, io.EOF
}

func TestUpload_RefusesWhatTheHolderRefusesWithoutReadingTheBody(t *testing.T) {
	for _, refused := range []error{
		binary.ErrNotFound, binary.ErrForbidden, binary.ErrSubmitted, binary.ErrWithdrawn, binary.ErrPastDue,
		binary.ErrAppointmentClosed, binary.ErrConsentRequired, binary.ErrNotAttachmentItem,
	} {
		t.Run(refused.Error(), func(t *testing.T) {
			f := newFixture(t)
			f.holder.uploadRefusal = refused
			c := me
			if refused == binary.ErrForbidden {
				c = pract
			}
			_, _, err := f.svc.Upload(t.Context(), c, binary.Upload{Body: unread{t}, ResponseID: responseA, LinkID: "files", Title: title, Size: 100})
			if !errors.Is(err, refused) {
				t.Fatalf("Upload() = %v, want %v", err, refused)
			}
			purpose, entity := auditevent.CodePATRQT, "QuestionnaireResponse/"+responseA
			if c.OrganizationID != "" {
				purpose = auditevent.CodeTREAT
			}
			if refused == binary.ErrNotFound {
				entity = ""
			}
			want := []auditevent.AuditEvent{eventIn(c, c.OrganizationID, auditevent.CodeCreate, purpose, entity, auditevent.OutcomeMinorFailure)}
			if !reflect.DeepEqual(f.audit.events, want) {
				t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
			}
			if len(f.repo.files) != 0 || len(f.store.puts) != 0 || len(f.holder.attached) != 0 || len(f.jobs.scans) != 0 {
				t.Errorf("a refused upload left rows %v, puts %v, attachments %v, jobs %v", f.repo.files, f.store.puts, f.holder.attached, f.jobs.scans)
			}
		})
	}
}

// refusedEvent is the one event of an upload refused once the target is known.
func refusedEvent() []auditevent.AuditEvent {
	return []auditevent.AuditEvent{event(me, auditevent.CodeCreate, auditevent.CodePATRQT, "QuestionnaireResponse/"+responseA, auditevent.OutcomeMinorFailure)}
}

func TestUpload_RefusesByTheDeclaredLengthBeforeReadingTheBody(t *testing.T) {
	for _, c := range []struct {
		want   error
		name   string
		accept []binary.Kind
		files  []int64 // sizes already on the item ("files") of the response
		others []int64 // sizes on another item
		size   int64
		max    int
	}{
		{name: "no length", size: -1, accept: allKinds, max: 3, want: binary.ErrLengthRequired},
		{name: "past every accepted kind's limit", size: binary.MaxVideoSize + 1, accept: allKinds, max: 3, want: binary.ErrTooLarge},
		{name: "past the photo limit", size: binary.MaxPhotoSize + 1, accept: []binary.Kind{binary.KindPhoto}, max: 3, want: binary.ErrTooLarge},
		{name: "past the document limit", size: binary.MaxDocumentSize + 1, accept: []binary.Kind{binary.KindDocument, binary.KindPhoto}, max: 3, want: binary.ErrTooLarge},
		{name: "the response's bytes", size: 2, accept: allKinds, max: 3, others: []int64{binary.MaxVideoSize, binary.MaxVideoSize, binary.MaxResponseSize - 2*binary.MaxVideoSize - 1}, want: binary.ErrResponseFull},
		{name: "the item's files", size: 100, accept: allKinds, max: 2, files: []int64{10, 20}, want: binary.ErrTooManyFiles},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			f.holder.target.Accept, f.holder.target.MaxFiles = c.accept, c.max
			for i, n := range c.files {
				f.repo.files[string(rune('a'+i))] = binary.Binary{ID: string(rune('a' + i)), PatientID: patientA, OrganizationID: clinicA, ResponseID: responseA, LinkID: "files", Size: n, Status: binary.StatusClean}
			}
			for i, n := range c.others {
				f.repo.files[string(rune('m'+i))] = binary.Binary{ID: string(rune('m' + i)), PatientID: patientA, OrganizationID: clinicA, ResponseID: responseA, LinkID: "other", Size: n, Status: binary.StatusClean}
			}
			_, _, err := f.svc.Upload(t.Context(), me, binary.Upload{Body: unread{t}, ResponseID: responseA, LinkID: "files", Title: title, Size: c.size})
			if !errors.Is(err, c.want) {
				t.Fatalf("Upload() = %v, want %v", err, c.want)
			}
			if !reflect.DeepEqual(f.audit.events, refusedEvent()) {
				t.Errorf("audited %+v,\nwant %+v", f.audit.events, refusedEvent())
			}
			if len(f.store.puts) != 0 {
				t.Errorf("a refused upload was stored: %v", f.store.puts)
			}
		})
	}
}

// padded is b followed by zeros up to n bytes.
func padded(b []byte, n int) []byte { return append(slices.Clone(b), make([]byte, n-len(b))...) }

func TestUpload_RefusesByTheBytes(t *testing.T) {
	hevc := mp4Head(5000, "hvc1")
	hevc = append(hevc, make([]byte, 5000)...)
	for _, c := range []struct {
		want   error
		name   string
		body   []byte
		accept []binary.Kind
		size   int64 // the declared length, len(body) when 0
		put    bool  // whether the bytes reached a put (which must then leave nothing)
	}{
		{name: "text", body: []byte("plain text, no file type at all"), accept: allKinds, want: binary.ErrUnsupportedType},
		{name: "a PDF to a photo item", body: pdf(), accept: []binary.Kind{binary.KindPhoto}, want: binary.ErrUnsupportedType},
		{name: "a JPEG to a video item", body: jpegBytes(t), accept: []binary.Kind{binary.KindVideo}, want: binary.ErrUnsupportedType},
		{name: "a JPEG past the photo limit", body: padded(jpegBytes(t), 8192), size: binary.MaxPhotoSize + 1, accept: allKinds, want: binary.ErrTooLarge},
		{name: "an HEVC video", body: hevc, accept: allKinds, want: binary.ErrUnsupportedType, put: true},
		{name: "a body shorter than its length", body: padded(pdf(), 100_000), size: 100_001, accept: allKinds, want: binary.ErrShortBody, put: true},
		{name: "a body shorter than its head", body: pdf()[:10], size: 5000, accept: allKinds, want: binary.ErrShortBody},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			f.holder.target.Accept = c.accept
			size := c.size
			if size == 0 {
				size = int64(len(c.body))
			}
			_, _, err := f.svc.Upload(t.Context(), me, binary.Upload{Body: bytes.NewReader(c.body), ResponseID: responseA, LinkID: "files", Title: title, Size: size})
			if !errors.Is(err, c.want) {
				t.Fatalf("Upload() = %v, want %v", err, c.want)
			}
			if !reflect.DeepEqual(f.audit.events, refusedEvent()) {
				t.Errorf("audited %+v,\nwant %+v", f.audit.events, refusedEvent())
			}
			if (len(f.store.puts) == 1) != c.put {
				t.Errorf("puts %v, want a put: %v", f.store.puts, c.put)
			}
			for _, key := range f.store.puts {
				gone(t, f, key)
			}
			if len(f.repo.files) != 0 || len(f.holder.attached) != 0 || len(f.jobs.scans) != 0 {
				t.Errorf("a refused upload left rows %v, attachments %v, jobs %v", f.repo.files, f.holder.attached, f.jobs.scans)
			}
		})
	}
}

func TestUpload_AnIdenticalRetryReturnsTheFirstFileAndStoresNothingNew(t *testing.T) {
	for _, max := range []int{1, 3} { // a full item still takes the retry of its file
		f := newFixture(t)
		f.holder.target.MaxFiles = max
		body := pdf()
		first, _, err := f.upload(t, body)
		if err != nil {
			t.Fatal(err)
		}
		f.audit.events = nil
		again, created, err := f.upload(t, body)
		if err != nil || created || !reflect.DeepEqual(again, first) {
			t.Fatalf("Upload() again = %+v, %v, %v; want the first file, not created", again, created, err)
		}
		if len(f.store.puts) != 2 || !slices.Equal(f.store.deleted, f.store.puts[1:]) {
			t.Errorf("puts %v, deleted %v; want the second object deleted", f.store.puts, f.store.deleted)
		}
		gone(t, f, f.store.puts[1])
		if len(f.repo.files) != 1 || len(f.holder.attached) != 1 || len(f.jobs.scans) != 1 {
			t.Errorf("rows %d, attachments %d, jobs %d; want 1 each", len(f.repo.files), len(f.holder.attached), len(f.jobs.scans))
		}
		if want := events(me, auditevent.CodeCreate, auditevent.CodePATRQT, first.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
			t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
		}
	}
}

func TestUpload_AFileOfTheSameSizeOnAFullItemIsReadThenRefused(t *testing.T) {
	f := newFixture(t)
	f.holder.target.MaxFiles = 1
	body := pdf()
	if _, _, err := f.upload(t, body); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	other := slices.Clone(body)
	other[len(other)-2] = 'X' // same size, other bytes
	if _, _, err := f.upload(t, other); !errors.Is(err, binary.ErrTooManyFiles) {
		t.Fatalf("Upload() = %v, want ErrTooManyFiles", err)
	}
	if !slices.Equal(f.store.deleted, f.store.puts[1:]) {
		t.Errorf("puts %v, deleted %v; want the refused object deleted", f.store.puts, f.store.deleted)
	}
	gone(t, f, f.store.puts[1])
	if !reflect.DeepEqual(f.audit.events, refusedEvent()) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, refusedEvent())
	}
}

func TestUpload_AFailureAfterThePutDeletesTheObject(t *testing.T) {
	t.Run("the holder's attach fails", func(t *testing.T) {
		f := newFixture(t)
		f.holder.attachErr = errBackend
		if _, _, err := f.upload(t, pdf()); !errors.Is(err, errBackend) || errors.Is(err, binary.ErrNotFound) {
			t.Fatalf("Upload() = %v, want the backend's error", err)
		}
		if len(f.store.puts) != 1 || !slices.Equal(f.store.deleted, f.store.puts) {
			t.Errorf("puts %v, deleted %v; want the object deleted", f.store.puts, f.store.deleted)
		}
		gone(t, f, f.store.puts[0])
		if len(f.repo.files) != 0 || len(f.jobs.scans) != 0 || len(f.audit.events) != 0 {
			t.Errorf("a failed upload left rows %v, jobs %v, events %v", f.repo.files, f.jobs.scans, f.audit.events)
		}
	})
	t.Run("the response was submitted meanwhile", func(t *testing.T) {
		f := newFixture(t)
		f.holder.secondRefusal = binary.ErrSubmitted
		if _, _, err := f.upload(t, pdf()); !errors.Is(err, binary.ErrSubmitted) {
			t.Fatalf("Upload() = %v, want ErrSubmitted", err)
		}
		if len(f.store.puts) != 1 || !slices.Equal(f.store.deleted, f.store.puts) {
			t.Errorf("puts %v, deleted %v; want the object deleted", f.store.puts, f.store.deleted)
		}
		gone(t, f, f.store.puts[0])
		if len(f.repo.files) != 0 || !reflect.DeepEqual(f.audit.events, refusedEvent()) {
			t.Errorf("rows %v, events %+v; want none and the refusal", f.repo.files, f.audit.events)
		}
	})
	t.Run("the store fails", func(t *testing.T) {
		f := newFixture(t)
		f.store.putErr = errBackend
		if _, _, err := f.upload(t, pdf()); !errors.Is(err, errBackend) {
			t.Fatalf("Upload() = %v, want the store's error", err)
		}
		if len(f.repo.files) != 0 || len(f.audit.events) != 0 {
			t.Errorf("a failed upload left rows %v, events %v", f.repo.files, f.audit.events)
		}
	})
}

func TestUpload_AStoppedPutLeavesTheSealingGoroutineNothingToBlockOn(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	body := mp4(2 * envelope.SegmentSize)
	_, _, err := f.svc.Upload(ctx, me, binary.Upload{Body: bytes.NewReader(body), ResponseID: responseA, LinkID: "files", Title: title, Size: int64(len(body))})
	if err == nil {
		t.Fatal("Upload() with a cancelled context = nil")
	}
}

// goneClient is the body of a client that goes away after n bytes: net/http's read error, and the request's
// context cancelled, as net/http cancels it when the connection fails.
type goneClient struct {
	r      io.Reader
	cancel context.CancelFunc
	n      int
}

func (g *goneClient) Read(p []byte) (int, error) {
	if g.n <= 0 {
		g.cancel()
		return 0, io.ErrUnexpectedEOF
	}
	if len(p) > g.n {
		p = p[:g.n]
	}
	k, err := g.r.Read(p)
	g.n -= k
	return k, err
}

func TestUpload_AClientThatGoesAwayMidBodyIsStillAudited(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	body := mp4(3 * envelope.SegmentSize)
	in := binary.Upload{Body: &goneClient{r: bytes.NewReader(body), cancel: cancel, n: 10_000}, ResponseID: responseA, LinkID: "files", Title: title, Size: int64(len(body))}
	if _, _, err := f.svc.Upload(ctx, me, in); !errors.Is(err, binary.ErrShortBody) {
		t.Fatalf("Upload() = %v, want ErrShortBody", err)
	}
	if !reflect.DeepEqual(f.audit.events, refusedEvent()) {
		t.Errorf("audited %+v,\nwant the refusal %+v", f.audit.events, refusedEvent())
	}
	if len(f.repo.files) != 0 {
		t.Errorf("an interrupted upload left rows %v", f.repo.files)
	}
}

func TestUpload_AClientThatGoesAwayOnceTheFileIsStoredStillGetsItCommitted(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.store.afterPut = cancel
	body := pdf()
	b, created, err := f.svc.Upload(ctx, me, binary.Upload{Body: bytes.NewReader(body), ResponseID: responseA, LinkID: "files", Title: title, Size: int64(len(body))})
	if err != nil || !created {
		t.Fatalf("Upload() = %v, %v; want created", created, err)
	}
	if _, ok := f.repo.files[b.ID]; !ok || len(f.store.deleted) != 0 {
		t.Errorf("row stored %v, objects deleted %v; want the file kept", ok, f.store.deleted)
	}
	if want := events(me, auditevent.CodeCreate, auditevent.CodePATRQT, b.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
}
