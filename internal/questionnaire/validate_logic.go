package questionnaire

import (
	"regexp"
	"time"
)

// decimalPattern is a decimal written plainly: FHIR's decimal without an exponent, as answers are.
var decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// logic tells whether the item's conditions are sound (docs/clinic-cms, Q1): each names an earlier question,
// with an operator that fits its type and exactly the one answer[x] the operator and the type take; two or more
// are combined by enableBehavior all or any.
func (v *validator) logic(it Item) bool {
	switch {
	case it.EnableBehavior != "" && it.EnableBehavior != BehaviorAll && it.EnableBehavior != BehaviorAny:
		return false
	case it.EnableBehavior != "" && len(it.EnableWhen) == 0, it.EnableBehavior == "" && len(it.EnableWhen) > 1:
		return false
	}
	for _, c := range it.EnableWhen {
		if !v.condition(c) {
			return false
		}
	}
	return true
}

// condition tells whether c fits the earlier question it names.
func (v *validator) condition(c Condition) bool {
	on, ok := v.seen[c.Question]
	if !ok || on.Type == ItemGroup || on.Type == ItemDisplay || on.Type == ItemAttachment {
		return false
	}
	if c.Operator == OpExists {
		return answers(c) == 1 && c.Boolean != nil
	}
	if answers(c) != 1 || !operator(c.Operator) {
		return false
	}
	ordered := c.Operator != OpEqual && c.Operator != OpNotEqual
	switch on.Type {
	case ItemBoolean:
		return !ordered && c.Boolean != nil
	case ItemChoice:
		return !ordered && option(on.Options, c.String)
	case ItemInteger:
		return c.Integer != nil
	case ItemDecimal:
		return decimalPattern.MatchString(string(c.Decimal))
	case ItemDate:
		d, err := time.Parse(time.DateOnly, c.Date)
		return err == nil && d.Year() > 0
	}
	return false // a string or a text: answered or not only
}

// answers counts the answer[x] that c sets.
func answers(c Condition) int {
	n := 0
	for _, set := range []bool{c.Boolean != nil, c.Integer != nil, c.Decimal != "", c.String != "", c.Date != ""} {
		if set {
			n++
		}
	}
	return n
}

func operator(op Operator) bool {
	switch op {
	case OpEqual, OpNotEqual, OpGreater, OpLess, OpGreaterEqual, OpLessEqual:
		return true
	}
	return false
}

func option(opts []Option, s string) bool {
	for _, o := range opts {
		if o.Value == s {
			return true
		}
	}
	return false
}
