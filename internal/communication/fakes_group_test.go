package communication_test

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// The group consult's synthetic ids (SAL-28): an inactive role of clinic A, clinic B's listed role, a clinic that is
// not listed, and the doctor of clinic B.
const (
	roleA3   = "a3000000-0000-4000-8000-0000000000a3" // inactive
	roleB1   = "b1000000-0000-4000-8000-0000000000b1" // clinic B, in the public directory
	roleB2   = "b2000000-0000-4000-8000-0000000000b2" // clinic B, not in the public directory
	clinicC  = "c0000000-0000-4000-8000-00000000000c" // not listed
	roleC1   = "c1000000-0000-4000-8000-0000000000c1"
	pGuestID = "p3000000-0000-4000-8000-0000000000b1"
)

var guest = communication.Caller{AccountID: "ac300000-0000-4000-8000-0000000000b1", OrganizationID: clinicB, PractitionerID: pGuestID}

// fakeDirectory is clinic A's roles (A1 and A2 active, A3 not) and clinic B's public directory (B1 only).
type fakeDirectory struct{}

func (fakeDirectory) Lookup(_ context.Context, org, id string) (practitionerrole.PractitionerRole, error) {
	if org == clinicA && (id == roleA1 || id == roleA2 || id == roleA3) {
		return practitionerrole.PractitionerRole{ID: id, OrganizationID: clinicA, Active: id != roleA3}, nil
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

func (fakeDirectory) PublicByOrganization(_ context.Context, org string) ([]practitionerrole.PractitionerRole, error) {
	if org == clinicB {
		return []practitionerrole.PractitionerRole{{ID: roleB1, OrganizationID: clinicB, Active: true}}, nil
	}
	return nil, nil
}

// fakeChatConsents keeps the fixture's chat consents, granted and revoked by the patient only.
type fakeChatConsents struct{ f *fixture }

func (c fakeChatConsents) GrantChat(_ context.Context, by consent.Caller, g consent.ChatGrant) (consent.Consent, error) {
	c.f.mu.Lock()
	defer c.f.mu.Unlock()
	if by.OrganizationID != "" || by.PatientID == "" {
		return consent.Consent{}, consent.ErrForbidden
	}
	if g.PractitionerOrganizationID != clinicB || g.PractitionerRoleID != roleB1 {
		return consent.Consent{}, errors.New("the grant names another role or clinic than the invited doctor's")
	}
	c.f.chat[g.ConversationID+g.PractitionerRoleID] = true
	return consent.Consent{ID: "f9000000-0000-4000-8000-0000000000f9", Kind: consent.KindChat, PatientID: by.PatientID,
		OrganizationID: g.OrganizationID, ConversationID: g.ConversationID, PractitionerRoleID: g.PractitionerRoleID, Status: consent.StatusActive}, nil
}

func (c fakeChatConsents) RevokeChat(_ context.Context, by consent.Caller, conv, role string) (consent.Consent, error) {
	c.f.mu.Lock()
	defer c.f.mu.Unlock()
	if by.OrganizationID != "" {
		return consent.Consent{}, consent.ErrForbidden
	}
	if !c.f.chat[conv+role] {
		return consent.Consent{}, consent.ErrInactive
	}
	c.f.chat[conv+role] = false
	return consent.Consent{Kind: consent.KindChat, Status: consent.StatusInactive}, nil
}

func (c fakeChatConsents) ChatGranted(_ context.Context, conv, role string) (bool, error) {
	c.f.mu.Lock()
	defer c.f.mu.Unlock()
	return c.f.chat[conv+role], nil
}

// consented is the guest's consents for the fake's row-level security.
func (f *fixture) consented(conv, role, patient, clinic string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.chat[conv+role] && f.consent[patient+clinic].Active
}

// guestOf is the guest's row-level security of 00044, under r.mu: the conversation (or the one a side thread belongs
// to) has the scope's role as an active participant of the scope's clinic, another than the conversation's, with the
// chat consent and the care consent active.
func (r *fakeRepo) guestOf(sc scope, c communication.Conversation) bool {
	root := c.ID
	if c.ParentID != "" {
		root = c.ParentID
	}
	conv, ok := r.convs[root]
	if !ok || conv.OrganizationID == sc.clinic {
		return false
	}
	for _, p := range r.parts[root] {
		if p.PractitionerRoleID == sc.role && p.OrganizationID == sc.clinic && p.Status == communication.StatusActive {
			return r.consented(root, sc.role, conv.PatientID, conv.OrganizationID)
		}
	}
	return false
}

func (r *fakeRepo) SetPractitionerRole(ctx context.Context, id string) error {
	txOf(ctx).scope.role = id
	return nil
}

func (r *fakeRepo) Participant(ctx context.Context, id string) (communication.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conv, parts := range r.parts {
		for _, p := range parts {
			if p.ID == id && r.visible(txOf(ctx).scope, r.convs[conv]) {
				return p, nil
			}
		}
	}
	return communication.Participant{}, communication.ErrNotFound
}

func (r *fakeRepo) InsertParticipant(_ context.Context, p communication.Participant) (communication.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return communication.Participant{}, err
	}
	p.ID = r.id("d")
	p.ReadSeq = p.HistoryFromSeq
	r.parts[p.ConversationID] = append(r.parts[p.ConversationID], p)
	return p, nil
}

func (r *fakeRepo) ActivateParticipant(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conv, parts := range r.parts {
		for i, p := range parts {
			if p.ID == id {
				r.parts[conv][i].Status = communication.StatusActive
			}
		}
	}
	return nil
}

// RemoveParticipant drops the row, and the role's rows in the conversation's side threads (the trigger of 00044).
func (r *fakeRepo) RemoveParticipant(_ context.Context, id string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conv, parts := range r.parts {
		i := slices.IndexFunc(parts, func(p communication.Participant) bool { return p.ID == id })
		if i < 0 {
			continue
		}
		role := parts[i].PractitionerRoleID
		r.parts[conv] = slices.Delete(parts, i, i+1)
		for thread, c := range r.convs {
			if c.ParentID == conv {
				r.parts[thread] = slices.DeleteFunc(r.parts[thread], func(p communication.Participant) bool { return p.PractitionerRoleID == role })
			}
		}
		return nil
	}
	return communication.ErrNotFound
}

func (r *fakeRepo) CreateThread(ctx context.Context, c communication.Conversation, parts []communication.Participant) (communication.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return communication.Conversation{}, err
	}
	parent, ok := r.convs[c.ParentID]
	if !ok || !r.visible(txOf(ctx).scope, parent) || !c.StaffOnly {
		return communication.Conversation{}, communication.ErrNotFound
	}
	c.ID = r.id("c")
	r.convs[c.ID] = c
	for _, p := range parts {
		p.ID, p.ConversationID = r.id("d"), c.ID
		r.parts[c.ID] = append(r.parts[c.ID], p)
	}
	c.Participants = slices.Clone(r.parts[c.ID])
	return c, nil
}

func (r *fakeRepo) Threads(ctx context.Context, parentID, roleID string) ([]communication.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []communication.Conversation
	for _, c := range r.convs {
		if c.ParentID != parentID || !r.visible(txOf(ctx).scope, c) {
			continue
		}
		for _, p := range r.parts[c.ID] {
			if p.PractitionerRoleID == roleID {
				c.Unread = c.LastSeq - p.ReadSeq
				out = append(out, c)
			}
		}
	}
	slices.SortFunc(out, func(a, b communication.Conversation) int { return b.Created.Compare(a.Created) })
	return out, nil
}

func (r *fakeRepo) Invitations(ctx context.Context, patientID string) ([]communication.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []communication.Invitation
	for conv, parts := range r.parts {
		c := r.convs[conv]
		if c.PatientID != patientID || !r.visible(txOf(ctx).scope, c) {
			continue
		}
		for _, p := range parts {
			if p.Status == communication.StatusPendingConsent {
				out = append(out, communication.Invitation{InvitedAt: p.AddedAt, ID: p.ID, ConversationID: conv, OrganizationID: p.OrganizationID,
					PractitionerRoleID: p.PractitionerRoleID, Reason: p.Reason})
			}
		}
	}
	return out, nil
}
