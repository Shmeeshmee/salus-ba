package communication_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

func TestOpen_PatientAndPractitionerOpenTheOneConversationOfTheirCare(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c, created, err := f.svc.Open(t.Context(), me, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1})
	if err != nil || !created {
		t.Fatalf("patient opens: %v, created %v", err, created)
	}
	if c.PatientID != patientA || c.OrganizationID != clinicA || c.PractitionerRoleID != roleA1 || c.CreatedBy != me.AccountID {
		t.Errorf("conversation = %+v", c)
	}
	if len(c.Participants) != 2 || c.Participants[0].Role != communication.RolePatient || c.Participants[1].PractitionerRoleID != roleA1 {
		t.Errorf("participants = %+v", c.Participants)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionCreate || e.EntityReference != "Patient/"+patientA || e.Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("audit = %+v", e)
	}

	again, created, err := f.svc.Open(t.Context(), pract, communication.Open{PatientID: patientA})
	if err != nil || created || again.ID != c.ID {
		t.Fatalf("practitioner opens the same: %v, created %v, id %s want %s", err, created, again.ID, c.ID)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionRead || e.Purpose.Code != auditevent.CodeTREAT {
		t.Errorf("audit of the existing one = %+v", e)
	}

	linked, created, err := f.svc.Open(t.Context(), me, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1, EncounterID: encA})
	if err != nil || !created || linked.ID == c.ID || linked.EncounterID != encA {
		t.Errorf("a conversation about the encounter: %v, created %v, %+v", err, created, linked)
	}
}

func TestOpen_RefusesWithoutConsentAppointmentOrRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want   error
		in     communication.Open
		name   string
		caller appointment.Caller
		revoke bool
	}{
		{communication.ErrForbidden, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA2}, "a patient with another practitioner", me, false},
		{communication.ErrForbidden, communication.Open{OrganizationID: clinicB, PractitionerRoleID: roleA1}, "a patient with a clinic they never visited", me, false},
		{communication.ErrConsentRequired, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1}, "a patient after their revocation", me, true},
		{communication.ErrNotFound, communication.Open{PatientID: patientA}, "a practitioner without an appointment with the patient", pract2, false},
		{communication.ErrNotFound, communication.Open{PatientID: patientB}, "a practitioner and a patient of no appointment", pract, false},
		{communication.ErrConsentRequired, communication.Open{PatientID: patientA}, "a practitioner after the revocation", pract, true},
		{communication.ErrForbidden, communication.Open{PatientID: patientA}, "an admin", admin, false},
		{communication.ErrNotFound, communication.Open{PatientID: patientA, EncounterID: encOther}, "someone else's encounter", pract, false},
		{communication.ErrNotFound, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1, EncounterID: unknownID}, "an unknown encounter", me, false},
		{communication.ErrInvalid, communication.Open{OrganizationID: clinicA, PractitionerRoleID: "role"}, "a role that is no id", me, false},
		{communication.ErrInvalid, communication.Open{}, "no patient", pract, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			if tt.revoke {
				f.revoke()
			}
			if _, _, err := f.svc.Open(t.Context(), tt.caller, tt.in); !errors.Is(err, tt.want) {
				t.Errorf("Open = %v, want %v", err, tt.want)
			}
			if len(f.repo.convs) != 0 {
				t.Error("a conversation was stored")
			}
		})
	}
}

