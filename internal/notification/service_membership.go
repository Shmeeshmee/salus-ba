package notification

import (
	"context"
	"fmt"
	"strconv"
)

// MembershipInvoice is Salus's monthly invoice to a clinic, sent to its billing e-mail with the PDF
// (docs/plans/clinic-cms.md, package I). Nothing in it is personal data.
type MembershipInvoice struct {
	To       string // the clinic's billing e-mail
	Language string // the clinic's default language
	Clinic   string
	PDF      []byte
	Number   int64
}

// SendMembershipInvoice sends in (subscription.Service.SendInvoice, from the subscription_invoice_email job). Errors
// carry no address.
func (s *Service) SendMembershipInvoice(ctx context.Context, in MembershipInvoice) error {
	number := strconv.FormatInt(in.Number, 10)
	msg, err := s.in(in.Language).message(in.To, "membership_invoice", letter{Clinic: in.Clinic, Number: number})
	if err != nil {
		return fmt.Errorf("notification: membership invoice: %w", err)
	}
	msg.Attachment = &Attachment{Name: "salus-invoice-" + number + ".pdf", ContentType: "application/pdf", Data: in.PDF}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: membership invoice: %w", err)
	}
	return nil
}
