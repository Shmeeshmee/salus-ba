package binary_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const fileID = "b0000000-0000-4000-8000-0000000000f1"

// stubFiles answers each call with what it is set to and records the requests.
type stubFiles struct {
	err        error
	contentErr error
	upload     binary.Upload
	caller     binary.Caller
	body       []byte // the file's plaintext, for Content
	uploaded   []byte // the body Upload read
	ranges     [][2]int64
	file       binary.Binary
	created    bool
	readBody   bool
}

func (s *stubFiles) Upload(_ context.Context, c binary.Caller, in binary.Upload) (binary.Binary, bool, error) {
	s.caller, s.upload = c, in
	if s.readBody {
		s.uploaded, _ = io.ReadAll(in.Body)
	}
	return s.file, s.created, s.err
}

func (s *stubFiles) Status(_ context.Context, c binary.Caller, id string) (binary.Binary, error) {
	s.caller = c
	return s.file, s.err
}

func (s *stubFiles) Download(_ context.Context, c binary.Caller, id string) (binary.File, error) {
	s.caller = c
	return binary.File{Binary: s.file}, s.err
}

func (s *stubFiles) Content(_ context.Context, f binary.File, from, to int64) (io.ReadCloser, error) {
	s.ranges = append(s.ranges, [2]int64{from, to})
	if s.contentErr != nil {
		return nil, s.contentErr
	}
	return io.NopCloser(bytes.NewReader(s.body[from : to+1])), nil
}

func (s *stubFiles) Delete(_ context.Context, c binary.Caller, id string) error {
	s.caller = c
	return s.err
}

