package signup_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/signup"
)

// A clinic's self-signup and its setup wizard are the wave 3 contract (docs/clinic-cms, 3.0): refused until package
// O1 builds them.
func TestSignup_RefusesUntilPackageO1(t *testing.T) {
	t.Parallel()
	var s signup.Service
	ctx := t.Context()
	c := signup.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	start := s.Start(ctx, signup.Request{Email: "admin@example.test", ClinicName: "Example clinic", Language: "bs"})
	_, verify := s.Verify(ctx, "token")
	_, setup := s.Setup(ctx, c)
	_, save := s.SaveSetup(ctx, c, signup.Setup{Steps: map[signup.Step]signup.StepState{signup.StepDetails: signup.StepDone}})
	for name, err := range map[string]error{"Start": start, "Verify": verify, "Setup": setup, "SaveSetup": save} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s = %v, want errors.ErrUnsupported", name, err)
		}
	}
}
