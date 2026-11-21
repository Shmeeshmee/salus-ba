package chargeitemdefinition

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Service holds the price list's rules: the clinic's staff see its services, only its admin changes them, the public
// directory sees the active services of listed clinics, and a receipt's lines are priced from it. Package B builds
// each method; until then each refuses with errors.ErrUnsupported.
type Service struct {
	repo  Repository
	clock clock.Clock
}

// NewService returns a Service on its ports.
func NewService(repo Repository, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// List returns the clinic's services to its staff (GET /clinic/services).
func (s *Service) List(_ context.Context, _ Caller) ([]Definition, error) {
	return nil, errors.ErrUnsupported
}

// Create adds a service to the clinic's price list, by its admin (POST /clinic/services).
func (s *Service) Create(_ context.Context, _ Caller, _ Definition) (Definition, error) {
	return Definition{}, errors.ErrUnsupported
}

// Update changes the service d.ID at version d.VersionID, by the clinic's admin (PUT /clinic/services/{id}).
func (s *Service) Update(_ context.Context, _ Caller, _ Definition) (Definition, error) {
	return Definition{}, errors.ErrUnsupported
}

// Offered returns the active services of the listed clinic org offered in mode, with their names in language where
// translated: the prices shown at booking.
func (s *Service) Offered(_ context.Context, _ string, _ schedule.VisitMode, _ string) ([]Definition, error) {
	return nil, errors.ErrUnsupported
}

// Lines prices the picks of a receipt of the clinic org from its active services, in the caller's transaction:
// internal/invoice's Prices port.
func (s *Service) Lines(_ context.Context, _ string, _ []Pick) ([]Line, error) {
	return nil, errors.ErrUnsupported
}
