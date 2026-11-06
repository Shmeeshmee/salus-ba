package questionnaireresponse

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// The jobs that request and discard responses, as the agents of their audit events (their job kinds).
const (
	agentDefaultRequest = "questionnaire_default_request"
	agentDiscard        = "questionnaire_responses_discard"
)

// RequestDefault is the questionnaire_default_request job, enqueued with every booking of an appointment of
// the clinic org (SPEC.md §2 "Requests"): it requests the active version of the default questionnaire of the
// appointment's role (source default), once. In one transaction in the clinic's scope, which shows a future
// appointment only while the patient's consent there is active: an appointment it does not show, one no
// longer booked or already started, a role without a default or a default without an active version, and a
// series the appointment already has a live request of, are left as they are. The request is audited as a
// create with the job as agent (the read of the default by the questionnaire's service). Repeated, it changes
// nothing.
func (s *Service) RequestDefault(ctx context.Context, appointmentID, org string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		now := s.Clock.Now()
		a, err := s.Appointments.Lookup(ctx, appointmentID)
		switch {
		case errors.Is(err, appointment.ErrNotFound):
			return nil
		case err != nil:
			return err
		case a.OrganizationID != org || a.Status != appointment.StatusBooked || !now.Before(a.Start):
			return nil
		}
		q, err := s.Questionnaires.ActiveDefault(ctx, agentDefaultRequest, org, a.PractitionerRoleID)
		switch {
		case errors.Is(err, questionnaire.ErrNotFound), errors.Is(err, questionnaire.ErrNotActive):
			return nil
		case err != nil:
			return err
		}
		out, created, err := s.Repo.Insert(ctx, Response{
			OrganizationID: org, AppointmentID: a.ID, PatientID: a.PatientID, PractitionerRoleID: a.PractitionerRoleID,
			QuestionnaireID: q.ID, SeriesID: q.SeriesID, Source: SourceDefault, DueAt: a.Start,
		}, now)
		if err != nil || !created {
			return err
		}
		return s.recordJob(ctx, agentDefaultRequest, org, auditevent.CodeCreate, auditevent.ActionCreate, out.ID)
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: request default: %w", err)
	}
	return nil
}

// Discard is the questionnaire_responses_discard job, enqueued with every cancellation of an appointment of the
// clinic org, the revocation's included (SPEC.md §2 "Cancellation and revocation", D8): once the appointment is
// cancelled, it deletes every response of it, submitted or not. In one transaction in the clinic's scope with
// the appointment's notice, which shows the cancelled appointment after a revocation too; each response
// deleted is audited as a delete with the job as agent. An appointment not cancelled (or not shown) is left as
// it is; repeated, it changes nothing.
func (s *Service) Discard(ctx context.Context, appointmentID, org string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		if err := s.Repo.SetNotice(ctx, appointmentID); err != nil {
			return err
		}
		a, err := s.Appointments.Lookup(ctx, appointmentID)
		switch {
		case errors.Is(err, appointment.ErrNotFound):
			return nil
		case err != nil:
			return err
		case a.OrganizationID != org || a.Status != appointment.StatusCancelled:
			return nil
		}
		ids, err := s.Repo.DeleteOfAppointment(ctx, a.ID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := s.recordJob(ctx, agentDiscard, org, auditevent.CodeDelete, auditevent.ActionDelete, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: discard: %w", err)
	}
	return nil
}

// recordJob audits interaction on QuestionnaireResponse/id ("" for none) by the job agent (its kind) in the
// clinic org, for operations.
func (s *Service) recordJob(ctx context.Context, agent, org, interaction string, action auditevent.Action, id string) error {
	entity := ""
	if id != "" {
		entity = "QuestionnaireResponse/" + id
	}
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		OrganizationID:  org,
		EntityReference: entity,
	})
}
