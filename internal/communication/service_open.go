package communication

import (
	"context"
	"errors"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// Open opens the conversation between a patient and a practitioner role of a clinic, or returns the one they have
// for that encounter (none counts as one), with its participants, and whether it is new (docs/plans/chat.md "Who
// opens a conversation"). Staff name the patient and open it with their own role; a patient names the clinic and
// the role. access.OpensChat decides: the active care consent and an appointment of the patient with that role.
// ErrInvalid for an id that is no UUID, ErrConsentRequired without the consent, ErrNotFound when staff have no
// such patient or the encounter is not theirs, and ErrForbidden for a patient without an appointment with the role.
func (s *Service) Open(ctx context.Context, c Caller, in Open) (Conversation, bool, error) {
	var out Conversation
	var created bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		w, err := s.identify(ctx, c)
		if err != nil {
			return err
		}
		want, err := s.wanted(w, c, in)
		if err != nil {
			return err
		}
		if err := s.mayOpen(ctx, w, want); err != nil {
			return err
		}
		now := s.Clock.Now().UTC()
		patient := Participant{Role: RolePatient, PatientID: want.PatientID, AddedBy: c.AccountID, AddedAt: now}
		doctor := Participant{Role: RolePractitioner, PractitionerRoleID: want.PractitionerRoleID, AddedBy: c.AccountID, AddedAt: now}
		want.Created, want.CreatedBy = now, c.AccountID
		out, created, err = s.Repo.CreateConversation(ctx, want, []Participant{patient, doctor})
		if err != nil {
			return err
		}
		action, interaction := auditevent.ActionRead, auditevent.CodeRead
		if created {
			action, interaction = auditevent.ActionCreate, auditevent.CodeCreate
		}
		return s.record(ctx, c, action, interaction, "Patient/"+out.PatientID, entityConversation, 1)
	})
	if err != nil {
		return Conversation{}, false, wrap("open", s.refusal(ctx, c, "", err))
	}
	return out, created, nil
}

// wanted is the conversation the caller asks for, in their own clinic or with their own role.
func (s *Service) wanted(w who, c Caller, in Open) (Conversation, error) {
	conv := Conversation{EncounterID: in.EncounterID}
	if w.patient() {
		conv.PatientID, conv.OrganizationID, conv.PractitionerRoleID = c.PatientID, in.OrganizationID, in.PractitionerRoleID
	} else {
		conv.PatientID, conv.OrganizationID, conv.PractitionerRoleID = in.PatientID, c.OrganizationID, w.role
	}
	for _, f := range [...]struct{ name, id string }{
		{"patient_id", conv.PatientID}, {"organization_id", conv.OrganizationID}, {"practitioner_role_id", conv.PractitionerRoleID},
	} {
		if !validUUID(f.id) {
			return Conversation{}, invalid(f.name)
		}
	}
	if conv.EncounterID != "" && !validUUID(conv.EncounterID) {
		return Conversation{}, invalid("encounter_id")
	}
	return conv, nil
}

// mayOpen checks access.OpensChat over the patient's facts in the clinic, and that the encounter, when one is
// named, is the patient's in that clinic.
func (s *Service) mayOpen(ctx context.Context, w who, want Conversation) error {
	state, err := s.Consents.State(ctx, want.PatientID, want.OrganizationID)
	if err != nil {
		return err
	}
	visits, err := s.Visits.Visits(ctx, want.PatientID, want.OrganizationID)
	if err != nil {
		return err
	}
	f := access.Facts{Patient: want.PatientID, Clinic: want.OrganizationID, Consent: state, Visits: visits}
	switch {
	case access.OpensChat(w.viewer, f, want.PractitionerRoleID):
	case len(visits) > 0 && !state.Active:
		return ErrConsentRequired
	case w.patient():
		return ErrForbidden
	default:
		return ErrNotFound // staff learn nothing of a patient they have no appointment with
	}
	if want.EncounterID == "" {
		return nil
	}
	enc, err := s.Encounters.Lookup(ctx, want.EncounterID)
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return ErrNotFound
	case err != nil:
		return err
	case enc.PatientID != want.PatientID || enc.OrganizationID != want.OrganizationID:
		return ErrNotFound
	}
	return nil
}

// Get returns the conversation id with its active participants, for a caller who may read it (enter), audited.
func (s *Service) Get(ctx context.Context, c Caller, id string) (Conversation, error) {
	var out Conversation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		out = e.conv
		out.Unread = max(out.LastSeq-e.part.ReadSeq, 0)
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Patient/"+out.PatientID, entityConversation, 1)
	})
	if err != nil {
		return Conversation{}, wrap("get", s.refusal(ctx, c, id, err))
	}
	return out, nil
}

// List returns a page of the caller's conversations (ListSize), newest activity first, with their unread counts,
// and the cursor of the next page (nil at the end), audited as one search. A practitioner sees only those they take
// part in while the patient's care consent is active (the clinic's row-level security); an admin none.
func (s *Service) List(ctx context.Context, c Caller, before *ListCursor) ([]Conversation, *ListCursor, error) {
	var out []Conversation
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		w, err := s.identify(ctx, c)
		if err != nil {
			return err
		}
		f := ListFilter{Before: before, Limit: ListSize + 1}
		if w.patient() {
			f.PatientID = c.PatientID
		} else {
			f.PractitionerRoleID = w.role
		}
		if out, err = s.Repo.Conversations(ctx, f); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeSearchType, "", entityConversation, int64(min(len(out), ListSize)))
	})
	if err != nil {
		return nil, nil, wrap("list", s.refusal(ctx, c, "", err))
	}
	if len(out) <= ListSize {
		return out, nil, nil
	}
	out = out[:ListSize]
	last := out[len(out)-1]
	return out, &ListCursor{At: activity(last), ID: last.ID}, nil
}

// activity is when the conversation last changed: its last message, or its opening.
func activity(c Conversation) time.Time {
	if c.LastSentAt.IsZero() {
		return c.Created
	}
	return c.LastSentAt
}
