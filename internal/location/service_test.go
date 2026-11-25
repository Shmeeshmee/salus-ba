package location_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Until their packages (LC1 and RM1 of docs/clinic-cms/WAVE6.md) build them, every method refuses and touches nothing.
func TestService_RefusesUntilItsPackageBuildsIt(t *testing.T) {
	t.Parallel()
	s := location.NewService(location.Ports{Clock: clock.Fixed{T: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}})
	ctx := t.Context()
	admin := location.Staff{AccountID: "00000000-0000-4000-8000-000000000001", OrganizationID: "00000000-0000-4000-8000-000000000002", Admin: true}
	const id = "00000000-0000-4000-8000-000000000003"
	calls := map[string]func() error{
		"Locations": func() error { _, err := s.Locations(ctx, admin); return err },
		"SaveLocation": func() error {
			_, err := s.SaveLocation(ctx, admin, location.Location{Name: "Synthetic site", TimeZone: "Europe/Sarajevo"})
			return err
		},
		"PublicLocations": func() error { _, err := s.PublicLocations(ctx, admin.OrganizationID); return err },
		"Rooms":           func() error { _, err := s.Rooms(ctx, admin, id); return err },
		"SaveRoom": func() error {
			_, err := s.SaveRoom(ctx, admin, location.Room{LocationID: id, Name: "Room 1"})
			return err
		},
		"Devices": func() error { _, err := s.Devices(ctx, admin, id); return err },
		"SaveDevice": func() error {
			_, err := s.SaveDevice(ctx, admin, location.Device{LocationID: id, Name: "Ultrasound"})
			return err
		},
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: %v, want errors.ErrUnsupported", name, err)
		}
	}
}
