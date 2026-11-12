package communication_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// sal30Stub is what the stub answers for the files, searches, receipts and typing (SAL-30), and what it was asked.
type sal30Stub struct {
	err       error // of Upload, Attachment, Search and SearchAll
	uploaded  []communication.Upload
	body      []byte // the upload's bytes as the handler streamed them
	file      []byte // the opened file's bytes
	results   []communication.SearchResult
	receipts  []communication.Receipt
	typing    []string // participant ids
	opened    []string // "<attachment id> <thumbnail>"
	searches  []string // "<conversation id or all> <q>"
	attachRes communication.Attachment
}

func (s *stubChats) Upload(_ context.Context, _ communication.Caller, _ string, in communication.Upload) (communication.Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sal30.uploaded = append(s.sal30.uploaded, communication.Upload{Filename: in.Filename, Size: in.Size})
	s.sal30.body, _ = io.ReadAll(in.Body)
	return s.sal30.attachRes, s.sal30.err
}

func (s *stubChats) Attachment(_ context.Context, _ communication.Caller, _, id string, thumbnail bool) (binary.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sal30.opened = append(s.sal30.opened, id+" "+strconv.FormatBool(thumbnail))
	if s.sal30.err != nil {
		return binary.File{}, s.sal30.err
	}
	return binary.File{Binary: binary.Binary{ID: id, ContentType: "image/png", Title: "Synthetic scan.png", Size: int64(len(s.sal30.file))}}, nil
}

func (s *stubChats) Content(_ context.Context, _ binary.File, from, to int64) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return io.NopCloser(bytes.NewReader(s.sal30.file[from : to+1])), nil
}

func (s *stubChats) Search(_ context.Context, _ communication.Caller, id, q string) ([]communication.SearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sal30.searches = append(s.sal30.searches, id+" "+q)
	return s.sal30.results, s.sal30.err
}

func (s *stubChats) SearchAll(_ context.Context, _ communication.Caller, q string) ([]communication.SearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sal30.searches = append(s.sal30.searches, "all "+q)
	return s.sal30.results, s.sal30.err
}

func (s *stubChats) Typing(_ context.Context, _, participantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sal30.typing = append(s.sal30.typing, participantID)
	return nil
}

func (s *stubChats) Receipts(context.Context, communication.Caller, string) ([]communication.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sal30.receipts, nil
}

const fileID = "f1000000-0000-4000-8000-0000000000f1"

// The upload route streams the raw body to the service with its length and the Content-Disposition's file name, and
// answers 201 with the attachment, scanning; the files' refusals keep binary's codes.
func TestHTTP_UploadStreamsTheBodyAndAnswersTheAttachment(t *testing.T) {
	t.Parallel()
	svc := &stubChats{}
	svc.sal30.attachRes = communication.Attachment{ID: fileID, ContentType: "application/pdf", Filename: "lab.pdf", Status: communication.AttachmentScanning, Size: 4}
	srv := newServer(t, svc, newFakeLive(), true)
	code, body, h := do(t, srv, http.MethodPost, convPath+"/attachments", "%PDF", map[string]string{
		"Content-Disposition": `attachment; filename*=UTF-8''lab%20results.pdf`, "Content-Type": "application/octet-stream",
	})
	if code != http.StatusCreated || h.Get("Cache-Control") != "no-store" {
		t.Fatalf("upload = %d %s %v", code, body, h)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil || got["status"] != "scanning" || got["thumbnail"] != false || got["filename"] != "lab.pdf" {
		t.Errorf("body = %s", body)
	}
	if up := svc.sal30.uploaded; len(up) != 1 || up[0].Filename != "lab results.pdf" || up[0].Size != 4 || string(svc.sal30.body) != "%PDF" {
		t.Errorf("uploaded %+v %q", up, svc.sal30.body)
	}
	for err, want := range map[error]string{
		binary.ErrUnsupportedType: "415 unsupported_type", binary.ErrTooLarge: "413 too_large", binary.ErrLengthRequired: "411 length_required",
		binary.ErrShortBody: "400 invalid_request", binary.ErrTooManyFiles: "409 too_many_files", communication.ErrConsentRequired: "403 consent_required", communication.ErrNotFound: "404 not_found",
	} {
		svc.sal30.err = err
		code, body, _ := do(t, srv, http.MethodPost, convPath+"/attachments", "x", nil)
		if got := strconv.Itoa(code) + " " + errorCode(t, body); got != want {
			t.Errorf("%v: %s, want %s", err, got, want)
		}
	}
}

func errorCode(t *testing.T, body string) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal([]byte(body), &e)
	return e.Error
}

