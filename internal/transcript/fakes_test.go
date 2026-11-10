package transcript_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// Synthetic ids.
const (
	clinicA    = "a0000000-0000-4000-8000-00000000000a"
	patientA   = "aa000000-0000-4000-8000-0000000000aa"
	roleA1     = "a1000000-0000-4000-8000-0000000000a1"
	roleA2     = "a2000000-0000-4000-8000-0000000000a2"
	apptA      = "e1000000-0000-4000-8000-0000000000a1"
	practAcc   = "ac100000-0000-4000-8000-0000000000a1"
	pract2Acc  = "ac200000-0000-4000-8000-0000000000a2"
	adminAcc   = "ad000000-0000-4000-8000-0000000000ad"
	accountA   = "acc00000-0000-4000-8000-0000000000aa"
	unknownID  = "00000000-0000-4000-8000-000000000000"
	consumer   = "consult_stream"
	segmentID2 = "7d1e2f30-1a2b-4c3d-8e4f-5a6b7c8d9e10"
)

var (
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: "p1000000-0000-4000-8000-0000000000a1"}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: "p2000000-0000-4000-8000-0000000000a2"}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	errBackend = errors.New("the database is down")
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction": the scope set in it ends with it, and when fn fails what was
// stored and audited in it is dropped, as a rollback does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, segments := len(t.f.audit.events), maps.Clone(t.f.repo.segments)
	defer func() { t.f.scope = "" }()
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.segments = t.f.audit.events[:events], segments
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeRepository holds the stored segments by id, written and read only in a clinic's scope.
type fakeRepository struct {
	f        *fixture
	err      error
	segments map[string]transcript.Event
}

func (r *fakeRepository) Insert(ctx context.Context, e transcript.Event) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	if r.err != nil {
		return false, r.err
	}
	if r.f.scope == "" {
		return false, errors.New("insert outside a clinic's scope")
	}
	if _, ok := r.segments[e.SegmentID]; ok {
		return false, nil
	}
	r.segments[e.SegmentID] = e
	return true, nil
}

func (r *fakeRepository) ByEncounter(ctx context.Context, id string) ([]transcript.Event, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.f.scope == "" {
		return nil, errors.New("read outside a clinic's scope")
	}
	var out []transcript.Event
	for _, e := range r.segments {
		if e.EncounterID == id {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b transcript.Event) int { return a.TsStart.Compare(b.TsStart) })
	return out, nil
}

// fakeStream holds each encounter's entries in order, and the waits it was asked to block for.
type fakeStream struct {
	err     error
	entries map[string][]transcript.Entry
	waits   []time.Duration
}

func (s *fakeStream) Read(_ context.Context, encounterID, after string, wait time.Duration) ([]transcript.Entry, error) {
	s.waits = append(s.waits, wait)
	if s.err != nil {
		return nil, s.err
	}
	var out []transcript.Entry
	for _, e := range s.entries[encounterID] {
		if after == "0" || streamAfter(e.ID, after) {
			out = append(out, e)
		}
	}
	return out, nil
}

// streamAfter compares the fake's ids, "<n>-0" with n of one digit.
func streamAfter(id, after string) bool { return strings.Compare(id, after) > 0 }

// fakeEncounters gives encounter A (patient A, role A1, clinic A): Lookup to the system path, scoping the
// transaction to its clinic; Access with each caller's attendance (by account id).
type fakeEncounters struct {
	f          *fixture
	err        error
	encounters map[string]encounter.Encounter
	attendance map[string]appointment.Attendance
	accessed   int
}

func (e *fakeEncounters) Lookup(ctx context.Context, id string) (encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, err
	}
	if e.err != nil {
		return encounter.Encounter{}, e.err
	}
	got, ok := e.encounters[id]
	if !ok {
		return encounter.Encounter{}, encounter.ErrNotFound
	}
	e.f.scope = got.OrganizationID
	return got, nil
}

