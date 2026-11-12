package terminology_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// fakeCatalog is the CatalogRepository: two groups, one subgroup, two places.
type fakeCatalog struct{ err error }

func (f fakeCatalog) DiagnosisGroups(context.Context) ([]terminology.DiagnosisGroup, []terminology.GroupRange, error) {
	name := func(text string) []terminology.GroupName {
		return []terminology.GroupName{
			{Language: "bs", Text: text + " (bs)", Review: terminology.Review{Status: terminology.ReviewUnreviewed}},
			{Language: "en", Text: text + " (en)", Review: terminology.Review{Status: terminology.ReviewApproved}},
		}
	}
	return []terminology.DiagnosisGroup{
		{Code: "respiratory-infections", Names: name("Respiratory infections")},
		{Code: "common-cold", Parent: "respiratory-infections", Names: name("Common cold")},
		{Code: "other", Names: name("Other")},
	}, []terminology.GroupRange{
		{Group: "respiratory-infections", From: "J00", To: "J22"}, {Group: "common-cold", From: "J00", To: "J00"},
	}, f.err
}

func (f fakeCatalog) Grouping(ctx context.Context) ([]terminology.DiagnosisGroup, []terminology.GroupRange, error) {
	groups, ranges, err := f.DiagnosisGroups(ctx)
	for i := range groups {
		groups[i].Names = nil
	}
	return groups, ranges, err
}

func (f fakeCatalog) Places(context.Context) ([]terminology.Place, error) {
	return []terminology.Place{
		{ID: "BA-BIH", Kind: "entity", NameLatn: "Federacija Bosne i Hercegovine"},
		{ID: "BA-09", Kind: "region", Parent: "BA-BIH", NameLatn: "Kanton Sarajevo"},
	}, f.err
}

var catalogStaff = terminology.Caller{AccountID: "00000000-0000-4000-8000-0000000000a1", Staff: true}

func TestCatalogGroups_ReturnsTheGrouping(t *testing.T) {
	g, err := terminology.NewCatalog(fakeCatalog{}).Groups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if group, sub := g.Of("J00.1"); group != "respiratory-infections" || sub != "common-cold" {
		t.Errorf("Of(J00.1) = %s, %s", group, sub)
	}
}

func TestCatalogGroupNames_NamesEachGroupInTheLanguageForStaffOnly(t *testing.T) {
	cat := terminology.NewCatalog(fakeCatalog{})
	got, err := cat.GroupNames(context.Background(), catalogStaff, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := []terminology.GroupEntry{
		{Code: "respiratory-infections", Name: "Respiratory infections (en)", ReviewStatus: terminology.ReviewApproved},
		{Code: "common-cold", Parent: "respiratory-infections", Name: "Common cold (en)", ReviewStatus: terminology.ReviewApproved},
		{Code: "other", Name: "Other (en)", ReviewStatus: terminology.ReviewApproved},
	}
	if len(got) != len(want) {
		t.Fatalf("GroupNames = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GroupNames[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if _, err := cat.GroupNames(context.Background(), terminology.Caller{AccountID: catalogStaff.AccountID}, "en"); !errors.Is(err, terminology.ErrForbidden) {
		t.Errorf("a patient: %v, want ErrForbidden", err)
	}
	for _, lang := range []string{"", "la", "xx"} {
		if _, err := cat.GroupNames(context.Background(), catalogStaff, lang); !errors.Is(err, terminology.ErrInvalidSearch) {
			t.Errorf("language %q: %v, want ErrInvalidSearch", lang, err)
		}
	}
}

func TestCatalogPlaces_ForStaffOnly(t *testing.T) {
	cat := terminology.NewCatalog(fakeCatalog{})
	if got, err := cat.Places(context.Background(), catalogStaff); err != nil || len(got) != 2 || got[1].ID != "BA-09" {
		t.Errorf("Places = %+v, %v", got, err)
	}
	if _, err := cat.Places(context.Background(), terminology.Caller{}); !errors.Is(err, terminology.ErrForbidden) {
		t.Errorf("a patient: %v, want ErrForbidden", err)
	}
}

func TestCatalog_WrapsTheRepositorysError(t *testing.T) {
	boom := errors.New("boom")
	cat := terminology.NewCatalog(fakeCatalog{err: boom})
	if _, err := cat.Groups(context.Background()); !errors.Is(err, boom) {
		t.Errorf("Groups = %v", err)
	}
	if _, err := cat.Places(context.Background(), catalogStaff); !errors.Is(err, boom) {
		t.Errorf("Places = %v", err)
	}
}
