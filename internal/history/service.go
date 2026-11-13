package history

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

// The ports of the history shared by the tickets that build it (docs/plans/patient-history.md "Contract"): each is
// implemented by the owning domain's service with this exact method, which applies access.ProfileEntry and
// access.Attributed and audits every entry it returns. The service methods of the history and their other ports come
// with SAL-39 (the patient's timeline, service_patient.go) and SAL-40 (the practitioner's view and summary).

// Allergies gives the allergies of the patient the caller may see, entered-in-error ones left out
// (allergyintolerance.Service.Seen, SAL-41).
type Allergies interface {
	Seen(ctx context.Context, c appointment.Caller, patientID string) ([]allergyintolerance.Seen, error)
}

// Medications gives the medications of the patient the caller may see, entered-in-error ones left out
// (medicationstatement.Service.Seen, SAL-41).
type Medications interface {
	Seen(ctx context.Context, c appointment.Caller, patientID string) ([]medicationstatement.Seen, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	// The sources of the patient's timeline (service_patient.go).
	Visits    Visits
	Documents Documents
	Responses Responses
	Files     Files
	Clinics   Clinics
	Roles     Roles
	Audit     Auditor
	Tx        Transactor
}

// Service builds the history from the services of the domains that own it; it holds no data of its own.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}
