package videosession_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
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
	started := len(t.f.encounters.started)
	t.f.repo.patient, t.f.repo.clinic, t.f.repo.room, t.f.repo.narrowed = "", "", "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.sessions = t.f.audit.events[:events], sessions
		t.f.jobs.enqueued, t.f.encounters.finished = t.f.jobs.enqueued[:jobs], t.f.encounters.finished[:finished]
		t.f.encounters.started = t.f.encounters.started[:started]
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
	err             error
	practitionerErr error
	attendance      map[string]appointment.Attendance // by account id + appointment id
	repo            *fakeRepository                   // whose scope Lookup reads in
	hidden          bool                              // Lookup does not see the appointment in the clinic's scope (after a revocation); the patient's does
}

func (a *fakeAppointments) Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if ctx.Value(txKey{}) == nil {
		return appointment.Attendance{}, errors.New("Attendance outside a transaction")
	}
	if a.err != nil {
		return appointment.Attendance{}, a.err
	}
	att, ok := a.attendance[c.AccountID+id]
	if !ok || (a.hidden && c.OrganizationID != "") {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return att, nil
}

// Lookup gives the appointment as its practitioner sees it, in a transaction; ErrNotFound when hidden.
func (a *fakeAppointments) Lookup(ctx context.Context, id string) (appointment.Appointment, error) {
	if ctx.Value(txKey{}) == nil {
		return appointment.Appointment{}, errors.New("Lookup outside a transaction")
	}
	if a.err != nil {
		return appointment.Appointment{}, a.err
	}
	att, ok := a.attendance[practAcc+id]
	if !ok || (a.hidden && a.repo.patient != patientA) {
		return appointment.Appointment{}, appointment.ErrNotFound
	}
	return att.Appointment, nil
}

// LookupPractitioner gives the practitioner of the appointment's role in a transaction in the clinic's scope:
// when it is not hidden, or narrowed to the appointment (SetAppointment).
func (a *fakeAppointments) LookupPractitioner(ctx context.Context, id string) (string, error) {
	if ctx.Value(txKey{}) == nil {
		return "", errors.New("LookupPractitioner outside a transaction")
	}
	if a.practitionerErr != nil {
		return "", a.practitionerErr
	}
	att, ok := a.attendance[accountA+id]
	if !ok || a.repo.clinic != att.Appointment.OrganizationID || (a.hidden && a.repo.narrowed != id) {
		return "", appointment.ErrNotFound
	}
	return att.PractitionerID, nil
}

// fakeConsents gives the state of patient A's consent with clinic A in a transaction: active unless revoked.
type fakeConsents struct {
	err     error
	asked   []string
	revoked bool
}

func (c *fakeConsents) State(ctx context.Context, patientID, org string) (access.Consent, error) {
	if ctx.Value(txKey{}) == nil {
		return access.Consent{}, errors.New("State outside a transaction")
	}
	c.asked = append(c.asked, patientID+" "+org)
	if c.err != nil {
		return access.Consent{}, c.err
	}
	return access.Consent{Active: !c.revoked && patientID == patientA && org == clinicA}, nil
}

// fakeRepository holds the sessions by appointment, as Open stores them.
type fakeRepository struct {
	// practitionerJoinsAt records the practitioner's join when End is called, as a join that committed after
	// the end's reason was chosen and before its UPDATE.
	practitionerJoinsAt time.Time
	err                 error
	endErr              error
	joinedErr           error
	sessions            map[string]videosession.Session
	patient             string // set by SetPatient
	clinic              string // set by SetClinic
	room                string // set by SetRoom
	narrowed            string // the appointment set by SetAppointment
	opened              []videosession.Session
	next                int
	// hidden: the clinic's scope does not show the sessions (after a revocation, of the appointments that had
	// not started); the patient's and the room's do.
	hidden bool
	// endedElsewhere ends the session window_closed when End is called, as a concurrent end that committed
	// first: End then changes nothing.
	endedElsewhere bool
}

