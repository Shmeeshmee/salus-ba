package terminology_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// A slice of appendix A (docs/plans/analytics.md): two groups, one with subgroups, ranges of two lengths.
func sampleGroups(t *testing.T) terminology.Groups {
	t.Helper()
	g, err := terminology.NewGroups(
		[]terminology.DiagnosisGroup{
			{Code: "respiratory-infections"}, {Code: "common-cold", Parent: "respiratory-infections"},
			{Code: "upper-airway-infections", Parent: "respiratory-infections"},
			{Code: "covid-19"}, {Code: "other"},
		},
		[]terminology.GroupRange{
			{Group: "respiratory-infections", From: "J00", To: "J22"},
			{Group: "common-cold", From: "J00", To: "J00"},
			{Group: "upper-airway-infections", From: "J01", To: "J06"},
			{Group: "covid-19", From: "U07.1", To: "U07.2"},
			{Group: "covid-19", From: "U08", To: "U10"},
		})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGroupsOf_MapsACodeToItsGroupAndSubgroup(t *testing.T) {
	g := sampleGroups(t)
	for code, want := range map[string][2]string{
		"J00":   {"respiratory-infections", "common-cold"},
		"J00.0": {"respiratory-infections", "common-cold"},
		"J06.9": {"respiratory-infections", "upper-airway-infections"},
		"J18.9": {"respiratory-infections", ""},
		"J22":   {"respiratory-infections", ""},
		"J23":   {"other", ""},
		"U07.1": {"covid-19", ""},
		"U07.0": {"other", ""}, // vaping, not COVID-19
		"U07":   {"other", ""}, // shorter than the range's codes
		"U09.9": {"covid-19", ""},
		"A09":   {"other", ""},
	} {
		if group, sub := g.Of(code); group != want[0] || sub != want[1] {
			t.Errorf("Of(%q) = %q, %q; want %q, %q", code, group, sub, want[0], want[1])
		}
	}
}

func TestNewGroups_RefusesRangesThatOverlapOrLeaveTheirParent(t *testing.T) {
	groups := []terminology.DiagnosisGroup{{Code: "a"}, {Code: "b"}, {Code: "a1", Parent: "a"}, {Code: "a2", Parent: "a"}, {Code: "other"}}
	for name, ranges := range map[string][]terminology.GroupRange{
		"two groups":               {{Group: "a", From: "J00", To: "J10"}, {Group: "b", From: "J10", To: "J20"}},
		"two lengths":              {{Group: "a", From: "J00", To: "J10"}, {Group: "b", From: "J05.1", To: "J05.3"}},
		"two subgroups":            {{Group: "a", From: "J00", To: "J10"}, {Group: "a1", From: "J01", To: "J03"}, {Group: "a2", From: "J03", To: "J04"}},
		"a subgroup outside":       {{Group: "a", From: "J00", To: "J10"}, {Group: "a1", From: "J09", To: "J11"}},
		"a range of no group":      {{Group: "z", From: "J00", To: "J10"}},
		"a range of other":         {{Group: "other", From: "J00", To: "J10"}},
		"a range in reverse":       {{Group: "a", From: "J10", To: "J00"}},
		"ends of different length": {{Group: "a", From: "J10", To: "J10.5"}},
	} {
		if _, err := terminology.NewGroups(groups, ranges); !errors.Is(err, terminology.ErrInvalidGroups) {
			t.Errorf("%s: NewGroups = %v, want ErrInvalidGroups", name, err)
		}
	}
	if _, err := terminology.NewGroups(groups[:4], nil); !errors.Is(err, terminology.ErrInvalidGroups) {
		t.Errorf("without the group other: NewGroups = %v, want ErrInvalidGroups", err)
	}
	if _, err := terminology.NewGroups(append(groups, terminology.DiagnosisGroup{Code: "a3", Parent: "a1"}), nil); !errors.Is(err, terminology.ErrInvalidGroups) {
		t.Errorf("a subgroup of a subgroup: NewGroups = %v, want ErrInvalidGroups", err)
	}
}

func TestGroupsList_KeepsTheGivenOrder(t *testing.T) {
	g := sampleGroups(t)
	var codes []string
	for _, d := range g.List() {
		codes = append(codes, d.Code)
	}
	if want := "respiratory-infections common-cold upper-airway-infections covid-19 other"; strings.Join(codes, " ") != want {
		t.Errorf("List() = %s, want %s", strings.Join(codes, " "), want)
	}
}
