package communication_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

func int64p(n int64) *int64 { return &n }

// add adds the role of the clinic to the conversation as the treating practitioner, for the whole history, and fails
// the test otherwise.
func (f *fixture) add(t *testing.T, conv, role, clinic string) communication.Participant {
	t.Helper()
	p, err := f.svc.AddParticipant(t.Context(), pract, conv, communication.AddParticipant{
		PractitionerRoleID: role, OrganizationID: clinic, Reason: communication.ReasonSecondOpinion, History: communication.HistoryFull,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// invite invites clinic B's role into the conversation and has the patient grant it.
func (f *fixture) invite(t *testing.T, conv string) communication.Participant {
	t.Helper()
	p := f.add(t, conv, roleB1, clinicB)
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	return p
}

// lastMessage is the conversation's newest message as the caller reads it.
func (f *fixture) lastMessage(t *testing.T, c appointment.Caller, conv string) communication.Message {
	t.Helper()
	page, err := f.svc.Messages(t.Context(), c, conv, nil, 1)
	if err != nil || len(page.Messages) != 1 {
		t.Fatalf("the newest message: %v, %+v", err, page)
	}
	return page.Messages[0]
}

func checkSystem(t *testing.T, m communication.Message, code string, args map[string]string) {
	t.Helper()
	if m.Kind != communication.KindSystem || m.SenderRole != communication.RoleSystem || m.SystemCode != code || m.SenderParticipantID != "" {
		t.Errorf("message = %+v, want the system message %s", m, code)
	}
	if len(m.SystemArgs) != len(args) {
		t.Errorf("%s args = %v, want %v", code, m.SystemArgs, args)
	}
	for k, v := range args {
		if m.SystemArgs[k] != v {
			t.Errorf("%s args = %v, want %v", code, m.SystemArgs, args)
		}
	}
}

// The treating practitioner adds a doctor of their clinic (docs/plans/chat.md, SAL-28): active at once, the patient
// sees a system message naming the role, the clinic and why, in the conversation's order and on its live channel; the
// add is audited; the doctor then reads the whole history and writes.
func TestAddParticipant_TheTreatingPractitionerAddsADoctorOfTheirClinicAtOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	f.send(t, me, conv.ID, "Synthetic placeholder question")
	f.send(t, pract, conv.ID, "Synthetic placeholder answer")

	p, err := f.svc.AddParticipant(t.Context(), pract, conv.ID, communication.AddParticipant{
		PractitionerRoleID: roleA2, OrganizationID: clinicA, Reason: communication.ReasonSpecialistReview, History: communication.HistoryFull,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != communication.StatusActive || p.PractitionerRoleID != roleA2 || p.OrganizationID != clinicA || p.HistoryFromSeq != 0 ||
		p.Reason != communication.ReasonSpecialistReview || p.AddedBy != pract.AccountID || !p.AddedAt.Equal(start) || p.Role != communication.RolePractitioner {
		t.Errorf("participant = %+v", p)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionCreate || e.EntityReference != "Patient/"+patientA || e.EntityName != "conversation_participant" {
		t.Errorf("audit = %+v", e)
	}
	m := f.lastMessage(t, me, conv.ID)
	checkSystem(t, m, communication.SystemParticipantAdded, map[string]string{"practitioner_role_id": roleA2, "organization_id": clinicA, "reason": "specialist_review"})
	if m.Seq != 3 {
		t.Errorf("the system message is number %d, want 3", m.Seq)
	}
	if frames := f.pub.published(); len(frames) != 3 || frames[2].Type != communication.MessageType {
		t.Errorf("published %v, want the system message third", frames)
	}
	page, err := f.svc.Messages(t.Context(), pract2, conv.ID, nil, 0)
	if err != nil || len(page.Messages) != 3 {
		t.Errorf("the added doctor reads %v, %d messages; want all 3", err, len(page.Messages))
	}
	if reply := f.send(t, pract2, conv.ID, "Synthetic placeholder opinion"); reply.Seq != 4 {
		t.Errorf("the added doctor's message = %+v", reply)
	}
}

func TestAddParticipant_FromAChosenPointTheDoctorReadsOnlyWhatFollows(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	for range 3 {
		f.send(t, me, conv.ID, "Synthetic placeholder")
	}
	in := communication.AddParticipant{PractitionerRoleID: roleA2, OrganizationID: clinicA, Reason: communication.ReasonHandover,
		History: communication.HistoryFrom, FromSeq: int64p(4)}
	if _, err := f.svc.AddParticipant(t.Context(), pract, conv.ID, in); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("from a message after the last: %v, want ErrInvalid", err)
	}
	in.FromSeq = int64p(2)
	p, err := f.svc.AddParticipant(t.Context(), pract, conv.ID, in)
	if err != nil || p.HistoryFromSeq != 2 {
		t.Fatalf("AddParticipant = %+v, %v", p, err)
	}
	page, err := f.svc.Messages(t.Context(), pract2, conv.ID, nil, 0)
	if err != nil || len(page.Messages) != 2 || page.Messages[1].Seq != 3 {
		t.Errorf("the doctor reads %v, %+v; want messages 4 (the system message) and 3", err, page.Messages)
	}
	if got, err := f.svc.Get(t.Context(), pract2, conv.ID); err != nil || got.Unread != 2 {
		t.Errorf("the doctor's unread = %d, %v; want the 2 messages they read", got.Unread, err)
	}
}

func TestAddParticipant_RefusesWhatTheRulesDoNotAllowAndStoresNothing(t *testing.T) {
	t.Parallel()
	full := communication.AddParticipant{PractitionerRoleID: roleA2, OrganizationID: clinicA, Reason: communication.ReasonSecondOpinion, History: communication.HistoryFull}
	with := func(change func(*communication.AddParticipant)) communication.AddParticipant {
		in := full
		change(&in)
		return in
	}
	tests := []struct {
		want   error
		in     communication.AddParticipant
		name   string
		caller appointment.Caller
		revoke bool
		thread bool
	}{
		{name: "a practitioner who does not treat", caller: pract2, in: full, want: communication.ErrForbidden},
		{name: "the patient", caller: me, in: full, want: communication.ErrForbidden},
		{name: "the admin", caller: admin, in: full, want: communication.ErrForbidden},
		{name: "after the revocation", caller: pract, in: full, revoke: true, want: communication.ErrNotFound},
		{name: "into a side thread", caller: pract, in: full, thread: true, want: communication.ErrForbidden},
		{name: "the treating practitioner again", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID = roleA1 }), want: communication.ErrAlreadyParticipant},
		{name: "an inactive role", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID = roleA3 }), want: communication.ErrInvalid},
		{name: "a role that is not the clinic's", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID = roleB1 }), want: communication.ErrInvalid},
		{name: "a clinic out of the directory", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID, in.OrganizationID = roleC1, clinicC }), want: communication.ErrInvalid},
		{name: "a role out of the directory", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID, in.OrganizationID = roleB2, clinicB }), want: communication.ErrInvalid},
		{name: "no reason", caller: pract, in: with(func(in *communication.AddParticipant) { in.Reason = "" }), want: communication.ErrInvalid},
		{name: "a reason not in the list", caller: pract, in: with(func(in *communication.AddParticipant) { in.Reason = "curiosity" }), want: communication.ErrInvalid},
		{name: "no history", caller: pract, in: with(func(in *communication.AddParticipant) { in.History = "" }), want: communication.ErrInvalid},
		{name: "from without a point", caller: pract, in: with(func(in *communication.AddParticipant) { in.History = communication.HistoryFrom }), want: communication.ErrInvalid},
		{name: "full with a point", caller: pract, in: with(func(in *communication.AddParticipant) { in.FromSeq = int64p(0) }), want: communication.ErrInvalid},
		{name: "a negative point", caller: pract, in: with(func(in *communication.AddParticipant) { in.History, in.FromSeq = communication.HistoryFrom, int64p(-1) }), want: communication.ErrInvalid},
		{name: "a role that is no id", caller: pract, in: with(func(in *communication.AddParticipant) { in.PractitionerRoleID = "role" }), want: communication.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			conv := f.open(t)
			id := conv.ID
			if tt.thread {
				thread, err := f.svc.OpenThread(t.Context(), pract, conv.ID)
				if err != nil {
					t.Fatal(err)
				}
				id = thread.ID
			}
			if tt.revoke {
				f.revoke()
			}
			parts := len(f.repo.parts[id])
			if _, err := f.svc.AddParticipant(t.Context(), tt.caller, id, tt.in); !errors.Is(err, tt.want) {
				t.Errorf("AddParticipant = %v, want %v", err, tt.want)
			}
			if len(f.repo.parts[id]) != parts || len(f.pub.published()) != 0 || len(f.repo.msgs[id]) != 0 {
				t.Errorf("stored or published: %d participants (want %d), %d frames", len(f.repo.parts[id]), parts, len(f.pub.published()))
			}
		})
	}
}

