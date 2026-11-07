package binary

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Discard is the binary_discard job, enqueued by the holder in its transaction when a withdrawal clears a
// response or a submit removes the answers of hidden items (docs/attachments/SPEC.md §2 "Discard"). In the job
// scope of the response, every file the holder no longer lists is deleted, row first, each audited as a delete
// with the job as agent; the objects after the commit (best effort: the purger removes what is left). A response
// without files, the cancellation's included (its files went with it), asks the holder nothing. Repeated, it
// changes nothing. It returns how many files it deleted.
func (s *Service) Discard(ctx context.Context, job DiscardArgs) (int, error) {
	var gone []string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		gone = nil
		if err := s.Repo.SetJob(ctx, job.ResponseID, job.OrganizationID); err != nil {
			return err
		}
		files, err := s.Repo.OfResponse(ctx, job.ResponseID)
		if err != nil || len(files) == 0 {
			return err
		}
		attached, err := s.Holder.Attached(ctx, job.ResponseID)
		if err != nil {
			return err
		}
		for _, b := range files {
			if slices.Contains(attached, b.ID) {
				continue
			}
			deleted, err := s.Repo.Delete(ctx, b.ID)
			switch {
			case err != nil:
				return err
			case !deleted:
				return errors.New("the row was not deleted")
			}
			e := entry{agent: agentDiscard, org: b.OrganizationID, interaction: auditevent.CodeDelete, binaryID: b.ID, responseID: b.ResponseID}
			if err := s.record(ctx, e); err != nil {
				return err
			}
			gone = append(gone, b.ID)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("binary: discard: %w", err)
	}
	for _, id := range gone {
		_ = s.Store.Delete(context.WithoutCancel(ctx), id)
	}
	return len(gone), nil
}