// The content and thumbnail routes open the file through the service and serve it as binary serves a file: its
// type, no-store, nosniff, a range; scanning is 409, HEAD 405.
func TestHTTP_AnAttachmentsContentAndThumbnailAreServedOnceOpened(t *testing.T) {
	t.Parallel()
	svc := &stubChats{}
	svc.sal30.file = []byte("synthetic image bytes")
	srv := newServer(t, svc, newFakeLive(), true)
	path := convPath + "/attachments/" + fileID
	code, body, h := do(t, srv, http.MethodGet, path+"/content", "", nil)
	if code != http.StatusOK || body != "synthetic image bytes" || h.Get("Content-Type") != "image/png" || h.Get("Cache-Control") != "private, no-store" ||
		h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("content = %d %q %v", code, body, h)
	}
	if code, body, _ := do(t, srv, http.MethodGet, path+"/thumbnail", "", map[string]string{"Range": "bytes=0-8"}); code != http.StatusPartialContent || body != "synthetic" {
		t.Errorf("thumbnail range = %d %q", code, body)
	}
	if got := strings.Join(svc.sal30.opened, ","); got != fileID+" false,"+fileID+" true" {
		t.Errorf("opened %s", got)
	}
	svc.sal30.err = binary.ErrScanning
	if code, body, _ := do(t, srv, http.MethodGet, path+"/content", "", nil); code != http.StatusConflict || errorCode(t, body) != "scanning" {
		t.Errorf("scanning = %d %s", code, body)
	}
	if code, _, _ := do(t, srv, http.MethodHead, path+"/content", "", nil); code != http.StatusMethodNotAllowed {
		t.Errorf("HEAD = %d, want 405", code)
	}
}

// The search routes: within a conversation and across the caller's, {results: [...]} newest first as the service
// gives them, an empty list for none; a refused query is 400.
func TestHTTP_SearchAnswersTheResultsWithinAndAcrossConversations(t *testing.T) {
	t.Parallel()
	svc := &stubChats{}
	srv := newServer(t, svc, newFakeLive(), true)
	if code, body, h := do(t, srv, http.MethodGet, "/conversations/search?q=rash", "", nil); code != http.StatusOK || body != `{"results":[]}` ||
		h.Get("Cache-Control") != "no-store" {
		t.Errorf("none = %d %q", code, body)
	}
	svc.sal30.results = []communication.SearchResult{{ConversationID: sample.ConversationID, Message: sample}}
	code, body, _ := do(t, srv, http.MethodGet, convPath+"/search?q=rash+cream", "", nil)
	var got struct {
		Results []struct {
			ConversationID string          `json:"conversation_id"`
			Message        json.RawMessage `json:"message"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(body), &got); code != http.StatusOK || err != nil || len(got.Results) != 1 || got.Results[0].ConversationID != sample.ConversationID {
		t.Errorf("within = %d %s", code, body)
	}
	if want := "all rash," + sample.ConversationID + " rash cream"; strings.Join(svc.sal30.searches, ",") != want {
		t.Errorf("searches %v, want %s", svc.sal30.searches, want)
	}
	svc.sal30.err = communication.ErrInvalid
	if code, body, _ := do(t, srv, http.MethodGet, "/conversations/search?q=", "", nil); code != http.StatusBadRequest || errorCode(t, body) != "invalid_request" {
		t.Errorf("an empty query = %d %s", code, body)
	}
}

// A send names its files; the body may then be empty.
func TestHTTP_SendPassesTheAttachmentIDs(t *testing.T) {
	t.Parallel()
	svc := &stubChats{sent: sample, created: true}
	srv := newServer(t, svc, newFakeLive(), true)
	code, body, _ := do(t, srv, http.MethodPost, convPath+"/messages", `{"client_message_id":"`+clientID1+`","body":"","attachment_ids":["`+fileID+`"]}`, nil)
	if code != http.StatusCreated {
		t.Fatalf("send = %d %s", code, body)
	}
	if calls := svc.calls(); len(calls) != 1 || calls[0] != "send "+clientID1+"  "+fileID {
		t.Errorf("calls %q", calls)
	}
}
