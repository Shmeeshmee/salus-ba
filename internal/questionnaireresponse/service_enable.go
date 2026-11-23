package questionnaireresponse

import (
	"math/big"
	"strings"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// enabled tells, by linkId, which items of a version are shown with the answers given (FHIR R4 enableWhen,
// docs/clinic-cms Q1): an item is enabled when its parent is and its conditions hold, all of them or any one
// as enableBehavior says. A condition reads its question's answers only while that question is enabled: a
// hidden one counts as unanswered. A condition names an earlier question, so one pass in document order
// decides every item. Web (enableWhen.ts) and mobile (enable_when.dart) evaluate the same, on the shared
// vectors of test/testdata/enablewhen.
func enabled(items []questionnaire.Item, answers map[string][]Value) map[string]bool {
	on := map[string]bool{}
	var walk func(items []questionnaire.Item, shown bool)
	walk = func(items []questionnaire.Item, shown bool) {
		for _, it := range items {
			e := shown && holds(it, on, answers)
			on[it.LinkID] = e
			walk(it.Items, e)
		}
	}
	walk(items, true)
	return on
}

// holds tells whether the item's conditions are met (none always are).
func holds(it questionnaire.Item, on map[string]bool, answers map[string][]Value) bool {
	anyOne := it.EnableBehavior == questionnaire.BehaviorAny
	for _, c := range it.EnableWhen {
		if meets(c, on, answers) == anyOne {
			return anyOne
		}
	}
	return !anyOne || len(it.EnableWhen) == 0
}

// meets tells whether one condition holds: exists on whether the question has an answer; = when any answer
// equals the condition's, != when none does; the order operators on the answer. Every operator but exists is
// false on an unanswered question.
func meets(c questionnaire.Condition, on map[string]bool, answers map[string][]Value) bool {
	var given []Value
	if on[c.Question] {
		given = answers[c.Question]
	}
	if c.Operator == questionnaire.OpExists {
		return c.Boolean != nil && (len(given) > 0) == *c.Boolean
	}
	if len(given) == 0 {
		return false
	}
	if c.Operator == questionnaire.OpNotEqual {
		for _, v := range given {
			if d, ok := compare(c, v); !ok || d == 0 {
				return false
			}
		}
		return true
	}
	for _, v := range given {
		if d, ok := compare(c, v); ok && order(c.Operator, d) {
			return true
		}
	}
	return false
}

// order tells whether d, the answer compared with the condition's (-1, 0, 1), satisfies op.
func order(op questionnaire.Operator, d int) bool {
	switch op {
	case questionnaire.OpEqual:
		return d == 0
	case questionnaire.OpGreater:
		return d > 0
	case questionnaire.OpLess:
		return d < 0
	case questionnaire.OpGreaterEqual:
		return d >= 0
	case questionnaire.OpLessEqual:
		return d <= 0
	}
	return false
}

// compare gives the answer v compared with the condition's answer, and false when they are of different types.
// A yes/no or an option is only equal (0) or not (1).
func compare(c questionnaire.Condition, v Value) (int, bool) {
	switch {
	case c.Boolean != nil && v.Boolean != nil:
		return unequal(*v.Boolean != *c.Boolean), true
	case c.Integer != nil && v.Integer != nil:
		return compareInts(*v.Integer, *c.Integer), true
	case c.Decimal != "" && v.Decimal != "":
		a, okA := new(big.Rat).SetString(string(v.Decimal))
		b, okB := new(big.Rat).SetString(string(c.Decimal))
		if !okA || !okB {
			return 0, false
		}
		return a.Cmp(b), true
	case c.Date != "" && v.Date != "":
		return strings.Compare(v.Date, c.Date), true // YYYY-MM-DD orders as text
	case c.String != "" && v.String != "":
		return unequal(v.String != c.String), true
	}
	return 0, false
}

func unequal(b bool) int {
	if b {
		return 1
	}
	return 0
}

func compareInts(a, b int32) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
