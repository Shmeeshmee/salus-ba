package consent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
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

// Jobs enqueues background work in the transaction in ctx.
type Jobs interface {
	// EnqueueConsentRevoked enqueues consent_revoked with the two ids.
	EnqueueConsentRevoked(ctx context.Context, patientID, organizationID string) error
	// EnqueueTranscriptionRevoked enqueues transcription_consent_revoked with the two ids.
	EnqueueTranscriptionRevoked(ctx context.Context, patientID, organizationID string) error
}

// errNoProfile is a patient without a profile, mapped by each method to its own refusal.
var errNoProfile = errors.New("consent: no profile")

// Service holds the consent rules: a patient grants, revokes and lists their own consents; a consent is
// read by its patient and by staff of its clinic; every call is audited, also when it is refused.
// Revocation ends the clinic's access when its transaction commits, because row-level security and access
// read the consent's state; the job it enqueues cancels what was booked ahead. The package knows nothing of
// appointments.
type Service struct {
	repo  Repository
	audit Auditor
	tx    Transactor
	clock clock.Clock
	jobs  Jobs
}

// NewService returns a Service on its ports. jobs may be nil where Revoke is never called (the worker, which
// asks State only).
func NewService(repo Repository, audit Auditor, tx Transactor, clk clock.Clock, jobs Jobs) *Service {
	return &Service{repo: repo, audit: audit, tx: tx, clock: clk, jobs: jobs}
}

// Grant gives the clinic organizationID the calling patient's consent of the kind and returns it. ErrActive when
// one of the kind is active already, ErrProfileRequired without a profile, ErrClinicNotFound for an unknown
// clinic, ErrKind for an unknown kind, ErrForbidden to staff.
func (s *Service) Grant(ctx context.Context, c Caller, organizationID string, kind Kind) (Consent, error) {
	switch {
	case !c.patient():
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	case !kind.Valid():
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrKind)
	}
	var out Consent
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if errors.Is(err, errNoProfile) {
			refused = ErrProfileRequired
			return s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		out, err = s.repo.Grant(ctx, kind, patientID, organizationID, s.clock.Now())
		if errors.Is(err, ErrActive) {
			refused = ErrActive
			return s.record(ctx, c, auditevent.CodeCreate, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case errors.Is(err, ErrClinicNotFound):
		// The insert aborted the transaction: the refusal is audited in a transaction of its own.
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrClinicNotFound)
	case err != nil:
		return Consent{}, fmt.Errorf("consent: grant: %w", err)
	case refused != nil:
		return Consent{}, refused
	}
	return out, nil
}

// Revoke ends the calling patient's active consent id and returns it. A care consent's revocation also ends
// the patient's transcription consent with the clinic and enqueues consent_revoked (the patient and the clinic)
// in the same transaction: from its commit the clinic sees no more of the patient than SPEC.md "Past and future
// after revocation" leaves it, and the job cancels the future appointments there and removes the patient, and
// the agent with them, from the call. A transcription consent's withdrawal enqueues
// transcription_consent_revoked instead: from its commit the agent may not join or stay, the job takes it out
// of the call, and the call goes on. ErrNotFound for a consent that is not the caller's, ErrInactive for one
// revoked already, ErrForbidden to staff.
func (s *Service) Revoke(ctx context.Context, c Caller, id string) (Consent, error) {
	if !c.patient() {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	var out Consent
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.revoke(ctx, c, id); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		}
		if err := s.revoked(ctx, c, out); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Consent{}, fmt.Errorf("consent: revoke: %w", err)
	case refused != nil:
		return Consent{}, refused
	}
	return out, nil
}

// revoked does what follows the revocation of consent x, in the transaction in ctx: for a transcription
// consent, its job; for a care consent, the end of the transcription consent with the clinic (audited) and its
// job.
func (s *Service) revoked(ctx context.Context, c Caller, x Consent) error {
	if x.Kind == KindTranscription {
		return s.jobs.EnqueueTranscriptionRevoked(ctx, x.PatientID, x.OrganizationID)
	}
	t, err := s.repo.RevokeTranscription(ctx, x.PatientID, x.OrganizationID, s.clock.Now())
	switch {
	case errors.Is(err, ErrInactive):
	case err != nil:
		return err
	default:
		if err := s.record(ctx, c, auditevent.CodeUpdate, t.ID, auditevent.OutcomeSuccess); err != nil {
			return err
		}
	}
	return s.jobs.EnqueueConsentRevoked(ctx, x.PatientID, x.OrganizationID)
}

