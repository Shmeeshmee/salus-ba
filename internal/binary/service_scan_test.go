package binary_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clamd"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// eicar assembles the EICAR anti-malware test string, so the repository never holds it as a file.
func eicar() []byte {
	parts := []string{`X5O!P%@AP[4\PZX54(P^)7CC)7}`, `$EICAR-STANDARD-ANTIVIRUS`, `-TEST-FILE!$H+H*`}
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

// eicarPDF is a generated PDF whose one page's content stream is the EICAR string, uncompressed: the
// upload pipeline takes it as a document, and clamd finds the signature in it.
func eicarPDF() []byte {
	stream := eicar()
	return []byte("%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj\n" +
		"4 0 obj << /Length " + strconv.Itoa(len(stream)) + " >> stream\n" + string(stream) + "\nendstream endobj\n" +
		"trailer << /Root 1 0 R >>\n%%EOF\n")
}

// scanner is the compose clamd, as pkg/clamd's tests reach it; the tests fail without it.
func scanner(t *testing.T) binary.Scanner {
	t.Helper()
	addr := os.Getenv("TEST_CLAMD_ADDR")
	if addr == "" {
		t.Fatal("TEST_CLAMD_ADDR is not set; run the tests with `make test`, which points it at the compose clamd")
	}
	return clamd.New(addr)
}

// scanning is the fixture's service with sc as its scanner.
func (f *fixture) scanning(sc binary.Scanner) *binary.Service {
	p := f.svc.Ports
	p.Scanner = sc
	return binary.NewService(p)
}

// jobEvents is the pair of events of a job on the file id of response A: the job as agent, for operations.
func jobEvents(agent, interaction, id string) []auditevent.AuditEvent {
	var out []auditevent.AuditEvent
	for _, entity := range []string{"Binary/" + id, "QuestionnaireResponse/" + responseA} {
		e := event(binary.Caller{}, interaction, auditevent.CodeHOPERAT, entity, auditevent.OutcomeSuccess)
		e.AgentName = agent
		if interaction == auditevent.CodeUpdate {
			e.Action = auditevent.ActionUpdate
		}
		out = append(out, e)
	}
	return out
}

func scanArgs(b binary.Binary) binary.ScanArgs {
	return binary.ScanArgs{BinaryID: b.ID, ResponseID: b.ResponseID, OrganizationID: b.OrganizationID}
}

func TestScan_MarksACleanFileCleanAndItsAttachmentNoLongerPending(t *testing.T) {
	f := newFixture(t)
	body := mp4(2*envelope.SegmentSize + 77)
	b := f.seed(t, body, binary.StatusQuarantined)

	got, err := f.scanning(scanner(t)).Scan(t.Context(), scanArgs(b))

	if err != nil || got != (binary.ScanResult{Status: binary.StatusClean}) {
		t.Fatalf("Scan() = %+v, %v; want clean", got, err)
	}
	after := f.repo.files[b.ID]
	if after.Status != binary.StatusClean || !after.ScannedAt.Equal(now) || !bytes.Equal(after.WrappedKey, b.WrappedKey) {
		t.Errorf("the file is %s scanned at %v, want clean at %v with its key", after.Status, after.ScannedAt, now)
	}
	if len(f.holder.attached) != 1 || f.holder.attached[0].Pending || !reflect.DeepEqual(f.holder.cleared, []string{b.ID}) {
		t.Errorf("attachments %+v, cleared %v; want the file no longer pending", f.holder.attached, f.holder.cleared)
	}
	if want := jobEvents("binary_scan", auditevent.CodeUpdate, b.ID); !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
	if !bytes.Equal(stored(t, f, after), body) {
		t.Error("the clean file's object changed")
	}
}

func TestScan_RejectsAPDFCarryingTheEICARStringAndDeletesItsKeyAndObject(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, eicarPDF(), binary.StatusQuarantined)
	if b.ContentType != "application/pdf" {
		t.Fatalf("the fixture uploaded as %s, want a PDF", b.ContentType)
	}

	got, err := f.scanning(scanner(t)).Scan(t.Context(), scanArgs(b))

	// clamd 1.5.4 names it Eicar-Signature inside the PDF (Eicar-Test-Signature for the bare string).
	if err != nil || got.Status != binary.StatusRejected || !strings.HasPrefix(got.Signature, "Eicar") {
		t.Fatalf("Scan() = %+v, %v; want rejected with an EICAR signature", got, err)
	}
	after := f.repo.files[b.ID]
	if after.Status != binary.StatusRejected || !after.ScannedAt.Equal(now) || after.WrappedKey != nil {
		t.Errorf("the file is %s scanned at %v with a key of %d bytes, want rejected at %v without one", after.Status, after.ScannedAt, len(after.WrappedKey), now)
	}
	if len(f.holder.attached) != 0 || !reflect.DeepEqual(f.holder.detached, []string{b.ID}) {
		t.Errorf("attachments %+v, detached %v; want the file detached", f.holder.attached, f.holder.detached)
	}
	if want := jobEvents("binary_scan", auditevent.CodeDelete, b.ID); !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
	if _, err := f.store.GetRange(t.Context(), b.ID, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
		t.Errorf("GetRange() of the rejected file = %v, want ErrObjectNotFound", err)
	}
}