func (e *fakeEncounters) Access(ctx context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error) {
	e.accessed++
	if e.err != nil {
		return encounter.Encounter{}, appointment.Attendance{}, e.err
	}
	got, ok := e.encounters[id]
	att, seen := e.attendance[c.AccountID]
	if !ok || !seen || att.Level == access.None {
		return encounter.Encounter{}, appointment.Attendance{}, encounter.ErrNotFound
	}
	if ctx.Value(txKey{}) != nil && c.OrganizationID != "" {
		e.f.scope = c.OrganizationID
	}
	if ctx.Value(txKey{}) != nil && c.OrganizationID == "" {
		e.f.scope = "patient:" + c.PatientID
	}
	return got, att, nil
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
// fakeClinics answers whether clinic A shares its transcripts with its patients (copy), and counts the calls.
type fakeClinics struct {
	err   error
	calls int
	copy  bool
}

func (c *fakeClinics) TranscriptCopy(ctx context.Context, org string) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	c.calls++
	return c.copy && org == clinicA, c.err
}

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
	stream     *fakeStream
	encounters *fakeEncounters
	consents   *fakeConsents
	clinics    *fakeClinics
	audit      *fakeAuditor
	svc        *transcript.Service
	scope      string // the clinic the current transaction is scoped to
}

// newFixture is clinic A's encounter of appointment A (patient A with role A1, started at the segments' start), with patient
// A's active consent: its practitioner sees all of it, another practitioner of the clinic nothing, the admin
// and the patient the appointment but not what was said.
func newFixture() *fixture {
	enc := encounter.Encounter{
		ID: encounterID, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1,
		Status: encounter.StatusInProgress, PeriodStart: start,
	}
	appt := appointment.Appointment{ID: apptA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1, Start: start}
	f := &fixture{stream: &fakeStream{entries: map[string][]transcript.Entry{}}, consents: &fakeConsents{state: access.Consent{Active: true, Transcription: true}}, clinics: &fakeClinics{}, audit: &fakeAuditor{}}
	f.repo = &fakeRepository{f: f, segments: map[string]transcript.Event{}}
	f.encounters = &fakeEncounters{f: f, encounters: map[string]encounter.Encounter{encounterID: enc}, attendance: map[string]appointment.Attendance{
		practAcc:  {Appointment: appt, Level: access.Full, ConsentActive: true, Practitioner: true},
		pract2Acc: {Appointment: appt, Level: access.Full, ConsentActive: true},
		adminAcc:  {Appointment: appt, Level: access.Full, ConsentActive: true},
		accountA:  {Appointment: appt, Level: access.Full, ConsentActive: true, Patient: true},
	}}
	f.svc = transcript.NewService(transcript.Ports{
		Repo: f.repo, Stream: f.stream, Encounters: f.encounters, Consents: f.consents, Sharing: f.clinics, Audit: f.audit, Tx: fakeTransactor{f: f},
	})
	return f
}

// revoke is the patient's revocation: the consent inactive, and the practitioner's attendance of the
// appointment, which started before it, at the name level.
func (f *fixture) revoke() {
	f.consents.state = access.Consent{RevokedAt: start.Add(10 * time.Minute)}
	for acc, att := range f.encounters.attendance {
		att.ConsentActive, att.RevokedAt, att.Level = false, start.Add(10*time.Minute), access.Name
		f.encounters.attendance[acc] = att
	}
}

func checkEvents(t *testing.T, name string, a *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(a.events, want) {
		t.Errorf("%s: audited %+v,\nwant %+v", name, a.events, want)
	}
}

// systemEvent is the stream consumer's audit event of a segment's write to encounter A.
func systemEvent(outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          auditevent.ActionCreate,
		Outcome:         outcome,
		AgentName:       consumer,
		OrganizationID:  clinicA,
		EntityReference: "Encounter/" + encounterID,
		EntityName:      "transcript_segment",
		EntityCount:     count,
	}
}

// callerEvent is c's audit event of interaction on encounter id ("" for none disclosed): a read of the
// transcript (with the segments returned) or the subscription to the stream (an operation).
func callerEvent(c appointment.Caller, interaction, id string, outcome auditevent.Outcome, count int64) auditevent.AuditEvent {
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         auditevent.ActionRead,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if c.OrganizationID == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	if interaction == auditevent.CodeOperation {
		e.Action = auditevent.ActionExecute
	}
	if id != "" {
		e.EntityReference = "Encounter/" + id
	}
	if interaction == auditevent.CodeRead && outcome == auditevent.OutcomeSuccess {
		e.EntityName, e.EntityCount = "transcript_segment", count
	}
	return e
}
