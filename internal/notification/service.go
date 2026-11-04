package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// Invites issues the token of an invite's link and gives the invite's e-mail (auth.Service).
type Invites interface {
	IssueInviteToken(ctx context.Context, inviteID, org string) (email, token string, err error)
}

// Clinics reads a clinic (organization.Service).
type Clinics interface {
	Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error)
}

// Service composes and sends the e-mails.
type Service struct {
	invites    Invites
	clinics    Clinics
	sender     Sender
	webBaseURL string // WEB_BASE_URL: links point at <webBaseURL>/invite#<token>
}

// NewService returns a Service on its ports.
func NewService(invites Invites, clinics Clinics, sender Sender, webBaseURL string) *Service {
	return &Service{invites: invites, clinics: clinics, sender: sender, webBaseURL: strings.TrimRight(webBaseURL, "/")}
}

// SendInvite sends the e-mail of the invite inviteID of the clinic org (the invite_email job): the clinic's
// name and a link with a new token, in the URL fragment so it never reaches a server log. Nothing else about
// the invite goes into it. The token is issued before the send, in its own transaction; a failed send is
// retried by the job, which issues a new one. Errors carry no address or token.
func (s *Service) SendInvite(ctx context.Context, inviteID, org string) error {
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return fmt.Errorf("notification: invite: %w", err)
	}
	email, token, err := s.invites.IssueInviteToken(ctx, inviteID, org)
	if err != nil {
		return fmt.Errorf("notification: invite: %w", err)
	}
	if err := s.sender.Send(ctx, inviteMessage(email, clinic.Name, s.webBaseURL+"/invite#"+token)); err != nil {
		return fmt.Errorf("notification: invite: %w", err)
	}
	return nil
}

// inviteMessage is the invite e-mail: the clinic's name and the link, and fixed text.
func inviteMessage(to, clinic, link string) Message {
	return Message{
		To:      to,
		Subject: "Your invitation to Salus",
		Body: "You are invited to join " + clinic + " on Salus.\n\n" +
			"To accept, open this link, choose a password and set up a sign-in code:\n" +
			link + "\n\n" +
			"The link works once and expires 72 hours after the invitation was made. If you did not expect this e-mail, you can ignore it.\n",
	}
}
