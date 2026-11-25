package chargeitemdefinition_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A service may need a device of its clinic (U4 of docs/clinic-cms/WAVE6.md, RM1): stored, listed, cleared by a
// change without one; another clinic's device or an id that is not one is ErrInvalid.
func TestPostgres_AServicesDevice(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	var deviceA, deviceB string
	if err := owner.QueryRow(t.Context(), `WITH l AS (INSERT INTO location (organization_id, name) VALUES ($1, 'Site A'), ($2, 'Site B') RETURNING id, organization_id),
		d AS (INSERT INTO device (organization_id, location_id, name) SELECT organization_id, id, 'Ultrasound' FROM l RETURNING id, organization_id)
		SELECT (SELECT id::text FROM d WHERE organization_id = $1), (SELECT id::text FROM d WHERE organization_id = $2)`, clinicA, clinicB).
		Scan(&deviceA, &deviceB); err != nil {
		t.Fatal(err)
	}
	s := store{t: t, app: app, repo: chargeitemdefinition.NewPostgresRepository(app)}
	in := chargeitemdefinition.Definition{Name: "Synthetic scan", PriceMinor: 5000, Modes: []schedule.VisitMode{schedule.ModeInPerson}, Active: true, DeviceID: deviceA}
	created := s.create(clinicA, in)
	list := func() []chargeitemdefinition.Definition {
		var out []chargeitemdefinition.Definition
		if err := scoped(t, app, s.repo, clinicA, func(ctx context.Context) (err error) { out, err = s.repo.List(ctx, clinicA); return err }); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if got := list(); len(got) != 1 || got[0].DeviceID != deviceA || created.DeviceID != deviceA {
		t.Fatalf("listed %+v, want the service with device A", got)
	}
	created.DeviceID = ""
	if err := scoped(t, app, s.repo, clinicA, func(ctx context.Context) error { _, err := s.repo.Update(ctx, created, now); return err }); err != nil {
		t.Fatal(err)
	}
	if got := list(); got[0].DeviceID != "" {
		t.Errorf("after a change without a device: %q, want none", got[0].DeviceID)
	}
	for name, device := range map[string]string{"another clinic's device": deviceB, "not a UUID": "x"} {
		d := in
		d.DeviceID, d.OrganizationID, d.Currency = device, clinicA, "BAM"
		if err := scoped(t, app, s.repo, clinicA, func(ctx context.Context) error { _, err := s.repo.Create(ctx, d, now); return err }); !errors.Is(err, chargeitemdefinition.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, err)
		}
	}
}
