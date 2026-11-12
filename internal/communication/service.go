package communication

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Consents gives the state of a patient's consents with a clinic, in the caller's scope (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// Visits gives the patient's appointments in a clinic, in the caller's scope (appointment.Facts).
type Visits interface {
	Visits(ctx context.Context, patientID, organizationID string) ([]access.Visit, error)
}

// Roles gives a practitioner's role in their clinic, in the clinic's scope (practitionerrole.Service).
type Roles interface {
	OfPractitioner(ctx context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error)
}

// Encounters gives an encounter as the caller's scope shows it (encounter.Service), to link a conversation to it.
type Encounters interface {
	Lookup(ctx context.Context, id string) (encounter.Encounter, error)
}

// ChatConsents grants, revokes and checks the patient's consents for doctors of other clinics in their
// conversations (consent.Service, SAL-28), in the caller's transaction and scope; the clinic ends one when it removes
// the doctor (EndChat).
type ChatConsents interface {
	GrantChat(ctx context.Context, c consent.Caller, g consent.ChatGrant) (consent.Consent, error)
	RevokeChat(ctx context.Context, c consent.Caller, conversationID, roleID string) (consent.Consent, error)
	EndChat(ctx context.Context, c consent.Caller, conversationID, roleID string) (consent.Consent, error)
	ChatGranted(ctx context.Context, conversationID, roleID string) (bool, error)
}

// Directory gives the practitioner roles a doctor may add (practitionerrole.Service): a role of their own clinic,
// and the public directory of another.
type Directory interface {
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
	PublicByOrganization(ctx context.Context, org string) ([]practitionerrole.PractitionerRole, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo         Repository
	Publisher    Publisher
	Consents     Consents
	Visits       Visits
	Roles        Roles
	Encounters   Encounters
	ChatConsents ChatConsents
	Directory    Directory
	Audit        Auditor
	Tx           Transactor
	Jobs         Jobs // a send enqueues its notifications (SAL-29)
	Clock        clock.Clock
	Log          *slog.Logger
}

// Service holds the chat's rules (docs/plans/chat.md "Decisions"): who opens a conversation (access.OpensChat),
// who reads and writes in it (access.ReadsChat, access.WritesChat), the order and idempotency of a send, edits
// only by their sender, and an AuditEvent for every read and write.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }

// Caller is who asks (internal/api resolves it from the session), as the appointment domain names it.
type Caller = appointment.Caller

// who is the caller as the rules see them: a patient, or a practitioner with their role in the clinic.
type who struct {
	role   string // the practitioner's role; empty for a patient
	viewer access.Viewer
}

func (w who) patient() bool { return w.viewer.Role == access.RolePatient }

// identify scopes the transaction to the caller and returns who they are: a patient with a profile, or a
// practitioner of a clinic with a role there. An admin, or staff without a role, is ErrForbidden.
func (s *Service) identify(ctx context.Context, c Caller) (who, error) {
	if c.OrganizationID == "" {
		if c.PatientID == "" {
			return who{}, ErrForbidden
		}
		return who{viewer: access.Viewer{Role: access.RolePatient, Patient: c.PatientID}}, s.Repo.SetPatient(ctx, c.PatientID)
	}
	if c.Admin || c.PractitionerID == "" {
		return who{}, ErrForbidden
	}
	if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
		return who{}, err
	}
	role, err := s.Roles.OfPractitioner(ctx, c.OrganizationID, c.PractitionerID)
	switch {
	case errors.Is(err, practitionerrole.ErrNotFound):
		return who{}, ErrForbidden
	case err != nil:
		return who{}, err
	}
	if err := s.Repo.SetPractitionerRole(ctx, role.ID); err != nil {
		return who{}, err
	}
	return who{viewer: access.Viewer{Role: access.RolePractitioner, Clinic: c.OrganizationID, PractitionerRole: role.ID}, role: role.ID}, nil
}

// entered is a conversation the caller has entered: who they are, their participant row, and the patient's facts;
// in is whether they take part (an active participant), granted whether they are a doctor of another clinic whose
// invitation the patient granted (SAL-28).
type entered struct {
	who     who
	facts   access.Facts
	part    Participant
	conv    Conversation
	in      bool
	granted bool
}

// guest reports whether the caller is a practitioner of another clinic than the conversation's.
func (e entered) guest() bool {
	return !e.who.patient() && e.who.viewer.Clinic != e.conv.OrganizationID
}

// root is the conversation a side thread belongs to, or the conversation itself.
func root(c Conversation) string {
	if c.ParentID != "" {
		return c.ParentID
	}
	return c.ID
}

// fromSeq is the number after which the caller reads (their history_from_seq).
func (e entered) fromSeq() int64 { return e.part.HistoryFromSeq }

// enter reads the conversation id for the caller (locked for a send) and checks that they may read it, or write in
// it when write is set; then it scopes the transaction to the conversation's messages. ErrNotFound when the scope
// does not show it, also to staff after a revocation (the clinic's row-level security hides it then);
// ErrConsentRequired for a patient's write without the active consent; ErrForbidden otherwise.
func (s *Service) enter(ctx context.Context, c Caller, id string, write, lock bool) (entered, error) {
	if !validUUID(id) {
		return entered{}, ErrNotFound
	}
	w, err := s.identify(ctx, c)
	if err != nil {
		return entered{}, err
	}
	conv, err := s.Repo.Conversation(ctx, id, lock)
	if err != nil {
		return entered{}, err
	}
	parts, err := s.Repo.Participants(ctx, id)
	if err != nil {
		return entered{}, err
	}
	conv.Participants = parts
	part, in := participantOf(parts, w)
	state, err := s.Consents.State(ctx, conv.PatientID, conv.OrganizationID)
	if err != nil {
		return entered{}, err
	}
	e := entered{conv: conv, part: part, who: w, in: in, facts: access.Facts{Patient: conv.PatientID, Clinic: conv.OrganizationID, Consent: state}}
	if e.guest() && in {
		if e.granted, err = s.ChatConsents.ChatGranted(ctx, root(conv), w.role); err != nil {
			return entered{}, err
		}
	}
	allowed := access.ReadsChat(w.viewer, e.facts, in)
	if write {
		allowed = access.WritesChat(w.viewer, e.facts, in)
	}
	allowed = allowed || access.GuestChat(w.viewer, e.facts, e.granted)
	switch {
	case !allowed && e.guest():
		return entered{}, ErrNotFound // a doctor of another clinic learns nothing they were not granted
	case !allowed && !state.Active && w.patient():
		return entered{}, ErrConsentRequired
	case !allowed && !state.Active:
		return entered{}, ErrNotFound // as the clinic's row-level security answers: nothing of the chat is disclosed
	case !allowed:
		return entered{}, ErrForbidden
	}
	return e, s.Repo.SetConversation(ctx, id)
}

// participantOf is the caller's active participant row, and whether they have one; an invitation waiting for the
// patient's consent is none.
func participantOf(parts []Participant, w who) (Participant, bool) {
	for _, p := range parts {
		if p.Status != StatusActive {
			continue
		}
		if (w.patient() && p.Role == RolePatient && p.PatientID == w.viewer.Patient) ||
			(!w.patient() && p.Role == RolePractitioner && p.PractitionerRoleID == w.role) {
			return p, true
		}
	}
	return Participant{}, false
}

// The audit's entity names: a table, with the number of rows the event covers.
const (
	entityConversation  = "conversation"
	entityCommunication = "communication"
)

// record audits action by c on the patient's chat: ref is the resource (Communication/<id>) or the patient
// (Patient/<id>), name and count the rows when the event covers several. The purpose is the patient's own request
// for a patient and treatment for staff.
func (s *Service) record(ctx context.Context, c Caller, action auditevent.Action, interaction, ref, name string, count int64) error {
	e := auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          action,
		Outcome:         auditevent.OutcomeSuccess,
		AgentAccountID:  c.AccountID,
		OrganizationID:  c.OrganizationID,
		EntityReference: ref,
		EntityName:      name,
		EntityCount:     count,
	}
	if c.OrganizationID == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	return s.Audit.Record(ctx, e)
}

