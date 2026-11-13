package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// joinOpensBefore is when the consultation's join window opens before the start (docs/consultation/SPEC.md §2 "Join
// window", videosession): a reschedule is refused from then on, whatever the clinic's notice (ADR 0015).
const joinOpensBefore = 10 * time.Minute

// settingsFor reads the appointment settings of the clinic org in the transaction in ctx, then drops the clinic's scope
// that For sets and sets the patient's again, so what follows runs in the patient's scope only.
func (s *Service) settingsFor(ctx context.Context, org, patientID string) (organization.AppointmentSettings, error) {
	set, err := s.Rules.For(ctx, org)
	if err != nil {
		return organization.AppointmentSettings{}, err
	}
	if err := s.Repo.SetClinic(ctx, ""); err != nil {
		return organization.AppointmentSettings{}, err
	}
	return set, s.Repo.SetPatient(ctx, patientID)
}

// rescheduleRefusals are the refusals of a reschedule. Each rolls its transaction back (a slot may have been held
// already, or the exclusion constraint aborted it) and is audited in a transaction of its own.
var rescheduleRefusals = []error{
	ErrNotFound, ErrForbidden, ErrClosed, ErrConsentRequired, ErrSlotUnavailable,
	ErrNotReschedulable, ErrRescheduleWindowClosed, ErrRescheduleLimit,
}

// rescheduleUntil is the moment a's rescheduling closes: notice hours before the start, and never later than the
// opening of the consultation's join window.
func rescheduleUntil(a Appointment, noticeHours int) time.Time {
	return a.Start.Add(-max(time.Duration(noticeHours)*time.Hour, joinOpensBefore))
}

// Reschedule moves the patient's booked appointment id to the free slot slotID of the same practitioner role, and
// returns it at its new time and version (ADR 0015 "A reschedule is the same Appointment with a new time"). The slot is
// resolved in the public directory first; then, in one transaction in the patient's scope, the appointment is locked
// and checked against the clinic's rules (self-service on, the same role, the join window not open, its notice not
// begun, reschedules left) and the patient's consent, the new slot is held, the appointment moved (and confirmed in
// the app when it was not), the old slot freed and handed to the waitlist while ahead, the reminders scheduled again,
// the e-mail with the new time and the statistics of both days enqueued, the history kept and the change audited.
// ErrForbidden to staff, ErrNotFound for an appointment that is not the patient's, ErrClosed for one no longer
// booked (or moved by a concurrent reschedule), ErrNotReschedulable, ErrRescheduleWindowClosed, ErrRescheduleLimit,
// ErrConsentRequired, ErrSlotUnavailable for a slot that is not free and ahead or a time the role is booked at; each
// refusal is audited.
func (s *Service) Reschedule(ctx context.Context, c Caller, id, slotID string) (Appointment, error) {
	if !c.patient() {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	now := s.Clock.Now()
	b, err := s.Slots.Bookable(ctx, slotID)
	switch {
	case errors.Is(err, slot.ErrUnavailable), err == nil && !b.Start.After(now):
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrSlotUnavailable)
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: reschedule: %w", err)
	}
	var out Appointment
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.reschedule(ctx, c, id, b, now); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	for _, refusal := range rescheduleRefusals {
		if errors.Is(err, refusal) {
			return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, refusal)
		}
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("appointment: reschedule: %w", err)
	}
	return out, nil
}

// reschedule moves the appointment id to the slot b for the patient c in the transaction in ctx; a refusal is
// returned as the error, rolling the transaction back.
func (s *Service) reschedule(ctx context.Context, c Caller, id string, b slot.Bookable, now time.Time) (Appointment, error) {
	v, err := s.view(ctx, c, id)
	switch {
	case err != nil:
		return Appointment{}, err
	case v.level == access.None:
		return Appointment{}, ErrNotFound
	case v.level != access.Full:
		return Appointment{}, ErrForbidden
	}
	a, err := s.SelfService.ByIDForUpdate(ctx, id) // a concurrent reschedule or cancellation waits, or was first
	if err != nil {
		return Appointment{}, err
	}
	if err := s.mayReschedule(ctx, a, b, now); err != nil {
		return Appointment{}, err
	}
	if _, err := s.Slots.Hold(ctx, b.OrganizationID, b.ID); errors.Is(err, slot.ErrUnavailable) {
		return Appointment{}, ErrSlotUnavailable
	} else if err != nil {
		return Appointment{}, err
	}
	out, err := s.SelfService.Reschedule(ctx, id, Move{Start: b.Start, End: b.End, SlotID: b.ID, From: a.RescheduleCount}, now)
	if err != nil {
		return Appointment{}, err
	}
	return out, s.rescheduled(ctx, c, a, out, now)
}

// mayReschedule is nil when the booked appointment a may move to the slot b at now by its clinic's rules and the
// patient's consent, and the refusal otherwise.
func (s *Service) mayReschedule(ctx context.Context, a Appointment, b slot.Bookable, now time.Time) error {
	if a.Status != StatusBooked {
		return ErrClosed
	}
	set, err := s.settingsFor(ctx, a.OrganizationID, a.PatientID)
	if err != nil {
		return err
	}
	switch {
	case set.RescheduleMax == 0, b.PractitionerRoleID != a.PractitionerRoleID, b.OrganizationID != a.OrganizationID,
		!now.Before(a.Start.Add(-joinOpensBefore)):
		return ErrNotReschedulable
	case !now.Before(rescheduleUntil(a, set.RescheduleNoticeHours)):
		return ErrRescheduleWindowClosed
	case int(a.RescheduleCount) >= set.RescheduleMax:
		return ErrRescheduleLimit
	case b.ID == a.SlotID:
		return ErrSlotUnavailable
	}
	consent, err := s.Consents.State(ctx, a.PatientID, a.OrganizationID)
	switch {
	case err != nil:
		return err
	case !consent.Active:
		return ErrConsentRequired
	}
	return s.settle(ctx, a)
}

// rescheduled does, in the transaction in ctx, what follows the move of from to out: the old slot freed and handed to
// the waitlist while ahead, the reminders cancelled and scheduled again for the new start, the history (and the
// confirmation that choosing the time is), the e-mail with the new time and the refresh of both days' statistics.
func (s *Service) rescheduled(ctx context.Context, c Caller, from, out Appointment, now time.Time) error {
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
	if err := s.scheduleReminders(ctx, out, now); err != nil {
		return err
	}
	if err := s.recordChange(ctx, Change{
		At: now, AppointmentID: out.ID, Kind: ChangeRescheduled, Via: c.via(), ByAccountID: c.AccountID,
		FromStart: from.Start, FromEnd: from.End, FromSlotID: from.SlotID, ToStart: out.Start, ToEnd: out.End, ToSlotID: out.SlotID,
	}); err != nil {
		return err
	}
	if from.ConfirmedAt.IsZero() {
		if err := s.recordChange(ctx, Change{At: now, AppointmentID: out.ID, Kind: ChangeConfirmed, Via: ViaApp, ByAccountID: c.AccountID}); err != nil {
			return err
		}
	}
	if err := s.Jobs.EnqueueRescheduled(ctx, out.ID, out.OrganizationID); err != nil {
		return err
	}
	if err := s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, from.Start); err != nil {
		return err
	}
	return s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start)
}
