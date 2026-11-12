package history_test

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// Synthetic ids of the practitioner's view: patient A's visit V1 at clinic A with practitioner A1, and V2 at clinic B,
// which the patient shared with clinic A; a signed document of each, a completed response with a file in V1.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	roleA1    = "a1000000-0000-4000-8000-0000000000a1"
	roleB1    = "b1000000-0000-4000-8000-0000000000b1"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	visit1    = "c1000000-0000-4000-8000-0000000000a1"
	visit2    = "c2000000-0000-4000-8000-0000000000b1"
	appt1     = "e1000000-0000-4000-8000-0000000000a1"
	appt2     = "e2000000-0000-4000-8000-0000000000b1"
	apptNow   = "e3000000-0000-4000-8000-0000000000a3" // the appointment the summary is for
	visitNow  = "c3000000-0000-4000-8000-0000000000a3"
	doc1      = "d1000000-0000-4000-8000-0000000000a1"
	doc2      = "d2000000-0000-4000-8000-0000000000b1"
	draftNow  = "d3000000-0000-4000-8000-0000000000a3"
	resp1     = "f1000000-0000-4000-8000-0000000000a1"
	respNow   = "f3000000-0000-4000-8000-0000000000a3"
	version1  = "91000000-0000-4000-8000-0000000000a1"
	file1     = "b0000000-0000-4000-8000-0000000000f1"
	allergy1  = "a7000000-0000-4000-8000-000000000001"
	allergy2  = "a7000000-0000-4000-8000-000000000002"
	medicine1 = "a8000000-0000-4000-8000-000000000001"
	medicine2 = "a8000000-0000-4000-8000-000000000002"
)

var (
	pract      = appointment.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: "p1000000-0000-4000-8000-0000000000a1"}
	errBackend = errors.New("the database is down")
	day        = func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction"; what was audited in one that fails is dropped, as a rollback does.
type fakeTransactor struct{ f *practFixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	n := len(t.f.audit.events)
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		t.f.audit.events = t.f.audit.events[:n]
	}
	return err
}

func inTx(ctx context.Context) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeAuditor records every event as "<domain> <subtype> <outcome> <entity>", the history's own as "history".
type fakeAuditor struct {
	err    error
	log    *[]string
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
	*a.log = append(*a.log, "history "+e.Subtype.Code+" "+e.EntityReference)
	return nil
}

// disclosed records each domain's Disclosed call in the shared log.
func disclosed(ctx context.Context, log *[]string, domain string, err error, ids []string) error {
	if e := inTx(ctx); e != nil {
		return e
	}
	if err != nil {
		return err
	}
	for _, id := range ids {
		*log = append(*log, domain+" "+id)
	}
	return nil
}

type fakeEncounters struct {
	err       error
	discloseE error
	visits    []encounter.Visit
	byAppt    map[string][]encounter.Encounter
	log       *[]string
	asked     []string
}

func (e *fakeEncounters) History(ctx context.Context, c appointment.Caller, patientID string) ([]encounter.Visit, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	e.asked = append(e.asked, c.AccountID+" "+patientID)
	if e.err != nil {
		return nil, e.err
	}
	return slices.Clone(e.visits), nil
}

func (e *fakeEncounters) Disclosed(ctx context.Context, _ appointment.Caller, ids []string) error {
	return disclosed(ctx, e.log, "encounter", e.discloseE, ids)
}

func (e *fakeEncounters) Search(ctx context.Context, _ appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error) {
	*e.log = append(*e.log, "encounter search "+f.AppointmentID)
	return e.byAppt[f.AppointmentID], inTx(ctx)
}

type fakeCompositions struct {
	err      error
	signed   map[string][]composition.Draft // by encounter
	drafts   map[string][]composition.Draft
	draftErr error
	log      *[]string
}

func (f *fakeCompositions) History(ctx context.Context, _ appointment.Caller, visits []encounter.Visit) ([]composition.Document, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	var out []composition.Document
	for _, v := range visits {
		for _, d := range f.signed[v.ID] {
			out = append(out, composition.Document{Draft: d, Encounter: v.Encounter})
		}
	}
	return out, f.err
}

func (f *fakeCompositions) Disclosed(ctx context.Context, _ appointment.Caller, ids []string) error {
	return disclosed(ctx, f.log, "composition", nil, ids)
}

