package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
)

func updateMeRequest(t *testing.T, accountID, body string) *http.Request {
	ctx := t.Context()
	if accountID != "" {
		ctx = auth.WithSession(ctx, auth.Session{ID: sessionID, AccountID: accountID})
	}
	return httptest.NewRequestWithContext(ctx, http.MethodPatch, "/auth/me", strings.NewReader(body))
}

// signInBodies are the three requests that carry the languages, with the rest of a valid body.
var signInBodies = map[string]func(t *testing.T, languages string) *http.Request{
	"register": func(t *testing.T, languages string) *http.Request {
		return post(t, "/auth/register", `{"email":"ana@example.test","password":"`+password+`"`+languages+`}`)
	},
	"login": func(t *testing.T, languages string) *http.Request {
		return post(t, "/auth/login", `{"email":"ana@example.test","password":"`+password+`"`+languages+`}`)
	},
	"accept invite": func(t *testing.T, languages string) *http.Request {
		return post(t, "/invites/accept", `{"token":"`+token+`","password":"`+password+`"`+languages+`}`)
	},
}

func TestSignInHandlers_PassTheLanguagesOn(t *testing.T) {
	tests := []struct {
		languages string
		want      [2]string
	}{
		{languages: ``},
		{languages: `,"language":null,"detectedLanguage":null`},
		{languages: `,"language":"de","detectedLanguage":"ar"`, want: [2]string{"de", "ar"}},
		{languages: `,"detectedLanguage":"sr-Cyrl"`, want: [2]string{"", "sr-Cyrl"}},
		{languages: `,"language":"xx"`, want: [2]string{"xx", ""}}, // the service refuses it
	}
	for name, request := range signInBodies {
		for _, tt := range tests {
			svc := &stubService{account: patientAccount(), result: doneResult()}

			rec := serve(t, svc, discard(), request(t, tt.languages))

			if rec.Code >= 300 || svc.languages == nil || *svc.languages != tt.want {
				t.Errorf("%s with %s = %d %s, languages %v; want them passed on as %v", name, tt.languages, rec.Code, rec.Body, svc.languages, tt.want)
			}
		}
	}
}

func TestSignInHandlers_RefuseALanguageThatIsNotATag(t *testing.T) {
	for name, request := range signInBodies {
		for _, languages := range []string{`,"language":""`, `,"detectedLanguage":""`, `,"language":7`, `,"detectedLanguage":["de"]`} {
			svc := &stubService{}

			rec := serve(t, svc, discard(), request(t, languages))

			if want := `{"error":"invalid_request"}`; rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != want || svc.languages != nil {
				t.Errorf("%s with %s = %d %s, service called: %v; want 400 %s", name, languages, rec.Code, rec.Body, svc.languages != nil, want)
			}
		}
	}
}

func TestSignInHandlers_MapLanguageErrors(t *testing.T) {
	for name, request := range signInBodies {
		rec := serve(t, &stubService{err: auth.ErrInvalidLanguage}, discard(), request(t, `,"language":"xx"`))
		if want := `{"error":"invalid_request"}`; rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("%s with ErrInvalidLanguage = %d %s, want 400 %s", name, rec.Code, rec.Body, want)
		}
	}
}

func TestUpdateMeHandler_SetsOrClearsTheLanguage(t *testing.T) {
	for body, want := range map[string]string{`{"language":"hr"}`: "hr", `{"language":null}`: ""} {
		a := patientAccount()
		a.Language = want
		svc := &stubService{account: a}

		rec := serve(t, svc, discard(), updateMeRequest(t, accountID, body))

		if svc.setLanguage == nil || *svc.setLanguage != want || svc.meFor != accountID {
			t.Fatalf("PATCH /auth/me %s: SetLanguage(%q, %v), want (%q, %q)", body, svc.meFor, svc.setLanguage, accountID, want)
		}
		language := ""
		if want != "" {
			language = `,"language":"` + want + `"`
		}
		wantBody := `{"id":"` + accountID + `","email":"ana@example.test","role":"patient"` + language + `,"mfaEnrolled":false}` + "\n"
		if rec.Code != http.StatusOK || rec.Body.String() != wantBody {
			t.Errorf("PATCH /auth/me %s = %d %s, want 200 %s", body, rec.Code, rec.Body, wantBody)
		}
	}
}

func TestUpdateMeHandler_RefusesABodyThatIsNotTheRequest(t *testing.T) {
	for _, body := range []string{`{}`, `{"language":""}`, `{"language":7}`, `{"language":"hr","role":"admin"}`, `{`, `{"language":"hr"}{}`} {
		svc := &stubService{}

		rec := serve(t, svc, discard(), updateMeRequest(t, accountID, body))

		if want := `{"error":"invalid_request"}`; rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != want || svc.setLanguage != nil {
			t.Errorf("PATCH /auth/me %s = %d %s, service called: %v; want 400 %s", body, rec.Code, rec.Body, svc.setLanguage != nil, want)
		}
	}
}

func TestUpdateMeHandler_MapsErrorsToFixedCodes(t *testing.T) {
	tests := []struct {
		err     error
		account string
		code    string
		status  int
	}{
		{account: "", code: "unauthenticated", status: http.StatusUnauthorized},
		{err: auth.ErrNotFound, account: accountID, code: "unauthenticated", status: http.StatusUnauthorized},
		{err: auth.ErrInvalidLanguage, account: accountID, code: "invalid_request", status: http.StatusBadRequest},
		{err: auth.ErrLanguageUnavailable, account: accountID, code: "language_unavailable", status: http.StatusUnprocessableEntity},
		{err: context.DeadlineExceeded, account: accountID, code: "internal error", status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		rec := serve(t, &stubService{err: tt.err}, discard(), updateMeRequest(t, tt.account, `{"language":"de"}`))
		if want := `{"error":"` + tt.code + `"}`; rec.Code != tt.status || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("PATCH /auth/me with %v = %d %s, want %d %s", tt.err, rec.Code, rec.Body, tt.status, want)
		}
	}
}

func TestMeHandler_ReturnsTheChosenLanguageOnly(t *testing.T) {
	a := patientAccount()
	a.Language, a.DetectedLanguage = "sr-Cyrl", "de"

	rec := serve(t, &stubService{account: a}, discard(), meRequest(t, accountID))

	want := `{"id":"` + accountID + `","email":"ana@example.test","role":"patient","language":"sr-Cyrl","mfaEnrolled":false}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET /auth/me = %d %s, want 200 %s (the detected language stays on the server)", rec.Code, rec.Body, want)
	}
}
