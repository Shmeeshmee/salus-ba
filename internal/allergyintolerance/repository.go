package allergyintolerance

import "context"

// Repository stores the allergies, in the scope set on the transaction in ctx (00043's row-level security and its
// profile_entry_update_check trigger say the same as the service, as a second line).
type Repository interface {
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// ByPatient returns the patient's allergies the scope sees, entered-in-error ones too, newest first.
	ByPatient(ctx context.Context, patientID string) ([]AllergyIntolerance, error)
	// ByID returns the allergy id, ErrNotFound when the scope does not see it.
	ByID(ctx context.Context, id string) (AllergyIntolerance, error)
	// Insert writes a new allergy, recorded (and last updated) at a.RecordedDate. ErrForbidden when the scope may
	// not write it, ErrInvalid when the table does not take it.
	Insert(ctx context.Context, a AllergyIntolerance) error
	// Update writes a's changeable fields over the version a.VersionID, last updated at a.LastUpdated, and returns
	// the new version: ErrVersionConflict when the entry is no longer at a.VersionID, ErrForbidden when the scope
	// may not make the change.
	Update(ctx context.Context, a AllergyIntolerance) (int32, error)
	// Delete deletes the allergy id at version: ErrVersionConflict when it is no longer at it (or the scope may not
	// delete it).
	Delete(ctx context.Context, id string, version int32) error
}
