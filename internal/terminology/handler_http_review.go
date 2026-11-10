package terminology

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// The review panel's request bounds in the handler.
const (
	defaultQueueLimit = 25
	maxReviewBody     = 64 << 10 // bytes
)

// ReviewService is the review panel's service (Reviews).
type ReviewService interface {
	Queue(ctx context.Context, c Caller, q QueueQuery) (QueuePage, error)
	Progress(ctx context.Context, c Caller) ([]LanguageProgress, error)
	EditTranslation(ctx context.Context, c Caller, termID, language string, edit LabelsEdit, version int32) (Translation, error)
	ReviewTranslation(ctx context.Context, c Caller, termID, language string, d Decision, version int32) (Translation, error)
	WriteExplanation(ctx context.Context, c Caller, termID, language, text string, version int32) (Explanation, error)
	ReviewExplanation(ctx context.Context, c Caller, termID, language string, d Decision, version int32) (Explanation, error)
}

// ReviewHandler serves the review panel's JSON API (api/openapi.yaml, listTermReview, getTermReviewProgress and
// the edits). Errors are {"error": "<code>"} with a fixed code per case; no label, explanation or note reaches a
// log line.
type ReviewHandler struct {
	svc    ReviewService
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewReviewHandler returns the handler on svc; caller is who is signed in (internal/api), false for nobody.
func NewReviewHandler(svc ReviewService, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *ReviewHandler {
	return &ReviewHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the review panel's routes on mux.
func (h *ReviewHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /terminology/review", h.queue)
	mux.HandleFunc("GET /terminology/review/progress", h.progress)
	mux.HandleFunc("PUT /terminology/terms/{id}/translations/{language}", h.editTranslation)
	mux.HandleFunc("POST /terminology/terms/{id}/translations/{language}/review", h.reviewTranslation)
	mux.HandleFunc("PUT /terminology/terms/{id}/explanations/{language}", h.writeExplanation)
	mux.HandleFunc("POST /terminology/terms/{id}/explanations/{language}/review", h.reviewExplanation)
}

// reviewJSON is the Review schema.
type reviewJSON struct {
	Status     string `json:"status"`
	ReviewedBy string `json:"reviewedBy,omitempty"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
	Note       string `json:"note,omitempty"`
}

// translationJSON is the TermTranslation schema.
type translationJSON struct {
	TermID      string     `json:"termId"`
	Language    string     `json:"language"`
	Review      reviewJSON `json:"review"`
	LastUpdated string     `json:"lastUpdated"`
	VersionID   int32      `json:"versionId"`
}

// explanationJSON is the TermExplanation schema.
type explanationJSON struct {
	TermID      string     `json:"termId"`
	Language    string     `json:"language"`
	Text        string     `json:"text"`
	Review      reviewJSON `json:"review"`
	LastUpdated string     `json:"lastUpdated"`
	VersionID   int32      `json:"versionId"`
}

// entryJSON is the TermReviewEntry schema.
type entryJSON struct {
	Translation *translationJSON `json:"translation,omitempty"`
	Explanation *explanationJSON `json:"explanation,omitempty"`
	Status      string           `json:"status"`
	Term        termJSON         `json:"term"`
}

// progressJSON is an item of the TermReviewProgress schema.
type progressJSON struct {
	Language     string `json:"language"`
	Terms        int32  `json:"terms"`
	Approved     int32  `json:"approved"`
	NeedsChanges int32  `json:"needsChanges"`
}

func (h *ReviewHandler) queue(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v := r.URL.Query()
	q := QueueQuery{Language: v.Get("language"), Status: v.Get("status"), After: v.Get("after"), Limit: defaultQueueLimit}
	if v.Has("limit") {
		n, err := strconv.ParseInt(v.Get("limit"), 10, 32)
		if err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		q.Limit = int32(n)
	}
	page, err := h.svc.Queue(r.Context(), c, q)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	entries := make([]entryJSON, len(page.Entries))
	for i, e := range page.Entries {
		entries[i] = entryJSON{Status: e.Status, Term: termOut(Match{Term: e.Term, Labels: e.Labels})}
		if e.Translation != nil {
			t := translationOut(*e.Translation)
			entries[i].Translation = &t
		}
		if e.Explanation != nil {
			x := explanationOut(*e.Explanation)
			entries[i].Explanation = &x
		}
	}
	httpx.JSON(w, http.StatusOK, struct {
		Next    string      `json:"next,omitempty"`
		Entries []entryJSON `json:"entries"`
	}{page.Next, entries})
}

func (h *ReviewHandler) progress(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.Progress(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]progressJSON, len(rows))
	for i, p := range rows {
		out[i] = progressJSON(p)
	}
	httpx.JSON(w, http.StatusOK, map[string][]progressJSON{"languages": out})
}

func (h *ReviewHandler) editTranslation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Preferred *string  `json:"preferred"`
		Synonyms  []string `json:"synonyms"`
	}
	c, version, ok := h.write(w, r, &req, func() bool { return req.Preferred != nil && req.Synonyms != nil })
	if !ok {
		return
	}
	t, err := h.svc.EditTranslation(r.Context(), c, r.PathValue("id"), r.PathValue("language"), LabelsEdit{Preferred: *req.Preferred, Synonyms: req.Synonyms}, version)
	h.writeTranslation(w, r, t, err)
}

func (h *ReviewHandler) reviewTranslation(w http.ResponseWriter, r *http.Request) {
	var req decisionJSON
	c, version, ok := h.write(w, r, &req, func() bool { return req.Status != nil })
	if !ok {
		return
	}
	t, err := h.svc.ReviewTranslation(r.Context(), c, r.PathValue("id"), r.PathValue("language"), req.decision(), version)
	h.writeTranslation(w, r, t, err)
}

func (h *ReviewHandler) writeExplanation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text *string `json:"text"`
	}
	c, version, ok := h.write(w, r, &req, func() bool { return req.Text != nil })
	if !ok {
		return
	}
	e, err := h.svc.WriteExplanation(r.Context(), c, r.PathValue("id"), r.PathValue("language"), *req.Text, version)
	h.writeExplanationOut(w, r, e, err)
}

func (h *ReviewHandler) reviewExplanation(w http.ResponseWriter, r *http.Request) {
	var req decisionJSON
	c, version, ok := h.write(w, r, &req, func() bool { return req.Status != nil })
	if !ok {
		return
	}
	e, err := h.svc.ReviewExplanation(r.Context(), c, r.PathValue("id"), r.PathValue("language"), req.decision(), version)
	h.writeExplanationOut(w, r, e, err)
}

// decisionJSON is the ReviewDecision schema.
type decisionJSON struct {
	Status *string `json:"status"`
	Note   string  `json:"note"`
}

func (d decisionJSON) decision() Decision { return Decision{Status: *d.Status, Note: d.Note} }

// write reads a write's caller, If-Match and body into req, which complete reports as having its required
// fields; false when it has answered (401, 400 or 500).
func (h *ReviewHandler) write(w http.ResponseWriter, r *http.Request, req any, complete func() bool) (Caller, int32, bool) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return Caller{}, 0, false
	}
	version, okVersion := reviewIfMatch(r)
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxReviewBody))
	dec.DisallowUnknownFields()
	okBody := dec.Decode(req) == nil && dec.Decode(&struct{}{}) == io.EOF && complete()
	if !okVersion || !okBody {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return Caller{}, 0, false
	}
	return c, version, true
}

// reviewIfMatch is the version the If-Match header names (W/"<n>" or "<n>", 0 for an entry not written yet), and
// false for none or another form.
func reviewIfMatch(r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 32)
	return int32(n), err == nil && n >= 0
}

func (h *ReviewHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
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

func (h *ReviewHandler) writeTranslation(w http.ResponseWriter, r *http.Request, t Translation, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, translationOut(t))
}

func (h *ReviewHandler) writeExplanationOut(w http.ResponseWriter, r *http.Request, e Explanation, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, explanationOut(e))
}

// reviewErrors maps each domain error to its status and fixed code (the Error schema).
var reviewErrors = []struct {
	err    error
	code   string
	status int
}{
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrInvalidReview, "invalid_request", http.StatusBadRequest},
	{ErrTermNotFound, "not_found", http.StatusNotFound},
	{ErrNoTranslation, "not_found", http.StatusNotFound},
	{ErrNoExplanation, "not_found", http.StatusNotFound},
	{ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
	{ErrNoteRequired, "note_required", http.StatusUnprocessableEntity},
}

func (h *ReviewHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range reviewErrors {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func reviewOut(rv Review) reviewJSON {
	out := reviewJSON{Status: rv.Status, ReviewedBy: rv.ReviewedBy, Note: rv.Note}
	if !rv.ReviewedAt.IsZero() {
		out.ReviewedAt = rv.ReviewedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func translationOut(t Translation) translationJSON {
	return translationJSON{
		TermID: t.TermID, Language: t.Language, Review: reviewOut(t.Review), VersionID: t.VersionID,
		LastUpdated: t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func explanationOut(e Explanation) explanationJSON {
	return explanationJSON{
		TermID: e.TermID, Language: e.Language, Text: e.Text, Review: reviewOut(e.Review), VersionID: e.VersionID,
		LastUpdated: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
