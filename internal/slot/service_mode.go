package slot

import "github.com/Shmeeshmee/salus/internal/schedule"

// ruleMode is the visit mode of the slots of r (docs/plans/clinic-cms.md, ADR 0016): the rule's, and video for a rule
// that names none, as every rule made before the clinic system.
func ruleMode(r schedule.Rule) schedule.VisitMode {
	if r.Mode == "" {
		return schedule.ModeVideo
	}
	return r.Mode
}
