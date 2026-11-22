package invoice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// AgentFiscal names the fiscal job as the agent of its writes' audit events.
const AgentFiscal = "invoice_fiscalize"

// Fiscalize reports the receipt id of the clinic org through the fiscal port and stores the authority's answer: the
// invoice_fiscalize job. The receipt stays locked while it is reported, so two runs never report it at once, and a
// fiscalized receipt is not reported again. What is reported carries the seller's tax id, the number, amounts and item
// names, never the patient. A receipt the authority rejects for good is marked failed and the job ends; any other
// failure is returned, the receipt stays pending and River retries the job (GiveUp when it stops). A reversal is
// reported with the fiscal id of the receipt it reverses: ErrOriginalPending until that receipt is fiscalized.
//
// A commit that fails after the report leaves the receipt pending; the retried job reports the same reference (the
// receipt's id), which the authority recognises.
func (s *Service) Fiscalize(ctx context.Context, org, id string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		inv, err := s.p.Repo.Locked(ctx, id)
		if err != nil || inv.FiscalStatus == FiscalFiscalized {
			return err
		}
		f, err := s.report(ctx, inv)
		switch {
		case errors.Is(err, fiscal.ErrRejected), errors.Is(err, fiscal.ErrInvalidReceipt):
			if err := s.p.Repo.SetFailed(ctx, id); err != nil {
				return err
			}
			return s.p.Audit.Record(ctx, jobEvent(org, id, auditevent.OutcomeMinorFailure))
		case err != nil:
			return err
		}
		if err := s.p.Repo.SetFiscalized(ctx, id, f); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, jobEvent(org, id, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("invoice: fiscalize: %w", err)
	}
	return nil
}

// report sends inv to the authority: a receipt as it is, a reversal with the fiscal id of the receipt it reverses.
func (s *Service) report(ctx context.Context, inv Invoice) (fiscal.Fiscalized, error) {
	r := fiscal.Receipt{
		IssuedAt: inv.IssuedAt, Reference: inv.ID, SellerTaxID: inv.SellerTaxID, Currency: inv.Currency,
		PaymentMethod: fiscal.PaymentMethod(inv.PaymentMethod), Number: inv.Number, TotalMinor: inv.TotalMinor,
	}
	for _, l := range inv.Lines {
		r.Lines = append(r.Lines, fiscal.Line{
			VATRateBP: l.VATRateBP, Name: l.Name, UnitPriceMinor: l.UnitPriceMinor, DiscountMinor: l.DiscountMinor,
			TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		})
	}
	if inv.ReversesID == "" {
		return s.p.Reporter.Report(ctx, r)
	}
	orig, err := s.p.Repo.ByID(ctx, inv.ReversesID)
	switch {
	case err != nil:
		return fiscal.Fiscalized{}, err
	case orig.FiscalID == "":
		return fiscal.Fiscalized{}, ErrOriginalPending
	}
	r.Reverses = orig.FiscalID
	return s.p.Reporter.Reverse(ctx, r)
}

// GiveUp marks the pending receipt id of the clinic org failed, when its job has stopped retrying: the clinic sees
// it, and a later report (Fiscalize) may still fiscalize it. A fiscalized or failed receipt is left as it is.
func (s *Service) GiveUp(ctx context.Context, org, id string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		inv, err := s.p.Repo.Locked(ctx, id)
		if err != nil || inv.FiscalStatus != FiscalPending {
			return err
		}
		if err := s.p.Repo.SetFailed(ctx, id); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, jobEvent(org, id, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("invoice: give up: %w", err)
	}
	return nil
}

// jobEvent is the AuditEvent of the fiscal job's update of the receipt id.
func jobEvent(org, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(Caller{OrganizationID: org}, auditevent.CodeUpdate, id, outcome)
	e.AgentName = AgentFiscal
	return e
}
