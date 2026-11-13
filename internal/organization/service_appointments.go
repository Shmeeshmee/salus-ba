package organization

import (
	"context"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The ranges of the appointment settings and the defaults of a clinic that has stored none (docs/plans/appointments.md
// "Contract", ADR 0015 "Clinic rules"); the table's checks say the same.
const (
	maxReminders              = 3
	minReminderOffset         = 15 // minutes before the start
	maxReminderOffset         = 7 * 24 * 60
	maxRescheduleNoticeHours  = 168
	maxReschedules            = 5
	minWaitlistHoldMinutes    = 5
	maxWaitlistHoldMinutes    = 120
	maxNoShowFlagThreshold    = 10
	minNoShowFlagWindowDays   = 30
	maxNoShowFlagWindowDays   = 730
	defaultRescheduleNotice   = 24
	defaultReschedules        = 2
	defaultWaitlistHold       = 15
	defaultNoShowFlagWindow   = 180
	defaultReminderDayBefore  = 24 * 60
	defaultReminderHourBefore = 60
)

// DefaultAppointmentSettings are a clinic's settings until its admin stores any: reminders 24 hours and 1 hour before,
// no SMS, rescheduling until 24 hours before and twice at most, no waitlist (a 15-minute hold once on), no no-show flag.
func DefaultAppointmentSettings() AppointmentSettings {
	return AppointmentSettings{
		ReminderOffsetsMinutes: []int{defaultReminderDayBefore, defaultReminderHourBefore},
		RescheduleNoticeHours:  defaultRescheduleNotice, RescheduleMax: defaultReschedules,
		WaitlistHoldMinutes: defaultWaitlistHold, NoShowFlagWindowDays: defaultNoShowFlagWindow,
	}
}

// AppointmentsPorts are what Appointments runs on.
type AppointmentsPorts struct {
	Repo  AppointmentsRepository
	Audit Auditor
	Tx    Transactor
	Clock clock.Clock
}

// Appointments holds the clinic's appointment settings: only its admin reads and changes them (a change is audited,
// also when refused); the appointment, notification and waitlist rules read them through For.
type Appointments struct{ AppointmentsPorts }

// NewAppointments returns Appointments on its ports.
func NewAppointments(p AppointmentsPorts) *Appointments { return &Appointments{AppointmentsPorts: p} }

// Settings returns the settings of the admin's clinic, the defaults when it has stored none. ErrForbidden for its
// other staff, ErrNotFound for anyone else.
func (s *Appointments) Settings(ctx context.Context, c Caller) (AppointmentSettings, error) {
	if err := adminOf(c); err != nil {
		return AppointmentSettings{}, err
	}
	out, err := s.For(ctx, c.OrganizationID)
	if err != nil {
		return AppointmentSettings{}, fmt.Errorf("organization: appointment settings: %w", err)
	}
	return out, nil
}

// SetSettings replaces the settings of the admin's clinic with in, audited as an update of the clinic; refusals as
// Settings', audited too. ErrInvalidAppointmentSettings for settings out of their ranges.
func (s *Appointments) SetSettings(ctx context.Context, c Caller, in AppointmentSettings) (AppointmentSettings, error) {
	if err := adminOf(c); err != nil {
		if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			return s.Audit.Record(ctx, settingsEvent(c, auditevent.OutcomeMinorFailure))
		}); aerr != nil {
			return AppointmentSettings{}, fmt.Errorf("organization: appointment settings: %w", aerr)
		}
		return AppointmentSettings{}, err
	}
	if err := validAppointmentSettings(in); err != nil {
		return AppointmentSettings{}, err
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := s.Repo.PutAppointmentSettings(ctx, c.OrganizationID, in, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, settingsEvent(c, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return AppointmentSettings{}, fmt.Errorf("organization: appointment settings: %w", err)
	}
	return in, nil
}

// For returns the settings of the clinic org, the defaults when it has stored none, in the transaction in ctx (or one
// of its own) with the clinic's scope set: for the other domains' rules. Unaudited: no personal data.
func (s *Appointments) For(ctx context.Context, org string) (AppointmentSettings, error) {
	var out AppointmentSettings
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		set, ok, err := s.Repo.AppointmentSettings(ctx, org)
		switch {
		case err != nil:
			return err
		case !ok:
			set = DefaultAppointmentSettings()
		}
		out = set
		return nil
	})
	if err != nil {
		return AppointmentSettings{}, fmt.Errorf("organization: appointment settings for a clinic: %w", err)
	}
	return out, nil
}

// validAppointmentSettings is nil for settings within their ranges, with distinct reminder offsets.
func validAppointmentSettings(in AppointmentSettings) error {
	offsets := slices.Clone(in.ReminderOffsetsMinutes)
	slices.Sort(offsets)
	if len(offsets) > maxReminders || len(slices.Compact(slices.Clone(offsets))) != len(offsets) {
		return ErrInvalidAppointmentSettings
	}
	for _, m := range offsets {
		if m < minReminderOffset || m > maxReminderOffset {
			return ErrInvalidAppointmentSettings
		}
	}
	if !within(in.RescheduleNoticeHours, 0, maxRescheduleNoticeHours) || !within(in.RescheduleMax, 0, maxReschedules) ||
		!within(in.WaitlistHoldMinutes, minWaitlistHoldMinutes, maxWaitlistHoldMinutes) ||
		!within(in.NoShowFlagThreshold, 0, maxNoShowFlagThreshold) ||
		!within(in.NoShowFlagWindowDays, minNoShowFlagWindowDays, maxNoShowFlagWindowDays) {
		return ErrInvalidAppointmentSettings
	}
	return nil
}

func within(v, lo, hi int) bool { return v >= lo && v <= hi }
