package organization_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A verified signup creates its clinic from the API (docs/clinic-cms, O1): as salus_app, in the new clinic's own
// scope, unlisted. Create leaves the transaction in that scope, so what follows (the admin's invite, the trial) writes
// into the new clinic only.
func TestCreate_AsTheAppInTheNewClinicsScopeUnlisted(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := organization.NewPostgresRepository(app)

	var created, read organization.Organization
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		var err error
		if created, err = repo.Create(ctx, "Synthetic clinic"); err != nil {
			return err
		}
		read, err = repo.ByID(ctx, created.ID) // the scope is the new clinic's
		return err
	})
	if err != nil || created.ID == "" || created.Name != "Synthetic clinic" || created.Listed {
		t.Fatalf("Create as %s = %+v, %v; want an unlisted clinic", pgtest.AppRole, created, err)
	}
	if read.ID != created.ID {
		t.Errorf("ByID(new clinic) after Create = %+v, want the new clinic", read)
	}
}

// salusctl signup approve lists the clinic, once (as the owner).
func TestList_ListsAnUnlistedClinicOnce(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(owner)
	for i, want := range []bool{true, false} {
		if got, err := repo.List(t.Context(), clinicA); err != nil || got != want {
			t.Errorf("List(clinic A) #%d = %v, %v; want %v", i+1, got, err, want)
		}
	}
	if got, err := repo.List(t.Context(), "not-a-uuid"); err != nil || got {
		t.Errorf("List(not a uuid) = %v, %v; want false", got, err)
	}
	var listed bool
	if err := owner.QueryRow(t.Context(), "SELECT listed FROM organization WHERE id = $1", clinicA).Scan(&listed); err != nil || !listed {
		t.Errorf("clinic A listed = %v, %v", listed, err)
	}
}
