package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The chat's files and search through api.New (SAL-30) with the compose Postgres and object store: the patient
// uploads a PDF to the conversation (scanning), sends it as a message without a text; nobody opens it while it is
// scanned, nor before it is sent but its uploader; once clean the practitioner reads it; a rejected file never
// opens; the search finds the messages each may read; after the patient's revocation the practitioner reads and
// finds nothing. The log holds no file name and no message text.
func TestNew_ServesTheChatsFilesAndSearchUnderTheChatsAccess(t *testing.T) {
	c := newConsultationApp(t)
	c.book(t, wallNow().Add(-48*time.Hour))
	rec := c.d.patient.do(http.MethodPost, "/conversations", `{"organization_id":"`+inviteClinic+`","practitioner_role_id":"`+c.d.roleID+`"}`)
	var conv struct{ ID string }
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &conv) != nil {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	path := "/conversations/" + conv.ID
	body := filePDF()
	disposition := map[string]string{"Content-Disposition": "attachment; filename*=UTF-8''Synthetic%20lab%20result.pdf", "Content-Type": "application/octet-stream"}
	upload := func() string {
		rec := send(c.d.patient, http.MethodPost, path+"/attachments", body, disposition)
		var a struct{ ID, Status, Filename string }
		if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &a) != nil || a.Status != "scanning" || a.Filename != "Synthetic lab result.pdf" {
			t.Fatalf("upload = %d %s", rec.Code, rec.Body)
		}
		return a.ID
	}
	file, infected := upload(), upload()
	if rec := send(c.d.practitioner, http.MethodGet, path+"/attachments/"+file+"/content", nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("the practitioner opens the patient's unsent file: %d, want 404", rec.Code)
	}
	rec = c.d.patient.do(http.MethodPost, path+"/messages", `{"client_message_id":"c1000000-0000-4000-8000-0000000000c1","body":"","attachment_ids":["`+file+`","`+infected+`"]}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"attachments":[`) || strings.Contains(rec.Body.String(), `"body"`) {
		t.Fatalf("send = %d %s", rec.Code, rec.Body)
	}
	content := path + "/attachments/" + file + "/content"
	if rec := send(c.d.practitioner, http.MethodGet, content, nil, nil); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "scanning") {
		t.Errorf("while scanning: %d %s, want 409 scanning", rec.Code, rec.Body)
	}
	if _, err := c.owner.Exec(t.Context(), `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE id = $1`, file); err != nil {
		t.Fatal(err)
	}
	if _, err := c.owner.Exec(t.Context(), `UPDATE "binary" SET status = 'rejected', scanned_at = now(), wrapped_key = NULL WHERE id = $1`, infected); err != nil {
		t.Fatal(err)
	}
	if rec := send(c.d.practitioner, http.MethodGet, content, nil, nil); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), body) ||
		rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("the practitioner's download = %d, %d bytes", rec.Code, rec.Body.Len())
	}
	if rec := send(c.d.practitioner, http.MethodGet, path+"/attachments/"+infected+"/content", nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("the rejected file = %d, want 404", rec.Code)
	}
	expectHTTP(t, c.d.patient, http.MethodGet, path+"/messages", http.StatusOK, `"status":"clean"`, `"status":"rejected"`)
	if rec := send(c.stranger, http.MethodGet, content, nil, nil); rec.Code == http.StatusOK {
		t.Errorf("a stranger's download = %d", rec.Code)
	}

	if rec := c.d.patient.do(http.MethodPost, path+"/messages", `{"client_message_id":"c2000000-0000-4000-8000-0000000000c2","body":"Synthetic mučnina question"}`); rec.Code != http.StatusCreated {
		t.Fatalf("send = %d %s", rec.Code, rec.Body)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/conversations/search?q=mucnina", http.StatusOK, `"conversation_id":"`+conv.ID+`"`, "Synthetic mučnina question")
	expectHTTP(t, c.d.patient, http.MethodGet, path+"/search?q=MUCN", http.StatusOK, "Synthetic mučnina question")
	expectHTTP(t, c.stranger, http.MethodGet, "/conversations/search?q=mucnina", http.StatusForbidden) // no patient profile

	expectHTTP(t, c.d.patient, http.MethodPost, "/consents/"+c.consent+"/revoke", http.StatusOK)
	if rec := send(c.d.practitioner, http.MethodGet, content, nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("the practitioner's download after the revocation = %d, want 404", rec.Code)
	}
	expectHTTP(t, c.d.practitioner, http.MethodGet, "/conversations/search?q=mucnina", http.StatusOK, `{"results":[]}`)
	for _, s := range []string{"Synthetic lab result", "Synthetic%20lab", "mučnina", "placeholder content"} {
		if strings.Contains(c.log.String(), s) {
			t.Errorf("the log holds %q", s)
		}
	}
}
