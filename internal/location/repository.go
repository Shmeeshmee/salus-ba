package location

import "context"

// Repository stores locations, rooms and devices (location, room, device). Package LC1 implements the locations and
// RM1 the rooms and devices in repository_postgres.go.
type Repository interface {
	// SetOrganization sets the clinic's scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// Locations returns the clinic's locations, active first, by name.
	Locations(ctx context.Context, organizationID string) ([]Location, error)
	// PublicLocations returns the active locations of a listed clinic, by name.
	PublicLocations(ctx context.Context, organizationID string) ([]Location, error)
	// SaveLocation inserts l (no id) or updates it at l.VersionID-1; false when the version moved on.
	SaveLocation(ctx context.Context, l Location) (Location, bool, error)
	// Rooms returns the location's rooms, by name.
	Rooms(ctx context.Context, locationID string) ([]Room, error)
	// SaveRoom inserts r (no id) or updates it at r.VersionID-1; false when the version moved on.
	SaveRoom(ctx context.Context, r Room) (Room, bool, error)
	// Devices returns the location's devices, by name.
	Devices(ctx context.Context, locationID string) ([]Device, error)
	// SaveDevice inserts d (no id) or updates it at d.VersionID-1; false when the version moved on.
	SaveDevice(ctx context.Context, d Device) (Device, bool, error)
}
