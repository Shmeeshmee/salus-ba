package appointment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

const pageToken = "q83vEjRWeJCrze8SNFZ4kKvN7xI0VniQq83vEjRWeJA" // 43 characters of base64url

// stubLinks answers the link handler's calls and records them.
type stubLinks struct {
	page   appointment.LinkPage
	err    error
	useErr error
	calls  []string
}

func (s *stubLinks) Link(_ context.Context, token, addr string) (appointment.LinkPage, error) {
	s.calls = append(s.calls, "Link "+token+" "+addr)
	return s.page, s.err
}

func (s *stubLinks) UseLink(_ context.Context, token, addr string, action appointment.LinkAction, confirmCancel bool) error {
	s.calls = append(s.calls, "UseLink "+token+" "+addr+" "+string(action)+map[bool]string{true: " ticked", false: ""}[confirmCancel])
	return s.useErr
}

func serveLink(t *testing.T, svc *stubLinks, r *http.Request) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var log bytes.Buffer
	mux := http.NewServeMux()
	appointment.NewLinkHandler(svc, nil, slog.New(slog.NewTextHandler(&log, nil))).Routes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w, log.String()
}

func openPage(language string) appointment.LinkPage {
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	return appointment.LinkPage{
		Start: time.Date(2030, 1, 7, 10, 0, 0, 0, sarajevo), Clinic: "Synthetic <clinic>", TimeZone: "Europe/Sarajevo", Language: language,
		CanConfirm: true, CanCancel: true,
	}
}

func post(form url.Values, origin string) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://api.salus.invalid/a/"+pageToken, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.RemoteAddr = "203.0.113.7:5000"
	return r
}

