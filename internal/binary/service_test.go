package binary_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// seed uploads body and sets the file's status, and returns it with the audit cleared.
func (f *fixture) seed(t *testing.T, body []byte, status binary.Status) binary.Binary {
	t.Helper()
	b, _, err := f.upload(t, body)
	if err != nil {
		t.Fatal(err)
	}
	b.Status = status
	if status == binary.StatusRejected {
		b.WrappedKey = nil
	}
	f.repo.files[b.ID] = b
	f.audit.events, f.holder.uploadTargets = nil, 0
	return b
}

func TestStatus_ReturnsTheFileToWhoeverMayReadTheResponse(t *testing.T) {
	for _, c := range []struct {
		purpose string
		caller  binary.Caller
	}{{auditevent.CodePATRQT, me}, {auditevent.CodeTREAT, pract}} {
		f := newFixture(t)
		b := f.seed(t, pdf(), binary.StatusQuarantined)
		got, err := f.svc.Status(t.Context(), c.caller, b.ID)
		if err != nil || !reflect.DeepEqual(got, b) {
			t.Errorf("Status() as %s = %+v, %v; want %+v", c.purpose, got, err, b)
		}
		if f.holder.readResponseID != responseA || !reflect.DeepEqual(f.holder.readCallers, []binary.Caller{c.caller}) {
			t.Errorf("the holder was asked about %s by %v", f.holder.readResponseID, f.holder.readCallers)
		}
		if want := events(c.caller, auditevent.CodeRead, c.purpose, b.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
			t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
		}
	}
}

func TestStatus_RefusesWithoutDisclosingTheFile(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	other := binary.Caller{AccountID: accountA, PatientID: patientB}
	for name, c := range map[string]struct {
		refusal error
		id      string
		caller  binary.Caller
	}{
		"another patient's file":      {caller: other, id: b.ID},
		"a patient without a profile": {caller: binary.Caller{AccountID: accountA}, id: b.ID},
		"no such file":                {caller: me, id: unknownID},
		"not a UUID":                  {caller: me, id: "not-a-uuid"},
		"the holder refuses":          {caller: pract, id: b.ID, refusal: binary.ErrNotFound},
	} {
		f.audit.events, f.holder.readRefusal = nil, c.refusal
		if _, err := f.svc.Status(t.Context(), c.caller, c.id); !errors.Is(err, binary.ErrNotFound) {
			t.Errorf("Status() for %s = %v, want ErrNotFound", name, err)
		}
		purpose := auditevent.CodePATRQT
		if c.caller.OrganizationID != "" {
			purpose = auditevent.CodeTREAT
		}
		want := []auditevent.AuditEvent{eventIn(c.caller, c.caller.OrganizationID, auditevent.CodeRead, purpose, "", auditevent.OutcomeMinorFailure)}
		if !reflect.DeepEqual(f.audit.events, want) {
			t.Errorf("%s audited %+v,\nwant %+v", name, f.audit.events, want)
		}
	}
}

