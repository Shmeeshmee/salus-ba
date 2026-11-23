//go:build e2e

package e2e_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

var signupLink = regexp.MustCompile(`/signup/verify#([A-Za-z0-9_-]+)`)

// TestClinic_SignsItselfUpAndIsListedOnceApproved (docs/clinic-cms, O1): a clinic signs up, verifies its e-mail from
// Mailpit, its admin accepts the invite and finds the setup wizard open, through the API and in the web app (the
// banner, test/e2e/web/signup.spec.ts); a practitioner of the clinic is refused the wizard; the clinic is unlisted
// until the operator approves it with salusctl.
func TestClinic_SignsItselfUpAndIsListedOnceApproved(t *testing.T) {
	waitReady(t)
	clearLoginLimits(t) // the signup's own limits (ratelimit:signup:*) too, for a rerun within the hour
	c := newAPI(t)
	addr := email(t, "signup")
	name := "E2E synthetic signup clinic " + suffix(t)

	c.must(http.StatusAccepted, http.MethodPost, "/signup", map[string]string{"email": addr, "clinicName": name, "language": "en"}, nil)
	m := signupLink.FindStringSubmatch(mailText(t, addr, ""))
	if m == nil {
		t.Fatal("the signup e-mail holds no verification link")
	}
	c.must(http.StatusNoContent, http.MethodPost, "/signup/verify", map[string]string{"token": m[1]}, nil)
	if got, code := c.refusal(http.MethodPost, "/signup/verify", map[string]string{"token": m[1]}); got != http.StatusBadRequest || code != "invalid_verification" {
		t.Errorf("POST /signup/verify again = %d %s, want 400 invalid_verification", got, code)
	}

	admin, totp := enrolFromInvite(t, addr)
	me := admin.me()
	if me.Role != "admin" || me.OrganizationID == "" {
		t.Fatalf("GET /auth/me of the signed-up admin = %+v, want an admin of the new clinic", me)
	}
	checkSetupWizard(t, admin, me.OrganizationID)
	signupWebJourney(t, addr, totp)

	listed := func() bool {
		var bundle struct {
			Entry []struct {
				Resource struct{ ID string } `json:"resource"`
			} `json:"entry"`
		}
		newAPI(t).must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization?listed=true", nil, &bundle)
		for _, e := range bundle.Entry {
			if e.Resource.ID == me.OrganizationID {
				return true
			}
		}
		return false
	}
	if listed() {
		t.Fatal("the clinic is listed before the operator approved it")
	}
	if out := salusctl(t, "signup", "list"); !strings.Contains(out, me.OrganizationID+" verified") || strings.Contains(out, addr) {
		t.Errorf("salusctl signup list = %q, want the clinic and no e-mail", out)
	}
	salusctl(t, "signup", "approve", me.OrganizationID)
	if !listed() {
		t.Error("the clinic is not listed after the operator approved it")
	}
}

// checkSetupWizard reads and saves the wizard's progress as the admin, and checks that a practitioner the admin
// invites is refused it: the wizard is the admin's.
func checkSetupWizard(t *testing.T, admin *api, clinic string) {
	t.Helper()
	var setup map[string]string
	admin.must(http.StatusOK, http.MethodGet, "/clinic/setup", nil, &setup)
	if setup["details"] != "open" || len(setup) != 5 {
		t.Errorf("GET /clinic/setup = %v, want five open steps", setup)
	}
	setup["details"] = "done"
	admin.must(http.StatusOK, http.MethodPut, "/clinic/setup", setup, &setup)
	if setup["details"] != "done" {
		t.Errorf("PUT /clinic/setup = %v, want details done", setup)
	}

	practitionerAddr := email(t, "signup-practitioner")
	admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": practitionerAddr, "role": "practitioner", "givenName": "Ada", "familyName": "Synthetic",
	}, nil)
	practitioner, _ := enrolFromInvite(t, practitionerAddr)
	if me := practitioner.me(); me.OrganizationID != clinic {
		t.Fatalf("the practitioner's clinic = %q, want %s", me.OrganizationID, clinic)
	}
	if got, code := practitioner.refusal(http.MethodGet, "/clinic/setup", nil); got != http.StatusForbidden || code != "forbidden" {
		t.Errorf("GET /clinic/setup by a practitioner = %d %s, want 403 forbidden", got, code)
	}
}

// signupWebJourney signs the admin in to the web app and finds the setup banner (test/e2e/web/signup.spec.ts).
func signupWebJourney(t *testing.T, addr string, totp *authenticator) {
	t.Helper()
	clearLoginLimits(t)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "signup.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+addr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", totp.last),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}
}
