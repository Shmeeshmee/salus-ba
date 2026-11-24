package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// MoveForPatient moves the booked appointment id at version to the free slot slotID of the clinic as its admin or
// receptionist c (docs/clinic-cms WAVE4 M1, V4): any free slot of the clinic of the visit's type, another
// practitioner's too, listed or not. Unlike the patient's Reschedule it does not count toward the patient's limit, does
// not confirm the visit and follows none of the clinic's self-service rules. The slot is resolved in the clinic's own
// scope first; then, in one transaction in the clinic's scope, the appointment is read and checked, the patient's
// active care consent checked, the slot held, the appointment moved, the old slot freed and handed to the waitlist
// while ahead, the reminders scheduled again and the e-mail with the new time enqueued (neither for a patient without
// an account), the statistics of both days refreshed, the history kept (via staff, with the account) and the move
// audited. ErrForbidden to anyone but the desk, ErrNotFound for an appointment the clinic does not see, ErrClosed for
// one not booked (arrived, fulfilled, cancelled, a no-show), ErrVersionConflict for another version (also a change
// that committed first), ErrNotReschedulable for a slot of the other visit type and for a video visit whose join window
// has opened, ErrConsentRequired, ErrSlotUnavailable for a slot the clinic does not have, one not free and ahead, of
// an inactive role, or a time the role is booked at; each refusal is audited.
func (s *Service) MoveForPatient(ctx context.Context, c Caller, id, slotID string, version int32) (Appointment, error) {
	if !c.desk() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	now := s.Clock.Now()
	b, err := s.Slots.BookableInClinic(ctx, c.OrganizationID, slotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable), err == nil && !b.Start.After(now):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: move for patient: %w", err)
	}
	var out Appointment
	var refused error
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.moveFor(ctx, c, id, b, version, now); err != nil || refused != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrSlotUnavailable), errors.Is(err, ErrVersionConflict): // the update aborted the transaction or changed nothing
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, err)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: move for patient: %w", err)
	case refused != nil:
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, refused)
	}
	return out, nil
}

// moveFor checks and makes the move of the appointment id to the slot b in the transaction in ctx, or gives the
// refusal.
func (s *Service) moveFor(ctx context.Context, c Caller, id string, b slot.Bookable, version int32, now time.Time) (out Appointment, refused, err error) {
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Appointment{}, nil, err
	}
	a, err := s.Repo.ByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound), err == nil && a.OrganizationID != c.OrganizationID:
		return Appointment{}, ErrNotFound, nil
	case err != nil:
		return Appointment{}, nil, err
	}
	if refused := mayMove(a, b, version, now); refused != nil {
		return Appointment{}, refused, nil
	}
	role, err := s.Roles.Lookup(ctx, c.OrganizationID, b.PractitionerRoleID)
	if errors.Is(err, practitionerrole.ErrNotFound) || err == nil && !role.Active {
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	p, err := s.Patients.Lookup(ctx, a.PatientID)
	if errors.Is(err, patient.ErrNotFound) {
		return Appointment{}, ErrNotFound, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	consent, err := s.Consents.State(ctx, a.PatientID, a.OrganizationID)
	if err != nil || !consent.Active {
		return Appointment{}, ErrConsentRequired, err
	}
	if _, err := s.Slots.Hold(ctx, c.OrganizationID, b.ID); errors.Is(err, slot.ErrUnavailable) {
		return Appointment{}, ErrSlotUnavailable, nil
	} else if err != nil {
		return Appointment{}, nil, err
	}
	out, err = s.Repo.MoveByStaff(ctx, id, StaffMove{Start: b.Start, End: b.End, SlotID: b.ID, PractitionerRoleID: b.PractitionerRoleID, Version: version}, now)
	if err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, s.movedByStaff(ctx, c, a, out, p.AccountID != "", now)
}

// mayMove is nil when the desk may move the appointment a at version to the slot b at now, and the refusal otherwise.
// An in-person visit moves while booked (a late patient too); a video visit until its join window opens, so the window
// always follows the start it was given.
func mayMove(a Appointment, b slot.Bookable, version int32, now time.Time) error {
	switch {
	case a.Status != StatusBooked:
		return ErrClosed
	case a.VersionID != version:
		return ErrVersionConflict
	case !sameMode(a, b), visitMode(a.Mode) == schedule.ModeVideo && !now.Before(a.Start.Add(-joinOpensBefore)):
		return ErrNotReschedulable
	case b.ID == a.SlotID:
		return ErrSlotUnavailable
	}
	return nil
}

// movedByStaff does, in the transaction in ctx, what follows the desk's move of from to out: the old slot freed and
// handed to the waitlist while ahead, the reminders cancelled and, for a patient with an account, scheduled again with
// the e-mail with the new time, the history, and the refresh of both days' statistics.
func (s *Service) movedByStaff(ctx context.Context, c Caller, from, out Appointment, account bool, now time.Time) error {
	freed, err := s.Slots.Release(ctx, from.OrganizationID, from.SlotID)
	if err != nil {
		return err
	}
	if err := s.slotFreed(ctx, freed, from.OrganizationID, from.SlotID); err != nil {
		return err
	}
	if err := s.cancelReminders(ctx, out.ID); err != nil {
		return err
	}
	if account {
		if err := s.scheduleReminders(ctx, out, now); err != nil {
			return err
		}
		if err := s.Jobs.EnqueueRescheduled(ctx, out.ID, out.OrganizationID); err != nil {
			return err
		}
	}
	if err := s.recordChange(ctx, Change{
		At: now, AppointmentID: out.ID, Kind: ChangeRescheduled, Via: ViaStaff, ByAccountID: c.AccountID,
		FromStart: from.Start, FromEnd: from.End, FromSlotID: from.SlotID, ToStart: out.Start, ToEnd: out.End, ToSlotID: out.SlotID,
	}); err != nil {
		return err
	}
	if err := s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, from.Start); err != nil {
		return err
	}
	return s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start)
}
