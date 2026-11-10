package notification_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
)

// SAL-17: the e-mail that a document is ready goes to the appointment's patient, in their language, with the
// clinic, the appointment's local start and the link to the app's appointments, whatever the appointment's status
// (a document is signed during or after the consultation); it names no document, practitioner or content. Its reads
// are audited with the job's kind as agent.
func TestSendDocumentSigned_TellsThePatientADocumentIsReadyAndNothingElse(t *testing.T) {
	for _, status := range []appointment.Status{appointment.StatusBooked, appointment.StatusFulfilled, appointment.StatusCancelled} {
		appts, accounts, sender := &fakeAppointments{status: status}, &fakeAccounts{language: "en"}, &fakeSender{}
		if err := newMail(appts, accounts, zonedClinics{}, sender).SendDocumentSigned(t.Context(), apptID, clinicID); err != nil {
			t.Fatalf("%s: SendDocumentSigned() = %v", status, err)
		}
		agent := composition.SignedEmailArgs{}.Kind()
		if !slices.Equal(appts.asked, []string{agent, clinicID, apptID}) || !slices.Equal(accounts.asked, []string{agent, accountID}) {
			t.Errorf("%s: asked %v and %v, want the job's kind as agent", status, appts.asked, accounts.asked)
		}
		if len(sender.sent) != 1 || sender.sent[0].To != "ana@example.test" {
			t.Fatalf("%s: sent %+v, want one e-mail to the patient", status, sender.sent)
		}
		m := sender.sent[0]
		golden(t, "signed.txt", mail(m))
		for _, never := range []string{"Placeholder", "report", "diagnosis", "Dr"} {
			if strings.Contains(m.Subject+m.Body, never) {
				t.Errorf("%s: the e-mail holds %q", status, never)
			}
		}
	}
}

func TestSendDocumentSigned_PassesNotFoundAndFailures(t *testing.T) {
	err := newMail(&fakeAppointments{err: appointment.ErrNotFound}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{}).
		SendDocumentSigned(t.Context(), apptID, clinicID)
	if !errors.Is(err, appointment.ErrNotFound) {
		t.Errorf("SendDocumentSigned(unknown) = %v, want ErrNotFound for the job to cancel", err)
	}
	down := errors.New("smtp down")
	err = newMail(&fakeAppointments{status: appointment.StatusFulfilled}, &fakeAccounts{language: "en"}, zonedClinics{}, &fakeSender{err: down}).
		SendDocumentSigned(t.Context(), apptID, clinicID)
	if !errors.Is(err, down) || strings.Contains(err.Error(), "ana@example.test") {
		t.Errorf("SendDocumentSigned(send fails) = %v, want the failure without the address", err)
	}
}
