package appointment_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a" // listed
	clinicB   = "b0000000-0000-4000-8000-00000000000b" // listed
	clinicU   = "c0000000-0000-4000-8000-00000000000c" // unlisted
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	patientB  = "bb000000-0000-4000-8000-0000000000bb"
	accountA  = "acc00000-0000-4000-8000-0000000000aa"
	accountB  = "acc00000-0000-4000-8000-0000000000bb"
	roleA1    = "r1000000-0000-4000-8000-0000000000a1"
	roleA2    = "r2000000-0000-4000-8000-0000000000a2"
	roleB     = "r3000000-0000-4000-8000-0000000000b1"
	roleU     = "r4000000-0000-4000-8000-0000000000c1"
	practA1   = "p1000000-0000-4000-8000-0000000000a1"
	practA2   = "p2000000-0000-4000-8000-0000000000a2"
	adminAcc  = "ad000000-0000-4000-8000-0000000000ad"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	pract2Acc = "ac200000-0000-4000-8000-0000000000a2"
	slotFree  = "51000000-0000-4000-8000-000000000001"
	slotSoon  = "51000000-0000-4000-8000-000000000002" // 23 hours ahead
	slotPast  = "51000000-0000-4000-8000-000000000003"
	slotU     = "51000000-0000-4000-8000-000000000004" // of the unlisted clinic
	slotB     = "51000000-0000-4000-8000-000000000005" // of clinic B
	newID     = "f9000000-0000-4000-8000-0000000000f9"
)

var (
	now        = time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC)
	start      = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	revokedAt  = now.Add(-48 * time.Hour)
	me         = appointment.Caller{AccountID: accountA, PatientID: patientA}
	other      = appointment.Caller{AccountID: accountB, PatientID: patientB}
	noProfile  = appointment.Caller{AccountID: "acc00000-0000-4000-8000-0000000000cc"}
	admin      = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicA, Admin: true}
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}
	pract2     = appointment.Caller{AccountID: pract2Acc, OrganizationID: clinicA, PractitionerID: practA2}
	adminB     = appointment.Caller{AccountID: adminAcc, OrganizationID: clinicB, Admin: true}
	nameA      = appointment.Name{Given: "Ada", Family: "Example"}
	errBackend = errors.New("the database is down")
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction"; when fn fails it drops what was recorded in it (audit events,
// jobs, slot changes, appointment changes), as a rollback does.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	events, jobs := len(t.f.audit.events), len(t.f.jobs.enqueued)
	appts, held, released := slices.Clone(t.f.repo.appts), slices.Clone(t.f.slots.held), slices.Clone(t.f.slots.released)
	t.f.repo.patient, t.f.repo.clinic, t.f.repo.notice = "", "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events, t.f.jobs.enqueued = t.f.audit.events[:events], t.f.jobs.enqueued[:jobs]
		t.f.repo.appts, t.f.slots.held, t.f.slots.released = appts, held, released
	}
	return err
}

// fakeRepository holds appointments and shows them as row-level security does.
type fakeRepository struct {
	err      error // returned by every read and write when set
	consents *fakeConsents
	appts    []appointment.Appointment
	patient  string // set by SetPatient
	clinic   string // set by SetClinic
	notice   string // set by SetNotice
	calls    []string
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	r.calls = append(r.calls, call)
}

func (r *fakeRepository) visible(a appointment.Appointment) bool {
	if r.patient != "" {
		return a.PatientID == r.patient
	}
	if r.clinic == "" || a.OrganizationID != r.clinic {
		return false
	}
	st := r.consents.state[a.PatientID+a.OrganizationID]
	return st.Active || a.Start.Before(st.RevokedAt) || a.ID == r.notice
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	r.seen(ctx, "SetPatient")
	r.patient = id
	return nil
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	r.seen(ctx, "SetClinic")
	r.clinic = id
	return nil
}

func (r *fakeRepository) SetNotice(ctx context.Context, id string) error {
	r.seen(ctx, "SetNotice")
	r.notice = id
	return nil
}

func (r *fakeRepository) Create(ctx context.Context, a appointment.Appointment) (appointment.Appointment, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return appointment.Appointment{}, r.err
	}
	for _, b := range r.appts {
		if b.PractitionerRoleID == a.PractitionerRoleID && b.Status != appointment.StatusCancelled && b.Start.Before(a.End) && a.Start.Before(b.End) {
			return appointment.Appointment{}, appointment.ErrSlotUnavailable
		}
	}
	a.ID, a.Status, a.VersionID, a.LastUpdated = newID, appointment.StatusBooked, 1, a.Created
	r.appts = append(r.appts, a)
	return a, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (appointment.Appointment, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return appointment.Appointment{}, r.err
	}
	for _, a := range r.appts {
		if a.ID == id && r.visible(a) {
			return a, nil
		}
	}
	return appointment.Appointment{}, appointment.ErrNotFound
}

