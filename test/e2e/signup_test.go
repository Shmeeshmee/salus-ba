//go:build e2e

package e2e_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var signupLink = regexp.MustCompile(`/signup/verify#([A-Za-z0-9_-]+)`)

// TestClinic_SignsItselfUpAndIsListedOnceApproved (docs/clinic-cms, O1): a clinic signs up, verifies its e-mail from
// Mailpit, its admin accepts the invite and finds the setup wizard open; the clinic is unlisted until the operator
// approves it with salusctl.
func TestClinic_SignsItselfUpAndIsListedOnceApproved(t *testing.T) {
	waitReady(t)
	c := newAPI(t)
	addr := email(t, "signup")
	name := "E2E synthetic signup clinic"

	c.must(http.StatusAccepted, http.MethodPost, "/signup", map[string]string{"email": addr, "clinicName": name, "language": "en"}, nil)
	m := signupLink.FindStringSubmatch(mailText(t, addr, ""))
	if m == nil {
		t.Fatal("the signup e-mail holds no verification link")
	}
	c.must(http.StatusNoContent, http.MethodPost, "/signup/verify", map[string]string{"token": m[1]}, nil)
	if got, code := c.refusal(http.MethodPost, "/signup/verify", map[string]string{"token": m[1]}); got != http.StatusBadRequest || code != "invalid_verification" {
		t.Errorf("POST /signup/verify again = %d %s, want 400 invalid_verification", got, code)
	}

	admin := acceptInvite(t, addr)
	me := admin.me()
	if me.Role != "admin" || me.OrganizationID == "" {
		t.Fatalf("GET /auth/me of the signed-up admin = %+v, want an admin of the new clinic", me)
	}
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
