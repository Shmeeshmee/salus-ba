package careplan

import "context"

// Repository stores plans and their steps (care_plan, care_plan_activity). Package TP1 implements it in
// repository_postgres.go.
type Repository interface {
	// SetOrganization sets the clinic's scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// SetPatient sets the patient's scope on the transaction in ctx.
	SetPatient(ctx context.Context, patientID string) error
	// Insert stores a new draft with its steps and returns it with its id.
	Insert(ctx context.Context, p Plan, createdBy string) (Plan, error)
	// Update stores p's status, title, quote, dates and steps if its version is still p.VersionID-1; false when it
	// moved on.
	Update(ctx context.Context, p Plan, changedBy string) (bool, error)
	// ByID returns the plan visible in the scope, with its steps in order, or ErrNotFound.
	ByID(ctx context.Context, id string) (Plan, error)
	// ForPatient returns the patient's plans visible in the scope, newest first.
	ForPatient(ctx context.Context, patientID string) ([]Plan, error)
}
