package clinicstats

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// PracticeClinics reads the clinic, for its time zone (organization.Service).
type PracticeClinics interface {
	Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error)
}

// PracticeSchedules gives the clinic's schedules, for the hours offered (schedule.Service).
type PracticeSchedules interface {
	OfClinic(ctx context.Context, org string) ([]schedule.Schedule, error)
}

// PracticeVisits gives the clinic's appointments starting in [from, to) (appointment.Service).
type PracticeVisits interface {
	Visits(ctx context.Context, org string, from, to time.Time) ([]appointment.Visit, error)
}

// PracticeReceipts gives the clinic's receipts issued in a period, to its admin, audited (invoice.Service).
type PracticeReceipts interface {
	List(ctx context.Context, c invoice.Caller, p invoice.Period) ([]invoice.Invoice, error)
}

// practiceKey is one row of the report.
type practiceKey struct {
	role string
	mode schedule.VisitMode
}

// Practice is the online practice report (GET /clinic/reports/practice, docs/plans/clinic-cms.md E): per practitioner
// role and visit mode in [q.From, q.To), the minutes offered by the role's availability rules as they are now, the
// visits held (fulfilled appointments starting in the period) and the receipted amount of those visits (receipts issued
// for them from the period's start until now, a reversed receipt and its reversal left out), in BAM minor units. For the
// clinic's admin only: it is revenue per practitioner. ErrInvalidQuery for a bad period (not audited); ErrForbidden,
// audited, to anyone else. The read is audited, and the receipts' read is audited by invoice as well.
func (s *Service) Practice(ctx context.Context, c Caller, q PracticeQuery) ([]PracticeRow, error) {
	if q.From.IsZero() || !q.To.After(q.From) || q.To.Sub(q.From) >= maxRange*24*time.Hour {
		return nil, ErrInvalidQuery
	}
	if c.OrganizationID == "" || !c.Admin {
		return nil, s.refuse(ctx, c)
	}
	rows, err := s.practice(ctx, c, q)
	if err != nil {
		return nil, fmt.Errorf("clinicstats: practice: %w", err)
	}
	return rows, nil
}

func (s *Service) practice(ctx context.Context, c Caller, q PracticeQuery) ([]PracticeRow, error) {
	clinic, err := s.p.Clinics.Read(ctx, organization.Caller{OrganizationID: c.OrganizationID}, c.OrganizationID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return nil, err
	}
	schedules, err := s.p.Schedules.OfClinic(ctx, c.OrganizationID)
	if err != nil {
		return nil, err
	}
	sums := map[practiceKey]*PracticeRow{}
	row := func(k practiceKey) *PracticeRow {
		if sums[k] == nil {
			sums[k] = &PracticeRow{PractitionerRoleID: k.role, Mode: k.mode}
		}
		return sums[k]
	}
	for _, sc := range schedules {
		for _, r := range sc.Rules {
			if m := offered(r, loc, q.From, q.To); m > 0 {
				row(practiceKey{sc.PractitionerRoleID, mode(r.Mode)}).MinutesOffered += m
			}
		}
	}
	held := map[string]practiceKey{}
	err = s.read(ctx, c, func(ctx context.Context) error {
		visits, err := s.p.Visits.Visits(ctx, c.OrganizationID, q.From, q.To)
		for _, v := range visits {
			if v.Status == appointment.StatusFulfilled {
				k := practiceKey{v.PractitionerRoleID, mode(v.Mode)}
				held[v.ID] = k
				row(k).Visits++
			}
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := s.receipted(ctx, c, q, held, row); err != nil {
		return nil, err
	}
	out := make([]PracticeRow, 0, len(sums))
	for _, r := range sums {
		out = append(out, *r)
	}
	slices.SortFunc(out, func(a, b PracticeRow) int {
		return cmp.Or(cmp.Compare(a.PractitionerRoleID, b.PractitionerRoleID), cmp.Compare(a.Mode, b.Mode))
	})
	return out, nil
}

// receipted adds the receipts of the visits held to their rows: those issued from the period's start until now, none
// for a period that has not begun. A reversed receipt and its reversal cancel out and are left out.
func (s *Service) receipted(ctx context.Context, c Caller, q PracticeQuery, held map[string]practiceKey, row func(practiceKey) *PracticeRow) error {
	now := s.p.Clock.Now()
	if len(held) == 0 || !now.After(q.From) {
		return nil
	}
	receipts, err := s.p.Receipts.List(ctx, invoice.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin},
		invoice.Period{From: q.From, To: now})
	if err != nil {
		return err
	}
	for _, r := range receipts {
		if k, ok := held[r.AppointmentID]; ok && r.ReversesID == "" && r.ReversedByID == "" {
			row(k).ReceiptedMinor += r.TotalMinor
		}
	}
	return nil
}

// offered is the minutes of the weekly rule r, a wall-clock window in loc, that fall in [from, to).
func offered(r schedule.Rule, loc *time.Location, from, to time.Time) int64 {
	start, okStart := schedule.Minutes(r.Start)
	end, okEnd := schedule.Minutes(r.End)
	if !okStart || !okEnd {
		return 0
	}
	var total time.Duration
	first := from.In(loc)
	for d := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc); d.Before(to); d = d.AddDate(0, 0, 1) {
		if (int(d.Weekday())+6)%7+1 != r.Day { // ISO: Monday 1 to Sunday 7
			continue
		}
		ws := time.Date(d.Year(), d.Month(), d.Day(), 0, start, 0, 0, loc)
		we := time.Date(d.Year(), d.Month(), d.Day(), 0, end, 0, 0, loc)
		if ws.Before(from) {
			ws = from
		}
		if we.After(to) {
			we = to
		}
		if we.After(ws) {
			total += we.Sub(ws)
		}
	}
	return int64(total / time.Minute)
}

// mode is m, or video for a rule or appointment from before the modes.
func mode(m schedule.VisitMode) schedule.VisitMode {
	if m == "" {
		return schedule.ModeVideo
	}
	return m
}
