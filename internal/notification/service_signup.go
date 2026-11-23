package notification

import (
	"context"
	"fmt"
)

// SendSignupVerification sends the verification e-mail of a clinic's signup to email (the signup_email job, which
// issues token with signup.Service.IssueVerification): the clinic's name and a link with the token, in the URL fragment
// so it never reaches a server log, in language. Errors carry no address or token.
func (s *Service) SendSignupVerification(ctx context.Context, email, clinic, language, token string) error {
	msg, err := s.in(language).message(email, "signup", letter{Clinic: clinic, Link: s.webBaseURL + "/signup/verify#" + token})
	if err != nil {
		return fmt.Errorf("notification: signup: %w", err)
	}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: signup: %w", err)
	}
	return nil
}
