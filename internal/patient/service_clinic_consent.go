package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
)

// RecordConsent records, as the clinic's admin or receptionist c, the care consent that the patient id the clinic
// registered gave in person, when they have none active with the clinic: after an import (V5 of docs/clinic-cms
// WAVE4.md) or a withdrawal. consent.Service.RecordInPerson records it with c's account and audits it; the read of the
// profile is audited here, in the same transaction. ErrForbidden to anyone else, ErrNotFound for a profile the clinic
// did not register (also one that goes as the consent is recorded), consent.ErrActive when a care consent with the
// clinic is active already.
func (s *Service) RecordConsent(ctx context.Context, c Caller, id string) (consent.Consent, error) {
	if !c.desk() {
		return consent.Consent{}, s.refuseRights(ctx, c, auditevent.CodeCreate, c.purpose(), ErrForbidden)
	}
	var out consent.Consent
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.registered(ctx, c, id)
		if errors.Is(err, ErrNotFound) {
			refused = err
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		if err := s.record(ctx, c, auditevent.CodeRead, id, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		out, err = s.consents.RecordInPerson(ctx, consent.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, id)
		switch {
		case errors.Is(err, consent.ErrActive): // refused and audited by the consent domain, in this transaction
			refused, err = err, nil
		case errors.Is(err, consent.ErrNotFound): // the profile went in the same instant; refused and audited there too
			refused, err = ErrNotFound, nil
		}
		return err
	})
	switch {
	case err != nil:
		return consent.Consent{}, fmt.Errorf("patient: record consent: %w", err)
	case refused != nil:
		return consent.Consent{}, refused
	}
	return out, nil
}

// consentActive tells whether the patient id has an active care consent with the clinic org, in the transaction and
// scope in ctx.
func (s *Service) consentActive(ctx context.Context, id, org string) (bool, error) {
	state, err := s.consents.State(ctx, id, org)
	return state.Active, err
}
