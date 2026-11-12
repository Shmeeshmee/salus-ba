package terminology

import (
	"fmt"
	"strings"
)

// GroupOther is the diagnosis group of every code outside the ranges (docs/plans/analytics.md, appendix A).
const GroupOther = "other"

// DiagnosisGroup is one of our own diagnosis groups (ADR 0012): a stable code, the group it is a subgroup of
// (empty for a group), and its names, by language in the enum's order. Reference data.
type DiagnosisGroup struct {
	Code   string
	Parent string
	Names  []GroupName
}

// GroupName is a group's name in one language, with its review (the term review panel).
type GroupName struct {
	Review
	Language  string
	Text      string
	VersionID int32
}

// GroupRange is an inclusive range of ICD-10 codes of a group, compared on a code's first len(From) characters.
type GroupRange struct {
	Group, From, To string
}

// Groups is the grouping of ICD-10 codes: every code has one group and at most one subgroup.
type Groups struct {
	parent map[string]string
	groups []DiagnosisGroup
	ranges []GroupRange
}

// NewGroups checks that groups and ranges form one grouping (ErrInvalidGroups otherwise) and returns it.
func NewGroups(groups []DiagnosisGroup, ranges []GroupRange) (Groups, error) {
	g := Groups{groups: groups, ranges: ranges, parent: make(map[string]string, len(groups))}
	for _, d := range groups {
		g.parent[d.Code] = d.Parent
	}
	if _, ok := g.parent[GroupOther]; !ok {
		return Groups{}, fmt.Errorf("%w: no group %q", ErrInvalidGroups, GroupOther)
	}
	for _, d := range groups {
		if p, ok := g.parent[d.Parent]; d.Parent != "" && (!ok || p != "") {
			return Groups{}, fmt.Errorf("%w: %s is not a subgroup of a group", ErrInvalidGroups, d.Code)
		}
	}
	for i, r := range ranges {
		if err := g.check(r, ranges[:i]); err != nil {
			return Groups{}, err
		}
	}
	return g, nil
}

// check refuses r when it is malformed, overlaps an earlier range of its level, or lies outside its parent.
func (g Groups) check(r GroupRange, earlier []GroupRange) error {
	parent, known := g.parent[r.Group]
	switch {
	case !known || r.Group == GroupOther:
		return fmt.Errorf("%w: a range of %q", ErrInvalidGroups, r.Group)
	case len(r.From) != len(r.To) || r.From > r.To:
		return fmt.Errorf("%w: the range %s-%s of %s", ErrInvalidGroups, r.From, r.To, r.Group)
	}
	for _, e := range earlier {
		if (g.parent[e.Group] == "") == (parent == "") && overlap(r, e) {
			return fmt.Errorf("%w: %s-%s of %s and %s-%s of %s", ErrInvalidGroups, r.From, r.To, r.Group, e.From, e.To, e.Group)
		}
	}
	if parent != "" && !g.within(r, parent) {
		return fmt.Errorf("%w: %s-%s of %s is outside %s", ErrInvalidGroups, r.From, r.To, r.Group, parent)
	}
	return nil
}

// within reports whether every code of r falls in a range of group.
func (g Groups) within(r GroupRange, group string) bool {
	for _, p := range g.ranges {
		if p.Group == group && len(p.From) <= len(r.From) && matches(p, r.From) && matches(p, r.To) {
			return true
		}
	}
	return false
}

// overlap reports whether some code falls in both ranges.
func overlap(a, b GroupRange) bool {
	if len(a.From) > len(b.From) {
		a, b = b, a
	}
	n := len(a.From)
	return b.From[:n] <= a.To && b.To[:n] >= a.From
}

// matches reports whether code falls in r.
func matches(r GroupRange, code string) bool {
	if len(code) < len(r.From) {
		return false
	}
	prefix := code[:len(r.From)]
	return r.From <= prefix && prefix <= r.To
}

// Of returns the group of an ICD-10 code (GroupOther when no range has it) and its subgroup, if any.
func (g Groups) Of(code string) (group, subgroup string) {
	code = strings.ToUpper(code)
	group = GroupOther
	for _, r := range g.ranges {
		if !matches(r, code) {
			continue
		}
		if p := g.parent[r.Group]; p != "" {
			subgroup = r.Group
		} else {
			group = r.Group
		}
	}
	return group, subgroup
}

// List returns the groups in their order.
func (g Groups) List() []DiagnosisGroup { return g.groups }

// Place is a place of a clinic's location (00041): an entity, a leaf region (an ISO 3166-2:BA code) or a city (a
// Wikidata id), with its parent (empty for an entity and for a region that is its own entity) and its name, a
// proper noun, in Latin and Cyrillic script. Reference data.
type Place struct {
	ID       string
	Kind     string
	Parent   string
	NameLatn string
	NameCyrl string
}
