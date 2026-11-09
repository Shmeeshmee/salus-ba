package videosession_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

const webhookAgent = "livekit_webhook"

// joinedAt is when LiveKit says the participant joined, to the millisecond.
var joinedAt = start.Add(-3*time.Minute + 250*time.Millisecond)

// deliver hands the event to the service as a signed webhook and returns what it gives.
func deliver(t *testing.T, f *fixture, e videosession.Event) (videosession.EventKind, error) {
	t.Helper()
	f.provider.event = e
	return f.svc.HandleWebhook(t.Context(), []byte("body"), "signed")
}

func joined(identity string) videosession.Event {
	return videosession.Event{Kind: videosession.EventParticipantJoined, Room: "r_room", Identity: identity, At: joinedAt}
}

func left(identity string) videosession.Event {
	return videosession.Event{Kind: videosession.EventParticipantLeft, Room: "r_room", Identity: identity, At: joinedAt}
}

func roomFinished() videosession.Event {
	return videosession.Event{Kind: videosession.EventRoomFinished, Room: "r_room", At: joinedAt}
}

// webhookEvent is the audit event of the webhook's change to appointment A's session.
func webhookEvent() auditevent.AuditEvent {
	e := endEvent()
	e.AgentName = webhookAgent
	return e
}

// The practitioner's first join is recorded at LiveKit's time and starts the Encounter then, in the clinic's
// scope, audited; a repeat changes, starts and audits nothing.
func TestHandleWebhook_ThePractitionersJoinStartsTheEncounter(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	for range 2 {
		if kind, err := deliver(t, f, joined("p_practitioner")); err != nil || kind != videosession.EventParticipantJoined {
			t.Fatalf("HandleWebhook() = %q, %v; want the join handled", kind, err)
		}
	}
	if s := f.session(); !s.PractitionerJoinedAt.Equal(joinedAt) || !s.PatientJoinedAt.IsZero() || !s.EndedAt.IsZero() {
		t.Errorf("session %+v; want the practitioner joined at LiveKit's time", s)
	}
	want := []string{webhookAgent + " " + apptA + " " + joinedAt.Format(time.RFC3339Nano) + " in " + clinicA}
	if !slices.Equal(f.encounters.started, want) {
		t.Errorf("started %v, want %v", f.encounters.started, want)
	}
	checkEvents(t, "HandleWebhook", f.audit, webhookEvent())
	if len(f.provider.removed) != 0 || len(f.consents.asked) != 0 {
		t.Errorf("removed %v, consents asked %v; want neither for the practitioner", f.provider.removed, f.consents.asked)
	}
}

// The patient's join is recorded, audited once, and their access checked again on every join: with the
// consent active and the appointment booked they stay.
func TestHandleWebhook_ThePatientsJoinIsRecordedAndTheyStay(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	for range 2 {
		if _, err := deliver(t, f, joined("p_patient")); err != nil {
			t.Fatal(err)
		}
	}
	if s := f.session(); !s.PatientJoinedAt.Equal(joinedAt) || !s.PractitionerJoinedAt.IsZero() {
		t.Errorf("session %+v; want the patient joined at LiveKit's time", s)
	}
	if len(f.encounters.started) != 0 || len(f.provider.removed) != 0 {
		t.Errorf("started %v, removed %v; want neither", f.encounters.started, f.provider.removed)
	}
	if want := []string{patientA + " " + clinicA, patientA + " " + clinicA}; !slices.Equal(f.consents.asked, want) {
		t.Errorf("consents asked %v, want %v", f.consents.asked, want)
	}
	checkEvents(t, "HandleWebhook", f.audit, webhookEvent())
}

