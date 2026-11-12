package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// History returns the completed responses of the visits' appointments, with their scanned files only, in the order
// of visits: for the practitioner's view of the patient's history (SAL-40), with visits exactly as
// encounter.Service.History gave them to the caller (access.HistoryVisit applied). Never one in progress. Read in the
// caller's clinic scope (a shared visit's through questionnaire_response_shared) and unaudited: the history audits the
// ones it shows (Disclosed). ErrNotFound for anyone but a practitioner.
func (s *Service) History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]Response, error) {
	if c.OrganizationID == "" || c.Admin {
		return nil, ErrNotFound
	}
	var out []Response
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		for _, v := range visits {
			rs, err := s.Repo.Search(ctx, Filter{AppointmentID: v.AppointmentID, PatientID: v.PatientID, Status: StatusCompleted})
			if err != nil {
				return err
			}
			for _, r := range rs {
				r.Attachments = scanned(r.Attachments)
				out = append(out, r)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("questionnaireresponse: history: %w", err)
	}
	return out, nil
}

// Disclosed audits the responses ids the history shows the caller (with their files), each as a search of it, in
// the transaction in ctx (or one of its own).
func (s *Service) Disclosed(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.record(ctx, c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: disclosed: %w", err)
	}
	return nil
}
