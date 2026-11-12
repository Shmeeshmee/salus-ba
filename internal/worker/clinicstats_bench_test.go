package worker

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// benchClinic is a synthetic clinic of 20 practitioners with three years of appointments and Conditions
// (2024-01-01 to 2026-12-31, up to eight of each per practitioner and day), refreshed day by day: the size the plan's
// benchmark names (docs/plans/analytics.md §4). make bench-clinicstats runs both benchmarks.
func benchClinic(b *testing.B) (statsPools, string, *clinicstats.Builder) {
	b.Helper()
	p := newStatsPools(b)
	org, roles := seedStatsRoles(b, p.owner, 20)
	first, last := date(2024, 1, 1), date(2026, 12, 31)
	synthetic(b, p.owner, org, roles, first, last, 8)
	if _, err := p.owner.Exec(context.Background(), "ANALYZE"); err != nil {
		b.Fatal(err)
	}
	builder := newClinicStats(p.stats, clock.Fixed{T: localAt(2027, 1, 1, 9, 0)})
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if err := builder.Refresh(context.Background(), org, localAt(d.Year(), d.Month(), d.Day(), 12, 0)); err != nil {
			b.Fatal(err)
		}
	}
	if _, err := p.owner.Exec(context.Background(), "ANALYZE"); err != nil {
		b.Fatal(err)
	}
	return p, org, builder
}

// The refresh of one clinic-day: about 150 appointments and 160 Conditions read, counted and written in one
// transaction.
func BenchmarkClinicStatsRefresh_OneClinicDay(b *testing.B) {
	_, org, builder := benchClinic(b)
	at := localAt(2026, 6, 15, 12, 0)
	for b.Loop() {
		if err := builder.Refresh(b.Context(), org, at); err != nil {
			b.Fatal(err)
		}
	}
}

// A yearly overview as the admin opens it: the appointments (series, heatmap, practitioners) and the diagnoses by
// group, of one year of the three, each audited.
func BenchmarkClinicStatsYearlyOverview(b *testing.B) {
	p, org, _ := benchClinic(b)
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(p.app), Audit: recordNothing{}, Tx: postgres.NewTransactor(p.app)})
	c := clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true}
	from, to := date(2025, 1, 1), date(2025, 12, 31)
	for b.Loop() {
		f, err := svc.Appointments(b.Context(), c, clinicstats.AppointmentQuery{From: from, To: to})
		if err != nil || len(f.Series) != 12 || len(f.Practitioners) != 20 {
			b.Fatalf("Appointments() = %d months, %d practitioners, %v", len(f.Series), len(f.Practitioners), err)
		}
		if _, err := svc.Diagnoses(b.Context(), c, clinicstats.DiagnosisQuery{From: from, To: to, By: clinicstats.ByGroup}); err != nil {
			b.Fatal(err)
		}
	}
}
