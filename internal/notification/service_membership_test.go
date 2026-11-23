package notification_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// The membership invoice's e-mail (docs/plans/clinic-cms.md, package I) names the clinic and the invoice's number,
// in the clinic's language, and carries the PDF; a failed send is returned.
func TestSendMembershipInvoice_MailsThePDFInTheClinicsLanguage(t *testing.T) {
	sender := &fakeSender{}
	svc := notification.NewService(notification.Ports{Sender: sender})
	pdf := []byte("%PDF-synthetic")
	in := notification.MembershipInvoice{To: "billing@example.test", Language: "en", Clinic: "Synthetic clinic", Number: 7, PDF: pdf}

	if err := svc.SendMembershipInvoice(t.Context(), in); err != nil {
		t.Fatalf("SendMembershipInvoice() error = %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d, want 1", len(sender.sent))
	}
	m := sender.sent[0]
	if m.To != in.To || m.Subject != "Salus membership invoice no. 7" || !strings.Contains(m.Body, "Synthetic clinic") {
		t.Errorf("sent %+v", m)
	}
	if m.Attachment == nil || m.Attachment.Name != "salus-invoice-7.pdf" || m.Attachment.ContentType != "application/pdf" || string(m.Attachment.Data) != string(pdf) {
		t.Errorf("attachment %+v, want salus-invoice-7.pdf", m.Attachment)
	}

	in.Language = "sr-Cyrl"
	if err := svc.SendMembershipInvoice(t.Context(), in); err != nil || !strings.Contains(sender.sent[1].Subject, "7") || sender.sent[1].Subject == sender.sent[0].Subject {
		t.Errorf("in sr-Cyrl: %v, subject %q", err, sender.sent[1].Subject)
	}

	sender.err = errors.New("down")
	if err := svc.SendMembershipInvoice(t.Context(), in); err == nil {
		t.Error("a failed send was not returned")
	}
}