// revoke revokes the calling patient's consent id in the transaction in ctx, or gives the refusal.
func (s *Service) revoke(ctx context.Context, c Caller, id string) (out Consent, refused, err error) {
	patientID, err := s.ownPatient(ctx, c)
	if errors.Is(err, errNoProfile) {
		return Consent{}, ErrNotFound, nil
	}
	if err != nil {
		return Consent{}, nil, err
	}
	cur, err := s.repo.ByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound), err == nil && cur.PatientID != patientID: // not on row-level security alone
		return Consent{}, ErrNotFound, nil
	case err != nil:
		return Consent{}, nil, err
	case cur.Status != StatusActive:
		return Consent{}, ErrInactive, nil
	}
	out, err = s.repo.Revoke(ctx, cur.Kind, id, s.clock.Now())
	if errors.Is(err, ErrInactive) { // a concurrent revocation committed first
		return Consent{}, ErrInactive, nil
	}
	return out, nil, err
}

// Mine returns the calling patient's consents, the latest granted first, and an empty list without a
// profile. Each consent returned is audited; a list of none writes one event without an entity.
// ErrForbidden to staff.
func (s *Service) Mine(ctx context.Context, c Caller) ([]Consent, error) {
	if !c.patient() {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	out := []Consent{}
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if err != nil && !errors.Is(err, errNoProfile) {
			return err
		}
		if err == nil {
			all, err := s.repo.ByPatient(ctx, patientID)
			if err != nil {
				return err
			}
			for _, x := range all {
				if x.PatientID == patientID { // not on row-level security alone
					out = append(out, x)
				}
			}
		}
		if len(out) == 0 {
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		for _, x := range out {
			if err := s.record(ctx, c, auditevent.CodeSearchType, x.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("consent: mine: %w", err)
	}
	return out, nil
}

// Read returns the consent id to its patient and to staff of its clinic, and ErrNotFound to anyone else.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Consent, error) {
	var out Consent
	var found bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, found, err = s.read(ctx, c, id); err != nil {
			return err
		}
		if !found {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		return s.record(ctx, c, auditevent.CodeRead, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Consent{}, fmt.Errorf("consent: read: %w", err)
	case !found:
		return Consent{}, ErrNotFound
	}
	return out, nil
}

// read reads the consent id in the caller's scope, the patient's own or the clinic's, in the transaction in
// ctx, and reports whether the caller may see it. The check does not rest on row-level security alone.
func (s *Service) read(ctx context.Context, c Caller, id string) (Consent, bool, error) {
	var patientID string
	if c.patient() {
		var err error
		patientID, err = s.ownPatient(ctx, c)
		if errors.Is(err, errNoProfile) {
			return Consent{}, false, nil
		}
		if err != nil {
			return Consent{}, false, err
		}
	} else if err := s.repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return Consent{}, false, err
	}
	out, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Consent{}, false, nil
	}
	if err != nil {
		return Consent{}, false, err
	}
	if c.patient() {
		return out, out.PatientID == patientID, nil
	}
	return out, out.OrganizationID == c.OrganizationID, nil
}

// State returns whether the patient has an active care consent with the clinic, its latest revocation, and
// whether their transcription consent there is active, as access needs them. It runs in the caller's
// transaction and scope (a clinic, or the patient) and is not audited: it is the access check inside the
// caller's audited access, and discloses no consent to anyone.
func (s *Service) State(ctx context.Context, patientID, organizationID string) (access.Consent, error) {
	st, err := s.repo.State(ctx, patientID, organizationID)
	if err != nil {
		return access.Consent{}, fmt.Errorf("consent: state: %w", err)
	}
	return st, nil
}

// ownPatient scopes the transaction to the calling patient and returns their id; errNoProfile when they have
// no profile yet.
func (s *Service) ownPatient(ctx context.Context, c Caller) (string, error) {
	if c.PatientID == "" {
		return "", errNoProfile
	}
	return c.PatientID, s.repo.SetPatient(ctx, c.PatientID)
}

// refuse audits a refused interaction by c in a transaction of its own and returns refusal.
func (s *Service) refuse(ctx context.Context, c Caller, interaction string, refusal error) error {
	if err := s.record(ctx, c, interaction, "", auditevent.OutcomeMinorFailure); err != nil {
		return fmt.Errorf("consent: %s: %w", interaction, err)
	}
	return refusal
}

// record audits interaction by c on Consent/id ("" when none was disclosed: no id from the request is
// stored). The purpose is the patient's own request for a patient and treatment for staff; a search is an
// execute (FHIR AuditEvent.action).
func (s *Service) record(ctx context.Context, c Caller, interaction, id string, outcome auditevent.Outcome) error {
	action := auditevent.ActionRead
	switch interaction {
	case auditevent.CodeCreate:
		action = auditevent.ActionCreate
	case auditevent.CodeUpdate:
		action = auditevent.ActionUpdate
	case auditevent.CodeSearchType:
		action = auditevent.ActionExecute
	}
	purpose := auditevent.CodePATRQT
	if !c.patient() {
		purpose = auditevent.CodeTREAT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = "Consent/" + id
	}
	return s.audit.Record(ctx, e)
}

// GrantChat records the calling patient's consent that the practitioner role of another clinic invited into their
// conversation reads and writes in it (KindChat, SAL-28), and returns it. It runs in the caller's transaction (the
// invitation's decision) and ends an earlier active one for the same conversation and role first (the doctor was
// removed and invited again), so at most one is active; each change is audited. ErrForbidden to staff,
// ErrProfileRequired without a profile. That the conversation and the invitation are the patient's is the caller's
// check; the database refuses a conversation of another patient or clinic.
func (s *Service) GrantChat(ctx context.Context, c Caller, g ChatGrant) (Consent, error) {
	if !c.patient() {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	if c.PatientID == "" {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrProfileRequired)
	}
	var out Consent
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		patientID, err := s.ownPatient(ctx, c)
		if err != nil {
			return err
		}
		earlier, err := s.repo.RevokeChat(ctx, g.ConversationID, g.PractitionerRoleID, s.clock.Now())
		switch {
		case err == nil:
			if err := s.record(ctx, c, auditevent.CodeUpdate, earlier.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		case !errors.Is(err, ErrInactive):
			return err
		}
		if out, err = s.repo.GrantChat(ctx, patientID, g, s.clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess)
	})
	if err != nil {
		return Consent{}, fmt.Errorf("consent: grant chat: %w", err)
	}
	return out, nil
}

