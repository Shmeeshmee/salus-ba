package notification_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/invoice"
)

// clinic-cms X1: the e-mail that a treatment quote is in the app goes to the visit's patient, in their language, with
// the clinic, the visit's local start and the link to the app's quotes, whatever the visit's status; it names no
// service, amount or practitioner. Its reads are audited with the job's kind as agent.
func TestSendQuote_TellsThePatientAQuoteIsInTheAppAndNothingElse(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusBooked, appointment.StatusFulfilled} {
		appts, accounts, sender := &fakeAppointments{status: status}, &fakeAccounts{language: "en"}, &fakeSender{}
		if err := newMail(appts, accounts, zonedClinics{}, sender).SendQuote(t.Context(), apptID, clinicID); err != nil {
			t.Fatalf("%s: SendQuote() = %v", status, err)
		}
		agent := invoice.QuoteEmailArgs{}.Kind()
		if !slices.Equal(appts.asked, []string{agent, clinicID, apptID}) || !slices.Equal(accounts.asked, []string{agent, accountID}) {
			t.Errorf("%s: asked %v and %v, want the job's kind as agent", status, appts.asked, accounts.asked)
		}
		if len(sender.sent) != 1 || sender.sent[0].To != "ana@example.test" {
			t.Fatalf("%s: sent %+v, want one e-mail to the patient", status, sender.sent)
		}
		m := sender.sent[0]
		golden(t, "quote.txt", mail(m))
		if !strings.Contains(m.Body, "/quotes") {
			t.Errorf("%s: the e-mail lacks the link to the app's quotes:\n%s", status, m.Body)
		}
		for _, never := range []string{"Placeholder", "BAM", "Dr"} {
			if strings.Contains(m.Subject+m.Body, never) {
				t.Errorf("%s: the e-mail holds %q", status, never)
			}
		}
	}
	err := newMail(&fakeAppointments{err: appointment.ErrNotFound}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{}).
		SendQuote(t.Context(), apptID, clinicID)
	if !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("SendQuote(unknown) = %v, want ErrNotFound for the job to cancel", err)
	}
}
