package binary

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Holder is the owner of the files: the questionnaire response whose answers hold them
// (questionnaireresponse.Service, docs/attachments/SPEC.md §2 "binary.Holder"). Its checks are unaudited,
// because binary audits the call they serve; they and its list of attachments run in binary's transaction.
// A refusal is one of this package's errors.
type Holder interface {
	// UploadTarget is where the caller may upload a file to the item linkID of the response, or the refusal:
	// ErrNotFound, ErrForbidden, ErrSubmitted, ErrWithdrawn, ErrPastDue, ErrAppointmentClosed,
	// ErrConsentRequired, ErrNotAttachmentItem.
	UploadTarget(ctx context.Context, c Caller, responseID, linkID string) (Target, error)
	// Editable is nil when the caller may remove a file of the response, or UploadTarget's refusal.
	Editable(ctx context.Context, c Caller, responseID string) error
	// Readable is the caller's grant to read the response's files, or ErrNotFound.
	Readable(ctx context.Context, c Caller, responseID string) (Grant, error)
	// Attach adds a to the response's attachments.
	Attach(ctx context.Context, responseID string, a Attachment) error
	// Detach removes the file binaryID from the response's attachments.
	Detach(ctx context.Context, responseID, binaryID string) error
	// Cleared marks the file binaryID of the response no longer pending.
	Cleared(ctx context.Context, responseID, binaryID string) error
	// Attached lists the ids of the files the response's attachments hold.
	Attached(ctx context.Context, responseID string) ([]string, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo    Repository
	Store   Store
	Holder  Holder
	Audit   Auditor
	Tx      Transactor
	Jobs    Jobs
	Scanner Scanner // the scan job's; nil in the api, which scans nothing
	Clock   clock.Clock
	KEK     *envelope.KEK // wraps each file's data key
}

// Service holds the files' rules (docs/attachments/SPEC.md §2).
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	return &Service{Ports: p}
}

// scope scopes the transaction in ctx to the caller: a patient's own files, or the staff caller's clinic's;
// false for a patient without a profile, who has none.
func (s *Service) scope(ctx context.Context, c Caller) (bool, error) {
	if c.OrganizationID != "" {
		return true, s.Repo.SetClinic(ctx, c.OrganizationID)
	}
	if c.PatientID == "" {
		return false, nil
	}
	return true, s.Repo.SetPatient(ctx, c.PatientID)
}

// seen reads the file id in the caller's scope in the transaction in ctx; ErrNotFound when the caller sees
// no such file.
func (s *Service) seen(ctx context.Context, c Caller, id string) (Binary, error) {
	ok, err := s.scope(ctx, c)
	if err != nil {
		return Binary{}, err
	}
	if !ok {
		return Binary{}, ErrNotFound
	}
	return s.Repo.ByID(ctx, id)
}

// Status returns the file id to whoever may read its response in full (Holder.Readable), so the app can show
// that it is still being checked. Read in the caller's scope and audited, with the file and its response as
// entities; ErrNotFound to anyone else, audited without them.
func (s *Service) Status(ctx context.Context, c Caller, id string) (Binary, error) {
	out, _, err := s.read(ctx, c, id, func(Binary) error { return nil })
	if err != nil {
		return Binary{}, fmt.Errorf("binary: status: %w", err)
	}
	return out, nil
}

// read runs a read of the file id by the caller in one transaction: the row in the caller's scope, the
// holder's grant, the file still listed by its response, then check, whose refusal is audited with the file as entity; the grant's purpose.
func (s *Service) read(ctx context.Context, c Caller, id string, check func(Binary) error) (Binary, Grant, error) {
	var out Binary
	var g Grant
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		b, err := s.seen(ctx, c, id)
		if err == nil {
			g, err = s.Holder.Readable(ctx, c, b.ResponseID)
		}
		if err == nil {
			err = s.listed(ctx, b)
		}
		switch {
		case errors.Is(err, ErrNotFound):
			refused = ErrNotFound
			return s.record(ctx, entry{c: c, org: c.OrganizationID, interaction: auditevent.CodeRead, outcome: auditevent.OutcomeMinorFailure})
		case err != nil:
			return err
		}
		// Under the staff reader's clinic, also for a file of another clinic's visit shared with it (ADR 0014).
		e := entry{c: c, org: cmp.Or(c.OrganizationID, b.OrganizationID), interaction: auditevent.CodeRead, purpose: g.Purpose, binaryID: b.ID, responseID: b.ResponseID}
		if refused = check(b); refused != nil {
			e.outcome = auditevent.OutcomeMinorFailure
			return s.record(ctx, e)
		}
		out = b
		return s.record(ctx, e)
	})
	switch {
	case err != nil:
		return Binary{}, Grant{}, err
	case refused != nil:
		return Binary{}, Grant{}, refused
	}
	return out, g, nil
}

