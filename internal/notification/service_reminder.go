package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
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
	if err := s.delivers(false); err != nil || s.p.Reminders == nil {
		return errors.Join(err, errors.New("notification: dispatch reminder: no reminders"))
	}
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
// reads of the account's settings and devices are audited with the kind's agent (agentOf). The clinic's settings are
// read in its scope; the caller's clinic scope is set back after (as it was, or none).
func (s *Service) Deliver(ctx context.Context, kind Kind, sourceID, org, accountID string, c Content) error {
	if err := s.delivers(false); err != nil {
		return err
	}
	agent := agentOf(kind)
	return s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		prefs, err := s.preferencesOf(ctx, agent, org, accountID)
		if err != nil {
			return err
		}
		rules, err := s.rulesKeepingScope(ctx, org)
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

// agentOf is the audit agent of a kind's deliveries: "<kind>_delivery" (reminder_delivery, waitlist_offer_delivery).
func agentOf(kind Kind) string { return string(kind) + "_delivery" }

// rulesKeepingScope reads the clinic org's settings (organization.Appointments.For sets app.organization_id) and sets
// the transaction's clinic scope back to what it was before.
func (s *Service) rulesKeepingScope(ctx context.Context, org string) (organization.AppointmentSettings, error) {
	before, err := s.p.Deliveries.ClinicScope(ctx)
	if err != nil {
		return organization.AppointmentSettings{}, err
	}
	rules, err := s.p.Rules.For(ctx, org)
	if err != nil {
		return organization.AppointmentSettings{}, err
	}
	return rules, s.p.Devices.SetClinic(ctx, before)
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
