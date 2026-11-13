package allergyintolerance_test

import (
	"context"
	"errors"
	"testing"

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
	stale.ID, stale.Activity, stale.Version = "a1000000-0000-4000-8000-0000000000ab", fhir.ActivityUpdate, 2
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
