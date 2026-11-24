package appointment

import (
	"context"
	"errors"
	"time"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// Reminders of patients without an account (docs/clinic-cms/WAVE4.md R3, decision V3): a profile the clinic
// registered is reminded by SMS only, to its phone, while the clinic offers SMS; the notifier decides at each
// dispatch and send (notification.deliverPatientSMS).

// bookedAtDesk enqueues what follows the desk's booking of a at now for a patient without an account, in its
// transaction: no confirmation and no questionnaire request (they have no e-mail or app), the refresh of the clinic's
// statistics and the visit's reminders.
func (s *Service) bookedAtDesk(ctx context.Context, a Appointment, now time.Time) error {
	if err := s.Stats.EnqueueStatsRefresh(ctx, a.OrganizationID, a.Start); err != nil {
		return err
	}
	return s.scheduleReminders(ctx, a, now)
}

// withReminders marks each desk visit whose patient a reminder will reach: one with an account always (by the
// channels they chose); one without only while the clinic org offers SMS and the profile has a phone. The clinic's
// settings are read in the transaction in ctx (its scope is set).
func (s *Service) withReminders(ctx context.Context, org string, visits []DeskVisit) error {
	if len(visits) == 0 {
		return nil
	}
	if s.Rules == nil {
		return errors.New("appointment: desk reminders: no clinic rules")
	}
	rules, err := s.Rules.For(ctx, org)
	if err != nil {
		return err
	}
	for i := range visits {
		visits[i].ReminderOff = reminderOff(visits[i].Patient, rules.SMSReminders)
		visits[i].Reminder = visits[i].ReminderOff == ""
	}
	return nil
}

// Why no reminder will go to a patient without an account (DeskVisit.ReminderOff, the desk row's reason).
const (
	ReminderOffSMS     = "sms_off"  // the clinic does not offer SMS
	ReminderOffNoPhone = "no_phone" // the profile has no phone
)

// reminderOff is why no reminder reaches p at a clinic that does (sms) or does not offer SMS, or empty when one does:
// a patient with an account always gets one by the channels they chose.
func reminderOff(p patient.Patient, sms bool) string {
	switch {
	case p.AccountID != "":
		return ""
	case !sms:
		return ReminderOffSMS
	case p.Phone == "":
		return ReminderOffNoPhone
	}
	return ""
}
