package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// agentRevocation names the consent_revoked job as the agent of its audit events.
const agentRevocation = "consent_revoked"

// view is an appointment with what the caller may see of it and the facts that decided it.
type view struct {
	viewer access.Viewer
	facts  access.Facts
	a      Appointment
	level  access.Level
}

// view reads the appointment id in the caller's scope, the patient's own or the clinic's, in the transaction
// in ctx, and decides the caller's level through access; an appointment the caller may not see is at None.
// The check does not rest on row-level security alone.
func (s *Service) view(ctx context.Context, c Caller, id string) (view, error) {
	if c.patient() {
		if c.PatientID == "" {
			return view{}, nil
		}
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return view{}, err
		}
	} else if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return view{}, err
	}
	a, err := s.Repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return view{}, nil
	}
	if err != nil {
		return view{}, err
	}
	if c.patient() {
		v := view{a: a, viewer: access.Viewer{Role: access.RolePatient, Patient: c.PatientID}, facts: access.Facts{Patient: a.PatientID, Clinic: a.OrganizationID}}
		v.level = access.Appointment(v.viewer, v.facts, visit(a))
		return v, nil
	}
	viewer, err := s.staffViewer(ctx, c, a.PractitionerRoleID)
	if err != nil {
		return view{}, err
	}
	facts, err := s.facts(ctx, a.PatientID, c.OrganizationID)
	if err != nil {
		return view{}, err
	}
	return view{a: a, viewer: viewer, facts: facts, level: access.Appointment(viewer, facts, visit(a))}, nil
}

// staffViewer is staff caller c as access sees them: the clinic's admin, or a practitioner whose role is
// roleID when the role is theirs (read through the role's service), and none otherwise.
func (s *Service) staffViewer(ctx context.Context, c Caller, roleID string) (access.Viewer, error) {
	if c.Admin {
		return access.Viewer{Role: access.RoleAdmin, Clinic: c.OrganizationID}, nil
	}
	v := access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID}
	if c.PractitionerID == "" || roleID == "" {
		return v, nil
	}
	role, err := s.Roles.Read(ctx, practitionerrole.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}, roleID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return v, nil
	case err != nil:
		return access.Viewer{}, err
	}
	if role.OrganizationID == c.OrganizationID && role.PractitionerID == c.PractitionerID {
		v.PractitionerRole = role.ID
	}
	return v, nil
}

// facts are the patient's consent state and appointments in the clinic, as the scope shows them.
func (s *Service) facts(ctx context.Context, patientID, org string) (access.Facts, error) {
	state, err := s.Consents.State(ctx, patientID, org)
	if err != nil {
		return access.Facts{}, err
	}
	all, err := s.Repo.Search(ctx, Filter{PatientID: patientID, OrganizationID: org})
	if err != nil {
		return access.Facts{}, err
	}
	return access.Facts{Patient: patientID, Clinic: org, Consent: state, Visits: visits(all)}, nil
}

func visit(a Appointment) access.Visit {
	return access.Visit{PractitionerRole: a.PractitionerRoleID, Start: a.Start}
}

func visits(all []Appointment) []access.Visit {
	out := make([]access.Visit, 0, len(all))
	for _, a := range all {
		out = append(out, visit(a))
	}
	return out
}

// Cancel cancels the booked appointment id and returns it: the patient their own until 24 hours before the
// start (reason pat), its practitioner and the clinic's admin at any time (prov), each with full access to
// it. In one transaction it frees the slot while its start is ahead, enqueues the cancellation e-mail, the
// end of the consultation's access (consultation_access_end) and the discard of its questionnaire responses
// (questionnaire_responses_discard), and audits. ErrNotFound for an appointment the
// caller may not see, ErrForbidden for one seen at the name only (a past one after a revocation), ErrClosed for
// one no longer booked, ErrCancellationWindowClosed for the patient inside the 24 hours; each refusal is
// audited.
func (s *Service) Cancel(ctx context.Context, c Caller, id string) (Appointment, error) {
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.cancel(ctx, c, id); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: cancel: %w", err)
	case refused != nil:
		return Appointment{}, refused
	}
	return out, nil
}

// cancel cancels the appointment id for c in the transaction in ctx, or gives the refusal.
func (s *Service) cancel(ctx context.Context, c Caller, id string) (out Appointment, refused, err error) {
	v, err := s.view(ctx, c, id)
	switch {
	case err != nil:
		return Appointment{}, nil, err
	case v.level == access.None:
		return Appointment{}, ErrNotFound, nil
	case v.level != access.Full:
		return Appointment{}, ErrForbidden, nil
	case v.a.Status != StatusBooked:
		return Appointment{}, ErrClosed, nil
	}
	now := s.Clock.Now()
	reason := ReasonProvider
	if c.patient() {
		if v.a.Start.Sub(now) < patientCancelBefore {
			return Appointment{}, ErrCancellationWindowClosed, nil
		}
		reason = ReasonPatient
		// The appointment is checked: its clinic is set for the slot's release.
		if err := s.Repo.SetClinic(ctx, v.a.OrganizationID); err != nil {
			return Appointment{}, nil, err
		}
	}
	out, err = s.Repo.Cancel(ctx, id, reason, c.AccountID, now)
	if errors.Is(err, ErrClosed) { // a concurrent change committed first
		return Appointment{}, ErrClosed, nil
	}
	if err != nil {
		return Appointment{}, nil, err
	}
	return out, nil, s.cancelled(ctx, out)
}

