package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// Read returns the response id as the caller may see it (access.QuestionnaireResponse): the patient their
// own; the appointment's practitioner the answers once submitted (also after a revocation, for a past
// appointment: D9) and the name before; the clinic's admin the name; a practitioner the history shows another
// clinic's shared visit to its completed responses (ADR 0014). A Name view carries no answers. Read in
// the caller's own scope and audited; ErrNotFound to anyone else, audited without the entity.
func (s *Service) Read(ctx context.Context, c appointment.Caller, id string) (Seen, error) {
	var out Seen
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, _, level, err := s.view(ctx, c, id, s.Repo.ByID)
		if err == nil && level == access.None {
			r, level, err = s.shared(ctx, c, id)
		}
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
	f.AfterID = ""
	p, err := s.find(ctx, c, f, 0)
	return p.Seen, err
}

// SearchPage is Search, a page at a time, for the FHIR search (api/fhir/CapabilityStatement.json): the first count
// responses after the response f.AfterID in the search's order. A staff search not narrowed to one appointment (a
// few responses at most) returns the clinic's whole history, so it is paged, by appointment.DefaultCount unless count
// asks for another number; so is any search given a count or a position. A patient's own returns every match, as
// Search does. The responses the caller may not see are read past and left out as Search leaves them out; the page's
// are audited as Search audits them. ErrPosition for a position the scope does not show.
func (s *Service) SearchPage(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error) {
	if count == 0 && (f.AfterID != "" || (c.OrganizationID != "" && f.AppointmentID == "")) {
		count = appointment.DefaultCount
	}
	return s.find(ctx, c, f, count)
}

// find is Search, a page of count when count is set: it runs the search in a transaction and audits an empty result
// with one event.
func (s *Service) find(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error) {
	var out Page
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.search(ctx, c, f, count); err != nil {
			return err
		}
		if len(out.Seen) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	if err != nil {
		return Page{}, fmt.Errorf("questionnaireresponse: search: %w", err)
	}
	return out, nil
}

// search runs the caller's search in the transaction in ctx and audits each result.
func (s *Service) search(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error) {
	if c.OrganizationID == "" {
		f.PatientID = c.PatientID
	}
	if ok, err := s.scope(ctx, c); err != nil || !ok {
		return Page{}, err
	}
	type known struct {
		att appointment.Attendance
		ok  bool
	}
	attendances := map[string]known{}
	keep := func(batch []Response) ([]Seen, error) {
		var kept []Seen
		for _, r := range batch {
			k, cached := attendances[r.AppointmentID]
			if !cached {
				var err error
				if k.att, k.ok, err = s.attendance(ctx, c, r.AppointmentID); err != nil {
					return nil, err
				}
				attendances[r.AppointmentID] = k
			}
			if !k.ok {
				continue
			}
			if level := level(c, k.att, r); level != access.None {
				kept = append(kept, seen(r, level))
			}
		}
		return kept, nil
	}
	var out Page
	var err error
	f.Limit = 0
	if count == 0 {
		var all []Response
		if all, err = s.Repo.Search(ctx, f); err == nil {
			out.Seen, err = keep(all)
		}
	} else {
		out.Seen, out.More, err = keyset.Fill(count, func(after *Response, limit int) ([]Response, error) {
			g := f
			if after != nil {
				g.AfterID = after.ID
			}
			g.Limit = limit
			return s.Repo.Search(ctx, g)
		}, keep)
	}
	if err != nil {
		return Page{}, err
	}
	for _, x := range out.Seen {
		if err := s.record(ctx, c, auditevent.CodeSearchType, x.Response.ID, auditevent.OutcomeSuccess); err != nil {
			return Page{}, err
		}
	}
	return out, nil
}

// seen is r at level: a Name view without its answers and files.
func seen(r Response, level access.Level) Seen {
	if level != access.Full {
		r.Answers, r.Attachments = nil, nil
	}
	return Seen{Response: r, Level: level}
}
