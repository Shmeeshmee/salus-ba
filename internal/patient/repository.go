package patient

import (
	"context"
	"time"
)

// Repository stores profiles. Every read sees only the patient whose account is set by SetAccount, or, read
// only, the patients with an active consent for the clinic set by SetClinic (row-level security).
type Repository interface {
	// SetAccount scopes the rest of the transaction in ctx to the patient of the account id.
	SetAccount(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// RestrictForErasure restricts the record of the profile patientID at the clinic org (the scope set by SetClinic,
	// with the erasure scope of 00075) at at, as the erasure job's; nothing when the record is restricted there already.
	RestrictForErasure(ctx context.Context, patientID, org string, at time.Time) error
	// ByAccount returns the profile of the account, or ErrNotFound.
	ByAccount(ctx context.Context, accountID string) (Patient, error)
	// ByID returns the profile, or ErrNotFound (also for one outside the scope, or an id that is not a UUID).
	ByID(ctx context.Context, id string) (Patient, error)
	// Create stores p as the profile of p.AccountID, last updated at, and returns it; ErrConflict when the
	// account has one already.
	Create(ctx context.Context, p Patient, at time.Time) (Patient, error)
	// Update replaces the fields of the profile of p.AccountID when its version is p.VersionID, and returns it
	// with the next version; ErrConflict when no profile has that version.
	Update(ctx context.Context, p Patient, at time.Time) (Patient, error)
	// Register stores p as the profile of a patient without an account, registered by the clinic org (the scope set
	// by SetClinic), last updated at, and returns it.
	Register(ctx context.Context, p Patient, org string, at time.Time) (Patient, error)
	// Correct replaces the names, birth date and phone of the profile p.ID the clinic org registered when its version is
	// p.VersionID, last updated at, and returns it with the next version; ErrConflict when no such profile has that
	// version.
	Correct(ctx context.Context, p Patient, org string, at time.Time) (Patient, error)
	// SearchClinic returns at most limit patients of the clinic org (registered by it or with its active care consent)
	// matching q, by family and given name; for a practitioner's role roleID (empty for any) only those with an
	// appointment with it.
	SearchClinic(ctx context.Context, org, roleID string, q ClinicSearch, limit int32) ([]Patient, error)
	// Matching returns the patients of the clinic org (registered by it or with its active care consent) whose given
	// and family name (case does not matter) and birth date are those of a profile of in.
	Matching(ctx context.Context, org string, in []Patient) ([]Patient, error)
	// RegisteredBy returns the ids of the profiles the clinic org registered, in the transaction and scope in ctx.
	RegisteredBy(ctx context.Context, org string) ([]string, error)
	// Restrict restricts the patient's record at the clinic org (the scope set by SetClinic), by the staff account by,
	// for the request requestID ("" for none), at; nothing changes for a record restricted already.
	Restrict(ctx context.Context, patientID, org, by, requestID string, at time.Time) error
	// Unrestrict lifts a staff restriction of the patient's record at the clinic org; the erasure's stays.
	Unrestrict(ctx context.Context, patientID, org string) error
	// Restricted reports whether the patient's record is restricted at the clinic org, as the scope in ctx shows it.
	Restricted(ctx context.Context, patientID, org string) (bool, error)
}
