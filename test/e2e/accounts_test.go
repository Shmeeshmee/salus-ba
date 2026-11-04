//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"testing"
)

// TestPatient_RegistersLogsInAndLogsOut: a patient registers, logs in without MFA, /auth/me answers, and
// after logout the token fails.
func TestPatient_RegistersLogsInAndLogsOut(t *testing.T) {
	c := newAPI(t)
	addr := email(t, "patient")

	var registered account
	c.must(http.StatusCreated, http.MethodPost, "/auth/register", map[string]string{"email": addr, "password": password}, &registered)
	l := c.login(addr)
	if l.Next != "done" || l.Token == "" {
		t.Fatalf("patient login: next %q, token set %t; want done with a token", l.Next, l.Token != "")
	}
	c.bearer = l.Token
	if me := c.me(); me.ID != registered.ID || me.Email != addr || me.Role != "patient" || me.OrganizationID != "" {
		t.Errorf("GET /auth/me = %+v, want the registered patient %s", me, registered.ID)
	}

	c.must(http.StatusNoContent, http.MethodPost, "/auth/logout", nil, nil)
	if got := c.do(http.MethodGet, "/auth/me", nil, nil); got != http.StatusUnauthorized {
		t.Errorf("GET /auth/me after logout = %d, want 401", got)
	}
}

// TestClinic_AdminAndPractitionerSignInAndReadTheDirectory: salusctl creates a clinic, its admin accepts
// the invite from Mailpit, enrols TOTP and logs in with a code; reset-mfa makes the next login ask for
// enrolment again; the admin invites a practitioner, who accepts, enrols, logs in with a code and reads the
// clinic's directory.
func TestClinic_AdminAndPractitionerSignInAndReadTheDirectory(t *testing.T) {
	waitReady(t)
	adminAddr, practitionerAddr := email(t, "admin"), email(t, "practitioner")
	out := salusctl(t, "create-clinic", "-name", "E2E synthetic clinic", "-admin-email", adminAddr)
	clinic := field(t, out, "organization")

	admin := acceptInvite(t, adminAddr)
	adminAccount := admin.me()
	if adminAccount.Role != "admin" || adminAccount.OrganizationID != clinic || !adminAccount.MFAEnrolled {
		t.Fatalf("GET /auth/me after accepting = %+v, want an enrolled admin of %s", adminAccount, clinic)
	}

	salusctl(t, "reset-mfa", adminAccount.ID)
	if got := admin.do(http.MethodGet, "/auth/me", nil, nil); got != http.StatusUnauthorized {
		t.Errorf("GET /auth/me with the session from before reset-mfa = %d, want 401", got)
	}
	admin.bearer = ""
	l := admin.login(adminAddr)
	if l.Next != "enrol" || l.Challenge == "" || l.Token != "" {
		t.Fatalf("admin login after reset-mfa: next %q; want enrol with a challenge and no session", l.Next)
	}
	var enrolment struct{ Secret, URI string }
	admin.must(http.StatusOK, http.MethodPost, "/auth/mfa/enrol", map[string]string{"challenge": l.Challenge}, &enrolment)
	admin.bearer = confirm(t, admin, l.Challenge, newAuthenticator(t, enrolment.Secret))

	admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": practitionerAddr, "role": "practitioner", "givenName": "Ada", "familyName": "Synthetic",
	}, nil)
	practitioner := acceptInvite(t, practitionerAddr)
	if me := practitioner.me(); me.Role != "practitioner" || me.OrganizationID != clinic || !me.MFAEnrolled {
		t.Fatalf("GET /auth/me of the practitioner = %+v, want an enrolled practitioner of %s", me, clinic)
	}

	readDirectory(t, practitioner, clinic)
}

// acceptInvite accepts the invite e-mailed to addr, enrols TOTP with the first code, then logs in again
// with the next code. It returns the client signed in with that last session.
func acceptInvite(t *testing.T, addr string) *api {
	t.Helper()
	c := newAPI(t)
	var acc struct{ Challenge, Secret, URI string }
	c.must(http.StatusCreated, http.MethodPost, "/invites/accept", map[string]string{"token": inviteToken(t, addr), "password": password}, &acc)
	totp := newAuthenticator(t, acc.Secret)
	c.bearer = confirm(t, c, acc.Challenge, totp)
	if c.me().Email != addr {
		t.Fatal("GET /auth/me after enrolment: not the invited account")
	}

	c.bearer = ""
	l := c.login(addr)
	if l.Next != "verify" || l.Challenge == "" || l.Token != "" {
		t.Fatalf("login after enrolment: next %q; want verify with a challenge and no session", l.Next)
	}
	var verified login
	c.must(http.StatusOK, http.MethodPost, "/auth/mfa/verify", map[string]string{"challenge": l.Challenge, "code": totp.code()}, &verified)
	if verified.Next != "done" || verified.Token == "" {
		t.Fatalf("POST /auth/mfa/verify: next %q; want done with a token", verified.Next)
	}
	c.bearer = verified.Token
	return c
}

// confirm confirms an enrolment from a challenge with the first code and returns the session it opens.
func confirm(t *testing.T, c *api, challenge string, totp *authenticator) string {
	t.Helper()
	var l login
	c.must(http.StatusOK, http.MethodPost, "/auth/mfa/confirm", map[string]string{"challenge": challenge, "code": totp.code()}, &l)
	if l.Next != "done" || l.Token == "" {
		t.Fatalf("POST /auth/mfa/confirm: next %q; want done with a token", l.Next)
	}
	return l.Token
}

// readDirectory reads the clinic's Organization and checks that the PractitionerRole search returns the one
// practitioner with their Practitioner included.
func readDirectory(t *testing.T, c *api, clinic string) {
	t.Helper()
	var org struct{ ResourceType, ID, Name string }
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization/"+clinic, nil, &org)
	if org.ResourceType != "Organization" || org.ID != clinic || org.Name != "E2E synthetic clinic" {
		t.Errorf("GET Organization/%s = %+v", clinic, org)
	}

	var bundle struct {
		ResourceType, Type string
		Entry              []struct {
			Resource struct {
				ResourceType, ID string
				Practitioner     struct{ Reference string }
				Organization     struct{ Reference string }
				Name             []struct {
					Family string
					Given  []string
				}
			}
		}
	}
	q := url.Values{"organization": {clinic}, "_include": {"PractitionerRole:practitioner"}}
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/PractitionerRole?"+q.Encode(), nil, &bundle)
	if bundle.ResourceType != "Bundle" || bundle.Type != "searchset" || len(bundle.Entry) != 2 {
		t.Fatalf("PractitionerRole search = %s %s with %d entries, want a searchset with the role and its practitioner",
			bundle.ResourceType, bundle.Type, len(bundle.Entry))
	}
	role, prac := bundle.Entry[0].Resource, bundle.Entry[1].Resource
	if role.ResourceType != "PractitionerRole" || role.Organization.Reference != "Organization/"+clinic {
		t.Errorf("first entry %s with organization %q, want the clinic's PractitionerRole", role.ResourceType, role.Organization.Reference)
	}
	if prac.ResourceType != "Practitioner" || role.Practitioner.Reference != "Practitioner/"+prac.ID ||
		len(prac.Name) != 1 || prac.Name[0].Family != "Synthetic" || len(prac.Name[0].Given) != 1 || prac.Name[0].Given[0] != "Ada" {
		t.Errorf("second entry %+v, want the invited practitioner the role references (%s)", prac, role.Practitioner.Reference)
	}
}
