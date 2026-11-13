package appointment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// linkFS holds the one-time link's page (ADR 0015 "One-tap links"): page.html and its catalog in each of the ten patient
// languages, templates/link/<language>.json (ICU messages with simple placeholders only; legal ones as {defaultMessage,
// description: "legal"}, checked by make i18n-check), with the date's pattern and the weekdays (from Sunday) and
// months, separated by |, of the form the date uses.
//
//go:embed templates/link
var linkFS embed.FS

// linkLanguages are the page's languages in the order an Accept-Language without an exact match is matched by its
// primary subtag; the first is the fallback (docs/i18n/SPEC.md §3).
var linkLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

// linkStyle is the page's only style, allowed by its hash in the CSP: no other style, script, image or font loads.
const linkStyle = `body{margin:0;background:#f6f8f7;color:#1c2421;font:16px/1.5 "Helvetica Neue",Helvetica,Arial,sans-serif}` +
	`main{max-width:32rem;margin:0 auto;padding:2rem 1rem}h1{font-size:1.5rem;margin:0 0 1rem}h2{font-size:1.1rem;margin:1.5rem 0 .5rem}` +
	`.when{font-weight:600}.state.confirmed,.outcome{color:#1f7a5a;font-weight:600}.notice{background:#fff4e5;border-left:4px solid #c77700;padding:.5rem .75rem}` +
	`form{margin:1rem 0}label{display:block;margin:.5rem 0 1rem}button{font:inherit;padding:.6rem 1.2rem;border-radius:6px;border:1px solid #1f7a5a;` +
	`background:#1f7a5a;color:#fff;cursor:pointer}button.secondary{background:#fff;color:#a8322d;border-color:#a8322d}` +
	`section{border-top:1px solid #dfe6e2;margin-top:1.5rem}.hint{color:#55615c}`

// maxLinkForm bounds the link form's body: action and confirm_cancel.
const maxLinkForm = 256

// LinkService is what the link's page calls (Service).
type LinkService interface {
	Link(ctx context.Context, token, addr string) (LinkPage, error)
	UseLink(ctx context.Context, token, addr string, action LinkAction, confirmCancel bool) error
}

// LinkHandler serves the page of a reminder's one-time link, GET and POST /a/{token} (api/openapi.yaml): HTML without
// JavaScript, cookies or third-party requests, never cached or indexed, in a frame nowhere, and referred by its origin
// only (strict-origin: under no-referrer browsers send "Origin: null" with the form, which the POST refuses). A GET
// changes nothing; the POST, from the page's own origin only, confirms or cancels and answers 303 to the page. The token
// is in the path only: the access log records the route pattern, and no error carries it.
type LinkHandler struct {
	svc      LinkService
	log      *slog.Logger
	page     *template.Template
	catalogs map[string]map[string]string
	csp      string
	trusted  []netip.Prefix // the proxies whose X-Forwarded-For names the client, for the rate limit
}

// NewLinkHandler returns the handler on svc. The page and its catalogs are part of the binary, so a broken one is a
// programming error the tests catch: it panics.
func NewLinkHandler(svc LinkService, trusted []netip.Prefix, log *slog.Logger) *LinkHandler {
	catalogs := map[string]map[string]string{}
	for _, lang := range linkLanguages {
		b, err := fs.ReadFile(linkFS, "templates/link/"+lang+".json")
		if err != nil {
			panic(err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			panic(fmt.Sprintf("appointment: templates/link/%s.json: %v", lang, err))
		}
		cat := make(map[string]string, len(raw))
		for k, v := range raw {
			var legal struct {
				DefaultMessage string `json:"defaultMessage"`
			}
			if json.Unmarshal(v, &legal) == nil { // a legal message
				cat[k] = legal.DefaultMessage
			} else if err := json.Unmarshal(v, &legal.DefaultMessage); err == nil {
				cat[k] = legal.DefaultMessage
			} else {
				panic(fmt.Sprintf("appointment: templates/link/%s.json %s: %v", lang, k, err))
			}
		}
		catalogs[lang] = cat
	}
	sum := sha256.Sum256([]byte(linkStyle))
	return &LinkHandler{
		svc: svc, log: log, trusted: trusted, catalogs: catalogs,
		page: template.Must(template.New("page.html").ParseFS(linkFS, "templates/link/page.html")),
		csp: "default-src 'none'; style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'; form-action 'self'; " +
			"frame-ancestors 'none'; base-uri 'none'",
	}
}

// Routes declares the link's routes on mux.
func (h *LinkHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /a/{token}", h.show)
	mux.HandleFunc("POST /a/{token}", h.act)
}

func (h *LinkHandler) show(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "")
}

