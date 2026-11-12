package encounter_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/keyset"
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
	apptA     = "e1000000-0000-4000-8000-0000000000a1" // patient A with role A1
	apptA2    = "e2000000-0000-4000-8000-0000000000a2" // patient A with role A2
	apptB     = "e3000000-0000-4000-8000-0000000000b1" // patient B with role A1
	unknownID = "00000000-0000-4000-8000-000000000000"
	agent     = "consultation_window_close"
)

var (
	start      = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC) // appointment A, 09:00 to 09:30
	joined     = start.Add(2 * time.Minute)                  // the practitioner joined
	now        = start.Add(40 * time.Minute)
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	other      = appointment.Caller{AccountID: accountB, PatientID: patientB}
	noProfile  = appointment.Caller{AccountID: "acc00000-0000-4000-8000-0000000000cc"}
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: practA2}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	adminB     = appointment.Caller{AccountID: "ad000000-0000-4000-8000-0000000000bd", OrganizationID: clinicB, Admin: true}
	errBackend = errors.New("the database is down")
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction"; when fn fails it drops what was recorded in it, as a rollback
// does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, encounters, outcomes := len(t.f.audit.events), slices.Clone(t.f.repo.encounters), len(t.f.appts.outcomes)
	t.f.repo.patient, t.f.repo.clinic = "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.repo.encounters, t.f.appts.outcomes = t.f.audit.events[:events], encounters, t.f.appts.outcomes[:outcomes]
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeRepository holds the encounters and shows them as row-level security does: the patient's own, or the
// clinic's.
type fakeRepository struct {
	err        error
	history    historyAsk
	patient    string
	clinic     string
	encounters []encounter.Encounter
	searched   []encounter.Filter
}

// historyAsk is what PatientHistory was last asked.
type historyAsk struct {
	patient string
	org     string
	page    keyset.Page
}

func (r *fakeRepository) visible(e encounter.Encounter) bool {
	if r.patient != "" {
		return e.PatientID == r.patient
	}
	return r.clinic != "" && e.OrganizationID == r.clinic
}

func (r *fakeRepository) SetPatient(_ context.Context, id string) error {
	r.patient = id
	return nil
}

func (r *fakeRepository) SetClinic(_ context.Context, id string) error {
	r.clinic = id
	return nil
}

func (r *fakeRepository) Start(ctx context.Context, e encounter.Encounter) (encounter.Encounter, bool, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, false, err
	}
	if r.err != nil {
		return encounter.Encounter{}, false, r.err
	}
	for _, got := range r.encounters {
		if got.AppointmentID == e.AppointmentID {
			return got, false, nil
		}
	}
	e.ID, e.VersionID, e.LastUpdated = "c0000000-0000-4000-8000-00000000000"+string(rune('1'+len(r.encounters))), 1, e.PeriodStart
	r.encounters = append(r.encounters, e)
	return e, true, nil
}

func (r *fakeRepository) Finish(ctx context.Context, appointmentID string, at time.Time) (encounter.Encounter, bool, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, false, err
	}
	if r.err != nil {
		return encounter.Encounter{}, false, r.err
	}
	for i, e := range r.encounters {
		if e.AppointmentID == appointmentID && r.visible(e) && e.Status == encounter.StatusInProgress {
			e.Status, e.PeriodEnd, e.LastUpdated = encounter.StatusFinished, at, at
			e.VersionID++
			r.encounters[i] = e
			return e, true, nil
		}
	}
	return encounter.Encounter{}, false, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return encounter.Encounter{}, err
	}
	if r.err != nil {
		return encounter.Encounter{}, r.err
	}
	for _, e := range r.encounters {
		if e.ID == id && r.visible(e) {
			return e, nil
		}
	}
	return encounter.Encounter{}, encounter.ErrNotFound
}

// Organization gives the clinic of any encounter, whatever the scope, as encounter_organization does.
func (r *fakeRepository) Organization(_ context.Context, id string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	for _, e := range r.encounters {
		if e.ID == id {
			return e.OrganizationID, nil
		}
	}
	return "", encounter.ErrNotFound
}