// A doctor of another clinic is invited (SAL-28 acceptance): pending until the patient consents, and before that sees
// nothing of the conversation, not even that it exists, on any path. The patient sees the invitation, and the grant
// opens the whole history to the doctor.
func TestAddParticipant_ADoctorOfAnotherClinicSeesNothingBeforeThePatientConsents(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	f.send(t, me, conv.ID, "Synthetic placeholder question")
	p, err := f.svc.AddParticipant(t.Context(), pract, conv.ID, communication.AddParticipant{
		PractitionerRoleID: roleB1, OrganizationID: clinicB, Reason: communication.ReasonSecondOpinion, History: communication.HistoryFull,
	})
	if err != nil || p.Status != communication.StatusPendingConsent || p.OrganizationID != clinicB {
		t.Fatalf("AddParticipant = %+v, %v; want a pending invitation", p, err)
	}
	checkSystem(t, f.lastMessage(t, me, conv.ID), communication.SystemParticipantInvited,
		map[string]string{"practitioner_role_id": roleB1, "organization_id": clinicB, "reason": "second_opinion"})
	thread, err := f.svc.OpenThread(t.Context(), pract, conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{conv.ID, thread.ID} {
		checkNothing(t, f, guest, id)
	}

	invitations, err := f.svc.Invitations(t.Context(), me)
	if err != nil || len(invitations) != 1 {
		t.Fatalf("Invitations = %+v, %v", invitations, err)
	}
	if inv := invitations[0]; inv.ID != p.ID || inv.ConversationID != conv.ID || inv.OrganizationID != clinicB || inv.PractitionerRoleID != roleB1 ||
		inv.Reason != communication.ReasonSecondOpinion || !inv.InvitedAt.Equal(start) {
		t.Errorf("invitation = %+v", inv)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionExecute || e.EntityName != "conversation_participant" || e.EntityCount != 1 || e.Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("the invitations' audit = %+v", e)
	}
	if got, _ := f.svc.Invitations(t.Context(), other); len(got) != 0 {
		t.Errorf("another patient's invitations = %+v", got)
	}

	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionUpdate || e.EntityName != "conversation_participant" || e.AgentAccountID != me.AccountID {
		t.Errorf("the decision's audit = %+v", e)
	}
	if !f.chat[conv.ID+roleB1] {
		t.Error("no chat consent recorded")
	}
	checkSystem(t, f.lastMessage(t, me, conv.ID), communication.SystemConsentGranted, map[string]string{"practitioner_role_id": roleB1})
	page, err := f.svc.Messages(t.Context(), guest, conv.ID, nil, 0)
	if err != nil || len(page.Messages) != 3 {
		t.Fatalf("the guest reads %v, %d messages; want the question and both system messages", err, len(page.Messages))
	}
	f.send(t, guest, conv.ID, "Synthetic placeholder opinion")
	if list, _, err := f.svc.List(t.Context(), guest, nil); err != nil || len(list) != 1 || list[0].ID != conv.ID {
		t.Errorf("the guest's list = %+v, %v", list, err)
	}
	if _, err := f.svc.Get(t.Context(), guest, thread.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("a side thread opened before the grant: %v, want ErrNotFound (it keeps the doctors it was opened with)", err)
	}
	if got, _ := f.svc.Invitations(t.Context(), me); len(got) != 0 {
		t.Errorf("invitations after the grant = %+v", got)
	}
}

// checkNothing checks that the caller is told nothing of the conversation on every path.
func checkNothing(t *testing.T, f *fixture, c appointment.Caller, id string) {
	t.Helper()
	if _, err := f.svc.Get(t.Context(), c, id); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("Get(%s) = %v, want ErrNotFound", id, err)
	}
	if _, err := f.svc.Messages(t.Context(), c, id, nil, 0); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("Messages(%s) = %v, want ErrNotFound", id, err)
	}
	if _, err := f.svc.Subscribe(t.Context(), c, id, ""); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("Subscribe(%s) = %v, want ErrNotFound", id, err)
	}
	if _, err := f.svc.After(t.Context(), c, id, communication.Cursor{}); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("After(%s) = %v, want ErrNotFound", id, err)
	}
	if _, _, err := f.svc.Send(t.Context(), c, id, clientID2, "Synthetic placeholder"); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("Send(%s) = %v, want ErrNotFound", id, err)
	}
	if _, err := f.svc.Threads(t.Context(), c, id); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("Threads(%s) = %v, want ErrNotFound", id, err)
	}
	if _, err := f.svc.OpenThread(t.Context(), c, id); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("OpenThread(%s) = %v, want ErrNotFound", id, err)
	}
	list, _, err := f.svc.List(t.Context(), c, nil)
	if err != nil || slices.ContainsFunc(list, func(conv communication.Conversation) bool { return conv.ID == id }) {
		t.Errorf("List = %+v, %v; want it without %s", list, err, id)
	}
}

