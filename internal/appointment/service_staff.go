package appointment

import (
	"context"
	"errors"
	"time"
)

// StaffBooking is a visit staff book for a patient of their clinic (package F2 of docs/clinic-cms): the patient,
// any free slot of the clinic, and optionally the service (empty for none).
type StaffBooking struct {
	PatientID string
	SlotID    string
	ServiceID string
}

// BookForPatient books b for its patient as the staff caller c. The wave 3 contract: refused until package F2.
func (s *Service) BookForPatient(ctx context.Context, c Caller, b StaffBooking) (Appointment, error) {
	return Appointment{}, errors.ErrUnsupported
}

// MoveForPatient moves the appointment id to the slot slotID as the staff caller c. Refused until package F2.
func (s *Service) MoveForPatient(ctx context.Context, c Caller, id, slotID string) (Appointment, error) {
	return Appointment{}, errors.ErrUnsupported
}

// CancelForPatient cancels the appointment id for its patient as the staff caller c. Refused until package F2.
func (s *Service) CancelForPatient(ctx context.Context, c Caller, id string) (Appointment, error) {
	return Appointment{}, errors.ErrUnsupported
}

// Arrive marks the in-person appointment id arrived at the desk (StatusArrived). Refused until package F2.
func (s *Service) Arrive(ctx context.Context, c Caller, id string) (Appointment, error) {
	return Appointment{}, errors.ErrUnsupported
}

// DeskDay is the clinic's visits of the calendar day day in the clinic's zone, by practitioner and time, for the
// desk (package F3's screen). Refused until package F2.
func (s *Service) DeskDay(ctx context.Context, c Caller, day time.Time) ([]Appointment, error) {
	return nil, errors.ErrUnsupported
}
