package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// NoticeAccount returns the account of the patient id, for a message the job agent sends them about the clinic
// org (the chat's notifications, SAL-29): read in the clinic's scope, which shows the patient while their care
// consent with it is active, and audited as a read of the patient with the job as agent, also when it is not found
// (ErrNotFound). Nothing else of the profile is returned.
func (s *Service) NoticeAccount(ctx context.Context, agent, org, id string) (string, error) {
	var p Patient
	var missing bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, org); err != nil {
			return err
		}
		var err error
		p, err = s.repo.ByID(ctx, id)
		if missing = errors.Is(err, ErrNotFound); err != nil && !missing {
			return err
		}
		e := auditevent.AuditEvent{
			Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
			Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
			Action:         auditevent.ActionRead,
			Outcome:        auditevent.OutcomeSuccess,
			AgentName:      agent,
			OrganizationID: org,
		}
		if missing {
			e.Outcome = auditevent.OutcomeMinorFailure
		} else {
			e.EntityReference = "Patient/" + p.ID
		}
		return s.audit.Record(ctx, e)
	})
	switch {
	case err != nil:
		return "", fmt.Errorf("patient: notice account: %w", err)
	case missing:
		return "", ErrNotFound
	}
	return p.AccountID, nil
}
