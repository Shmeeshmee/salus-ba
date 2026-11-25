package location_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until its package (LC1 of docs/clinic-cms/WAVE6.md) builds them, every locations method refuses and touches nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	s := location.NewService(location.Ports{Clock: clock.Fixed{T: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}})
	ctx := t.Context()
	admin := location.Staff{AccountID: "00000000-0000-4000-8000-000000000001", OrganizationID: "00000000-0000-4000-8000-000000000002", Admin: true}
	calls := map[string]func() error{
		"Locations": func() error { _, err := s.Locations(ctx, admin); return err },
		"SaveLocation": func() error {
			_, err := s.SaveLocation(ctx, admin, location.Location{Name: "Synthetic site", TimeZone: "Europe/Sarajevo"})
			return err
		},
		"PublicLocations": func() error { _, err := s.PublicLocations(ctx, admin.OrganizationID); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: %v, want errors.ErrUnsupported", name, err)
		}
	}
}
