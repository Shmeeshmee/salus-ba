package api_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// multipartFile posts content as the multipart part "file", as the web's imports do: the matrix sends JSON, which
// the imports reject before any access, so their refusals are held here with a real file.
func multipartFile(t *testing.T, c *client, path, name string, content []byte) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content)
	_ = w.Close()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.RemoteAddr = "192.0.2.10:5000"
	if c.web {
		r.Header.Set("X-Salus-CSRF", "1")
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	if c.bearer != "" {
		r.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// TestRoleMatrix_RequestsPastTheirValidationAreRefusedToTheRolesTheyAreNotFor holds the routes whose matrix
// request once stopped at a 400 (a missing name, a JSON body to an import, an invite's name) with a request that
// passes the validation: each role the route is not for is refused, the roles it is for are not (WAVE7 integration's
// review; the matrix had them as allowed to every role, which a 400 always passed).
func TestRoleMatrix_RequestsPastTheirValidationAreRefusedToTheRolesTheyAreNotFor(t *testing.T) {
	t.Parallel()
	m := newRoleWorld(t)
	type role struct {
		c    *client
		name string
	}
	roles := []role{{m.a.patient.c, "patient"}, {m.a.practitioner, "practitioner"}, {m.a.receptionist, "receptionist"}, {m.a.admin, "admin"}}
	refusedTo := func(code int) bool { return code == http.StatusForbidden || code == http.StatusNotFound }
	for _, r := range roles {
		code, got := answer(r.c, http.MethodGet, "/clinic/patients?name=Zz", "", nil)
		if want := r.name == "patient"; refusedTo(code) != want || code >= http.StatusInternalServerError {
			t.Errorf("GET /clinic/patients?name=Zz as the %s = %d %.120s, refused want %v", r.name, code, got, want)
		}
		code, got = answer(r.c, http.MethodPost, "/invites",
			`{"email":"valid-`+r.name+`@example.test","role":"practitioner","givenName":"Synthetic","familyName":"Invitee"}`, nil)
		if want := r.name != "admin"; refusedTo(code) != want || code >= http.StatusInternalServerError {
			t.Errorf("POST /invites as the %s = %d %.120s, refused want %v", r.name, code, got, want)
		}
		for _, imp := range []struct{ path, file string }{
			{"/clinic/patients/import", "given_name,family_name\nSynthetic,Importee\n"},
			{"/clinic/services/import", "name,price\nSynthetic import exam,50.00\n"},
		} {
			code, got := multipartFile(t, r.c, imp.path, "import.csv", []byte(imp.file))
			if want := r.name != "admin"; refusedTo(code) != want || code >= http.StatusInternalServerError {
				t.Errorf("POST %s as the %s = %d %.120s, refused want %v", imp.path, r.name, code, got, want)
			}
		}
	}
}
