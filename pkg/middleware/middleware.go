// Package middleware holds generic HTTP middleware. It is mounted only in internal/api/router.
//
// Nothing here logs a path, query string, header value or body: those can carry personal data. A request
// is identified by its request id and its route pattern.
package middleware

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

type requestIDKey struct{}

// safeRequestID is what an incoming X-Request-ID must look like to be kept: short, and nothing that could
// break a log line or carry an e-mail address.
var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID keeps a safe incoming X-Request-ID or generates one, puts it in the context and echoes it in
// the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !safeRequestID.MatchString(id) {
			id = rand.Text()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom returns the request id in ctx, or "" when there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// LogHandler adds the context's request id to every record logged with a context.
type LogHandler struct {
	slog.Handler
}

// NewLogHandler wraps h.
func NewLogHandler(h slog.Handler) *LogHandler { return &LogHandler{Handler: h} }

// Handle adds request_id when ctx has one.
func (h *LogHandler) Handle(ctx context.Context, rec slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		rec.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, rec)
}

// WithAttrs keeps the wrapper.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup keeps the wrapper.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithGroup(name)}
}

// Router finds the pattern a request matches; *http.ServeMux is one.
type Router interface {
	Handler(r *http.Request) (h http.Handler, pattern string)
}

// AccessLog logs one line per request: method, route pattern, status, duration. The pattern is looked up
// on routes rather than read from r.Pattern, because handlers further in (http.TimeoutHandler) serve a copy
// of the request.
func AccessLog(log *slog.Logger, routes Router) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			_, route := routes.Handler(r)
			if route == "" {
				route = "unmatched"
			}
			log.InfoContext(r.Context(), "request",
				"method", r.Method,
				"route", route,
				"status", rec.statusOr200(),
				"duration_ms", float64(time.Since(start).Microseconds())/1000,
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *statusRecorder) statusOr200() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// Recover turns a panic into a logged, fixed 500. http.ErrAbortHandler is passed on: it is how a handler
// asks the server to drop the connection.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				if p == http.ErrAbortHandler { // panicked as is, never wrapped
					panic(p)
				}
				log.ErrorContext(r.Context(), "panic", "panic", p, "stack", string(debug.Stack()))
				httpx.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			}()
			next.ServeHTTP(w, r)
		})
	}
}