// refusal audits a refused access to the conversation id by c, in its own transaction (the refused one rolls
// back), and returns err. Not-found is not audited: nothing was disclosed.
func (s *Service) refusal(ctx context.Context, c Caller, id string, err error) error {
	if !errors.Is(err, ErrForbidden) && !errors.Is(err, ErrConsentRequired) {
		return err
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:         auditevent.ActionRead,
		Outcome:        auditevent.OutcomeMinorFailure,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     entityConversation,
	}
	if c.OrganizationID == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error { return s.Audit.Record(ctx, e) }); aerr != nil {
		return fmt.Errorf("communication: audit refusal: %w", aerr)
	}
	return err
}

// publish puts the frame of m on its conversation's live channel; a failure is logged, without the message: the
// sockets fill the gap from Postgres (ADR 0012).
func (s *Service) publish(ctx context.Context, typ string, m Message) {
	frame, err := wire.Encode(typ, FrameVersion, m)
	if err == nil {
		err = s.Publisher.Publish(ctx, ChannelKey(m.ConversationID), frame)
	}
	if err != nil {
		s.Log.WarnContext(ctx, "chat publish failed", "conversation_id", m.ConversationID, "type", typ, "error", err.Error())
	}
}

// wrap names the operation in an unexpected error and passes the domain's own errors through.
func wrap(op string, err error) error {
	for _, known := range []error{ErrInvalid, ErrInvalidCursor, ErrNotFound, ErrForbidden, ErrConsentRequired, ErrVersionConflict} {
		if errors.Is(err, known) {
			return err
		}
	}
	if err == nil {
		return nil
	}
	return fmt.Errorf("communication: %s: %w", op, err)
}
