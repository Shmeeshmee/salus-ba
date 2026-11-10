package auth_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// fakeInvites holds invites by id and shows only those of the clinic set on the transaction, as row-level
// security does. It is part of fakeRepository.
type fakeInvites struct {
	invites map[string]*auth.Invite
	links   map[string]string // account id -> practitioner id
	hashes  map[string][]byte // invite id -> token hash
	clinic  string            // set by SetOrganization
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization")
	r.clinic = id
	return r.err
}

func (r *fakeRepository) PutInvite(ctx context.Context, inv auth.Invite) (string, error) {
	r.seen(ctx, "PutInvite")
	if r.err != nil {
		return "", r.err
	}
	for id, open := range r.invites {
		if open.OrganizationID == inv.OrganizationID && open.Email == inv.Email && open.AcceptedAt.IsZero() {
			inv.ID = id // replaced, not duplicated
		}
	}
	if inv.ID == "" {
		inv.ID = fmt.Sprintf("1a000000-0000-4000-8000-%012d", len(r.invites)+1)
	}
	if r.hashes != nil {
		delete(r.hashes, inv.ID)
	}
	r.invites[inv.ID] = &inv
	return inv.ID, nil
}

func (r *fakeRepository) visibleInvite(id string) (*auth.Invite, error) {
	inv, ok := r.invites[id]
	if !ok || inv.OrganizationID != r.clinic {
		return nil, auth.ErrInvalidInvite
	}
	return inv, nil
}

func (r *fakeRepository) OpenInvites(ctx context.Context, org string, at time.Time) ([]auth.Invite, error) {
	r.seen(ctx, "OpenInvites")
	var open []auth.Invite
	for _, id := range slices.Sorted(maps.Keys(r.invites)) {
		inv := r.invites[id]
		if inv.OrganizationID == org && org == r.clinic && inv.AcceptedAt.IsZero() && inv.ExpiresAt.After(at) {
			open = append(open, *inv)
		}
	}
	return open, r.err
}

func (r *fakeRepository) InviteByID(ctx context.Context, id string) (auth.Invite, error) {
	r.seen(ctx, "InviteByID")
	if r.err != nil {
		return auth.Invite{}, r.err
	}
	inv, err := r.visibleInvite(id)
	if err != nil {
		return auth.Invite{}, err
	}
	return *inv, nil
}

func (r *fakeRepository) SetInviteToken(ctx context.Context, id string, hash []byte) error {
	r.seen(ctx, "SetInviteToken")
	if r.hashes == nil {
		r.hashes = map[string][]byte{}
	}
	r.hashes[id] = hash
	return r.err
}

func (r *fakeRepository) InviteOrganization(ctx context.Context, hash []byte) (string, error) {
	r.seen(ctx, "InviteOrganization")
	for id, h := range r.hashes {
		if string(h) == string(hash) {
			return r.invites[id].OrganizationID, r.err
		}
	}
	return "", auth.ErrInvalidInvite
}

func (r *fakeRepository) InviteByTokenHash(ctx context.Context, hash []byte) (auth.Invite, error) {
	r.seen(ctx, "InviteByTokenHash")
	for id, h := range r.hashes {
		if string(h) == string(hash) {
			return r.InviteByID(ctx, id)
		}
	}
	return auth.Invite{}, auth.ErrInvalidInvite
}

func (r *fakeRepository) AcceptInvite(ctx context.Context, id string, at time.Time) error {
	r.seen(ctx, "AcceptInvite")
	inv, err := r.visibleInvite(id)
	if err != nil || !inv.AcceptedAt.IsZero() || !inv.ExpiresAt.After(at) {
		return auth.ErrInvalidInvite
	}
	inv.AcceptedAt = at
	return r.err
}

func (r *fakeRepository) SetPractitioner(ctx context.Context, id, practitionerID string) error {
	r.seen(ctx, "SetPractitioner")
	r.links[id] = practitionerID
	return r.err
}

func (r *fakeRepository) ResetMFA(ctx context.Context, id string) error {
	r.seen(ctx, "ResetMFA")
	a := r.accounts[id]
	a.MFAEnrolledAt = time.Time{}
	r.accounts[id] = a
	delete(r.secrets, id)
	delete(r.lastStep, id)
	return r.err
}

func (r *fakeRepository) RevokeSessions(ctx context.Context, accountID string, at time.Time) error {
	r.seen(ctx, "RevokeSessions")
	for _, s := range r.sessions {
		if s.AccountID == accountID && s.revokedAt.IsZero() {
			s.revokedAt = at
		}
	}
	return r.sessionErr
}

// fakeJobs keeps the invite_email jobs enqueued and whether each was enqueued inside a transaction.
type fakeJobs struct {
	err  error
	jobs []auth.InviteEmailArgs
	inTx []bool
}

func (j *fakeJobs) EnqueueInviteEmail(ctx context.Context, inviteID, organizationID string) error {
	j.jobs = append(j.jobs, auth.InviteEmailArgs{InviteID: inviteID, OrganizationID: organizationID})
	j.inTx = append(j.inTx, ctx.Value(txKey{}) != nil)
	return j.err
}

// fakeDirectory stands in for the practitioner and practitionerrole services: it keeps who created what,
// and whether inside a transaction.
type fakeDirectory struct {
	err           error // returned by both creates when set
	practitioners []practitioner.Caller
	names         [][2]string
	roles         []string // practitioner ids given a role
	roleCallers   []practitionerrole.Caller
	outside       int // creates made outside a transaction
}

const createdPractitioner = "a3000000-0000-4000-8000-00000000c0de"

func (d *fakeDirectory) count(ctx context.Context) {
	if ctx.Value(txKey{}) == nil {
		d.outside++
	}
}

func (d *fakeDirectory) Create(ctx context.Context, c practitioner.Caller, given, family string) (practitioner.Practitioner, error) {
	d.count(ctx)
	d.practitioners, d.names = append(d.practitioners, c), append(d.names, [2]string{given, family})
	return practitioner.Practitioner{ID: createdPractitioner, OrganizationID: c.OrganizationID, GivenName: given, FamilyName: family}, d.err
}

// CreateRole is wired as the PractitionerRoles port through roleCreator.
func (d *fakeDirectory) CreateRole(ctx context.Context, c practitionerrole.Caller, practitionerID string) (practitionerrole.PractitionerRole, error) {
	d.count(ctx)
	d.roles, d.roleCallers = append(d.roles, practitionerID), append(d.roleCallers, c)
	return practitionerrole.PractitionerRole{ID: "a4000000-0000-4000-8000-00000000c0de", PractitionerID: practitionerID, OrganizationID: c.OrganizationID}, d.err
}

type roleCreator struct{ d *fakeDirectory }

func (r roleCreator) Create(ctx context.Context, c practitionerrole.Caller, practitionerID string) (practitionerrole.PractitionerRole, error) {
	return r.d.CreateRole(ctx, c, practitionerID)
}

func (r *fakeRepository) SetTermReviewer(ctx context.Context, id string, reviewer bool) error {
	r.seen(ctx, "SetTermReviewer")
	if r.err != nil {
		return r.err
	}
	a := r.accounts[id]
	a.TermReviewer = reviewer
	r.accounts[id] = a
	return nil
}
