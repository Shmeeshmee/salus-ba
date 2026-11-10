package suggestion_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/suggestion"
)

// A suggestion is stored when first shown, in the encounter's clinic scope, and its write audited.
func TestRecord_StoresANewSuggestion(t *testing.T) {
	f := newFixture()
	e := valid()
	if err := f.svc.Record(t.Context(), consumer, e); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if got := f.repo.rows[suggestionID]; got.ID != e.ID || got.Text != e.Text || got.Status != e.Status || !slices.Equal(got.SourceSegmentIDs, e.SourceSegmentIDs) {
		t.Errorf("stored %+v, want %+v", got, e)
	}
	checkEvents(t, "new", f.audit, systemEvent(auditevent.CodeCreate, auditevent.OutcomeSuccess, 1))
}

// A later event of a stored suggestion changes nothing (SAL-7): whoever changed its status (the practitioner,
// through SetStatus) stored it before the event was published, so a stale event delivered again after a newer
// one cannot set an older status back. An event delivered again is not audited again.
func TestRecord_StoresOnceAndALaterEventChangesNothing(t *testing.T) {
	f := newFixture()
	pinned := valid()
	pinned.Status, pinned.Text = suggestion.StatusPinned, "Placeholder text changed"
	for _, e := range []suggestion.Event{valid(), valid(), pinned, pinned} {
		if err := f.svc.Record(t.Context(), consumer, e); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}
	if got := f.repo.rows[suggestionID]; got.Status != suggestion.StatusShown || got.Text != valid().Text {
		t.Errorf("stored %+v, want the first event as it was", got)
	}
	checkEvents(t, "once", f.audit, systemEvent(auditevent.CodeCreate, auditevent.OutcomeSuccess, 1))
}

func TestRecord_RefusesAnInvalidEvent(t *testing.T) {
	f := newFixture()
	e := valid()
	e.SourceSegmentIDs = nil
	if err := f.svc.Record(t.Context(), consumer, e); !errors.Is(err, suggestion.ErrInvalid) {
		t.Errorf("Record(invalid) error = %v, want ErrInvalid", err)
	}
	if len(f.repo.rows) != 0 || len(f.audit.events) != 0 {
		t.Errorf("stored %d and audited %d, want none", len(f.repo.rows), len(f.audit.events))
	}
}

func TestRecord_IsNotFoundForAnUnknownEncounter(t *testing.T) {
	f := newFixture()
	e := valid()
	e.EncounterID = unknownID
	if err := f.svc.Record(t.Context(), consumer, e); !errors.Is(err, suggestion.ErrNotFound) {
		t.Errorf("Record() error = %v, want ErrNotFound", err)
	}
	if len(f.repo.rows) != 0 {
		t.Errorf("stored %d, want none", len(f.repo.rows))
	}
}

// After a revocation nothing more is written; the refusal is audited.
func TestRecord_WritesNothingAfterARevocation(t *testing.T) {
	f := newFixture()
	f.consents.state.Active = false
	if err := f.svc.Record(t.Context(), consumer, valid()); !errors.Is(err, suggestion.ErrConsentRequired) {
		t.Errorf("Record() error = %v, want ErrConsentRequired", err)
	}
	if len(f.repo.rows) != 0 {
		t.Errorf("stored %d, want none", len(f.repo.rows))
	}
	checkEvents(t, "revoked", f.audit, systemEvent(auditevent.CodeCreate, auditevent.OutcomeMinorFailure, 0))
}

// Without the patient's transcription consent nothing is written either (SAL-12); the refusal is audited.
func TestRecord_WritesNothingWithoutTheTranscriptionConsent(t *testing.T) {
	f := newFixture()
	f.consents.state.Transcription = false
	if err := f.svc.Record(t.Context(), consumer, valid()); !errors.Is(err, suggestion.ErrConsentRequired) {
		t.Errorf("Record() error = %v, want ErrConsentRequired", err)
	}
	if len(f.repo.rows) != 0 {
		t.Errorf("stored %d, want none", len(f.repo.rows))
	}
	checkEvents(t, "withdrawn", f.audit, systemEvent(auditevent.CodeCreate, auditevent.OutcomeMinorFailure, 0))
}

func TestRecord_FailsWithNothingStoredWhenABackendFails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the repository": func(f *fixture) { f.repo.err = errBackend },
		"the encounters": func(f *fixture) { f.encounters.err = errBackend },
		"the consents":   func(f *fixture) { f.consents.err = errBackend },
		"the audit":      func(f *fixture) { f.audit.err = errBackend },
	} {
		f := newFixture()
		fail(f)
		if err := f.svc.Record(t.Context(), consumer, valid()); !errors.Is(err, errBackend) {
			t.Errorf("%s: Record() error = %v, want the backend's", name, err)
		}
		if len(f.repo.rows) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: stored %d and audited %d, want none", name, len(f.repo.rows), len(f.audit.events))
		}
	}
}
