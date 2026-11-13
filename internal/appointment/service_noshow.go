package appointment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

const (
	// correctBefore is how long after its start a no-show or an attendance may be corrected (Q6).
	correctBefore = 30 * 24 * time.Hour
	// maxFlagRange is the longest range the no-show flags are asked for: two months of a calendar.
	maxFlagRange = 62 * 24 * time.Hour
)

// corrects reports whether moving an appointment in status from to status to is a correction: a no-show into an
// attendance or back.
func corrects(from, to Status) bool {
	return from != to && (from == StatusNoShow || from == StatusFulfilled) && (to == StatusNoShow || to == StatusFulfilled)
}

// correctStatus corrects the appointment of v to status at now for c, in the transaction in ctx (Q6): by its
// practitioner or the clinic's admin (access.CorrectsOutcome), within 30 days of its start; kept in its history as a
// status_corrected by staff and the clinic's day refreshed. It gives the refusal, or the appointment.
func (s *Service) correctStatus(ctx context.Context, c Caller, v view, status Status, now time.Time) (out Appointment, refused, err error) {
	switch {
	case !access.CorrectsOutcome(v.viewer, v.facts, visit(v.a)):
		return Appointment{}, ErrForbidden, nil
	case now.Sub(v.a.Start) > correctBefore:
		return Appointment{}, ErrCorrectionWindowClosed, nil
	}
	out, err = s.NoShows.CorrectStatus(ctx, v.a.ID, v.a.Status, status, now)
	switch {
	case errors.Is(err, ErrClosed):
		return Appointment{}, ErrClosed, nil
	case err != nil:
		return Appointment{}, nil, err
	}
	if err := s.recordChange(ctx, Change{
		At: now, AppointmentID: out.ID, Kind: ChangeStatusCorrected, Via: ViaStaff, ByAccountID: c.AccountID,
		FromStatus: v.a.Status, ToStatus: status,
	}); err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start)
}

// NoShowFlags returns the repeat no-show flags (Q7) of the appointments starting in [from, to) that the staff caller
// sees in full: those whose patient has at least the clinic's threshold of noshow appointments at the clinic that
// started in the clinic's window up to now, read in the clinic's scope; none while the threshold is 0 (the default).
// Every appointment of the range seen in full is audited, flagged or not (that one is not flagged says something of
// its patient too), or one event without an entity when there is none. ErrInvalidRange for a range missing, reversed
// or longer than 62 days (nothing read); ErrNotFound, audited, to a patient: the flag is never theirs to see.
func (s *Service) NoShowFlags(ctx context.Context, c Caller, from, to time.Time) ([]NoShowFlag, error) {
	if from.IsZero() || !to.After(from) || to.Sub(from) > maxFlagRange {
		return nil, ErrInvalidRange
	}
	if c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrNotFound)
	}
	var out []NoShowFlag
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		set, err := s.Rules.For(ctx, c.OrganizationID)
		if err != nil || set.NoShowFlagThreshold == 0 {
			return err
		}
		seen, err := s.flagRange(ctx, c, from, to)
		if err != nil {
			return err
		}
		if out, err = s.flagged(ctx, c.OrganizationID, set, seen); err != nil {
			return err
		}
		for _, a := range seen {
			if err := s.record(ctx, c, auditevent.CodeSearchType, "Appointment", a.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		if len(seen) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: no-show flags: %w", err)
	}
	return out, nil
}

// flagRange gives the appointments of [from, to) that the staff caller sees in full, by start, in the clinic's scope
// set on the transaction in ctx: an admin's of every role, a practitioner's of their own role only.
func (s *Service) flagRange(ctx context.Context, c Caller, from, to time.Time) ([]Appointment, error) {
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return nil, err
	}
	f := Filter{OrganizationID: c.OrganizationID, From: from, To: to}
	if !c.Admin {
		role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
			return nil, nil
		case err != nil:
			return nil, err
		}
		f.PractitionerRoleID = role.ID
	}
	viewer, err := s.staffViewer(ctx, c, f.PractitionerRoleID)
	if err != nil {
		return nil, err
	}
	all, err := s.Repo.Search(ctx, f)
	if err != nil {
		return nil, err
	}
	consents := map[string]access.Consent{}
	var out []Appointment
	for _, a := range all {
		state, ok := consents[a.PatientID]
		if !ok {
			if state, err = s.Consents.State(ctx, a.PatientID, c.OrganizationID); err != nil {
				return nil, err
			}
			consents[a.PatientID] = state
		}
		if access.Appointment(viewer, access.Facts{Patient: a.PatientID, Clinic: c.OrganizationID, Consent: state}, visit(a)) == access.Full {
			out = append(out, a)
		}
	}
	return out, nil
}

// flagged gives the flags of the appointments seen under the clinic's settings set: their patients' no-shows in the
// window up to now, counted in one query.
func (s *Service) flagged(ctx context.Context, org string, set organization.AppointmentSettings, seen []Appointment) ([]NoShowFlag, error) {
	if len(seen) == 0 {
		return nil, nil
	}
	known := map[string]bool{}
	var patients []string
	for _, a := range seen {
		if !known[a.PatientID] {
			known[a.PatientID] = true
			patients = append(patients, a.PatientID)
		}
	}
	now := s.Clock.Now()
	counts, err := s.NoShows.NoShowsSince(ctx, org, patients, now.AddDate(0, 0, -set.NoShowFlagWindowDays), now)
	if err != nil {
		return nil, err
	}
	var out []NoShowFlag
	for _, a := range seen {
		if n := counts[a.PatientID]; n >= set.NoShowFlagThreshold {
			out = append(out, NoShowFlag{AppointmentID: a.ID, NoShows: n, WindowDays: set.NoShowFlagWindowDays})
		}
	}
	return out, nil
}
