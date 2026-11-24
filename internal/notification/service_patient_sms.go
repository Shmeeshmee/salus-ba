package notification

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// SMS reminders for patients without an account (docs/clinic-cms/WAVE4.md R3, decision V3): a profile the clinic
// registered has no account, so no settings, devices or e-mail. Its reminder goes by SMS only, to the profile's phone,
// while the clinic offers SMS; without the option or a phone, none.

// deliverPatientSMS stores the SMS delivery of the reminder n to its patient without an account, with its send job, in
// the caller's transaction: none unless the clinic offers SMS; released now, or when the default quiet hours (22:00 to
// 08:00, in the clinic's zone) end if that is at least 30 minutes before the start, else none. Nothing of the
// patient is read here: the phone is read only for the send.
func (s *Service) deliverPatientSMS(ctx context.Context, n appointment.ReminderNotice) error {
	rules, err := s.rulesKeepingScope(ctx, n.OrganizationID)
	if err != nil || !rules.SMSReminders {
		return err
	}
	release, held, err := s.release(ctx, n.OrganizationID, DefaultPreferences(), s.p.Clock.Now(), n.Start)
	if err != nil || held {
		return err
	}
	d := Delivery{Kind: KindReminder, SourceID: n.ReminderID, OrganizationID: n.OrganizationID, PatientID: n.PatientID}
	return s.addDelivery(ctx, d, ChannelSMS, "", release)
}

// composePatientSMS is the reminder SMS of d to its patient without an account: the clinic's name and the time in its
// zone, in the clinic's default language, without a one-time link (a link acts as the patient's account, which they do
// not have). errGone when the reminder no longer applies, the patient has an account now, the clinic turned SMS off,
// or the profile has no phone; the phone is read here only, audited with the send's job as agent.
func (s *Service) composePatientSMS(ctx context.Context, d Delivery) (outgoing, error) {
	agent := agentSend[ChannelSMS]
	n, err := s.p.Reminders.ReminderNotice(ctx, agent, d.OrganizationID, d.SourceID)
	switch {
	case errors.Is(err, appointment.ErrReminderGone), errors.Is(err, appointment.ErrNotFound),
		err == nil && (n.PatientID != d.PatientID || n.AccountID != ""):
		return outgoing{}, errGone
	case err != nil:
		return outgoing{}, err
	}
	rules, err := s.p.Rules.For(ctx, d.OrganizationID) // the send's scope is the clinic's already
	if err != nil || !rules.SMSReminders {
		return outgoing{}, errors.Join(err, errGone)
	}
	phone, err := s.p.Phones.NoticePhone(ctx, agent, d.OrganizationID, d.PatientID)
	if err != nil || phone == "" {
		return outgoing{}, errors.Join(err, errGone)
	}
	st, err := s.startTimes(ctx, d, n, Preferences{})
	if err != nil {
		return outgoing{}, err
	}
	body, err := s.in(st.clinic.DefaultLanguage).text("reminder_sms", reminderSMS(st, n, ""))
	if err != nil {
		return outgoing{}, err
	}
	return outgoing{sms: sms.Message{To: phone, Body: body, Reference: d.ID}, parts: sms.Segments(body), ready: true}, nil
}
