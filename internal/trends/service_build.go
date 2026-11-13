package trends

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Diagnoses are a clinic's confirmed Conditions, each patient as the build's pseudonym (condition.Service on
// salus_stats).
type Diagnoses interface {
	Recorded(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]condition.Recorded, error)
}

// Appointments are a clinic's fulfilled appointments, each patient as the build's pseudonym (appointment.Stats).
type Appointments interface {
	Fulfilled(ctx context.Context, orgID string, from, to time.Time, key []byte) ([]appointment.Start, error)
}

// Specialties are a clinic's roles with their specialty (practitionerrole.Stats).
type Specialties interface {
	Specialties(ctx context.Context, orgID string) (map[string]string, error)
}

// Locations are the clinics with their location and contribution (organization.Stats).
type Locations interface {
	Locations(ctx context.Context) ([]organization.Location, error)
}

// Catalog is the grouping of ICD-10 codes and the places (terminology.Catalog): reference data.
type Catalog interface {
	Groups(ctx context.Context) (terminology.Groups, error)
	AllPlaces(ctx context.Context) ([]terminology.Place, error)
}

// Auditor records an AuditEvent (auditevent.Service, on the app's pool: salus_stats cannot write audit_event).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Transactor runs fn in a transaction carried by the context (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// BuildPorts are what a build reads and writes.
type BuildPorts struct {
	Diagnoses    Diagnoses
	Appointments Appointments
	Specialties  Specialties
	Locations    Locations
	Catalog      Catalog
	Repo         Repository // where the period is written, once
	Tx           Transactor // Repo's transactions
	Audit        Auditor
	Clock        clock.Clock
	// Agent names the build in its AuditEvent: the worker's job or the operator command.
	Agent string
	K     int // ANALYTICS_MIN_COUNT
}

// Builder builds the closed periods of the platform figures, each once (§5, §6).
type Builder struct{ p BuildPorts }

// NewBuilder returns the Builder on its ports.
func NewBuilder(p BuildPorts) *Builder { return &Builder{p: p} }

// purposeStatistics is the purpose of a build's and a view's AuditEvent: statistics over health data, not care.
const purposeStatistics = "HRESCH"

// Build builds the period (2026-10, 2026-Q3 or 2026-W44) and freezes it. A period that is not closed yet gives
// ErrInvalidPeriod and one built already ErrFrozen, before anything is read. Its pseudonym key is 32 random bytes
// made for this build and dropped with it. The build is audited once its sources are read and before anything is
// written: who built it, and how many rows of each source it read.
func (b *Builder) Build(ctx context.Context, period string) (Summary, error) {
	p, err := parsePeriod(period)
	if err != nil {
		return Summary{}, err
	}
	if b.p.Clock.Now().Before(p.due()) {
		return Summary{}, ErrInvalidPeriod
	}
	built, err := b.p.Repo.Built(ctx, period)
	if err != nil {
		return Summary{}, fmt.Errorf("trends: build: %w", err)
	}
	if built {
		return Summary{}, ErrFrozen
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Summary{}, ErrShortKey
	}
	defer clear(key)
	src, err := b.sources(ctx)
	if err != nil {
		return Summary{}, err
	}
	var s Summary
	switch p.kind {
	case KindMonth:
		s, err = b.month(ctx, p, src, key)
	case KindQuarter:
		s, err = b.quarter(ctx, p, src, key)
	default:
		s, err = b.week(ctx, p, src, key)
	}
	if err != nil {
		return Summary{}, fmt.Errorf("trends: build: %w", err)
	}
	return s, nil
}

// sources are the reference data and the clinics a build starts from.
type sources struct {
	groups    terminology.Groups
	entity    map[string]string // a leaf region's entity: its parent, or itself when it has none
	locations []organization.Location
}

func (b *Builder) sources(ctx context.Context) (sources, error) {
	groups, err := b.p.Catalog.Groups(ctx)
	if err != nil {
		return sources{}, fmt.Errorf("trends: build: %w", err)
	}
	places, err := b.p.Catalog.AllPlaces(ctx)
	if err != nil {
		return sources{}, fmt.Errorf("trends: build: %w", err)
	}
	locations, err := b.p.Locations.Locations(ctx)
	if err != nil {
		return sources{}, fmt.Errorf("trends: build: %w", err)
	}
	entity := map[string]string{}
	for _, pl := range places {
		if pl.Kind == "region" {
			entity[pl.ID] = pl.ID
			if pl.Parent != "" {
				entity[pl.ID] = pl.Parent
			}
		}
	}
	return sources{groups: groups, entity: entity, locations: locations}, nil
}

// counts says the clinic loc counts in a period that starts at since in its zone (§5 "Which clinics count", §6):
// it contributes (O6) and has a location, and neither changed since then.
func counts(loc organization.Location, since time.Time, entity map[string]string) bool {
	_, known := entity[loc.Region]
	return loc.Contributes && loc.City != "" && known && !loc.LocationSince.After(since) && !loc.ContributesSince.After(since)
}

// zone is the clinic's time zone; UTC when it cannot be loaded.
func zone(loc organization.Location) *time.Location {
	z, err := time.LoadLocation(loc.TimeZone)
	if err != nil {
		return time.UTC
	}
	return z
}

// interner numbers a build's pseudonyms: a patient is an int32 inside the build only.
type interner map[[32]byte]int32

func (in interner) of(p [32]byte) int32 {
	id, ok := in[p]
	if !ok {
		id = int32(len(in))
		in[p] = id
	}
	return id
}

// cells accumulates a build's contributions.
type cells map[Key]map[int32]map[int32]struct{}

func (c cells) add(k Key, clinic, patient int32) {
	if c[k] == nil {
		c[k] = map[int32]map[int32]struct{}{}
	}
	if c[k][clinic] == nil {
		c[k][clinic] = map[int32]struct{}{}
	}
	c[k][clinic][patient] = struct{}{}
}

// contributions are the accumulated cells as Suppress takes them.
func (c cells) contributions() map[Key]Contribution {
	out := make(map[Key]Contribution, len(c))
	for k, byClinic := range c {
		cc := make(Contribution, len(byClinic))
		for clinic, ps := range byClinic {
			list := make([]int32, 0, len(ps))
			for p := range ps {
				list = append(list, p)
			}
			cc[clinic] = merge(nil, sortInts(list))
		}
		out[k] = cc
	}
	return out
}

// save writes the period with its cells or flags, in one transaction; ErrFrozen when another build wrote it first.
func (b *Builder) save(ctx context.Context, p Period, rows []Row, flags []Flag) error {
	return b.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := b.p.Repo.SavePeriod(ctx, p); err != nil {
			return err
		}
		if len(rows) > 0 {
			if err := b.p.Repo.SaveCells(ctx, p.Period, rows); err != nil {
				return err
			}
		}
		if len(flags) > 0 {
			return b.p.Repo.SaveFlags(ctx, p.Period, flags)
		}
		return nil
	})
}

// audit records the build's read of its sources: rows of the source table, by the build's agent.
func (b *Builder) audit(ctx context.Context, table string, rows int) error {
	err := b.p.Audit.Record(ctx, auditevent.AuditEvent{
		Type:        auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:     auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeOperation},
		Purpose:     auditevent.Coding{System: auditevent.SystemActReason, Code: purposeStatistics},
		Action:      auditevent.ActionExecute,
		Outcome:     auditevent.OutcomeSuccess,
		AgentName:   b.p.Agent,
		EntityName:  table,
		EntityCount: int64(rows),
	})
	if err != nil {
		return errors.Join(errors.New("trends: audit"), err)
	}
	return nil
}
