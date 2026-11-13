package trends_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/trends"
)

type fakeClinics map[string]bool

func (f fakeClinics) Contributes(_ context.Context, id string) (bool, error) { return f[id], nil }

var (
	staff      = trends.Caller{AccountID: "account-1", OrganizationID: "clinic-0", Staff: true}
	optedOut   = trends.Caller{AccountID: "account-2", OrganizationID: "clinic-9", Staff: true}
	patientOne = trends.Caller{AccountID: "account-3"}
)

func newView(repo *memRepo, audit *fakeAudit) *trends.Service {
	return trends.NewService(trends.Ports{
		Repo: repo, Clinics: fakeClinics{"clinic-0": true}, Catalog: fakeCatalog{}, Audit: audit, Tx: fakeTx{},
		Specialties: []string{"419772000", "394537008"}, K: 10,
	})
}

func cellRow(period, table string, count int, suppressed bool, set func(*trends.Row)) trends.Row {
	r := trends.Row{Period: period, Table: table, Count: count, Suppressed: suppressed}
	if set != nil {
		set(&r)
	}
	return r
}

// published is two months of G×D cells: BA-09 published in both; BA-01 suppressed in October, absent (0) in
// September; BA-SRP absent in both.
func published() *memRepo {
	repo := newMemRepo()
	for _, m := range []string{"2026-09", "2026-10"} {
		repo.periods[m] = trends.Period{Period: m, Kind: trends.KindMonth, Status: trends.StatusPublished, MinCount: 10}
	}
	repo.periods["2026-Q3"] = trends.Period{Period: "2026-Q3", Kind: trends.KindQuarter, Status: trends.StatusPublished}
	repo.periods["2026-W43"] = trends.Period{Period: "2026-W43", Kind: trends.KindWeek, Status: trends.StatusPublished}
	repo.periods["2026-W44"] = trends.Period{Period: "2026-W44", Kind: trends.KindWeek, Status: trends.StatusPublished}
	repo.cells["2026-09"] = []trends.Row{
		cellRow("2026-09", trends.TableGD, 40, false, func(r *trends.Row) { r.Entity, r.Region = "BA-BIH", "BA-09" }),
		cellRow("2026-09", trends.TableGD, 15, false, func(r *trends.Row) { r.Code, r.Group = "J06.9", "respiratory-infections" }),
	}
	repo.cells["2026-10"] = []trends.Row{
		cellRow("2026-10", trends.TableGD, 45, false, func(r *trends.Row) { r.Entity, r.Region = "BA-BIH", "BA-09" }),
		cellRow("2026-10", trends.TableGD, 0, true, func(r *trends.Row) { r.Entity, r.Region = "BA-BIH", "BA-01" }),
		cellRow("2026-10", trends.TableGD, 0, true, func(r *trends.Row) { r.Code, r.Group = "J06.9", "respiratory-infections" }),
		cellRow("2026-10", trends.TableGD, 20, false, func(r *trends.Row) { r.Code, r.Group, r.Subgroup = "J00", "respiratory-infections", "common-cold" }),
		cellRow("2026-10", trends.TableGD, 10, false, func(r *trends.Row) {
			r.Entity, r.Region, r.Code, r.Group = "BA-BIH", "BA-09", "J06.9", "respiratory-infections"
		}),
	}
	repo.cells["2026-Q3"] = []trends.Row{
		cellRow("2026-Q3", trends.TableHeatmap, 35, false, func(r *trends.Row) { r.Weekday, r.Hour = "1", "9" }),
		cellRow("2026-Q3", trends.TableHeatmap, 0, true, func(r *trends.Row) { r.Weekday, r.Hour = "7", "3" }),
	}
	repo.flags["2026-W44"] = []trends.Flag{{Week: "2026-W44", Region: "BA-09", Group: "respiratory-infections"}, {Week: "2026-W44", Region: "BA-01", Group: "flu"}}
	return repo
}

func value(c trends.Count) any {
	if c.Count == nil {
		return nil
	}
	return *c.Count
}

