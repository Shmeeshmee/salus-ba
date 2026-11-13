package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// Reminders gives the notifier its reminders (appointment.Service): the dispatch takes a scheduled one and marks it
// dispatched in its transaction, a send reads it again and issues its one-time link; each read audited with the job as
// agent.
type Reminders interface {
	TakeReminder(ctx context.Context, agent, org, id string) (appointment.ReminderNotice, error)
	ReminderDispatched(ctx context.Context, org, id string) error
	ReminderNotice(ctx context.Context, agent, org, id string) (appointment.ReminderNotice, error)
	IssueLink(ctx context.Context, agent string, l appointment.Link) (string, error)
}

// ClinicRules gives the clinic's appointment settings in the caller's transaction (organization.Appointments): whether
// it offers SMS. It sets the clinic's scope.
type ClinicRules interface {
	For(ctx context.Context, org string) (organization.AppointmentSettings, error)
}

// Languages gives the language to write to an account in ("" for none), audited with the job as agent (auth.Service).
type Languages interface {
	AccountLanguage(ctx context.Context, agent, id string) (string, error)
}

// Phones gives a patient's phone (E.164, "" for none) for an SMS they opted into, read in the clinic's scope and
// audited with the job as agent (patient.Service).
type Phones interface {
	NoticePhone(ctx context.Context, agent, org, id string) (string, error)
}

// quietLead is how long before its deadline a push or SMS held by quiet hours must still go out; later, it is dropped.
const quietLead = 30 * time.Minute

// The agents of the reminders' jobs: their kinds.
var (
	agentDispatch = appointment.ReminderArgs{}.Kind()
	agentSend     = map[Channel]string{
		ChannelEmail: EmailDeliveryArgs{}.Kind(), ChannelPush: PushDeliveryArgs{}.Kind(), ChannelSMS: SMSDeliveryArgs{}.Kind(),
	}
)

// DispatchReminder dispatches the reminder of a (the appointment_reminder job), in one transaction: the reminder is
// taken (locked, its appointment checked), its deliveries are stored with their send jobs (Deliver), and it is marked
// dispatched; a restart before the commit redoes all of it, after it none. A reminder no longer to be sent (dispatched
// already, cancelled, its appointment changed) does nothing and its skip is kept; one not of the clinic is ErrNotFound.
func (s *Service) DispatchReminder(ctx context.Context, a appointment.ReminderArgs) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		n, err := s.p.Reminders.TakeReminder(ctx, agentDispatch, a.OrganizationID, a.ReminderID)
		if errors.Is(err, appointment.ErrReminderGone) {
			return nil // committed: a skip is kept
		}
		if err != nil {
			return err
		}
		if err := s.Deliver(ctx, KindReminder, n.ReminderID, n.OrganizationID, n.AccountID, Content{Deadline: n.Start}); err != nil {
			return err
		}
		return s.p.Reminders.ReminderDispatched(ctx, n.OrganizationID, n.ReminderID)
	})
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.ErrNotFound
	case err != nil:
		return fmt.Errorf("notification: dispatch reminder: %w", err)
	}
	return nil
}