func (r *fakeRepository) Search(ctx context.Context, f encounter.Filter) ([]encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	r.searched = append(r.searched, f)
	if r.err != nil {
		return nil, r.err
	}
	var out []encounter.Encounter
	for _, e := range r.encounters {
		if r.visible(e) && (f.AppointmentID == "" || e.AppointmentID == f.AppointmentID) && (f.PatientID == "" || e.PatientID == f.PatientID) {
			out = append(out, e)
		}
	}
	return out, nil
}

// OfPatient gives the patient's encounters the scope shows.
func (r *fakeRepository) OfPatient(ctx context.Context, patientID string) ([]encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	var out []encounter.Encounter
	for _, e := range r.encounters {
		if r.visible(e) && e.PatientID == patientID {
			out = append(out, e)
		}
	}
	return out, nil
}

// PatientHistory gives the patient's encounters the scope shows after the page's position, newest first by start.
func (r *fakeRepository) PatientHistory(ctx context.Context, patientID, org string, p keyset.Page) ([]encounter.Encounter, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	r.history = historyAsk{page: p, patient: patientID, org: org}
	if r.err != nil {
		return nil, r.err
	}
	var out []encounter.Encounter
	for _, e := range r.encounters {
		ref := "Encounter/" + e.ID
		if r.visible(e) && e.PatientID == patientID && (org == "" || e.OrganizationID == org) && inPage(p, e.PeriodStart, ref) {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b encounter.Encounter) int {
		if c := b.PeriodStart.Compare(a.PeriodStart); c != 0 {
			return c
		}
		return strings.Compare(b.ID, a.ID)
	})
	return out[:min(len(out), p.Limit)], nil
}

// inPage reports whether an item at at with the reference ref is in the page: after its position, in its range.
func inPage(p keyset.Page, at time.Time, ref string) bool {
	after := p.At.IsZero() || at.Before(p.At) || (at.Equal(p.At) && ref < p.Ref)
	return after && (p.From.IsZero() || !at.Before(p.From)) && (p.To.IsZero() || at.Before(p.To))
}

// fakeAppointments holds the appointments (for Lookup, in clinic A's scope), each caller's attendance of
// them (by account id + appointment id) and the outcomes recorded.
type fakeAppointments struct {
	err        error
	repo       *fakeRepository
	appts      map[string]appointment.Appointment
	attendance map[string]appointment.Attendance
	outcomes   []string
}

func (a *fakeAppointments) Lookup(ctx context.Context, id string) (appointment.Appointment, error) {
	if err := inTx(ctx); err != nil {
		return appointment.Appointment{}, err
	}
	if a.err != nil {
		return appointment.Appointment{}, a.err
	}
	got, ok := a.appts[id]
	if !ok || got.OrganizationID != a.repo.clinic {
		return appointment.Appointment{}, appointment.ErrNotFound
	}
	return got, nil
}

func (a *fakeAppointments) Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if err := inTx(ctx); err != nil {
		return appointment.Attendance{}, err
	}
	if a.err != nil {
		return appointment.Attendance{}, a.err
	}
	att, ok := a.attendance[c.AccountID+id]
	if !ok || att.Level == access.None {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return att, nil
}

func (a *fakeAppointments) RecordConsultationOutcome(ctx context.Context, agent, id string, patientJoined bool) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if a.err != nil {
		return a.err
	}
	outcome := "noshow"
	if patientJoined {
		outcome = "fulfilled"
	}
	a.outcomes = append(a.outcomes, agent+" "+id+" "+outcome)
	return nil
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
	repo  *fakeRepository
	appts *fakeAppointments
	audit *fakeAuditor
	svc   *encounter.Service
}

func bookedAt(id, patientID, role string, at time.Time) appointment.Appointment {
	return appointment.Appointment{
		ID: id, PatientID: patientID, PractitionerRoleID: role, OrganizationID: clinicA,
		Start: at, End: at.Add(30 * time.Minute), Status: appointment.StatusBooked, VersionID: 1,
	}
}

