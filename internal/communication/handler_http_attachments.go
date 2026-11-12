package communication

import (
	"net/http"
	"slices"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// fileErrorCodes are the codes of the files' refusals (SAL-30), as binary's own routes answer them.
var fileErrorCodes = []struct {
	err    error
	code   string
	status int
}{
	{binary.ErrNotFound, "not_found", http.StatusNotFound},
	{binary.ErrLengthRequired, "length_required", http.StatusLengthRequired},
	{binary.ErrShortBody, "invalid_request", http.StatusBadRequest},
	{binary.ErrTooLarge, "too_large", http.StatusRequestEntityTooLarge},
	{binary.ErrUnsupportedType, "unsupported_type", http.StatusUnsupportedMediaType},
	{binary.ErrScanning, "scanning", http.StatusConflict},
}

// allErrorCodes are the chat's codes, then the files'.
var allErrorCodes = slices.Concat(errorCodes, fileErrorCodes)

// upload reads the raw body, exactly Content-Length bytes (none given is -1); the file name is the
// Content-Disposition's, never anything from the URL (as binary's upload).
func (h *HTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	binary.Deadlines(w, h.clock.Now().Add(binary.UploadDeadline))
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Upload(r.Context(), c, r.PathValue("id"), Upload{
		Body: r.Body, Filename: binary.TitleOf(r.Header.Get("Content-Disposition")), Size: r.ContentLength,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *HTTPHandler) content(w http.ResponseWriter, r *http.Request)   { h.file(w, r, false) }
func (h *HTTPHandler) thumbnail(w http.ResponseWriter, r *http.Request) { h.file(w, r, true) }

// file sends a message's file, or its thumbnail, or the one range asked for, as binary's content route does; HEAD
// is refused (405): it would decrypt and audit the file for no byte.
func (h *HTTPHandler) file(w http.ResponseWriter, r *http.Request, thumbnail bool) {
	if r.Method == http.MethodHead {
		w.Header().Set("Allow", http.MethodGet)
		writeCode(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	binary.Deadlines(w, h.clock.Now().Add(binary.ContentDeadline))
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	f, err := h.svc.Attachment(r.Context(), c, r.PathValue("id"), r.PathValue("attachment_id"), thumbnail)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	binary.ServeContent(w, r, h.svc, f, h.log)
}

// searchJSON is the ChatSearchResults schema.
type searchJSON struct {
	Results []SearchResult `json:"results"`
}

func (h *HTTPHandler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	rs, err := h.svc.Search(r.Context(), c, r.PathValue("id"), r.URL.Query().Get("q"))
	h.writeResults(w, r, rs, err)
}

func (h *HTTPHandler) searchAll(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	rs, err := h.svc.SearchAll(r.Context(), c, r.URL.Query().Get("q"))
	h.writeResults(w, r, rs, err)
}

func (h *HTTPHandler) writeResults(w http.ResponseWriter, r *http.Request, rs []SearchResult, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if rs == nil {
		rs = []SearchResult{}
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, searchJSON{Results: rs})
}
