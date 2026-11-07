package binary

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// The routes' own deadlines (SPEC.md §2 "Timeouts and body limits"): a 200 MB video from a phone takes far
// longer than the server's read timeout and the request timeout, from which these routes are exempt.
const (
	UploadDeadline  = 30 * time.Minute
	ContentDeadline = 10 * time.Minute // per request: a video seeks with many
)

// Files are the calls the handler serves (Service).
type Files interface {
	Upload(ctx context.Context, c Caller, in Upload) (Binary, bool, error)
	Status(ctx context.Context, c Caller, id string) (Binary, error)
	Download(ctx context.Context, c Caller, id string) (File, error)
	Content(ctx context.Context, f File, from, to int64) (io.ReadCloser, error)
	Delete(ctx context.Context, c Caller, id string) error
}

// HTTPHandler serves the files of the JSON API (api/openapi.yaml, "attachments"). Errors are
// {"error": "<code>"} with a fixed code per case; no title, byte or content type reaches the log.
type HTTPHandler struct {
	svc Files
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	clock  clock.Clock // for the streaming routes' deadlines
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Files, caller func(ctx context.Context) (Caller, bool, error), clk clock.Clock, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, clock: clk, log: log}
}

// Routes declares the status and deletion routes on mux; they keep the request timeout.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /binaries/{id}", h.status)
	mux.HandleFunc("DELETE /binaries/{id}", h.delete)
}

// StreamRoutes declares the upload and the content routes on mux. They stream, set their own read and write
// deadlines, and are mounted without the request timeout or a body limit (internal/api/router).
func (h *HTTPHandler) StreamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /questionnaire-responses/{id}/items/{linkId}/attachments", h.upload)
	mux.HandleFunc("GET /binaries/{id}/content", h.content)
}

// binaryJSON is the Binary schema of the JSON API.
type binaryJSON struct {
	ID          string `json:"id"`
	ContentType string `json:"contentType"`
	Title       string `json:"title"`
	Status      Status `json:"status"`
	Size        int64  `json:"size"`
}

