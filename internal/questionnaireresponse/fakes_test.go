package questionnaireresponse_test

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	patientB  = "bb000000-0000-4000-8000-0000000000bb"
	accountA  = "acc00000-0000-4000-8000-0000000000aa"
	accountB  = "acc00000-0000-4000-8000-0000000000bb"
	roleA1    = "r1000000-0000-4000-8000-0000000000a1"
	roleA2    = "r2000000-0000-4000-8000-0000000000a2"
	practA1   = "p1000000-0000-4000-8000-0000000000a1"
	practA2   = "p2000000-0000-4000-8000-0000000000a2"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	pract2Acc = "ac200000-0000-4000-8000-0000000000a2"
	adminAcc  = "ad000000-0000-4000-8000-0000000000ad"
	apptA     = "e1000000-0000-4000-8000-0000000000a1" // patient A with role A1, ahead
	apptB     = "e3000000-0000-4000-8000-0000000000b1" // patient B with role A1, ahead
	qActive   = "f1000000-0000-4000-8000-0000000000f1" // clinic A, active
	qDraft    = "f2000000-0000-4000-8000-0000000000f2" // clinic A, the next version's draft
	qRetired  = "f3000000-0000-4000-8000-0000000000f3" // clinic A, retired
	qOther    = "f4000000-0000-4000-8000-0000000000f4" // clinic B, active
	seriesA   = "5a000000-0000-4000-8000-0000000000a1"
	seriesB   = "5b000000-0000-4000-8000-0000000000b1"
	unknownID = "00000000-0000-4000-8000-000000000000"
)

var (
	now        = time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	start      = now.Add(24 * time.Hour) // appointment A's start
	revokedAt  = now.Add(-time.Hour)
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	other      = appointment.Caller{AccountID: accountB, PatientID: patientB}
	noProfile  = appointment.Caller{AccountID: "acc00000-0000-4000-8000-0000000000cc"}
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: practA2}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	adminB     = appointment.Caller{AccountID: "ad000000-0000-4000-8000-0000000000bd", OrganizationID: clinicB, Admin: true}
	errBackend = errors.New("the database is down")
	yes        = true
	answers    = []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{Boolean: &yes}}}}
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction"; when fn fails it drops what was written in it, as a rollback
// does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, responses := len(t.f.audit.events), slices.Clone(t.f.repo.responses)
	t.f.repo.patient, t.f.repo.clinic = "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.responses = t.f.audit.events[:events], responses
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	a.events = append(a.events, e)
	return nil
}

// fakeRepository holds the responses and shows them as row-level security does, roughly: the patient's own,
// or the clinic's. The service must not rest on it alone.
type fakeRepository struct {
	err       error
	responses []questionnaireresponse.Response
	searched  []questionnaireresponse.Filter
	patient   string
	clinic    string
	locked    []string
}

func (r *fakeRepository) visible(x questionnaireresponse.Response) bool {
	if r.patient != "" {
		return x.PatientID == r.patient
	}
	return r.clinic != "" && x.OrganizationID == r.clinic
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	r.patient = id
	return inTx(ctx)
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	r.clinic = id
	return inTx(ctx)
}

func (r *fakeRepository) Insert(ctx context.Context, x questionnaireresponse.Response, at time.Time) (questionnaireresponse.Response, bool, error) {
	if err := inTx(ctx); err != nil {
		return questionnaireresponse.Response{}, false, err
	}
	if r.err != nil {
		return questionnaireresponse.Response{}, false, r.err
	}
	for _, got := range r.responses {
		if got.AppointmentID == x.AppointmentID && got.SeriesID == x.SeriesID && got.Status != questionnaireresponse.StatusStopped {
			return questionnaireresponse.Response{}, false, nil
		}
	}
	x.ID = "c0000000-0000-4000-8000-00000000000" + string(rune('1'+len(r.responses)))
	x.VersionID, x.LastUpdated, x.Status, x.Answers = 1, at, questionnaireresponse.StatusInProgress, nil
	r.responses = append(r.responses, x)
	return x, true, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (questionnaireresponse.Response, error) {
	if err := inTx(ctx); err != nil {
		return questionnaireresponse.Response{}, err
	}
	if r.err != nil {
		return questionnaireresponse.Response{}, r.err
	}
	for _, x := range r.responses {
		if x.ID == id && r.visible(x) {
			return x, nil
		}
	}
	return questionnaireresponse.Response{}, questionnaireresponse.ErrNotFound
}

func (r *fakeRepository) Lock(ctx context.Context, id string) (questionnaireresponse.Response, error) {
	r.locked = append(r.locked, id)
	return r.ByID(ctx, id)
}

func (r *fakeRepository) Stop(ctx context.Context, id string, at time.Time) (questionnaireresponse.Response, error) {
	if err := inTx(ctx); err != nil {
		return questionnaireresponse.Response{}, err
	}
	for i, x := range r.responses {
		if x.ID == id && r.visible(x) {
			x.Status, x.Answers, x.LastUpdated = questionnaireresponse.StatusStopped, nil, at
			x.VersionID++
			r.responses[i] = x
			return x, nil
		}
	}
	return questionnaireresponse.Response{}, questionnaireresponse.ErrNotFound
}

func (r *fakeRepository) SaveAnswers(ctx context.Context, id string, answers []questionnaireresponse.Answer, at time.Time) (questionnaireresponse.Response, error) {
	return r.update(ctx, id, func(x *questionnaireresponse.Response) { x.Answers = answers }, at)
}

func (r *fakeRepository) Complete(ctx context.Context, id string, answers []questionnaireresponse.Answer, at time.Time) (questionnaireresponse.Response, error) {
	return r.update(ctx, id, func(x *questionnaireresponse.Response) {
		x.Answers, x.Status, x.AuthoredAt = answers, questionnaireresponse.StatusCompleted, at
	}, at)
}

// update changes the in-progress response id with set at at, as the patient's update does.
func (r *fakeRepository) update(ctx context.Context, id string, set func(*questionnaireresponse.Response), at time.Time) (questionnaireresponse.Response, error) {
	if err := inTx(ctx); err != nil {
		return questionnaireresponse.Response{}, err
	}
	if r.err != nil {
		return questionnaireresponse.Response{}, r.err
	}
	for i, x := range r.responses {
		if x.ID == id && r.visible(x) && x.Status == questionnaireresponse.StatusInProgress {
			set(&x)
			x.LastUpdated = at
			x.VersionID++
			r.responses[i] = x
			return x, nil
		}
	}
	return questionnaireresponse.Response{}, questionnaireresponse.ErrNotFound
}

func (r *fakeRepository) Search(ctx context.Context, f questionnaireresponse.Filter) ([]questionnaireresponse.Response, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	r.searched = append(r.searched, f)
	if r.err != nil {
		return nil, r.err
	}
	var out []questionnaireresponse.Response
	for _, x := range r.responses {
		if r.visible(x) && (f.AppointmentID == "" || x.AppointmentID == f.AppointmentID) &&
			(f.PatientID == "" || x.PatientID == f.PatientID) && (f.Status == "" || x.Status == f.Status) {
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *fakeRepository) DeleteUnsubmittedBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("not for the service")
}

// fakeAppointments decides each caller's attendance of the appointments as appointment.Service does: through
// access.Appointment, over the patient's consent with the clinic and the practitioner's role.
type fakeAppointments struct {
	err      error
	appts    map[string]appointment.Appointment
	consents map[string]access.Consent // by patient + clinic
	roles    map[string]string         // the role of each practitioner, in clinic A
	asked    []string                  // the appointments whose attendance was asked
}

func (a *fakeAppointments) Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if err := inTx(ctx); err != nil {
		return appointment.Attendance{}, err
	}
	a.asked = append(a.asked, id)
	if a.err != nil {
		return appointment.Attendance{}, a.err
	}
	appt, ok := a.appts[id]
	if !ok {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	consent := a.consents[appt.PatientID+appt.OrganizationID]
	visit := access.Visit{PractitionerRole: appt.PractitionerRoleID, Start: appt.Start}
	facts := access.Facts{Patient: appt.PatientID, Clinic: appt.OrganizationID, Consent: consent, Visits: []access.Visit{visit}}
	v := access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	if c.OrganizationID != "" {
		v = access.Viewer{Role: access.RoleAdmin, Clinic: c.OrganizationID}
		if !c.Admin {
			v.Role, v.PractitionerRole = access.RolePractitioner, a.roles[c.PractitionerID]
		}
	}
	level := access.Appointment(v, facts, visit)
	if level == access.None {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return appointment.Attendance{
		Appointment: appt, PractitionerID: practA1, Level: level, Profile: access.Profile(v, facts),
		ConsentActive: consent.Active, RevokedAt: consent.RevokedAt,
		Patient:      v.Role == access.RolePatient,
		Practitioner: v.Role == access.RolePractitioner && v.PractitionerRole == appt.PractitionerRoleID,
	}, nil
}

// fakeQuestionnaires holds the versions and reads one of the caller's clinic, audited as questionnaire.Service
// does.
type fakeQuestionnaires struct {
	err      error
	audit    *fakeAudit
	versions map[string]questionnaire.Questionnaire
	callers  []questionnaire.Caller
	patients []string // the patients Referenced was asked for
}

// Referenced reads a version for a patient, audited as questionnaire.Service does; the fake shows every
// version (the service has checked the reference).
func (q *fakeQuestionnaires) Referenced(ctx context.Context, c questionnaire.Caller, patientID, id string) (questionnaire.Questionnaire, error) {
	if err := inTx(ctx); err != nil {
		return questionnaire.Questionnaire{}, err
	}
	if q.err != nil {
		return questionnaire.Questionnaire{}, q.err
	}
	q.callers, q.patients = append(q.callers, c), append(q.patients, patientID)
	got, ok := q.versions[id]
	if !ok {
		return questionnaire.Questionnaire{}, questionnaire.ErrNotFound
	}
	q.audit.events = append(q.audit.events, auditevent.AuditEvent{EntityReference: "Questionnaire/" + id, Outcome: auditevent.OutcomeSuccess, AgentName: "questionnaire read"})
	return got, nil
}

func (q *fakeQuestionnaires) Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	if err := inTx(ctx); err != nil {
		return questionnaire.Questionnaire{}, err
	}
	q.callers = append(q.callers, c)
	got, ok := q.versions[id]
	if !ok || got.OrganizationID != c.OrganizationID {
		q.audit.events = append(q.audit.events, auditevent.AuditEvent{EntityReference: "", Outcome: auditevent.OutcomeMinorFailure, AgentName: "questionnaire read"})
		return questionnaire.Questionnaire{}, questionnaire.ErrNotFound
	}
	q.audit.events = append(q.audit.events, auditevent.AuditEvent{EntityReference: "Questionnaire/" + id, Outcome: auditevent.OutcomeSuccess, AgentName: "questionnaire read"})
	return got, nil
}

// fixture is a Service on fakes: appointment A (patient A with practitioner A1, a day ahead) and B (patient
// B with A1), patient A's consent with clinic A active and patient B's too, and clinic A's versions.
type fixture struct {
	svc            *questionnaireresponse.Service
	repo           *fakeRepository
	audit          *fakeAudit
	appts          *fakeAppointments
	questionnaires *fakeQuestionnaires
}

func newFixture(at time.Time) *fixture {
	f := &fixture{repo: &fakeRepository{}, audit: &fakeAudit{}}
	f.appts = &fakeAppointments{
		appts: map[string]appointment.Appointment{
			apptA: {ID: apptA, PatientID: patientA, PractitionerRoleID: roleA1, OrganizationID: clinicA, Status: appointment.StatusBooked,
				Start: start, End: start.Add(30 * time.Minute)},
			apptB: {ID: apptB, PatientID: patientB, PractitionerRoleID: roleA1, OrganizationID: clinicA, Status: appointment.StatusBooked,
				Start: start.Add(time.Hour), End: start.Add(90 * time.Minute)},
		},
		consents: map[string]access.Consent{patientA + clinicA: {Active: true}, patientB + clinicA: {Active: true}},
		roles:    map[string]string{practA1: roleA1, practA2: roleA2},
	}
	f.questionnaires = &fakeQuestionnaires{audit: f.audit, versions: map[string]questionnaire.Questionnaire{
		qActive: {ID: qActive, SeriesID: seriesA, OrganizationID: clinicA, Status: questionnaire.StatusActive, Version: 1, Items: []questionnaire.Item{
			{LinkID: "q1", Text: "Placeholder question", Type: questionnaire.ItemBoolean, Required: true},
			{LinkID: "q2", Text: "Placeholder question", Type: questionnaire.ItemString},
		}},
		qDraft:   {ID: qDraft, SeriesID: seriesA, OrganizationID: clinicA, Status: questionnaire.StatusDraft, Version: 2},
		qRetired: {ID: qRetired, SeriesID: seriesB, OrganizationID: clinicA, Status: questionnaire.StatusRetired, Version: 1},
		qOther:   {ID: qOther, SeriesID: seriesB, OrganizationID: clinicB, Status: questionnaire.StatusActive, Version: 1},
	}}
	f.svc = questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: at},
		Appointments: f.appts, Questionnaires: f.questionnaires,
	})
	return f
}

