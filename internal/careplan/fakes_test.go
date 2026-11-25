package careplan_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/careplan"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	clinicA   = "0a000000-0000-4000-8000-0000000000a0"
	patientA  = "0b000000-0000-4000-8000-0000000000b0"
	serviceIn = "0c000000-0000-4000-8000-0000000000c1" // in person only
	serviceVd = "0c000000-0000-4000-8000-0000000000c2" // video only
	roleDoc   = "0a000000-0000-4000-8000-0000000000a1" // the practitioner role of doc, who has a visit with patient A
	roleOther = "0a000000-0000-4000-8000-0000000000a2" // the role of other, who has none
)

var (
	now   = time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC)
	admin = careplan.Staff{AccountID: "0d000000-0000-4000-8000-0000000000d1", OrganizationID: clinicA, Admin: true}
	desk  = careplan.Staff{AccountID: "0d000000-0000-4000-8000-0000000000d2", OrganizationID: clinicA, Reception: true}
	me    = careplan.Patient{AccountID: "0d000000-0000-4000-8000-0000000000d3", PatientID: patientA}
	doc   = careplan.Staff{AccountID: "0d000000-0000-4000-8000-0000000000d4", OrganizationID: clinicA, PractitionerID: "p-doc", Practitioner: true}
	other = careplan.Staff{AccountID: "0d000000-0000-4000-8000-0000000000d5", OrganizationID: clinicA, PractitionerID: "p-other", Practitioner: true}
)

type txKey struct{}

// fakeTx runs fn; on an error it restores the plans and drops the audit events written in it, as a rollback does.
type fakeTx struct{ f *fixture }

func (t fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	plans, events, visits := clonePlans(t.f.repo.plans), len(t.f.audit.events), slices.Clone(t.f.visits.visits)
	t.f.repo.org, t.f.repo.patient = "", ""
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.repo.plans, t.f.audit.events, t.f.visits.visits = plans, t.f.audit.events[:events], visits
	}
	return err
}

func clonePlans(ps []careplan.Plan) []careplan.Plan {
	out := slices.Clone(ps)
	for i := range out {
		out[i].Steps = slices.Clone(out[i].Steps)
	}
	return out
}

type fakeRepo struct {
	org, patient string
	plans        []careplan.Plan
	n            int
}

func (r *fakeRepo) SetOrganization(_ context.Context, id string) error { r.org = id; return nil }
func (r *fakeRepo) SetPatient(_ context.Context, id string) error      { r.patient = id; return nil }

func (r *fakeRepo) visible(p careplan.Plan) bool {
	if r.patient != "" {
		return p.PatientID == r.patient && p.BookedAt != nil
	}
	return r.org == p.OrganizationID
}

func (r *fakeRepo) Insert(_ context.Context, p careplan.Plan, _ string) (careplan.Plan, error) {
	r.n++
	p.ID, p.VersionID, p.CreatedAt = "0e000000-0000-4000-8000-00000000000"+strconv.Itoa(r.n), 1, now.Add(time.Duration(r.n)*time.Minute)
	for i := range p.Steps {
		p.Steps[i].ID = p.ID + "-" + strconv.Itoa(i)
	}
	r.plans = append(r.plans, p)
	return p, nil
}

func (r *fakeRepo) Update(_ context.Context, p careplan.Plan, _ string) (bool, error) {
	for i, q := range r.plans {
		if q.ID == p.ID && r.visible(q) {
			if q.VersionID != p.VersionID-1 {
				return false, nil
			}
			r.plans[i] = p
			return true, nil
		}
	}
	return false, careplan.ErrNotFound
}

func (r *fakeRepo) ByID(_ context.Context, id string) (careplan.Plan, error) {
	for _, p := range r.plans {
		if p.ID == id && r.visible(p) {
			p.Steps = slices.Clone(p.Steps)
			return p, nil
		}
	}
	return careplan.Plan{}, careplan.ErrNotFound
}

func (r *fakeRepo) ForPatient(_ context.Context, patientID string) ([]careplan.Plan, error) {
	var out []careplan.Plan
	for _, p := range slices.Backward(r.plans) {
		if p.PatientID == patientID && r.visible(p) {
			p.Steps = slices.Clone(p.Steps)
			out = append(out, p)
		}
	}
	return out, nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("audited outside a transaction")
	}
	a.events = append(a.events, e)
	return nil
}

type fakeConsents struct{ state map[string]access.Consent }

func (c fakeConsents) State(_ context.Context, patientID, org string) (access.Consent, error) {
	return c.state[patientID+org], nil
}

type fakePrices struct{}

