package questionnaire

import (
	"context"
	"time"
)

// Repository stores questionnaires and the roles' defaults. Every read and write sees only the clinic set
// by SetOrganization (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// ByID returns the version id, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Questionnaire, error)
	// Lock returns the version id, locked until the transaction ends so lifecycle steps of one version run
	// one after the other, or ErrNotFound.
	Lock(ctx context.Context, id string) (Questionnaire, error)
	// Search returns the versions the filter and the scope give, oldest first.
	Search(ctx context.Context, f Filter) ([]Questionnaire, error)
	// Create stores q's content as version 1 of a new series, a draft of q's clinic changed at at.
	Create(ctx context.Context, q Questionnaire, at time.Time) (Questionnaire, error)
	// CreateVersion stores a copy of from's content as a draft of its series, at the series' next version
	// number, changed at at; or returns ErrDraftExists when the series has a draft.
	CreateVersion(ctx context.Context, from Questionnaire, at time.Time) (Questionnaire, error)
	// UpdateDraft sets the draft q.ID's content when q.VersionID is its current version, moving it to the
	// next, changed at at; or returns ErrConflict (also when it is no longer a draft).
	UpdateDraft(ctx context.Context, q Questionnaire, at time.Time) (Questionnaire, error)
	// DeleteDraft deletes the draft id, or returns ErrNotFound when there is no such draft.
	DeleteDraft(ctx context.Context, id string) error
	// Publish makes the draft id active at at, or returns ErrConflict when it is not a draft.
	Publish(ctx context.Context, id string, at time.Time) (Questionnaire, error)
	// Retire retires the active version id at at, or returns ErrConflict when it is not active.
	Retire(ctx context.Context, id string, at time.Time) (Questionnaire, error)
	// RetireActive retires the active version of the series, if it has one, at at.
	RetireActive(ctx context.Context, seriesID string, at time.Time) error
	// Default returns the default of the role roleID, or ErrNotFound when it has none.
	Default(ctx context.Context, roleID string) (Default, error)
	// SetDefault stores d as its role's default, replacing the one it has, changed at at.
	SetDefault(ctx context.Context, d Default, at time.Time) (Default, error)
	// ClearDefault removes the default of the role roleID, if it has one.
	ClearDefault(ctx context.Context, roleID string) error
}
