package diagnosticreport

import (
	"context"
	"time"
)

// Repository stores reports and their results (diagnostic_report, diagnostic_report_result). Implemented in
// repository_postgres.go.
type Repository interface {
	// SetOrganization sets the clinic's scope on the transaction in ctx.
	SetOrganization(ctx context.Context, organizationID string) error
	// SetPatient sets the patient's scope on the transaction in ctx.
	SetPatient(ctx context.Context, patientID string) error
	// Insert stores a new preliminary report with its results and returns it with its id.
	Insert(ctx context.Context, r Report, createdBy string) (Report, error)
	// Update stores r's title, code, date and results if its version is still r.VersionID-1; false when it moved on.
	Update(ctx context.Context, r Report) (bool, error)
	// Touch moves the preliminary report to its next version if it is still at version (its PDF changed); false when
	// it moved on or was released.
	Touch(ctx context.Context, id string, version int) (bool, error)
	// Release marks the report final by the practitioner at its version, at the time at; false when it moved on.
	Release(ctx context.Context, id, releasedBy string, version int, at time.Time) (bool, error)
	// ByID returns the report visible in the scope, with its results, or ErrNotFound.
	ByID(ctx context.Context, id string) (Report, error)
	// ForAppointment returns the visit's reports visible in the scope, oldest first.
	ForAppointment(ctx context.Context, appointmentID string) ([]Report, error)
	// ForPatient returns the patient's released reports, newest first.
	ForPatient(ctx context.Context, patientID string) ([]Report, error)
}
