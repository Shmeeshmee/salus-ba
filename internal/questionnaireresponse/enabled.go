package questionnaireresponse

import "github.com/Shmeeshmee/salus/internal/questionnaire"

// enabled tells, by linkId, which items of a version are shown with the answers given (SPEC.md §2 "Item
// model"): an item is enabled when its section is and its condition holds on the answer to an enabled item;
// the answer to a hidden item counts as none (R4). A condition names an earlier item, so one pass in document
// order decides every item. The mobile app evaluates conditions the same way.
func enabled(items []questionnaire.Item, answers map[string][]Value) map[string]bool {
	on := map[string]bool{}
	var walk func(items []questionnaire.Item, shown bool)
	walk = func(items []questionnaire.Item, shown bool) {
		for _, it := range items {
			e := shown && holds(it.EnableWhen, on, answers)
			on[it.LinkID] = e
			walk(it.Items, e)
		}
	}
	walk(items, true)
	return on
}

// holds tells whether the condition c (none always holds) is met: its question is enabled and answered with
// its boolean, or, for a choice, any of its answers is the option.
func holds(c *questionnaire.Condition, on map[string]bool, answers map[string][]Value) bool {
	if c == nil {
		return true
	}
	if !on[c.Question] {
		return false
	}
	for _, v := range answers[c.Question] {
		switch {
		case c.Boolean != nil && v.Boolean != nil && *v.Boolean == *c.Boolean:
			return true
		case c.Boolean == nil && c.String != "" && v.String == c.String:
			return true
		}
	}
	return false
}
