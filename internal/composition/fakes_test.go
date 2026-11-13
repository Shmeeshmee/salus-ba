package composition_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	roleA1    = "a1000000-0000-4000-8000-0000000000a1"
	apptA     = "e1000000-0000-4000-8000-0000000000a1"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	pract2Acc = "ac200000-0000-4000-8000-0000000000a2"
	adminAcc  = "ad000000-0000-4000-8000-0000000000ad"
	accountA  = "acc00000-0000-4000-8000-0000000000aa"
	unknownID = "00000000-0000-4000-8000-000000000000"
	seg1      = "51000000-0000-4000-8000-000000000001"
	seg2      = "51000000-0000-4000-8000-000000000002"
)

var (
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: "p1000000-0000-4000-8000-0000000000a1"}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: "p2000000-0000-4000-8000-0000000000a2"}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	errBackend = errors.New("the database is down")
	now        = time.Date(2030, 1, 7, 9, 30, 0, 0, time.UTC)
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction": the scope set in it ends with it, and when fn fails what was
// stored and audited in it is dropped, as a rollback does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	t.f.mu.Lock()
	events, templates, drafts := len(t.f.audit.events), maps.Clone(t.f.repo.templates), maps.Clone(t.f.repo.drafts)
	jobs, conds, refreshes := len(t.f.jobs.signed), len(t.f.conditions.recorded), len(t.f.jobs.refreshes)
	t.f.mu.Unlock()
	defer func() { t.f.repo.scope = "" }()
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.mu.Lock()
		t.f.audit.events, t.f.repo.templates, t.f.repo.drafts = t.f.audit.events[:events], templates, drafts
		t.f.jobs.signed, t.f.conditions.recorded = t.f.jobs.signed[:jobs], t.f.conditions.recorded[:conds]
		t.f.jobs.refreshes = t.f.jobs.refreshes[:refreshes]
		t.f.mu.Unlock()
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeRepository holds the templates and drafts by id; templates are seen only in their clinic's scope, and a
// patient's scope (patientScope) sees only the final drafts, as row-level security shows it.
type fakeRepository struct {
	err       error
	templates map[string]composition.Template
	drafts    map[string]composition.Draft
	scope     string
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	r.scope = id
	return r.err
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	r.scope = patientScope + id
	return r.err
}

// patientScope prefixes the scope of a patient.
const patientScope = "patient:"

// visible reports whether the scope shows d: a patient's only a final one.
func (r *fakeRepository) visible(d composition.Draft) bool {
	return !strings.HasPrefix(r.scope, patientScope) || d.Status == composition.StatusFinal
}