// A8 and A9: every region is a key, null for 0 and for suppressed alike; a sum over months leaves a null month out
// and is partial.
func TestCells_EveryKeyWithNullForZeroAndSuppressedAlike(t *testing.T) {
	audit := &fakeAudit{}
	b, err := newView(published(), audit).Cells(t.Context(), staff, trends.Query{Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "region"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]any{}
	for _, r := range b.Rows {
		got[r.Key] = value(r)
	}
	want := map[string]any{"BA-09": 85, "BA-01": nil, "BA-SRP": nil}
	if len(got) != len(want) || got["BA-09"] != 85 || got["BA-01"] != nil || got["BA-SRP"] != nil {
		t.Errorf("rows %v, want %v", got, want)
	}
	if b.Partial {
		t.Errorf("partial, want not: no published count leaves a month out")
	}
	b, _ = newView(published(), audit).Cells(t.Context(), staff, trends.Query{Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "code"})
	got = map[string]any{}
	for _, r := range b.Rows {
		got[r.Key] = value(r)
	}
	// J06.9 is a key (published at geography = all in September) but suppressed in October; J00 is a key in October.
	if len(got) != 2 || got["J06.9"] != 15 || got["J00"] != 20 || !b.Partial {
		t.Errorf("codes %v partial %v, want J06.9 15 (September only) and J00 20, partial", got, b.Partial)
	}
	if e := audit.events[0]; e.EntityName != "trends_cell" || e.Outcome != auditevent.OutcomeSuccess || e.AgentAccountID != staff.AccountID {
		t.Errorf("audit %+v, want the read by the caller", e)
	}
}

// A8: a code that is not a key of a month is null in every cell of that month: a region's J06.9 in October (published
// there, but J06.9 is suppressed at geography = all) reads null.
func TestCells_ACodeThatIsNoKeyIsNullEverywhere(t *testing.T) {
	b, err := newView(published(), &fakeAudit{}).Cells(t.Context(), staff, trends.Query{
		Table: trends.TableGD, From: "2026-10", To: "2026-10", By: "region", Fixed: map[string]string{"code": "J06.9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Rows {
		if r.Count != nil {
			t.Errorf("%s = %d, want null", r.Key, *r.Count)
		}
	}
}

func TestCells_RefusesCombinationsOutsideTheTables(t *testing.T) {
	for name, q := range map[string]trends.Query{
		"no table":                  {Table: "xx", From: "2026-09", To: "2026-10", By: "region"},
		"specialty in G×D":          {Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "specialty"},
		"region in S×D":             {Table: trends.TableSD, From: "2026-09", To: "2026-10", By: "region"},
		"code in G×S":               {Table: trends.TableGS, From: "2026-09", To: "2026-10", By: "code"},
		"by a fixed dimension":      {Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "region", Fixed: map[string]string{"region": "BA-09"}},
		"by above a fixed level":    {Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "region", Fixed: map[string]string{"city": "Q1"}},
		"region × specialty × code": {Table: trends.TableGS, From: "2026-09", To: "2026-10", By: "specialty", Fixed: map[string]string{"code": "J00"}},
		"a reversed range":          {Table: trends.TableGD, From: "2026-10", To: "2026-09", By: "region"},
		"a quarter":                 {Table: trends.TableGD, From: "2026-Q3", To: "2026-Q3", By: "region"},
		"no breakdown":              {Table: trends.TableGD, From: "2026-09", To: "2026-10"},
		"an empty value":            {Table: trends.TableGD, From: "2026-09", To: "2026-10", By: "month", Fixed: map[string]string{"region": ""}},
		"six years":                 {Table: trends.TableGD, From: "2020-01", To: "2026-10", By: "month"},
	} {
		if _, err := newView(published(), &fakeAudit{}).Cells(t.Context(), staff, q); !errors.Is(err, trends.ErrInvalidQuery) {
			t.Errorf("%s: %v, want ErrInvalidQuery", name, err)
		}
	}
}

// By month: the built months of the range, each with its published count.
func TestCells_ByMonth(t *testing.T) {
	b, err := newView(published(), &fakeAudit{}).Cells(t.Context(), staff, trends.Query{
		Table: trends.TableGD, From: "2026-08", To: "2026-10", By: "month", Fixed: map[string]string{"region": "BA-09"},
	})
	if err != nil || len(b.Rows) != 3 || value(b.Rows[1]) != 40 || value(b.Rows[2]) != 45 || b.Rows[0].Count != nil {
		t.Errorf("by month = %+v, %v; want August null, September 40, October 45", b.Rows, err)
	}
}

// O5, O6: only staff of a contributing clinic see the figures; a refusal is audited too.
func TestView_OnlyStaffOfAContributingClinic(t *testing.T) {
	for _, tt := range []struct {
		want   error
		caller trends.Caller
	}{{trends.ErrNotContributing, optedOut}, {trends.ErrForbidden, patientOne}} {
		audit := &fakeAudit{}
		svc := newView(published(), audit)
		_, err := svc.Overview(t.Context(), tt.caller)
		if !errors.Is(err, tt.want) {
			t.Errorf("%+v: %v, want %v", tt.caller, err, tt.want)
		}
		if _, err := svc.Rises(t.Context(), tt.caller, "", 8); !errors.Is(err, tt.want) {
			t.Errorf("%+v: rises %v, want %v", tt.caller, err, tt.want)
		}
		if len(audit.events) != 2 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure || audit.events[0].EntityName != "" {
			t.Errorf("audit %+v, want the refusals", audit.events)
		}
	}
}

func TestOverviewHeatmapAndRises(t *testing.T) {
	svc := newView(published(), &fakeAudit{})
	o, err := svc.Overview(t.Context(), staff)
	if err != nil || len(o.Months) != 2 || o.Months[1] != "2026-10" || len(o.Quarters) != 1 || o.MinCount != 10 {
		t.Errorf("Overview = %+v, %v", o, err)
	}
	cells, err := svc.Heatmap(t.Context(), staff, "2026-Q3", "")
	if err != nil || len(cells) != 168 || cells[9].Count == nil || *cells[9].Count != 35 || cells[6*24+3].Count != nil {
		t.Errorf("Heatmap = %d cells, %v; want 168 with Monday 9:00 at 35 and Sunday 3:00 null", len(cells), err)
	}
	for _, q := range [][2]string{{"2026-10", ""}, {"2026-Q3", "Q1"}, {"2026-Q3", "BA-99"}} {
		if _, err := svc.Heatmap(t.Context(), staff, q[0], q[1]); !errors.Is(err, trends.ErrInvalidQuery) {
			t.Errorf("Heatmap(%v) = %v, want ErrInvalidQuery", q, err)
		}
	}
	flags, err := svc.Rises(t.Context(), staff, "BA-09", 8)
	if err != nil || len(flags) != 1 || flags[0].Group != "respiratory-infections" {
		t.Errorf("Rises = %+v, %v", flags, err)
	}
	if _, err := svc.Rises(t.Context(), staff, "", 0); !errors.Is(err, trends.ErrInvalidQuery) {
		t.Errorf("Rises(0 weeks) = %v, want ErrInvalidQuery", err)
	}
}
