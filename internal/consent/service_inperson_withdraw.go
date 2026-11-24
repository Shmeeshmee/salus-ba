package consent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// WithdrawInPerson records, as the staff caller c, that the patient patientID the clinic registered withdrew the care
// consent they gave it in person (docs/clinic-cms R2, V2), in the clinic's scope and in one transaction: the consent
// is revoked and the rest follows as for a patient's own revocation of a care consent (Revoke): the clinic's access
// ends at the commit and consent_revoked cancels the visits ahead and withdraws the waitlist entries. Audited with c's
// account. ErrForbidden to a patient, ErrInactive when the patient has no active consent with c's clinic given in
// person (one given in the app is the patient's to revoke); each refusal is audited.
func (s *Service) WithdrawInPerson(ctx context.Context, c Caller, patientID string) (Consent, error) {
	if c.patient() {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Consent
	var inactive bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.repo.RevokeInPerson(ctx, patientID, c.OrganizationID, s.clock.Now())
		if inactive = errors.Is(err, ErrInactive); inactive {
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		if err := s.revoked(ctx, c, out); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Consent{}, fmt.Errorf("consent: withdraw in person: %w", err)
	case inactive:
		return Consent{}, ErrInactive
	}
	return out, nil
}

// OfClinicPatient returns the consents the patient patientID gave the staff caller c's clinic, read in the clinic's
// scope and in the caller's transaction (the hand-over of a registered patient's record, docs/clinic-cms R2); each
// consent returned is audited. That the patient is one the clinic registered is the caller's check. ErrForbidden to a
// patient.
func (s *Service) OfClinicPatient(ctx context.Context, c Caller, patientID string) ([]Consent, error) {
	if c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeRead, ErrForbidden)
	}
	var out []Consent
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		all, err := s.repo.ByPatient(ctx, patientID)
		if err != nil {
			return err
		}
		for _, x := range all {
			if x.OrganizationID != c.OrganizationID || x.PatientID != patientID { // not on row-level security alone
				continue
			}
			if err := s.record(ctx, c, auditevent.CodeRead, x.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
			out = append(out, x)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("consent: of clinic patient: %w", err)
	}
	return out, nil
}