func (r *fakeRepository) Templates(ctx context.Context, org string) ([]composition.Template, error) {
	if err := r.check(ctx); err != nil {
		return nil, err
	}
	var out []composition.Template
	for _, t := range r.templates {
		if t.OrganizationID == org && org == r.scope {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

func (r *fakeRepository) Template(ctx context.Context, id string, _ bool) (composition.Template, error) {
	if err := r.check(ctx); err != nil {
		return composition.Template{}, err
	}
	t, ok := r.templates[id]
	if !ok || t.OrganizationID != r.scope {
		return composition.Template{}, composition.ErrTemplateNotFound
	}
	return t, nil
}

func (r *fakeRepository) InsertTemplate(ctx context.Context, org string, c composition.TemplateContent, at time.Time) (composition.Template, error) {
	if err := r.check(ctx); err != nil {
		return composition.Template{}, err
	}
	t := composition.Template{
		ID: "70000000-0000-4000-8000-" + strconv.FormatInt(int64(100000000000+len(r.templates)), 10), OrganizationID: org,
		LastUpdated: at, VersionID: 1,
	}
	r.templates[t.ID] = withContent(t, c)
	return r.templates[t.ID], nil
}

func (r *fakeRepository) UpdateTemplate(ctx context.Context, id string, c composition.TemplateContent, at time.Time) (composition.Template, error) {
	if err := r.check(ctx); err != nil {
		return composition.Template{}, err
	}
	t := r.templates[id]
	t.LastUpdated, t.VersionID = at, t.VersionID+1
	r.templates[id] = withContent(t, c)
	return r.templates[id], nil
}

func withContent(t composition.Template, c composition.TemplateContent) composition.Template {
	t.Title, t.Language, t.Status, t.TypeCode = c.Title, c.Language, c.Status, c.TypeCode
	t.Fields, t.Translations = slices.Clone(c.Fields), slices.Clone(c.Translations)
	return t
}

func (r *fakeRepository) SaveDraft(ctx context.Context, d composition.Draft, at time.Time) (composition.Draft, bool, error) {
	if err := r.check(ctx); err != nil {
		return composition.Draft{}, false, err
	}
	for id, open := range r.drafts {
		if open.EncounterID == d.EncounterID && open.TemplateID == d.TemplateID && open.Status == composition.StatusPreliminary {
			open.Fields, open.Language, open.Title, open.TypeCode = d.Fields, d.Language, d.Title, d.TypeCode
			open.UpdatedAt, open.VersionID = at, open.VersionID+1
			r.drafts[id] = open
			return open, false, nil
		}
	}
	d.ID = "d0000000-0000-4000-8000-" + strconv.FormatInt(int64(100000000000+len(r.drafts)), 10)
	d.Status, d.UpdatedAt, d.VersionID = composition.StatusPreliminary, at, 1
	r.drafts[d.ID] = d
	return d, true, nil
}

func (r *fakeRepository) Draft(ctx context.Context, id string, _ bool) (composition.Draft, error) {
	if err := r.check(ctx); err != nil {
		return composition.Draft{}, err
	}
	d, ok := r.drafts[id]
	if !ok || !r.visible(d) {
		return composition.Draft{}, composition.ErrNotFound
	}
	return d, nil
}

func (r *fakeRepository) ByDocumentReference(ctx context.Context, id string) (composition.Draft, error) {
	return r.by(ctx, func(d composition.Draft) bool { return d.DocumentReferenceID == id })
}

func (r *fakeRepository) ByProvenance(ctx context.Context, id string) (composition.Draft, error) {
	return r.by(ctx, func(d composition.Draft) bool { return d.ProvenanceID == id })
}

func (r *fakeRepository) by(ctx context.Context, match func(composition.Draft) bool) (composition.Draft, error) {
	if err := r.check(ctx); err != nil {
		return composition.Draft{}, err
	}
	for _, d := range r.drafts {
		if d.Status == composition.StatusFinal && match(d) {
			return d, nil
		}
	}
	return composition.Draft{}, composition.ErrNotFound
}

func (r *fakeRepository) Drafts(ctx context.Context, encounterID string) ([]composition.Draft, error) {
	if err := r.check(ctx); err != nil {
		return nil, err
	}
	var out []composition.Draft
	for _, d := range r.drafts {
		if d.EncounterID == encounterID && r.visible(d) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeRepository) Signed(ctx context.Context, encounterID string) ([]composition.Draft, error) {
	all, err := r.Drafts(ctx, encounterID)
	return slices.DeleteFunc(all, func(d composition.Draft) bool { return d.Status != composition.StatusFinal }), err
}

func (r *fakeRepository) SignedOf(ctx context.Context, encounterIDs []string) ([]composition.Draft, error) {
	var out []composition.Draft
	for _, id := range encounterIDs {
		ds, err := r.Signed(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, ds...)
	}
	return out, nil
}

func (r *fakeRepository) Sign(ctx context.Context, d composition.Draft, at time.Time) (composition.Draft, error) {
	if err := r.check(ctx); err != nil {
		return composition.Draft{}, err
	}
	d.Status, d.UpdatedAt, d.VersionID = composition.StatusFinal, at, d.VersionID+1
	r.drafts[d.ID] = d
	return d, nil
}

func (r *fakeRepository) check(ctx context.Context) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if r.scope == "" {
		return errors.New("no scope set")
	}
	return r.err
}

// fakeEncounters gives the encounter and the caller's attendance by caller account; attendances[i] answers the
// i-th call (the last one every later call), so a revocation can land between two calls.
type fakeEncounters struct {
	err         error
	f           *fixture
	attendances map[string][]appointment.Attendance
	calls       map[string]int
	shared      map[string]encounter.Encounter // another clinic's visits shared with the caller, by account + encounter id
	enc         encounter.Encounter
}

func (e *fakeEncounters) Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, appointment.Attendance{}, err
	}
	if e.err != nil {
		return encounter.Encounter{}, appointment.Attendance{}, e.err
	}
	all, ok := e.attendances[c.AccountID]
	if !ok || id != e.enc.ID {
		return encounter.Encounter{}, appointment.Attendance{}, encounter.ErrNotFound
	}
	n := e.calls[c.AccountID]
	e.calls[c.AccountID]++
	e.f.repo.scope = c.OrganizationID // the encounter's service scopes the transaction to the caller
	if c.OrganizationID == "" {
		e.f.repo.scope = patientScope + c.PatientID
	}
	return e.enc, all[min(n, len(all)-1)], nil
}

// Shared gives the visits shared with the caller, scoping the transaction to the caller's clinic as encounter does.
func (e *fakeEncounters) Shared(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, err
	}
	got, ok := e.shared[c.AccountID+id]
	if !ok {
		return encounter.Encounter{}, encounter.ErrNotFound
	}
	e.f.repo.scope = c.OrganizationID
	return got, nil
}

