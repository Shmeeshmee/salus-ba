package appointment

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// SearchPage is Search, a page at a time, for the FHIR search (api/fhir/CapabilityStatement.json): the first count
// matches after f's position (AfterStart, AfterID) in the search's order, by start then id. A staff search without a
// closed date range (a ge and an lt) returns the clinic's or the role's whole history, so it is paged, by DefaultCount
// unless count asks for another number; so is any search given a count or a position. The rest (the calendar's week,
// a patient's own) returns every match, as Search does. The appointments the caller may not see are read past and
// left out as Search leaves them out; the page's are audited as Search audits them, and a page without a match with
// one event without an entity.
func (s *Service) SearchPage(ctx context.Context, c Caller, f Filter, count int) (Page, error) {
	if count == 0 && (f.AfterID != "" || (!c.patient() && (f.From.IsZero() || f.To.IsZero()))) {
		count = DefaultCount
	}
	return s.find(ctx, c, f, count, false)
}

// matches reads the search's appointments and decides the viewer's view of each: every one, or with count the first
// count the viewer may see after f's position, read past those they may not (keyset.Fill), and whether more may
// follow.
func (s *Service) matches(ctx context.Context, c Caller, viewer access.Viewer, f Filter, count int, record bool) ([]shownView, bool, error) {
	keep := func(batch []Appointment) ([]shownView, error) { return s.views(ctx, c, viewer, f, batch, record) }
	if count == 0 {
		all, err := s.Repo.Search(ctx, f)
		if err != nil {
			return nil, false, err
		}
		views, err := keep(all)
		return views, false, err
	}
	return keyset.Fill(count, func(after *Appointment, limit int) ([]Appointment, error) {
		g := f
		if after != nil {
			g.AfterStart, g.AfterID = after.Start, after.ID
		}
		g.Limit = limit
		return s.Repo.Search(ctx, g)
	}, keep)
}
