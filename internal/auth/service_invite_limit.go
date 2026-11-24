package auth

import "context"

// maxOpenInvites bounds a clinic's open invites (V6 of docs/clinic-cms/WAVE4.md).
const maxOpenInvites = 50

// inviteLimitReached reports whether the clinic org already has maxOpenInvites open invites, none of them for
// email: an invite that replaces an open one for the same e-mail adds none. Inside the caller's transaction.
func (s *Service) inviteLimitReached(ctx context.Context, org, email string) (bool, error) {
	if err := s.repo.SetOrganization(ctx, org); err != nil {
		return false, err
	}
	open, err := s.repo.OpenInvites(ctx, org, s.clock.Now())
	if err != nil {
		return false, err
	}
	for _, inv := range open {
		if inv.Email == email {
			return false, nil
		}
	}
	return len(open) >= maxOpenInvites, nil
}
