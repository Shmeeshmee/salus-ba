package transcript_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// Export gives the encounter's practitioner its stored transcript in the order it was said, read in the
// clinic's scope, and audits the read with the number of segments.
func TestExport_GivesThePractitionerTheTranscriptInOrder(t *testing.T) {
	f := newFixture()
	second := f.storeAt(t, segmentID2, time.Minute)
	first := f.storeAt(t, segmentID, 0)

	got, err := f.svc.Export(t.Context(), pract, encounterID)
	if err != nil || !slices.Equal(got, []transcript.Event{first, second}) {
		t.Errorf("Export() = %+v, %v; want both segments in order", got, err)
	}
	checkEvents(t, "export", f.audit, callerEvent(pract, auditevent.CodeRead, encounterID, auditevent.OutcomeSuccess, 2))
}

func TestExport_GivesAnEmptyTranscriptWhenNothingWasSaid(t *testing.T) {
	f := newFixture()
	got, err := f.svc.Export(t.Context(), pract, encounterID)
	if err != nil || len(got) != 0 {
		t.Errorf("Export() = %+v, %v; want none", got, err)
	}
	checkEvents(t, "empty", f.audit, callerEvent(pract, auditevent.CodeRead, encounterID, auditevent.OutcomeSuccess, 0))
}

// refusals are the callers who may not read what was said, and why; each refusal is audited.
var refusals = []struct {
	want   error
	setup  func(*fixture)
	name   string
	id     string // "" when the refusal discloses no encounter
	caller appointment.Caller
}{
	{name: "the clinic's admin", caller: admin, id: encounterID, want: transcript.ErrForbidden},
	{name: "the patient, while the clinic does not share transcripts", caller: me, id: encounterID, want: transcript.ErrForbidden},
	{name: "another practitioner of the clinic", caller: pract2, id: encounterID, want: transcript.ErrForbidden},
	{name: "the practitioner after a revocation", caller: pract, id: encounterID, want: transcript.ErrConsentRequired, setup: (*fixture).revoke},
	{name: "someone the appointment does not show", caller: appointment.Caller{AccountID: "ac900000-0000-4000-8000-000000000009", OrganizationID: clinicA}, want: transcript.ErrNotFound},
}

func TestExport_RefusesEveryoneElse(t *testing.T) {
	for _, tt := range refusals {
		f := newFixture()
		f.storeAt(t, segmentID, 0)
		if tt.setup != nil {
			tt.setup(f)
		}
		if got, err := f.svc.Export(t.Context(), tt.caller, encounterID); !errors.Is(err, tt.want) || got != nil {
			t.Errorf("%s: Export() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		checkEvents(t, tt.name, f.audit, callerEvent(tt.caller, auditevent.CodeRead, tt.id, auditevent.OutcomeMinorFailure, 0))
	}
}

func TestExport_FailsWhenABackendFails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the repository": func(f *fixture) { f.repo.err = errBackend },
		"the encounters": func(f *fixture) { f.encounters.err = errBackend },
		"the audit":      func(f *fixture) { f.audit.err = errBackend },
	} {
		f := newFixture()
		fail(f)
		if _, err := f.svc.Export(t.Context(), pract, encounterID); !errors.Is(err, errBackend) {
			t.Errorf("%s: Export() error = %v, want the backend's", name, err)
		}
	}
}

