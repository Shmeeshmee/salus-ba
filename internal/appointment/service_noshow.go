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

// ClinicRules gives the clinic's appointment settings in the transaction in ctx, setting the clinic's scope
// (organization.Appointments): the repeat no-show flag's threshold and window.
type ClinicRules interface {
	For(ctx context.Context, org string) (organization.AppointmentSettings, error)
}

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
// Each flagged appointment's read is audited, or one event without an entity when none is. ErrInvalidRange for a
// range missing, reversed or longer than 62 days (nothing read); ErrNotFound, audited, to a patient: the flag is
// never theirs to see.
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
		if out, err = s.noShowFlags(ctx, c, set, from, to); err != nil {
			return err
		}
		for _, f := range out {
			if err := s.record(ctx, c, auditevent.CodeSearchType, "Appointment", f.AppointmentID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: no-show flags: %w", err)
	}
	return out, nil
}

// noShowFlags gives the flags of the range under the clinic's settings set, in the transaction in ctx: an admin's of
// every role, a practitioner's of their own role only.
func (s *Service) noShowFlags(ctx context.Context, c Caller, set organization.AppointmentSettings, from, to time.Time) ([]NoShowFlag, error) {
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
	since := s.Clock.Now().AddDate(0, 0, -set.NoShowFlagWindowDays)
	known := map[string]missed{}
	var out []NoShowFlag
	for _, a := range all {
		m, ok := known[a.PatientID]
		if !ok {
			if m, err = s.missed(ctx, a.PatientID, c.OrganizationID, since); err != nil {
				return nil, err
			}
			known[a.PatientID] = m
		}
		if m.noShows >= set.NoShowFlagThreshold && access.Appointment(viewer, m.facts, visit(a)) == access.Full {
			out = append(out, NoShowFlag{AppointmentID: a.ID, NoShows: m.noShows, WindowDays: set.NoShowFlagWindowDays})
		}
	}
	return out, nil
}

// missed is a patient's access facts in a clinic and how many of their appointments there since a time were no-shows.
type missed struct {
	facts   access.Facts
	noShows int
}

// missed gathers the patient's facts in the clinic org, as the scope shows them, and counts their noshow
// appointments there that started since since, up to now.
func (s *Service) missed(ctx context.Context, patientID, org string, since time.Time) (missed, error) {
	state, err := s.Consents.State(ctx, patientID, org)
	if err != nil {
		return missed{}, err
	}
	all, err := s.Repo.Search(ctx, Filter{PatientID: patientID, OrganizationID: org})
	if err != nil {
		return missed{}, err
	}
	out := missed{facts: access.Facts{Patient: patientID, Clinic: org, Consent: state, Visits: visits(all)}}
	now := s.Clock.Now()
	for _, a := range all {
		if a.Status == StatusNoShow && !a.Start.Before(since) && !a.Start.After(now) {
			out.noShows++
		}
	}
	return out, nil
}