// RevokeChat ends the calling patient's active chat consent for the conversation and role and returns it, audited;
// from its commit the doctor sees nothing of the conversation (row-level security reads it). It runs in the caller's
// transaction. ErrInactive when there is none, ErrForbidden to staff.
func (s *Service) RevokeChat(ctx context.Context, c Caller, conversationID, roleID string) (Consent, error) {
	if !c.patient() {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrForbidden)
	}
	if c.PatientID == "" {
		return Consent{}, s.refuse(ctx, c, auditevent.CodeUpdate, ErrInactive)
	}
	var out Consent
	var refused error
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.ownPatient(ctx, c); err != nil {
			return err
		}
		var err error
		out, err = s.repo.RevokeChat(ctx, conversationID, roleID, s.clock.Now())
		switch {
		case errors.Is(err, ErrInactive):
			refused = ErrInactive
			return s.record(ctx, c, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)
		case err != nil:
			return err
		}
		return s.record(ctx, c, auditevent.CodeUpdate, out.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Consent{}, fmt.Errorf("consent: revoke chat: %w", err)
	case refused != nil:
		return Consent{}, refused
	}
	return out, nil
}

// ChatGranted reports whether the chat consent for the conversation and the practitioner role is active, as the
// caller's scope shows it (the patient's, the conversation's clinic's, or the invited doctor's own). Like State, it
// is the access check inside the caller's audited access and is not audited itself.
func (s *Service) ChatGranted(ctx context.Context, conversationID, roleID string) (bool, error) {
	ok, err := s.repo.ChatGranted(ctx, conversationID, roleID)
	if err != nil {
		return false, fmt.Errorf("consent: chat granted: %w", err)
	}
	return ok, nil
}
