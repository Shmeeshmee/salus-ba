package allergyintolerance_test

import (
	"context"
	"maps"
	"slices"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// fakeRepo holds the allergies by id and records the scope; it does not apply row-level security (the service must
// decide alone) but keeps the versions.
type fakeRepo struct {
	byID    map[string]allergyintolerance.AllergyIntolerance
	clinic  string
	patient string
	deleted []string
}

func newRepo(as ...allergyintolerance.AllergyIntolerance) *fakeRepo {
	r := &fakeRepo{byID: map[string]allergyintolerance.AllergyIntolerance{}}
	for _, a := range as {
		r.byID[a.ID] = a
	}
	return r
}

func (r *fakeRepo) SetClinic(_ context.Context, id string) error  { r.clinic = id; return nil }
func (r *fakeRepo) SetPatient(_ context.Context, id string) error { r.patient = id; return nil }

func (r *fakeRepo) ByPatient(_ context.Context, patientID string) ([]allergyintolerance.AllergyIntolerance, error) {
	var out []allergyintolerance.AllergyIntolerance
	for _, id := range slices.Sorted(maps.Keys(r.byID)) {
		if a := r.byID[id]; a.PatientID == patientID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeRepo) ByID(_ context.Context, id string) (allergyintolerance.AllergyIntolerance, error) {
	a, ok := r.byID[id]
	if !ok {
		return allergyintolerance.AllergyIntolerance{}, allergyintolerance.ErrNotFound
	}
	return a, nil
}

func (r *fakeRepo) Insert(_ context.Context, a allergyintolerance.AllergyIntolerance) error {
	a.VersionID, a.LastUpdated = 1, a.RecordedDate
	r.byID[a.ID] = a
	return nil
}

func (r *fakeRepo) Update(_ context.Context, a allergyintolerance.AllergyIntolerance) (int32, error) {
	old, ok := r.byID[a.ID]
	if !ok || old.VersionID != a.VersionID {
		return 0, allergyintolerance.ErrVersionConflict
	}
	a.VersionID++
	r.byID[a.ID] = a
	return a.VersionID, nil
}

func (r *fakeRepo) Delete(_ context.Context, id string, version int32) error {
	if old, ok := r.byID[id]; !ok || old.VersionID != version {
		return allergyintolerance.ErrVersionConflict
	}
	delete(r.byID, id)
	r.deleted = append(r.deleted, id)
	return nil
}

type fakeConsents map[string]access.Consent // by clinic

func (f fakeConsents) State(_ context.Context, _, org string) (access.Consent, error) {
	return f[org], nil
}

type fakeVisits map[string][]access.Visit // by clinic

func (f fakeVisits) Visits(_ context.Context, _, org string) ([]access.Visit, error) {
	return f[org], nil
}

type fakeRoles map[string]string // role id by practitioner id

func (f fakeRoles) OfPractitioner(_ context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error) {
	id, ok := f[practitionerID]
	if !ok {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return practitionerrole.PractitionerRole{ID: id, OrganizationID: org, PractitionerID: practitionerID}, nil
}

// fakeEncounters gives each encounter and whether the caller attends it as its practitioner, with the consent.
type fakeEncounters struct {
	byID    map[string]encounter.Encounter
	consent bool
}

func (f fakeEncounters) Access(_ context.Context, c appointment.Caller, id string) (encounter.Encounter, appointment.Attendance, error) {
	e, ok := f.byID[id]
	if !ok || e.OrganizationID != c.OrganizationID {
		return encounter.Encounter{}, appointment.Attendance{}, encounter.ErrNotFound
	}
	practitioner := practitionerOfRole[e.PractitionerRoleID] == c.PractitionerID
	return e, appointment.Attendance{
		Appointment:   appointment.Appointment{PatientID: e.PatientID, OrganizationID: e.OrganizationID, PractitionerRoleID: e.PractitionerRoleID},
		ConsentActive: f.consent, Practitioner: practitioner,
	}, nil
}

type fakeTerms map[string]string // ATC by drug term id

// drugLabel is the one label of every drug term of fakeTerms.
const drugLabel = "Placeholder substance"

func (f fakeTerms) DrugLabelled(_ context.Context, id, label string) (string, bool, error) {
	atc, ok := f[id]
	return atc, ok && label == drugLabel, nil
}

func (f fakeTerms) DrugCodes(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if atc, ok := f[id]; ok {
			out[id] = atc
		}
	}
	return out, nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (f *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAudit) refs() []string {
	out := make([]string, len(f.events))
	for i, e := range f.events {
		out[i] = e.EntityReference
	}
	return out
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }
