package suggestion_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/suggestion"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	unknownID = "00000000-0000-4000-8000-000000000000"
	consumer  = "consult_stream"
)

var errBackend = errors.New("the database is down")

type txKey struct{}

// fakeTransactor runs fn in a "transaction": the scope set in it ends with it, and when fn fails what was
// stored and audited in it is dropped, as a rollback does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, rows := len(t.f.audit.events), maps.Clone(t.f.repo.rows)
	defer func() { t.f.scope = "" }()
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.rows = t.f.audit.events[:events], rows
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeRepository holds the suggestions by id, written only in a clinic's scope: what was suggested never
// changes, its status does.
type fakeRepository struct {
	f    *fixture
	err  error
	rows map[string]suggestion.Event
}

func (r *fakeRepository) Save(ctx context.Context, e suggestion.Event) (suggestion.Saved, error) {
	if err := inTx(ctx); err != nil {
		return 0, err
	}
	if r.err != nil {
		return 0, r.err
	}
	if r.f.scope == "" {
		return 0, errors.New("write outside a clinic's scope")
	}
	got, ok := r.rows[e.ID]
	switch {
	case !ok:
		r.rows[e.ID] = e
		return suggestion.SavedCreated, nil
	case got.EncounterID != e.EncounterID || got.Status == e.Status:
		return suggestion.SavedUnchanged, nil
	}
	got.Status = e.Status
	r.rows[e.ID] = got
	return suggestion.SavedUpdated, nil
}

// fakeEncounters gives encounter A (patient A, clinic A) to the system path, scoping the transaction to its
// clinic.
type fakeEncounters struct {
	f   *fixture
	err error
}

func (e *fakeEncounters) Lookup(ctx context.Context, id string) (encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, err
	}
	if e.err != nil {
		return encounter.Encounter{}, e.err
	}
	if id != encounterID {
		return encounter.Encounter{}, encounter.ErrNotFound
	}
	e.f.scope = clinicA
	return encounter.Encounter{ID: encounterID, OrganizationID: clinicA, PatientID: patientA, Status: encounter.StatusInProgress}, nil
}

// fakeConsents gives the state of patient A's consent with clinic A.
type fakeConsents struct {
	err   error
	state access.Consent
}

func (c *fakeConsents) State(_ context.Context, patientID, org string) (access.Consent, error) {
	if c.err != nil {
		return access.Consent{}, c.err
	}
	if patientID != patientA || org != clinicA {
		return access.Consent{}, nil
	}
	return c.state, nil
}

// fakeAuditor records events; recording outside a transaction is an error.
type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

type fixture struct {
	repo       *fakeRepository
	encounters *fakeEncounters
	consents   *fakeConsents
	audit      *fakeAuditor
	svc        *suggestion.Service
	scope      string
}

// newFixture is clinic A's encounter A of patient A, with an active consent.
func newFixture() *fixture {
	f := &fixture{consents: &fakeConsents{state: access.Consent{Active: true}}, audit: &fakeAuditor{}}
	f.repo = &fakeRepository{f: f, rows: map[string]suggestion.Event{}}
	f.encounters = &fakeEncounters{f: f}
	f.svc = suggestion.NewService(suggestion.Ports{Repo: f.repo, Encounters: f.encounters, Consents: f.consents, Audit: f.audit, Tx: fakeTransactor{f: f}})
	return f
}

func checkEvents(t *testing.T, name string, a *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(a.events, want) {
		t.Errorf("%s: audited %+v,\nwant %+v", name, a.events, want)
	}
}

// systemEvent is the stream consumer's audit event of a write of a suggestion of encounter A.
func systemEvent(interaction string, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	action := auditevent.ActionCreate
	if interaction == auditevent.CodeUpdate {
		action = auditevent.ActionUpdate
	}
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         outcome,
		AgentName:       consumer,
		OrganizationID:  clinicA,
		EntityReference: "Encounter/" + encounterID,
		EntityName:      "suggestion",
		EntityCount:     count,
	}
}
