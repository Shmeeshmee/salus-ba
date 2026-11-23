package consent

import (
	"context"
	"errors"
)

// RecordInPerson records, as the staff caller c, the care consent the clinic-registered patient patientID gave in
// person to c's clinic (W1 of docs/clinic-cms): given_in_person, with c's account as who recorded it. The wave 3
// contract: refused until package F2.
func (s *Service) RecordInPerson(ctx context.Context, c Caller, patientID string) (Consent, error) {
	return Consent{}, errors.ErrUnsupported
}
