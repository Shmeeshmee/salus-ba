package patient_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// The chat's notifier finds the account of a conversation's patient to tell them of a new message (SAL-29): in the
// clinic's scope (it sees the patient while the care consent is active), audited as a read of the patient with the
// job as agent; nothing else of the profile leaves the domain.
func TestNoticeAccount_GivesTheJobThePatientsAccountInTheClinicsScopeAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).NoticeAccount(t.Context(), "chat_message_sent", clinicA, patientA)
	if err != nil || got != accountA {
		t.Fatalf("NoticeAccount() = %q, %v; want patient A's account", got, err)
	}
	if want := []string{"SetClinic", "ByID"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicA {
		t.Errorf("calls %v in %s, want %v in clinic A's scope", repo.calls, repo.clinic, want)
	}
	if len(audit.events) != 1 || audit.events[0].AgentName != "chat_message_sent" || audit.events[0].EntityReference != "Patient/"+patientA ||
		audit.events[0].OrganizationID != clinicA || audit.events[0].Action != auditevent.ActionRead || audit.events[0].AgentAccountID != "" {
		t.Errorf("audit %+v, want one read of patient A by the job", audit.events)
	}

	repo, audit = newFakeRepository(), &fakeAuditor{}
	if _, err := newService(repo, audit).NoticeAccount(t.Context(), "chat_message_sent", clinicA, "d0000000-0000-4000-8000-0000000000d0"); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("NoticeAccount(unknown) = %v, want ErrNotFound", err)
	}
	if len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure || audit.events[0].EntityReference != "" {
		t.Errorf("audit %+v, want the miss audited without an entity", audit.events)
	}
}

// The reminders' SMS (SAL-33) reads the patient's phone the same way: in the clinic's scope, audited as a read of the
// patient with the job as agent; "" for a patient without one.
func TestNoticePhone_GivesTheJobThePatientsPhoneInTheClinicsScopeAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	p := repo.profiles[accountA]
	p.Phone = "+38760000001"
	repo.profiles[accountA] = p
	got, err := newService(repo, audit).NoticePhone(t.Context(), "reminder_sms", clinicA, patientA)
	if err != nil || got != "+38760000001" {
		t.Fatalf("NoticePhone() = %q, %v; want patient A's phone", got, err)
	}
	if repo.clinic != clinicA || len(audit.events) != 1 || audit.events[0].AgentName != "reminder_sms" || audit.events[0].EntityReference != "Patient/"+patientA {
		t.Errorf("in %s, audit %+v, want one read of patient A by the job in clinic A", repo.clinic, audit.events)
	}
	if got, err := newService(repo, &fakeAuditor{}).NoticePhone(t.Context(), "reminder_sms", clinicA, patientB); err != nil || got != "" {
		t.Errorf("NoticePhone(no phone) = %q, %v; want none", got, err)
	}
	if _, err := newService(repo, &fakeAuditor{}).NoticePhone(t.Context(), "reminder_sms", clinicA, "d0000000-0000-4000-8000-0000000000d0"); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("NoticePhone(unknown) = %v, want ErrNotFound", err)
	}
}
