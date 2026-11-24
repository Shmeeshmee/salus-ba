package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

// Admins gives the e-mail and language of each admin of a clinic, audited with the job as agent (auth.Service).
type Admins interface {
	ClinicAdmins(ctx context.Context, agent, org string) ([]auth.Contact, error)
}

// agentDataRequestDeadline names the warning's job in the audit events of the reads it makes.
const agentDataRequestDeadline = "data_request_deadline_warning"

// DataRequestDeadline is a data request's deadline warning (S3): the request's opaque id, its clinic ("" for the
// account target), its kind's code, its deadline and its answerer's time zone, and the days left.
type DataRequestDeadline struct {
	Deadline       time.Time
	Zone           *time.Location
	RequestID      string
	OrganizationID string
	Kind           string
	Days           int
}

// SendDataRequestDeadline warns the request's answerers of its deadline (docs/clinic-cms/WAVE5.md D5): each admin of
// the clinic for a clinic target, in their own language, with the link to the request; the operator for the account
// target, in the operator's language, pointing at salusctl (ErrNoOperator without an operator address). It names the
// kind, the deadline and the days left, never the patient. It returns how many it sent: none for a clinic without an
// admin. An address SMTP cannot carry is skipped and its ErrRecipientNotASCII returned after the others are sent; a
// retry after another failed send warns the earlier admins again. Errors carry no address.
func (s *Service) SendDataRequestDeadline(ctx context.Context, w DataRequestDeadline) (int, error) {
	to, link, err := s.deadlineRecipients(ctx, w)
	if err != nil {
		return 0, fmt.Errorf("notification: data request deadline: %w", err)
	}
	sent, skipped := 0, error(nil)
	for _, c := range to {
		l := s.in(c.Language)
		data := letter{Request: w.RequestID, Kind: w.Kind, Days: w.Days, Link: link}
		if data.When, err = l.when(w.Deadline.In(w.Zone), w.Zone.String()); err != nil {
			return sent, fmt.Errorf("notification: data request deadline: %w", err)
		}
		msg, err := l.message(c.Email, "data_request_deadline", data)
		if err != nil {
			return sent, fmt.Errorf("notification: data request deadline: %w", err)
		}
		err = s.sender.Send(ctx, msg)
		switch {
		case errors.Is(err, ErrRecipientNotASCII):
			skipped = ErrRecipientNotASCII
			continue
		case err != nil:
			return sent, fmt.Errorf("notification: data request deadline: %w", err)
		}
		sent++
	}
	return sent, skipped
}

// deadlineRecipients are who answers w's request and the link they follow ("" for the operator, who uses salusctl).
func (s *Service) deadlineRecipients(ctx context.Context, w DataRequestDeadline) ([]auth.Contact, string, error) {
	if w.OrganizationID == "" {
		if s.p.OperatorEmail == "" {
			return nil, "", ErrNoOperator
		}
		return []auth.Contact{{Email: s.p.OperatorEmail, Language: s.p.OperatorLanguage}}, "", nil
	}
	admins, err := s.p.Admins.ClinicAdmins(ctx, agentDataRequestDeadline, w.OrganizationID)
	return admins, s.webBaseURL + "/data-requests/" + w.RequestID, err
}
