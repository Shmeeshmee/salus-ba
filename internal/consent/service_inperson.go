package consent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// RecordInPerson records, as the staff caller c, the care consent the clinic-registered patient patientID gave in
// person to c's clinic (W1 of docs/clinic-cms): given_in_person, with c's account as who recorded it, in the clinic's
// scope and in the caller's transaction when there is one (the patient's registration), audited. ErrForbidden to a
// patient, ErrActive when the patient has an active care consent with the clinic already, ErrNotFound for a patient
// the clinic did not register (the database's guard, which aborts the transaction); each refusal is audited.
func (s *Service) RecordInPerson(ctx context.Context, c Caller, patientID string) (Consent, error) {
	if c.patient() {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	var out Consent
	var active bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.repo.GrantInPerson(ctx, patientID, c.OrganizationID, c.AccountID, s.clock.Now())
		if active = errors.Is(err, ErrActive); active {
			return s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrNotFound):
		// The insert aborted the transaction: the refusal is audited in a transaction of its own.
		return Consent{}, s.refuseTx(ctx, c, ErrNotFound)
	case err != nil:
		return Consent{}, fmt.Errorf("consent: record in person: %w", err)
	case active:
		return Consent{}, ErrActive
	}
	return out, nil
}

// refuseTx audits a refused create by c in a transaction of its own and returns refusal.
func (s *Service) refuseTx(ctx context.Context, c Caller, refusal error) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		return s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)
	})
	if err != nil {
		return fmt.Errorf("consent: record in person: %w", err)
	}
	return refusal
}
