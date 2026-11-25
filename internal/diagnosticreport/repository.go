package diagnosticreport

import "context"

// Repository stores reports and their results (diagnostic_report, diagnostic_report_result). Package LB1 implements it
// in repository_postgres.go, and decides where the PDF is stored.
type Repository interface {
	// SetOrganization sets the clinic's scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// SetPatient sets the patient's scope on the transaction in ctx.
	SetPatient(ctx context.Context, patientID string) error
	// Insert stores a new preliminary report with its results and returns it with its id.
	Insert(ctx context.Context, r Report, createdBy string) (Report, error)
	// Update stores r's title, code, date and results if its version is still r.VersionID-1; false when it moved on.
	Update(ctx context.Context, r Report) (bool, error)
	// Release marks the report final by the practitioner at its version; false when it moved on.
	Release(ctx context.Context, id, releasedBy string, version int) (bool, error)
	// ByID returns the report visible in the scope, with its results, or ErrNotFound.
	ByID(ctx context.Context, id string) (Report, error)
	// ForAppointment returns the visit's reports visible in the scope, oldest first.
	ForAppointment(ctx context.Context, appointmentID string) ([]Report, error)
	// ForPatient returns the patient's released reports, newest first.
	ForPatient(ctx context.Context, patientID string) ([]Report, error)
}