func (r *fakeRepository) Search(ctx context.Context, f appointment.Filter) ([]appointment.Appointment, error) {
	r.seen(ctx, "Search")
	if r.err != nil {
		return nil, r.err
	}
	var out []appointment.Appointment
	for _, a := range r.appts {
		if r.visible(a) && (f.PatientID == "" || a.PatientID == f.PatientID) && (f.OrganizationID == "" || a.OrganizationID == f.OrganizationID) &&
			(f.PractitionerRoleID == "" || a.PractitionerRoleID == f.PractitionerRoleID) && (f.Status == "" || a.Status == f.Status) &&
			(f.From.IsZero() || !a.Start.Before(f.From)) && (f.To.IsZero() || a.Start.Before(f.To)) {
			out = append(out, a)
		}
	}
	slices.SortFunc(out, func(x, y appointment.Appointment) int { return x.Start.Compare(y.Start) })
	return out, nil
}

func (r *fakeRepository) change(ctx context.Context, call, id string, fn func(*appointment.Appointment)) (appointment.Appointment, error) {
	r.seen(ctx, call)
	if r.err != nil {
		return appointment.Appointment{}, r.err
	}
	for i, a := range r.appts {
		if a.ID == id && r.visible(a) && a.Status == appointment.StatusBooked {
			fn(&a)
			a.VersionID++
			r.appts[i] = a
			return a, nil
		}
	}
	return appointment.Appointment{}, appointment.ErrClosed
}

func (r *fakeRepository) Cancel(ctx context.Context, id string, reason appointment.Reason, by string, at time.Time) (appointment.Appointment, error) {
	return r.change(ctx, "Cancel", id, func(a *appointment.Appointment) {
		a.Status, a.Reason, a.CancelledBy, a.CancelledAt, a.LastUpdated = appointment.StatusCancelled, reason, by, at, at
	})
}

func (r *fakeRepository) SetStatus(ctx context.Context, id string, status appointment.Status, at time.Time) (appointment.Appointment, error) {
	return r.change(ctx, "SetStatus:"+string(status), id, func(a *appointment.Appointment) { a.Status, a.LastUpdated = status, at })
}

// fakeConsents gives the consent state per patient and clinic.
type fakeConsents struct {
	err   error
	state map[string]access.Consent // by patient id + clinic id
	asked int
}

func (c *fakeConsents) State(_ context.Context, patientID, org string) (access.Consent, error) {
	c.asked++
	return c.state[patientID+org], c.err
}

// fakeSlots holds the bookable slots and records holds and releases.
type fakeSlots struct {
	err      error
	bookable map[string]slot.Bookable
	held     []string
	released []string
}

func (s *fakeSlots) Bookable(ctx context.Context, id string) (slot.Bookable, error) {
	if ctx.Value(txKey{}) != nil {
		return slot.Bookable{}, errors.New("Bookable inside the booking's transaction")
	}
	b, ok := s.bookable[id] // the directory may still offer a slot another booking holds
	if !ok {
		return slot.Bookable{}, slot.ErrUnavailable
	}
	return b, s.err
}

func (s *fakeSlots) Hold(ctx context.Context, org, id string) (slot.Slot, error) {
	if ctx.Value(txKey{}) == nil {
		return slot.Slot{}, errors.New("Hold outside a transaction")
	}
	b, ok := s.bookable[id]
	if !ok || b.OrganizationID != org || slices.Contains(s.held, id) {
		return slot.Slot{}, slot.ErrUnavailable
	}
	s.held = append(s.held, id)
	b.Status = slot.StatusBusy
	return b.Slot, nil
}

func (s *fakeSlots) Release(ctx context.Context, org, id string) (bool, error) {
	if ctx.Value(txKey{}) == nil {
		return false, errors.New("Release outside a transaction")
	}
	s.released = append(s.released, org+"/"+id)
	return true, s.err
}

// fakePatients gives a profile when the scope of the fake repository shows the patient.
type fakePatients struct {
	err    error
	repo   *fakeRepository
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
	switch id {
	case patientA:
		return patient.Patient{ID: patientA, AccountID: accountA, GivenName: "Ada", FamilyName: "Example"}, nil
	case patientB:
		return patient.Patient{ID: patientB, AccountID: accountB, GivenName: "Bo", FamilyName: "Example"}, nil
	}
	return patient.Patient{}, patient.ErrNotFound
}

type fakeClinics struct{ err error }

func (c *fakeClinics) Read(_ context.Context, cl organization.Caller, id string) (organization.Organization, error) {
	if c.err != nil {
		return organization.Organization{}, c.err
	}
	if cl.OrganizationID != id {
		return organization.Organization{}, organization.ErrNotFound
	}
	return organization.Organization{ID: id, Name: "Synthetic clinic", Listed: id != clinicU, TimeZone: "Europe/Sarajevo"}, nil
}

