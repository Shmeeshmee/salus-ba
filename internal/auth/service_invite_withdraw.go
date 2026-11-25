package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// WithdrawInvite ends the open invite id of the clinic of the admin signed in with sess, and its e-mailed link with
// it (DELETE /invites/{id}, U7 of docs/clinic-cms/WAVE6.md). Anyone but an admin of the clinic gets ErrForbidden; an
// id that is no open invite of the clinic gets ErrInvalidInvite. Audited as a delete of the invite, also when
// refused (then without an entity: nothing was reached).
func (s *Service) WithdrawInvite(ctx context.Context, sess Session, id string) error {
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		admin, ok, err := s.clinicAdmin(ctx, sess)
		if err != nil {
			return err
		}
		if !ok {
			refused = ErrForbidden
		} else if err := s.repo.SetOrganization(ctx, admin.OrganizationID); err != nil {
			return err
		} else if err := s.repo.WithdrawInvite(ctx, id, s.clock.Now()); errors.Is(err, ErrInvalidInvite) {
			refused = err
		} else if err != nil {
			return err
		}
		if refused != nil {
			return s.audit.Record(ctx, inviteEvent(auditevent.CodeDelete, auditevent.ActionDelete, admin, "", auditevent.OutcomeMinorFailure))
		}
		return s.audit.Record(ctx, inviteEvent(auditevent.CodeDelete, auditevent.ActionDelete, admin, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("auth: withdraw invite: %w", err)
	}
	return refused
}
