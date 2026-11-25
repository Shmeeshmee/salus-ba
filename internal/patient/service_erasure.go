package patient

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// ErasureAgent names the erasure job as the agent of its writes (docs/clinic-cms/WAVE5.md S6).
const ErasureAgent = "data_request_erasure"

// RestrictForErasure restricts the record the clinic org keeps of the profile patientID, once its account's erasure
// date has come (S6, S7): the clinic keeps it under the medical-record retention, hidden from its everyday screens, and
// never lifts this restriction. Runs in the erasure's transaction (its scope set, datarequest.Erasure), in the clinic's
// scope, audited as the job's update of the profile at that clinic.
func (s *Service) RestrictForErasure(ctx context.Context, patientID, org string) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetClinic(ctx, org); err != nil {
			return err
		}
		if err := s.repo.RestrictForErasure(ctx, patientID, org, s.clock.Now()); err != nil {
			return err
		}
		return s.audit.Record(ctx, auditevent.AuditEvent{
			Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate},
			Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
			Action:          auditevent.ActionUpdate,
			Outcome:         auditevent.OutcomeSuccess,
			AgentName:       ErasureAgent,
			EntityReference: "Patient/" + patientID,
			OrganizationID:  org,
		})
	})
	if err != nil {
		return fmt.Errorf("patient: restrict for erasure: %w", err)
	}
	return nil
}
