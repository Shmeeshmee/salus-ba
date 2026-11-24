package patient

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// ObjectToAnalytics marks the patient id as objecting to the anonymised analytics (docs/clinic-cms/WAVE5.md S9): from
// the request on, the aggregate builds leave them out (analytics_left_out). Called by the data request's filing, in
// its transaction and scope (the patient's own, or the registering clinic's for one it files at the desk); a mark
// already there stays. Audited as an update of the Patient at the patient's request.
func (s *Service) ObjectToAnalytics(ctx context.Context, id string) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.ObjectToAnalytics(ctx, id, s.clock.Now()); err != nil {
			return err
		}
		return s.audit.Record(ctx, auditevent.AuditEvent{
			Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate},
			Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodePATRQT},
			Action:          auditevent.ActionUpdate,
			Outcome:         auditevent.OutcomeSuccess,
			EntityReference: "Patient/" + id,
		})
	})
	if err != nil {
		return fmt.Errorf("patient: object to analytics: %w", err)
	}
	return nil
}
