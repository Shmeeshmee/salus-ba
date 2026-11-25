package auth_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
)

// U8 (docs/clinic-cms/WAVE6.md): once a clinic's admin accepts their invite, the clinic's signup forgets the admin's
// e-mail, in the acceptance's transaction; another role's acceptance leaves it.
func TestAcceptInvite_TellsTheSignupThatTheAdminJoined(t *testing.T) {
	for role, want := range map[auth.Role][]string{auth.RoleAdmin: {clinicID}, auth.RolePractitioner: nil, auth.RoleReception: nil} {
		f := newFixture()
		_, token := invited(t, f, auth.Invite{Email: "cy@example.test", Role: role, GivenName: "Cy", FamilyName: "Example"})

		if _, err := f.svc.AcceptInvite(t.Context(), token, password, "", ""); err != nil {
			t.Fatalf("AcceptInvite(%s) error = %v", role, err)
		}
		if !slices.Equal(f.signups.joined, want) || slices.Contains(f.signups.inTx, false) {
			t.Errorf("AcceptInvite(%s): admin joined %v (in the transaction %v), want %v", role, f.signups.joined, f.signups.inTx, want)
		}
	}
}
