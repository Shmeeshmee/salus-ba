package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// Rules gives the clinic's appointment settings in the caller's transaction, unaudited (organization.Appointments):
// the reminders' offsets. It sets the clinic's scope on the transaction.
type Rules interface {
	For(ctx context.Context, org string) (organization.AppointmentSettings, error)
}

// scheduleReminders schedules the reminders of the booked appointment a in the transaction in ctx (docs/plans/
// appointments.md SAL-33, ADR 0015 "Reminders"): one appointment_reminder per offset of the clinic's settings now, due
// the offset before the start, with its dispatch job at that instant; an offset whose due time is not ahead of now
// gives none. Booking calls it; SAL-34's reschedule and SAL-35's acceptance too, so the clinic's offsets of that
// moment apply. The caller's scope must show a's reminders (the patient's own, or the clinic's).
func (s *Service) scheduleReminders(ctx context.Context, a Appointment, now time.Time) error {
	if s.Reminders == nil || s.Rules == nil {
		return errors.New("appointment: schedule reminders: no reminder store or clinic rules")
	}
	rules, err := s.Rules.For(ctx, a.OrganizationID)
	if err != nil {
		return err
	}
	var due []Reminder
	for _, offset := range rules.ReminderOffsetsMinutes {
		at := a.Start.Add(-time.Duration(offset) * time.Minute)
		if !at.After(now) {
			continue
		}
		r, ok, err := s.Reminders.AddReminder(ctx, Reminder{
			StartAt: a.Start, DueAt: at, AppointmentID: a.ID, OrganizationID: a.OrganizationID, PatientID: a.PatientID, OffsetMinutes: offset,
		})
		if err != nil {
			return err
		}
		if ok {
			due = append(due, r)
		}
	}
	if len(due) == 0 {
		return nil
	}
	return s.Jobs.EnqueueReminders(ctx, due)
}

// cancelReminders cancels the scheduled reminders of the appointment id in the transaction in ctx (every
// cancellation, and SAL-34's reschedule before it schedules the new start's). A reminder already dispatched is left:
// its sends re-check the appointment and send nothing once it no longer holds.
func (s *Service) cancelReminders(ctx context.Context, id string) error {
	if s.Reminders == nil {
		return errors.New("appointment: cancel reminders: no reminder store")
	}
	_, err := s.Reminders.CancelReminders(ctx, id)
	return err
}

// TakeReminder locks the reminder id of the clinic org for its dispatch by the job agent and returns what its sends
// need, in the transaction in ctx (or one of its own): in the clinic's scope with the reminder's appointment set, the
// appointment is read and the read audited with the job as agent. ErrNotFound for a reminder that is not the clinic's;
// ErrReminderGone for one no longer scheduled, and for one whose appointment is not booked at the start it was
// scheduled for or has started, which is then skipped: the caller commits its transaction on ErrReminderGone, so the
// skip is kept.
func (s *Service) TakeReminder(ctx context.Context, agent, org, id string) (ReminderNotice, error) {
	var out ReminderNotice
	var gone error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		r, err := s.Reminders.LockReminder(ctx, id)
		if err != nil {
			return err
		}
		if r.State != ReminderScheduled {
			gone = ErrReminderGone
			return nil
		}
		if out, err = s.reminderNotice(ctx, agent, r); errors.Is(err, ErrReminderGone) {
			gone = err
			return s.Reminders.SetReminderState(ctx, r.ID, ReminderSkipped, s.Clock.Now())
		}
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return ReminderNotice{}, ErrNotFound
	case err != nil:
		return ReminderNotice{}, fmt.Errorf("appointment: take reminder: %w", err)
	case gone != nil:
		return ReminderNotice{}, gone
	}
	return out, nil
}

// ReminderDispatched marks the reminder id of the clinic org dispatched now, in the transaction in ctx (the one its
// TakeReminder and its deliveries ran in).
func (s *Service) ReminderDispatched(ctx context.Context, org, id string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		return s.Reminders.SetReminderState(ctx, id, ReminderDispatched, s.Clock.Now())
	})
	if err != nil {
		return fmt.Errorf("appointment: reminder dispatched: %w", err)
	}
	return nil
}

