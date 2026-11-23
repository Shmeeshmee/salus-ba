package organization

import (
	"context"
	"errors"
)

// Profile returns the profile of the clinic id, as the public directory shows it. The wave 3 contract: refused until
// package X2.
func (s *Service) Profile(ctx context.Context, id string) (Profile, error) {
	return Profile{}, errors.ErrUnsupported
}

// SaveProfile replaces the profile of the admin caller's clinic at version (the organization's If-Match). Refused
// until package X2.
func (s *Service) SaveProfile(ctx context.Context, c Caller, p Profile, version int32) (Profile, error) {
	return Profile{}, errors.ErrUnsupported
}