func TestSend_NumbersEachMessagePublishesItAndMarksItReadForTheSender(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	first, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "  Synthetic placeholder question  ")
	if err != nil || !created {
		t.Fatalf("Send = %v, created %v", err, created)
	}
	if first.Seq != 1 || first.Body != "Synthetic placeholder question" || first.SenderRole != communication.RolePatient ||
		first.SenderAccountID != me.AccountID || !first.SentAt.Equal(start) || first.VersionID != 1 {
		t.Errorf("first = %+v", first)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionCreate || e.EntityReference != "Communication/"+first.ID {
		t.Errorf("audit = %+v", e)
	}
	second := f.send(t, pract, c.ID, "Synthetic placeholder answer")
	if second.Seq != 2 || second.SenderRole != communication.RolePractitioner {
		t.Errorf("second = %+v", second)
	}
	frames := f.pub.published()
	if len(frames) != 2 || frames[0].Type != communication.MessageType || !strings.Contains(string(frames[1].Payload), `"seq":2`) {
		t.Errorf("published %v", frames)
	}
	got, err := f.svc.Get(t.Context(), pract, c.ID)
	if err != nil || got.LastSeq != 2 || got.Unread != 0 {
		t.Errorf("the practitioner's view: %v, %+v", err, got)
	}
	got, err = f.svc.Get(t.Context(), me, c.ID)
	if err != nil || got.Unread != 1 {
		t.Errorf("the patient's view: %v, unread %d, want 1", err, got.Unread)
	}
}

// The client retries with the same id (docs/plans/chat.md "Idempotent send"): the first message comes back, no
// second one is stored, published or audited. Another sender's same client id is their own message.
func TestSend_ARepeatOfTheClientsIDReturnsTheFirstMessageOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	first, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder")
	if err != nil {
		t.Fatal(err)
	}
	events := f.audit.count()
	again, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder, typed again")
	if err != nil || created || again.ID != first.ID || again.Body != first.Body {
		t.Fatalf("repeat = %v, created %v, %+v", err, created, again)
	}
	if len(f.pub.published()) != 1 || len(f.repo.msgs[c.ID]) != 1 {
		t.Errorf("a repeat was published or stored")
	}
	if e := f.audit.last(t); f.audit.count() != events+1 || e.Action != auditevent.ActionRead || e.EntityReference != "Communication/"+first.ID {
		t.Errorf("the repeat's read is not audited: %+v", e)
	}
	theirs, created, err := f.svc.Send(t.Context(), pract, c.ID, clientID1, "Synthetic placeholder")
	if err != nil || !created || theirs.Seq != 2 {
		t.Errorf("another sender's same id: %v, created %v, seq %d", err, created, theirs.Seq)
	}
}

// The order by time is the order by number (ADR 0012): a clock that goes back (another api process, a correction)
// does not put a message before the previous one.
func TestSend_TheTimeNeverGoesBackWithinAConversation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	first := f.send(t, me, c.ID, "Synthetic placeholder")
	f.clock.set(start.Add(-time.Minute))
	second := f.send(t, me, c.ID, "Synthetic placeholder")
	if second.SentAt.Before(first.SentAt) || second.Seq != 2 {
		t.Errorf("second at %v (seq %d), first at %v", second.SentAt, second.Seq, first.SentAt)
	}
}

func TestSend_ConcurrentSendsGetDistinctNumbers(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { f.send(t, me, c.ID, "Synthetic placeholder") })
	}
	wg.Wait()
	seen := map[int64]bool{}
	for _, m := range f.repo.msgs[c.ID] {
		seen[m.Seq] = true
	}
	if len(seen) != 20 {
		t.Errorf("%d distinct numbers, want 20", len(seen))
	}
}

func TestSend_RefusesWhoMayNotWriteAndWhatTheContractRefuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want   error
		name   string
		body   string
		client string
		caller appointment.Caller
		revoke bool
	}{
		{communication.ErrInvalid, "an empty body", "   ", clientID1, me, false},
		{communication.ErrInvalid, "a body over the limit", strings.Repeat("a", communication.MaxBody+1), clientID1, me, false},
		{communication.ErrInvalid, "a client id that is no UUID", "Synthetic placeholder", "x", me, false},
		{communication.ErrForbidden, "a practitioner who does not take part", "Synthetic placeholder", clientID1, pract2, false},
		{communication.ErrForbidden, "an admin", "Synthetic placeholder", clientID1, admin, false},
		{communication.ErrNotFound, "another patient", "Synthetic placeholder", clientID1, other, false},
		{communication.ErrConsentRequired, "the patient after the revocation", "Synthetic placeholder", clientID1, me, true},
		{communication.ErrNotFound, "the practitioner after the revocation", "Synthetic placeholder", clientID1, pract, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			c := f.open(t)
			if tt.revoke {
				f.revoke()
			}
			if _, _, err := f.svc.Send(t.Context(), tt.caller, c.ID, tt.client, tt.body); !errors.Is(err, tt.want) {
				t.Errorf("Send = %v, want %v", err, tt.want)
			}
			if len(f.repo.msgs[c.ID]) != 0 || len(f.pub.published()) != 0 {
				t.Error("a refused message was stored or published")
			}
		})
	}
}