func (f *fakeCompositions) Drafts(ctx context.Context, _ appointment.Caller, encounterID string) ([]composition.Draft, error) {
	*f.log = append(*f.log, "composition drafts "+encounterID)
	return f.drafts[encounterID], cmpOr(inTx(ctx), f.draftErr)
}

func cmpOr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

type fakeResponses struct {
	err       error
	completed map[string][]questionnaireresponse.Response // by appointment
	all       map[string][]questionnaireresponse.Response
	log       *[]string
}

func (f *fakeResponses) History(ctx context.Context, _ appointment.Caller, visits []encounter.Visit) ([]questionnaireresponse.Response, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	var out []questionnaireresponse.Response
	for _, v := range visits {
		out = append(out, f.completed[v.AppointmentID]...)
	}
	return out, f.err
}

func (f *fakeResponses) Disclosed(ctx context.Context, _ appointment.Caller, ids []string) error {
	return disclosed(ctx, f.log, "questionnaireresponse", nil, ids)
}

func (f *fakeResponses) Search(ctx context.Context, _ appointment.Caller, flt questionnaireresponse.Filter) ([]questionnaireresponse.Seen, error) {
	*f.log = append(*f.log, "questionnaireresponse search "+flt.AppointmentID)
	var out []questionnaireresponse.Seen
	for _, r := range f.all[flt.AppointmentID] {
		out = append(out, questionnaireresponse.Seen{Response: r})
	}
	return out, inTx(ctx)
}

type fakeQuestionnaires struct {
	titles map[string]string
	log    *[]string
}

func (f *fakeQuestionnaires) Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	if err := inTx(ctx); err != nil {
		return questionnaire.Questionnaire{}, err
	}
	*f.log = append(*f.log, "questionnaire "+id)
	title, ok := f.titles[id]
	if !ok || c.OrganizationID != clinicA || c.PractitionerID == "" {
		return questionnaire.Questionnaire{}, questionnaire.ErrNotFound
	}
	return questionnaire.Questionnaire{ID: id, Title: title}, nil
}

type fakeClinics struct{}

func (fakeClinics) Names(ctx context.Context, org string, ids []string) (map[string]string, error) {
	all := map[string]string{clinicA: "Synthetic clinic A", clinicB: "Synthetic clinic B"}
	out := map[string]string{}
	for _, id := range ids {
		out[id] = all[id]
	}
	if org != clinicA {
		return nil, errors.New("not clinic A's scope")
	}
	return out, inTx(ctx)
}

type fakeRoles struct{}

func (fakeRoles) Specialties(ctx context.Context, org string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		out[id] = map[string]string{roleA1: "394579002", roleB1: "419192003"}[id]
	}
	if org != clinicA {
		return nil, errors.New("not clinic A's scope")
	}
	return out, inTx(ctx)
}

type fakeAppointments struct {
	attendance map[string]appointment.Attendance
}

func (f *fakeAppointments) Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error) {
	if err := inTx(ctx); err != nil {
		return appointment.Attendance{}, err
	}
	att, ok := f.attendance[c.AccountID+id]
	if !ok {
		return appointment.Attendance{}, appointment.ErrNotFound
	}
	return att, nil
}

type practFixture struct {
	svc          *history.Service
	audit        *fakeAuditor
	encounters   *fakeEncounters
	compositions *fakeCompositions
	responses    *fakeResponses
	allergies    *fakeAllergies
	medications  *fakeMedications
	log          *[]string
}

