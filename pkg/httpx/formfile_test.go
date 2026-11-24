package httpx_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// upload is a multipart request whose part field holds data, after a part named other.
func upload(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("other", "x"); err != nil {
		t.Fatal(err)
	}
	w, err := mw.CreateFormFile(field, "rows.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestFormFile_ReadsThePartOrSaysWhyNot(t *testing.T) {
	plain := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader("a;b"))
	for name, tc := range map[string]struct {
		r      *http.Request
		want   string
		status int
	}{
		"the part":         {r: upload(t, "file", []byte("a;b")), want: "a;b"},
		"no such part":     {r: upload(t, "other2", []byte("a;b")), status: http.StatusBadRequest},
		"not a form":       {r: plain, status: http.StatusBadRequest},
		"over the limit":   {r: upload(t, "file", bytes.Repeat([]byte("x"), 2000)), status: http.StatusRequestEntityTooLarge},
		"headers too long": {r: upload(t, "file"+strings.Repeat("x", 1100), nil), status: http.StatusRequestEntityTooLarge},
	} {
		file, data, status := httpx.FormFile(httptest.NewRecorder(), tc.r, "file", 1000)
		if string(data) != tc.want || status != tc.status || (status == 0) != (file == "rows.csv") {
			t.Errorf("%s: FormFile() = %q %q, %d, want %q, %d", name, file, data, status, tc.want, tc.status)
		}
	}
}
