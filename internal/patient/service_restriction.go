package patient

import (
	"context"
	"errors"
)

// Restrict restricts the patient's record at the admin's clinic (GDPR Art. 18; docs/clinic-cms/WAVE5.md S6, S7),
// optionally for the data request requestID: hidden from the clinic's everyday screens and searches until lifted.
// Package D4 builds it.
func (s *Service) Restrict(_ context.Context, _ Caller, _, _ string) error {
	return errors.ErrUnsupported
}

// LiftRestriction lifts the restriction of the patient's record at the admin's clinic (D4).
func (s *Service) LiftRestriction(_ context.Context, _ Caller, _ string) error {
	return errors.ErrUnsupported
}
