package patient

import (
	"context"
	"errors"
	"time"
)

// ClinicSearch finds the clinic's patients (package F2 of docs/clinic-cms): by name and, when given, birth date.
// The clinic's patients are those with its care consent and those it registered.
type ClinicSearch struct {
	BirthDate time.Time // zero for any
	Name      string    // given or family name, or both
}

// SearchClinic returns the patients of the staff caller's clinic that match q. The wave 3 contract: refused until
// package F2.
func (s *Service) SearchClinic(ctx context.Context, c Caller, q ClinicSearch) ([]Patient, error) {
	return nil, errors.ErrUnsupported
}

// Register stores the profile of a patient who has no Salus account, registered by the staff caller's clinic and
// seen by that clinic only (W1 of docs/clinic-cms). A patient's own profile is unchanged. Refused until package F2.
func (s *Service) Register(ctx context.Context, c Caller, in Patient) (Patient, error) {
	return Patient{}, errors.ErrUnsupported
}