func (fakePrices) OfClinic(_ context.Context, org string) ([]chargeitemdefinition.Definition, error) {
	if org != clinicA {
		return nil, nil
	}
	return []chargeitemdefinition.Definition{
		{ID: serviceIn, Name: "Synthetic exam", Active: true, Modes: []schedule.VisitMode{schedule.ModeInPerson}},
		{ID: serviceVd, Name: "Synthetic video check", Active: true, Modes: []schedule.VisitMode{schedule.ModeVideo}},
	}, nil
}

// fakeVisits books a slot "<mode>@<hours ahead>" as a visit of that mode at that time; "taken" is refused.
type fakeVisits struct {
	visits    []appointment.Appointment
	cancelled []string
}

func (v *fakeVisits) BookPlan(_ context.Context, _ appointment.Caller, patientID string, steps []appointment.StaffBooking) ([]appointment.Appointment, error) {
	var out []appointment.Appointment
	for _, b := range steps {
		var mode string
		var hours int
		switch b.SlotID {
		case "taken":
			return nil, appointment.ErrSlotUnavailable
		case "busy": // its service needs a device another visit holds then
			return nil, fmt.Errorf("appointment: book plan: %w", appointment.ErrResourceBusy)
		}
		for i := range b.SlotID {
			if b.SlotID[i] == '@' {
				mode = b.SlotID[:i]
				hours, _ = strconv.Atoi(b.SlotID[i+1:])
			}
		}
		at := now.Add(time.Duration(hours) * time.Hour)
		a := appointment.Appointment{
			ID: "0f000000-0000-4000-8000-" + strconv.Itoa(100000000000+len(v.visits)), PatientID: patientID, OrganizationID: clinicA,
			Start: at, End: at.Add(30 * time.Minute), Mode: schedule.VisitMode(mode), Status: appointment.StatusBooked,
		}
		v.visits = append(v.visits, a)
		out = append(out, a)
	}
	return out, nil
}

func (v *fakeVisits) PlanVisits(_ context.Context, _ string, ids []string) ([]appointment.Appointment, error) {
	var out []appointment.Appointment
	for _, a := range v.visits {
		if slices.Contains(ids, a.ID) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (v *fakeVisits) CancelPlan(_ context.Context, _ appointment.Caller, ids []string) error {
	for i, a := range v.visits {
		if slices.Contains(ids, a.ID) && a.Start.After(now) {
			v.visits[i].Status = appointment.StatusCancelled
			v.cancelled = append(v.cancelled, a.ID)
		}
	}
	return nil
}

// fakeCare is the patients' visits by patient and clinic: patient A has one with doc's role.
type fakeCare map[string][]access.Visit

func (c fakeCare) Visits(_ context.Context, patientID, org string) ([]access.Visit, error) {
	return c[patientID+org], nil
}

// fakeRoles gives each practitioner's role in clinic A.
type fakeRoles struct{}

func (fakeRoles) OfPractitioner(_ context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error) {
	role := map[string]string{"p-doc": roleDoc, "p-other": roleOther}[practitionerID]
	if org != clinicA || role == "" {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return practitionerrole.PractitionerRole{ID: role, OrganizationID: org, PractitionerID: practitionerID, Active: true}, nil
}

type fixture struct {
	repo   *fakeRepo
	audit  *fakeAudit
	visits *fakeVisits
	clock  *clock.Fixed
	svc    *careplan.Service
}

func newFixture() *fixture {
	f := &fixture{repo: &fakeRepo{}, audit: &fakeAudit{}, visits: &fakeVisits{}, clock: &clock.Fixed{T: now}}
	f.svc = careplan.NewService(careplan.Ports{
		Repo: f.repo, Tx: fakeTx{f: f}, Audit: f.audit, Prices: fakePrices{}, Visits: f.visits, Clock: f.clock,
		Consents: fakeConsents{state: map[string]access.Consent{patientA + clinicA: {Active: true}}},
		Care:     fakeCare{patientA + clinicA: {{PractitionerRole: roleDoc, Start: now.Add(-24 * time.Hour)}}}, Roles: fakeRoles{},
	})
	return f
}

// draft is a two-step draft for patient A: in person on day 0, by video on day 14.
func draft() careplan.Draft {
	return careplan.Draft{PatientID: patientA, Title: "Synthetic plan", Steps: []careplan.Step{
		{ServiceID: serviceIn, VisitMode: "in-person"}, {ServiceID: serviceVd, VisitMode: "video", DayOffset: 14},
	}}
}

// event is the audit event of interaction by staff c on CarePlan/id ("" for none).
func event(account, org, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeCreate: auditevent.ActionCreate,
		auditevent.CodeUpdate: auditevent.ActionUpdate, auditevent.CodeSearchType: auditevent.ActionExecute,
	}[interaction]
	purpose := auditevent.CodeTREAT
	if org == "" {
		purpose = auditevent.CodePATRQT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: account,
		OrganizationID: org,
	}
	if id != "" {
		e.EntityReference = "CarePlan/" + id
	}
	return e
}
