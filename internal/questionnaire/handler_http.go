package questionnaire

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

// maxBody bounds a JSON request body (SPEC.md §2 "Limits"): a questionnaire of 200 items at their limits
// fits.
const maxBody = 256 << 10

// canonicalBase is the canonical url of a series without its id (SPEC.md §2 "Canonical url"), on the
// placeholder base until the real domain is known.
const canonicalBase = "https://salus.invalid/fhir/Questionnaire/"

// CanonicalURL is the canonical url of the series seriesID: Questionnaire.url, and with "|<version>"
// QuestionnaireResponse.questionnaire.
func CanonicalURL(seriesID string) string { return canonicalBase + seriesID }

// Library edits a clinic's questionnaires and its roles' defaults for a caller (Service).
type Library interface {
	Create(ctx context.Context, c Caller, in Content) (Questionnaire, error)
	UpdateDraft(ctx context.Context, c Caller, id string, in Content, version int32) (Questionnaire, error)
	DeleteDraft(ctx context.Context, c Caller, id string) error
	Publish(ctx context.Context, c Caller, id string, version int32) (Questionnaire, error)
	Retire(ctx context.Context, c Caller, id string) (Questionnaire, error)
	Revise(ctx context.Context, c Caller, id string) (Questionnaire, error)
	Default(ctx context.Context, c Caller, roleID string) (Default, error)
	SetDefault(ctx context.Context, c Caller, roleID, questionnaireID string) (Default, error)
	ClearDefault(ctx context.Context, c Caller, roleID string) error
}

// HTTPHandler serves the questionnaire writes and the roles' defaults of the JSON API (api/openapi.yaml).
// Errors are {"error": "<code>"} with a fixed code per case, plus the offending linkIds on 422; internal
// error text goes to the log only, and no title, text or option ever does.
type HTTPHandler struct {
	svc Library
	// caller is who is signed in for the request, with their clinic role, and false for nobody (internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Library, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the JSON questionnaire routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /questionnaires", h.create)
	mux.HandleFunc("PUT /questionnaires/{id}", h.update)
	mux.HandleFunc("DELETE /questionnaires/{id}", h.delete)
	mux.HandleFunc("POST /questionnaires/{id}/publish", h.publish)
	mux.HandleFunc("POST /questionnaires/{id}/retire", h.retire)
	mux.HandleFunc("POST /questionnaires/{id}/revise", h.revise)
	mux.HandleFunc("GET /practitioner-roles/{id}/questionnaire", h.getDefault)
	mux.HandleFunc("PUT /practitioner-roles/{id}/questionnaire", h.setDefault)
	mux.HandleFunc("DELETE /practitioner-roles/{id}/questionnaire", h.clearDefault)
}

// questionnaireJSON is the Questionnaire schema; the items are the model's own JSON.
type questionnaireJSON struct {
	ID           string            `json:"id"`
	URL          string            `json:"url"`
	Status       Status            `json:"status"`
	Title        string            `json:"title"`
	Description  string            `json:"description,omitempty"`
	Language     string            `json:"language"`
	LastUpdated  string            `json:"lastUpdated"`
	Items        []Item            `json:"item"`
	Translations []translationJSON `json:"translations"`
	Version      int32             `json:"version"`
	VersionID    int32             `json:"versionId"`
}

// translationJSON is the QuestionnaireTranslation schema; the items' texts are the model's own JSON.
type translationJSON struct {
	Items       map[string]ItemTranslation `json:"items"`
	Language    string                     `json:"language"`
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
}

// defaultJSON is the QuestionnaireDefault schema: the series' canonical url, absent while the role has none.
type defaultJSON struct {
	PractitionerRoleID string `json:"practitionerRoleId"`
	URL                string `json:"url,omitempty"`
	VersionID          int32  `json:"versionId"`
}