// ReminderNotice returns, for a send of the job agent, the dispatched reminder id of the clinic org while its
// appointment is booked at the start it was scheduled for and ahead, read and audited as TakeReminder's. ErrNotFound
// for a reminder that is not the clinic's, ErrReminderGone otherwise.
func (s *Service) ReminderNotice(ctx context.Context, agent, org, id string) (ReminderNotice, error) {
	var out ReminderNotice
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		r, err := s.Reminders.Reminder(ctx, id)
		if err != nil {
			return err
		}
		if r.State != ReminderDispatched {
			return ErrReminderGone
		}
		out, err = s.reminderNotice(ctx, agent, r)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrReminderGone):
		return ReminderNotice{}, err
	case err != nil:
		return ReminderNotice{}, fmt.Errorf("appointment: reminder notice: %w", err)
	}
	return out, nil
}

// reminderNotice reads the appointment of r with its patient's account in the clinic's scope with the appointment
// set, audited with the job as agent; ErrReminderGone when it no longer holds.
func (s *Service) reminderNotice(ctx context.Context, agent string, r Reminder) (ReminderNotice, error) {
	if err := s.Repo.SetNotice(ctx, r.AppointmentID); err != nil {
		return ReminderNotice{}, err
	}
	a, err := s.Repo.ByID(ctx, r.AppointmentID)
	switch {
	case errors.Is(err, ErrNotFound):
		return ReminderNotice{}, ErrReminderGone
	case err != nil:
		return ReminderNotice{}, err
	case a.Status != StatusBooked || !a.Start.Equal(r.StartAt) || !a.Start.After(s.Clock.Now()):
		return ReminderNotice{}, ErrReminderGone
	}
	p, err := s.Patients.Lookup(ctx, a.PatientID)
	if err != nil {
		return ReminderNotice{}, err
	}
	if err := s.recordJob(ctx, agent, r.OrganizationID, auditevent.CodeRead, "Appointment", a.ID, auditevent.OutcomeSuccess); err != nil {
		return ReminderNotice{}, err
	}
	return ReminderNotice{
		Start: a.Start, ReminderID: r.ID, AppointmentID: a.ID, OrganizationID: a.OrganizationID, PatientID: a.PatientID, AccountID: p.AccountID,
	}, nil
}

// SweepReminders enqueues again the dispatch of every reminder of every clinic still scheduled past its due time
// while its start is ahead (the appointment_reminders_sweep job): a lost job is repaired, and a job still there is
// kept (unique by the reminder). In pages of page reminders, each in a transaction of its own; it returns how many it
// found.
func (s *Service) SweepReminders(ctx context.Context, page int) (int, error) {
	now := s.Clock.Now()
	afterDue, afterID, n := time.Time{}, "00000000-0000-0000-0000-000000000000", 0
	for {
		var rs []Reminder
		err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			var err error
			if rs, err = s.Reminders.OverdueReminders(ctx, now, afterDue, afterID, page); err != nil || len(rs) == 0 {
				return err
			}
			return s.Jobs.EnqueueReminders(ctx, rs)
		})
		if err != nil {
			return n, fmt.Errorf("appointment: sweep reminders: %w", err)
		}
		n += len(rs)
		if len(rs) < page {
			return n, nil
		}
		afterDue, afterID = rs[len(rs)-1].DueAt, rs[len(rs)-1].ID
	}
}

// ReminderPurger applies the retention rule of appointment_reminder (internal/retention): 90 days after its due time.
type ReminderPurger struct {
	repo ReminderRepository
}

// NewReminderPurger returns the purger of appointment_reminder on repo, built on the retention role's pool.
func NewReminderPurger(repo ReminderRepository) *ReminderPurger { return &ReminderPurger{repo: repo} }

// Purge deletes the reminders due before before and returns how many.
func (p *ReminderPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	n, err := p.repo.DeleteRemindersDueBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("appointment: purge reminders: %w", err)
	}
	return n, nil
}

// slotFreed hands the slot of the clinic org to the waitlist (waitlist_slot_freed, the seam of SAL-35) when the
// change that released it freed it, that is, while it is still ahead.
func (s *Service) slotFreed(ctx context.Context, freed bool, org, slotID string) error {
	if !freed {
		return nil
	}
	return s.Jobs.EnqueueSlotFreed(ctx, slotID, org)
}
