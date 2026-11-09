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

// Ports are what the Service runs on, and the bases of its links.
type Ports struct {
	Invites      Invites
	Appointments Appointments
	Accounts     Accounts
	Clinics      Clinics
	// Questionnaires is asked by the booking confirmation and the reminder only.
	Questionnaires Questionnaires
	Sender         Sender
	// WebBaseURL is WEB_BASE_URL: invite links point at <WebBaseURL>/invite#<token>.
	WebBaseURL string
	// PatientAppURL is PATIENT_APP_URL: appointment e-mails link to <PatientAppURL>/appointments.
	PatientAppURL string
}

// Service composes and sends the e-mails, each in its reader's language.
type Service struct {
	letters        map[string]letters
	invites        Invites
	appointments   Appointments
	accounts       Accounts
	clinics        Clinics
	questionnaires Questionnaires
	sender         Sender
	webBaseURL     string
	patientAppURL  string
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{
		letters: loadLetters(),
		invites: p.Invites, appointments: p.Appointments, accounts: p.Accounts, clinics: p.Clinics,
		questionnaires: p.Questionnaires, sender: p.Sender,
		webBaseURL: strings.TrimRight(p.WebBaseURL, "/"), patientAppURL: strings.TrimRight(p.PatientAppURL, "/"),
	}
}

// SendInvite sends the e-mail of the invite inviteID of the clinic org (the invite_email job): the clinic's
// name and a link with a new token, in the URL fragment so it never reaches a server log. Nothing else about
// the invite goes into it. It is written in the clinic's default language: the invitee has no account yet. The token is issued before the send, in its own transaction; a failed send is
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
	msg, err := s.in(clinic.DefaultLanguage).message(email, "invite", letter{Clinic: clinic.Name, Link: s.webBaseURL + "/invite#" + token})
	if err != nil {
		return fmt.Errorf("notification: invite: %w", err)
	}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: invite: %w", err)
	}
	return nil
}