func (h *HTTPHandler) create(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	in, ok := decodeContent(w, r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	q, err := h.svc.Create(r.Context(), c, in)
	h.write(w, r, http.StatusCreated, q, err)
}

// update sets a draft's content; If-Match is the version it was read at.
func (h *HTTPHandler) update(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, okVersion := ifMatch(r)
	in, okBody := decodeContent(w, r)
	if !okVersion || !okBody {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	q, err := h.svc.UpdateDraft(r.Context(), c, r.PathValue("id"), in, version)
	h.write(w, r, http.StatusOK, q, err)
}

func (h *HTTPHandler) delete(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	h.writeEmpty(w, r, h.svc.DeleteDraft(r.Context(), c, r.PathValue("id")))
}

// publish publishes a draft as it was read: If-Match is that version.
func (h *HTTPHandler) publish(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	q, err := h.svc.Publish(r.Context(), c, r.PathValue("id"), version)
	h.write(w, r, http.StatusOK, q, err)
}

func (h *HTTPHandler) retire(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, err := h.svc.Retire(r.Context(), c, r.PathValue("id"))
	h.write(w, r, http.StatusOK, q, err)
}

// revise creates the next version of an active one as a draft.
func (h *HTTPHandler) revise(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, err := h.svc.Revise(r.Context(), c, r.PathValue("id"))
	h.write(w, r, http.StatusCreated, q, err)
}

func (h *HTTPHandler) getDefault(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	d, err := h.svc.Default(r.Context(), c, r.PathValue("id"))
	h.writeDefault(w, r, d, err)
}

// setDefault: the body is {"questionnaireId": "<any version of the series>"}.
func (h *HTTPHandler) setDefault(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		QuestionnaireID *string `json:"questionnaireId"`
	}
	if !decode(w, r, &req) || req.QuestionnaireID == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	d, err := h.svc.SetDefault(r.Context(), c, r.PathValue("id"), *req.QuestionnaireID)
	h.writeDefault(w, r, d, err)
}

func (h *HTTPHandler) clearDefault(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	h.writeEmpty(w, r, h.svc.ClearDefault(r.Context(), c, r.PathValue("id")))
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

func (h *HTTPHandler) write(w http.ResponseWriter, r *http.Request, status int, q Questionnaire, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	items := q.Items
	if items == nil {
		items = []Item{}
	}
	translations := make([]translationJSON, 0, len(q.Translations))
	for _, t := range q.Translations {
		if t.Items == nil {
			t.Items = map[string]ItemTranslation{}
		}
		translations = append(translations, translationJSON(t))
	}
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(q.VersionID), 10)))
	httpx.JSON(w, status, questionnaireJSON{
		ID: q.ID, URL: CanonicalURL(q.SeriesID), Version: q.Version, Status: q.Status, Title: q.Title, Language: q.Language,
		Description: q.Description, Items: items, Translations: translations, VersionID: q.VersionID,
		LastUpdated: q.LastUpdated.UTC().Format(time.RFC3339),
	})
}

func (h *HTTPHandler) writeDefault(w http.ResponseWriter, r *http.Request, d Default, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := defaultJSON{PractitionerRoleID: d.PractitionerRoleID, VersionID: d.VersionID}
	if d.SeriesID != "" {
		out.URL = CanonicalURL(d.SeriesID)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *HTTPHandler) writeEmpty(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeContent reads one SaveQuestionnaireRequest: a title and the items, the language and the translations
// when given, every field known, and false when it cannot. Translations absent are nil (the draft's kept).
func decodeContent(w http.ResponseWriter, r *http.Request) (Content, bool) {
	var req struct {
		Title        *string            `json:"title"`
		Items        *[]Item            `json:"item"`
		Translations *[]translationJSON `json:"translations"`
		Description  string             `json:"description"`
		Language     string             `json:"language"`
	}
	if !decode(w, r, &req) || req.Title == nil || req.Items == nil {
		return Content{}, false
	}
	in := Content{Title: *req.Title, Description: req.Description, Items: *req.Items, Language: req.Language}
	if req.Translations != nil {
		in.Translations = make([]Translation, 0, len(*req.Translations))
		for _, t := range *req.Translations {
			in.Translations = append(in.Translations, Translation(t))
		}
	}
	return in, true
}

// decode reads exactly one JSON object of known fields, at most maxBody bytes, into v.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && dec.Decode(&struct{}{}) == io.EOF
}

// ifMatch is the version the If-Match header names (W/"<n>" or "<n>"), and false for none.
func ifMatch(r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 32)
	return int32(n), err == nil
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrConflict, "version_conflict", http.StatusPreconditionFailed},
	{ErrNotDraft, "not_draft", http.StatusConflict},
	{ErrNotActive, "not_active", http.StatusConflict},
	{ErrDraftExists, "draft_exists", http.StatusConflict},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if incomplete, ok := errors.AsType[*TranslationIncompleteError](err); ok {
		ids := incomplete.LinkIDs
		if ids == nil {
			ids = []string{}
		}
		httpx.JSON(w, http.StatusUnprocessableEntity, struct {
			Error    string   `json:"error"`
			Language string   `json:"language"`
			LinkIDs  []string `json:"linkIds"`
		}{"translation_incomplete", incomplete.Language, ids})
		return
	}
	if invalid, ok := errors.AsType[*InvalidError](err); ok {
		ids := invalid.LinkIDs
		if ids == nil {
			ids = []string{}
		}
		httpx.JSON(w, http.StatusUnprocessableEntity, struct {
			Error   string   `json:"error"`
			LinkIDs []string `json:"linkIds"`
		}{"invalid_questionnaire", ids})
		return
	}
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