// Deliver stores the deliveries of the reminder or offer sourceID (kind) of the clinic org to the account, with their
// send jobs, in the caller's transaction (SAL-35 reuses it): e-mail when the patient keeps it on, now (never held);
// a push per device of the account and an SMS (when the patient opted in and the clinic offers SMS) now, or when the
// patient's quiet hours end if that is at least 30 minutes before c.Deadline, else none. Quiet hours are in the
// patient's zone, else the clinic's. A source delivered before on a channel (and device) gets nothing more there. The
// reads of the account's settings and devices are audited with the kind's job as agent.
func (s *Service) Deliver(ctx context.Context, kind Kind, sourceID, org, accountID string, c Content) error {
	agent := agentOf(kind)
	return s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		prefs, err := s.preferencesOf(ctx, agent, org, accountID)
		if err != nil {
			return err
		}
		rules, err := s.p.Rules.For(ctx, org)
		if err != nil {
			return err
		}
		now := s.p.Clock.Now()
		release, held, err := s.release(ctx, org, prefs, now, c.Deadline)
		if err != nil {
			return err
		}
		d := Delivery{Kind: kind, SourceID: sourceID, OrganizationID: org, AccountID: accountID}
		if prefs.Channels.Email {
			if err := s.addDelivery(ctx, d, ChannelEmail, "", now); err != nil {
				return err
			}
		}
		if held {
			return nil
		}
		if prefs.Channels.Push {
			if err := s.addPushes(ctx, agent, org, d, release); err != nil {
				return err
			}
		}
		if prefs.Channels.SMS && rules.SMSReminders {
			return s.addDelivery(ctx, d, ChannelSMS, "", release)
		}
		return nil
	})
}

// agentOf is the job agent of a kind's dispatch.
func agentOf(kind Kind) string {
	if kind == KindWaitlistOffer {
		return "waitlist_offer_notify"
	}
	return agentDispatch
}

// addPushes stores a push of d per device of its account, released at at.
func (s *Service) addPushes(ctx context.Context, agent, org string, d Delivery, at time.Time) error {
	devices, err := s.p.Devices.Devices(ctx, d.AccountID)
	if err != nil || len(devices) == 0 {
		return err
	}
	if err := s.p.Audit.Record(ctx, accountRead(agent, org, d.AccountID, "push_device", len(devices))); err != nil {
		return err
	}
	for _, dev := range devices {
		if err := s.addDelivery(ctx, d, ChannelPush, dev.ID, at); err != nil {
			return err
		}
	}
	return nil
}

// addDelivery stores d on channel (to device) and enqueues its send at at, unless it was stored before.
func (s *Service) addDelivery(ctx context.Context, d Delivery, channel Channel, device string, at time.Time) error {
	d.Channel, d.DeviceID = channel, device
	out, ok, err := s.p.Deliveries.AddDelivery(ctx, d)
	if err != nil || !ok {
		return err
	}
	return s.p.DeliveryJobs.EnqueueDelivery(ctx, channel, out.ID, at)
}

// release is when a push or SMS may go out: now outside the patient's quiet hours; when they end, inside them, if
// that is at least quietLead before deadline; held (none at all) otherwise.
func (s *Service) release(ctx context.Context, org string, p Preferences, now, deadline time.Time) (at time.Time, held bool, err error) {
	if p.Quiet == nil {
		return now, false, nil
	}
	loc, err := s.zoneOf(ctx, org, p)
	if err != nil {
		return time.Time{}, false, err
	}
	end, inside := quietEnd(now, *p.Quiet, loc)
	switch {
	case !inside:
		return now, false, nil
	case end.Add(quietLead).After(deadline):
		return time.Time{}, true, nil
	}
	return end, false, nil
}

// zoneOf is the patient's zone, else the clinic's.
func (s *Service) zoneOf(ctx context.Context, org string, p Preferences) (*time.Location, error) {
	if p.TimeZone != "" {
		if loc, err := time.LoadLocation(p.TimeZone); err == nil {
			return loc, nil
		}
	}
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
	if err != nil {
		return nil, err
	}
	return time.LoadLocation(clinic.TimeZone)
}

// quietEnd tells whether now is inside the quiet hours q in loc, and when they end then.
func quietEnd(now time.Time, q QuietHours, loc *time.Location) (time.Time, bool) {
	start, okS := clockMinutes(q.Start)
	end, okE := clockMinutes(q.End)
	if !okS || !okE || start == end {
		return time.Time{}, false
	}
	local := now.In(loc)
	t := local.Hour()*60 + local.Minute()
	inside := start <= t && t < end
	if start > end {
		inside = t >= start || t < end
	}
	if !inside {
		return time.Time{}, false
	}
	day := local
	if start > end && t >= start {
		day = local.AddDate(0, 0, 1) // the end is tomorrow
	}
	return time.Date(day.Year(), day.Month(), day.Day(), end/60, end%60, 0, 0, loc), true
}