func TestDecideInvitation_ADeclineEndsTheInvitation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	p := f.add(t, conv.ID, roleB1, clinicB)
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionDecline); err != nil {
		t.Fatal(err)
	}
	checkSystem(t, f.lastMessage(t, me, conv.ID), communication.SystemConsentDeclined, map[string]string{"practitioner_role_id": roleB1})
	if f.chat[conv.ID+roleB1] {
		t.Error("a chat consent was recorded for a decline")
	}
	checkNothing(t, f, guest, conv.ID)
	if got, _ := f.svc.Get(t.Context(), me, conv.ID); len(got.Participants) != 2 {
		t.Errorf("participants after the decline = %+v", got.Participants)
	}
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionGrant); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("a decision on a declined invitation: %v, want ErrNotFound", err)
	}
}

func TestDecideInvitation_RefusesAnyoneButThePatientAndAGrantWithoutTheCareConsent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want     error
		id       func(invited, colleague communication.Participant) string
		name     string
		decision communication.Decision
		caller   appointment.Caller
		revoke   bool
	}{
		{name: "staff", caller: pract, decision: communication.DecisionGrant, want: communication.ErrForbidden},
		{name: "the guest", caller: guest, decision: communication.DecisionGrant, want: communication.ErrForbidden},
		{name: "another patient", caller: other, decision: communication.DecisionGrant, want: communication.ErrNotFound},
		{name: "a grant after the care consent's revocation", caller: me, decision: communication.DecisionGrant, revoke: true, want: communication.ErrConsentRequired},
		{name: "an unknown decision", caller: me, decision: "maybe", want: communication.ErrInvalid},
		{name: "an unknown invitation", caller: me, decision: communication.DecisionGrant, want: communication.ErrNotFound,
			id: func(_, _ communication.Participant) string { return unknownID }},
		{name: "a colleague of the clinic, active already", caller: me, decision: communication.DecisionGrant, want: communication.ErrNotFound,
			id: func(_, colleague communication.Participant) string { return colleague.ID }},
		{name: "an invitation that is no id", caller: me, decision: communication.DecisionGrant, want: communication.ErrNotFound,
			id: func(_, _ communication.Participant) string { return "x" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			conv := f.open(t)
			invited, colleague := f.add(t, conv.ID, roleB1, clinicB), f.add(t, conv.ID, roleA2, clinicA)
			if tt.revoke {
				f.revoke()
			}
			id := invited.ID
			if tt.id != nil {
				id = tt.id(invited, colleague)
			}
			if err := f.svc.DecideInvitation(t.Context(), tt.caller, id, tt.decision); !errors.Is(err, tt.want) {
				t.Errorf("DecideInvitation = %v, want %v", err, tt.want)
			}
			if f.chat[conv.ID+roleB1] {
				t.Error("a chat consent was recorded")
			}
		})
	}
	// A decline needs no active care consent: it shares nothing.
	f := newFixture(t)
	conv := f.open(t)
	p := f.add(t, conv.ID, roleB1, clinicB)
	f.revoke()
	if err := f.svc.DecideInvitation(t.Context(), me, p.ID, communication.DecisionDecline); err != nil {
		t.Errorf("a decline after the revocation: %v", err)
	}
}

