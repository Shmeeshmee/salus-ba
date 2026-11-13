package clinicstats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// recentDays is how many local days, today included, the nightly pass recomputes (docs/plans/analytics.md §4).
const recentDays = 35

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Appointments gives a clinic's appointments starting in [from, to) (appointment.Stats), asked without a key.
type Appointments interface {
	Starts(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]appointment.Start, error)
}

// Conditions gives a clinic's confirmed Conditions recorded in [from, to) (condition.Service), asked without a key.
type Conditions interface {
	Recorded(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]condition.Recorded, error)
}

// Specialties gives a clinic's roles with their specialty (practitionerrole.Stats).
type Specialties interface {
	Specialties(ctx context.Context, orgID string) (map[string]string, error)
}

// Clinics gives every clinic with its time zone (organization.Stats).
type Clinics interface {
	Locations(ctx context.Context) ([]organization.Location, error)
}

// Grouping gives the diagnosis groups of the ICD-10 codes (terminology.Catalog).
type Grouping interface {
	Groups(ctx context.Context) (terminology.Groups, error)
}

// BuildPorts are the Builder's dependencies, all on salus_stats.
type BuildPorts struct {
	Repo         BuildRepository
	Tx           Transactor
	Clock        clock.Clock
	Clinics      Clinics
	Appointments Appointments
	Conditions   Conditions
	Specialties  Specialties
	Groups       Grouping
}

// Builder computes the aggregates from their sources. It reads no patient identifier: the sources are asked
// without a pseudonym key. Not audited: it reads and writes counts only, on behalf of no one.
type Builder struct{ p BuildPorts }

// NewBuilder returns the Builder on its ports.
func NewBuilder(p BuildPorts) *Builder { return &Builder{p: p} }

// Refresh recomputes the clinic's local day that holds the instant at, in one transaction (the clinicstats_refresh
// job). ErrUnknownClinic for a clinic the builds do not know.
func (b *Builder) Refresh(ctx context.Context, orgID string, at time.Time) error {
	locations, err := b.p.Clinics.Locations(ctx)
	if err != nil {
		return fmt.Errorf("clinicstats: refresh: %w", err)
	}
	for _, l := range locations {
		if l.ID != orgID {
			continue
		}
		zone, err := time.LoadLocation(l.TimeZone)
		if err != nil {
			return fmt.Errorf("clinicstats: refresh: %w", err)
		}
		d := date(at.In(zone))
		return b.build(ctx, orgID, zone, d, d)
	}
	return ErrUnknownClinic
}

// RefreshRecent recomputes the last 35 local days of every clinic, today included, one transaction per clinic
// (the clinicstats_nightly job). A failing clinic does not stop the others; their errors are joined.
func (b *Builder) RefreshRecent(ctx context.Context) error {
	locations, err := b.p.Clinics.Locations(ctx)
	if err != nil {
		return fmt.Errorf("clinicstats: refresh recent: %w", err)
	}
	now := b.p.Clock.Now()
	var errs []error
	for _, l := range locations {
		zone, err := time.LoadLocation(l.TimeZone)
		if err != nil {
			errs = append(errs, fmt.Errorf("clinicstats: refresh recent: %w", err))
			continue
		}
		last := date(now.In(zone))
		errs = append(errs, b.build(ctx, l.ID, zone, last.AddDate(0, 0, 1-recentDays), last))
	}
	return errors.Join(errs...)
}

// build recomputes the clinic's local days first to last in zone, in one transaction under the clinic's lock.
func (b *Builder) build(ctx context.Context, orgID string, zone *time.Location, first, last time.Time) error {
	from := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, zone)
	to := time.Date(last.Year(), last.Month(), last.Day()+1, 0, 0, 0, 0, zone)
	err := b.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := b.p.Repo.Lock(ctx, orgID); err != nil {
			return err
		}
		specialties, err := b.p.Specialties.Specialties(ctx, orgID)
		if err != nil {
			return err
		}
		groups, err := b.p.Groups.Groups(ctx)
		if err != nil {
			return err
		}
		starts, err := b.p.Appointments.Starts(ctx, orgID, from, to, nil)
		if err != nil {
			return err
		}
		recorded, err := b.p.Conditions.Recorded(ctx, orgID, from, to, nil)
		if err != nil {
			return err
		}
		return b.p.Repo.Replace(ctx, orgID, first, last,
			countAppointments(starts, specialties, zone), countDiagnoses(recorded, specialties, groups, zone))
	})
	if err != nil {
		return fmt.Errorf("clinicstats: build: %w", err)
	}
	return nil
}

// date is t's calendar date as the aggregates hold it: midnight UTC.
func date(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// countAppointments counts the appointments by local day and hour of their start, role, status and how they were
// booked.
func countAppointments(starts []appointment.Start, specialties map[string]string, zone *time.Location) []AppointmentCount {
	type key struct {
		day               time.Time
		role, status, via string
		hour              int
	}
	n := map[key]int32{}
	var order []key
	for _, a := range starts {
		t := a.Start.In(zone)
		k := key{day: date(t), hour: t.Hour(), role: a.PractitionerRoleID, status: string(a.Status), via: string(a.BookedVia)}
		if n[k] == 0 {
			order = append(order, k)
		}
		n[k]++
	}
	out := make([]AppointmentCount, len(order))
	for i, k := range order {
		out[i] = AppointmentCount{
			Day: k.day, Hour: int32(k.hour), PractitionerRoleID: k.role, SpecialtyCode: specialties[k.role], Status: k.status, BookedVia: k.via, Count: n[k],
		}
	}
	return out
}

// countDiagnoses counts the Conditions by local day and hour of their signing, role and code, with its groups.
func countDiagnoses(recorded []condition.Recorded, specialties map[string]string, groups terminology.Groups, zone *time.Location) []DiagnosisCount {
	type key struct {
		day        time.Time
		role, code string
		hour       int
	}
	n := map[key]int32{}
	var order []key
	for _, c := range recorded {
		t := c.RecordedAt.In(zone)
		k := key{day: date(t), hour: t.Hour(), role: c.PractitionerRoleID, code: c.Code}
		if n[k] == 0 {
			order = append(order, k)
		}
		n[k]++
	}
	out := make([]DiagnosisCount, len(order))
	for i, k := range order {
		group, subgroup := groups.Of(k.code)
		out[i] = DiagnosisCount{
			Day: k.day, Hour: int32(k.hour), PractitionerRoleID: k.role, SpecialtyCode: specialties[k.role],
			Group: group, Subgroup: subgroup, Code: k.code, Count: n[k],
		}
	}
	return out
}
