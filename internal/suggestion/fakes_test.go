package suggestion_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
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
	f          *fixture
	err        error
	failInsert error // Insert's alone
	rows       map[string]suggestion.Event
}

func (r *fakeRepository) Insert(ctx context.Context, e suggestion.Event) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	if r.err != nil {
		return false, r.err
	}
	if r.failInsert != nil {
		return false, r.failInsert
	}
	if r.f.scope == "" {
		return false, errors.New("write outside a clinic's scope")
	}
	if _, ok := r.rows[e.ID]; ok {
		return false, nil
	}
	r.rows[e.ID] = e
	return true, nil
}

func (r *fakeRepository) SetStatus(ctx context.Context, encounterID, id string, status suggestion.Status) (suggestion.Event, bool, error) {
	if err := inTx(ctx); err != nil {
		return suggestion.Event{}, false, err
	}
	if r.err != nil {
		return suggestion.Event{}, false, r.err
	}
	got, ok := r.rows[id]
	if !ok || r.f.scope == "" || got.EncounterID != encounterID {
		return suggestion.Event{}, false, suggestion.ErrNotFound
	}
	changed := got.Status != status
	got.Status = status
	r.rows[id] = got
	return got, changed, nil
}

// ByEncounter gives the encounter's suggestions in the order of their ids (the fake keeps no time).
func (r *fakeRepository) ByEncounter(ctx context.Context, encounterID string) ([]suggestion.Event, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.f.scope == "" {
		return nil, errors.New("read outside a clinic's scope")
	}
	var out []suggestion.Event
	for _, id := range slices.Sorted(maps.Keys(r.rows)) {
		if e := r.rows[id]; e.EncounterID == encounterID {
			out = append(out, e)
		}
	}
	return out, nil
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

// fakeRecords is the stream's rule (transcript.Service.Allowed): refused with err, else the encounter's
// practitioner may act on encounter A, scoping the transaction to its clinic; it checks it runs in a
// transaction.
type fakeRecords struct {
	f   *fixture
	err error
	id  string // the encounter disclosed with err
}

func (r *fakeRecords) Allowed(ctx context.Context, _ appointment.Caller, id string) (string, error) {
	if err := inTx(ctx); err != nil {
		return "", err
	}
	if r.err != nil {
		return r.id, r.err
	}
	if id != encounterID {
		return "", transcript.ErrNotFound
	}
	r.f.scope = clinicA
	return id, nil
}

// fakePublisher records what was published, by encounter.
type fakePublisher struct {
	err       error
	published []published
	mu        sync.Mutex
	inTx      bool // a publish was made inside a transaction
}

type published struct {
	encounterID string
	envelope    []byte
}

func (p *fakePublisher) Publish(ctx context.Context, encounterID string, envelope []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx.Value(txKey{}) != nil {
		p.inTx = true
	}
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, published{encounterID, slices.Clone(envelope)})
	return nil
}

type fixture struct {
	records    *fakeRecords
	publisher  *fakePublisher
	repo       *fakeRepository
	encounters *fakeEncounters
	consents   *fakeConsents
	audit      *fakeAuditor
	svc        *suggestion.Service
	scope      string
}

// newFixture is clinic A's encounter A of patient A, with an active consent.
func newFixture() *fixture {
	f := &fixture{consents: &fakeConsents{state: access.Consent{Active: true, Transcription: true}}, audit: &fakeAuditor{}, publisher: &fakePublisher{}}
	f.repo = &fakeRepository{f: f, rows: map[string]suggestion.Event{}}
	f.encounters = &fakeEncounters{f: f}
	f.records = &fakeRecords{f: f}
	f.svc = suggestion.NewService(suggestion.Ports{Repo: f.repo, Encounters: f.encounters, Consents: f.consents, Audit: f.audit, Tx: fakeTransactor{f: f},
		Records: f.records, Publisher: f.publisher,
	})
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
