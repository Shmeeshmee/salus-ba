package api_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// probeIDs are malformed path parameters: not a UUID, a NUL byte, invalid UTF-8, and one far too long.
var probeIDs = []string{"not-a-uuid", "%00", "%C0%AF", strings.Repeat("a", 300)}

// probeBodies are malformed JSON bodies: cut short, null, of the wrong type, and an object of wrong field types.
var probeBodies = []string{"{", "null", "[1]", `"synthetic"`, `{"id":1,"name":[],"status":{},"version":"x"}`}

// probeSkipped are the routes the probe leaves: they end the signed-in session it goes on with.
var probeSkipped = []string{"POST /auth/logout", "POST /me/account/close"}

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

// BG4: every route of the contract, called by the clinic's admin, its practitioner and its receptionist with malformed
// path ids and malformed bodies, answers a client error (or succeeds), never a 5xx: a malformed request is the
// caller's fault.
func TestStaffRoutes_MalformedIDsAndBodiesNeverAnswer500(t *testing.T) {
	t.Parallel()
	r, _ := newReception(t)
	routes := make([]string, 0, len(receptionRoutes))
	for op := range receptionRoutes {
		if !slices.Contains(probeSkipped, op) {
			routes = append(routes, op)
		}
	}
	slices.Sort(routes)
	for who, as := range map[string]*client{"admin": r.admin, "practitioner": r.practitioner, "receptionist": r.receptionist} {
		for _, op := range routes {
			for _, req := range probeRequests(r, op) {
				as.ifMatch = ""
				if req.method == http.MethodPatch || req.method == http.MethodPut {
					as.ifMatch = `W/"1"` // past the precondition
				}
				if rec := as.do(req.method, req.path, req.body); rec.Code >= http.StatusInternalServerError {
					t.Errorf("%s as the %s, %.40q with %q: %d %s", op, who, req.path, req.body, rec.Code, rec.Body)
				}
			}
		}
	}
}

type probeRequest struct{ method, path, body string }

// probeRequests are op's malformed requests: each malformed id in every path parameter (with an empty object as the
// body of a write), then each malformed body on the route's real path.
func probeRequests(r reception, op string) []probeRequest {
	method, route, _ := strings.Cut(op, " ")
	write := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
	var out []probeRequest
	if pathParam.MatchString(route) {
		body := ""
		if write {
			body = "{}"
		}
		for _, id := range probeIDs {
			path := pathParam.ReplaceAllLiteralString(route, id)
			if q, ok := receptionQueries[route]; ok {
				path += "?" + q
			}
			out = append(out, probeRequest{method, path, body})
		}
	}
	if write {
		for _, body := range probeBodies {
			out = append(out, probeRequest{method, receptionPath(r, route, true), body})
		}
	}
	return out
}
