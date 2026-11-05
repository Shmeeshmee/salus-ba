package videosession_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Synthetic ids and names.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	accountA  = "acc00000-0000-4000-8000-0000000000aa"
	accountB  = "acc00000-0000-4000-8000-0000000000bb"
	roleA1    = "r1000000-0000-4000-8000-0000000000a1"
	practA1   = "p1000000-0000-4000-8000-0000000000a1"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	pract2Acc = "ac200000-0000-4000-8000-0000000000a2"
	adminAcc  = "ad000000-0000-4000-8000-0000000000ad"
	apptA     = "e1000000-0000-4000-8000-0000000000a1"
	apptA2    = "e3000000-0000-4000-8000-0000000000a2"
	publicURL = "ws://localhost:7880"
)

var (
	start      = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC) // the appointment, 09:00 to 09:30
	end        = start.Add(30 * time.Minute)
	now        = start.Add(-5 * time.Minute) // inside the window
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	other      = appointment.Caller{AccountID: accountB, PatientID: "bb000000-0000-4000-8000-0000000000bb"}
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: "p2000000-0000-4000-8000-0000000000a2"}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	adminB     = appointment.Caller{AccountID: "ad000000-0000-4000-8000-0000000000bd", OrganizationID: clinicB, Admin: true}
	anonymous  = appointment.Caller{}
	errBackend = errors.New("the backend is down")
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction"; when fn fails it drops what was recorded in it, as a rollback
// does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, sessions, jobs, finished := len(t.f.audit.events), clone(t.f.repo.sessions), len(t.f.jobs.enqueued), len(t.f.encounters.finished)
	t.f.repo.patient, t.f.repo.clinic = "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.sessions = t.f.audit.events[:events], sessions
		t.f.jobs.enqueued, t.f.encounters.finished = t.f.jobs.enqueued[:jobs], t.f.encounters.finished[:finished]
	}
	return err
}

func clone(m map[string]videosession.Session) map[string]videosession.Session {
	out := make(map[string]videosession.Session, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// fakeAppointments gives each caller (by account) their attendance of each appointment.
type fakeAppointments struct {
	err        error
	attendance map[string]appointment.Attendance // by account id + appointment id
}

func (a *fakeAppointments) Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if ctx.Value(txKey{}) == nil {
		return appointment.Attendance{}, errors.New("Attendance outside a transaction")
	}
	if a.err != nil {
		return appointment.Attendance{}, a.err
	}
	att, ok := a.attendance[c.AccountID+id]
	if !ok {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return att, nil
}

// fakeRepository holds the sessions by appointment, as Open stores them.
type fakeRepository struct {
	err      error
	endErr   error
	sessions map[string]videosession.Session
	patient  string // set by SetPatient
	clinic   string // set by SetClinic
	opened   []videosession.Session
	next     int
}

func (r *fakeRepository) SetPatient(_ context.Context, id string) error {
	r.patient = id
	return nil
}

func (r *fakeRepository) SetClinic(_ context.Context, id string) error {
	r.clinic = id
	return nil
}

func (r *fakeRepository) Open(ctx context.Context, s videosession.Session) (videosession.Session, bool, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, false, errors.New("Open outside a transaction")
	}
	if r.err != nil {
		return videosession.Session{}, false, r.err
	}
	r.opened = append(r.opened, s)
	if got, ok := r.sessions[s.AppointmentID]; ok {
		return got, false, nil
	}
	r.next++
	s.ID = "5e000000-0000-4000-8000-00000000000" + string(rune('0'+r.next))
	r.sessions[s.AppointmentID] = s
	return s, true, nil
}

// ByID gives the session in the clinic's scope.
func (r *fakeRepository) ByID(ctx context.Context, id string) (videosession.Session, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, errors.New("ByID outside a transaction")
	}
	if r.err != nil {
		return videosession.Session{}, r.err
	}
	for _, s := range r.sessions {
		if s.ID == id && r.clinic != "" && s.OrganizationID == r.clinic {
			return s, nil
		}
	}
	return videosession.Session{}, videosession.ErrNotFound
}

func (r *fakeRepository) End(ctx context.Context, id string, reason videosession.EndReason, at time.Time) (videosession.Session, bool, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, false, errors.New("End outside a transaction")
	}
	if r.endErr != nil {
		return videosession.Session{}, false, r.endErr
	}
	for k, s := range r.sessions {
		if s.ID == id && s.OrganizationID == r.clinic && s.EndedAt.IsZero() {
			s.EndedAt, s.EndReason = at, reason
			r.sessions[k] = s
			return s, true, nil
		}
	}
	return videosession.Session{}, false, nil
}

func (r *fakeRepository) DeleteEndedBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("not used by the service")
}

// fakeProvider records the rooms created and deleted and the grants signed; its token is
// "token:<identity>"; the participants of every room are in.
type fakeProvider struct {
	err       error
	tokenErr  error
	deleteErr error
	listErr   error
	rooms     []videosession.Room
	grants    []videosession.Grant
	deleted   []string
	listed    []string
	in        []string
	inTx      bool // a call was made inside a transaction
}

func (p *fakeProvider) CreateRoom(ctx context.Context, r videosession.Room) error {
	p.inTx = p.inTx || ctx.Value(txKey{}) != nil
	p.rooms = append(p.rooms, r)
	return p.err
}

func (p *fakeProvider) Token(g videosession.Grant) (string, error) {
	p.grants = append(p.grants, g)
	return "token:" + g.Identity, p.tokenErr
}

func (p *fakeProvider) RemoveParticipant(context.Context, string, string) error {
	return errors.New("not used by Join")
}

func (p *fakeProvider) DeleteRoom(ctx context.Context, room string) error {
	p.inTx = p.inTx || ctx.Value(txKey{}) != nil
	p.deleted = append(p.deleted, room)
	return p.deleteErr
}

