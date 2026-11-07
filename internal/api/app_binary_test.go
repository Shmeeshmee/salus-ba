package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// Synthetic: a placeholder item and file name, asserted absent from the log.
const (
	fileTitle = "Synthetic placeholder scan.pdf"
	filesBody = `{"title":"Placeholder questionnaire with a file","item":[` +
		`{"linkId":"f1","text":"Placeholder upload","type":"attachment","accept":["document"],"maxFiles":2},` +
		`{"linkId":"q1","text":"Placeholder yes or no","type":"boolean"}]}`
)

// filePDF is a generated PDF of a few kilobytes.
func filePDF() []byte {
	return append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("% placeholder content line\n"), 300)...)
}

// send is a request with a raw body and headers as c.
func send(c *client, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	c.t.Helper()
	r := httptest.NewRequestWithContext(c.t.Context(), method, path, bytes.NewReader(body))
	if c.bearer != "" {
		r.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
		r.Header.Set("X-Salus-CSRF", "1")
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, r)
	return rec
}

// The files through the JSON API with row-level security on (salus_app) and the compose object store: the
// patient uploads to an attachment item of their open response, reads its status, cannot download it while it
// is quarantined, downloads it whole and by a range once clean; the practitioner sees nothing before the submit;
// staff cannot upload or delete; the patient deletes it. Every call is audited and the log holds no title.
func TestNew_ServesTheFilesOfAResponseUnderItsAccessRuleAndAuditsThem(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	m := newMatrix(t, app.Handler, owner, cfg.FHIRBaseURL)
	r := request(t, m, m.future, publishQuestionnaire(t, m.practitioner, filesBody))
	path := "/questionnaire-responses/" + r.ID + "/items/f1/attachments"
	body := filePDF()
	disposition := map[string]string{"Content-Disposition": "attachment; filename*=UTF-8''Synthetic%20placeholder%20scan.pdf", "Content-Type": "application/octet-stream"}

	rec := send(m.patient, http.MethodPost, path, body, disposition)
	var file struct{ ID, ContentType, Title, Status string }
	decode(t, rec, &file)
	if rec.Code != http.StatusCreated || file.ContentType != "application/pdf" || file.Title != fileTitle || file.Status != "quarantined" {
		t.Fatalf("the upload = %d %s, want 201 quarantined", rec.Code, rec.Body)
	}
	t.Cleanup(func() { send(m.patient, http.MethodDelete, "/binaries/"+file.ID, nil, nil) })
	if rec := send(m.patient, http.MethodPost, path, body, disposition); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), file.ID) {
		t.Errorf("the identical retry = %d %s, want 200 with the first file", rec.Code, rec.Body)
	}
	checkFileRefusals(t, m, path, file.ID, body)
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE id = $1`, file.ID); err != nil {
		t.Fatal(err)
	}
	content := "/binaries/" + file.ID + "/content"
	if rec := send(m.patient, http.MethodGet, content, nil, nil); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), body) {
		t.Errorf("the patient's download = %d, %d bytes; want 200 with the file", rec.Code, rec.Body.Len())
	}
	if rec := send(m.patient, http.MethodGet, content, nil, map[string]string{"Range": "bytes=9-99"}); rec.Code != http.StatusPartialContent ||
		!bytes.Equal(rec.Body.Bytes(), body[9:100]) || rec.Header().Get("Content-Range") != "bytes 9-99/"+strconv.Itoa(len(body)) {
		t.Errorf("the patient's range = %d %s, want 206 with bytes 9 to 99", rec.Code, rec.Header().Get("Content-Range"))
	}
	if rec := send(m.practitioner, http.MethodGet, content, nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("the practitioner's download before the submit = %d, want 404", rec.Code)
	}
	if rec := send(m.patient, http.MethodDelete, "/binaries/"+file.ID, nil, nil); rec.Code != http.StatusNoContent {
		t.Errorf("the patient's deletion = %d %s, want 204", rec.Code, rec.Body)
	}
	if rec := send(m.patient, http.MethodGet, "/binaries/"+file.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("the status after the deletion = %d, want 404", rec.Code)
	}
	checkFileAudit(t, owner, file.ID, r.ID)
	for _, s := range []string{fileTitle, "Synthetic%20placeholder", "placeholder content"} {
		if strings.Contains(log.String(), s) {
			t.Errorf("the log holds %q", s)
		}
	}
}

// checkFileRefusals: what the patient, staff and nobody may not do with the quarantined file id.
func checkFileRefusals(t *testing.T, m matrix, path, id string, body []byte) {
	t.Helper()
	for _, c := range []struct {
		who    *client
		name   string
		method string
		path   string
		code   string
		body   []byte
		status int
	}{
		{m.patient, "a download while quarantined", http.MethodGet, "/binaries/" + id + "/content", "scanning", nil, http.StatusConflict},
		{m.patient, "a text file", http.MethodPost, path, "unsupported_type", []byte("plain text, no file type"), http.StatusUnsupportedMediaType},
		{m.patient, "a boolean item", http.MethodPost, strings.Replace(path, "/f1/", "/q1/", 1), "not_attachment_item", body, http.StatusUnprocessableEntity},
		{m.practitioner, "the practitioner's upload", http.MethodPost, path, "forbidden", body, http.StatusForbidden},
		{m.practitioner, "the practitioner's status read before the submit", http.MethodGet, "/binaries/" + id, "not_found", nil, http.StatusNotFound},
		{m.practitioner, "the practitioner's deletion", http.MethodDelete, "/binaries/" + id, "forbidden", nil, http.StatusForbidden},
		{m.other, "another practitioner's upload", http.MethodPost, path, "not_found", body, http.StatusNotFound},
		{m.anonymous, "nobody's upload", http.MethodPost, path, "unauthenticated", body, http.StatusUnauthorized},
	} {
		rec := send(c.who, c.method, c.path, c.body, nil)
		var e struct{ Error string }
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		if rec.Code != c.status || e.Error != c.code {
			t.Errorf("%s = %d %s, want %d %s", c.name, rec.Code, rec.Body, c.status, c.code)
		}
	}
	rec := send(m.patient, http.MethodGet, "/binaries/"+id, nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"quarantined"`) {
		t.Errorf("the patient's status read = %d %s, want 200 quarantined", rec.Code, rec.Body)
	}
}

