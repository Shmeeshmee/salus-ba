package communication_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// A doctor-only side thread (docs/plans/chat.md, SAL-28): opened by a practitioner of the conversation, with its
// active practitioners; kept in the record like any conversation; the patient never sees it on any path (acceptance).
func TestOpenThread_ADoctorOnlyThreadThePatientNeverSees(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	f.add(t, conv.ID, roleA2, clinicA)
	f.add(t, conv.ID, roleB1, clinicB) // pending: not in the thread
	thread, err := f.svc.OpenThread(t.Context(), pract2, conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !thread.StaffOnly || thread.ParentID != conv.ID || thread.PatientID != patientA || thread.OrganizationID != clinicA ||
		thread.PractitionerRoleID != roleA1 || thread.CreatedBy != pract2.AccountID || len(thread.Participants) != 2 {
		t.Errorf("thread = %+v", thread)
	}
	for _, p := range thread.Participants {
		if p.Role != communication.RolePractitioner || p.Status != communication.StatusActive || (p.PractitionerRoleID != roleA1 && p.PractitionerRoleID != roleA2) {
			t.Errorf("thread participant = %+v", p)
		}
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionCreate || e.EntityName != "conversation" || e.EntityReference != "Patient/"+patientA {
		t.Errorf("audit = %+v", e)
	}
	m := f.send(t, pract, thread.ID, "Synthetic placeholder between doctors")
	if !m.StaffOnly || m.Seq != 1 {
		t.Errorf("the thread's message = %+v", m)
	}
	threads, err := f.svc.Threads(t.Context(), pract, conv.ID)
	if err != nil || len(threads) != 1 || threads[0].ID != thread.ID || threads[0].Unread != 0 {
		t.Errorf("Threads = %+v, %v", threads, err)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionExecute || e.EntityName != "conversation" || e.EntityCount != 1 {
		t.Errorf("the list's audit = %+v", e)
	}
	if threads, _ := f.svc.Threads(t.Context(), pract2, conv.ID); len(threads) != 1 || threads[0].Unread != 1 {
		t.Errorf("the colleague's threads = %+v", threads)
	}

	// The patient.
	if _, err := f.svc.Threads(t.Context(), me, conv.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the patient's Threads = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.OpenThread(t.Context(), me, conv.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the patient's OpenThread = %v, want ErrForbidden", err)
	}
	for _, c := range []communication.Caller{me, other} {
		checkNothing(t, f, c, thread.ID)
	}
	if list, _, _ := f.svc.List(t.Context(), me, nil); len(list) != 1 || list[0].ID != conv.ID {
		t.Errorf("the patient's list = %+v", list)
	}
	page, err := f.svc.Messages(t.Context(), me, conv.ID, nil, 0)
	for _, m := range page.Messages {
		if m.ConversationID != conv.ID || m.StaffOnly {
			t.Errorf("the patient reads %+v (%v)", m, err)
		}
	}
	if list, _, _ := f.svc.List(t.Context(), pract, nil); len(list) != 2 {
		t.Errorf("the practitioner's list = %+v, want the conversation and its thread", list)
	}
}

// A granted guest opens a side thread too; a doctor added after a thread was opened is not in it, and the admin and a
// practitioner who takes no part open none.
func TestOpenThread_AGrantedGuestOpensOneAndOthersDoNot(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	f.invite(t, conv.ID)
	thread, err := f.svc.OpenThread(t.Context(), guest, conv.ID)
	if err != nil || len(thread.Participants) != 2 || thread.PractitionerRoleID != roleA1 {
		t.Fatalf("the guest's thread = %+v, %v", thread, err)
	}
	f.send(t, guest, thread.ID, "Synthetic placeholder between doctors")
	if got := f.lastMessage(t, pract, thread.ID); got.SenderRole != communication.RolePractitioner {
		t.Errorf("the treating practitioner reads %+v", got)
	}
	f.add(t, conv.ID, roleA2, clinicA)
	if threads, err := f.svc.Threads(t.Context(), pract2, conv.ID); err != nil || len(threads) != 0 {
		t.Errorf("Threads of the doctor added later = %+v, %v; want none", threads, err)
	}
	if _, err := f.svc.Get(t.Context(), pract2, thread.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the thread to the doctor added later: %v, want ErrForbidden", err)
	}
	if _, err := f.svc.OpenThread(t.Context(), admin, conv.ID); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("the admin's OpenThread = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.OpenThread(t.Context(), pract, thread.ID); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("a thread of a thread = %v, want ErrInvalid", err)
	}
	f.revoke()
	if _, err := f.svc.Get(t.Context(), guest, thread.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("the guest's thread after the care consent's revocation: %v, want ErrNotFound", err)
	}
}
