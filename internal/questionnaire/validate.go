package questionnaire

import (
	"strings"
	"unicode/utf8"
)

// The limits of SPEC.md §2 "Limits", in characters.
const (
	maxItems       = 200 // sections and instructions included
	maxTitle       = 200
	maxDescription = 1000
	maxText        = 500
	maxHelp        = 500
	minOptions     = 2
	maxOptions     = 50
	maxOption      = 200
	maxLinkID      = 64
)

// validate checks a questionnaire's content against SPEC.md §2 "Item model", "Limits" and "linkIds".
// earlier is the kind of every linkId of the series' other versions, which a new version may not change. It
// returns an *InvalidError, or nil.
func validate(in Content, earlier map[string]string) error {
	v := validator{seen: map[string]Item{}, earlier: earlier}
	for _, it := range in.Items {
		v.item(it, true)
	}
	if blank(in.Title) || chars(in.Title) > maxTitle || chars(in.Description) > maxDescription ||
		v.count > maxItems || v.malformed || len(v.bad) > 0 {
		return &InvalidError{LinkIDs: v.bad}
	}
	return nil
}

// validator walks the item tree in document order.
type validator struct {
	seen      map[string]Item   // the items before the current one, by linkId
	earlier   map[string]string // kind by linkId in the series' other versions
	bad       []string          // the offending items' linkIds, in document order
	count     int
	malformed bool // a linkId is malformed, so it is not named
}

func (v *validator) item(it Item, top bool) {
	v.count++
	if !linkID(it.LinkID) {
		v.malformed = true
	} else if _, dup := v.seen[it.LinkID]; dup {
		v.bad = append(v.bad, it.LinkID)
	} else {
		if !v.valid(it, top) {
			v.bad = append(v.bad, it.LinkID)
		}
		v.seen[it.LinkID] = it
	}
	for _, child := range it.Items {
		v.item(child, false)
	}
}

// valid tells whether the item's own fields keep the rules; its children are checked on their own.
func (v *validator) valid(it Item, top bool) bool {
	if blank(it.Text) || chars(it.Text) > maxText || chars(it.Help) > maxHelp {
		return false
	}
	if kind, ok := v.earlier[it.LinkID]; ok && kind != kindOf(it) {
		return false
	}
	if it.EnableWhen != nil && !v.condition(*it.EnableWhen) {
		return false
	}
	question := it.Type != ItemGroup && it.Type != ItemDisplay
	switch {
	case it.Required && !question, it.Repeats && it.Type != ItemChoice, len(it.Options) > 0 && it.Type != ItemChoice:
		return false
	case len(it.Items) > 0 && it.Type != ItemGroup:
		return false
	}
	switch it.Type {
	case ItemGroup:
		return top
	case ItemDisplay:
		return it.Help == ""
	case ItemChoice:
		return options(it.Options)
	case ItemString, ItemText, ItemBoolean, ItemInteger, ItemDecimal, ItemDate:
		return true
	}
	return false // ItemAttachment (reserved, D1) and any type not accepted yet
}

// condition tells whether c depends on an earlier boolean item with a boolean answer, or an earlier choice
// item with one of its options.
func (v *validator) condition(c Condition) bool {
	on, ok := v.seen[c.Question]
	switch {
	case !ok:
		return false
	case on.Type == ItemBoolean:
		return c.Boolean != nil && c.String == ""
	case on.Type == ItemChoice && c.Boolean == nil:
		for _, o := range on.Options {
			if o.Value == c.String {
				return true
			}
		}
	}
	return false
}

func options(opts []Option) bool {
	if len(opts) < minOptions || len(opts) > maxOptions {
		return false
	}
	seen := make(map[string]bool, len(opts))
	for _, o := range opts {
		if blank(o.Value) || chars(o.Value) > maxOption || seen[o.Value] {
			return false
		}
		seen[o.Value] = true
	}
	return true
}

// kindOf is the item's type as a new version must keep it: single and multiple choice differ.
func kindOf(it Item) string {
	if it.Type == ItemChoice && it.Repeats {
		return string(it.Type) + "+repeats"
	}
	return string(it.Type)
}

// addKinds adds the kind of every item of the tree to kinds.
func addKinds(kinds map[string]string, items []Item) {
	for _, it := range items {
		kinds[it.LinkID] = kindOf(it)
		addKinds(kinds, it.Items)
	}
}

// linkID tells whether id is 1 to 64 of A–Z, a–z, 0–9 and "-".
func linkID(id string) bool {
	if id == "" || len(id) > maxLinkID {
		return false
	}
	for _, r := range id {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

func chars(s string) int { return utf8.RuneCountInString(s) }