// fakeTranscripts gives the stored segments, and counts the calls.
type fakeTranscripts struct {
	err      error
	segments []transcript.Event
	calls    int
}

func (t *fakeTranscripts) Export(ctx context.Context, _ appointment.Caller, _ string) ([]transcript.Event, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	t.calls++
	return slices.Clone(t.segments), t.err
}

// fakeStore holds objects in memory.
type fakeStore struct {
	putErr  error
	objects map[string][]byte
	mu      sync.Mutex
}

func (s *fakeStore) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	if ctx.Value(txKey{}) != nil {
		return errors.New("a put inside a transaction")
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) != size {
		return errors.New("short body")
	}
	if s.putErr != nil {
		return s.putErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = b
	return nil
}

func (s *fakeStore) GetRange(_ context.Context, key string, from, to int64) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.objects[key]
	if !ok {
		return nil, errors.New("no such key")
	}
	return io.NopCloser(bytes.NewReader(b[from : to+1])), nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

// fakePublisher records what was put on each encounter's stream.
type fakePublisher struct {
	err       error
	envelopes []wire.Envelope
	streams   []string
}

func (p *fakePublisher) Publish(ctx context.Context, encounterID string, b []byte) error {
	if ctx.Value(txKey{}) != nil {
		return errors.New("a publish inside a transaction")
	}
	env, err := wire.Decode(b)
	if err != nil {
		return err
	}
	p.envelopes, p.streams = append(p.envelopes, env), append(p.streams, encounterID)
	return p.err
}

// fakeJobs records the e-mail jobs enqueued, each "<appointment> <clinic>", and the statistics' refreshes, each
// "<clinic> <instant>", in the transaction.
type fakeJobs struct {
	err       error
	signed    []string
	refreshes []string
}

func (j *fakeJobs) EnqueueStatsRefresh(ctx context.Context, org string, day time.Time) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	j.refreshes = append(j.refreshes, org+" "+day.UTC().Format(time.RFC3339))
	return nil
}

func (j *fakeJobs) EnqueueSigned(ctx context.Context, appointmentID, org string) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	j.signed = append(j.signed, appointmentID+" "+org)
	return j.err
}

// fakeConditions keeps the Conditions recorded in a transaction; a rollback drops them.
type fakeConditions struct {
	err      error
	recorded []condition.Condition
}

func (c *fakeConditions) Record(ctx context.Context, cs []condition.Condition) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if c.err != nil {
		return c.err
	}
	c.recorded = append(c.recorded, cs...)
	return nil
}