// newPractFixture is patient A's history as practitioner A1 of clinic A may see it.
func newPractFixture() *practFixture {
	log := &[]string{}
	v1 := encounter.Visit{Encounter: encounter.Encounter{ID: visit1, AppointmentID: appt1, OrganizationID: clinicA, PatientID: patientA,
		PractitionerRoleID: roleA1, Status: encounter.StatusFinished, PeriodStart: day(1, 9)}}
	v2 := encounter.Visit{Encounter: encounter.Encounter{ID: visit2, AppointmentID: appt2, OrganizationID: clinicB, PatientID: patientA,
		PractitionerRoleID: roleB1, Status: encounter.StatusFinished, PeriodStart: day(1, 8)}, Shared: true}
	f := &practFixture{log: log, audit: &fakeAuditor{log: log}}
	f.encounters = &fakeEncounters{visits: []encounter.Visit{v1, v2}, log: log, byAppt: map[string][]encounter.Encounter{
		apptNow: {{ID: visitNow, AppointmentID: apptNow, OrganizationID: clinicA, PatientID: patientA, Status: encounter.StatusInProgress}},
	}}
	coded := []composition.DraftField{
		{Type: composition.FieldCoded, Field: composition.Field{Key: "diagnosis", Value: "J06.9"}},
		{Type: composition.FieldText, Field: composition.Field{Key: "findings", Value: "Placeholder findings"}},
		{Type: composition.FieldCoded, Field: composition.Field{Key: "secondary", Missing: true}},
	}
	f.compositions = &fakeCompositions{log: log,
		signed: map[string][]composition.Draft{
			visit1: {{ID: doc1, EncounterID: visit1, Status: composition.StatusFinal, SignedAt: day(1, 10), Title: "Placeholder report", Fields: coded}},
			visit2: {{ID: doc2, EncounterID: visit2, Status: composition.StatusFinal, SignedAt: day(1, 8), Title: "Placeholder referral"}},
		},
		drafts: map[string][]composition.Draft{visitNow: {
			{ID: draftNow, EncounterID: visitNow, Status: composition.StatusPreliminary},
			{ID: "d4000000-0000-4000-8000-0000000000a4", EncounterID: visitNow, Status: composition.StatusFinal},
		}},
	}
	f.responses = &fakeResponses{log: log,
		completed: map[string][]questionnaireresponse.Response{appt1: {{
			ID: resp1, AppointmentID: appt1, QuestionnaireID: version1, Status: questionnaireresponse.StatusCompleted, AuthoredAt: day(1, 7),
			Attachments: []binary.Attachment{{BinaryID: file1, Title: "Placeholder photo"}},
		}}},
		all: map[string][]questionnaireresponse.Response{apptNow: {
			{ID: respNow, AppointmentID: apptNow, Status: questionnaireresponse.StatusInProgress},
			{ID: "f4000000-0000-4000-8000-0000000000a4", AppointmentID: apptNow, Status: questionnaireresponse.StatusCompleted},
			{ID: "f5000000-0000-4000-8000-0000000000a5", AppointmentID: apptNow, Status: questionnaireresponse.StatusStopped},
		}},
	}
	f.allergies = &fakeAllergies{seen: map[string][]allergyintolerance.Seen{patientA: {
		{AllergyIntolerance: allergyintolerance.AllergyIntolerance{ID: allergy1, Code: allergyintolerance.Code{Text: "Placeholder substance"},
			Reaction: "Placeholder reaction", Severity: allergyintolerance.SeveritySevere, ClinicalStatus: allergyintolerance.ClinicalActive,
			VerificationStatus: allergyintolerance.VerificationConfirmed}},
		{AllergyIntolerance: allergyintolerance.AllergyIntolerance{ID: allergy2, Code: allergyintolerance.Code{Text: "Placeholder old substance"},
			ClinicalStatus: allergyintolerance.ClinicalResolved, VerificationStatus: allergyintolerance.VerificationUnconfirmed}},
	}}}
	f.medications = &fakeMedications{seen: map[string][]medicationstatement.Seen{patientA: {
		{MedicationStatement: medicationstatement.MedicationStatement{ID: medicine1, Code: medicationstatement.Code{Text: "Placeholder medicine"},
			Status: medicationstatement.StatusActive, Dosage: medicationstatement.Dosage{DoseValue: 1, DoseUnit: "{tablet}"}}},
		{MedicationStatement: medicationstatement.MedicationStatement{ID: medicine2, Code: medicationstatement.Code{Text: "Placeholder stopped"},
			Status: medicationstatement.StatusStopped, Confirmation: medicationstatement.Confirmation{At: day(2, 9)}}},
	}}}
	appts := &fakeAppointments{attendance: map[string]appointment.Attendance{
		practAcc + apptNow: {Practitioner: true, Level: 2, Appointment: appointment.Appointment{ID: apptNow, PatientID: patientA, OrganizationID: clinicA}},
		practAcc + appt2:   {Level: 2, Appointment: appointment.Appointment{ID: appt2, PatientID: patientA, OrganizationID: clinicA}},
	}}
	f.svc = history.NewService(history.Ports{
		Tx: fakeTransactor{f: f}, Audit: f.audit, Encounters: f.encounters, Compositions: f.compositions, Responses: f.responses,
		Questionnaires: &fakeQuestionnaires{titles: map[string]string{version1: "Placeholder intake"}, log: log},
		Clinics:        fakeClinics{}, Roles: fakeRoles{}, Appointments: appts, Allergies: f.allergies, Medications: f.medications,
	})
	return f
}
