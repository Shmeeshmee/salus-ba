package practitionerrole

import (
	"context"
	"time"
)

// Repository stores practitioner roles. Every read sees only the clinic set by SetOrganization, or the active
// roles of listed clinics after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// ByID returns the role, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (PractitionerRole, error)
	// ByPractitioner returns the role of the practitioner, or ErrNotFound (also outside the scope, or for an
	// id that is not a UUID).
	ByPractitioner(ctx context.Context, practitionerID string) (PractitionerRole, error)
	// ByOrganization returns the roles of the clinic org, oldest first; none for an id that is not a UUID.
	ByOrganization(ctx context.Context, org string) ([]PractitionerRole, error)
	// PublicByOrganization returns the active roles of the clinic org that the scope shows, oldest first;
	// none for an id that is not a UUID.
	PublicByOrganization(ctx context.Context, org string) ([]PractitionerRole, error)
	// Create stores the active role of practitionerID in the clinic org and returns it; row-level security
	// refuses a clinic other than the one set.
	Create(ctx context.Context, org, practitionerID string) (PractitionerRole, error)
	// SetSpecialty sets the role's specialty code at version, moving it to the next version, last updated at
	// at; or returns ErrConflict when version is not the current one (or the role is outside the scope).
	SetSpecialty(ctx context.Context, id, code string, version int32, at time.Time) (PractitionerRole, error)
}

// StatsRepository reads the roles for the aggregate builds, on salus_stats.
type StatsRepository interface {
	// Specialties returns a clinic's roles with their specialty's SNOMED CT code, empty for none, by role id.
	Specialties(ctx context.Context, orgID string) (map[string]string, error)
}
