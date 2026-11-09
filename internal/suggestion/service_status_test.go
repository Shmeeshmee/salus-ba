package suggestion_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// pract is the encounter's practitioner, signed in at clinic A.
var pract = appointment.Caller{AccountID: "ac000000-0000-4000-8000-000000000001", OrganizationID: clinicA}

// callerEvent is the practitioner's audit event of a status change of count suggestions of the encounter id
// ("" when none was disclosed).
func callerEvent(id string, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         auditevent.ActionUpdate,
		Outcome:        outcome,
		AgentAccountID: pract.AccountID,
		OrganizationID: clinicA,
		EntityName:     "suggestion",
		EntityCount:    count,
	}
	if id != "" {
		e.EntityReference = "Encounter/" + id
	}
	return e
}

// stored is the fixture with the suggestion of valid() stored, as the worker stores it before it is shown.
func stored(t *testing.T) *fixture {
	t.Helper()
	f := newFixture()
	if err := f.svc.Record(t.Context(), consumer, valid()); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	return f
}

// decodePublished is the suggestion event of the envelope published.
func decodePublished(t *testing.T, p published) suggestion.Event {
	t.Helper()
	env, err := wire.Decode(p.envelope)
	if err != nil || env.Type != suggestion.MessageType || env.Version != suggestion.MessageVersion {
		t.Fatalf("published %s, %v; want a suggestion envelope", p.envelope, err)
	}
	var e suggestion.Event
	if err := json.Unmarshal(env.Payload, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

// The practitioner dismisses or pins a suggestion of the encounter: the status is stored, audited as their
// update, and the suggestion with its new status is published on the encounter's stream after the transaction,
// so every open view and the worker see it. The same status again stores nothing new but is published again
// (a retry after a lost event).
func TestSetStatus_StoresAuditsAndPublishesTheNewStatus(t *testing.T) {
	f := stored(t)
	for _, status := range []suggestion.Status{suggestion.StatusPinned, suggestion.StatusDismissed, suggestion.StatusDismissed} {
		got, err := f.svc.SetStatus(t.Context(), pract, encounterID, suggestionID, status)
		want := valid()
		want.Status = status
		if err != nil || got.Status != status || got.Text != want.Text {
			t.Fatalf("SetStatus(%s) = %+v, %v; want %+v", status, got, err, want)
		}
		if p := f.publisher.published[len(f.publisher.published)-1]; p.encounterID != encounterID || decodePublished(t, p).Status != status {
			t.Errorf("published %s for %s, want the suggestion %s", p.envelope, p.encounterID, status)
		}
	}
	if got := f.repo.rows[suggestionID].Status; got != suggestion.StatusDismissed {
		t.Errorf("stored %s, want dismissed", got)
	}
	if len(f.publisher.published) != 3 || f.publisher.inTx {
		t.Errorf("published %d (in a transaction: %v), want 3, after the transaction", len(f.publisher.published), f.publisher.inTx)
	}
	ok := callerEvent(encounterID, auditevent.OutcomeSuccess, 1)
	checkEvents(t, "statuses", f.audit, ok, ok, ok)
}

// Only a status the contract has, for a suggestion id; nothing is checked, stored or audited otherwise.
func TestSetStatus_RefusesAnInvalidChange(t *testing.T) {
	for name, tt := range map[string]struct {
		id     string
		status suggestion.Status
	}{
		"an unknown status": {suggestionID, "accepted"},
		"no status":         {suggestionID, ""},
		"an id no UUID":     {"s1", suggestion.StatusDismissed},
	} {
		f := stored(t)
		if _, err := f.svc.SetStatus(t.Context(), pract, encounterID, tt.id, tt.status); !errors.Is(err, suggestion.ErrInvalid) {
			t.Errorf("%s: error = %v, want ErrInvalid", name, err)
		}
		if len(f.audit.events) != 0 || len(f.publisher.published) != 0 || f.repo.rows[suggestionID].Status != suggestion.StatusShown {
			t.Errorf("%s: audited %d, published %d, stored %s; want nothing", name, len(f.audit.events), len(f.publisher.published), f.repo.rows[suggestionID].Status)
		}
	}
}

// The stream's rule decides (the encounter's practitioner with an active consent), and a suggestion of another
// encounter is not found: each refusal is audited, nothing is stored or published.
func TestSetStatus_RefusesWhomTheStreamRefusesAndAnotherEncountersSuggestion(t *testing.T) {
	for name, tt := range map[string]struct {
		want      error
		records   fakeRecords
		encounter string
		id        string // disclosed
	}{
		"a revoked consent":    {records: fakeRecords{err: transcript.ErrConsentRequired, id: encounterID}, encounter: encounterID, id: encounterID, want: suggestion.ErrConsentRequired},
		"another practitioner": {records: fakeRecords{err: transcript.ErrForbidden, id: encounterID}, encounter: encounterID, id: encounterID, want: suggestion.ErrForbidden},
		"an unseen encounter":  {records: fakeRecords{err: transcript.ErrNotFound}, encounter: encounterID, want: suggestion.ErrNotFound},
		"another's suggestion": {encounter: encounterID, id: encounterID, want: suggestion.ErrNotFound},
	} {
		f := stored(t)
		f.records.err, f.records.id = tt.records.err, tt.records.id
		id := suggestionID
		if name == "another's suggestion" {
			id = unknownID
		}
		if _, err := f.svc.SetStatus(t.Context(), pract, tt.encounter, id, suggestion.StatusDismissed); !errors.Is(err, tt.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tt.want)
		}
		if len(f.publisher.published) != 0 || f.repo.rows[suggestionID].Status != suggestion.StatusShown {
			t.Errorf("%s: published %d, stored %s; want nothing", name, len(f.publisher.published), f.repo.rows[suggestionID].Status)
		}
		checkEvents(t, name, f.audit, callerEvent(tt.id, auditevent.OutcomeMinorFailure, 0))
	}
}

// A backend's failure fails the change with nothing published; a failed publish is returned (the status is
// stored, and the practitioner's retry publishes it again).
func TestSetStatus_FailsWhenABackendFails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the repository": func(f *fixture) { f.repo.err = errBackend },
		"the rule":       func(f *fixture) { f.records.err = errBackend },
		"the audit":      func(f *fixture) { f.audit.err = errBackend },
		"the publisher":  func(f *fixture) { f.publisher.err = errBackend },
	} {
		f := stored(t)
		fail(f)
		if _, err := f.svc.SetStatus(t.Context(), pract, encounterID, suggestionID, suggestion.StatusPinned); !errors.Is(err, errBackend) {
			t.Errorf("%s: error = %v, want the backend's", name, err)
		}
		if len(f.publisher.published) != 0 {
			t.Errorf("%s: published %d, want none", name, len(f.publisher.published))
		}
	}
}
