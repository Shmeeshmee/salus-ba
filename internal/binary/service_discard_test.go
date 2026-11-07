package binary_test

import (
	"cmp"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
)

// A withdrawal (or a submit's hidden items) leaves some files no longer listed by the holder: the discard job
// deletes each of them, row then object, audited with the job as agent, and keeps the files still listed.
func TestDiscard_DeletesEveryFileTheHolderNoLongerListsAndAuditsEach(t *testing.T) {
	f := newFixture(t)
	kept := f.seed(t, pdf(), binary.StatusClean)
	pending := f.seed(t, jpegBytes(t), binary.StatusQuarantined)
	rejected := f.seed(t, pngBytes(t), binary.StatusRejected)
	f.holder.attached = slices.DeleteFunc(f.holder.attached, func(a binary.Attachment) bool { return a.BinaryID != kept.ID })

	n, err := f.svc.Discard(t.Context(), binary.DiscardArgs{ResponseID: responseA, OrganizationID: clinicA})

	if err != nil || n != 2 {
		t.Fatalf("Discard() = %d, %v; want 2", n, err)
	}
	var left []string
	for id := range f.repo.files {
		left = append(left, id)
	}
	if !reflect.DeepEqual(left, []string{kept.ID}) {
		t.Errorf("files left %v, want only %s", left, kept.ID)
	}
	var want []auditevent.AuditEvent
	for _, b := range []binary.Binary{pending, rejected} {
		want = append(want, jobEvents("binary_discard", auditevent.CodeDelete, b.ID)...)
	}
	if !sameEvents(f.audit.events, want) {
		t.Errorf("audited %+v,\nwant %+v", f.audit.events, want)
	}
	for _, b := range []binary.Binary{pending, rejected} {
		if _, err := f.store.GetRange(t.Context(), b.ID, 0, 0); !errors.Is(err, binary.ErrObjectNotFound) {
			t.Errorf("the discarded file's object: %v, want ErrObjectNotFound", err)
		}
	}
	if _, err := f.store.GetRange(t.Context(), kept.ID, 0, 0); err != nil {
		t.Errorf("the kept file's object: %v", err)
	}

	f.audit.events = nil
	if n, err := f.svc.Discard(t.Context(), binary.DiscardArgs{ResponseID: responseA, OrganizationID: clinicA}); err != nil || n != 0 || len(f.audit.events) != 0 {
		t.Errorf("repeated: Discard() = %d, %v with %d events; want nothing done", n, err, len(f.audit.events))
	}
}

// The job's scope is the response and its clinic: a response without files (a cancellation took them with it)
// or another clinic's scope does nothing, and asks the holder nothing.
func TestDiscard_DoesNothingWithoutFilesInItsScope(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	f.holder.attached = nil
	for name, args := range map[string]binary.DiscardArgs{
		"another response":       {ResponseID: unknownID, OrganizationID: clinicA},
		"another clinic's scope": {ResponseID: responseA, OrganizationID: patientB},
	} {
		if n, err := f.svc.Discard(t.Context(), args); err != nil || n != 0 {
			t.Errorf("%s: Discard() = %d, %v; want 0", name, n, err)
		}
	}
	if _, ok := f.repo.files[b.ID]; !ok || len(f.audit.events) != 0 {
		t.Errorf("the file was discarded or %d events written", len(f.audit.events))
	}
}

func TestDiscard_FailsAndKeepsEveryFileWhenTheDatabaseFails(t *testing.T) {
	f := newFixture(t)
	b := f.seed(t, pdf(), binary.StatusClean)
	f.holder.attached, f.repo.err = nil, errBackend

	if _, err := f.svc.Discard(t.Context(), binary.DiscardArgs{ResponseID: responseA, OrganizationID: clinicA}); !errors.Is(err, errBackend) {
		t.Fatalf("Discard() = %v, want the backend's error", err)
	}
	if _, err := f.store.GetRange(t.Context(), b.ID, 0, 0); err != nil {
		t.Errorf("the object is gone: %v", err)
	}
}

// sameEvents compares two lists of events regardless of their order.
func sameEvents(a, b []auditevent.AuditEvent) bool {
	key := func(e auditevent.AuditEvent) string { return e.EntityReference + e.Subtype.Code }
	a, b = slices.Clone(a), slices.Clone(b)
	slices.SortFunc(a, func(x, y auditevent.AuditEvent) int { return cmp.Compare(key(x), key(y)) })
	slices.SortFunc(b, func(x, y auditevent.AuditEvent) int { return cmp.Compare(key(x), key(y)) })
	return reflect.DeepEqual(a, b)
}
