package medicationstatement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The clinic appends the Provenance of its write and reads it back as written; the patient reads it; another
// clinic's scope neither sees it nor writes one; a Provenance of another version than the entry's is refused.
func TestRepository_AppendsAndReadsProvenances(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetClinic(ctx, v.clinic) }
	other := func(ctx context.Context, r medicationstatement.Repository) error {
		return r.SetClinic(ctx, "00000000-0000-4000-8000-0000000000ff")
	}
	recorded := medicationstatement.MedicationStatement{ID: clinicID, PatientID: v.patient, DateAsserted: at,
		Source:       medicationstatement.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation: medicationstatement.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:         medicationstatement.Code{Text: "Placeholder medication"}, Status: medicationstatement.StatusActive}
	p := medicationstatement.Provenance{Recorded: at, ID: "a1000000-0000-4000-8000-0000000000aa", EntryID: clinicID, RoleID: v.role,
		OrganizationID: v.clinic, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1}
	if err := in(t, app, clinic, func(ctx context.Context, r medicationstatement.Repository) error {
		if err := r.Insert(ctx, recorded); err != nil {
			return err
		}
		return r.InsertProvenance(ctx, p)
	}); err != nil {
		t.Fatalf("the clinic records with its Provenance: %v", err)
	}
	for name, scope := range map[string]func(ctx context.Context, r medicationstatement.Repository) error{"the clinic": clinic, "the patient": patient} {
		var got medicationstatement.Provenance
		err := in(t, app, scope, func(ctx context.Context, r medicationstatement.Repository) (err error) {
			got, err = r.Provenance(ctx, p.ID)
			return err
		})
		if err != nil || got != p {
			t.Errorf("%s reads %+v, %v; want it as written", name, got, err)
		}
	}
	for name, id := range map[string]string{"another clinic's scope": p.ID, "an id that is none": "x"} {
		err := in(t, app, other, func(ctx context.Context, r medicationstatement.Repository) error {
			_, err := r.Provenance(ctx, id)
			return err
		})
		if !errors.Is(err, medicationstatement.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	stale := p
	stale.ID, stale.Activity, stale.Version = "a1000000-0000-4000-8000-0000000000ab", fhir.ActivityUpdate, 3
	for name, tt := range map[string]struct {
		scope func(ctx context.Context, r medicationstatement.Repository) error
		p     medicationstatement.Provenance
	}{
		"another clinic's scope": {other, func() medicationstatement.Provenance { q := p; q.ID = stale.ID; return q }()},
		"another version":        {clinic, stale},
		"an id that is no uuid":  {clinic, func() medicationstatement.Provenance { q := p; q.ID = "x"; return q }()},
	} {
		err := in(t, app, tt.scope, func(ctx context.Context, r medicationstatement.Repository) error {
			return r.InsertProvenance(ctx, tt.p)
		})
		if !errors.Is(err, medicationstatement.ErrForbidden) && !errors.Is(err, medicationstatement.ErrInvalid) {
			t.Errorf("%s: %v, want it refused", name, err)
		}
	}
}

// Whether an entry has a Provenance, in the patient's scope (Remove deletes only one without); the patient's own
// Provenance of a clinic's entry they added to their profile is read back as written.
func TestRepository_TellsAWrittenEntryAndAppendsThePatientsProvenance(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r medicationstatement.Repository) error { return r.SetClinic(ctx, v.clinic) }
	recorded := medicationstatement.MedicationStatement{ID: clinicID, PatientID: v.patient, DateAsserted: at,
		Source:       medicationstatement.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation: medicationstatement.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:         medicationstatement.Code{Text: "Placeholder medication"}, Status: medicationstatement.StatusActive}
	if err := in(t, app, clinic, func(ctx context.Context, r medicationstatement.Repository) error {
		if err := r.Insert(ctx, recorded); err != nil {
			return err
		}
		return r.InsertProvenance(ctx, medicationstatement.Provenance{Recorded: at, ID: "a1000000-0000-4000-8000-0000000000aa", EntryID: clinicID,
			RoleID: v.role, OrganizationID: v.clinic, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1})
	}); err != nil {
		t.Fatalf("the clinic records with its Provenance: %v", err)
	}
	for name, tt := range map[string]struct {
		id   string
		want bool
	}{"the written entry": {clinicID, true}, "another": {ownID, false}, "no id": {"x", false}} {
		var got bool
		err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) (err error) {
			got, err = r.HasProvenance(ctx, tt.id)
			return err
		})
		if err != nil || got != tt.want {
			t.Errorf("has a Provenance, %s: %v, %v; want %v", name, got, err, tt.want)
		}
	}
	// The patient adds the clinic's entry to their profile: their Provenance, the patient the agent.
	added := recorded
	added.InProfile, added.VersionID, added.LastUpdated = true, 1, at.Add(time.Hour)
	byPatient := medicationstatement.Provenance{Recorded: added.LastUpdated, ID: "a1000000-0000-4000-8000-0000000000ac", EntryID: clinicID,
		PatientID: v.patient, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentAuthor, Version: 2}
	var read medicationstatement.Provenance
	if err := in(t, app, patient, func(ctx context.Context, r medicationstatement.Repository) (err error) {
		if _, err := r.Update(ctx, added); err != nil {
			return err
		}
		if err := r.InsertProvenance(ctx, byPatient); err != nil {
			return err
		}
		read, err = r.Provenance(ctx, byPatient.ID)
		return err
	}); err != nil || read != byPatient {
		t.Errorf("the patient's Provenance: %+v, %v; want it as written", read, err)
	}
}