// seed stores a response of appointment appt in the status, with answers unless it is stopped, and returns
// its id.
func (f *fixture) seed(appt string, status questionnaireresponse.Status) string {
	a := f.appts.appts[appt]
	x := questionnaireresponse.Response{
		ID: "d0000000-0000-4000-8000-00000000000" + string(rune('1'+len(f.repo.responses))), OrganizationID: a.OrganizationID,
		AppointmentID: appt, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID, QuestionnaireID: qActive, SeriesID: seriesA,
		Source: questionnaireresponse.SourceAdded, Status: status, DueAt: a.Start, VersionID: 3, LastUpdated: now.Add(-time.Hour),
	}
	if status != questionnaireresponse.StatusStopped {
		x.Answers = answers
	}
	if status == questionnaireresponse.StatusCompleted {
		x.AuthoredAt = now.Add(-time.Hour)
	}
	f.repo.responses = append(f.repo.responses, x)
	return x.ID
}

// revoke revokes patient A's consent with clinic A at revokedAt, and moves appointment A's start to start
// (before or after the revocation).
func (f *fixture) revoke(apptStart time.Time) {
	f.appts.consents[patientA+clinicA] = access.Consent{RevokedAt: revokedAt}
	a := f.appts.appts[apptA]
	a.Start = apptStart
	f.appts.appts[apptA] = a
}

// event is the audit event the service writes: interaction by c on QuestionnaireResponse/id ("" for none).
func event(c appointment.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeUpdate: auditevent.ActionUpdate,
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeSearchType: auditevent.ActionExecute,
	}[interaction]
	purpose := auditevent.CodeTREAT
	if c.OrganizationID == "" {
		purpose = auditevent.CodePATRQT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "QuestionnaireResponse/" + id
	}
	return e
}

// own is the audit events the service itself wrote (not the questionnaire's read).
func own(events []auditevent.AuditEvent) []auditevent.AuditEvent {
	var out []auditevent.AuditEvent
	for _, e := range events {
		if e.AgentName != "questionnaire read" {
			out = append(out, e)
		}
	}
	return out
}
