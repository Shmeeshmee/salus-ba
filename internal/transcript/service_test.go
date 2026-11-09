package transcript_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// A final segment is stored once, in the encounter's clinic scope, and its write audited as the consumer's.
func TestStore_StoresAFinalSegmentInTheEncountersClinicScope(t *testing.T) {
	f := newFixture()
	e := valid()
	if err := f.svc.Store(t.Context(), consumer, e); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if got, ok := f.repo.segments[segmentID]; !ok || got != e {
		t.Errorf("stored %+v, want %+v", f.repo.segments, e)
	}
	checkEvents(t, "store", f.audit, systemEvent(auditevent.OutcomeSuccess, 1))
}

// The stream delivers at least once: a segment delivered again is neither stored nor audited again.
func TestStore_IsIdempotentForASegmentDeliveredAgain(t *testing.T) {
	f := newFixture()
	for range 2 {
		if err := f.svc.Store(t.Context(), consumer, valid()); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}
	if len(f.repo.segments) != 1 {
		t.Errorf("stored %d segments, want 1", len(f.repo.segments))
	}
	checkEvents(t, "store twice", f.audit, systemEvent(auditevent.OutcomeSuccess, 1))
}

// Partial segments are only pushed live (plan "Pipeline"): nothing is stored, read or audited.
func TestStore_IgnoresAPartialSegment(t *testing.T) {
	f := newFixture()
	e := valid()
	e.IsFinal = false
	f.encounters.err = errBackend // never asked
	if err := f.svc.Store(t.Context(), consumer, e); err != nil {
		t.Fatalf("Store(partial) error = %v", err)
	}
	if len(f.repo.segments) != 0 || len(f.audit.events) != 0 {
		t.Errorf("stored %d segments and audited %d events, want none", len(f.repo.segments), len(f.audit.events))
	}
}

func TestStore_RefusesAnInvalidEvent(t *testing.T) {
	f := newFixture()
	e := valid()
	e.Speaker = "doctor"
	if err := f.svc.Store(t.Context(), consumer, e); !errors.Is(err, transcript.ErrInvalid) {
		t.Errorf("Store(invalid) error = %v, want ErrInvalid", err)
	}
	if len(f.repo.segments) != 0 || len(f.audit.events) != 0 {
		t.Errorf("stored %d segments and audited %d events, want none", len(f.repo.segments), len(f.audit.events))
	}
}

// An encounter the system path cannot find (unknown, or hidden from its clinic) gets nothing stored.
func TestStore_IsNotFoundForAnUnknownEncounter(t *testing.T) {
	f := newFixture()
	e := valid()
	e.EncounterID = unknownID
	if err := f.svc.Store(t.Context(), consumer, e); !errors.Is(err, transcript.ErrNotFound) {
		t.Errorf("Store() error = %v, want ErrNotFound", err)
	}
	if len(f.repo.segments) != 0 {
		t.Errorf("stored %d segments, want none", len(f.repo.segments))
	}
}

// After a revocation nothing more is written (CLAUDE.md "Revocation takes effect immediately"; migration 00031),
// also for a consultation that started before it, which the clinic's scope still shows: the refusal is audited.
func TestStore_WritesNothingAfterARevocation(t *testing.T) {
	f := newFixture()
	f.revoke()
	if err := f.svc.Store(t.Context(), consumer, valid()); !errors.Is(err, transcript.ErrConsentRequired) {
		t.Errorf("Store() error = %v, want ErrConsentRequired", err)
	}
	if len(f.repo.segments) != 0 {
		t.Errorf("stored %d segments, want none", len(f.repo.segments))
	}
	checkEvents(t, "revoked", f.audit, systemEvent(auditevent.OutcomeMinorFailure, 0))
}

func TestStore_FailsWithNothingStoredWhenABackendFails(t *testing.T) {
	for name, fail := range map[string]func(*fixture){
		"the repository": func(f *fixture) { f.repo.err = errBackend },
		"the encounters": func(f *fixture) { f.encounters.err = errBackend },
		"the consents":   func(f *fixture) { f.consents.err = errBackend },
		"the audit":      func(f *fixture) { f.audit.err = errBackend },
	} {
		f := newFixture()
		fail(f)
		if err := f.svc.Store(t.Context(), consumer, valid()); !errors.Is(err, errBackend) {
			t.Errorf("%s: Store() error = %v, want the backend's", name, err)
		}
		if len(f.repo.segments) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: stored %d segments and audited %d events, want none", name, len(f.repo.segments), len(f.audit.events))
		}
	}
}

// storeAt stores a final segment of encounter A with the id, starting at offset after the first.
func (f *fixture) storeAt(t *testing.T, id string, offset time.Duration) transcript.Event {
	t.Helper()
	e := valid()
	e.SegmentID, e.TsStart, e.TsEnd = id, e.TsStart.Add(offset), e.TsEnd.Add(offset)
	if err := (fakeTransactor{f: f}).WithTx(t.Context(), func(ctx context.Context) error {
		f.scope = clinicA
		_, err := f.repo.Insert(ctx, e)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return e
}
