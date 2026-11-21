package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Service holds the membership's rules: the clinic's admin sees its membership and invoices, the scheduler issues one
// invoice per clinic and month, the owner marks invoices paid and moves statuses in salusctl, each move audited.
// Package D builds each method; until then each refuses with errors.ErrUnsupported.
type Service struct {
	repo  Repository
	clock clock.Clock
}

// NewService returns a Service on its ports.
func NewService(repo Repository, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// Current returns the clinic's membership to its admin (GET /clinic/subscription).
func (s *Service) Current(_ context.Context, _ Caller) (Subscription, error) {
	return Subscription{}, errors.ErrUnsupported
}

// Invoices returns the clinic's monthly invoices to its admin, newest first (GET /clinic/subscription/invoices).
func (s *Service) Invoices(_ context.Context, _ Caller) ([]Invoice, error) {
	return nil, errors.ErrUnsupported
}

// StartTrial gives the new clinic org its 30-day trial (the operator's clinic creation).
func (s *Service) StartTrial(_ context.Context, _ string) (Subscription, error) {
	return Subscription{}, errors.ErrUnsupported
}

// IssueMonthly issues the clinic org's invoice for the month that starts at month, once: the subscription_monthly job.
func (s *Service) IssueMonthly(_ context.Context, _ string, _ time.Time) (Invoice, error) {
	return Invoice{}, errors.ErrUnsupported
}

// MarkPaid marks the invoice number paid at at (salusctl).
func (s *Service) MarkPaid(_ context.Context, _ int64, _ time.Time) (Invoice, error) {
	return Invoice{}, errors.ErrUnsupported
}
