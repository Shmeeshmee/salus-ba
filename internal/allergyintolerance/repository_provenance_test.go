package allergyintolerance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The clinic appends the Provenance of its write and reads it back as written; the patient reads it; another
// clinic's scope neither sees it nor writes one; a Provenance of another version than the entry's is refused.
func TestRepository_AppendsAndReadsProvenances(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetClinic(ctx, v.clinic) }
	other := func(ctx context.Context, r allergyintolerance.Repository) error {
		return r.SetClinic(ctx, "00000000-0000-4000-8000-0000000000ff")
	}
	recorded := allergyintolerance.AllergyIntolerance{ID: clinicID, PatientID: v.patient, RecordedDate: at,
		Source:         allergyintolerance.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation:   allergyintolerance.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:           allergyintolerance.Code{Text: "Placeholder food"},
		ClinicalStatus: allergyintolerance.ClinicalActive, VerificationStatus: allergyintolerance.VerificationConfirmed}
	p := allergyintolerance.Provenance{Recorded: at, ID: "a1000000-0000-4000-8000-0000000000aa", EntryID: clinicID, RoleID: v.role,
		OrganizationID: v.clinic, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1}
	if err := in(t, app, clinic, func(ctx context.Context, r allergyintolerance.Repository) error {
		if err := r.Insert(ctx, recorded); err != nil {
			return err
		}
		return r.InsertProvenance(ctx, p)
	}); err != nil {
		t.Fatalf("the clinic records with its Provenance: %v", err)
	}
	for name, scope := range map[string]func(ctx context.Context, r allergyintolerance.Repository) error{"the clinic": clinic, "the patient": patient} {
		var got allergyintolerance.Provenance
		err := in(t, app, scope, func(ctx context.Context, r allergyintolerance.Repository) (err error) {
			got, err = r.Provenance(ctx, p.ID)
			return err
		})
		if err != nil || got != p {
			t.Errorf("%s reads %+v, %v; want it as written", name, got, err)
		}
	}
	for name, id := range map[string]string{"another clinic's scope": p.ID, "an id that is none": "x"} {
		err := in(t, app, other, func(ctx context.Context, r allergyintolerance.Repository) error {
			_, err := r.Provenance(ctx, id)
			return err
		})
		if !errors.Is(err, allergyintolerance.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	stale := p
	stale.ID, stale.Activity, stale.Version = "a1000000-0000-4000-8000-0000000000ab", fhir.ActivityUpdate, 3
	for name, tt := range map[string]struct {
		scope func(ctx context.Context, r allergyintolerance.Repository) error
		p     allergyintolerance.Provenance
	}{
		"another clinic's scope": {other, func() allergyintolerance.Provenance { q := p; q.ID = stale.ID; return q }()},
		"another version":        {clinic, stale},
		"an id that is no uuid":  {clinic, func() allergyintolerance.Provenance { q := p; q.ID = "x"; return q }()},
	} {
		err := in(t, app, tt.scope, func(ctx context.Context, r allergyintolerance.Repository) error { return r.InsertProvenance(ctx, tt.p) })
		if !errors.Is(err, allergyintolerance.ErrForbidden) && !errors.Is(err, allergyintolerance.ErrInvalid) {
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
	patient := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetClinic(ctx, v.clinic) }
	recorded := allergyintolerance.AllergyIntolerance{ID: clinicID, PatientID: v.patient, RecordedDate: at,
		Source:         allergyintolerance.Source{OrganizationID: v.clinic, EncounterID: v.encounter, RecorderRoleID: v.role},
		Confirmation:   allergyintolerance.Confirmation{At: at, RoleID: v.role, OrganizationID: v.clinic},
		Code:           allergyintolerance.Code{Text: "Placeholder food"},
		ClinicalStatus: allergyintolerance.ClinicalActive, VerificationStatus: allergyintolerance.VerificationConfirmed}
	if err := in(t, app, clinic, func(ctx context.Context, r allergyintolerance.Repository) error {
		if err := r.Insert(ctx, recorded); err != nil {
			return err
		}
		return r.InsertProvenance(ctx, allergyintolerance.Provenance{Recorded: at, ID: "a1000000-0000-4000-8000-0000000000aa", EntryID: clinicID,
			RoleID: v.role, OrganizationID: v.clinic, Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1})
	}); err != nil {
		t.Fatalf("the clinic records with its Provenance: %v", err)
	}
	for name, tt := range map[string]struct {
		id   string
		want bool
	}{"the written entry": {clinicID, true}, "another": {ownID, false}, "no id": {"x", false}} {
		var got bool
		err := in(t, app, patient, func(ctx context.Context, r allergyintolerance.Repository) (err error) {
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
	byPatient := allergyintolerance.Provenance{Recorded: added.LastUpdated, ID: "a1000000-0000-4000-8000-0000000000ac", EntryID: clinicID,
		PatientID: v.patient, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentAuthor, Version: 2}
	var read allergyintolerance.Provenance
	if err := in(t, app, patient, func(ctx context.Context, r allergyintolerance.Repository) (err error) {
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

// A delete that would lose a clinic's Provenance fails as a conflict: the clinic confirmed the patient's own entry,
// the patient's change cleared the confirmation, and the entry is no longer what the patient may delete.
func TestRepository_ADeleteThatWouldLoseAProvenanceIsAConflict(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	v := seedVisit(t, owner)
	patient := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetPatient(ctx, v.patient) }
	clinic := func(ctx context.Context, r allergyintolerance.Repository) error { return r.SetClinic(ctx, v.clinic) }
	own := allergyintolerance.AllergyIntolerance{ID: ownID, PatientID: v.patient, RecordedDate: at, InProfile: true,
		Code: allergyintolerance.Code{Text: "Placeholder substance"}, ClinicalStatus: allergyintolerance.ClinicalActive,
		VerificationStatus: allergyintolerance.VerificationUnconfirmed}
	write := func(scope func(ctx context.Context, r allergyintolerance.Repository) error, fn func(ctx context.Context, r allergyintolerance.Repository) error) {
		t.Helper()
		if err := in(t, app, scope, fn); err != nil {
			t.Fatal(err)
		}
	}
	write(patient, func(ctx context.Context, r allergyintolerance.Repository) error { return r.Insert(ctx, own) })
	confirmed := own
	confirmed.VersionID, confirmed.LastUpdated = 1, at.Add(time.Minute)
	confirmed.Confirmation = allergyintolerance.Confirmation{At: at.Add(time.Minute), RoleID: v.role, OrganizationID: v.clinic}
	confirmed.VerificationStatus = allergyintolerance.VerificationConfirmed
	write(clinic, func(ctx context.Context, r allergyintolerance.Repository) error {
		if _, err := r.Update(ctx, confirmed); err != nil {
			return err
		}
		return r.InsertProvenance(ctx, allergyintolerance.Provenance{Recorded: confirmed.LastUpdated, ID: "a1000000-0000-4000-8000-0000000000ad",
			EntryID: ownID, RoleID: v.role, OrganizationID: v.clinic, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentVerifier, Version: 2})
	})
	changed := own
	changed.VersionID, changed.LastUpdated, changed.Reaction = 2, at.Add(time.Hour), "Placeholder rash"
	write(patient, func(ctx context.Context, r allergyintolerance.Repository) error {
		_, err := r.Update(ctx, changed)
		return err
	})
	err := in(t, app, patient, func(ctx context.Context, r allergyintolerance.Repository) error { return r.Delete(ctx, ownID, 3) })
	if !errors.Is(err, allergyintolerance.ErrVersionConflict) {
		t.Errorf("the delete: %v, want ErrVersionConflict", err)
	}
}
