package organization

import (
	"context"
	"time"
)

// Repository stores clinics. Every read sees only the clinic set by SetOrganization, or the listed clinics
// after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// ByID returns the clinic, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (Organization, error)
	// Listed returns the listed clinics the scope shows, by name.
	Listed(ctx context.Context) ([]Organization, error)
	// Update sets the name, listing, time zone and default language of the clinic o.ID at version o.VersionID, moving it to the
	// next version, last updated at at; or returns ErrConflict when that version is not the current one.
	Update(ctx context.Context, o Organization, at time.Time) (Organization, error)
	// ListedSpeaking returns the listed clinics the scope shows whose staff speak language, by name.
	ListedSpeaking(ctx context.Context, language string) ([]Organization, error)
	// ProfileOf returns the profile of the clinic id at its version, or ErrNotFound for one the scope does not show.
	ProfileOf(ctx context.Context, id string) (Profile, error)
	// PutProfile replaces the profile of the clinic id at version p.VersionID, moving the clinic to the next version,
	// last updated at at; or returns ErrConflict when that version is not the current one.
	PutProfile(ctx context.Context, id string, p Profile, at time.Time) (Profile, error)
	// Create stores a new active, unlisted clinic and returns it, setting its scope on the transaction in ctx: the
	// operator command as the owner, or a verified signup as salus_app (00066).
	Create(ctx context.Context, name string) (Organization, error)
	// List puts the unlisted clinic id in the public directory; false when there is no such unlisted clinic. Only the
	// owner may (salusctl signup approve).
	List(ctx context.Context, id string) (bool, error)
}

// ChatRepository stores the clinics' chat settings (migration 00047), row-level secured like the clinics: every
// method sees only the clinic set by SetOrganization.
type ChatRepository interface {
	SetOrganization(ctx context.Context, id string) error
	ByID(ctx context.Context, id string) (Organization, error)
	// ChatSettings returns the clinic's settings, and false when it has none stored.
	ChatSettings(ctx context.Context, org string) (ChatSettings, bool, error)
	// PutChatSettings stores the clinic's settings, last updated at at.
	PutChatSettings(ctx context.Context, org string, s ChatSettings, at time.Time) error
}

// AppointmentsRepository stores the clinics' appointment settings (migration 00054), row-level secured like the
// clinics: every method sees only the clinic set by SetOrganization.
type AppointmentsRepository interface {
	SetOrganization(ctx context.Context, id string) error
	// AppointmentSettings returns the clinic's settings, and false when it has none stored.
	AppointmentSettings(ctx context.Context, org string) (AppointmentSettings, bool, error)
	// PutAppointmentSettings stores the clinic's settings read at s.Version, last updated at at, and returns the new
	// version; ErrSettingsChanged when the stored version is no longer s.Version.
	PutAppointmentSettings(ctx context.Context, org string, s AppointmentSettings, at time.Time) (int, error)
}

// StatsRepository reads the clinics for the aggregate builds, on salus_stats.
type StatsRepository interface {
	// Locations returns every clinic with its time zone, by id.
	Locations(ctx context.Context) ([]Location, error)
}