func (p *fakeProvider) Participants(ctx context.Context, room string) ([]string, error) {
	p.inTx = p.inTx || ctx.Value(txKey{}) != nil
	p.listed = append(p.listed, room)
	return p.in, p.listErr
}

// fakeJobs records the window-close jobs enqueued in a transaction.
type fakeJobs struct {
	err      error
	enqueued []string
}

func (j *fakeJobs) EnqueueWindowClose(ctx context.Context, sessionID, org string, at time.Time) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("EnqueueWindowClose outside a transaction")
	}
	if j.err != nil {
		return j.err
	}
	j.enqueued = append(j.enqueued, sessionID+" "+org+" "+at.Format(time.RFC3339))
	return nil
}

// fakeEncounters records the encounters finished in a transaction, in the clinic's scope.
type fakeEncounters struct {
	err      error
	repo     *fakeRepository
	finished []string
}

func (e *fakeEncounters) Finish(ctx context.Context, agent, appointmentID string, at time.Time, patientJoined bool) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("Finish outside a transaction")
	}
	if e.err != nil {
		return e.err
	}
	e.finished = append(e.finished, fmt.Sprintf("%s %s %s in %s, patient joined %t", agent, appointmentID, at.Format(time.RFC3339), e.repo.clinic, patientJoined))
	return nil
}

// fakePatients gives patient A's profile in a transaction.
type fakePatients struct {
	err    error
	lookup []string
}

func (p *fakePatients) Lookup(ctx context.Context, id string) (patient.Patient, error) {
	p.lookup = append(p.lookup, id)
	if ctx.Value(txKey{}) == nil {
		return patient.Patient{}, errors.New("Lookup outside a transaction")
	}
	if p.err != nil {
		return patient.Patient{}, p.err
	}
	if id != patientA {
		return patient.Patient{}, patient.ErrNotFound
	}
	return patient.Patient{ID: patientA, AccountID: accountA, GivenName: "Ada", FamilyName: "Example"}, nil
}

// fakePractitioners is the public directory: practitioner A1 only.
type fakePractitioners struct {
	err   error
	asked [][]string
}

func (p *fakePractitioners) PublicNames(ctx context.Context, ids []string) ([]practitioner.Practitioner, error) {
	if ctx.Value(txKey{}) != nil {
		return nil, errors.New("PublicNames inside the join's transaction")
	}
	p.asked = append(p.asked, ids)
	if p.err != nil {
		return nil, p.err
	}
	if slices.Contains(ids, practA1) {
		return []practitioner.Practitioner{{ID: practA1, GivenName: "Cy", FamilyName: "Example", Active: true}}, nil
	}
	return nil, nil
}

// fakeAuditor records events; recording outside a transaction is an error.
type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("audit outside a transaction")
	}
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

type fixture struct {
	jobs          *fakeJobs
	encounters    *fakeEncounters
	appts         *fakeAppointments
	repo          *fakeRepository
	provider      *fakeProvider
	patients      *fakePatients
	practitioners *fakePractitioners
	audit         *fakeAuditor
	svc           *videosession.Service
}

// booked is appointment A (patient A with role A1 of clinic A, 09:00 to 09:30), booked.
func booked() appointment.Appointment {
	return appointment.Appointment{
		ID: apptA, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA,
		Start: start, End: end, Status: appointment.StatusBooked, VersionID: 1,
	}
}

// newFixture is appointment A at the time at, with patient A's active consent: its patient, its practitioner
// and the clinic's admin see it, nobody else.
func newFixture(at time.Time) *fixture {
	a := booked()
	f := &fixture{
		appts: &fakeAppointments{attendance: map[string]appointment.Attendance{
			accountA + apptA: {Appointment: a, PractitionerID: practA1, Level: access.Full, Profile: access.Full, ConsentActive: true, Patient: true},
			practAcc + apptA: {Appointment: a, PractitionerID: practA1, Level: access.Full, Profile: access.Full, ConsentActive: true, Practitioner: true},
			adminAcc + apptA: {Appointment: a, PractitionerID: practA1, Level: access.Full, Profile: access.Name, ConsentActive: true},
		}},
		repo:     &fakeRepository{sessions: map[string]videosession.Session{}},
		provider: &fakeProvider{}, patients: &fakePatients{}, practitioners: &fakePractitioners{}, audit: &fakeAuditor{},
		jobs: &fakeJobs{},
	}
	f.encounters = &fakeEncounters{repo: f.repo}
	f.svc = videosession.NewService(videosession.Ports{
		Repo: f.repo, Provider: f.provider, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: at},
		Appointments: f.appts, Patients: f.patients, Practitioners: f.practitioners, PublicURL: publicURL,
		Jobs: f.jobs, Encounters: f.encounters,
	})
	return f
}

// change changes appointment A as every caller sees it.
func (f *fixture) change(fn func(*appointment.Attendance)) {
	for k, att := range f.appts.attendance {
		fn(&att)
		f.appts.attendance[k] = att
	}
}

// joinEvent is the audit event of a join by c on entity ("" for none) with outcome.
func joinEvent(c appointment.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	purpose := auditevent.CodePATRQT
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
	}
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeOperation},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:          auditevent.ActionExecute,
		Outcome:         outcome,
		AgentAccountID:  c.AccountID,
		OrganizationID:  c.OrganizationID,
		EntityReference: entity,
	}
}

// patientRead is the audit event of the practitioner c's read of patient A's name.
func patientRead(c appointment.Caller) auditevent.AuditEvent {
	e := joinEvent(c, "Patient/"+patientA, auditevent.OutcomeSuccess)
	e.Subtype.Code, e.Action = auditevent.CodeRead, auditevent.ActionRead
	return e
}