// SAL-17: while the clinic shares transcripts with its patients, the encounter's patient reads their copy, in their
// own scope, audited as their own request; also after revoking the consent (their own record). The practitioner's
// read does not ask for the clinic's setting.
func TestExport_GivesThePatientTheirCopyWhileTheClinicSharesIt(t *testing.T) {
	for name, revoked := range map[string]bool{"active consent": false, "revoked": true} {
		f := newFixture()
		first := f.storeAt(t, segmentID, 0)
		f.clinics.copy = true
		if revoked {
			f.revoke()
		}
		got, err := f.svc.Export(t.Context(), me, encounterID)
		if err != nil || !slices.Equal(got, []transcript.Event{first}) {
			t.Errorf("%s: Export() = %+v, %v; want the segment", name, got, err)
		}
		checkEvents(t, name, f.audit, callerEvent(me, auditevent.CodeRead, encounterID, auditevent.OutcomeSuccess, 1))
	}
	f := newFixture()
	f.clinics.copy = true
	if _, err := f.svc.Export(t.Context(), pract, encounterID); err != nil || f.clinics.calls != 0 {
		t.Errorf("the practitioner's Export() = %v after %d setting reads, want none", err, f.clinics.calls)
	}
	f.clinics.err = errBackend
	if _, err := f.svc.Export(t.Context(), me, encounterID); !errors.Is(err, errBackend) {
		t.Errorf("Export() with the clinics down = %v, want the failure", err)
	}
}

// The patient's copy is the stored transcript only: never the live stream, whatever the clinic's setting.
func TestSubscribe_RefusesThePatientWhileTheClinicSharesTheTranscript(t *testing.T) {
	f := newFixture()
	f.clinics.copy = true
	if _, err := f.svc.Subscribe(t.Context(), me, encounterID, ""); !errors.Is(err, transcript.ErrForbidden) {
		t.Errorf("the patient's Subscribe() = %v, want ErrForbidden", err)
	}
}

// Subscribe lets the encounter's practitioner follow its stream from the cursor they give (everything still on
// the stream when none), and audits the subscription as an operation on the encounter.
func TestSubscribe_LetsThePractitionerFollowTheStream(t *testing.T) {
	for _, tt := range []struct{ after, want string }{
		{after: "", want: "0"},
		{after: "0", want: "0"},
		{after: "1731150000000-0", want: "1731150000000-0"},
		{after: "1731150000000-12", want: "1731150000000-12"},
	} {
		f := newFixture()
		got, err := f.svc.Subscribe(t.Context(), pract, encounterID, tt.after)
		if err != nil || got != tt.want {
			t.Errorf("Subscribe(after %q) = %q, %v; want %q", tt.after, got, err, tt.want)
		}
		checkEvents(t, "subscribe", f.audit, callerEvent(pract, auditevent.CodeOperation, encounterID, auditevent.OutcomeSuccess, 0))
	}
}

func TestSubscribe_RefusesACursorThatIsNoStreamID(t *testing.T) {
	for _, after := range []string{"$", ">", "-", "+", "1", "1-", "-1", "a-1", "1-1-1", "1 -1", "99999999999999999999999-0"} {
		f := newFixture()
		if _, err := f.svc.Subscribe(t.Context(), pract, encounterID, after); !errors.Is(err, transcript.ErrInvalidCursor) {
			t.Errorf("Subscribe(after %q) error = %v, want ErrInvalidCursor", after, err)
		}
		if f.encounters.accessed != 0 || len(f.audit.events) != 0 {
			t.Errorf("after %q: read the encounter %d times and audited %d events, want none", after, f.encounters.accessed, len(f.audit.events))
		}
	}
}

func TestSubscribe_RefusesEveryoneElse(t *testing.T) {
	for _, tt := range refusals {
		f := newFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		if _, err := f.svc.Subscribe(t.Context(), tt.caller, encounterID, ""); !errors.Is(err, tt.want) {
			t.Errorf("%s: Subscribe() error = %v; want %v", tt.name, err, tt.want)
		}
		checkEvents(t, tt.name, f.audit, callerEvent(tt.caller, auditevent.CodeOperation, tt.id, auditevent.OutcomeMinorFailure, 0))
	}
}

func entries(ids ...string) []transcript.Entry {
	var out []transcript.Entry
	for _, id := range ids {
		out = append(out, transcript.Entry{ID: id, Envelope: []byte(`{"type":"transcript","version":1,"payload":{"n":"` + id + `"}}`)})
	}
	return out
}