// clockMinutes is HH:MM as minutes after midnight.
func clockMinutes(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// preferencesOf is the account's settings (the defaults when none are stored), the read audited with the job agent.
func (s *Service) preferencesOf(ctx context.Context, agent, org, accountID string) (Preferences, error) {
	p, ok, err := s.p.Deliveries.Preferences(ctx, accountID)
	if err != nil {
		return Preferences{}, err
	}
	if !ok {
		p = DefaultPreferences()
	}
	return p, s.p.Audit.Record(ctx, accountRead(agent, org, accountID, "notification_preference", 1))
}

// accountRead is the audit event of the job agent's read of n rows of the account's entity.
func accountRead(agent, org, account, entity string, n int) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          auditevent.ActionRead,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		OrganizationID:  org,
		EntityReference: "Account/" + account,
		EntityName:      entity,
		EntityCount:     int64(n),
	}
}

// errGone marks a delivery that no longer applies: it is cancelled, nothing sent.
var errGone = errors.New("notification: the delivery no longer applies")

// outgoing is one composed message of a delivery.
type outgoing struct {
	mail  Message
	push  push.Message
	sms   sms.Message
	parts int
	ready bool
}

// SendDelivery sends the delivery id (the reminder_email, reminder_push and reminder_sms jobs): locked and checked in a
// transaction, the reminder and its appointment read again and the message composed (with a new one-time link for an
// e-mail or SMS); sent after that commits, then marked sent, an SMS counted in its clinic's usage in the same
// transaction. A delivery no longer pending sends nothing; one whose reminder, appointment, device or phone is gone is
// cancelled; a refusal no retry can change (an address, number or token the provider will not take) fails it, as
// does the last attempt (last). Errors carry no address, number, token or text.
func (s *Service) SendDelivery(ctx context.Context, id string, last bool) error {
	var d Delivery
	var out outgoing
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if d, err = s.p.Deliveries.LockDelivery(ctx, id); err != nil {
			return err
		}
		if d.State != DeliveryPending {
			return ErrDeliveryDone
		}
		out, err = s.compose(ctx, d)
		if errors.Is(err, errGone) {
			return s.p.Deliveries.SetDeliveryState(ctx, id, DeliveryCancelled, s.p.Clock.Now(), 0)
		}
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrDeliveryDone):
		return nil
	case err != nil:
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	case !out.ready:
		return nil
	}
	if err := s.send(ctx, d, out); err != nil {
		if permanent(err) || last {
			if ferr := s.finish(ctx, d, DeliveryFailed, 0); ferr != nil {
				return errors.Join(err, ferr)
			}
			if permanent(err) {
				return nil
			}
		}
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	}
	if err := s.finish(ctx, d, DeliverySent, out.parts); err != nil {
		return fmt.Errorf("notification: send %s: %w", d.Channel, err)
	}
	return nil
}

// permanent tells whether err is a refusal no retry changes.
func permanent(err error) bool {
	return errors.Is(err, ErrRecipientNotASCII) || errors.Is(err, sms.ErrRejected) || errors.Is(err, sms.ErrInvalidMessage) ||
		errors.Is(err, push.ErrTokenGone)
}

// send hands out to its channel's sender; a push token the platform says is gone is forgotten.
func (s *Service) send(ctx context.Context, d Delivery, out outgoing) error {
	switch d.Channel {
	case ChannelEmail:
		return s.sender.Send(ctx, out.mail)
	case ChannelSMS:
		_, err := s.p.SMS.Send(ctx, out.sms)
		return err
	}
	err := s.p.Pusher.Send(ctx, out.push)
	if errors.Is(err, push.ErrTokenGone) {
		if ferr := s.p.Devices.DeleteDevice(ctx, d.DeviceID, out.push.Token); ferr != nil {
			return errors.Join(err, ferr)
		}
	}
	return err
}

