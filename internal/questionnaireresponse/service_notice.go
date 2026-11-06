package questionnaireresponse

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// Outstanding tells the e-mail job agent (its kind) whether the appointment with the id appointmentID, of the
// clinic org, has a questionnaire for the patient to fill in (SPEC.md §2 "Notifications", D6): a response in progress, or the
// active default of the appointment's role not yet requested (the booking confirmation can run before the
// default-request job). In one transaction in the clinic's scope with the appointment's notice; the responses
// read are audited as a search by the job (one event without an entity when there are none), the default's
// read by the questionnaire's service. An appointment the scope does not show, or of another clinic, has
// nothing outstanding. Only the answer leaves: never a title.
func (s *Service) Outstanding(ctx context.Context, agent, org, appointmentID string) (bool, error) {
	var out bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		if err := s.Repo.SetNotice(ctx, appointmentID); err != nil {
			return err
		}
		rs, err := s.Repo.Search(ctx, Filter{AppointmentID: appointmentID})
		if err != nil {
			return err
		}
		rs = slices.DeleteFunc(rs, func(r Response) bool { return r.OrganizationID != org }) // not on row-level security alone
		if err := s.recordNotice(ctx, agent, org, rs); err != nil {
			return err
		}
		if slices.ContainsFunc(rs, func(r Response) bool { return r.Status == StatusInProgress }) {
			out = true
			return nil
		}
		out, err = s.defaultOutstanding(ctx, agent, org, appointmentID, rs)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("questionnaireresponse: outstanding: %w", err)
	}
	return out, nil
}

// defaultOutstanding tells whether the appointment's role has an active default of a series none of rs is of.
func (s *Service) defaultOutstanding(ctx context.Context, agent, org, appointmentID string, rs []Response) (bool, error) {
	a, err := s.Appointments.Lookup(ctx, appointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	case a.OrganizationID != org:
		return false, nil
	}
	q, err := s.Questionnaires.ActiveDefault(ctx, agent, org, a.PractitionerRoleID)
	switch {
	case errors.Is(err, questionnaire.ErrNotFound), errors.Is(err, questionnaire.ErrNotActive):
		return false, nil
	case err != nil:
		return false, err
	}
	return !slices.ContainsFunc(rs, func(r Response) bool { return r.SeriesID == q.SeriesID }), nil
}

// recordNotice audits the job's read of rs as a search: one event per response, or one without an entity.
func (s *Service) recordNotice(ctx context.Context, agent, org string, rs []Response) error {
	if len(rs) == 0 {
		return s.recordJob(ctx, agent, org, auditevent.CodeSearchType, auditevent.ActionExecute, "")
	}
	for _, r := range rs {
		if err := s.recordJob(ctx, agent, org, auditevent.CodeSearchType, auditevent.ActionExecute, r.ID); err != nil {
			return err
		}
	}
	return nil
}