type fakeAuditor struct {
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	a.events = append(a.events, e)
	return nil
}

// fixture is a Service on fakes: the encounter encA of clinic A's appointment with roleA1 for patientA, whose
// practitioner (practAcc) attends it with an active consent, as do its other practitioner, its admin and its
// patient in their roles; and the two stored segments seg1 and seg2.
type fixture struct {
	svc         *composition.Service
	repo        *fakeRepository
	encounters  *fakeEncounters
	transcripts *fakeTranscripts
	drafter     *llmtest.Drafter
	store       *fakeStore
	publisher   *fakePublisher
	jobs        *fakeJobs
	conditions  *fakeConditions
	audit       *fakeAuditor
	logs        *bytes.Buffer
	mu          sync.Mutex
}

const encA = "ec000000-0000-4000-8000-0000000000a1"

func attendance(practitioner, patient, active bool) appointment.Attendance {
	return appointment.Attendance{
		Appointment: appointment.Appointment{
			ID: apptA, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA, Start: now.Add(-10 * time.Minute),
		},
		ConsentActive: active, TranscriptionConsent: active, Practitioner: practitioner, Patient: patient,
	}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	kek, err := envelope.NewKEK("kek-test", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		repo:       &fakeRepository{templates: map[string]composition.Template{}, drafts: map[string]composition.Draft{}},
		drafter:    &llmtest.Drafter{},
		store:      &fakeStore{objects: map[string][]byte{}},
		publisher:  &fakePublisher{},
		jobs:       &fakeJobs{},
		conditions: &fakeConditions{},
		audit:      &fakeAuditor{},
		logs:       &bytes.Buffer{},
		transcripts: &fakeTranscripts{segments: []transcript.Event{
			{EncounterID: encA, SegmentID: seg1, Speaker: transcript.SpeakerPatient, Lang: "de", Original: "Placeholder complaint", IsFinal: true, TsStart: now.Add(-5 * time.Minute)},
			{EncounterID: encA, SegmentID: seg2, Speaker: transcript.SpeakerPractitioner, Lang: "bs", Original: "Placeholder question", IsFinal: true, TsStart: now.Add(-4 * time.Minute)},
		}},
	}
	f.encounters = &fakeEncounters{
		f: f, calls: map[string]int{},
		enc: encounter.Encounter{ID: encA, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1, Status: encounter.StatusInProgress},
		attendances: map[string][]appointment.Attendance{
			practAcc:  {attendance(true, false, true)},
			pract2Acc: {attendance(false, false, true)},
			adminAcc:  {attendance(false, false, true)},
			accountA:  {attendance(false, true, true)},
		},
	}
	f.svc = composition.NewService(composition.Ports{
		Repo: f.repo, Encounters: f.encounters, Transcripts: f.transcripts, Drafter: f.drafter, Store: f.store,
		Publisher: f.publisher, Jobs: f.jobs, Stats: f.jobs, Conditions: f.conditions, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now}, KEK: kek,
		Log: slog.New(slog.NewTextHandler(f.logs, nil)),
	})
	return f
}

// template stores the findings report's placeholder template (content()) in clinic A and returns it.
func (f *fixture) template(t *testing.T) composition.Template {
	t.Helper()
	tpl, err := f.svc.CreateTemplate(t.Context(), pract, content())
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	return tpl
}

// filled is the drafter's script: the findings from seg1, the tests from seg2.
func filled() map[string]llm.DraftField {
	return map[string]llm.DraftField{
		"findings": {Value: "Placeholder findings", SourceIDs: []string{seg1}},
		"tests":    {Value: "Placeholder test one\nPlaceholder test two", SourceIDs: []string{seg2}},
	}
}

// codes returns each audit event's subtype, outcome and entity reference.
func (f *fixture) codes() []string {
	out := make([]string, len(f.audit.events))
	for i, e := range f.audit.events {
		out[i] = e.Subtype.Code + " " + strconv.Itoa(int(e.Outcome)) + " " + e.EntityReference
	}
	return out
}
