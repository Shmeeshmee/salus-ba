package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Confirm records that the patient will come to their booked appointment id, while it is ahead, and returns it
// (docs/plans/appointments.md, SAL-34): in one transaction in the patient's scope, the confirmation is stored (with the
// way they confirmed: their app, or a reminder's link), kept in the history and audited. Confirming again answers the
// same appointment and changes nothing. ErrForbidden to staff, ErrNotFound for an appointment that is not the
// patient's, ErrClosed for one no longer booked or started; each refusal is audited.
func (s *Service) Confirm(ctx context.Context, c Caller, id string) (Appointment, error) {
	if !c.patient() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.confirm(ctx, c, id); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: confirm: %w", err)
	case refused != nil:
		return Appointment{}, refused
	}
	return out, nil
}

// confirm confirms the appointment id for the patient c in the transaction in ctx, or gives the refusal.
func (s *Service) confirm(ctx context.Context, c Caller, id string) (out Appointment, refused, err error) {
	v, err := s.view(ctx, c, id)
	now := s.Clock.Now()
	switch {
	case err != nil:
		return Appointment{}, nil, err
	case v.level == access.None:
		return Appointment{}, ErrNotFound, nil
	case v.level != access.Full:
		return Appointment{}, ErrForbidden, nil
	case v.a.Status != StatusBooked || !v.a.Start.After(now):
		return Appointment{}, ErrClosed, nil
	case !v.a.ConfirmedAt.IsZero():
		return v.a, nil, nil
	}
	out, err = s.SelfService.Confirm(ctx, id, c.via(), now)
	if errors.Is(err, ErrClosed) { // a concurrent confirmation or cancellation committed first
		a, err := s.Repo.ByID(ctx, id)
		if err != nil {
			return Appointment{}, nil, err
		}
		if a.Status == StatusBooked && !a.ConfirmedAt.IsZero() {
			return a, nil, nil
		}
		return Appointment{}, ErrClosed, nil
	}
	if err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, s.recordChange(ctx, Change{At: now, AppointmentID: id, Kind: ChangeConfirmed, Via: c.via(), ByAccountID: c.AccountID})
}

// Actions tells the patient what they may do with their appointment id now, by the clinic's rules (GET
// /appointments/{id}/actions): confirm while unconfirmed and ahead; cancel until 24 hours before the start; reschedule
// while the clinic allows self-service rescheduling, reschedules are left, its notice has not begun and the
// consultation's join window has not opened. None for an appointment no longer booked. Audited as a read of the
// appointment. ErrForbidden to staff, ErrNotFound for one that is not the patient's; each refusal is audited.
func (s *Service) Actions(ctx context.Context, c Caller, id string) (Actions, error) {
	if !c.patient() {
		return Actions{}, s.refuse(ctx, c, auditevent.CodeRead, ErrForbidden)
	}
	var out Actions
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		v, err := s.view(ctx, c, id)
		switch {
		case err != nil:
			return err
		case v.level == access.None:
			refused = ErrNotFound
			return s.record(ctx, c, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure)
		}
		if out, err = s.actions(ctx, v.a, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeRead, "Appointment", id, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Actions{}, fmt.Errorf("appointment: actions: %w", err)
	case refused != nil:
		return Actions{}, refused
	}
	return out, nil
}

// actions are the patient's actions on a at now, by its clinic's settings (read in the transaction in ctx).
func (s *Service) actions(ctx context.Context, a Appointment, now time.Time) (Actions, error) {
	if a.Status != StatusBooked || !a.Start.After(now) {
		return Actions{}, nil
	}
	set, err := s.settingsFor(ctx, a.OrganizationID, a.PatientID)
	if err != nil {
		return Actions{}, err
	}
	out := Actions{Confirm: a.ConfirmedAt.IsZero()}
	if until := a.Start.Add(-patientCancelBefore); !now.After(until) {
		out.Cancel, out.CancelUntil = true, until
	}
	if set.RescheduleMax == 0 {
		return out, nil
	}
	out.ReschedulesLeft = max(set.RescheduleMax-int(a.RescheduleCount), 0)
	if until := rescheduleUntil(a, set.RescheduleNoticeHours); out.ReschedulesLeft > 0 && now.Before(until) {
		out.Reschedule, out.RescheduleUntil = true, until
	}
	return out, nil
}