// finish moves d from pending to state now, counting a sent SMS in its clinic's month, in one transaction.
func (s *Service) finish(ctx context.Context, d Delivery, state DeliveryState, parts int) error {
	return s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.p.Clock.Now()
		if err := s.p.Deliveries.SetDeliveryState(ctx, d.ID, state, now, parts); err != nil {
			return err
		}
		if state != DeliverySent || d.Channel != ChannelSMS {
			return nil
		}
		if err := s.p.Devices.SetClinic(ctx, d.OrganizationID); err != nil {
			return err
		}
		u := now.UTC()
		return s.p.Deliveries.AddSMSUsage(ctx, d.OrganizationID, time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC), parts)
	})
}

// compose writes d's message, or errGone when it no longer applies. Only reminders are composed here; a waitlist
// offer's composition comes with SAL-35.
func (s *Service) compose(ctx context.Context, d Delivery) (outgoing, error) {
	if d.Kind != KindReminder {
		return outgoing{}, fmt.Errorf("notification: no composer for %s", d.Kind)
	}
	agent := agentSend[d.Channel]
	n, err := s.p.Reminders.ReminderNotice(ctx, agent, d.OrganizationID, d.SourceID)
	if errors.Is(err, appointment.ErrReminderGone) || errors.Is(err, appointment.ErrNotFound) || err == nil && n.AccountID != d.AccountID {
		return outgoing{}, errGone
	}
	if err != nil {
		return outgoing{}, err
	}
	switch d.Channel {
	case ChannelPush:
		return s.composePush(ctx, agent, d)
	case ChannelEmail:
		return s.composeEmail(ctx, agent, d, n)
	}
	return s.composeSMS(ctx, agent, d, n)
}

