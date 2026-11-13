package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// PatientHistory returns the calling patient's own completed responses at every clinic, or at the clinic org when
// set, without their answers, after the page's position, newest first by submission, p.Limit at most: the
// questionnaire answers of their timeline (SAL-39), which the patient sees in full (access.QuestionnaireResponse).
// Never one in progress or withdrawn. Read in the patient's own scope, joining the transaction in ctx, unaudited:
// history.Timeline audits the items of the page it returns. The timeline is the patient's: staff get ErrForbidden.
// A patient without a profile has none.
func (s *Service) PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Response, error) {
	var out []Response
	err := s.patientList(ctx, c, "patient history", func(ctx context.Context) error {
		all, err := s.Repo.CompletedOf(ctx, c.PatientID, org, p)
		out = ownCompleted(c, all, func(r Response) Response { return r })
		return err
	})
	return out, err
}

// PatientAttachments returns the files of the calling patient's own completed responses that are no longer
// pending (their scan found them clean), each with its response, after the page's position, newest first by the
// response's submission, p.Limit at most: the files of their timeline (SAL-39). Read as PatientHistory, unaudited.
func (s *Service) PatientAttachments(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]Attached, error) {
	var out []Attached
	err := s.patientList(ctx, c, "patient attachments", func(ctx context.Context) error {
		all, err := s.Repo.FilesOf(ctx, c.PatientID, org, p)
		out = ownCompleted(c, all, func(a Attached) Response { return a.Response })
		return err
	})
	return out, err
}

// patientList runs list in the patient's own scope, joining the transaction in ctx; staff get ErrForbidden and a
// patient without a profile nothing.
func (s *Service) patientList(ctx context.Context, c appointment.Caller, op string, list func(ctx context.Context) error) error {
	switch {
	case c.OrganizationID != "":
		return ErrForbidden
	case c.PatientID == "":
		return nil
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		return list(ctx)
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: %s: %w", op, err)
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
