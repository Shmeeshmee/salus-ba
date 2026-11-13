package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/sms"
)

// compose writes d's message, or errGone when it no longer applies: its reminder or appointment changed, or its
// channel is off now (the patient turned it off, or the clinic its SMS option, while the send waited for the quiet
// hours to end). Only reminders are composed here: any other kind is errUnsupported (a waitlist offer's composer
// comes with SAL-35).
func (s *Service) compose(ctx context.Context, d Delivery) (outgoing, error) {
	if d.Kind != KindReminder {
		return outgoing{}, errUnsupported
	}
	agent := agentSend[d.Channel]
	n, err := s.p.Reminders.ReminderNotice(ctx, agent, d.OrganizationID, d.SourceID)
	if errors.Is(err, appointment.ErrReminderGone) || errors.Is(err, appointment.ErrNotFound) || err == nil && n.AccountID != d.AccountID {
		return outgoing{}, errGone
	}
	if err != nil {
		return outgoing{}, err
	}
	prefs, err := s.preferencesOf(ctx, agent, d.OrganizationID, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	if on, err := s.stillOn(ctx, d, prefs); err != nil || !on {
		return outgoing{}, errors.Join(err, errGone)
	}
	switch d.Channel {
	case ChannelPush:
		return s.composePush(ctx, agent, d)
	case ChannelEmail:
		return s.composeEmail(ctx, agent, d, n, prefs)
	}
	return s.composeSMS(ctx, agent, d, n, prefs)
}

// stillOn tells whether d's channel is still on: in the patient's settings, and for an SMS also the clinic's option.
func (s *Service) stillOn(ctx context.Context, d Delivery, p Preferences) (bool, error) {
	switch d.Channel {
	case ChannelPush:
		return p.Channels.Push, nil
	case ChannelEmail:
		return p.Channels.Email, nil
	}
	if !p.Channels.SMS {
		return false, nil
	}
	rules, err := s.p.Rules.For(ctx, d.OrganizationID) // the send's scope is the clinic's already
	return rules.SMSReminders, err
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

// startTimes is the start of n in the clinic's zone and, when the patient's zone (in p) gives another wall time, in
// theirs (zero otherwise), with the clinic and its zone's name.
type startTimes struct {
	there, local time.Time
	localZone    string
	clinic       organization.Organization
}

func (s *Service) startTimes(ctx context.Context, d Delivery, n appointment.ReminderNotice, p Preferences) (startTimes, error) {
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: d.OrganizationID}, d.OrganizationID)
	if err != nil {
		return startTimes{}, err
	}
	loc, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		return startTimes{}, fmt.Errorf("the clinic's time zone: %w", err)
	}
	out := startTimes{clinic: clinic, there: n.Start.In(loc)}
	if p.TimeZone == "" || p.TimeZone == clinic.TimeZone {
		return out, nil
	}
	theirs, err := time.LoadLocation(p.TimeZone)
	if err != nil || n.Start.In(theirs).Format(time.DateTime) == out.there.Format(time.DateTime) {
		return out, nil // a zone that no longer loads, or the same wall time: one time only
	}
	out.local, out.localZone = n.Start.In(theirs), p.TimeZone
	return out, nil
}

// composeEmail is the reminder e-mail: the clinic's name, the time (twice when the patient's zone differs), whether a
// questionnaire is outstanding, a one-time link to confirm or cancel, and the link to the app's appointments.
func (s *Service) composeEmail(ctx context.Context, agent string, d Delivery, n appointment.ReminderNotice, p Preferences) (outgoing, error) {
	to, language, err := s.accounts.AccountEmail(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	l := s.in(language)
	st, err := s.startTimes(ctx, d, n, p)
	if err != nil {
		return outgoing{}, err
	}
	data := letter{Clinic: st.clinic.Name, Link: s.patientAppURL + "/appointments"}
	if data.When, err = l.when(st.there, st.clinic.TimeZone); err != nil {
		return outgoing{}, err
	}
	if !st.local.IsZero() {
		if data.WhenLocal, err = l.when(st.local, st.localZone); err != nil {
			return outgoing{}, err
		}
	}
	if data.Questionnaire, err = s.questionnaires.Outstanding(ctx, agent, d.OrganizationID, n.AppointmentID); err != nil {
		return outgoing{}, err
	}
	if data.ActionLink, err = s.issue(ctx, agent, n, language, appointment.ChannelEmail); err != nil {
		return outgoing{}, err
	}
	msg, err := l.message(to, "reminder_email", data)
	return outgoing{mail: msg, ready: true}, err
}

// composeSMS is the reminder SMS to the patient's phone: the clinic's name, the date and time in its zone with the
// zone's name, the time in the patient's zone when it differs, and the one-time link. No phone any more cancels it.
func (s *Service) composeSMS(ctx context.Context, agent string, d Delivery, n appointment.ReminderNotice, p Preferences) (outgoing, error) {
	phone, err := s.p.Phones.NoticePhone(ctx, agent, d.OrganizationID, n.PatientID)
	if err != nil || phone == "" {
		return outgoing{}, errors.Join(err, errGone)
	}
	language, err := s.p.Languages.AccountLanguage(ctx, agent, d.AccountID)
	if err != nil {
		return outgoing{}, err
	}
	st, err := s.startTimes(ctx, d, n, p)
	if err != nil {
		return outgoing{}, err
	}
	link, err := s.issue(ctx, agent, n, language, appointment.ChannelSMS)
	if err != nil {
		return outgoing{}, err
	}
	data := letter{
		Clinic: st.clinic.Name, ActionLink: link, Zone: st.clinic.TimeZone,
		Day: st.there.Format("02"), Month: st.there.Format("01"), Year: st.there.Year(), Time: st.there.Format("15:04"),
	}
	if !st.local.IsZero() {
		data.ZoneLocal, data.DayLocal, data.MonthLocal, data.TimeLocal = st.localZone, st.local.Format("02"), st.local.Format("01"), st.local.Format("15:04")
	}
	body, err := s.in(language).text("reminder_sms", data)
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