func (h *LinkHandler) act(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		h.write(w, r, http.StatusForbidden, pageData{Lang: acceptLanguage(r), Notice: "forbidden"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLinkForm)
	action := LinkAction(r.PostFormValue("action"))
	if action != LinkConfirm && action != LinkCancel {
		h.render(w, r, http.StatusBadRequest, "invalid_request")
		return
	}
	token := r.PathValue("token")
	err := h.svc.UseLink(r.Context(), token, httpx.ClientIP(r, h.trusted), action, r.PostFormValue("confirm_cancel") == "yes")
	switch {
	case errors.Is(err, ErrConfirmCancelRequired):
		h.render(w, r, http.StatusBadRequest, "confirm_cancel_required")
	case errors.Is(err, ErrLinkRateLimited):
		h.write(w, r, http.StatusTooManyRequests, pageData{Lang: acceptLanguage(r), Notice: "rate_limited"})
	case err == nil, errors.Is(err, ErrLinkInvalid), errors.Is(err, ErrClosed), errors.Is(err, ErrCancellationWindowClosed):
		// The page shows what is true now: the outcome, the link no longer working, or the clinic to contact.
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "strict-origin")
		http.Redirect(w, r, "/a/"+url.PathEscape(token), http.StatusSeeOther)
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

// render looks the link up and writes its page with status, and the notice (a catalog key) when set.
func (h *LinkHandler) render(w http.ResponseWriter, r *http.Request, status int, notice string) {
	p, err := h.svc.Link(r.Context(), r.PathValue("token"), httpx.ClientIP(r, h.trusted))
	d := pageData{Lang: p.Language, Notice: notice, Page: p}
	switch {
	case errors.Is(err, ErrLinkRateLimited):
		d, status = pageData{Notice: "rate_limited"}, http.StatusTooManyRequests
	case errors.Is(err, ErrLinkInvalid):
		d.Invalid = true
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	default:
		d.Open = p.Used == ""
	}
	if d.Lang == "" {
		d.Lang = acceptLanguage(r)
	}
	h.write(w, r, status, d)
}

// pageData is what page.html is filled with; write fills T, When, Used, Dir and Style.
type pageData struct {
	T       map[string]string
	Lang    string
	Dir     string
	Notice  string // a catalog key, then its text
	When    string
	Used    string
	Style   template.CSS
	Page    LinkPage
	Open    bool
	Invalid bool
}

// write renders d with status and the page's headers.
func (h *LinkHandler) write(w http.ResponseWriter, r *http.Request, status int, d pageData) {
	t, ok := h.catalogs[d.Lang]
	if !ok {
		d.Lang, t = linkLanguages[0], h.catalogs[linkLanguages[0]]
	}
	d.T, d.Style, d.Dir = t, template.CSS(linkStyle), "ltr"
	if d.Lang == "ar" {
		d.Dir = "rtl"
	}
	if d.Notice != "" {
		d.Notice = t[d.Notice]
	}
	if d.Page.Clinic != "" {
		d.When = fill(t["appointment"], map[string]string{"clinic": d.Page.Clinic, "when": when(t, d.Page)})
	}
	switch d.Page.Used {
	case LinkConfirm:
		d.Used = t["used_confirm"]
	case LinkCancel:
		d.Used = t["used_cancel"]
	}
	var b bytes.Buffer
	if err := h.page.Execute(&b, d); err != nil {
		httpx.InternalError(w, r, h.log, err)
		return
	}
	for k, v := range map[string]string{
		"Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store", "Referrer-Policy": "strict-origin",
		"X-Robots-Tag": "noindex", "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Content-Security-Policy": h.csp,
	} {
		w.Header().Set(k, v)
	}
	w.WriteHeader(status)
	_, _ = w.Write(b.Bytes()) // the status is sent; a failed write means the client went away
}

// when writes the page's start, already in the clinic's zone, by the catalog's date pattern: 24-hour time and Latin
// digits in every language (docs/i18n/SPEC.md §3 "Formatting").
func when(t map[string]string, p LinkPage) string {
	weekdays, months := strings.Split(t["weekdays"], "|"), strings.Split(t["months"], "|")
	if len(weekdays) != 7 || len(months) != 12 {
		return p.Start.Format("2006-01-02 15:04") + " (" + p.TimeZone + ")"
	}
	return fill(t["date"], map[string]string{
		"weekday": weekdays[p.Start.Weekday()], "day": strconv.Itoa(p.Start.Day()), "month": months[p.Start.Month()-1],
		"year": strconv.Itoa(p.Start.Year()), "time": p.Start.Format("15:04"), "zone": p.TimeZone,
	})
}

// fill replaces each {name} of the message with its value (the catalogs hold simple placeholders only).
func fill(msg string, args map[string]string) string {
	pairs := make([]string, 0, 2*len(args))
	for k, v := range args {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(msg)
}

// sameOrigin reports whether the request's Origin is the page's own: its host is the request's Host. A browser sends
// Origin with every form POST; a request without one, or with "null", is refused.
func sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" || o == "null" {
		return false
	}
	u, err := url.Parse(o)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host)
}

// acceptLanguage is the first of the page's languages the request's Accept-Language names, exactly or by its primary
// subtag, and "" for none (write then uses the fallback).
func acceptLanguage(r *http.Request) string {
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		for _, l := range linkLanguages {
			if strings.EqualFold(tag, l) {
				return l
			}
		}
		primary, _, _ := strings.Cut(tag, "-")
		for _, l := range linkLanguages {
			if p, _, _ := strings.Cut(l, "-"); strings.EqualFold(primary, p) {
				return l
			}
		}
	}
	return ""
}