type fakeRoles struct{ err error }

func (r *fakeRoles) Read(_ context.Context, c practitionerrole.Caller, id string) (practitionerrole.PractitionerRole, error) {
	if r.err != nil {
		return practitionerrole.PractitionerRole{}, r.err
	}
	roles := map[string]practitionerrole.PractitionerRole{
		roleA1: {ID: roleA1, OrganizationID: clinicA, PractitionerID: practA1, Active: true},
		roleA2: {ID: roleA2, OrganizationID: clinicA, PractitionerID: practA2, Active: true},
		roleB:  {ID: roleB, OrganizationID: clinicB, PractitionerID: "p3000000-0000-4000-8000-0000000000b1", Active: true},
	}
	ro, ok := roles[id]
	if !ok || ro.OrganizationID != c.OrganizationID {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return ro, nil
}

// fakeJobs records what was enqueued: "<kind> <appointment id> <clinic id>[ at <time>]".
type fakeJobs struct {
	err      error
	enqueued []string
}

func (j *fakeJobs) add(ctx context.Context, s string) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("enqueued outside a transaction")
	}
	if j.err != nil {
		return j.err
	}
	j.enqueued = append(j.enqueued, s)
	return nil
}

func (j *fakeJobs) EnqueueBooked(ctx context.Context, id, org string) error {
	return j.add(ctx, "booked "+id+" "+org)
}

func (j *fakeJobs) EnqueueCancelled(ctx context.Context, id, org string) error {
	return j.add(ctx, "cancelled "+id+" "+org)
}

func (j *fakeJobs) EnqueueReminder(ctx context.Context, id, org string, at time.Time) error {
	return j.add(ctx, fmt.Sprintf("reminder %s %s at %s", id, org, at.Format(time.RFC3339)))
}

type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("audited outside a transaction")
	}
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

type fixture struct {
	repo     *fakeRepository
	consents *fakeConsents
	slots    *fakeSlots
	patients *fakePatients
	clinics  *fakeClinics
	roles    *fakeRoles
	jobs     *fakeJobs
	audit    *fakeAuditor
	svc      *appointment.Service
}

// slotAt is a bookable half-hour slot of the role and clinic at t.
func slotAt(id, role, org string, t time.Time) slot.Bookable {
	return slot.Bookable{Slot: slot.Slot{ID: id, OrganizationID: org, Status: slot.StatusFree, Period: slot.Period{Start: t, End: t.Add(30 * time.Minute)}}, PractitionerRoleID: role}
}

// newFixture is clinic A and B with patient A's active consent with A, and the slots.
func newFixture(appts ...appointment.Appointment) *fixture {
	f := &fixture{
		consents: &fakeConsents{state: map[string]access.Consent{patientA + clinicA: {Active: true}, patientA + clinicU: {Active: true}}},
		slots: &fakeSlots{bookable: map[string]slot.Bookable{
			slotFree: slotAt(slotFree, roleA1, clinicA, start),
			slotSoon: slotAt(slotSoon, roleA1, clinicA, now.Add(23*time.Hour)),
			slotPast: slotAt(slotPast, roleA1, clinicA, now.Add(-time.Minute)),
			slotU:    slotAt(slotU, roleU, clinicU, start),
			slotB:    slotAt(slotB, roleB, clinicB, start),
		}},
		clinics: &fakeClinics{}, roles: &fakeRoles{}, jobs: &fakeJobs{}, audit: &fakeAuditor{},
	}
	f.repo = &fakeRepository{consents: f.consents, appts: appts}
	f.patients = &fakePatients{repo: f.repo}
	f.svc = appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now},
		Consents: f.consents, Slots: f.slots, Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs,
	})
	return f
}

// booked is a booked appointment of the patient with the role of the clinic from at for half an hour.
func booked(id, patientID, role, org string, at time.Time) appointment.Appointment {
	return appointment.Appointment{
		ID: id, PatientID: patientID, PractitionerRoleID: role, OrganizationID: org, SlotID: "51" + id[2:],
		Start: at, End: at.Add(30 * time.Minute), Created: now.Add(-72 * time.Hour), Status: appointment.StatusBooked, VersionID: 1,
	}
}

// event is the audit event of interaction by c on resource/id ("" for none).
func event(c appointment.Caller, interaction, resource, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
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
		e.EntityReference = resource + "/" + id
	}
	return e
}

// jobEvent is the audit event of a job's interaction on resource/id in the clinic org.
func jobEvent(agent, org, interaction, resource, id string) auditevent.AuditEvent {
	e := event(appointment.Caller{}, interaction, resource, id, auditevent.OutcomeSuccess)
	e.Purpose.Code, e.AgentName, e.OrganizationID = auditevent.CodeHOPERAT, agent, org
	return e
}