// checkFileAudit: every call on the file named it and its response, the refusals included.
func checkFileAudit(t *testing.T, owner *pgxpool.Pool, id, response string) {
	t.Helper()
	var created, read, deleted, failed int
	err := owner.QueryRow(t.Context(), `SELECT
		count(*) FILTER (WHERE subtype_code = 'create' AND outcome = 0),
		count(*) FILTER (WHERE subtype_code = 'read' AND outcome = 0),
		count(*) FILTER (WHERE subtype_code = 'delete' AND outcome = 0),
		count(*) FILTER (WHERE outcome = 4)
		FROM audit_event WHERE entity_reference = 'Binary/' || $1::text`, id).Scan(&created, &read, &deleted, &failed)
	if err != nil {
		t.Fatal(err)
	}
	// created: the upload and its retry; read: the status read and two downloads; failed: the quarantined
	// download and the practitioner's deletion (the others disclose no file)
	if created != 2 || read != 3 || deleted != 1 || failed != 2 {
		t.Errorf("Binary/%s audited %d creates, %d reads, %d deletes, %d refusals; want 2, 3, 1, 2", id, created, read, deleted, failed)
	}
	var paired int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event b JOIN audit_event r USING (request_id, subtype_code, outcome)
		WHERE b.entity_reference = 'Binary/' || $1::text AND r.entity_reference = 'QuestionnaireResponse/' || $2::text`, id, response).Scan(&paired); err != nil {
		t.Fatal(err)
	}
	if paired != created+read+deleted+failed {
		t.Errorf("%d of the file's events have their response's beside them, want all %d", paired, created+read+deleted+failed)
	}
}
