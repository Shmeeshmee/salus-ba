package location

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction that travels in the context (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Ports are what the Service consumes.
type Ports struct {
	Repo  Repository
	Tx    Transactor
	Audit Auditor
	Clock clock.Clock
}

// Service is the locations' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Locations returns the staff's clinic's locations (LC1).
func (s *Service) Locations(_ context.Context, _ Staff) ([]Location, error) {
	return nil, errors.ErrUnsupported
}

// SaveLocation adds a location to the admin's clinic, or changes one at its version (LC1).
func (s *Service) SaveLocation(_ context.Context, _ Staff, _ Location) (Location, error) {
	return Location{}, errors.ErrUnsupported
}

// PublicLocations returns the active locations of a listed clinic, for the patient booking in person (LC1).
func (s *Service) PublicLocations(_ context.Context, _ string) ([]Location, error) {
	return nil, errors.ErrUnsupported
}