// A revocation of the chat consent, or of the care consent with the treating clinic, ends the guest's access at
// once: the socket's next check (After) is refused.
func TestRevokeInvitation_EndsTheGuestsAccessAtOnce(t *testing.T) {
	t.Parallel()
	for _, care := range []bool{false, true} {
		f := newFixture(t)
		conv := f.open(t)
		p := f.invite(t, conv.ID)
		if _, err := f.svc.Subscribe(t.Context(), guest, conv.ID, ""); err != nil {
			t.Fatal(err)
		}
		if care {
			f.revoke()
		} else {
			if err := f.svc.RevokeInvitation(t.Context(), me, p.ID); err != nil {
				t.Fatal(err)
			}
			if f.chat[conv.ID+roleB1] {
				t.Error("the chat consent is still active")
			}
			if e := f.audit.last(t); e.Action != auditevent.ActionUpdate || e.EntityName != "conversation_participant" {
				t.Errorf("the revocation's audit = %+v", e)
			}
			checkSystem(t, f.lastMessage(t, me, conv.ID), communication.SystemParticipantRemoved, map[string]string{"practitioner_role_id": roleB1})
			if err := f.svc.RevokeInvitation(t.Context(), me, p.ID); !errors.Is(err, communication.ErrNotFound) {
				t.Errorf("a second revocation: %v, want ErrNotFound", err)
			}
		}
		checkNothing(t, f, guest, conv.ID)
	}
}

