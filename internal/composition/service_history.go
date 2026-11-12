package composition

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// History returns the signed documents of visits, each with its visit, in the order of visits and then of signing:
// for the practitioner's view of the patient's history (SAL-40), with visits exactly as encounter.Service.History
// gave them to the caller (access.HistoryVisit applied). Never a draft. Read in the caller's clinic scope (the
// shared visits' documents through document_draft_shared) and unaudited: the history audits the ones it shows
// (Disclosed). ErrForbidden for anyone but a practitioner.
func (s *Service) History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]Document, error) {
	var out []Document
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if c.OrganizationID == "" || c.Admin {
			return errRefused{ErrForbidden}
		}
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		for _, v := range visits {
			drafts, err := s.Repo.Signed(ctx, v.ID)
			if err != nil {
				return err
			}
			for _, d := range drafts {
				out = append(out, Document{Draft: d, Encounter: v.Encounter})
			}
		}
		return nil
	})
	if r := refusal(err); r != nil {
		return nil, r
	}
	return out, wrapErr("history", err, nil)
}

// Disclosed audits the signed documents ids (Composition ids) the history shows the caller, each as a search of
// it, in the transaction in ctx (or one of its own).
func (s *Service) Disclosed(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.record(ctx, entry{c: c, interaction: auditevent.CodeSearchType, reference: "Composition/" + id}); err != nil {
				return err
			}
		}
		return nil
	})
	return wrapErr("disclosed", err, nil)
}