// upload reads the raw body, exactly Content-Length bytes (none given is -1); the title is the
// Content-Disposition's file name, never anything from the URL.
func (h *HTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	deadlines(w, h.clock.Now().Add(UploadDeadline))
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	b, created, err := h.svc.Upload(r.Context(), c, Upload{
		Body: r.Body, ResponseID: r.PathValue("id"), LinkID: r.PathValue("linkId"),
		Title: titleOf(r.Header.Get("Content-Disposition")), Size: r.ContentLength,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeBinary(w, status, b)
}

func (h *HTTPHandler) status(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	b, err := h.svc.Status(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeBinary(w, http.StatusOK, b)
}

func (h *HTTPHandler) delete(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// content sends the file, or the one range asked for; HEAD is refused (405). The read is audited and committed before the first
// byte; a failure after it cuts the connection, so the client sees a short body, never a wrong one.
func (h *HTTPHandler) content(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead { // the GET route matches HEAD: it would decrypt and audit the file for no byte
		w.Header().Set("Allow", http.MethodGet)
		writeCode(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	deadlines(w, h.clock.Now().Add(ContentDeadline))
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	f, err := h.svc.Download(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	from, to, partial, ok := byteRange(r.Header.Get("Range"), f.Size)
	if !ok {
		unsatisfiable(w, f.Size)
		return
	}
	body, err := h.svc.Content(r.Context(), f, from, to)
	switch {
	case errors.Is(err, ErrRangeNotSatisfiable):
		unsatisfiable(w, f.Size)
		return
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	}
	defer func() { _ = body.Close() }()
	hd := w.Header()
	hd.Set("Content-Type", f.ContentType)
	hd.Set("Content-Length", strconv.FormatInt(to-from+1, 10))
	hd.Set("Accept-Ranges", "bytes")
	hd.Set("ETag", `"`+hex.EncodeToString(f.SHA1)+`"`)
	hd.Set("Cache-Control", "private, no-store")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Resource-Policy", "same-origin")
	hd.Set("Content-Disposition", disposition(downloadName(f.Title, f.ContentType)))
	hd.Set("Content-Security-Policy", contentPolicy(f.ContentType))
	status := http.StatusOK
	if partial {
		status = http.StatusPartialContent
		hd.Set("Content-Range", "bytes "+strconv.FormatInt(from, 10)+"-"+strconv.FormatInt(to, 10)+"/"+strconv.FormatInt(f.Size, 10))
	}
	w.WriteHeader(status)
	if _, err := io.Copy(w, body); err != nil {
		h.log.WarnContext(r.Context(), "content stream failed", "route", r.Pattern, "binary_id", f.ID, "error", err)
		panic(http.ErrAbortHandler) // the status is sent: cut the connection rather than end the body cleanly
	}
}

// contentPolicy is the Content-Security-Policy of a file of the type: nothing may load from it, and every
// type the browser still shows sandboxed is sandboxed. A PDF is not: Chrome's PDF viewer refuses to render a
// sandboxed document, in a tab or an <object> (SPEC.md §2 "Download", recorded in PROGRESS.md 2.1).
func contentPolicy(contentType string) string {
	if contentType == "application/pdf" {
		return "default-src 'none'"
	}
	return "default-src 'none'; sandbox"
}

// deadlines gives the request's connection read and write deadlines at at, past the server's own. A writer
// that cannot (a test's recorder) keeps the server's.
func deadlines(w http.ResponseWriter, at time.Time) {
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(at)
	_ = rc.SetWriteDeadline(at)
}

// byteRange is the one range of the Range header h on a file of size bytes (RFC 9110 §14.1.2): from and to
// inclusive, partial when a range was asked for; false when the range starts past the end (416). No header,
// several ranges and a header that does not parse give the whole file.
func byteRange(h string, size int64) (from, to int64, partial, ok bool) {
	spec, isBytes := strings.CutPrefix(strings.TrimSpace(h), "bytes=")
	first, last, isRange := strings.Cut(spec, "-")
	first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	if !isBytes || !isRange || strings.Contains(spec, ",") {
		return 0, size - 1, false, true
	}
	if first == "" { // the last n bytes
		n, err := strconv.ParseInt(last, 10, 64)
		switch {
		case err != nil || n < 0:
			return 0, size - 1, false, true
		case n == 0:
			return 0, 0, false, false
		}
		return max(0, size-n), size - 1, true, true
	}
	a, err := strconv.ParseInt(first, 10, 64)
	if err != nil || a < 0 {
		return 0, size - 1, false, true
	}
	b := size - 1
	if last != "" {
		if b, err = strconv.ParseInt(last, 10, 64); err != nil || b < a {
			return 0, size - 1, false, true
		}
	}
	if a >= size {
		return 0, 0, false, false
	}
	return a, min(b, size-1), true, true
}

func unsatisfiable(w http.ResponseWriter, size int64) {
	w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(size, 10))
	writeCode(w, http.StatusRequestedRangeNotSatisfiable, "range_not_satisfiable")
}

// titleOf is the file name of a Content-Disposition (its filename*, RFC 6266, or filename): the path
// removed, control characters dropped, cut to MaxTitle characters; "" for none.
func titleOf(header string) string {
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}
	name := params["filename"]
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == unicode.ReplacementChar {
			return -1
		}
		return r
	}, name))
	if r := []rune(name); len(r) > MaxTitle {
		name = string(r[:MaxTitle])
	}
	return name
}

// extensions are the file name extensions of each detected content type, the one a download gets first.
var extensions = map[string][]string{
	"image/jpeg": {".jpg", ".jpeg"}, "image/png": {".png"}, "application/pdf": {".pdf"}, "video/mp4": {".mp4"},
}

// downloadName is the title as a file name every system saves: the characters Windows refuses in a file name
// replaced with "_", and the detected type's extension added when the title does not end in one. "" stays "".
func downloadName(title, contentType string) string {
	if title == "" {
		return ""
	}
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`:\/*?"<>|`, r) {
			return '_'
		}
		return r
	}, title)
	exts := extensions[contentType]
	for _, ext := range exts {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			return name
		}
	}
	if len(exts) > 0 {
		name += exts[0]
	}
	return name
}

// disposition is the Content-Disposition of a download: inline, with the title as filename* (RFC 6266).
func disposition(title string) string {
	if title == "" {
		return "inline"
	}
	var b strings.Builder
	b.WriteString("inline; filename*=UTF-8''")
	for _, c := range []byte(title) {
		if c < 0x80 && (unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || strings.IndexByte("!#$&+-.^_`|~", c) >= 0) {
			b.WriteByte(c)
			continue
		}
		b.WriteString("%" + strings.ToUpper(hex.EncodeToString([]byte{c})))
	}
	return b.String()
}

// signedIn is the request's caller, or false after answering 401 or 500.
func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Caller{}, false
	}
	return c, true
}

func writeBinary(w http.ResponseWriter, status int, b Binary) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, binaryJSON{ID: b.ID, ContentType: b.ContentType, Size: b.Size, Title: b.Title, Status: b.Status})
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrSubmitted, "submitted", http.StatusConflict},
	{ErrWithdrawn, "withdrawn", http.StatusConflict},
	{ErrPastDue, "past_due", http.StatusConflict},
	{ErrAppointmentClosed, "appointment_closed", http.StatusConflict},
	{ErrConsentRequired, "consent_required", http.StatusConflict},
	{ErrNotAttachmentItem, "not_attachment_item", http.StatusUnprocessableEntity},
	{ErrLengthRequired, "length_required", http.StatusLengthRequired},
	{ErrShortBody, "invalid_request", http.StatusBadRequest},
	{ErrTooLarge, "too_large", http.StatusRequestEntityTooLarge},
	{ErrResponseFull, "response_full", http.StatusRequestEntityTooLarge},
	{ErrTooManyFiles, "too_many_files", http.StatusConflict},
	{ErrUnsupportedType, "unsupported_type", http.StatusUnsupportedMediaType},
	{ErrScanning, "scanning", http.StatusConflict},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			writeCode(w, c.status, c.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