func TestRevokeInvitation_RefusesWhatIsNoGrantedInvitationOfThePatient(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	granted := f.invite(t, conv.ID)
	colleague := f.add(t, conv.ID, roleA2, clinicA)
	for name, tt := range map[string]struct {
		want   error
		id     string
		caller appointment.Caller
	}{
		"staff":                     {caller: pract, id: granted.ID, want: communication.ErrForbidden},
		"another patient":           {caller: other, id: granted.ID, want: communication.ErrNotFound},
		"a colleague of the clinic": {caller: me, id: colleague.ID, want: communication.ErrNotFound},
		"the patient's own row":     {caller: me, id: conv.Participants[0].ID, want: communication.ErrNotFound},
	} {
		if err := f.svc.RevokeInvitation(t.Context(), tt.caller, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: RevokeInvitation = %v, want %v", name, err, tt.want)
		}
	}
	if !f.chat[conv.ID+roleB1] {
		t.Error("the chat consent was revoked")
	}
}

func TestRemoveParticipant_TheTreatingPractitionerRemovesADoctorWhoThenSeesNothing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	colleague := f.add(t, conv.ID, roleA2, clinicA)
	invited := f.invite(t, conv.ID)
	thread, err := f.svc.OpenThread(t.Context(), pract2, conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RemoveParticipant(t.Context(), pract, conv.ID, colleague.ID); err != nil {
		t.Fatal(err)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionUpdate || e.EntityName != "conversation_participant" || e.AgentAccountID != pract.AccountID {
		t.Errorf("audit = %+v", e)
	}
	checkSystem(t, f.lastMessage(t, me, conv.ID), communication.SystemParticipantRemoved, map[string]string{"practitioner_role_id": roleA2})
	for _, id := range []string{conv.ID, thread.ID} {
		if _, err := f.svc.Get(t.Context(), pract2, id); !errors.Is(err, communication.ErrForbidden) {
			t.Errorf("the removed doctor's Get(%s) = %v, want ErrForbidden", id, err)
		}
	}
	if err := f.svc.RemoveParticipant(t.Context(), pract, conv.ID, invited.ID); err != nil {
		t.Fatal(err)
	}
	checkNothing(t, f, guest, conv.ID)
	if got, err := f.svc.Get(t.Context(), me, conv.ID); err != nil || len(got.Participants) != 2 {
		t.Errorf("participants = %+v, %v; want the patient and the treating practitioner", got.Participants, err)
	}
}

func TestRemoveParticipant_RefusesThePatientTheTreatingPractitionerAndAnyoneElse(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	conv := f.open(t)
	colleague := f.add(t, conv.ID, roleA2, clinicA)
	for name, tt := range map[string]struct {
		want   error
		id     string
		caller appointment.Caller
	}{
		"the patient's row":               {caller: pract, id: conv.Participants[0].ID, want: communication.ErrForbidden},
		"the treating practitioner's row": {caller: pract, id: conv.Participants[1].ID, want: communication.ErrForbidden},
		"by the colleague":                {caller: pract2, id: conv.Participants[1].ID, want: communication.ErrForbidden},
		"by the patient":                  {caller: me, id: colleague.ID, want: communication.ErrForbidden},
		"an unknown participant":          {caller: pract, id: unknownID, want: communication.ErrNotFound},
	} {
		if err := f.svc.RemoveParticipant(t.Context(), tt.caller, conv.ID, tt.id); !errors.Is(err, tt.want) {
			t.Errorf("%s: RemoveParticipant = %v, want %v", name, err, tt.want)
		}
	}
	if got, _ := f.svc.Get(t.Context(), pract, conv.ID); len(got.Participants) != 3 {
		t.Errorf("participants = %+v", got.Participants)
	}
}
