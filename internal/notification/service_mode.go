package notification

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// whereToCome sets on l what a message about a visit of the mode at the clinic says of the place (docs/plans/clinic-cms.md,
// ADR 0016): for an in-person visit that it is at the clinic, with address (address); nothing for a video visit.
func whereToCome(l letter, mode schedule.VisitMode, address string) letter {
	if mode == schedule.ModeInPerson {
		l.InPerson, l.Address = true, address
	}
	return l
}

// address is where the patient comes for an in-person visit at locationID of clinic: its location's address
// (docs/clinic-cms/WAVE6.md U2), or the clinic's registered one when the location has none or the visit names none; ""
// for a video visit. The address is the clinic's, not personal data.
func (s *Service) address(ctx context.Context, clinic organization.Organization, mode schedule.VisitMode, locationID string) (string, error) {
	if mode != schedule.ModeInPerson {
		return "", nil
	}
	if locationID != "" && s.p.Sites != nil {
		site, err := s.p.Sites.Address(ctx, clinic.ID, locationID)
		if err != nil || site != "" {
			return site, err
		}
	}
	return clinic.Address(), nil
}