// content reads the bytes from to to of f.
func content(t *testing.T, s *binary.Service, file binary.File, from, to int64) []byte {
	t.Helper()
	rc, err := s.Content(t.Context(), file, from, to)
	if err != nil {
		t.Fatalf("Content(%d, %d) = %v", from, to, err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Content(%d, %d) read = %v", from, to, err)
	}
	return b
}

func TestDownload_ServesACleanFileByRangesAgainstTheStore(t *testing.T) {
	f := newFixture(t)
	body := mp4(3*envelope.SegmentSize + 1234)
	b := f.seed(t, body, binary.StatusClean)
	file, err := f.svc.Download(t.Context(), pract, b.ID)
	if err != nil {
		t.Fatalf("Download() = %v", err)
	}
	if !reflect.DeepEqual(file.Binary, b) {
		t.Errorf("Download() = %+v, want %+v", file.Binary, b)
	}
	if want := events(pract, auditevent.CodeRead, auditevent.CodeTREAT, b.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
	last := int64(len(body) - 1)
	for name, r := range map[string][2]int64{
		"whole": {0, last}, "the first byte": {0, 0}, "the last byte": {last, last},
		"across segments": {envelope.SegmentSize - 10, 2*envelope.SegmentSize + 10}, "a suffix": {last - 99, last},
	} {
		if got := content(t, f.svc, file, r[0], r[1]); !bytes.Equal(got, body[r[0]:r[1]+1]) {
			t.Errorf("%s: %d bytes, not the file's %d to %d", name, len(got), r[0], r[1])
		}
	}
	for _, r := range [][2]int64{{last + 1, last + 1}, {-1, 0}, {5, 4}} {
		if _, err := f.svc.Content(t.Context(), file, r[0], r[1]); !errors.Is(err, binary.ErrRangeNotSatisfiable) {
			t.Errorf("Content(%d, %d) = %v, want ErrRangeNotSatisfiable", r[0], r[1], err)
		}
	}
}

// A file of another clinic's visit the patient shared with the practitioner's clinic is read as the holder grants
// it, audited under the reader's clinic: the visit's clinic is told nothing of who it was shared with.
func TestDownload_ASharedVisitsFileIsAuditedUnderTheReadersClinic(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	b.OrganizationID = "b0000000-0000-4000-8000-00000000000b" // clinic B
	f.repo.files[b.ID], f.repo.shared = b, true
	if _, err := f.svc.Download(t.Context(), pract, b.ID); err != nil {
		t.Fatalf("Download() = %v", err)
	}
	for _, e := range f.audit.events {
		if e.OrganizationID != clinicA || e.Outcome != auditevent.OutcomeSuccess {
			t.Errorf("audited %+v, want a success under clinic A", e)
		}
	}
}

func TestDownload_RefusesAFileThatIsNotClean(t *testing.T) {
	for status, want := range map[binary.Status]error{binary.StatusQuarantined: binary.ErrScanning, binary.StatusRejected: binary.ErrNotFound} {
		f := newFixture(t)
		b := f.seed(t, pdf(), status)
		if _, err := f.svc.Download(t.Context(), me, b.ID); !errors.Is(err, want) {
			t.Errorf("Download() of a %s file = %v, want %v", status, err, want)
		}
		if want := events(me, auditevent.CodeRead, auditevent.CodePATRQT, b.ID, auditevent.OutcomeMinorFailure); !reflect.DeepEqual(f.audit.events, want) {
			t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
		}
	}
}

func TestDownload_RefusesWhatTheHolderRefuses(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	f.holder.readRefusal = binary.ErrNotFound
	if _, err := f.svc.Download(t.Context(), pract, b.ID); !errors.Is(err, binary.ErrNotFound) {
		t.Fatalf("Download() = %v, want ErrNotFound", err)
	}
	want := []auditevent.AuditEvent{eventIn(pract, pract.OrganizationID, auditevent.CodeRead, auditevent.CodeTREAT, "", auditevent.OutcomeMinorFailure)}
	if !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
}

func TestDownload_FailsOnAKeyOfAnotherKEK(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	b.KEKID = "retired-0"
	f.repo.files[b.ID] = b
	if _, err := f.svc.Download(t.Context(), me, b.ID); err == nil || errors.Is(err, binary.ErrNotFound) {
		t.Errorf("Download() with another KEK's key = %v, want a failure", err)
	}
}

func TestDelete_ThePatientDeletesTheFileAndItsObject(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusQuarantined)
	if err := f.svc.Delete(t.Context(), me, b.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}
	if _, ok := f.repo.files[b.ID]; ok || len(f.holder.attached) != 0 || !slices.Equal(f.holder.detached, []string{b.ID}) {
		t.Errorf("after Delete(): row kept %v, attached %v, detached %v", ok, f.holder.attached, f.holder.detached)
	}
	if !reflect.DeepEqual(f.holder.editCallers, []binary.Caller{me}) {
		t.Errorf("Editable asked by %v", f.holder.editCallers)
	}
	gone(t, f, b.ID)
	if want := events(me, auditevent.CodeDelete, auditevent.CodePATRQT, b.ID, auditevent.OutcomeSuccess); !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
}

func TestDelete_RefusesWhatTheHolderRefusesAndKeepsTheFile(t *testing.T) {
	for _, refused := range []error{binary.ErrSubmitted, binary.ErrWithdrawn, binary.ErrPastDue, binary.ErrForbidden, binary.ErrConsentRequired} {
		f := newFixture(t)
		b := f.seed(t, pdf(), binary.StatusClean)
		f.holder.editRefusal = refused
		c, purpose := me, auditevent.CodePATRQT
		if refused == binary.ErrForbidden {
			c, purpose = pract, auditevent.CodeTREAT
		}
		if err := f.svc.Delete(t.Context(), c, b.ID); !errors.Is(err, refused) {
			t.Errorf("Delete() = %v, want %v", err, refused)
		}
		if _, ok := f.repo.files[b.ID]; !ok || len(f.store.deleted) != 0 {
			t.Errorf("a refused deletion removed the row (%v) or the object (%v)", !ok, f.store.deleted)
		}
		if want := events(c, auditevent.CodeDelete, purpose, b.ID, auditevent.OutcomeMinorFailure); !reflect.DeepEqual(f.audit.events, want) {
			t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
		}
	}
}

func TestDelete_RefusesAFileTheCallerDoesNotSee(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	other := binary.Caller{AccountID: accountA, PatientID: patientB}
	if err := f.svc.Delete(t.Context(), other, b.ID); !errors.Is(err, binary.ErrNotFound) {
		t.Errorf("Delete() of another's file = %v, want ErrNotFound", err)
	}
	f.holder.editRefusal = binary.ErrNotFound
	if err := f.svc.Delete(t.Context(), me, b.ID); !errors.Is(err, binary.ErrNotFound) {
		t.Errorf("Delete() the holder hides = %v, want ErrNotFound", err)
	}
	want := []auditevent.AuditEvent{
		eventIn(other, "", auditevent.CodeDelete, auditevent.CodePATRQT, "", auditevent.OutcomeMinorFailure),
		eventIn(me, "", auditevent.CodeDelete, auditevent.CodePATRQT, "", auditevent.OutcomeMinorFailure),
	}
	if !reflect.DeepEqual(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
	if _, ok := f.repo.files[b.ID]; !ok {
		t.Error("a refused deletion removed the row")
	}
}

func TestBackendErrorsAreFailuresNotRefusals(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	f.repo.err = errBackend
	if _, err := f.svc.Status(t.Context(), me, b.ID); !errors.Is(err, errBackend) {
		t.Errorf("Status() = %v, want the backend's error", err)
	}
	if _, err := f.svc.Download(t.Context(), me, b.ID); !errors.Is(err, errBackend) {
		t.Errorf("Download() = %v, want the backend's error", err)
	}
	if err := f.svc.Delete(t.Context(), me, b.ID); !errors.Is(err, errBackend) {
		t.Errorf("Delete() = %v, want the backend's error", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("a failure left events %+v", f.audit.events)
	}
}

func TestJobKinds(t *testing.T) {
	if (binary.ScanArgs{}).Kind() != "binary_scan" || (binary.DiscardArgs{}).Kind() != "binary_discard" {
		t.Error("the job kinds are not binary_scan and binary_discard")
	}
	for k, want := range map[binary.Kind]int64{binary.KindPhoto: 20_000_000, binary.KindDocument: 20_000_000, binary.KindVideo: 200_000_000, "": 0} {
		if got := k.Limit(); got != want {
			t.Errorf("%q.Limit() = %d, want %d", k, got, want)
		}
	}
}

func TestDelete_RemovesTheObjectOfTheCanonicalID(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	if err := f.svc.Delete(t.Context(), me, strings.ToUpper(b.ID)); err != nil {
		t.Fatalf("Delete() by the uppercase id = %v", err)
	}
	if _, ok := f.repo.files[b.ID]; ok {
		t.Error("the row was kept")
	}
	gone(t, f, b.ID)
}

// A file its response no longer lists (detached when a hidden item's answer was dropped on submit, or by a
// withdrawal) is not read before binary_discard removes it.
func TestStatusAndDownload_RefuseAFileTheResponseNoLongerLists(t *testing.T) {
	for name, read := range map[string]func(*fixture, string) error{
		"Status":   func(f *fixture, id string) error { _, err := f.svc.Status(t.Context(), me, id); return err },
		"Download": func(f *fixture, id string) error { _, err := f.svc.Download(t.Context(), me, id); return err },
	} {
		for _, status := range []binary.Status{binary.StatusQuarantined, binary.StatusClean} {
			f := newFixture(t)
			b := f.seed(t, pdf(), status)
			f.holder.attached = nil
			if err := read(f, b.ID); !errors.Is(err, binary.ErrNotFound) {
				t.Errorf("%s() of a %s file no longer listed = %v, want ErrNotFound", name, status, err)
			}
			want := []auditevent.AuditEvent{eventIn(me, "", auditevent.CodeRead, auditevent.CodePATRQT, "", auditevent.OutcomeMinorFailure)}
			if !reflect.DeepEqual(f.audit.events, want) {
				t.Errorf("%s() audited %+v,\nwant %+v", name, f.audit.events, want)
			}
		}
	}
}

// The scan that rejects a file detaches it in the same commit; a status read whose row still showed the file
// quarantined and whose next statement already sees it detached (READ COMMITTED: a snapshot per statement) answers
// the file rejected, not 404: the app polling the status of a file being rejected never loses it (the attachments
// end-to-end test failed so, intermittently).
func TestStatus_AnswersAFileRejectedWhileItIsRead(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusQuarantined)
	f.holder.beforeAttached = func() {
		rejected := f.repo.files[b.ID]
		rejected.Status = binary.StatusRejected
		f.repo.files[b.ID] = rejected
		f.holder.attached = nil
	}
	got, err := f.svc.Status(t.Context(), me, b.ID)
	if err != nil || got.Status != binary.StatusRejected {
		t.Fatalf("Status() of a file rejected while read = %v, %v; want it rejected", got.Status, err)
	}
}

// A rejected file is detached by the scan, but the patient's app still reads its status to show M7: only its
// bytes are gone (no key, no object), so Status answers rejected and Download stays refused.
func TestStatus_StillAnswersARejectedFileTheResponseNoLongerLists(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusRejected)
	f.holder.attached = nil
	got, err := f.svc.Status(t.Context(), me, b.ID)
	if err != nil || got.Status != binary.StatusRejected {
		t.Fatalf("Status() of a rejected file = %v, %v, want it rejected", got.Status, err)
	}
	if _, err := f.svc.Download(t.Context(), me, b.ID); err == nil {
		t.Error("Download() of a rejected file succeeded")
	}
}
