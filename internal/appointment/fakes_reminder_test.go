package appointment_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// reminderID is the id the fake gives the nth reminder it stores (from 1).
func reminderID(n int) string { return fmt.Sprintf("e%07d-0000-4000-8000-000000000000", n) }

// fakeReminders holds the reminders and shows them as row-level security does: the patient's own, or the clinic's.
type fakeReminders struct {
	repo      *fakeRepository
	err       error
	reminders []appointment.Reminder
	locked    []string
}

func (r *fakeReminders) visible(x appointment.Reminder) bool {
	if r.repo.patient != "" {
		return x.PatientID == r.repo.patient
	}
	return r.repo.clinic != "" && x.OrganizationID == r.repo.clinic
}

func (r *fakeReminders) AddReminder(ctx context.Context, x appointment.Reminder) (appointment.Reminder, bool, error) {
	if ctx.Value(txKey{}) == nil {
		return appointment.Reminder{}, false, errors.New("outside a transaction")
	}
	if r.err != nil {
		return appointment.Reminder{}, false, r.err
	}
	for _, y := range r.reminders {
		if y.AppointmentID == x.AppointmentID && y.OffsetMinutes == x.OffsetMinutes && y.StartAt.Equal(x.StartAt) {
			return appointment.Reminder{}, false, nil
		}
	}
	if !x.DueAt.Equal(x.StartAt.Add(-time.Duration(x.OffsetMinutes) * time.Minute)) {
		return appointment.Reminder{}, false, errors.New("a due time off its offset")
	}
	x.ID, x.State = reminderID(len(r.reminders)+1), appointment.ReminderScheduled
	r.reminders = append(r.reminders, x)
	return x, true, nil
}

func (r *fakeReminders) CancelReminders(ctx context.Context, id string) (int64, error) {
	if ctx.Value(txKey{}) == nil {
		return 0, errors.New("outside a transaction")
	}
	var n int64
	for i, x := range r.reminders {
		if x.AppointmentID == id && x.State == appointment.ReminderScheduled && r.visible(x) {
			r.reminders[i].State = appointment.ReminderCancelled
			n++
		}
	}
	return n, r.err
}

func (r *fakeReminders) find(id string) (int, error) {
	for i, x := range r.reminders {
		if x.ID == id && r.visible(x) {
			return i, r.err
		}
	}
	return 0, appointment.ErrNotFound
}

func (r *fakeReminders) LockReminder(ctx context.Context, id string) (appointment.Reminder, error) {
	if ctx.Value(txKey{}) == nil {
		return appointment.Reminder{}, errors.New("locked outside a transaction")
	}
	i, err := r.find(id)
	if err != nil {
		return appointment.Reminder{}, err
	}
	r.locked = append(r.locked, id)
	return r.reminders[i], nil
}

func (r *fakeReminders) Reminder(_ context.Context, id string) (appointment.Reminder, error) {
	i, err := r.find(id)
	if err != nil {
		return appointment.Reminder{}, err
	}
	return r.reminders[i], nil
}

func (r *fakeReminders) SetReminderState(ctx context.Context, id string, state appointment.ReminderState, at time.Time) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("outside a transaction")
	}
	i, err := r.find(id)
	if err != nil {
		return err
	}
	if r.reminders[i].State != appointment.ReminderScheduled {
		return errors.New("a reminder leaves scheduled once")
	}
	r.reminders[i].State = state
	if state == appointment.ReminderDispatched {
		r.reminders[i].DispatchedAt = at
	}
	return nil
}

func (r *fakeReminders) OverdueReminders(_ context.Context, at, afterDue time.Time, afterID string, max int) ([]appointment.Reminder, error) {
	var out []appointment.Reminder
	for _, x := range r.reminders {
		after := x.DueAt.After(afterDue) || x.DueAt.Equal(afterDue) && x.ID > afterID
		if x.State == appointment.ReminderScheduled && !x.DueAt.After(at) && x.StartAt.After(at) && after {
			out = append(out, appointment.Reminder{ID: x.ID, OrganizationID: x.OrganizationID, DueAt: x.DueAt})
		}
	}
	slices.SortFunc(out, func(a, b appointment.Reminder) int {
		if c := a.DueAt.Compare(b.DueAt); c != 0 {
			return c
		}
		return compareStrings(a.ID, b.ID)
	})
	if len(out) > max {
		out = out[:max]
	}
	return out, r.err
}

func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (r *fakeReminders) DeleteRemindersDueBefore(_ context.Context, before time.Time) (int64, error) {
	n := len(r.reminders)
	r.reminders = slices.DeleteFunc(r.reminders, func(x appointment.Reminder) bool { return x.DueAt.Before(before) })
	return int64(n - len(r.reminders)), r.err
}

// fakeRules gives each clinic's appointment settings (the defaults unless set).
type fakeRules struct {
	err      error
	settings map[string]organization.AppointmentSettings
}

func (r *fakeRules) For(_ context.Context, org string) (organization.AppointmentSettings, error) {
	if s, ok := r.settings[org]; ok {
		return s, r.err
	}
	return organization.DefaultAppointmentSettings(), r.err
}
