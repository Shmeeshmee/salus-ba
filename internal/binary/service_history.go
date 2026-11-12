package binary

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Clean returns, of the files ids the holder lists in the calling patient's timeline (SAL-39), those that are
// their own and clean, read in their own scope, each audited as a search of their own request (in one insert). Nothing asked is
// nothing read or audited; a patient without a profile has none. The timeline is the patient's: staff get
// ErrForbidden, audited.
func (s *Service) Clean(ctx context.Context, c Caller, ids []string) ([]string, error) {
	staff := c.OrganizationID != ""
	if !staff && (len(ids) == 0 || c.PatientID == "") {
		return nil, nil
	}
	var out []string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if staff {
			return s.record(ctx, entry{c: c, org: c.OrganizationID, interaction: auditevent.CodeSearchType, outcome: auditevent.OutcomeMinorFailure})
		}
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		files, err := s.Repo.Clean(ctx, ids)
		if err != nil {
			return err
		}
		var events []auditevent.AuditEvent
		for _, b := range files {
			if b.PatientID != c.PatientID || b.Status != StatusClean { // not on row-level security alone
				continue
			}
			events = append(events, entry{c: c, org: b.OrganizationID, interaction: auditevent.CodeSearchType, binaryID: b.ID}.events()...)
			out = append(out, b.ID)
		}
		return s.Audit.RecordAll(ctx, events) // one insert for the page
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("binary: clean: %w", err)
	case staff:
		return nil, ErrForbidden
	}
	return out, nil
}