// newFixture is clinic A's appointments A (patient A, role A1), A2 (patient A, role A2) and B (patient B,
// role A1), with patient A's active consent and none of patient B: each patient sees their own, the
// practitioner of a role the appointments of that role with a consent, the admin those with a consent.
func newFixture() *fixture {
	a, a2, b := bookedAt(apptA, patientA, roleA1, start), bookedAt(apptA2, patientA, roleA2, start.Add(time.Hour)), bookedAt(apptB, patientB, roleA1, start.Add(2*time.Hour))
	f := &fixture{repo: &fakeRepository{}, audit: &fakeAuditor{}}
	f.appts = &fakeAppointments{
		repo:  f.repo,
		appts: map[string]appointment.Appointment{apptA: a, apptA2: a2, apptB: b},
		attendance: map[string]appointment.Attendance{
			accountA + apptA:   {Appointment: a, PractitionerID: practA1, Level: access.Full, Patient: true},
			accountA + apptA2:  {Appointment: a2, PractitionerID: practA2, Level: access.Full, Patient: true},
			accountB + apptB:   {Appointment: b, PractitionerID: practA1, Level: access.Full, Patient: true},
			practAcc + apptA:   {Appointment: a, PractitionerID: practA1, Level: access.Full, Practitioner: true},
			practAcc + apptA2:  {Appointment: a2, PractitionerID: practA2, Level: access.Full},
			pract2Acc + apptA:  {Appointment: a, PractitionerID: practA1, Level: access.Full},
			pract2Acc + apptA2: {Appointment: a2, PractitionerID: practA2, Level: access.Full, Practitioner: true},
			adminAcc + apptA:   {Appointment: a, PractitionerID: practA1, Level: access.Full},
			adminAcc + apptA2:  {Appointment: a2, PractitionerID: practA2, Level: access.Full},
		},
	}
	f.svc = encounter.NewService(encounter.Ports{Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now}, Appointments: f.appts})
	return f
}

// started is the fixture with the encounters of appointments A, A2 and B in progress.
func started() *fixture {
	f := newFixture()
	for _, id := range []string{apptA, apptA2, apptB} {
		a := f.appts.appts[id]
		f.repo.encounters = append(f.repo.encounters, encounter.Encounter{
			ID: "c" + id[1:], AppointmentID: id, OrganizationID: clinicA, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID,
			Status: encounter.StatusInProgress, PeriodStart: a.Start, LastUpdated: a.Start, VersionID: 1,
		})
	}
	return f
}

// encounterOf is the id of the encounter started() gives the appointment.
func encounterOf(appt string) string { return "c" + appt[1:] }

// inClinic runs fn in a transaction scoped to clinic A, as the lifecycle's callers do.
func (f *fixture) inClinic(ctx context.Context, fn func(ctx context.Context) error) error {
	return fakeTransactor{f: f}.WithTx(ctx, func(ctx context.Context) error {
		if err := f.repo.SetClinic(ctx, clinicA); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// event is the audit event of interaction by c on Encounter/id ("" for none).
func event(c appointment.Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeCreate: auditevent.ActionCreate,
		auditevent.CodeUpdate: auditevent.ActionUpdate, auditevent.CodeSearchType: auditevent.ActionExecute,
	}[interaction]
	purpose := auditevent.CodePATRQT
	if c.OrganizationID != "" {
		purpose = auditevent.CodeTREAT
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
		e.EntityReference = "Encounter/" + id
	}
	return e
}

// systemEvent is the audit event of the agent's interaction on Encounter/id in clinic A, for operations.
func systemEvent(interaction, id string) auditevent.AuditEvent {
	e := event(appointment.Caller{}, interaction, id, auditevent.OutcomeSuccess)
	e.Purpose.Code, e.AgentName, e.OrganizationID = auditevent.CodeHOPERAT, agent, clinicA
	return e
}