// A patient whose access is gone when they join (a token issued just before a revocation or a cancellation
// committed) is removed from the room, after the commit; their join is kept, and a repeat removes them again.
func TestHandleWebhook_RemovesAPatientWhoseAccessIsGone(t *testing.T) {
	for name, gone := range map[string]func(f *fixture){
		"consent revoked": func(f *fixture) { f.consents.revoked = true },
		"appointment cancelled": func(f *fixture) {
			f.change(func(a *appointment.Attendance) { a.Appointment.Status = appointment.StatusCancelled })
		},
		"appointment out of scope": func(f *fixture) { f.appts.hidden = true },
		"appointment marked by hand": func(f *fixture) {
			f.change(func(a *appointment.Attendance) { a.Appointment.Status = appointment.StatusFulfilled })
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := opened(now, start, time.Time{})
			gone(f)
			for range 2 {
				if _, err := deliver(t, f, joined("p_patient")); err != nil {
					t.Fatal(err)
				}
			}
			if want := []string{"r_room p_patient", "r_room p_patient"}; !slices.Equal(f.provider.removed, want) || f.provider.inTx {
				t.Errorf("removed %v (in a transaction: %t), want %v outside it", f.provider.removed, f.provider.inTx, want)
			}
			if s := f.session(); !s.PatientJoinedAt.Equal(joinedAt) || !s.EndedAt.IsZero() {
				t.Errorf("session %+v; want the patient's join kept and the session open", s)
			}
			checkEvents(t, "HandleWebhook", f.audit, webhookEvent())
		})
	}
}

// A failed removal fails the webhook, so LiveKit delivers it again, which removes the patient again.
func TestHandleWebhook_FailsWhenTheRemovalFails(t *testing.T) {
	f := opened(now, start, time.Time{})
	f.consents.revoked = true
	f.provider.removeErr = errBackend
	if _, err := deliver(t, f, joined("p_patient")); !errors.Is(err, errBackend) {
		t.Fatalf("HandleWebhook() = %v, want the provider's error", err)
	}
	f.provider.removeErr = nil
	if _, err := deliver(t, f, joined("p_patient")); err != nil || len(f.provider.removed) != 2 {
		t.Errorf("redelivery: %v, removed %v; want the patient removed again", err, f.provider.removed)
	}
}

// A join after the end records, starts and audits nothing, and deletes the room again: a join request that
// read the session open may have created the room after the end deleted it (CreateRoom is outside the join's
// transaction), and deleting it disconnects whoever is in it.
func TestHandleWebhook_AJoinAfterTheEndRecordsNothingAndDeletesTheRoomAgain(t *testing.T) {
	for _, identity := range []string{"p_practitioner", "p_patient"} {
		f := opened(now, time.Time{}, time.Time{})
		s := f.session()
		s.EndedAt, s.EndReason = now, videosession.EndPractitioner
		f.repo.sessions[apptA] = s
		f.consents.revoked = true
		if _, err := deliver(t, f, joined(identity)); err != nil {
			t.Fatal(err)
		}
		if got := f.session(); got != s || len(f.encounters.started) != 0 || len(f.provider.removed) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: session %+v, started %v, removed %v, events %v; want nothing", identity, got, f.encounters.started, f.provider.removed, f.audit.events)
		}
		if !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
			t.Errorf("%s: deleted %v (in a transaction: %t); want the room deleted again, after the commit", identity, f.provider.deleted, f.provider.inTx)
		}
	}
}

// A failed delete of the ended session's room fails the webhook, for LiveKit to deliver it again.
func TestHandleWebhook_AJoinAfterTheEndFailsWhenTheDeleteFails(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	s := f.session()
	s.EndedAt, s.EndReason = now, videosession.EndPractitioner
	f.repo.sessions[apptA] = s
	f.provider.deleteErr = errBackend
	if _, err := deliver(t, f, joined("p_patient")); !errors.Is(err, errBackend) {
		t.Errorf("HandleWebhook() = %v, want the provider's error", err)
	}
}

// The practitioner joining an appointment the clinic's scope no longer shows (cancelled by a revocation)
// records the join and starts no encounter.
func TestHandleWebhook_StartsNoEncounterOutOfScope(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	f.encounters.startErr = encounter.ErrNotFound
	if _, err := deliver(t, f, joined("p_practitioner")); err != nil {
		t.Fatalf("HandleWebhook() = %v, want no error", err)
	}
	if s := f.session(); !s.PractitionerJoinedAt.Equal(joinedAt) {
		t.Errorf("session %+v; want the join recorded", s)
	}
}

