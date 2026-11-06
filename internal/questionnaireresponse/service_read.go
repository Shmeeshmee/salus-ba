package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Read returns the response id as the caller may see it (access.QuestionnaireResponse): the patient their
// own; the appointment's practitioner the answers once submitted (also after a revocation, for a past
// appointment: D9) and the name before; the clinic's admin the name. A Name view carries no answers. Read in
// the caller's own scope and audited; ErrNotFound to anyone else, audited without the entity.
func (s *Service) Read(ctx context.Context, c appointment.Caller, id string) (Seen, error) {
	var out Seen
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, _, level, err := s.view(ctx, c, id, s.Repo.ByID)
		switch {
		case err != nil:
			return err
		case level == access.None:
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		out = seen(r, level)
		return s.record(ctx, c, auditevent.CodeRead, r.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Seen{}, fmt.Errorf("questionnaireresponse: read: %w", err)
	case out.Level == access.None:
		return Seen{}, ErrNotFound
	}
	return out, nil
}

// Search returns the responses of the filter as the caller may see them, scoped like Read: a patient only
// their own (whatever patient the filter names), staff their clinic's at each one's level, those at None left
// out (a practitioner sees only their own appointments'). Each response returned is audited; a search that
// returns none writes one event without an entity. The attendance of each appointment is read once.
func (s *Service) Search(ctx context.Context, c appointment.Caller, f Filter) ([]Seen, error) {
	var out []Seen
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.search(ctx, c, f); err != nil {
			return err
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("questionnaireresponse: search: %w", err)
	}
	return out, nil
}

// search runs the caller's search in the transaction in ctx and audits each result.
func (s *Service) search(ctx context.Context, c appointment.Caller, f Filter) ([]Seen, error) {
	if c.OrganizationID == "" {
		f.PatientID = c.PatientID
	}
	if ok, err := s.scope(ctx, c); err != nil || !ok {
		return nil, err
	}
	all, err := s.Repo.Search(ctx, f)
	if err != nil {
		return nil, err
	}
	type known struct {
		att appointment.Attendance
		ok  bool
	}
	attendances := map[string]known{}
	var out []Seen
	for _, r := range all {
		k, cached := attendances[r.AppointmentID]
		if !cached {
			if k.att, k.ok, err = s.attendance(ctx, c, r.AppointmentID); err != nil {
				return nil, err
			}
			attendances[r.AppointmentID] = k
		}
		if !k.ok {
			continue
		}
		level := level(c, k.att, r)
		if level == access.None {
			continue
		}
		if err := s.record(ctx, c, auditevent.CodeSearchType, r.ID, auditevent.OutcomeSuccess); err != nil {
			return nil, err
		}
		out = append(out, seen(r, level))
	}
	return out, nil
}

// seen is r at level: a Name view without its answers.
func seen(r Response, level access.Level) Seen {
	if level != access.Full {
		r.Answers = nil
	}
	return Seen{Response: r, Level: level}
}
