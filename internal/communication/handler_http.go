package communication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a request body: a message of MaxBody characters of up to four bytes each, and its JSON.
const maxBody = 4*MaxBody + 1024

// Chats is what the handler calls (Service).
type Chats interface {
	Open(ctx context.Context, c Caller, in Open) (Conversation, bool, error)
	List(ctx context.Context, c Caller, before *ListCursor) ([]Conversation, *ListCursor, error)
	Get(ctx context.Context, c Caller, id string) (Conversation, error)
	Messages(ctx context.Context, c Caller, id string, before *Cursor, limit int) (Page, error)
	Send(ctx context.Context, c Caller, id, clientMessageID, body string, attachmentIDs ...string) (Message, bool, error)
	Edit(ctx context.Context, c Caller, id, messageID, body string, version int32) (Message, error)
	Revisions(ctx context.Context, c Caller, id, messageID string) ([]Revision, error)
	MarkRead(ctx context.Context, c Caller, id string, seq int64) error
	Subscribe(ctx context.Context, c Caller, id, after string) (Subscription, error)
	After(ctx context.Context, c Caller, id string, after Cursor) ([]Message, error)
	// SAL-30: files, search, receipts and typing.
	Upload(ctx context.Context, c Caller, id string, in Upload) (Attachment, error)
	Attachment(ctx context.Context, c Caller, id, attachmentID string, thumbnail bool) (binary.File, error)
	Content(ctx context.Context, f binary.File, from, to int64) (io.ReadCloser, error)
	Search(ctx context.Context, c Caller, id, q string) ([]SearchResult, error)
	SearchAll(ctx context.Context, c Caller, q string) ([]SearchResult, error)
	Typing(ctx context.Context, conversationID, participantID string) error
	Receipts(ctx context.Context, c Caller, conversationID string) ([]Receipt, error)
}

// Subscriber gives a channel's live frames in this process (pkg/redis.Hub).
type Subscriber interface {
	Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error)
}

// HTTPHandler serves the chat's JSON API and its socket (api/openapi.yaml, tag chat). Errors are
// {"error": "<code>"}; internal error text goes to the log only, and no log line holds a message.
type HTTPHandler struct {
	svc    Chats
	live   Subscriber
	caller func(ctx context.Context) (Caller, bool, error)
	// active reports whether the request's session is still open (auth.Handler.Active): a socket asks it at every
	// check, so a logout or a revoked session ends it.
	active func(r *http.Request) (bool, error)
	clock  clock.Clock
	log    *slog.Logger
	// check is how often an open socket checks access again and fills from Postgres (CheckEvery; tests shorten it).
	check time.Duration
}

// CheckEvery is how often a socket checks the caller's access again and looks in Postgres for messages the live
// channel dropped (docs/plans/chat.md "Who reads and writes").
const CheckEvery = 5 * time.Second

// NewHTTPHandler returns the handler on svc and the live channel; check is CheckEvery but in tests; active tells
// whether a socket's session is still open.
func NewHTTPHandler(svc Chats, live Subscriber, caller func(ctx context.Context) (Caller, bool, error), active func(r *http.Request) (bool, error),
	clk clock.Clock, check time.Duration, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, live: live, caller: caller, active: active, clock: clk, check: check, log: log}
}

// Routes declares the chat's routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /conversations", h.open)
	mux.HandleFunc("GET /conversations", h.list)
	mux.HandleFunc("GET /conversations/{id}", h.get)
	mux.HandleFunc("GET /conversations/{id}/messages", h.messages)
	mux.HandleFunc("POST /conversations/{id}/messages", h.post)
	mux.HandleFunc("PATCH /conversations/{id}/messages/{message_id}", h.edit)
	mux.HandleFunc("GET /conversations/{id}/messages/{message_id}/revisions", h.revisions)
	mux.HandleFunc("PUT /conversations/{id}/read", h.read)
	mux.HandleFunc("GET /conversations/{id}/search", h.search)
	mux.HandleFunc("GET /conversations/search", h.searchAll)
}

// StreamRoutes declares the conversation's socket on mux: it lives as long as the socket, so it is served without
// the request timeout (router.Config.Streams).
func (h *HTTPHandler) StreamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /conversations/{id}/events", h.events)
	mux.HandleFunc("POST /conversations/{id}/attachments", h.upload)
	mux.HandleFunc("GET /conversations/{id}/attachments/{attachment_id}/content", h.content)
	mux.HandleFunc("GET /conversations/{id}/attachments/{attachment_id}/thumbnail", h.thumbnail)
}

