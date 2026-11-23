package notification_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// The verification e-mail of a clinic's signup (docs/clinic-cms, O1): the clinic's name and the link, in the
// signup's language, the token in the fragment.
func TestSendSignupVerification_MailsTheLinkInTheSignupsLanguage(t *testing.T) {
	sender := &fakeSender{}
	svc := notification.NewService(notification.Ports{Sender: sender, WebBaseURL: "https://app.example.test/"})

	if err := svc.SendSignupVerification(t.Context(), "admin@example.test", "Synthetic clinic", "en", token); err != nil {
		t.Fatalf("SendSignupVerification() = %v", err)
	}
	want := notification.Message{
		To:      "admin@example.test",
		Subject: "Confirm your clinic's sign-up to Salus",
		Body: "You asked to sign Synthetic clinic up to Salus.\n\n" +
			"To confirm your e-mail address, open this link:\n" +
			"https://app.example.test/signup/verify#" + token + "\n\n" +
			"After you confirm, we will send you an invitation to set up the clinic's admin account. The link expires 72 hours after the sign-up. If you did not sign up, you can ignore this e-mail.\n",
	}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Errorf("sent %+v,\nwant %+v", sender.sent, want)
	}
}

func TestSendSignupVerification_FailsWithoutLeakingTheMessage(t *testing.T) {
	down := errors.New("down")
	err := notification.NewService(notification.Ports{Sender: &fakeSender{err: down}}).SendSignupVerification(t.Context(), "admin@example.test", "Synthetic clinic", "en", token)
	if !errors.Is(err, down) || strings.Contains(err.Error(), "example.test") || strings.Contains(err.Error(), token) {
		t.Errorf("SendSignupVerification() = %v, want its error without the address or the token", err)
	}
}

// The verification e-mail in every language, against its golden file.
func TestSendSignupVerification_InEveryLanguage(t *testing.T) {
	for _, lang := range languages {
		sender := &fakeSender{}
		svc := notification.NewService(notification.Ports{Sender: sender, WebBaseURL: "https://app.example.test"})
		if err := svc.SendSignupVerification(t.Context(), "admin@example.test", "Synthetic clinic", lang, token); err != nil || len(sender.sent) != 1 {
			t.Fatalf("%s: SendSignupVerification() = %v", lang, err)
		}
		golden(t, lang+"/signup.txt", mail(sender.sent[0]))
		if m := sender.sent[0]; strings.Contains(m.Subject+m.Body, "{{") || strings.Contains(m.Subject+m.Body, "<no value>") {
			t.Errorf("%s: the e-mail holds a template's leftover", lang)
		}
	}
}
