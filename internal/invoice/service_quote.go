package invoice

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
)

// QuoteDraft is a quote as staff write it: lines picked from the clinic's price list.
type QuoteDraft struct {
	PatientID     string
	AppointmentID string
	Note          string
	Picks         []chargeitemdefinition.Pick
}

// SendQuote stores d as a quote and sends it to its patient. The wave 3 contract: refused until package X1.
func (s *Service) SendQuote(ctx context.Context, c Caller, d QuoteDraft) (Quote, error) {
	return Quote{}, errors.ErrUnsupported
}

// Quotes lists the staff caller's clinic's quotes, newest first. Refused until package X1.
func (s *Service) Quotes(ctx context.Context, c Caller) ([]Quote, error) {
	return nil, errors.ErrUnsupported
}

// ReadQuote returns the quote id of the staff caller's clinic. Refused until package X1.
func (s *Service) ReadQuote(ctx context.Context, c Caller, id string) (Quote, error) {
	return Quote{}, errors.ErrUnsupported
}

// MyQuotes lists the calling patient's quotes from every clinic, newest first. Refused until package X1.
func (s *Service) MyQuotes(ctx context.Context, c Caller) ([]Quote, error) {
	return nil, errors.ErrUnsupported
}

// QuotePDF renders the calling patient's quote id, which says it is not a receipt. Refused until package X1.
func (s *Service) QuotePDF(ctx context.Context, c Caller, id string) ([]byte, error) {
	return nil, errors.ErrUnsupported
}
