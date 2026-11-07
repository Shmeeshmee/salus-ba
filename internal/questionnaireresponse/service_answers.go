package questionnaireresponse

import (
	"context"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// SaveAnswers replaces the saved answers of the response id with answers, as its patient (SPEC.md §2
// "Patient flow"), and returns it at its next version. Each answer is checked against its item's type and
// limits; conditions and required questions are checked on submit. The response must be in progress, not yet
// due, its appointment booked, its patient's consent with the clinic active, and version (If-Match) its current
// version. In one transaction in the patient's scope, which locks the appointment against a cancellation (a
// cancellation that committed first refuses the change; one that comes later waits, and its discard job
// deletes the response), audited as an update beside the patient's audited read of the questionnaire version.
// ErrNotFound to anyone who may not see it, ErrForbidden to anyone else who sees it (staff), ErrSubmitted,
// ErrWithdrawn, ErrAppointmentClosed, ErrPastDue, ErrConsentRequired, ErrConflict, an *AnswersError; each
// refusal is audited.
func (s *Service) SaveAnswers(ctx context.Context, c appointment.Caller, id string, answers []Answer, version int32) (Response, error) {
	return s.answer(ctx, c, id, version, "save answers", func(ctx context.Context, r Response, items []questionnaire.Item) (Response, error, error) {
		if refused := checkAnswers(items, answers); refused != nil {
			return Response{}, refused, nil
		}
		out, err := s.Repo.SaveAnswers(ctx, r.ID, answers, s.Clock.Now())
		return out, nil, err
	})
}

// Submit submits the response id with its saved answers, as its patient, and returns it: completed, authored
// at the clock's time, final. The saved answers are checked in full: each against its item, and every enabled
// required question answered, a required attachment item with a file (an *AnswersError names the rest); then
// no file may be pending its scan (ErrAttachmentsPending, docs/attachments/SPEC.md §2 "Submit"). The answers of
// hidden items are removed, and their files detached with the binary_discard job enqueued in the transaction.
// The same conditions, transaction, audit and refusals as SaveAnswers.
func (s *Service) Submit(ctx context.Context, c appointment.Caller, id string, version int32) (Response, error) {
	return s.answer(ctx, c, id, version, "submit", func(ctx context.Context, r Response, items []questionnaire.Item) (Response, error, error) {
		kept, hidden, refused := submitted(items, r.Answers, r.Attachments)
		if refused == nil && slices.ContainsFunc(r.Attachments, func(a binary.Attachment) bool { return a.Pending }) {
			refused = ErrAttachmentsPending
		}
		if refused != nil {
			return Response{}, refused, nil
		}
		now := s.Clock.Now()
		for _, file := range hidden {
			if err := s.Repo.RemoveAttachment(ctx, r.ID, file, now); err != nil {
				return Response{}, nil, err
			}
		}
		out, err := s.Repo.Complete(ctx, r.ID, kept, now)
		if err != nil || len(hidden) == 0 {
			return out, nil, err
		}
		return out, nil, s.Jobs.EnqueueDiscard(ctx, binary.DiscardArgs{ResponseID: r.ID, OrganizationID: r.OrganizationID})
	})
}

// change is the work of a patient's change of their open response r, given the items of its version: the
// response changed, a refusal (audited, committed), or any other error (rolled back).
type change func(ctx context.Context, r Response, items []questionnaire.Item) (out Response, refused, err error)

// answer runs fn on the response id for its patient c in one transaction, after the checks every change of
// the answers shares, and audits it.
func (s *Service) answer(ctx context.Context, c appointment.Caller, id string, version int32, name string, fn change) (Response, error) {
	var out Response
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		read := s.Repo.Lock
		if c.OrganizationID != "" { // staff never change the answers: read to refuse, as the clinic after a revocation may not lock
			read = s.Repo.ByID
		}
		r, att, level, err := s.view(ctx, c, id, read)
		switch {
		case err != nil:
			return err
		case level == access.None:
			refused = ErrNotFound
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		if refused, err = s.open(ctx, r, att, version); err != nil {
			return err
		}
		if refused == nil {
			q, err := s.Questionnaires.Referenced(ctx, questionnaire.Caller{AccountID: c.AccountID}, c.PatientID, r.QuestionnaireID)
			if err != nil {
				return err // the version a response references is always there
			}
			if out, refused, err = fn(ctx, r, q.Items); err != nil {
				return err
			}
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, r.ID, auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeUpdate, r.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Response{}, fmt.Errorf("questionnaireresponse: %s: %w", name, err)
	case refused != nil:
		return Response{}, refused
	}
	return out, nil
}

// open gives the refusal of a change of r's answers by the caller whose attendance is att, or nil: only the
// patient changes them, while in progress, its appointment booked (locked in the transaction in ctx against a
// cancellation) and not yet due, with an active consent, on the current version.
func (s *Service) open(ctx context.Context, r Response, att appointment.Attendance, version int32) (refused, err error) {
	switch {
	case !att.Patient:
		return ErrForbidden, nil
	case r.Status == StatusCompleted:
		return ErrSubmitted, nil
	case r.Status == StatusStopped:
		return ErrWithdrawn, nil
	}
	if _, refused, err = s.booked(ctx, r.AppointmentID); err != nil || refused != nil {
		return refused, err
	}
	switch {
	case !s.Clock.Now().Before(r.DueAt):
		return ErrPastDue, nil
	case !att.ConsentActive:
		return ErrConsentRequired, nil
	case r.VersionID != version:
		return ErrConflict, nil
	}
	return nil, nil
}

// submitted gives the answers a submit keeps, those of the enabled items, and the ids of the files of hidden
// items, or an *AnswersError naming, in document order, the answers that break their item and the enabled
// required questions left unanswered (an attachment item is answered by its files).
func submitted(items []questionnaire.Item, answers []Answer, files []binary.Attachment) (kept []Answer, hidden []string, err error) {
	bad, unknown := invalid(items, answers)
	given := make(map[string][]Value, len(answers))
	for _, a := range answers {
		given[a.LinkID] = a.Values
	}
	attached := make(map[string]bool, len(files))
	for _, f := range files {
		attached[f.LinkID] = true
	}
	on := enabled(items, given)
	for id, it := range index(items) {
		if it.Required && on[id] && len(given[id]) == 0 && !attached[id] {
			bad[id] = true
		}
	}
	if len(bad) > 0 || unknown {
		return nil, nil, &AnswersError{LinkIDs: inOrder(items, bad)}
	}
	for _, a := range answers {
		if on[a.LinkID] {
			kept = append(kept, a)
		}
	}
	for _, f := range files {
		if !on[f.LinkID] {
			hidden = append(hidden, f.BinaryID)
		}
	}
	return kept, hidden, nil
}
