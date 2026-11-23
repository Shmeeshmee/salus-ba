package organization_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// The clinic's profile for patients abroad is the wave 3 contract (docs/clinic-cms, 3.0): refused until package X2
// builds it.
func TestProfile_RefusesUntilPackageX2(t *testing.T) {
	t.Parallel()
	var s organization.Service
	c := organization.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	if _, err := s.Profile(t.Context(), "o"); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Profile = %v, want errors.ErrUnsupported", err)
	}
	if _, err := s.SaveProfile(t.Context(), c, organization.Profile{Languages: []string{"bs", "de"}}, 1); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("SaveProfile = %v, want errors.ErrUnsupported", err)
	}
}
