package schedule

import (
	"context"
	"slices"

	"github.com/Shmeeshmee/salus/internal/location"
)

// Sites lists a clinic's locations in its scope (location.Service).
type Sites interface {
	OfClinic(ctx context.Context, org string) ([]location.Location, error)
}

// placed returns rules with every in-person rule at a location of the clinic org (docs/clinic-cms/WAVE6.md U2): the
// active location it names, or the clinic's only active one when it names none. A rule at no such location is
// returned as failed (ErrInvalidLocation); err is any other error. Video rules are left as they are.
func (s *Service) placed(ctx context.Context, org string, rules []Rule) (out []Rule, failed, err error) {
	if !slices.ContainsFunc(rules, func(r Rule) bool { return r.Mode == ModeInPerson }) {
		return rules, nil, nil
	}
	all, err := s.Sites.OfClinic(ctx, org)
	if err != nil {
		return nil, nil, err
	}
	active := slices.DeleteFunc(all, func(l location.Location) bool { return !l.Active || l.OrganizationID != org })
	out = slices.Clone(rules)
	for i, r := range out {
		switch {
		case r.Mode != ModeInPerson:
		case r.LocationID == "" && len(active) == 1:
			out[i].LocationID = active[0].ID
		case r.LocationID == "" || !slices.ContainsFunc(active, func(l location.Location) bool { return l.ID == r.LocationID }):
			return nil, ErrInvalidLocation, nil
		}
	}
	return out, nil, nil
}