// cancelled frees the slot of the cancelled appointment a and enqueues its e-mail, the end of its
// consultation's access (consultation_access_end) and the discard of its questionnaire responses
// (questionnaire_responses_discard) and the refresh of the clinic's statistics, in its transaction. Every
// cancellation goes through it.
func (s *Service) cancelled(ctx context.Context, a Appointment) error {
	if _, err := s.Slots.Release(ctx, a.OrganizationID, a.SlotID); err != nil {
		return err
	}
	if err := s.Jobs.EnqueueCancelled(ctx, a.ID, a.OrganizationID); err != nil {
		return err
	}
	if err := s.Jobs.EnqueueAccessEnd(ctx, a.ID, a.OrganizationID); err != nil {
		return err
	}
	if err := s.Stats.EnqueueStatsRefresh(ctx, a.OrganizationID, a.Start); err != nil {
		return err
	}
	return s.Jobs.EnqueueQuestionnaireDiscard(ctx, a.ID, a.OrganizationID)
}

// SetStatus marks the booked appointment id fulfilled or noshow, by its practitioner, once its start has
// passed, and returns it; after a revocation too, for an appointment that started before it (the care
// record, access.RecordsOutcome). ErrInvalidStatus for any other status (before any access), ErrForbidden to
// anyone but its practitioner, ErrNotFound for one the caller may not see, ErrNotStarted before the start, ErrClosed for one no longer booked; each refusal but the first is
// audited.
func (s *Service) SetStatus(ctx context.Context, c Caller, id string, status Status) (Appointment, error) {
	if status != StatusFulfilled && status != StatusNoShow {
		return Appointment{}, ErrInvalidStatus
	}
	if c.patient() || c.Admin || c.PractitionerID == "" {
		return Appointment{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Appointment
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.setStatus(ctx, c, id, status); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Appointment{}, fmt.Errorf("appointment: set status: %w", err)
	case refused != nil:
		return Appointment{}, refused
	}
	return out, nil
}

// setStatus marks the appointment id in the transaction in ctx, or gives the refusal.
func (s *Service) setStatus(ctx context.Context, c Caller, id string, status Status) (out Appointment, refused, err error) {
	v, err := s.view(ctx, c, id)
	now := s.Clock.Now()
	switch {
	case err != nil:
		return Appointment{}, nil, err
	case v.level == access.None:
		return Appointment{}, ErrNotFound, nil
	case !access.RecordsOutcome(v.viewer, v.facts, visit(v.a)):
		return Appointment{}, ErrForbidden, nil
	case v.a.Status != StatusBooked:
		return Appointment{}, ErrClosed, nil
	case now.Before(v.a.Start):
		return Appointment{}, ErrNotStarted, nil
	}
	out, err = s.Repo.SetStatus(ctx, id, status, now)
	switch {
	case errors.Is(err, ErrClosed):
		return Appointment{}, ErrClosed, nil
	case err != nil:
		return Appointment{}, nil, err
	}
	return out, nil, s.Stats.EnqueueStatsRefresh(ctx, out.OrganizationID, out.Start)
}

// CancelForRevokedConsent cancels, after the patient revoked their consent with the clinic org (the
// consent_revoked job), every booked appointment of theirs there that starts after the latest revocation and
// was booked before it (an appointment booked after consenting again stays), inside the 24 hours too, with
// reason pat. In one transaction in the patient's scope with the clinic set: each is cancelled, its slot freed
// while ahead, its e-mail, the end of its consultation's access and the discard of its questionnaire responses
// enqueued and the change audited with the job as agent. It returns how many it
// cancelled; a retry finds none left. Nothing happens when the patient never revoked a consent there.
func (s *Service) CancelForRevokedConsent(ctx context.Context, patientID, org string) (int, error) {
	var n int
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		n = 0
		if err := s.Repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		state, err := s.Consents.State(ctx, patientID, org)
		if err != nil || state.RevokedAt.IsZero() {
			return err
		}
		all, err := s.Repo.Search(ctx, Filter{PatientID: patientID, OrganizationID: org, Status: StatusBooked})
		if err != nil {
			return err
		}
		now := s.Clock.Now()
		for _, a := range all {
			if a.PatientID != patientID || a.OrganizationID != org || a.Start.Before(state.RevokedAt) || a.Created.After(state.RevokedAt) {
				continue
			}
			out, err := s.Repo.Cancel(ctx, a.ID, ReasonPatient, "", now)
			if err != nil {
				return err
			}
			if err := s.cancelled(ctx, out); err != nil {
				return err
			}
			if err := s.recordJob(ctx, agentRevocation, org, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("appointment: cancel for revoked consent: %w", err)
	}
	return n, nil
}
