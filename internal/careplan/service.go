package careplan

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Ports are what the Service consumes.
type Ports struct {
	Repo  Repository
	Clock clock.Clock
}

// Service is the treatment plans' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Write writes a new draft for a patient of the staff's clinic (TP1).
func (s *Service) Write(_ context.Context, _ Staff, _ Draft) (Plan, error) {
	return Plan{}, errors.ErrUnsupported
}

// Change changes a draft at its version: its title, quote and steps (TP1).
func (s *Service) Change(_ context.Context, _ Staff, _ string, _ Draft) (Plan, error) {
	return Plan{}, errors.ErrUnsupported
}

// Book books every step of a draft in one transaction, or none (TP1).
func (s *Service) Book(_ context.Context, _ Staff, _ string, _ Booking) (Plan, error) {
	return Plan{}, errors.ErrUnsupported
}

// Cancel revokes a plan at its version and cancels its steps' future visits (TP1).
func (s *Service) Cancel(_ context.Context, _ Staff, _ string, _ int) (Plan, error) {
	return Plan{}, errors.ErrUnsupported
}

// Read returns one plan of the staff's clinic, its steps in the order of their visits (TP1).
func (s *Service) Read(_ context.Context, _ Staff, _ string) (Plan, error) {
	return Plan{}, errors.ErrUnsupported
}

// ForPatient returns a patient's plans at the staff's clinic, newest first (TP1).
func (s *Service) ForPatient(_ context.Context, _ Staff, _ string) ([]Plan, error) {
	return nil, errors.ErrUnsupported
}

// Mine returns the patient's booked plans, newest first (TP1).
func (s *Service) Mine(_ context.Context, _ Patient) ([]Plan, error) {
	return nil, errors.ErrUnsupported
}