// A failure in the join's transaction rolls the join and its event back and fails the webhook, so LiveKit
// delivers it again.
func TestHandleWebhook_AFailedJoinChangesNothing(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"the join":      func(f *fixture) { f.repo.joinedErr = errBackend },
		"the encounter": func(f *fixture) { f.encounters.startErr = errBackend },
		"the audit":     func(f *fixture) { f.audit.err = errBackend },
		"the session":   func(f *fixture) { f.repo.err = errBackend },
	} {
		f := opened(now, time.Time{}, time.Time{})
		fail(f)
		if _, err := deliver(t, f, joined("p_practitioner")); !errors.Is(err, errBackend) {
			t.Errorf("%s fails: HandleWebhook() = %v, want its error", name, err)
		}
		if s := f.session(); !s.PractitionerJoinedAt.IsZero() || len(f.encounters.started) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s fails: session %+v, started %v, events %v; want nothing", name, s, f.encounters.started, f.audit.events)
		}
	}
	f := opened(now, start, time.Time{})
	f.consents.err = errBackend
	if _, err := deliver(t, f, joined("p_patient")); !errors.Is(err, errBackend) || !f.session().PatientJoinedAt.IsZero() || len(f.provider.removed) != 0 {
		t.Errorf("the consent fails: %v, session %+v, removed %v; want the error and nothing", err, f.session(), f.provider.removed)
	}
}

// A participant leaving or the room finishing before the window closes is not the end: a dropped connection
// reconnects. Nothing is asked of LiveKit and nothing changes.
func TestHandleWebhook_LeavingBeforeTheWindowClosesChangesNothing(t *testing.T) {
	for _, e := range []videosession.Event{left("p_patient"), left("p_practitioner"), roomFinished()} {
		f := opened(closes.Add(-time.Second), start, start)
		if _, err := deliver(t, f, e); err != nil {
			t.Fatal(err)
		}
		if s := f.session(); !s.EndedAt.IsZero() || len(f.provider.listed) != 0 || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: session %+v, listed %v, deleted %v; want nothing", e.Kind, s, f.provider.listed, f.provider.deleted)
		}
	}
}

// Once the window has closed, the last one leaving, or the room finishing, ends the session as the
// window-close job would: window_closed with the room empty (the encounter finishes, the room is deleted),
// practitioner_absent when the practitioner never joined, time_limit at the hard cap; with someone still in
// the room nothing changes.
func TestHandleWebhook_LeavingAfterTheWindowClosedEndsTheSession(t *testing.T) {
	for _, tc := range []struct {
		at           time.Time
		practitioner time.Time
		name         string
		want         videosession.EndReason
		in           []string
		finished     int
	}{
		{name: "empty", at: closes, practitioner: start, want: videosession.EndWindowClosed, finished: 1},
		{name: "occupied", at: closes.Add(time.Minute), practitioner: start, in: []string{"p_practitioner"}},
		{name: "practitioner never joined", at: closes, want: videosession.EndPractitionerAbsent},
		{name: "hard cap", at: hardCap, practitioner: start, in: []string{"p_practitioner"}, want: videosession.EndTimeLimit, finished: 1},
	} {
		for _, e := range []videosession.Event{left("p_patient"), roomFinished()} {
			f := opened(tc.at, tc.practitioner, start)
			f.provider.in = tc.in
			if _, err := deliver(t, f, e); err != nil {
				t.Fatalf("%s, %s: %v", tc.name, e.Kind, err)
			}
			s := f.session()
			if s.EndReason != tc.want || (tc.want != "" && !s.EndedAt.Equal(tc.at)) || len(f.encounters.finished) != tc.finished {
				t.Errorf("%s, %s: session %+v, finished %v; want ended %q", tc.name, e.Kind, s, f.encounters.finished, tc.want)
			}
			if tc.want == "" {
				if len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
					t.Errorf("%s, %s: deleted %v, events %v; want nothing", tc.name, e.Kind, f.provider.deleted, f.audit.events)
				}
				continue
			}
			if !slices.Equal(f.provider.deleted, []string{"r_room"}) || f.provider.inTx {
				t.Errorf("%s, %s: deleted %v (in a transaction: %t); want the room, after the commit", tc.name, e.Kind, f.provider.deleted, f.provider.inTx)
			}
			checkEvents(t, tc.name, f.audit, webhookEvent())
		}
	}
}