// Next gives the entries strictly after the cursor, blocking at most RecheckEvery for new ones, and is not
// audited: the subscription was.
func TestNext_GivesTheEntriesAfterTheCursor(t *testing.T) {
	f := newFixture()
	f.stream.entries[encounterID] = entries("1-0", "2-0", "3-0")
	for _, tt := range []struct {
		after string
		want  []string
	}{
		{after: "0", want: []string{"1-0", "2-0", "3-0"}},
		{after: "1-0", want: []string{"2-0", "3-0"}},
		{after: "3-0", want: nil},
	} {
		got, err := f.svc.Next(t.Context(), pract, encounterID, tt.after)
		var ids []string
		for _, e := range got {
			ids = append(ids, e.ID)
		}
		if err != nil || !slices.Equal(ids, tt.want) {
			t.Errorf("Next(after %q) = %v, %v; want %v", tt.after, ids, err, tt.want)
		}
	}
	for _, w := range f.stream.waits {
		if w != transcript.RecheckEvery {
			t.Errorf("blocked for %v, want %v", w, transcript.RecheckEvery)
		}
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %d events, want none", len(f.audit.events))
	}
}

// Access is checked again on every read, after it returns and before anything is given: once the consent is
// revoked (or the access otherwise gone) not one more entry leaves, also when the read found none.
func TestNext_GivesNothingOnceAccessHasEnded(t *testing.T) {
	for _, tt := range []struct {
		want   error
		change func(*fixture)
		name   string
		stream []transcript.Entry
	}{
		{name: "revoked, with entries waiting", change: (*fixture).revoke, stream: entries("1-0"), want: transcript.ErrConsentRequired},
		{name: "revoked, an idle stream", change: (*fixture).revoke, want: transcript.ErrConsentRequired},
		{name: "the appointment hidden", change: func(f *fixture) { delete(f.encounters.attendance, practAcc) }, stream: entries("1-0"), want: transcript.ErrNotFound},
	} {
		f := newFixture()
		f.stream.entries[encounterID] = tt.stream
		tt.change(f)
		if got, err := f.svc.Next(t.Context(), pract, encounterID, "0"); !errors.Is(err, tt.want) || got != nil {
			t.Errorf("%s: Next() = %v, %v; want nothing and %v", tt.name, got, err, tt.want)
		}
	}
}

func TestNext_FailsWhenABackendFails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the stream":     func(f *fixture) { f.stream.err = errBackend },
		"the encounters": func(f *fixture) { f.encounters.err = errBackend },
	} {
		f := newFixture()
		fail(f)
		if _, err := f.svc.Next(t.Context(), pract, encounterID, "0"); !errors.Is(err, errBackend) {
			t.Errorf("%s: Next() error = %v, want the backend's", name, err)
		}
	}
}

// Allowed is the stream's rule for another domain acting on what the stream shows (a suggestion's dismissal,
// SAL-7): the encounter's practitioner with an active consent gets the encounter's id; everyone else Export's
// refusal, with the id as far as they may know it. It audits nothing: the caller audits its own action.
func TestAllowed_IsTheStreamsRule(t *testing.T) {
	f := newFixture()
	if id, err := f.svc.Allowed(t.Context(), pract, encounterID); err != nil || id != encounterID {
		t.Errorf("Allowed(practitioner) = %q, %v; want the encounter", id, err)
	}
	for _, tt := range refusals {
		f := newFixture()
		if tt.setup != nil {
			tt.setup(f)
		}
		if id, err := f.svc.Allowed(t.Context(), tt.caller, encounterID); !errors.Is(err, tt.want) || id != tt.id {
			t.Errorf("%s: Allowed() = %q, %v; want %q, %v", tt.name, id, err, tt.id, tt.want)
		}
		checkEvents(t, tt.name, f.audit)
	}
	f = newFixture()
	f.encounters.err = errBackend
	if _, err := f.svc.Allowed(t.Context(), pract, encounterID); !errors.Is(err, errBackend) {
		t.Errorf("Allowed() error = %v, want the backend's", err)
	}
}