func TestSend_AFailedPublishKeepsTheMessage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.pub.err = errBackend
	if _, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder"); err != nil || !created {
		t.Fatalf("Send = %v, created %v; the message is the record, the channel only transport", err, created)
	}
}

func TestSend_AFailedWriteRollsBackAndIsNotPublished(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.repo.failNext = errBackend
	_, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder")
	if !errors.Is(err, errBackend) {
		t.Fatalf("Send = %v, want the backend's error", err)
	}
	if len(f.pub.published()) != 0 || f.repo.convs[c.ID].LastSeq != 0 {
		t.Error("a failed send was published or counted")
	}
}

func TestMessages_PagesNewestFirstWithTheCursorOfTheOlderPage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for range 5 {
		f.send(t, me, c.ID, "Synthetic placeholder")
	}
	page, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 2)
	if err != nil || len(page.Messages) != 2 || page.Messages[0].Seq != 5 || page.Messages[1].Seq != 4 || page.Next == nil {
		t.Fatalf("first page = %v, %+v", err, page)
	}
	if e := f.audit.last(t); e.EntityName != "communication" || e.EntityCount != 2 || e.EntityReference != "Patient/"+patientA {
		t.Errorf("audit = %+v", e)
	}
	page, err = f.svc.Messages(t.Context(), pract, c.ID, page.Next, 3)
	if err != nil || len(page.Messages) != 3 || page.Messages[0].Seq != 3 || page.Next != nil {
		t.Errorf("last page = %v, %+v", err, page)
	}
	if _, err := f.svc.Messages(t.Context(), pract, c.ID, nil, communication.MaxPageSize+1); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("a limit over the maximum: %v", err)
	}
}

// A participant added later reads from their history_from_seq (SAL-28's choice of point).
func TestMessages_AParticipantAddedLaterReadsFromTheirPoint(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for range 4 {
		f.send(t, me, c.ID, "Synthetic placeholder")
	}
	f.repo.addPractitioner(c.ID, roleA2, 2)
	page, err := f.svc.Messages(t.Context(), pract2, c.ID, nil, 0)
	if err != nil || len(page.Messages) != 2 || page.Messages[1].Seq != 3 {
		t.Errorf("Messages = %v, %+v", err, page)
	}
	if _, err := f.svc.Revisions(t.Context(), pract2, c.ID, f.repo.msgs[c.ID][0].ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("revisions of a message before their point: %v", err)
	}
	got, err := f.svc.After(t.Context(), pract2, c.ID, communication.Cursor{})
	if err != nil || len(got) != 2 {
		t.Errorf("After = %v, %d messages", err, len(got))
	}
	if sub, err := f.svc.Subscribe(t.Context(), pract2, c.ID, ""); err != nil || sub.HistoryFromSeq != 2 {
		t.Errorf("Subscribe = %v, %+v", err, sub)
	}
}

// After a revocation the clinic reads nothing of the chat; the patient keeps their own record (GDPR Art. 15).
func TestMessages_AfterARevocationOnlyThePatientReads(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic placeholder")
	f.revoke()
	if page, err := f.svc.Messages(t.Context(), me, c.ID, nil, 0); err != nil || len(page.Messages) != 1 {
		t.Errorf("the patient: %v, %+v", err, page)
	}
	if _, err := f.svc.Messages(t.Context(), pract, c.ID, nil, 0); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("the practitioner: %v, want not found, as the clinic's row-level security answers", err)
	}
	if _, err := f.svc.After(t.Context(), pract, c.ID, communication.Cursor{}); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("the practitioner's socket: %v", err)
	}
	events := f.audit.count()
	if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder"); !errors.Is(err, communication.ErrConsentRequired) {
		t.Errorf("the patient writes: %v", err)
	}
	if f.audit.count() != events+1 || f.audit.last(t).Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("the refusal is not audited: %+v", f.audit.last(t))
	}
}

