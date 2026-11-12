package terminology_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The seeded grouping (00041, appendix A of docs/plans/analytics.md) is one grouping, in the appendix's order,
// every group named in the ten languages and unreviewed, and maps the appendix's examples as listed.
func TestCatalogRepository_TheSeededGroupsMapTheAppendixExamples(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	groups, ranges, err := terminology.NewPostgresCatalogRepository(app).DiagnosisGroups(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	g, err := terminology.NewGroups(groups, ranges)
	if err != nil {
		t.Fatalf("the seeded groups: %v", err)
	}
	if len(groups) != 34 || groups[0].Code != "intestinal-infections" || groups[1].Parent != "intestinal-infections" ||
		groups[len(groups)-1].Code != terminology.GroupOther {
		t.Errorf("the groups' order: %d groups, first %+v", len(groups), groups[:2])
	}
	for _, d := range groups {
		if len(d.Names) != 10 || d.Names[0].Language != "bs" || d.Names[9].Language != "fr" {
			t.Fatalf("%s has %d names, want the ten languages in the enum's order", d.Code, len(d.Names))
		}
		for _, n := range d.Names {
			if n.Status != terminology.ReviewUnreviewed || n.Text == "" || n.VersionID != 1 {
				t.Errorf("%s in %s: %+v, want an unreviewed name at version 1", d.Code, n.Language, n)
			}
		}
	}
	for code, want := range map[string][2]string{
		"J00": {"respiratory-infections", "common-cold"}, "J06.9": {"respiratory-infections", "upper-airway-infections"},
		"J45.0": {"asthma-chronic-lung", ""}, "J30.1": {"other-respiratory", ""}, "I10": {"high-blood-pressure", ""},
		"E11.9": {"diabetes", ""}, "E03.9": {"endocrine-metabolic", ""}, "M54.5": {"musculoskeletal", "back-neck"},
		"A09": {"intestinal-infections", "gastroenteritis"}, "U07.1": {"covid-19", ""}, "U09.9": {"covid-19", ""},
		"U07.0": {"other", ""}, "W19": {"other", ""}, "Z00.0": {"checkups", ""},
	} {
		if group, sub := g.Of(code); group != want[0] || sub != want[1] {
			t.Errorf("Of(%q) = %q, %q; want %q, %q", code, group, sub, want[0], want[1])
		}
	}
}

// The places: every city under its leaf region, the regions after the entity, each in both scripts.
func TestCatalogRepository_ListsThePlaces(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	places, err := terminology.NewPostgresCatalogRepository(app).Places(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	byID := map[string]terminology.Place{}
	for _, p := range places {
		kinds[p.Kind]++
		byID[p.ID] = p
	}
	if kinds["entity"] != 1 || kinds["region"] != 12 || kinds["city"] != 143 {
		t.Errorf("kinds = %v, want 1 entity, 12 regions, 143 cities", kinds)
	}
	if p := byID["BA-09"]; p.Parent != "BA-BIH" || p.NameLatn != "Kanton Sarajevo" || p.NameCyrl != "Кантон Сарајево" {
		t.Errorf("BA-09 = %+v", p)
	}
	if p := byID["BA-SRP"]; p.Kind != "region" || p.Parent != "" {
		t.Errorf("BA-SRP = %+v, want a region of its own", p)
	}
	if p := byID["Q15940918"]; p.Kind != "city" || p.Parent != "BA-SRP" || p.NameLatn != "Banja Luka" {
		t.Errorf("Banja Luka = %+v", p)
	}
	if places[0].Kind != "entity" || places[len(places)-1].Kind != "city" {
		t.Errorf("places are not listed entity, regions, cities: first %+v, last %+v", places[0], places[len(places)-1])
	}
}
