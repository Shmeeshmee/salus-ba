package auth

import (
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// The session's transport (docs/accounts/SPEC.md "Transport").
const (
	cookieName = "salus_session"
	// csrfHeader is sent by the web app on every write. A cross-site form cannot set it, so a write
	// authenticated by the cookie without it is refused. It also tells the login endpoints to answer the
	// web app with the cookie rather than the token.
	csrfHeader = "X-Salus-CSRF"
)

// loginJSON is the LoginResponse schema.
type loginJSON struct {
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Next      Next       `json:"next"`
	Challenge string     `json:"challenge,omitempty"`
	Token     string     `json:"token,omitempty"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := h.svc.Login(r.Context(), req.Email, req.Password, clientIP(r, h.trustedProxies))
	h.writeLogin(w, r, res, err)
}

func (h *Handler) enrol(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Challenge string `json:"challenge"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	sess, _ := SessionFrom(r.Context())
	e, err := h.svc.Enrol(r.Context(), req.Challenge, sess)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, map[string]string{"secret": e.Secret, "uri": e.URI})
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	sess, _ := SessionFrom(r.Context())
	res, err := h.svc.Confirm(r.Context(), req.Challenge, sess, req.Code)
	if err == nil && res.Next == "" { // enrolled from a session: no new one
		w.WriteHeader(http.StatusNoContent)
		return
	}
	h.writeLogin(w, r, res, err)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := h.svc.Verify(r.Context(), req.Challenge, req.Code)
	h.writeLogin(w, r, res, err)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.svc.Logout(r.Context(), sess); err != nil {
		h.writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

// writeLogin answers a login step: the challenge for the next step, or the open session, in a cookie for
// the web app and in the body for the mobile app.
func (h *Handler) writeLogin(w http.ResponseWriter, r *http.Request, res LoginResult, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	body := loginJSON{Next: res.Next}
	switch {
	case res.Next != NextDone:
		body.Challenge = res.Challenge
	case r.Header.Get(csrfHeader) == "1":
		body.ExpiresAt = &res.Session.ExpiresAt
		http.SetCookie(w, &http.Cookie{
			Name: cookieName, Value: res.Token, Path: "/", Expires: res.Session.ExpiresAt,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode,
		})
	default:
		body.ExpiresAt, body.Token = &res.Session.ExpiresAt, res.Token
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, body)
}

// Session is the session middleware (mounted in internal/api/router): it authenticates the bearer token or
// the salus_session cookie and puts the session in the request's context. A request with no valid session
// goes on without one, and the handlers that need one answer 401. A write authenticated by the cookie must
// carry X-Salus-CSRF: 1, or it is refused with 403 before the session is read.
func (h *Handler) Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, fromCookie := sessionToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		if fromCookie && !safeMethod(r.Method) && r.Header.Get(csrfHeader) != "1" {
			writeCode(w, http.StatusForbidden, "csrf")
			return
		}
		sess, err := h.svc.Authenticate(r.Context(), token)
		switch {
		case errors.Is(err, ErrUnauthenticated):
			next.ServeHTTP(w, r)
		case err != nil:
			httpx.InternalError(w, r, h.log, err)
		default:
			next.ServeHTTP(w, r.WithContext(WithSession(r.Context(), sess)))
		}
	})
}

// sessionToken returns the bearer token, or else the session cookie's value and true.
func sessionToken(r *http.Request) (string, bool) {
	if t, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return t, false
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value, true
	}
	return "", false
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// clientIP is the connection's remote address. Only when that is a trusted proxy is X-Forwarded-For read,
// from the right: the first address that is not a trusted proxy is the client. The IP is used for the login
// rate limit only; it is never stored or logged.
func clientIP(r *http.Request, trusted []netip.Prefix) string {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	ip := ap.Addr().Unmap()
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0 && isTrusted(ip, trusted); i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		ip = hop.Unmap()
	}
	return ip.String()
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