// listed is ErrNotFound when b's response no longer lists it (its answer dropped on submit, or the response
// withdrawn): such a file waits for binary_discard and is read by nobody. A rejected file is detached by its scan
// and has no key or object left: its status stays readable, so the patient's app can show M7, and Download refuses
// it anyway because it is not clean.
func (s *Service) listed(ctx context.Context, b Binary) error {
	if b.Status == StatusRejected {
		return nil
	}
	ids, err := s.Holder.Attached(ctx, b.ResponseID)
	switch {
	case err != nil:
		return err
	case !slices.Contains(ids, b.ID):
		return ErrNotFound
	}
	return nil
}

// Delete deletes the file id, as the patient while they may still change the response's answers
// (Holder.Editable): its row and its place in the answer in one transaction, audited with the file and its
// response as entities; the object after the commit (best effort: the purger removes what is left).
// ErrNotFound to anyone who does not see it, the holder's refusals otherwise; each refusal is audited.
func (s *Service) Delete(ctx context.Context, c Caller, id string) error {
	var refused error
	var deleted string // the row's canonical id, the object's key: id may be in another case
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		b, err := s.seen(ctx, c, id)
		if err == nil {
			err = s.Holder.Editable(ctx, c, b.ResponseID)
		}
		e := entry{c: c, org: c.OrganizationID, interaction: auditevent.CodeDelete, outcome: auditevent.OutcomeMinorFailure}
		switch {
		case errors.Is(err, ErrNotFound):
			refused = ErrNotFound
			return s.record(ctx, e)
		case refusal(err):
			refused = err
			e.org, e.binaryID, e.responseID = b.OrganizationID, b.ID, b.ResponseID
			return s.record(ctx, e)
		case err != nil:
			return err
		}
		ok, err := s.Repo.Delete(ctx, b.ID)
		switch {
		case err != nil:
			return err
		case !ok:
			return errors.New("binary: delete: the row was not deleted")
		}
		if err := s.Holder.Detach(ctx, b.ResponseID, b.ID); err != nil {
			return err
		}
		deleted = b.ID
		e.org, e.binaryID, e.responseID, e.outcome = b.OrganizationID, b.ID, b.ResponseID, auditevent.OutcomeSuccess
		return s.record(ctx, e)
	})
	switch {
	case err != nil:
		return fmt.Errorf("binary: delete: %w", err)
	case refused != nil:
		return refused
	}
	_ = s.Store.Delete(context.WithoutCancel(ctx), deleted) // the purger removes an object left behind (SPEC.md §5)
	return nil
}

// entry is one audited call: interaction by c, or by the job agent, with outcome, on the file and its
// response when they are disclosed, in the clinic org. The purpose is the grant's, or else the caller's: the
// patient's own request for a patient, treatment for staff, operations for a job.
type entry struct {
	agent       string // a job's kind; empty for a caller
	org         string
	interaction string
	purpose     string
	binaryID    string
	responseID  string
	c           Caller
	outcome     auditevent.Outcome
}

// record audits e: one event per entity (AuditEvent holds one), or one without an entity when none is
// disclosed.
func (s *Service) record(ctx context.Context, e entry) error {
	purpose := e.purpose
	switch {
	case purpose != "":
	case e.agent != "":
		purpose = auditevent.CodeHOPERAT
	case e.c.OrganizationID != "":
		purpose = auditevent.CodeTREAT
	default:
		purpose = auditevent.CodePATRQT
	}
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeDelete: auditevent.ActionDelete,
		auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[e.interaction]
	if action == "" {
		action = auditevent.ActionRead
	}
	var entities []string
	if e.binaryID != "" {
		entities = append(entities, "Binary/"+e.binaryID)
	}
	if e.responseID != "" {
		entities = append(entities, "QuestionnaireResponse/"+e.responseID)
	}
	if len(entities) == 0 {
		entities = []string{""}
	}
	for _, entity := range entities {
		err := s.Audit.Record(ctx, auditevent.AuditEvent{
			Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: e.interaction},
			Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
			Action:          action,
			Outcome:         e.outcome,
			AgentAccountID:  e.c.AccountID,
			AgentName:       e.agent,
			OrganizationID:  e.org,
			EntityReference: entity,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