func (h *HTTPHandler) open(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in Open
	if !decode(w, r, &in) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	conv, created, err := h.svc.Open(r.Context(), c, in)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	noStore(w)
	httpx.JSON(w, status, conv)
}

// listJSON is the ConversationList schema.
type listJSON struct {
	Next          string         `json:"next,omitempty"`
	Conversations []Conversation `json:"conversations"`
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var before *ListCursor
	if s := r.URL.Query().Get("before"); s != "" {
		lc, err := parseListCursor(s)
		if err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_cursor")
			return
		}
		before = &lc
	}
	convs, next, err := h.svc.List(r.Context(), c, before)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := listJSON{Conversations: convs}
	if out.Conversations == nil {
		out.Conversations = []Conversation{}
	}
	if next != nil {
		out.Next = listCursorString(*next)
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, out)
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	conv, err := h.svc.Get(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, conv)
}

// pageJSON is the MessagePage schema.
type pageJSON struct {
	Next     string    `json:"next,omitempty"`
	Messages []Message `json:"messages"`
}

func (h *HTTPHandler) messages(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	var before *Cursor
	if s := q.Get("before"); s != "" {
		cur, err := ParseCursor(s)
		if err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_cursor")
			return
		}
		before = &cur
	}
	limit := 0
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		limit = n
	}
	page, err := h.svc.Messages(r.Context(), c, r.PathValue("id"), before, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := pageJSON{Messages: page.Messages}
	if out.Messages == nil {
		out.Messages = []Message{}
	}
	if page.Next != nil {
		out.Next = page.Next.String()
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, out)
}

func (h *HTTPHandler) post(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in SendCommand
	if !decode(w, r, &in) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	m, created, err := h.svc.Send(r.Context(), c, r.PathValue("id"), in.ClientMessageID, in.Body, in.AttachmentIDs...)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeMessage(w, status, m)
}

func (h *HTTPHandler) edit(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	version, err := strconv.ParseInt(v, 10, 32)
	if !ok || err != nil {
		writeCode(w, http.StatusPreconditionRequired, "if_match_required")
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &in) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	m, err := h.svc.Edit(r.Context(), c, r.PathValue("id"), r.PathValue("message_id"), in.Body, int32(version))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeMessage(w, http.StatusOK, m)
}

func (h *HTTPHandler) revisions(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	revs, err := h.svc.Revisions(r.Context(), c, r.PathValue("id"), r.PathValue("message_id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if revs == nil {
		revs = []Revision{}
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, struct {
		Revisions []Revision `json:"revisions"`
	}{revs})
}

func (h *HTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in struct {
		Seq *int64 `json:"seq"`
	}
	if !decode(w, r, &in) || in.Seq == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.svc.MarkRead(r.Context(), c, r.PathValue("id"), *in.Seq); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeMessage(w http.ResponseWriter, status int, m Message) {
	noStore(w)
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(int(m.VersionID))))
	httpx.JSON(w, status, m)
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

// decode reads exactly one JSON object of known fields, at most maxBody bytes, into v.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && dec.Decode(&struct{}{}) == io.EOF
}

// listCursorString is a list cursor as the client sees it: base64url of "<unix microseconds>.<id>".
func listCursorString(c ListCursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.At.UnixMicro(), 10) + "." + c.ID))
}

func parseListCursor(s string) (ListCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return ListCursor{}, ErrInvalidCursor
	}
	us, id, ok := strings.Cut(string(raw), ".")
	t, perr := strconv.ParseInt(us, 10, 64)
	if !ok || perr != nil || t < 0 || !validUUID(id) {
		return ListCursor{}, ErrInvalidCursor
	}
	return ListCursor{At: time.UnixMicro(t).UTC(), ID: id}, nil
}

// signedIn returns the caller, or answers 401 (or 500) and false.
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

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalid, "invalid_request", http.StatusBadRequest},
	{ErrInvalidCursor, "invalid_cursor", http.StatusBadRequest},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrConsentRequired, "consent_required", http.StatusForbidden},
	{ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range allErrorCodes {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
