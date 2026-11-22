package notification

import (
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// whereToCome sets on l what a message about a visit of the mode at the clinic says of the place (docs/plans/clinic-cms.md,
// ADR 0016): for an in-person visit that it is at the clinic, with the clinic's address when it has one; nothing for a
// video visit. The address is the clinic's, not personal data.
func whereToCome(l letter, mode schedule.VisitMode, clinic organization.Organization) letter {
	if mode == schedule.ModeInPerson {
		l.InPerson, l.Address = true, clinic.Address()
	}
	return l
}