// The practitioner's join commits after a leave chose practitioner_absent and before its end: the end does not
// apply, the room stays and nothing is audited; the window-close job decides later.
func TestHandleWebhook_LeavingDoesNotEndAbsentAPractitionerWhoJoinsMeanwhile(t *testing.T) {
	f := opened(closes, time.Time{}, start)
	f.repo.practitionerJoinsAt = closes.Add(-time.Second)
	if _, err := deliver(t, f, left("p_patient")); err != nil {
		t.Fatal(err)
	}
	if s := f.session(); !s.EndedAt.IsZero() || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, deleted %v, events %v; want nothing", s, f.provider.deleted, f.audit.events)
	}
}

// A late leave or room_finished after the session ended changes nothing and deletes nothing again.
func TestHandleWebhook_LeavingAfterTheEndChangesNothing(t *testing.T) {
	f := opened(closes, start, start)
	s := f.session()
	s.EndedAt, s.EndReason = closes, videosession.EndPractitioner
	f.repo.sessions[apptA] = s
	for _, e := range []videosession.Event{left("p_patient"), roomFinished()} {
		if _, err := deliver(t, f, e); err != nil {
			t.Fatal(err)
		}
	}
	if f.session() != s || len(f.provider.listed) != 0 || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, listed %v, deleted %v, events %v; want nothing", f.session(), f.provider.listed, f.provider.deleted, f.audit.events)
	}
}

// A failed check of who is in the room fails the webhook, which LiveKit delivers again.
func TestHandleWebhook_FailsWhenLiveKitCannotSayWhoIsIn(t *testing.T) {
	f := opened(closes, start, start)
	f.provider.listErr = errBackend
	if _, err := deliver(t, f, left("p_patient")); !errors.Is(err, errBackend) || !f.session().EndedAt.IsZero() {
		t.Errorf("HandleWebhook() = %v, session %+v; want the error and the session open", err, f.session())
	}
}

// Every other kind (room_started, track_published, …) is handled with no effect, without reading a session.
func TestHandleWebhook_OtherKindsHaveNoEffect(t *testing.T) {
	f := opened(closes, start, start)
	f.repo.err = errBackend // nothing may be read
	kind, err := deliver(t, f, videosession.Event{Kind: videosession.EventOther, Room: "r_room", Identity: "p_patient", At: joinedAt})
	if err != nil || kind != videosession.EventOther {
		t.Errorf("HandleWebhook() = %q, %v; want other, handled", kind, err)
	}
}

// An event for a room or an identity we do not know is ErrUnknownRoom, with no effect.
func TestHandleWebhook_UnknownRoomsAndIdentitiesHaveNoEffect(t *testing.T) {
	for _, e := range []videosession.Event{
		{Kind: videosession.EventParticipantJoined, Room: "r_other", Identity: "p_patient", At: joinedAt},
		{Kind: videosession.EventParticipantJoined, Room: "r_room", Identity: "p_other", At: joinedAt},
		{Kind: videosession.EventParticipantLeft, Room: "r_room", Identity: "p_other", At: joinedAt},
		{Kind: videosession.EventRoomFinished, Room: "r_other", At: joinedAt},
		{Kind: videosession.EventParticipantJoined, At: joinedAt},
	} {
		f := opened(closes, start, time.Time{})
		before := f.session()
		kind, err := deliver(t, f, e)
		if !errors.Is(err, videosession.ErrUnknownRoom) || kind != e.Kind {
			t.Errorf("HandleWebhook(%+v) = %q, %v; want ErrUnknownRoom", e, kind, err)
		}
		if f.session() != before || len(f.audit.events) != 0 || len(f.provider.deleted) != 0 || len(f.provider.removed) != 0 {
			t.Errorf("HandleWebhook(%+v) changed something: %+v, %v", e, f.session(), f.audit.events)
		}
	}
}

