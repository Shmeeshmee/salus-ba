package medicationstatement

import "context"

// Repository stores the medications, in the scope set on the transaction in ctx (00043's row-level security and its
// profile_entry_update_check trigger say the same as the service, as a second line).
type Repository interface {
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// ByPatient returns the patient's medications the scope sees, entered-in-error ones too, newest first.
	ByPatient(ctx context.Context, patientID string) ([]MedicationStatement, error)
	// ByID returns the medication id, ErrNotFound when the scope does not see it.
	ByID(ctx context.Context, id string) (MedicationStatement, error)
	// Insert writes a new medication, recorded (and last updated) at a.DateAsserted. ErrForbidden when the scope may
	// not write it, ErrInvalid when the table does not take it.
	Insert(ctx context.Context, a MedicationStatement) error
	// Update writes a's changeable fields over the version a.VersionID, last updated at a.LastUpdated, and returns
	// the new version: ErrVersionConflict when the entry is no longer at a.VersionID, ErrNotFound when the scope no
	// longer sees it, ErrForbidden when the scope may not make the change.
	Update(ctx context.Context, a MedicationStatement) (int32, error)
	// Delete deletes the medication id at version: ErrVersionConflict when it is no longer at it, ErrNotFound when the
	// scope no longer sees it, ErrForbidden when the scope may not delete it.
	Delete(ctx context.Context, id string, version int32) error
	// InsertProvenance appends the Provenance of a clinic's write. ErrForbidden when the scope may not write it (not
	// its clinic, an entry it does not see, or not at the entry's current version).
	InsertProvenance(ctx context.Context, p Provenance) error
	// Provenance returns the Provenance id, ErrNotFound when the scope does not see it.
	Provenance(ctx context.Context, id string) (Provenance, error)
}