// clamd unreachable (stopped, or not yet started): the job fails and River retries it; the file stays
// quarantined, so nobody downloads it, and nothing is audited, since nothing changed.
func TestScan_LeavesTheFileQuarantinedWhenClamdIsUnreachable(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusQuarantined)

	_, err := f.scanning(clamd.New("127.0.0.1:1")).Scan(t.Context(), scanArgs(b))

	if !errors.Is(err, clamd.ErrUnreachable) {
		t.Fatalf("Scan() = %v, want ErrUnreachable", err)
	}
	if after := f.repo.files[b.ID]; after.Status != binary.StatusQuarantined || !after.ScannedAt.IsZero() {
		t.Errorf("the file is %s, want quarantined", after.Status)
	}
	if len(f.audit.events) != 0 || len(f.holder.cleared) != 0 || len(f.holder.detached) != 0 || !f.holder.attached[0].Pending {
		t.Errorf("audited %v, cleared %v, detached %v; want nothing changed", f.audit.events, f.holder.cleared, f.holder.detached)
	}
}

func TestScan_RepeatedOrOfAGoneFileChangesNothing(t *testing.T) {
	f := newFixture(t)
	clean := f.seed(t, pdf(), binary.StatusQuarantined)
	svc := f.scanning(scanner(t))
	if _, err := svc.Scan(t.Context(), scanArgs(clean)); err != nil {
		t.Fatal(err)
	}
	f.audit.events, f.holder.cleared = nil, nil
	gone := clean
	gone.ID = unknownID
	elsewhere := scanArgs(clean)
	elsewhere.OrganizationID = patientB // another clinic's job scope does not see the file

	for name, args := range map[string]binary.ScanArgs{
		"repeated": scanArgs(clean), "a gone file": scanArgs(gone), "another clinic's scope": elsewhere,
	} {
		got, err := svc.Scan(t.Context(), args)
		if err != nil || got != (binary.ScanResult{}) {
			t.Errorf("%s: Scan() = %+v, %v; want nothing done", name, got, err)
		}
	}
	if after := f.repo.files[clean.ID]; after.Status != binary.StatusClean || len(f.audit.events) != 0 || len(f.holder.cleared) != 0 {
		t.Errorf("the file is %s, audited %v, cleared %v; want nothing changed", after.Status, f.audit.events, f.holder.cleared)
	}
}

// A failure in the commit (here the holder's) rolls the result back: the file stays quarantined, the job is
// retried, and the object is kept.
func TestScan_KeepsTheFileQuarantinedWhenTheResultCannotBeCommitted(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, eicarPDF(), binary.StatusQuarantined)
	f.repo.err = nil
	svc := f.scanning(scanner(t))
	svc.Holder = failingDetach{f.holder}

	if _, err := svc.Scan(t.Context(), scanArgs(b)); !errors.Is(err, errBackend) {
		t.Fatalf("Scan() = %v, want the holder's error", err)
	}
	if after := f.repo.files[b.ID]; after.Status != binary.StatusQuarantined || len(f.audit.events) != 0 {
		t.Errorf("the file is %s with %d events, want quarantined and none", after.Status, len(f.audit.events))
	}
	if _, err := f.store.GetRange(t.Context(), b.ID, 0, 0); err != nil {
		t.Errorf("the object is gone: %v", err)
	}
}

// failingDetach is a holder whose Detach fails.
type failingDetach struct{ *fakeHolder }

func (failingDetach) Detach(_ context.Context, _, _ string) error { return errBackend }