// A webhook LiveKit did not sign is ErrInvalidSignature and handles nothing.
func TestHandleWebhook_RefusesAnUnsignedWebhook(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	f.provider.event = joined("p_practitioner")
	kind, err := f.svc.HandleWebhook(t.Context(), []byte("body"), "forged")
	if !errors.Is(err, videosession.ErrInvalidSignature) || kind != "" {
		t.Errorf("HandleWebhook() = %q, %v; want ErrInvalidSignature", kind, err)
	}
	if !f.session().PractitionerJoinedAt.IsZero() || len(f.audit.events) != 0 {
		t.Errorf("session %+v, events %v; want nothing", f.session(), f.audit.events)
	}
}

// A verified body that is not an event fails the webhook (a 500, which LiveKit retries).
func TestHandleWebhook_FailsOnABodyThatIsNotAnEvent(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	f.provider.parseErr = errBackend
	if _, err := deliver(t, f, joined("p_practitioner")); !errors.Is(err, errBackend) || errors.Is(err, videosession.ErrInvalidSignature) {
		t.Errorf("HandleWebhook() = %v, want the parse error", err)
	}
}

// Every join of a person into an open consultation the practitioner has joined asks the transcription agent
// in (SAL-4): a transcription_join job for the session, in the join's transaction. Repeats and reconnects ask
// again; the agent joins once per session. A patient whose access is gone, a join before the practitioner's
// (no Encounter yet), and a join after the end ask nothing.
func TestHandleWebhook_AsksTheAgentInOnceThePractitionerJoined(t *testing.T) {
	session := "5e000000-0000-4000-8000-000000000001 " + clinicA
	for _, tc := range []struct {
		setup func(f *fixture)
		name  string
		who   string
		want  []string
	}{
		{name: "the practitioner's join", who: "p_practitioner", want: []string{session}},
		{name: "the patient's join after the practitioner's", who: "p_patient", want: []string{session}, setup: func(f *fixture) {
			s := f.session()
			s.PractitionerJoinedAt = start
			f.repo.sessions[apptA] = s
		}},
		{name: "the patient's join before the practitioner's", who: "p_patient"},
		{name: "a patient whose consent is revoked", who: "p_patient", setup: func(f *fixture) {
			s := f.session()
			s.PractitionerJoinedAt = start
			f.repo.sessions[apptA] = s
			f.consents.revoked = true
		}},
		{name: "a join after the end", who: "p_practitioner", setup: func(f *fixture) {
			s := f.session()
			s.EndedAt, s.EndReason = now, videosession.EndPractitioner
			f.repo.sessions[apptA] = s
		}},
	} {
		f := opened(now, time.Time{}, time.Time{})
		if tc.setup != nil {
			tc.setup(f)
		}
		if _, err := deliver(t, f, joined(tc.who)); err != nil {
			t.Fatalf("%s: HandleWebhook() = %v", tc.name, err)
		}
		if !slices.Equal(f.jobs.agentJoins, tc.want) {
			t.Errorf("%s: agent joins %v, want %v", tc.name, f.jobs.agentJoins, tc.want)
		}
	}
}

// A failed enqueue fails the webhook, so LiveKit delivers it again; the join is not recorded meanwhile.
func TestHandleWebhook_FailsWhenTheAgentCannotBeAskedIn(t *testing.T) {
	f := opened(now, time.Time{}, time.Time{})
	f.jobs.agentErr = errBackend
	if _, err := deliver(t, f, joined("p_practitioner")); !errors.Is(err, errBackend) {
		t.Fatalf("HandleWebhook() = %v, want the jobs' error", err)
	}
	if s := f.session(); !s.PractitionerJoinedAt.IsZero() {
		t.Errorf("session %+v; want the join rolled back", s)
	}
}
