package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Restrict restricts the patient's record at the admin's clinic (GDPR Art. 18; docs/clinic-cms/WAVE5.md S6, S7),
// optionally for the data request requestID ("" for none): hidden from the clinic's everyday screens and searches
// until lifted (access.Restricted). Restricting a restricted record changes nothing. Audited as an update of the
// Patient at the patient's request. ErrForbidden to anyone but the clinic's admin, ErrNotFound for a patient who is
// not the clinic's (registered by it, with a consent there, active or withdrawn, or a visit), each audited.
func (s *Service) Restrict(ctx context.Context, c Caller, id, requestID string) error {
	return s.restriction(ctx, c, id, func(ctx context.Context) error {
		return s.repo.Restrict(ctx, id, c.OrganizationID, c.AccountID, requestID, s.clock.Now())
	})
}

// LiftRestriction lifts the restriction of the patient's record at the admin's clinic (D4); lifting a record that is
// not restricted changes nothing. The erasure's restriction is not the clinic's to lift (S6): ErrForbidden. Audited
// and refused as Restrict.
func (s *Service) LiftRestriction(ctx context.Context, c Caller, id string) error {
	return s.restriction(ctx, c, id, func(ctx context.Context) error {
		if err := s.repo.Unrestrict(ctx, id, c.OrganizationID); err != nil {
			return err
		}
		kept, err := s.repo.Restricted(ctx, id, c.OrganizationID)
		if err == nil && kept {
			return ErrForbidden
		}
		return err
	})
}

// Restricted reports whether the patient's record is restricted at the clinic, in the caller's transaction and scope
// (the clinic's: a patient's scope sees no restriction). It is the access check inside the caller's audited access
// (consent.Service's State) and is not audited.
func (s *Service) Restricted(ctx context.Context, id, org string) (bool, error) {
	ok, err := s.repo.Restricted(ctx, id, org)
	if err != nil {
		return false, fmt.Errorf("patient: restricted: %w", err)
	}
	return ok, nil
}

// restriction runs change on the record of the clinic's patient id for the admin c, in one transaction with its audit.
func (s *Service) restriction(ctx context.Context, c Caller, id string, change func(ctx context.Context) error) error {
	if c.patient() || !c.Admin {
		return s.refuseRights(ctx, c, auditevent.CodeUpdate, auditevent.CodePATRQT, ErrForbidden)
	}
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		ok, err := s.ofClinic(ctx, c, id)
		if err == nil && ok {
			err = change(ctx)
		} else if err == nil {
			err = ErrNotFound
		}
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) {
			refused = err
			return s.audit.Record(ctx, s.event(c, auditevent.CodeUpdate, auditevent.CodePATRQT, "", auditevent.OutcomeMinorFailure))
		}
		if err != nil {
			return err
		}
		return s.audit.Record(ctx, s.event(c, auditevent.CodeUpdate, auditevent.CodePATRQT, id, auditevent.OutcomeSuccess))
	})
	switch {
	case err != nil:
		return fmt.Errorf("patient: restriction: %w", err)
	case refused != nil:
		return refused
	}
	return nil
}

// ofClinic scopes the transaction in ctx to c's clinic and reports whether the patient id is the clinic's: registered
// by it, with a consent there (active or withdrawn), or with a visit there.
func (s *Service) ofClinic(ctx context.Context, c Caller, id string) (bool, error) {
	if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return false, err
	}
	state, err := s.consents.State(ctx, id, c.OrganizationID)
	if err != nil || state.Active || !state.RevokedAt.IsZero() {
		return err == nil, err
	}
	visits, err := s.appointments.Visits(ctx, id, c.OrganizationID)
	if err != nil || len(visits) > 0 {
		return err == nil, err
	}
	p, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return err == nil && p.RegisteredBy == c.OrganizationID, err
}
