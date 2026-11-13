package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

// Profiles reads the signed-in patient's own profile, audited (patient.Service): whether it has a phone for SMS.
type Profiles interface {
	Me(ctx context.Context, c patient.Caller) (patient.Patient, error)
}

// maxZoneName bounds a time zone's name (notification_preference.time_zone).
const maxZoneName = 64

// Preferences returns the patient's notification settings, the defaults until they change them, audited as a read of
// their account's settings. ErrForbidden to staff.
func (s *Service) Preferences(ctx context.Context, c Caller) (Preferences, error) {
	if c.OrganizationID != "" {
		return Preferences{}, ErrForbidden
	}
	var out Preferences
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		p, ok, err := s.p.Deliveries.Preferences(ctx, c.AccountID)
		if err != nil {
			return err
		}
		if out = p; !ok {
			out = DefaultPreferences()
		}
		return s.p.Audit.Record(ctx, ownSettings(c.AccountID, auditevent.CodeRead, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Preferences{}, fmt.Errorf("notification: preferences: %w", err)
	}
	return out, nil
}

// SetPreferences replaces the patient's notification settings with in and returns them, audited as an update (a
// refusal too). ErrForbidden to staff; ErrInvalid for quiet hours that are not HH:MM or start and end alike, or a zone
// that is not an IANA zone; ErrPhoneRequired for SMS without a phone in their profile (or without a profile).
func (s *Service) SetPreferences(ctx context.Context, c Caller, in Preferences) (Preferences, error) {
	if c.OrganizationID != "" {
		return Preferences{}, ErrForbidden
	}
	refused := validPreferences(in)
	if refused == nil && in.Channels.SMS {
		me, err := s.p.Profiles.Me(ctx, patient.Caller{AccountID: c.AccountID})
		switch {
		case errors.Is(err, patient.ErrNotFound), err == nil && me.Phone == "":
			refused = ErrPhoneRequired
		case err != nil:
			return Preferences{}, fmt.Errorf("notification: set preferences: %w", err)
		}
	}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if refused != nil {
			return s.p.Audit.Record(ctx, ownSettings(c.AccountID, auditevent.CodeUpdate, auditevent.OutcomeMinorFailure))
		}
		if err := s.p.Deliveries.SetPreferences(ctx, c.AccountID, in, s.p.Clock.Now()); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, ownSettings(c.AccountID, auditevent.CodeUpdate, auditevent.OutcomeSuccess))
	})
	switch {
	case err != nil:
		return Preferences{}, fmt.Errorf("notification: set preferences: %w", err)
	case refused != nil:
		return Preferences{}, refused
	}
	return in, nil
}

// validPreferences is nil for settings the contract takes.
func validPreferences(in Preferences) error {
	if q := in.Quiet; q != nil {
		start, okS := clockMinutes(q.Start)
		end, okE := clockMinutes(q.End)
		if !okS || !okE || start == end || len(q.Start) != 5 || len(q.End) != 5 {
			return ErrInvalid
		}
	}
	if z := in.TimeZone; z != "" {
		if _, err := time.LoadLocation(z); err != nil || len(z) > maxZoneName || z == "Local" {
			return ErrInvalid
		}
	}
	return nil
}

// ownSettings is the audit event of the patient's own interaction with their account's notification settings.
func ownSettings(account, interaction string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := auditevent.ActionRead
	if interaction == auditevent.CodeUpdate {
		action = auditevent.ActionUpdate
	}
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodePATRQT},
		Action:          action,
		Outcome:         outcome,
		AgentAccountID:  account,
		EntityReference: "Account/" + account,
		EntityName:      "notification_preference",
	}
}
