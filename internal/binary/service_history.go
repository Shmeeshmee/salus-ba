package binary

import (
	"context"
	"fmt"
)

// Clean returns, of the files ids the holder lists in the calling patient's timeline (SAL-39), those that are
// their own and clean, read in their own scope, joining the transaction in ctx, unaudited: history.Timeline audits
// the files of the page it returns. Nothing asked is nothing read; a patient without a profile has none. The
// timeline is the patient's: staff get ErrForbidden.
func (s *Service) Clean(ctx context.Context, c Caller, ids []string) ([]string, error) {
	switch {
	case c.OrganizationID != "":
		return nil, ErrForbidden
	case len(ids) == 0 || c.PatientID == "":
		return nil, nil
	}
	var out []string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		files, err := s.Repo.Clean(ctx, ids)
		for _, b := range files {
			if b.PatientID == c.PatientID && b.Status == StatusClean { // not on row-level security alone
				out = append(out, b.ID)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("binary: clean: %w", err)
	}
	return out, nil
}
