package appointment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// Services gives the active services a listed clinic offers in a visit mode (chargeitemdefinition.Service.Offered),
// asked in the directory before the booking's transaction, as the slot is.
type Services interface {
	Offered(ctx context.Context, org string, mode schedule.VisitMode, language string) ([]chargeitemdefinition.Definition, error)
	// OfClinic is the clinic org's whole price list, read in its own scope, listed or not: a staff booking's service
	// and the desk's service names (docs/clinic-cms F2).
	OfClinic(ctx context.Context, org string) ([]chargeitemdefinition.Definition, error)
}

// offers reports whether the slot b's clinic offers the service id in b's mode: a booking may name a service of the
// price list (docs/plans/clinic-cms.md, E), shown to the clinic and the first line of the visit's receipt. An empty id
// names none and is always allowed; nothing is charged.
func (s *Service) offers(ctx context.Context, b slot.Bookable, id string) (bool, error) {
	if id == "" {
		return true, nil
	}
	all, err := s.Services.Offered(ctx, b.OrganizationID, visitMode(b.Mode), "")
	if errors.Is(err, chargeitemdefinition.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return slices.ContainsFunc(all, func(d chargeitemdefinition.Definition) bool { return d.ID == id }), nil
}

// offersInClinic is offers for a staff booking: the service id of the clinic's own price list, active and offered in
// the slot b's mode, read in the clinic's scope whether or not it is listed.
func (s *Service) offersInClinic(ctx context.Context, b slot.Bookable, id string) (bool, error) {
	if id == "" {
		return true, nil
	}
	all, err := s.Services.OfClinic(ctx, b.OrganizationID)
	if err != nil {
		return false, err
	}
	mode := visitMode(b.Mode)
	return slices.ContainsFunc(all, func(d chargeitemdefinition.Definition) bool {
		return d.ID == id && d.Active && slices.Contains(d.Modes, mode)
	}), nil
}

// Visits gives the clinic org's appointments starting in [from, to), read in the clinic's scope in the transaction in
// ctx, for the online practice report (clinicstats.Service.Practice), which authorizes the admin and audits the read:
// not audited here. Opaque ids, modes and statuses only.
func (s *Service) Visits(ctx context.Context, org string, from, to time.Time) ([]Visit, error) {
	var out []Visit
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.Visits(ctx, org, from, to)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: visits: %w", err)
	}
	return out, nil
}