func TestLinkPage_ShowsTheAppointmentAndItsActionsWithoutScriptOrCaching(t *testing.T) {
	svc := &stubLinks{page: openPage("en")}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil)
	r.RemoteAddr = "203.0.113.7:5000"
	w, _ := serveLink(t, svc, r)
	body := w.Body.String()
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("GET = %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	for header, want := range map[string]string{
		"Cache-Control": "no-store", "Referrer-Policy": "strict-origin", "X-Robots-Tag": "noindex", "X-Content-Type-Options": "nosniff",
		"X-Frame-Options": "DENY",
	} {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	csp := w.Header().Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "form-action 'self'", "frame-ancestors 'none'", "style-src 'sha256-"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q lacks %q", csp, want)
		}
	}
	for _, want := range []string{
		`<html lang="en" dir="ltr">`, "Appointment at Synthetic &lt;clinic&gt; on Monday 7 January 2030, 10:00 (Europe/Sarajevo).",
		`value="confirm"`, "Confirm I will come", `name="confirm_cancel" value="yes"`, "Yes, I want to cancel this appointment",
		"To choose another time, open the Salus app.", `method="post"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q:\n%s", want, body)
		}
	}
	for _, never := range []string{"<script", "http://", "https://", pageToken, "<clinic>"} {
		if strings.Contains(body, never) {
			t.Errorf("page carries %q", never)
		}
	}
	if want := []string{"Link " + pageToken + " 203.0.113.7"}; !slices.Equal(svc.calls, want) {
		t.Errorf("calls %v, want %v", svc.calls, want)
	}
}

func TestLinkPage_InTheLinksLanguageRightToLeftForArabic(t *testing.T) {
	svc := &stubLinks{page: openPage("ar")}
	w, _ := serveLink(t, svc, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil))
	if body := w.Body.String(); !strings.Contains(body, `<html lang="ar" dir="rtl">`) || !strings.Contains(body, "الاثنين 7 يناير 2030") {
		t.Errorf("page is not in Arabic, right to left:\n%s", body)
	}
}

func TestLinkPage_StatesItShows(t *testing.T) {
	used := openPage("bs")
	used.Used, used.CanConfirm, used.CanCancel = appointment.LinkCancel, false, false
	confirmed := openPage("en")
	confirmed.Confirmed, confirmed.CanConfirm = true, false
	closed := openPage("en")
	closed.CanCancel, closed.CancelClosed = false, true
	for _, tc := range []struct {
		name   string
		svc    *stubLinks
		accept string
		want   []string
		never  []string
		status int
	}{
		{"used: its outcome only", &stubLinks{page: used}, "", []string{"Otkazali ste ovaj termin.", "Synthetic &lt;clinic&gt;"}, []string{"<form", "<button"}, http.StatusOK},
		{"confirmed: cancel only", &stubLinks{page: confirmed}, "", []string{"You have confirmed that you will come.", `value="cancel"`}, []string{`value="confirm"`}, http.StatusOK},
		{"inside the 24 hours", &stubLinks{page: closed}, "", []string{"contact the clinic", `value="confirm"`}, []string{`value="cancel"`}, http.StatusOK},
		{"no longer works, in the link's language", &stubLinks{page: appointment.LinkPage{Language: "de"}, err: appointment.ErrLinkInvalid}, "",
			[]string{"Dieser Link funktioniert nicht mehr."}, []string{"<form"}, http.StatusOK},
		{"unknown link, in the browser's language", &stubLinks{err: appointment.ErrLinkInvalid}, "fr-CH, en;q=0.5",
			[]string{"Ce lien ne fonctionne plus."}, []string{"<form"}, http.StatusOK},
		{"unknown link, Bosnian by default", &stubLinks{err: appointment.ErrLinkInvalid}, "ja", []string{"Ovaj link više ne radi."}, nil, http.StatusOK},
		{"rate limited", &stubLinks{err: appointment.ErrLinkRateLimited}, "en", []string{"Too many requests"}, []string{"<form"}, http.StatusTooManyRequests},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil)
			if tc.accept != "" {
				r.Header.Set("Accept-Language", tc.accept)
			}
			w, _ := serveLink(t, tc.svc, r)
			body := w.Body.String()
			if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("GET = %d (Cache-Control %q), want %d", w.Code, w.Header().Get("Cache-Control"), tc.status)
			}
			for _, s := range tc.want {
				if !strings.Contains(body, s) {
					t.Errorf("page lacks %q:\n%s", s, body)
				}
			}
			for _, s := range tc.never {
				if strings.Contains(body, s) {
					t.Errorf("page carries %q", s)
				}
			}
		})
	}
}

func TestLinkPage_AFailureIsA500WithoutTheTokenInTheLog(t *testing.T) {
	w, log := serveLink(t, &stubLinks{err: errBackend}, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil))
	if w.Code != http.StatusInternalServerError || strings.Contains(log, pageToken) || strings.Contains(w.Body.String(), "database") {
		t.Errorf("GET = %d %q, log %q; want a fixed 500 and no token logged", w.Code, w.Body.String(), log)
	}
}

func TestLinkUse_ConfirmsAndCancelsThenSeeOtherToThePage(t *testing.T) {
	for _, tc := range []struct {
		form url.Values
		err  error
		want string
	}{
		{url.Values{"action": {"confirm"}}, nil, "UseLink " + pageToken + " 203.0.113.7 confirm"},
		{url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, nil, "UseLink " + pageToken + " 203.0.113.7 cancel ticked"},
		{url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, appointment.ErrCancellationWindowClosed, "UseLink " + pageToken + " 203.0.113.7 cancel ticked"},
		{url.Values{"action": {"confirm"}}, appointment.ErrLinkInvalid, "UseLink " + pageToken + " 203.0.113.7 confirm"},
		{url.Values{"action": {"confirm"}}, appointment.ErrClosed, "UseLink " + pageToken + " 203.0.113.7 confirm"},
	} {
		svc := &stubLinks{useErr: tc.err}
		w, _ := serveLink(t, svc, post(tc.form, "http://api.salus.invalid"))
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/a/"+pageToken || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("POST %v (%v) = %d to %q, want 303 to the page", tc.form, tc.err, w.Code, w.Header().Get("Location"))
		}
		if want := []string{tc.want}; !slices.Equal(svc.calls, want) {
			t.Errorf("calls %v, want %v", svc.calls, want)
		}
	}
}

func TestLinkUse_Refusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		r      *http.Request
		svc    *stubLinks
		want   string
		status int
		calls  int
	}{
		{"no Origin", post(url.Values{"action": {"confirm"}}, ""), &stubLinks{}, "The request was refused", http.StatusForbidden, 0},
		{"a foreign Origin", post(url.Values{"action": {"confirm"}}, "https://attacker.invalid"), &stubLinks{}, "The request was refused", http.StatusForbidden, 0},
		{"Origin null", post(url.Values{"action": {"confirm"}}, "null"), &stubLinks{}, "The request was refused", http.StatusForbidden, 0},
		{"no action", post(url.Values{}, "http://api.salus.invalid"), &stubLinks{page: openPage("en")}, "The request was not understood", http.StatusBadRequest, 1},
		{"an unknown action", post(url.Values{"action": {"delete"}}, "http://api.salus.invalid"), &stubLinks{page: openPage("en")}, "The request was not understood", http.StatusBadRequest, 1},
		{"a cancel without the box", post(url.Values{"action": {"cancel"}}, "http://api.salus.invalid"),
			&stubLinks{page: openPage("en"), useErr: appointment.ErrConfirmCancelRequired}, "tick the box", http.StatusBadRequest, 2},
		{"rate limited", post(url.Values{"action": {"confirm"}}, "http://api.salus.invalid"), &stubLinks{useErr: appointment.ErrLinkRateLimited}, "Too many requests", http.StatusTooManyRequests, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.r.Header.Set("Accept-Language", "en")
			w, _ := serveLink(t, tc.svc, tc.r)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.want) || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Errorf("POST = %d, want %d with %q:\n%s", w.Code, tc.status, tc.want, w.Body.String())
			}
			if len(tc.svc.calls) != tc.calls {
				t.Errorf("calls %v, want %d", tc.svc.calls, tc.calls)
			}
			for _, c := range tc.svc.calls {
				if strings.HasPrefix(c, "UseLink") && tc.status == http.StatusForbidden {
					t.Errorf("a refused POST used the link: %v", tc.svc.calls)
				}
			}
		})
	}
}

// Every catalog of the link page has the source's keys, and its messages only the placeholders the page fills.
func TestLinkCatalogs_HaveTheSourcesKeysAndOnlyKnownPlaceholders(t *testing.T) {
	dir := filepath.Join("templates", "link")
	read := func(lang string) map[string]any {
		b, err := os.ReadFile(filepath.Join(dir, lang+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s.json: %v", lang, err)
		}
		return m
	}
	en := read("en")
	files, _ := fs.Glob(os.DirFS(dir), "*.json")
	if len(files) != 10 {
		t.Errorf("catalogs %v, want the ten patient languages", files)
	}
	placeholder := regexp.MustCompile(`\{[^}]*\}`)
	known := []string{"{clinic}", "{when}", "{weekday}", "{day}", "{month}", "{year}", "{time}", "{zone}"}
	for _, f := range files {
		m := read(strings.TrimSuffix(f, ".json"))
		for k := range en {
			if _, ok := m[k]; !ok {
				t.Errorf("%s lacks %s", f, k)
			}
		}
		for k, v := range m {
			if _, ok := en[k]; !ok {
				t.Errorf("%s has an extra key %s", f, k)
			}
			s, ok := v.(string)
			if obj, isObj := v.(map[string]any); isObj {
				s, ok = obj["defaultMessage"].(string)
			}
			if !ok || strings.Contains(s, "''") {
				t.Errorf("%s %s: %v is not a plain message", f, k, v)
			}
			for _, p := range placeholder.FindAllString(s, -1) {
				if !slices.Contains(known, p) {
					t.Errorf("%s %s: unknown placeholder %s", f, k, p)
				}
			}
		}
	}
}

// The access log names the route's pattern, never the path that carries the token; nor does a refusal or an error.
func TestLinkPage_TheAccessLogNamesTheRouteNotTheToken(t *testing.T) {
	var log bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&log, nil))
	mux := http.NewServeMux()
	appointment.NewLinkHandler(&stubLinks{page: openPage("en"), useErr: errBackend}, nil, logger).Routes(mux)
	h := middleware.AccessLog(logger, mux)(mux)
	for _, r := range []*http.Request{
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil),
		post(url.Values{"action": {"confirm"}}, "http://api.salus.invalid"),
		post(url.Values{"action": {"confirm"}}, "https://attacker.invalid"),
	} {
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	if got := log.String(); strings.Contains(got, pageToken) || !strings.Contains(got, "route=\"GET /a/{token}\"") {
		t.Errorf("log:\n%s\nwant the route patterns and no token", got)
	}
}

// Browsers send an Origin with a form POST only under a referrer policy that allows it (Fetch, "append a request
// Origin header": no-referrer gives "null"): strict-origin, which also keeps the token's path out of any Referer.
func TestLinkPage_ItsReferrerPolicyLetsBrowsersSendTheOrigin(t *testing.T) {
	w, _ := serveLink(t, &stubLinks{page: openPage("en")}, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+pageToken, nil))
	if got := w.Header().Get("Referrer-Policy"); got != "strict-origin" {
		t.Errorf("Referrer-Policy %q, want strict-origin", got)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<meta name="referrer" content="strict-origin">`) || strings.Contains(body, "no-referrer") {
		t.Errorf("the page's referrer meta is not strict-origin:\n%s", body)
	}
}
