package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// PatientHistory returns the calling patient's own completed responses at every clinic, or at the clinic org when
// set, without their answers, after the page's position, newest first by submission, p.Limit at most: the
// questionnaire answers of their timeline (SAL-39), which the patient sees in full (access.QuestionnaireResponse).
// Never one in progress or withdrawn. Read in the patient's own scope and audited, each one returned as a search of
// their own request; none is one event without an entity. The timeline is the patient's: staff get ErrForbidden,
// audited. A patient without a profile has none.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Response, error) {
	var out []Response
	err := s.patientList(ctx, c, "patient history", func(ctx context.Context) ([]string, error) {
		all, err := s.Repo.CompletedOf(ctx, c.PatientID, org, p)
		out = ownCompleted(c, all, func(r Response) Response { return r })
		ids := make([]string, len(out))
		for i, r := range out {
			ids[i] = r.ID
		}
		return ids, err
	})
	return out, err
}

// PatientAttachments returns the files of the calling patient's own completed responses that are no longer
// pending (their scan found them clean), each with its response, after the page's position, newest first by the
// response's submission, p.Limit at most: the files of their timeline (SAL-39). Read and audited as
// PatientHistory, each file as a search of the response that lists it (binary audits the file itself).
func (s *Service) PatientAttachments(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Attached, error) {
	var out []Attached
	err := s.patientList(ctx, c, "patient attachments", func(ctx context.Context) ([]string, error) {
		all, err := s.Repo.FilesOf(ctx, c.PatientID, org, p)
		out = ownCompleted(c, all, func(a Attached) Response { return a.Response })
		ids := make([]string, len(out))
		for i, a := range out {
			ids[i] = a.Response.ID
		}
		return ids, err
	})
	return out, err
}

// patientList runs list in a transaction in the patient's own scope and audits each response id it gives as a
// search of their own request, or one event without an entity for none; staff get ErrForbidden, audited.
func (s *Service) patientList(ctx context.Context, c appointment.Caller, op string, list func(ctx context.Context) ([]string, error)) error {
	staff := c.OrganizationID != ""
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if staff {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		}
		var ids []string
		if c.PatientID != "" {
			if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
				return err
			}
			var err error
			if ids, err = list(ctx); err != nil {
				return err
			}
		}
		if len(ids) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, id := range ids {
			if err := s.record(ctx, c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	switch {
	case err != nil:
		return fmt.Errorf("questionnaireresponse: %s: %w", op, err)
	case staff:
		return ErrForbidden
	}
	return nil
}

// ownCompleted keeps the items whose response is the patient's own and completed, which
// access.QuestionnaireResponse shows them in full.
func ownCompleted[T any](c appointment.Caller, all []T, response func(T) Response) []T {
	viewer := access.Viewer{Role: access.RolePatient, Patient: c.PatientID}
	var out []T
	for _, x := range all {
		r := response(x)
		facts := access.Facts{Patient: r.PatientID, Clinic: r.OrganizationID}
		visit := access.Visit{PractitionerRole: r.PractitionerRoleID}
		if r.Status == StatusCompleted && access.QuestionnaireResponse(viewer, facts, visit, true) == access.Full {
			out = append(out, x)
		}
	}
	return out
}