// newHandler is the handler's routes on a mux, signed in as me (or nobody when signedIn is false), logging
// into log.
func newHandler(stub *stubFiles, signedIn bool, log *bytes.Buffer) http.Handler {
	h := binary.NewHTTPHandler(stub, func(context.Context) (binary.Caller, bool, error) { return me, signedIn, nil }, clock.System{}, slog.New(slog.NewJSONHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	h.StreamRoutes(mux)
	return mux
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func cleanPDF() (binary.Binary, []byte) {
	body := bytes.Repeat([]byte("%PDF-placeholder "), 1000)
	sum := sha1.Sum(body) //nolint:gosec // see the import
	return binary.Binary{ID: fileID, ContentType: "application/pdf", Title: title, Status: binary.StatusClean, Size: int64(len(body)), SHA1: sum[:]}, body
}

func TestHTTPHandler_UploadsTheRawBodyWithItsLengthAndTitle(t *testing.T) {
	var log bytes.Buffer
	b, body := cleanPDF()
	b.Status = binary.StatusQuarantined
	for _, c := range []struct {
		disposition string
		title       string
		created     bool
		status      int
	}{
		{`attachment; filename*=UTF-8''Synthetic%20scan%20of%20a%20placeholder.pdf`, title, true, http.StatusCreated},
		{`attachment; filename*=UTF-8''C%3A%5Cfolder%5Cmy%20file.pdf`, "my file.pdf", false, http.StatusOK},
		{`attachment; filename*=UTF-8''%2Fhome%2Fx%2Fscan.pdf`, "scan.pdf", true, http.StatusCreated},
		{`attachment; filename*=UTF-8''` + strings.Repeat("%C5%A1", 250), strings.Repeat("š", 200), true, http.StatusCreated},
		{`attachment; filename*=UTF-8''a%00b%0Ac.pdf`, "abc.pdf", true, http.StatusCreated},
		{`attachment; filename="plain.pdf"`, "plain.pdf", true, http.StatusCreated},
		{`garbage;;`, "", true, http.StatusCreated},
		{"", "", true, http.StatusCreated},
	} {
		stub := &stubFiles{file: b, created: c.created, readBody: true}
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/questionnaire-responses/"+responseA+"/items/files/attachments", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/png") // ignored: the bytes decide
		if c.disposition != "" {
			r.Header.Set("Content-Disposition", c.disposition)
		}
		rec := serve(newHandler(stub, true, &log), r)
		if rec.Code != c.status || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%q: POST = %d (%s), want %d with no-store", c.disposition, rec.Code, rec.Header().Get("Cache-Control"), c.status)
		}
		want := `{"id":"` + fileID + `","contentType":"application/pdf","title":"` + title + `","status":"quarantined","size":` + itoa(len(body)) + `}`
		if got := strings.TrimSpace(rec.Body.String()); got != want {
			t.Errorf("POST body %s,\nwant %s", got, want)
		}
		up := stub.upload
		if up.ResponseID != responseA || up.LinkID != "files" || up.Title != c.title || up.Size != int64(len(body)) || !bytes.Equal(stub.uploaded, body) || stub.caller != me {
			t.Errorf("%q: Upload(%s, %s, %q, %d, %d bytes) by %v", c.disposition, up.ResponseID, up.LinkID, up.Title, up.Size, len(stub.uploaded), stub.caller)
		}
	}
	if strings.Contains(log.String(), "placeholder") {
		t.Errorf("the log holds the title: %s", log.String())
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestHTTPHandler_AnUploadWithoutALengthIsSentAsMinusOne(t *testing.T) {
	stub := &stubFiles{err: binary.ErrLengthRequired}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/questionnaire-responses/"+responseA+"/items/files/attachments", strings.NewReader("x"))
	r.ContentLength = -1
	var log bytes.Buffer
	if rec := serve(newHandler(stub, true, &log), r); rec.Code != http.StatusLengthRequired || strings.TrimSpace(rec.Body.String()) != `{"error":"length_required"}` {
		t.Errorf("POST without a length = %d %s, want 411 length_required", rec.Code, rec.Body)
	}
	if stub.upload.Size != -1 {
		t.Errorf("Upload() size %d, want -1", stub.upload.Size)
	}
}

func TestHTTPHandler_MapsEveryErrorToItsStatusAndCode(t *testing.T) {
	codes := []struct {
		err    error
		code   string
		status int
	}{
		{binary.ErrNotFound, "not_found", 404}, {binary.ErrForbidden, "forbidden", 403},
		{binary.ErrSubmitted, "submitted", 409}, {binary.ErrWithdrawn, "withdrawn", 409}, {binary.ErrPastDue, "past_due", 409},
		{binary.ErrAppointmentClosed, "appointment_closed", 409}, {binary.ErrConsentRequired, "consent_required", 409},
		{binary.ErrNotAttachmentItem, "not_attachment_item", 422}, {binary.ErrLengthRequired, "length_required", 411},
		{binary.ErrShortBody, "invalid_request", 400}, {binary.ErrTooLarge, "too_large", 413}, {binary.ErrResponseFull, "response_full", 413},
		{binary.ErrTooManyFiles, "too_many_files", 409}, {binary.ErrUnsupportedType, "unsupported_type", 415}, {binary.ErrScanning, "scanning", 409},
	}
	routes := map[string]func(ctx context.Context) *http.Request{
		"upload": func(ctx context.Context) *http.Request {
			return httptest.NewRequestWithContext(ctx, http.MethodPost, "/questionnaire-responses/"+responseA+"/items/files/attachments", strings.NewReader("x"))
		},
		"status": func(ctx context.Context) *http.Request {
			return httptest.NewRequestWithContext(ctx, http.MethodGet, "/binaries/"+fileID, nil)
		},
		"content": func(ctx context.Context) *http.Request {
			return httptest.NewRequestWithContext(ctx, http.MethodGet, "/binaries/"+fileID+"/content", nil)
		},
		"delete": func(ctx context.Context) *http.Request {
			return httptest.NewRequestWithContext(ctx, http.MethodDelete, "/binaries/"+fileID, nil)
		},
	}
	var log bytes.Buffer
	for route, req := range routes {
		for _, c := range codes {
			rec := serve(newHandler(&stubFiles{err: c.err}, true, &log), req(t.Context()))
			if rec.Code != c.status || strings.TrimSpace(rec.Body.String()) != `{"error":"`+c.code+`"}` {
				t.Errorf("%s with %v = %d %s, want %d %s", route, c.err, rec.Code, rec.Body, c.status, c.code)
			}
		}
		rec := serve(newHandler(&stubFiles{err: errors.New("the database is down")}, true, &log), req(t.Context()))
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "database") {
			t.Errorf("%s with a failure = %d %s, want a fixed 500", route, rec.Code, rec.Body)
		}
		if rec := serve(newHandler(&stubFiles{}, false, &log), req(t.Context())); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s signed out = %d, want 401", route, rec.Code)
		}
	}
	if strings.Contains(log.String(), "placeholder") {
		t.Errorf("the log holds the title: %s", log.String())
	}
}

func TestHTTPHandler_StatusAndDelete(t *testing.T) {
	b, _ := cleanPDF()
	stub := &stubFiles{file: b}
	var log bytes.Buffer
	rec := serve(newHandler(stub, true, &log), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/binaries/"+fileID, nil))
	want := `{"id":"` + fileID + `","contentType":"application/pdf","title":"` + title + `","status":"clean","size":` + itoa(int(b.Size)) + `}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s (%s), want 200 %s no-store", rec.Code, rec.Body, rec.Header().Get("Cache-Control"), want)
	}
	rec = serve(newHandler(stub, true, &log), httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/binaries/"+fileID, nil))
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("DELETE = %d %s, want 204", rec.Code, rec.Body)
	}
}

// content is a GET of the file's content with the Range header rng ("" for none).
func getContent(t *testing.T, stub *stubFiles, rng string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/binaries/"+fileID+"/content", nil)
	if rng != "" {
		r.Header.Set("Range", rng)
	}
	var log bytes.Buffer
	return serve(newHandler(stub, true, &log), r)
}

func TestHTTPHandler_ServesTheContentWholeOrByOneRange(t *testing.T) {
	b, body := cleanPDF()
	size := int64(len(body))
	last := size - 1
	for _, c := range []struct {
		rng      string
		status   int
		from, to int64
	}{
		{"", 200, 0, last},
		{"bytes=0-0", 206, 0, 0},
		{"bytes=" + itoa(int(last)) + "-", 206, last, last},
		{"bytes=100-199", 206, 100, 199},
		{"bytes=100-", 206, 100, last},
		{"bytes=-500", 206, size - 500, last},
		{"bytes=-99999999", 206, 0, last},
		{"bytes=10-99999999", 206, 10, last},
		{"bytes=0-10,20-30", 200, 0, last}, // several ranges: the whole file
		{"bytes=20-10", 200, 0, last},      // invalid: ignored
		{"items=0-10", 200, 0, last},
		{"bytes=x-10", 200, 0, last},
		{"bytes=10", 200, 0, last},
	} {
		stub := &stubFiles{file: b, body: body}
		rec := getContent(t, stub, c.rng)
		want := body[c.from : c.to+1]
		if rec.Code != c.status || !bytes.Equal(rec.Body.Bytes(), want) || len(stub.ranges) != 1 || stub.ranges[0] != [2]int64{c.from, c.to} {
			t.Errorf("Range %q = %d, %d bytes, read %v; want %d, bytes %d-%d", c.rng, rec.Code, rec.Body.Len(), stub.ranges, c.status, c.from, c.to)
			continue
		}
		h := rec.Header()
		wantHeaders := map[string]string{
			"Content-Type": "application/pdf", "Content-Length": itoa(len(want)), "Accept-Ranges": "bytes",
			"ETag": `"` + hexOf(b.SHA1) + `"`, "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff",
			"Cross-Origin-Resource-Policy": "same-origin", "Content-Security-Policy": "default-src 'none'",
			"Content-Disposition": `inline; filename*=UTF-8''Synthetic%20scan%20of%20a%20placeholder.pdf`,
		}
		if c.status == 206 {
			wantHeaders["Content-Range"] = "bytes " + itoa(int(c.from)) + "-" + itoa(int(c.to)) + "/" + itoa(int(size))
		}
		for k, v := range wantHeaders {
			if got := h.Get(k); got != v {
				t.Errorf("Range %q: %s = %q, want %q", c.rng, k, got, v)
			}
		}
		if c.status == 200 && h.Get("Content-Range") != "" {
			t.Errorf("Range %q: a 200 with Content-Range %q", c.rng, h.Get("Content-Range"))
		}
	}
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, 2*len(b))
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&15])
	}
	return string(out)
}

func TestHTTPHandler_RefusesARangeOutsideTheFile(t *testing.T) {
	b, body := cleanPDF()
	for _, rng := range []string{"bytes=" + itoa(len(body)) + "-", "bytes=99999999-99999999", "bytes=-0"} {
		stub := &stubFiles{file: b, body: body}
		rec := getContent(t, stub, rng)
		if rec.Code != http.StatusRequestedRangeNotSatisfiable || strings.TrimSpace(rec.Body.String()) != `{"error":"range_not_satisfiable"}` ||
			rec.Header().Get("Content-Range") != "bytes */"+itoa(len(body)) || len(stub.ranges) != 0 {
			t.Errorf("Range %q = %d %s (%s), read %v; want 416 bytes */%d", rng, rec.Code, rec.Body, rec.Header().Get("Content-Range"), stub.ranges, len(body))
		}
	}
	stub := &stubFiles{file: b, body: body, contentErr: binary.ErrRangeNotSatisfiable}
	if rec := getContent(t, stub, ""); rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("the service's refusal of the range = %d, want 416", rec.Code)
	}
	stub = &stubFiles{file: b, body: body, contentErr: errors.New("the store is down")}
	if rec := getContent(t, stub, ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("the store's failure = %d, want 500", rec.Code)
	}
}

func TestHTTPHandler_SandboxesEveryTypeButPDF(t *testing.T) {
	for typ, csp := range map[string]string{
		"application/pdf": "default-src 'none'", // Chrome's PDF viewer refuses to render under sandbox
		"image/jpeg":      "default-src 'none'; sandbox", "image/png": "default-src 'none'; sandbox", "video/mp4": "default-src 'none'; sandbox",
	} {
		b, body := cleanPDF()
		b.ContentType, b.Title = typ, ""
		rec := getContent(t, &stubFiles{file: b, body: body}, "")
		if got := rec.Header().Get("Content-Security-Policy"); got != csp {
			t.Errorf("%s: Content-Security-Policy %q, want %q", typ, got, csp)
		}
		if got := rec.Header().Get("Content-Disposition"); got != "inline" {
			t.Errorf("%s without a title: Content-Disposition %q, want inline", typ, got)
		}
	}
}

// slowBody writes n bytes over d, then ends.
func slowBody(n int, d time.Duration) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		const steps = 10
		for range steps {
			time.Sleep(d / steps)
			if _, err := pw.Write(make([]byte, n/steps)); err != nil {
				return
			}
		}
		_ = pw.Close()
	}()
	return pr
}

// An upload slower than the server's read timeout completes: the route sets its own deadlines. The same
// body on a route that does not is cut.
func TestHTTPHandler_AnUploadSlowerThanTheServersReadTimeoutCompletes(t *testing.T) {
	stub := &stubFiles{file: binary.Binary{ID: fileID}, created: true, readBody: true}
	var log bytes.Buffer
	h := binary.NewHTTPHandler(stub, func(context.Context) (binary.Caller, bool, error) { return me, true, nil }, clock.System{}, slog.New(slog.NewJSONHandler(&log, nil)))
	mux := http.NewServeMux()
	h.StreamRoutes(mux)
	mux.HandleFunc("POST /plain", func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.Config.ReadTimeout, srv.Config.WriteTimeout = 300*time.Millisecond, 300*time.Millisecond
	srv.Start()
	defer srv.Close()

	post := func(path string) (int, error) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+path, slowBody(10_000, time.Second))
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = 10_000
		resp, err := srv.Client().Do(req)
		if err != nil {
			return 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode, nil
	}
	if status, err := post("/questionnaire-responses/" + responseA + "/items/files/attachments"); err != nil || status != http.StatusCreated || len(stub.uploaded) != 10_000 {
		t.Errorf("a slow upload = %d, %v, %d bytes read; want 201 with every byte", status, err, len(stub.uploaded))
	}
	if status, err := post("/plain"); err == nil && status == http.StatusOK {
		t.Error("the same body on a route without its own deadlines completed; the test proves nothing")
	}
}

// HEAD matches a GET route: it would open, decrypt and audit the whole file to send no byte of it.
func TestHTTPHandler_RefusesHEADOnTheContent(t *testing.T) {
	b, body := cleanPDF()
	stub := &stubFiles{file: b, body: body}
	rec := serve(newHandler(stub, true, &bytes.Buffer{}), httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/binaries/"+fileID+"/content", nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Errorf("HEAD = %d, Allow %q; want 405 with Allow: GET", rec.Code, rec.Header().Get("Allow"))
	}
	if stub.caller != (binary.Caller{}) || len(stub.ranges) != 0 {
		t.Errorf("HEAD opened the file as %v, read %v", stub.caller, stub.ranges)
	}
}
