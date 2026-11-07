package questionnaireresponse

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// The Service is the holder of the files attached to its answers (binary.Holder, docs/attachments/SPEC.md §2
// "binary.Holder"): binary asks it who may upload, remove and read a response's files, and keeps the answer's
// list of files in step through it. Its checks are unaudited, because binary audits the call they serve, and
// run in binary's transaction (each WithTx joins it). The list lives in its own column, outside answers and
// version_id, so a file that finishes uploading never makes the patient's next save stale.
var _ binary.Holder = (*Service)(nil)

// UploadTarget is where the patient c may upload a file to the item linkID of their response id: the
// response's clinic, patient and appointment and the item's accepted kinds and number of files. The same
// conditions as a change of the answers (in progress, appointment booked and locked against a cancellation,
// not yet due, an active consent), and linkID an attachment item of the response's version (read through the
// questionnaire's service, which audits that read as the patient's). The refusals are binary's.
func (s *Service) UploadTarget(ctx context.Context, c binary.Caller, id, linkID string) (binary.Target, error) {
	var out binary.Target
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, refused, err := s.editable(ctx, caller(c), id)
		if err != nil || refused != nil {
			return errors.Join(err, refused)
		}
		q, err := s.Questionnaires.Referenced(ctx, questionnaire.Caller{AccountID: c.AccountID}, c.PatientID, r.QuestionnaireID)
		if err != nil {
			return err // the version a response references is always there
		}
		it, ok := index(q.Items)[linkID]
		if !ok || it.Type != questionnaire.ItemAttachment {
			return binary.ErrNotAttachmentItem
		}
		out = binary.Target{OrganizationID: r.OrganizationID, PatientID: r.PatientID, AppointmentID: r.AppointmentID, MaxFiles: int(it.MaxFiles)}
		for _, k := range it.Accept {
			out.Accept = append(out.Accept, binary.Kind(k))
		}
		return nil
	})
	if err != nil {
		return binary.Target{}, holderError(err)
	}
	return out, nil
}

// Editable is nil when the patient c may remove a file of their response id: UploadTarget's conditions
// without the item.
func (s *Service) Editable(ctx context.Context, c binary.Caller, id string) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		_, refused, err := s.editable(ctx, caller(c), id)
		return errors.Join(err, refused)
	})
	return holderError(err)
}

// editable reads the response id for a change of its files by c in the transaction in ctx, or gives the
// refusal: as for the answers, on the response's current version.
func (s *Service) editable(ctx context.Context, c appointment.Caller, id string) (Response, error, error) {
	read := s.Repo.Lock
	if c.OrganizationID != "" { // staff never change it: read to refuse, as the clinic after a revocation may not lock
		read = s.Repo.ByID
	}
	r, att, level, err := s.view(ctx, c, id, read)
	switch {
	case err != nil:
		return Response{}, nil, err
	case level == access.None:
		return Response{}, ErrNotFound, nil
	}
	refused, err := s.open(ctx, r, att, r.VersionID)
	return r, refused, err
}

// Readable is c's grant to read the files of the response id: the response's access rule at Full (the
// patient on their own; the appointment's practitioner once it is submitted, D9 included), with the purpose
// of use; binary.ErrNotFound for anyone else.
func (s *Service) Readable(ctx context.Context, c binary.Caller, id string) (binary.Grant, error) {
	var out binary.Grant
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		_, _, level, err := s.view(ctx, caller(c), id, s.Repo.ByID)
		switch {
		case err != nil:
			return err
		case level != access.Full:
			return ErrNotFound
		case c.OrganizationID != "":
			out.Purpose = auditevent.CodeTREAT
		default:
			out.Purpose = auditevent.CodePATRQT
		}
		return nil
	})
	if err != nil {
		return binary.Grant{}, holderError(err)
	}
	return out, nil
}

// Attach adds a to the files of the response id, in the scope of the caller's transaction.
func (s *Service) Attach(ctx context.Context, id string, a binary.Attachment) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.AddAttachment(ctx, id, a, s.Clock.Now()); err != nil {
			return fmt.Errorf("questionnaireresponse: attach: %w", err)
		}
		return nil
	})
}

// Detach removes the file binaryID from the files of the response id.
func (s *Service) Detach(ctx context.Context, id, binaryID string) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.RemoveAttachment(ctx, id, binaryID, s.Clock.Now()); err != nil {
			return fmt.Errorf("questionnaireresponse: detach: %w", err)
		}
		return nil
	})
}

// Cleared marks the file binaryID of the response id no longer pending.
func (s *Service) Cleared(ctx context.Context, id, binaryID string) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.ClearAttachment(ctx, id, binaryID, s.Clock.Now()); err != nil {
			return fmt.Errorf("questionnaireresponse: cleared: %w", err)
		}
		return nil
	})
}

// Attached lists the ids of the files of the response id.
func (s *Service) Attached(ctx context.Context, id string) ([]string, error) {
	var out []string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		files, err := s.Repo.Attachments(ctx, id)
		for _, a := range files {
			out = append(out, a.BinaryID)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("questionnaireresponse: attached: %w", err)
	}
	return out, nil
}

// caller is c as the appointment's rules see the caller.
func caller(c binary.Caller) appointment.Caller {
	return appointment.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin}
}

// holderErrors are binary's refusals for the response's own.
var holderErrors = []struct{ own, binary error }{
	{ErrNotFound, binary.ErrNotFound}, {ErrForbidden, binary.ErrForbidden}, {ErrSubmitted, binary.ErrSubmitted},
	{ErrWithdrawn, binary.ErrWithdrawn}, {ErrPastDue, binary.ErrPastDue}, {ErrAppointmentClosed, binary.ErrAppointmentClosed},
	{ErrConsentRequired, binary.ErrConsentRequired}, {binary.ErrNotAttachmentItem, binary.ErrNotAttachmentItem},
}

// holderError is err as binary takes it: a refusal as binary's error, anything else wrapped.
func holderError(err error) error {
	if err == nil {
		return nil
	}
	for _, e := range holderErrors {
		if errors.Is(err, e.own) {
			return e.binary
		}
	}
	return fmt.Errorf("questionnaireresponse: holder: %w", err)
}
