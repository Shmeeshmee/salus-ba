package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// maxUsageMonths bounds a usage request: ten years of months.
const maxUsageMonths = 120

// SMSUsage returns the admin's clinic's SMS usage from the month from to the month to (YYYY-MM, UTC), one entry per
// month with usage, oldest first, audited as a read of the clinic. ErrForbidden to anyone but its admin; ErrInvalid
// for a month that is not YYYY-MM, to before from, or more than ten years.
func (s *Service) SMSUsage(ctx context.Context, c Caller, from, to string) ([]UsageMonth, error) {
	if c.OrganizationID == "" || !c.Admin {
		return nil, ErrForbidden
	}
	f, errF := time.Parse("2006-01", from)
	t, errT := time.Parse("2006-01", to)
	if errF != nil || errT != nil || t.Before(f) || t.After(f.AddDate(0, maxUsageMonths, 0)) {
		return nil, ErrInvalid
	}
	var out []UsageMonth
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Devices.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Deliveries.SMSUsage(ctx, c.OrganizationID, f, t); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, auditevent.AuditEvent{
			Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
			Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
			Action:          auditevent.ActionRead,
			Outcome:         auditevent.OutcomeSuccess,
			AgentAccountID:  c.AccountID,
			OrganizationID:  c.OrganizationID,
			EntityReference: "Organization/" + c.OrganizationID,
			EntityName:      "sms_usage",
		})
	})
	if err != nil {
		return nil, fmt.Errorf("notification: sms usage: %w", err)
	}
	if out == nil {
		out = []UsageMonth{}
	}
	return out, nil
}

// DeliveryPurger applies the retention rule of notification_delivery (internal/retention): 90 days after it was sent,
// or created when it never was.
type DeliveryPurger struct {
	repo DeliveryRepository
}

// NewDeliveryPurger returns the purger of notification_delivery on repo, built on the retention role's pool.
func NewDeliveryPurger(repo DeliveryRepository) *DeliveryPurger { return &DeliveryPurger{repo: repo} }

// Purge deletes the deliveries sent (or created) before before and returns how many.
func (p *DeliveryPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	n, err := p.repo.DeleteDeliveriesBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("notification: purge deliveries: %w", err)
	}
	return n, nil
}
