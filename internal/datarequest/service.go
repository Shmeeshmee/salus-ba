package datarequest

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Ports are what the Service consumes.
type Ports struct {
	Repo   Repository
	Clock  clock.Clock
	Export ExportPorts // the patient's export (D2, service_export.go)
}

// Service is the requests' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// File files the patient's request for its target (D1).
func (s *Service) File(_ context.Context, _ Patient, _ Filing) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// Mine returns the patient's requests, newest first (D1).
func (s *Service) Mine(_ context.Context, _ Patient) ([]Request, error) {
	return nil, errors.ErrUnsupported
}

// Withdraw withdraws the patient's open request id at version (D1).
func (s *Service) Withdraw(_ context.Context, _ Patient, _ string, _ int) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// FileOnBehalf files a request of kind for the patient the staff's clinic registered, targeting that clinic (S10, D1).
func (s *Service) FileOnBehalf(_ context.Context, _ Staff, _ string, _ Kind) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// ClinicRequests returns the requests targeting the admin's clinic, the open ones by due date first (D1).
func (s *Service) ClinicRequests(_ context.Context, _ Staff) ([]Request, error) {
	return nil, errors.ErrUnsupported
}

// ClinicRequest returns one request targeting the admin's clinic (D1).
func (s *Service) ClinicRequest(_ context.Context, _ Staff, _ string) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// Answer fulfils or refuses a request targeting the admin's clinic (D1).
func (s *Service) Answer(_ context.Context, _ Staff, _ string, _ Answer) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// Extend extends a request targeting the admin's clinic once, by ExtensionMonths, with its reason (D1).
func (s *Service) Extend(_ context.Context, _ Staff, _ string, _ ExtensionReason, _ int) (Request, error) {
	return Request{}, errors.ErrUnsupported
}

// CloseAccount closes the patient's account once the password confirms it (S5, D3).
func (s *Service) CloseAccount(_ context.Context, _ Patient, _ string) error {
	return errors.ErrUnsupported
}

// ReopenAccount reopens a closed account before its erasure (the operator's, from salusctl; D3).
func (s *Service) ReopenAccount(_ context.Context, _ string) error {
	return errors.ErrUnsupported
}