func TestEdit_OnlyTheSenderEditsTheCurrentVersionAndTheEarlierBodyIsKept(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	m := f.send(t, me, c.ID, "Synthetic placeholder")
	f.clock.set(start.Add(time.Minute))
	if _, err := f.svc.Edit(t.Context(), pract, c.ID, m.ID, "Synthetic placeholder, edited", 1); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("someone else edits: %v", err)
	}
	edited, err := f.svc.Edit(t.Context(), me, c.ID, m.ID, "Synthetic placeholder, edited", 1)
	if err != nil || edited.VersionID != 2 || edited.Body != "Synthetic placeholder, edited" || !edited.EditedAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("Edit = %v, %+v", err, edited)
	}
	if _, err := f.svc.Edit(t.Context(), me, c.ID, m.ID, "Synthetic placeholder, stale", 1); !errors.Is(err, communication.ErrVersionConflict) {
		t.Errorf("a stale version: %v", err)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionUpdate || e.EntityReference != "Communication/"+m.ID {
		t.Errorf("audit = %+v", e)
	}
	frames := f.pub.published()
	if len(frames) != 2 || frames[1].Type != communication.MessageEditedType {
		t.Errorf("published %v", frames)
	}
	revs, err := f.svc.Revisions(t.Context(), pract, c.ID, m.ID)
	if err != nil || len(revs) != 1 || revs[0].Body != "Synthetic placeholder" || revs[0].VersionID != 1 {
		t.Errorf("Revisions = %v, %+v", err, revs)
	}
	if _, err := f.svc.Edit(t.Context(), me, c.ID, unknownID, "x", 1); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("an unknown message: %v", err)
	}
}

func TestMarkRead_NeverGoesBackOrPastTheLastMessage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	for range 3 {
		f.send(t, pract, c.ID, "Synthetic placeholder")
	}
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.svc.Get(t.Context(), me, c.ID); got.Unread != 1 {
		t.Errorf("unread = %d, want 1", got.Unread)
	}
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 4); !errors.Is(err, communication.ErrInvalid) {
		t.Errorf("past the last: %v", err)
	}
}

func TestList_GivesEachCallerTheirConversationsWithUnreadCounts(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic placeholder")
	mine, next, err := f.svc.List(t.Context(), me, nil)
	if err != nil || len(mine) != 1 || mine[0].ID != c.ID || mine[0].Unread != 1 || next != nil {
		t.Errorf("patient: %v, %+v", err, mine)
	}
	theirs, _, err := f.svc.List(t.Context(), pract, nil)
	if err != nil || len(theirs) != 1 || theirs[0].Unread != 0 {
		t.Errorf("practitioner: %v, %+v", err, theirs)
	}
	if e := f.audit.last(t); e.Subtype.Code != auditevent.CodeSearchType || e.EntityCount != 1 {
		t.Errorf("audit = %+v", e)
	}
	none, _, err := f.svc.List(t.Context(), pract2, nil)
	if err != nil || len(none) != 0 {
		t.Errorf("another practitioner: %v, %+v", err, none)
	}
	if _, _, err := f.svc.List(t.Context(), admin, nil); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("admin: %v", err)
	}
}

