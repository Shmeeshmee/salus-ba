package invoice

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// VisitReceipts gives the receipts of the clinic org's visits appointmentIDs, by visit, each with its number and
// whether it was reversed (reversals left out), read in the clinic's scope in the transaction in ctx: the desk's day
// (appointment.Service.DeskDay), which authorizes its caller and audits each receipt shown. Not audited here.
func (s *Service) VisitReceipts(ctx context.Context, org string, appointmentIDs []string) (map[string][]appointment.VisitReceipt, error) {
	out := map[string][]appointment.VisitReceipt{}
	if len(appointmentIDs) == 0 {
		return out, nil
	}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		all, err := s.p.Repo.OfVisits(ctx, org, appointmentIDs)
		for _, inv := range all {
			if inv.OrganizationID != org { // not on row-level security alone
				continue
			}
			out[inv.AppointmentID] = append(out[inv.AppointmentID], appointment.VisitReceipt{
				ID: inv.ID, Number: inv.Number, Reversed: inv.ReversedByID != "",
			})
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: visit receipts: %w", err)
	}
	return out, nil
}

// ForVisit returns the receipts of the clinic's visit appointmentID (reversals left out, each with ReversedByID), by
// number, to its admin (GET /clinic/receipts?appointmentId=, the visit page); each is audited as a search's match,
// none found once. ErrForbidden, audited, to anyone else, as the list of all receipts.
func (s *Service) ForVisit(ctx context.Context, c Caller, appointmentID string) ([]Invoice, error) {
	if c.OrganizationID == "" || !c.Admin {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		all, err := s.p.Repo.OfVisits(ctx, c.OrganizationID, []string{appointmentID})
		if err != nil {
			return err
		}
		for _, inv := range all {
			if inv.OrganizationID == c.OrganizationID {
				out = append(out, inv)
			}
		}
		if len(out) == 0 {
			return s.p.Audit.Record(ctx, event(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
		}
		for _, inv := range out {
			if err := s.p.Audit.Record(ctx, event(c, auditevent.CodeSearchType, inv.ID, auditevent.OutcomeSuccess)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: for visit: %w", err)
	}
	return out, nil
}