// composePush is the generic reminder push to d's device, in its account's language: no clinic, time or payload.
func (s *Service) composePush(ctx context.Context, agent string, d Delivery) (outgoing, error) {
	dev, err := s.p.Devices.Device(ctx, d.DeviceID)
	switch {
	case errors.Is(err, ErrNotFound), err == nil && dev.AccountID != d.AccountID:
		return outgoing{}, errGone // unregistered, or the token moved to another account
	case err != nil:
		return outgoing{}, err
	}
	if err := s.p.Audit.Record(ctx, accountRead(agent, d.OrganizationID, d.AccountID, "push_device", 1)); err != nil {
		return outgoing{}, err
	}
	language, err := s.p.Languages.AccountLanguage(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	alert, err := s.in(language).message("", "reminder_push", letter{})
	if err != nil {
		return outgoing{}, err
	}
	return outgoing{push: push.Message{Token: dev.Token, Platform: dev.Platform, Title: alert.Subject, Body: strings.TrimSuffix(alert.Body, "\n")}, ready: true}, nil
}

// reminderTimes is the start in the clinic's zone with its name, and in the patient's zone when it differs ("" when
// not), written in l.
func (s *Service) reminderTimes(ctx context.Context, agent string, l letters, d Delivery, n appointment.ReminderNotice) (clinic organization.Organization, when, whenLocal string, local time.Time, err error) {
	clinic, err = s.clinics.Read(ctx, organization.Caller{OrganizationID: d.OrganizationID}, d.OrganizationID)
	if err != nil {
		return clinic, "", "", time.Time{}, err
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return clinic, "", "", time.Time{}, fmt.Errorf("the clinic's time zone: %w", err)
	}
	local = n.Start.In(loc)
	if when, err = l.when(local, clinic.TimeZone); err != nil {
		return clinic, "", "", time.Time{}, err
	}
	p, err := s.preferencesOf(ctx, agent, d.OrganizationID, d.AccountID)
	if err != nil || p.TimeZone == "" || p.TimeZone == clinic.TimeZone {
		return clinic, when, "", local, err
	}
	theirs, lerr := time.LoadLocation(p.TimeZone)
	// A zone that no longer loads, or the same wall time: one time only.
	if lerr != nil || n.Start.In(theirs).Format(time.DateTime) == local.Format(time.DateTime) {
		return clinic, when, "", local, nil
	}
	whenLocal, err = l.when(n.Start.In(theirs), p.TimeZone)
	return clinic, when, whenLocal, local, err
}

// composeEmail is the reminder e-mail: the clinic's name, the time (twice when the patient's zone differs), whether a
// questionnaire is outstanding, a one-time link to confirm or cancel, and the link to the app's appointments.
func (s *Service) composeEmail(ctx context.Context, agent string, d Delivery, n appointment.ReminderNotice) (outgoing, error) {
	to, language, err := s.accounts.AccountEmail(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	l := s.in(language)
	clinic, when, whenLocal, _, err := s.reminderTimes(ctx, agent, l, d, n)
	if err != nil {
		return outgoing{}, err
	}
	outstanding, err := s.questionnaires.Outstanding(ctx, agent, d.OrganizationID, n.AppointmentID)
	if err != nil {
		return outgoing{}, err
	}
	link, err := s.issue(ctx, agent, n, language, appointment.ChannelEmail)
	if err != nil {
		return outgoing{}, err
	}
	msg, err := l.message(to, "reminder", letter{
		Clinic: clinic.Name, When: when, WhenLocal: whenLocal, Link: s.patientAppURL + "/appointments", ActionLink: link, Questionnaire: outstanding,
	})
	return outgoing{mail: msg, ready: true}, err
}

// composeSMS is the reminder SMS to the patient's phone: the clinic's name, the date and time in its zone and the
// one-time link. No phone any more cancels it.
func (s *Service) composeSMS(ctx context.Context, agent string, d Delivery, n appointment.ReminderNotice) (outgoing, error) {
	phone, err := s.p.Phones.NoticePhone(ctx, agent, d.OrganizationID, n.PatientID)
	if err != nil || phone == "" {
		return outgoing{}, errors.Join(err, errGone)
	}
	language, err := s.p.Languages.AccountLanguage(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	l := s.in(language)
	clinic, _, _, local, err := s.reminderTimes(ctx, agent, l, d, n)
	if err != nil {
		return outgoing{}, err
	}
	link, err := s.issue(ctx, agent, n, language, appointment.ChannelSMS)
	if err != nil {
		return outgoing{}, err
	}
	body, err := l.text("reminder_sms", letter{
		Clinic: clinic.Name, ActionLink: link, Day: local.Format("02"), Month: local.Format("01"), Year: local.Year(), Time: local.Format("15:04"),
	})
	if err != nil {
		return outgoing{}, err
	}
	return outgoing{sms: sms.Message{To: phone, Body: body, Reference: d.ID}, parts: sms.Segments(body), ready: true}, nil
}

// issue issues n's one-time link for channel, in the language of the message; a reminder whose appointment closed
// meanwhile is gone.
func (s *Service) issue(ctx context.Context, agent string, n appointment.ReminderNotice, language string, channel appointment.Channel) (string, error) {
	link, err := s.p.Reminders.IssueLink(ctx, agent, appointment.Link{
		AppointmentID: n.AppointmentID, OrganizationID: n.OrganizationID, ReminderID: n.ReminderID, Language: s.writtenIn(language), Channel: channel,
	})
	if errors.Is(err, appointment.ErrClosed) || errors.Is(err, appointment.ErrNotFound) {
		return "", errGone
	}
	return link, err
}

// writtenIn is the language a message is written in: language when there are templates in it, else the fallback.
func (s *Service) writtenIn(language string) string {
	if _, ok := s.letters[language]; ok {
		return language
	}
	return fallbackLanguage
}
