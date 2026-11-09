package composition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body: a template of 50 fields with ten languages at their limits fits.
const maxBody = 1 << 20

// Documents is what the JSON handler calls (Service).
type Documents interface {
	Templates(ctx context.Context, c appointment.Caller) ([]Template, error)
	CreateTemplate(ctx context.Context, c appointment.Caller, in TemplateContent) (Template, error)
	UpdateTemplate(ctx context.Context, c appointment.Caller, id string, in TemplateContent, version int32) (Template, error)
	Generate(ctx context.Context, c appointment.Caller, encounterID string, in Generate) (Draft, error)
	Drafts(ctx context.Context, c appointment.Caller, encounterID string) ([]Draft, error)
	Sign(ctx context.Context, c appointment.Caller, id string, values []Value, version int32) (Draft, error)
	PDF(ctx context.Context, c appointment.Caller, id string) (io.ReadCloser, int64, error)
}

// HTTPHandler serves the templates, the drafting, the signing and the signed PDF of the JSON API
// (api/openapi.yaml). Errors are {"error": "<code>"}, with the field keys on a 422 about fields; internal error
// text goes to the log only, and no title, label or value ever does.
type HTTPHandler struct {
	svc Documents
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Documents, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /document-templates", h.templates)
	mux.HandleFunc("POST /document-templates", h.createTemplate)
	mux.HandleFunc("PUT /document-templates/{id}", h.updateTemplate)
	mux.HandleFunc("POST /encounters/{id}/drafts", h.generate)
	mux.HandleFunc("GET /encounters/{id}/drafts", h.drafts)
	mux.HandleFunc("POST /drafts/{id}/sign", h.sign)
	mux.HandleFunc("GET /document-references/{id}/content", h.content)
}

// templateJSON is the DocumentTemplate schema.
type templateJSON struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Language     string            `json:"language"`
	Status       TemplateStatus    `json:"status"`
	TypeCode     string            `json:"typeCode"`
	LastUpdated  string            `json:"lastUpdated"`
	Fields       []TemplateField   `json:"fields"`
	Translations []translationJSON `json:"translations"`
	VersionID    int32             `json:"versionId"`
}

// translationJSON is the DocumentTemplateTranslation schema.
type translationJSON struct {
	Fields   map[string]FieldTexts `json:"fields"`
	Language string                `json:"language"`
	Title    string                `json:"title"`
}

func toTemplateJSON(t Template) templateJSON {
	out := templateJSON{
		ID: t.ID, Title: t.Title, Language: t.Language, Status: t.Status, TypeCode: t.TypeCode, Fields: t.Fields,
		LastUpdated: t.LastUpdated.UTC().Format(time.RFC3339), VersionID: t.VersionID, Translations: []translationJSON{},
	}
	for _, tr := range t.Translations {
		out.Translations = append(out.Translations, translationJSON(tr))
	}
	return out
}

func (h *HTTPHandler) templates(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.Templates(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]templateJSON, len(all))
	for i, t := range all {
		out[i] = toTemplateJSON(t)
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, map[string][]templateJSON{"templates": out})
}

func (h *HTTPHandler) createTemplate(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	in, ok := decodeTemplate(w, r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	t, err := h.svc.CreateTemplate(r.Context(), c, in)
	h.writeTemplate(w, r, http.StatusCreated, t, err)
}

// updateTemplate replaces a template's content; If-Match is the version it was read at.
func (h *HTTPHandler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, okVersion := ifMatch(r)
	in, okBody := decodeTemplate(w, r)
	if !okVersion || !okBody {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	t, err := h.svc.UpdateTemplate(r.Context(), c, r.PathValue("id"), in, version)
	h.writeTemplate(w, r, http.StatusOK, t, err)
}

func (h *HTTPHandler) writeTemplate(w http.ResponseWriter, r *http.Request, status int, t Template, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(t.VersionID), 10)))
	httpx.JSON(w, status, toTemplateJSON(t))
}

// decodeTemplate reads a SaveDocumentTemplateRequest.
func decodeTemplate(w http.ResponseWriter, r *http.Request) (TemplateContent, bool) {
	var req struct {
		Title        string            `json:"title"`
		Language     string            `json:"language"`
		Status       TemplateStatus    `json:"status"`
		TypeCode     string            `json:"typeCode"`
		Fields       []TemplateField   `json:"fields"`
		Translations []translationJSON `json:"translations"`
	}
	if !decode(w, r, &req) {
		return TemplateContent{}, false
	}
	in := TemplateContent{Title: req.Title, Language: req.Language, Status: req.Status, TypeCode: req.TypeCode, Fields: req.Fields}
	for _, tr := range req.Translations {
		in.Translations = append(in.Translations, Translation(tr))
	}
	return in, true
}

// generate drafts (or redrafts) a template for the encounter and answers with the draft.
func (h *HTTPHandler) generate(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		TemplateID string `json:"templateId"`
		Language   string `json:"language"`
	}
	if !decode(w, r, &req) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	d, err := h.svc.Generate(r.Context(), c, r.PathValue("id"), Generate(req))
	h.writeDraft(w, r, d, err)
}

func (h *HTTPHandler) drafts(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.Drafts(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]Event, len(all))
	for i, d := range all {
		out[i] = d.Event()
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, map[string][]Event{"drafts": out})
}

// sign signs a draft with the practitioner's values; If-Match is the version they read.
func (h *HTTPHandler) sign(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, okVersion := ifMatch(r)
	var req struct {
		Fields []Value `json:"fields"`
	}
	if okBody := decode(w, r, &req); !okVersion || !okBody {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	d, err := h.svc.Sign(r.Context(), c, r.PathValue("id"), req.Fields, version)
	h.writeDraft(w, r, d, err)
}

func (h *HTTPHandler) writeDraft(w http.ResponseWriter, r *http.Request, d Draft, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(d.VersionID), 10)))
	httpx.JSON(w, http.StatusOK, d.Event())
}

// content sends the signed document's PDF, as a download.
func (h *HTTPHandler) content(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	body, size, err := h.svc.PDF(r.Context(), c, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer func() { _ = body.Close() }()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.pdf"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body) // a failure here is the client gone or the object unreadable; the status is sent
}

// signedIn returns the caller, or answers 401 (or 500) and false.
func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return appointment.Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return appointment.Caller{}, false
	}
	return c, true
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrConsentRequired, "consent_required", http.StatusForbidden},
	{ErrTemplateNotFound, "template_not_found", http.StatusNotFound},
	{ErrLanguageUnavailable, "language_unavailable", http.StatusUnprocessableEntity},
	{ErrVersionMismatch, "version_mismatch", http.StatusPreconditionFailed},
	{ErrNotPreliminary, "not_preliminary", http.StatusConflict},
	{ErrDraftingUnavailable, "drafting_unavailable", http.StatusServiceUnavailable},
}

// keyedError is a 422 that names the fields concerned.
type keyedError struct {
	Error string   `json:"error"`
	Keys  []string `json:"keys"`
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if invalid, ok := errors.AsType[*InvalidTemplateError](err); ok {
		httpx.JSON(w, http.StatusUnprocessableEntity, keyedError{"invalid_template", nonNil(invalid.Keys)})
		return
	}
	if fields, ok := errors.AsType[*FieldsError](err); ok {
		code := "invalid_values"
		if errors.Is(fields.Err, ErrRequiredMissing) {
			code = "required_missing"
		}
		httpx.JSON(w, http.StatusUnprocessableEntity, keyedError{code, nonNil(fields.Keys)})
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

func nonNil(keys []string) []string {
	if keys == nil {
		return []string{}
	}
	return keys
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
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
