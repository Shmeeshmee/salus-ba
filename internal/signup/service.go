package signup

import (
	"context"
	"errors"
)

// Service is the signup's business rules. The wave 3 contract: every method refuses until package O1.
type Service struct {
	repo Repository
}

// NewService builds a Service on repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Start records r and sends the verification e-mail. It answers the same whether or not the e-mail is known, so it
// enumerates nothing. Refused until package O1.
func (s *Service) Start(ctx context.Context, r Request) error {
	return errors.ErrUnsupported
}

// Verify checks token, creates the clinic unlisted, invites its admin and starts the trial. Refused until package O1.
func (s *Service) Verify(ctx context.Context, token string) (Signup, error) {
	return Signup{}, errors.ErrUnsupported
}

// Setup returns the setup wizard's progress of the caller's clinic. Refused until package O1.
func (s *Service) Setup(ctx context.Context, c Caller) (Setup, error) {
	return Setup{}, errors.ErrUnsupported
}

// SaveSetup stores the wizard's progress of the admin caller's clinic. Refused until package O1.
func (s *Service) SaveSetup(ctx context.Context, c Caller, in Setup) (Setup, error) {
	return Setup{}, errors.ErrUnsupported
}
