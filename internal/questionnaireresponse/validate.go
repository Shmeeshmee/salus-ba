package questionnaireresponse

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// The answer limits of SPEC.md §2 "Limits". The integer range is the type's (Value.Integer is an int32).
const (
	maxString   = 255  // characters
	maxText     = 4000 // characters
	maxDecimals = 15   // significant digits
)

// decimalPattern is a decimal written plainly: FHIR's decimal without an exponent.
var decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// checkAnswers checks answers against the items of their version: each answer to a question of the version,
// once, and fitting its item's type and limits. It returns an *AnswersError naming the offending items in
// document order, or nil. Conditions and required questions are checked on submit (submitted).
func checkAnswers(items []questionnaire.Item, answers []Answer) error {
	if bad, unknown := invalid(items, answers); len(bad) > 0 || unknown {
		return &AnswersError{LinkIDs: inOrder(items, bad)}
	}
	return nil
}

// invalid gives the linkIds of the answers that are to an item twice or break their item, and whether an
// answer is to no question of the version.
func invalid(items []questionnaire.Item, answers []Answer) (bad map[string]bool, unknown bool) {
	all := index(items)
	given := make(map[string]bool, len(answers))
	bad = map[string]bool{}
	for _, a := range answers {
		it, ok := all[a.LinkID]
		switch {
		case !ok:
			unknown = true
		case given[a.LinkID], !fits(it, a.Values):
			bad[a.LinkID] = true
		}
		given[a.LinkID] = true
	}
	return bad, unknown
}

// index gives every item of the tree by linkId.
func index(items []questionnaire.Item) map[string]questionnaire.Item {
	out := map[string]questionnaire.Item{}
	var walk func([]questionnaire.Item)
	walk = func(items []questionnaire.Item) {
		for _, it := range items {
			out[it.LinkID] = it
			walk(it.Items)
		}
	}
	walk(items)
	return out
}

// inOrder gives the linkIds of the set, in the items' document order.
func inOrder(items []questionnaire.Item, set map[string]bool) []string {
	var out []string
	var walk func([]questionnaire.Item)
	walk = func(items []questionnaire.Item) {
		for _, it := range items {
			if set[it.LinkID] {
				out = append(out, it.LinkID)
			}
			walk(it.Items)
		}
	}
	walk(items)
	return out
}

// fits tells whether values answer the item: at least one value, several only for a choice with repeats,
// each of the item's type and within its limits, a choice's options each once.
func fits(it questionnaire.Item, values []Value) bool {
	if len(values) == 0 || (len(values) > 1 && (it.Type != questionnaire.ItemChoice || !it.Repeats)) {
		return false
	}
	chosen := make(map[string]bool, len(values))
	for _, v := range values {
		if !fitsOne(it, v) || chosen[v.String] {
			return false
		}
		if it.Type == questionnaire.ItemChoice {
			chosen[v.String] = true
		}
	}
	return true
}

// fitsOne tells whether v, exactly one value[x], is of the item's type and within its limits. An attachment
// item has no valid answer here: its files are added and removed only through binary's endpoints
// (docs/attachments/SPEC.md §2 "Where an attachment answer lives").
func fitsOne(it questionnaire.Item, v Value) bool {
	switch it.Type {
	case questionnaire.ItemBoolean:
		return v == Value{Boolean: v.Boolean} && v.Boolean != nil
	case questionnaire.ItemInteger:
		return v == Value{Integer: v.Integer} && v.Integer != nil
	case questionnaire.ItemDecimal:
		return v == Value{Decimal: v.Decimal} && decimal(string(v.Decimal))
	case questionnaire.ItemDate:
		return v == Value{Date: v.Date} && date(v.Date)
	case questionnaire.ItemString:
		return v == Value{String: v.String} && text(v.String, maxString)
	case questionnaire.ItemText:
		return v == Value{String: v.String} && text(v.String, maxText)
	case questionnaire.ItemChoice:
		return v == Value{String: v.String} && option(it.Options, v.String)
	}
	return false
}

func text(s string, limit int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= limit
}

func option(opts []questionnaire.Option, s string) bool {
	for _, o := range opts {
		if o.Value == s {
			return true
		}
	}
	return false
}

// decimal tells whether s is a plainly written decimal of at most 15 significant digits (leading zeros are
// not significant, trailing ones are).
func decimal(s string) bool {
	if !decimalPattern.MatchString(s) {
		return false
	}
	digits := strings.TrimLeft(strings.ReplaceAll(strings.TrimPrefix(s, "-"), ".", ""), "0")
	return len(digits) <= maxDecimals
}

// date tells whether s is a full date, YYYY-MM-DD, that exists, from the year 1.
func date(s string) bool {
	d, err := time.Parse(time.DateOnly, s)
	return err == nil && len(s) == len(time.DateOnly) && d.Year() > 0
}
