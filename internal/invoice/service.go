package invoice

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// Prices prices a receipt's picks from the clinic's price list, in the caller's transaction
// (chargeitemdefinition.Service; package C builds against a fake until package B merges).
type Prices interface {
	Lines(ctx context.Context, org string, picks []chargeitemdefinition.Pick) ([]chargeitemdefinition.Line, error)
}

// Draft is a receipt to issue: to a patient, a company or neither, for a visit or on its own, its picks from the
// price list and how it was paid.
type Draft struct {
	PatientID     string
	CompanyName   string
	CompanyTaxID  string
	AppointmentID string
	Kind          Kind
	PaymentMethod PaymentMethod
	Picks         []chargeitemdefinition.Pick
}

// Service holds the receipts' rules: the clinic's admin and the practitioner of the visit issue and reverse receipts
// (the reception role waits for Q1), the clinic's staff see its receipts, a patient sees their own, every access is
// audited. Package C builds each method; until then each refuses with errors.ErrUnsupported.
type Service struct {
	repo     Repository
	prices   Prices
	reporter fiscal.Reporter
	clock    clock.Clock
}

// NewService returns a Service on its ports.
func NewService(repo Repository, prices Prices, reporter fiscal.Reporter, clk clock.Clock) *Service {
	return &Service{repo: repo, prices: prices, reporter: reporter, clock: clk}
}

// Issue numbers, stores and enqueues the fiscal report of a receipt (POST /clinic/receipts).
func (s *Service) Issue(_ context.Context, _ Caller, _ Draft) (Invoice, error) {
	return Invoice{}, errors.ErrUnsupported
}

// Read returns a receipt of the caller's clinic (GET /clinic/receipts/{id}).
func (s *Service) Read(_ context.Context, _ Caller, _ string) (Invoice, error) {
	return Invoice{}, errors.ErrUnsupported
}

// List returns the clinic's receipts issued in the period, newest first (GET /clinic/receipts).
func (s *Service) List(_ context.Context, _ Caller, _ Period) ([]Invoice, error) {
	return nil, errors.ErrUnsupported
}

// Reverse issues the receipt that reverses the receipt id, once (POST /clinic/receipts/{id}/reverse).
func (s *Service) Reverse(_ context.Context, _ Caller, _ string, _ PaymentMethod) (Invoice, error) {
	return Invoice{}, errors.ErrUnsupported
}

// PDF renders a receipt on the clinic's letterhead (GET /clinic/receipts/{id}.pdf).
func (s *Service) PDF(_ context.Context, _ Caller, _ string) ([]byte, error) {
	return nil, errors.ErrUnsupported
}

// Mine returns a patient's own receipts, newest first (GET /me/receipts).
func (s *Service) Mine(_ context.Context, _ Caller) ([]Invoice, error) {
	return nil, errors.ErrUnsupported
}

// Fiscalize reports the receipt id of the clinic org through the fiscal port: the invoice_fiscalize job.
func (s *Service) Fiscalize(_ context.Context, _, _ string) error {
	return errors.ErrUnsupported
}
