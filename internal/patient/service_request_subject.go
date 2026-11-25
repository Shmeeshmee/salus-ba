package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// RequestSubject returns, to the clinic's admin c, the patient a data request targeting the clinic names, so the admin
// knows whose request it is (docs/clinic-cms/walk/wave5 finding 3): the names, the birth date and the clinic that
// registered them, nothing else of the profile. It reads in the clinic's scope widened to the patients its requests
// name (whatever their consent or restriction: answering the request is the purpose), closed again after the read, in
// the caller's transaction, audited as a read of the Patient at the patient's request (the caller's context names the
// request). ErrForbidden to anyone but the admin and ErrNotFound for a patient no request of the clinic names, each
// audited without the id.
func (s *Service) RequestSubject(ctx context.Context, c Caller, id string) (Patient, error) {
	if c.patient() || !c.Admin {
		return Patient{}, s.refuseRights(ctx, c, auditevent.CodeRead, auditevent.CodePATRQT, ErrForbidden)
	}
	var out Patient
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := s.repo.SetDataRequests(ctx, true); err != nil {
			return err
		}
		p, err := s.repo.ByID(ctx, id)
		if missing = errors.Is(err, ErrNotFound); err != nil && !missing {
			return err
		}
		if err := s.repo.SetDataRequests(ctx, false); err != nil {
			return err
		}
		if missing {
			return s.audit.Record(ctx, s.event(c, auditevent.CodeRead, auditevent.CodePATRQT, "", auditevent.OutcomeMinorFailure))
		}
		out = Patient{ID: p.ID, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate, RegisteredBy: p.RegisteredBy}
		return s.audit.Record(ctx, s.event(c, auditevent.CodeRead, auditevent.CodePATRQT, p.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case err != nil:
		return Patient{}, fmt.Errorf("patient: request subject: %w", err)
	case missing:
		return Patient{}, ErrNotFound
	}
	return out, nil
}