func (r *fakeRepository) SetRoom(_ context.Context, room string) error {
	r.room = room
	return nil
}

// ByRoom gives the session of the room set by SetRoom.
func (r *fakeRepository) ByRoom(ctx context.Context, room string) (videosession.Session, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, errors.New("ByRoom outside a transaction")
	}
	if r.err != nil {
		return videosession.Session{}, r.err
	}
	for _, s := range r.sessions {
		if s.Room == room && r.room == room {
			return s, nil
		}
	}
	return videosession.Session{}, videosession.ErrNotFound
}

// Joined sets the role's join, as the SQL does: only when not set and the session has not ended.
func (r *fakeRepository) Joined(ctx context.Context, id string, role videosession.Role, at, now time.Time) (videosession.Session, bool, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, false, errors.New("Joined outside a transaction")
	}
	if r.joinedErr != nil {
		return videosession.Session{}, false, r.joinedErr
	}
	for k, s := range r.sessions {
		if s.ID != id || s.Room != r.room || !s.EndedAt.IsZero() {
			continue
		}
		joined := &s.PatientJoinedAt
		if role == videosession.RolePractitioner {
			joined = &s.PractitionerJoinedAt
		}
		if !joined.IsZero() {
			return videosession.Session{}, false, nil
		}
		*joined = at
		r.sessions[k] = s
		return s, true, nil
	}
	return videosession.Session{}, false, nil
}

func (r *fakeRepository) SetAppointment(_ context.Context, id string) error {
	r.narrowed = id
	return nil
}

// Removed records the removal's reason in the room's scope, unless one is recorded.
func (r *fakeRepository) Removed(ctx context.Context, id string, reason videosession.Reason, _ time.Time) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("Removed outside a transaction")
	}
	if r.err != nil {
		return r.err
	}
	for k, s := range r.sessions {
		if s.ID == id && s.Room == r.room && s.RemovalReason == "" {
			s.RemovalReason = reason
			r.sessions[k] = s
		}
	}
	return nil
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
		if s.ID == id && (r.inClinic(s) || r.room == s.Room) {
			return s, nil
		}
	}
	return videosession.Session{}, videosession.ErrNotFound
}

// ByAppointment gives the appointment's session in the patient's scope (every session is patient A's), the
// clinic's, or the clinic's narrowed to the appointment once its practitioner joined.
func (r *fakeRepository) ByAppointment(ctx context.Context, id string) (videosession.Session, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, errors.New("ByAppointment outside a transaction")
	}
	if r.err != nil {
		return videosession.Session{}, r.err
	}
	s, ok := r.sessions[id]
	attended := r.clinic == s.OrganizationID && r.narrowed == id && !s.PractitionerJoinedAt.IsZero()
	if !ok || (r.patient != patientA && !r.inClinic(s) && !attended) {
		return videosession.Session{}, videosession.ErrNotFound
	}
	return s, nil
}

// End ends the session as the SQL does: only one not ended yet, and practitioner_absent only while the
// practitioner has not joined.
func (r *fakeRepository) End(ctx context.Context, id string, reason videosession.EndReason, at time.Time) (videosession.Session, bool, error) {
	if ctx.Value(txKey{}) == nil {
		return videosession.Session{}, false, errors.New("End outside a transaction")
	}
	if r.endErr != nil {
		return videosession.Session{}, false, r.endErr
	}
	for k, s := range r.sessions {
		inScope := r.inClinic(s) || (r.room != "" && r.room == s.Room)
		if s.ID == id && inScope && s.EndedAt.IsZero() && r.endedElsewhere {
			s.EndedAt, s.EndReason = at.Add(-time.Second), videosession.EndWindowClosed
			r.sessions[k] = s
			return videosession.Session{}, false, nil
		}
		if s.ID == id && !r.practitionerJoinsAt.IsZero() && s.PractitionerJoinedAt.IsZero() {
			s.PractitionerJoinedAt = r.practitionerJoinsAt
			r.sessions[k] = s
		}
		absentButJoined := reason == videosession.EndPractitionerAbsent && !s.PractitionerJoinedAt.IsZero()
		if s.ID == id && inScope && s.EndedAt.IsZero() && !absentButJoined {
			s.EndedAt, s.EndReason = at, reason
			r.sessions[k] = s
			return s, true, nil
		}
	}
	return videosession.Session{}, false, nil
}

