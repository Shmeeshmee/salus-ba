package questionnaireresponse

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// History returns the completed responses of the visits' appointments, with their scanned files only, in the order
// of visits: for the practitioner's view of the patient's history (SAL-40), with visits exactly as
// encounter.Service.History gave them to the caller (access.HistoryVisit applied). Never one in progress. Read in the
// caller's clinic scope (a shared visit's through questionnaire_response_shared) and unaudited: the history audits the
// ones it shows (Disclosed). ErrNotFound for anyone but a practitioner.
func (s *Service) History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]Response, error) {
	if c.OrganizationID == "" || c.Admin {
		return nil, ErrNotFound
	}
	var out []Response
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		// One search per patient, not per visit (PROGRESS, SAL-40): the visits' appointments are picked from it.
		byAppointment := map[string][]Response{}
		patients := map[string]bool{}
		for _, v := range visits {
			if patients[v.PatientID] {
				continue
			}
			patients[v.PatientID] = true
			rs, err := s.Repo.Search(ctx, Filter{PatientID: v.PatientID, Status: StatusCompleted})
			if err != nil {
				return err
			}
			for _, r := range rs {
				byAppointment[r.AppointmentID] = append(byAppointment[r.AppointmentID], r)
			}
		}
		for _, v := range visits {
			for _, r := range byAppointment[v.AppointmentID] {
				if r.PatientID == v.PatientID {
					r.Attachments = scanned(r.Attachments)
					out = append(out, r)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("questionnaireresponse: history: %w", err)
	}
	return out, nil
}

// Disclosed audits the responses ids the history shows the caller (with their files), each as a search of it, in
// the transaction in ctx (or one of its own).
func (s *Service) Disclosed(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.record(ctx, c, auditevent.CodeSearchType, id, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: disclosed: %w", err)
	}
	return nil
}

// DisclosedFiles audits the files ids of the responses the history shows the caller, each as a search of its
// Binary, in the transaction in ctx (or one of its own).
func (s *Service) DisclosedFiles(ctx context.Context, c appointment.Caller, ids []string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.recordEntity(ctx, c, auditevent.CodeSearchType, "Binary/"+id, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("questionnaireresponse: disclosed files: %w", err)
	}
	return nil
}

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