func TestSubscribe_StartsAfterTheClientsCursorOrTheLatestMessage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	first := f.send(t, me, c.ID, "Synthetic placeholder")
	f.send(t, me, c.ID, "Synthetic placeholder")
	sub, err := f.svc.Subscribe(t.Context(), pract, c.ID, "")
	if err != nil || sub.From.Seq != 2 || sub.HistoryFromSeq != 0 {
		t.Errorf("without a cursor: %v, %+v", err, sub)
	}
	if e := f.audit.last(t); e.Action != auditevent.ActionExecute || e.Subtype.Code != auditevent.CodeOperation {
		t.Errorf("audit = %+v", e)
	}
	sub, err = f.svc.Subscribe(t.Context(), pract, c.ID, first.Cursor().String())
	if err != nil || sub.From.Seq != 1 {
		t.Errorf("with the first message's cursor: %v, %+v", err, sub)
	}
	got, err := f.svc.After(t.Context(), pract, c.ID, sub.From)
	if err != nil || len(got) != 1 || got[0].Seq != 2 {
		t.Errorf("After = %v, %+v", err, got)
	}
	if _, err := f.svc.Subscribe(t.Context(), pract, c.ID, "not-a-cursor"); !errors.Is(err, communication.ErrInvalidCursor) {
		t.Errorf("a bad cursor: %v", err)
	}
	if _, err := f.svc.Subscribe(t.Context(), pract2, c.ID, ""); !errors.Is(err, communication.ErrForbidden) {
		t.Errorf("a practitioner who does not take part: %v", err)
	}
	if _, err := f.svc.Get(t.Context(), me, "not-an-id"); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("an id that is no UUID: %v", err)
	}
}

func TestCursor_RoundTripsAndRefusesWhatTheAPIDidNotGive(t *testing.T) {
	t.Parallel()
	c := communication.Cursor{SentAt: start.Add(123456 * time.Microsecond), Seq: 42}
	got, err := communication.ParseCursor(c.String())
	if err != nil || !got.SentAt.Equal(c.SentAt) || got.Seq != 42 {
		t.Errorf("round trip = %+v, %v", got, err)
	}
	if start, err := communication.ParseCursor(communication.Cursor{}.String()); err != nil || start != (communication.Cursor{}) {
		t.Errorf("the start's cursor = %+v, %v", start, err)
	}
	empty := communication.Conversation{}
	if empty.Cursor() != (communication.Cursor{}) {
		t.Errorf("an empty conversation's cursor = %+v, want the start", empty.Cursor())
	}
	for _, bad := range []string{"", "!!", "MTIz", "YS5i", "LTEuMQ"} { // "123", "a.b", "-1.1"
		if _, err := communication.ParseCursor(bad); !errors.Is(err, communication.ErrInvalidCursor) {
			t.Errorf("ParseCursor(%q) = %v", bad, err)
		}
	}
}

// The scheduler's daily job keeps PartitionsAhead months of partitions from this month (ADR 0012).
func TestEnsurePartitions_AddsTheMonthsAheadFromTheClocksMonth(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.clock.set(time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC))
	if err := f.svc.EnsurePartitions(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.partitions; len(got) != 1 || got[0] != "2026-12-31/3" {
		t.Errorf("partitions asked = %v, want from 2026-12-31 for 3 months", got)
	}
	f.repo.failNext = errBackend
	if err := f.svc.EnsurePartitions(t.Context()); !errors.Is(err, errBackend) {
		t.Errorf("a failure: %v", err)
	}
}

// The repeat is looked for within IdempotencyWindow of the conversation's last message (ADR 0012): the lookup reads
// the partitions of that day only. A client that repeats an id a day later sends a new message.
func TestSend_LooksForARepeatWithinTheIdempotencyWindow(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	first, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder")
	if err != nil {
		t.Fatal(err)
	}
	f.clock.set(start.Add(communication.IdempotencyWindow - time.Minute))
	if again, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder"); err != nil || created || again.ID != first.ID {
		t.Errorf("a repeat within the window: %v, created %v", err, created)
	}
	if got := f.repo.since; !got.Equal(start.Add(-communication.IdempotencyWindow)) {
		t.Errorf("looked back from %v, want a day before the last message", got)
	}
	f.send(t, pract, c.ID, "Synthetic placeholder") // the last message is now this one
	f.clock.set(start.Add(communication.IdempotencyWindow + time.Hour))
	f.send(t, pract, c.ID, "Synthetic placeholder")
	if _, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder"); err != nil || !created {
		t.Errorf("a repeat a day later: %v, created %v, want a new message", err, created)
	}
}
