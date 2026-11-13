package trends

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Caller is who asks: the signed-in account, its clinic, and whether it is staff of it (an admin or a practitioner).
type Caller struct {
	AccountID      string
	OrganizationID string
	Staff          bool
}

// Contributing tells whether a clinic contributes to the platform figures (organization.Service).
type Contributing interface {
	Contributes(ctx context.Context, organizationID string) (bool, error)
}

// Ports are what the Service reads.
type Ports struct {
	Repo    Repository
	Clinics Contributing
	Catalog Catalog
	Audit   Auditor
	Tx      Transactor
	// Specialties are the specialty codes a role may have (practitionerrole.Specialties): the keys of a breakdown by
	// specialty.
	Specialties []string
	K           int // ANALYTICS_MIN_COUNT, shown with the figures
}

// Service serves the published platform figures to the staff of a contributing clinic (O5, O6): only frozen
// published cells, every key of a breakdown with null for 0 and for suppressed alike (A8), sums over months marked
// partial (A9). Every read is audited, a refused one too.
type Service struct{ p Ports }

// NewService returns the Service on its ports.
func NewService(p Ports) *Service { return &Service{p: p} }

// Overview is what the view starts from: the published months and quarters, and k.
type Overview struct {
	Months   []string
	Quarters []string
	MinCount int
}

// Count is one key of a breakdown and its count, nil for 0 or suppressed alike.
type Count struct {
	Count *int
	Key   string
}

// Breakdown is a breakdown by one dimension. Partial: some count leaves out a month that is null for its key (A9).
type Breakdown struct {
	Rows    []Count
	Partial bool
}

// HeatCell is a heatmap cell (ISO weekday, 1 = Monday; hour) and its count, nil for 0 or suppressed.
type HeatCell struct {
	Count   *int
	Weekday int
	Hour    int
}

// Overview returns the published months and quarters.
func (s *Service) Overview(ctx context.Context, c Caller) (Overview, error) {
	var o Overview
	err := s.read(ctx, c, "trends_period", func(ctx context.Context) error {
		for _, kind := range []string{KindMonth, KindQuarter} {
			ps, err := s.p.Repo.Periods(ctx, kind)
			if err != nil {
				return err
			}
			for _, p := range ps {
				if p.Status != StatusPublished {
					continue
				}
				if kind == KindMonth {
					o.Months = append(o.Months, p.Period)
				} else {
					o.Quarters = append(o.Quarters, p.Period)
				}
			}
		}
		return nil
	})
	o.MinCount = s.p.K
	return o, err
}

// Rises returns the rise flags of the last weeks built (1 to 52), of one leaf region or all.
func (s *Service) Rises(ctx context.Context, c Caller, region string, weeks int) ([]Flag, error) {
	if weeks < 1 || weeks > 52 {
		return nil, ErrInvalidQuery
	}
	var out []Flag
	err := s.read(ctx, c, "trends_rise", func(ctx context.Context) error {
		ps, err := s.p.Repo.Periods(ctx, KindWeek)
		if err != nil {
			return err
		}
		names := make([]string, 0, weeks)
		for i := len(ps) - 1; i >= 0 && len(names) < weeks; i-- {
			names = append(names, ps[i].Period)
		}
		flags, err := s.p.Repo.Flags(ctx, names)
		for _, f := range flags {
			if region == "" || f.Region == region {
				out = append(out, f)
			}
		}
		return err
	})
	return out, err
}

// Heatmap returns a published quarter's 7 × 24 cells, of one region or entity or of all.
func (s *Service) Heatmap(ctx context.Context, c Caller, quarter, region string) ([]HeatCell, error) {
	if p, err := parsePeriod(quarter); err != nil || p.kind != KindQuarter {
		return nil, ErrInvalidQuery
	}
	var out []HeatCell
	err := s.read(ctx, c, "trends_cell", func(ctx context.Context) error {
		known, err := s.isRegion(ctx, region)
		if err != nil {
			return err
		}
		if !known {
			return ErrInvalidQuery
		}
		rows, err := s.p.Repo.Cells(ctx, TableHeatmap, quarter, quarter)
		if err != nil {
			return err
		}
		out = heatCells(rows, region)
		return nil
	})
	return out, err
}

// isRegion reports whether region is "" or a leaf region (or an entity) of the places.
func (s *Service) isRegion(ctx context.Context, region string) (bool, error) {
	if region == "" {
		return true, nil
	}
	places, err := s.p.Catalog.AllPlaces(ctx)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(places, func(p terminology.Place) bool { return p.ID == region && p.Kind != "city" }), nil
}

// heatCells are the 168 weekday × hour cells of rows at the region (its own, or as an entity), nil where none.
func heatCells(rows []Row, region string) []HeatCell {
	counts := map[[2]string]int{}
	for _, r := range rows {
		if r.Weekday == "" || r.Hour == "" || r.Suppressed {
			continue
		}
		atRegion := r.Region == region && region != ""
		atEntity := r.Region == "" && r.Entity == region
		if atRegion || atEntity {
			counts[[2]string{r.Weekday, r.Hour}] = r.Count
		}
	}
	out := make([]HeatCell, 0, 7*24)
	for d := 1; d <= 7; d++ {
		for h := range 24 {
			cell := HeatCell{Weekday: d, Hour: h}
			if v, ok := counts[[2]string{fmt.Sprint(d), fmt.Sprint(h)}]; ok {
				cell.Count = &v
			}
			out = append(out, cell)
		}
	}
	return out
}

// read checks the caller (staff of a contributing clinic), runs fn in a transaction and audits it there; a refusal
// is audited too, naming no entity.
func (s *Service) read(ctx context.Context, c Caller, entity string, fn func(ctx context.Context) error) error {
	refused := error(nil)
	if !c.Staff || c.OrganizationID == "" {
		refused = ErrForbidden
	} else if ok, err := s.p.Clinics.Contributes(ctx, c.OrganizationID); err != nil {
		return fmt.Errorf("trends: read: %w", err)
	} else if !ok {
		refused = ErrNotContributing
	}
	if refused != nil {
		if err := s.p.Audit.Record(ctx, readEvent(c, "", auditevent.OutcomeMinorFailure)); err != nil {
			return fmt.Errorf("trends: read: %w", err)
		}
		return refused
	}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, readEvent(c, entity, auditevent.OutcomeSuccess))
	})
	if errors.Is(err, ErrInvalidQuery) {
		return ErrInvalidQuery
	}
	if err != nil {
		return fmt.Errorf("trends: read: %w", err)
	}
	return nil
}

// readEvent is a read by c of the table entity ("" when nothing was read).
func readEvent(c Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeSearchType},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purposeStatistics},
		Action:         auditevent.ActionRead,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     entity,
	}
}
