package invoice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Retry reports a receipt whose fiscal report failed once more (POST /clinic/receipts/{id}/retry): it enqueues the
// same invoice_fiscalize job, which fiscalizes it when the authority accepts it. By the clinic's admin, for a failed
// receipt only: ErrNotFailed for a pending one (its job is still running) and a fiscalized one (reported for good);
// ErrForbidden to anyone else, ErrNotFound for a receipt of another clinic, both audited. The receipt is returned as
// it is, still failed until the job runs.
func (s *Service) Retry(ctx context.Context, c Caller, id string) (Invoice, error) {
	if c.OrganizationID == "" || !c.Admin {
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.ByID(ctx, id); err != nil {
			return err
		}
		if out.FiscalStatus != FiscalFailed {
			return ErrNotFailed
		}
		if err := s.p.Jobs.EnqueueFiscalize(ctx, out.ID, out.OrganizationID); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNotFailed):
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeUpdate, err)
	case err != nil:
		return Invoice{}, fmt.Errorf("invoice: retry: %w", err)
	}
	return out, nil
}