// inClinic reports whether the clinic's scope shows s.
func (r *fakeRepository) inClinic(s videosession.Session) bool {
	return r.clinic != "" && s.OrganizationID == r.clinic && !r.hidden
}

// ClosingAfter gives, in the patient's scope (every session is patient A's), the sessions of the clinic org
// whose window closes after after, by appointment id.
func (r *fakeRepository) ClosingAfter(ctx context.Context, org string, after time.Time) ([]videosession.Session, error) {
	if ctx.Value(txKey{}) == nil {
		return nil, errors.New("ClosingAfter outside a transaction")
	}
	if r.err != nil {
		return nil, r.err
	}
	var out []videosession.Session
	for _, s := range r.sessions {
		if r.patient == patientA && s.OrganizationID == org && s.ClosesAt.After(after) {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b videosession.Session) int { return strings.Compare(a.AppointmentID, b.AppointmentID) })
	return out, nil
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
	removeErr error
	parseErr  error
	event     videosession.Event // what ParseWebhook reads
	rooms     []videosession.Room
	grants    []videosession.Grant
	deleted   []string
	listed    []string
	in        []string
	removed   []string
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

// RemoveParticipant records the removal and, unless it fails, takes the identity out of the room.
func (p *fakeProvider) RemoveParticipant(ctx context.Context, room, identity string) error {
	p.inTx = p.inTx || ctx.Value(txKey{}) != nil
	p.removed = append(p.removed, room+" "+identity)
	if p.removeErr == nil {
		p.in = slices.DeleteFunc(p.in, func(id string) bool { return id == identity })
	}
	return p.removeErr
}

func (p *fakeProvider) DeleteRoom(ctx context.Context, room string) error {
	p.inTx = p.inTx || ctx.Value(txKey{}) != nil
	p.deleted = append(p.deleted, room)
	return p.deleteErr
}

// ParseWebhook reads the event when the body is "body" and the authorization "signed".
func (p *fakeProvider) ParseWebhook(body []byte, authorization string) (videosession.Event, error) {
	if string(body) != "body" || authorization != "signed" {
		return videosession.Event{}, videosession.ErrInvalidSignature
	}
	return p.event, p.parseErr
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

// fakeEncounters records the encounters started and finished in a transaction, in the clinic's scope.
type fakeEncounters struct {
	err      error
	startErr error
	repo     *fakeRepository
	finished []string
	started  []string
}

func (e *fakeEncounters) Start(ctx context.Context, agent, appointmentID string, at time.Time) (encounter.Encounter, error) {
	if ctx.Value(txKey{}) == nil {
		return encounter.Encounter{}, errors.New("Start outside a transaction")
	}
	if e.startErr != nil {
		return encounter.Encounter{}, e.startErr
	}
	e.started = append(e.started, fmt.Sprintf("%s %s %s in %s", agent, appointmentID, at.Format(time.RFC3339Nano), e.repo.clinic))
	return encounter.Encounter{AppointmentID: appointmentID, PeriodStart: at, Status: encounter.StatusInProgress}, nil
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
	consents      *fakeConsents
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
		jobs: &fakeJobs{}, consents: &fakeConsents{},
	}
	f.encounters = &fakeEncounters{repo: f.repo}
	f.appts.repo = f.repo
	f.svc = videosession.NewService(videosession.Ports{
		Repo: f.repo, Provider: f.provider, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: at},
		Appointments: f.appts, Patients: f.patients, Practitioners: f.practitioners, PublicURL: publicURL,
		Jobs: f.jobs, Encounters: f.encounters, Consents: f.consents,
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
