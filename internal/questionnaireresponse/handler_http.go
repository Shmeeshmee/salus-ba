package questionnaireresponse

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

// maxBody bounds a JSON request body (SPEC.md §2 "Limits").
const maxBody = 256 << 10

// Writer makes the requests and changes the answers for a caller (Service).
type Writer interface {
	Request(ctx context.Context, c appointment.Caller, appointmentID, questionnaireID string) (Response, error)
	SaveAnswers(ctx context.Context, c appointment.Caller, id string, answers []Answer, version int32) (Response, error)
	Submit(ctx context.Context, c appointment.Caller, id string, version int32) (Response, error)
	Withdraw(ctx context.Context, c appointment.Caller, id string) (Response, error)
}

// HTTPHandler serves the questionnaire requests and the patient's answers of the JSON API (api/openapi.yaml).
// Errors are {"error": "<code>"} with a fixed code per case, plus the offending linkIds on 422; internal
// error text goes to the log only, and no answer ever does.
type HTTPHandler struct {
	svc Writer
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Writer, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the JSON questionnaire response routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /appointments/{id}/questionnaires", h.request)
	mux.HandleFunc("PUT /questionnaire-responses/{id}/answers", h.saveAnswers)
	mux.HandleFunc("POST /questionnaire-responses/{id}/submit", h.submit)
	mux.HandleFunc("POST /questionnaire-responses/{id}/withdraw", h.withdraw)
}

// responseJSON is the QuestionnaireResponse schema of the JSON API.
type responseJSON struct {
	ID              string   `json:"id"`
	AppointmentID   string   `json:"appointmentId"`
	QuestionnaireID string   `json:"questionnaireId"`
	Status          Status   `json:"status"`
	Source          Source   `json:"source"`
	DueAt           string   `json:"dueAt"`
	Authored        string   `json:"authored,omitempty"`
	LastUpdated     string   `json:"lastUpdated"`
	Answers         []Answer `json:"answers"`
	VersionID       int32    `json:"versionId"`
}

// request: the body is {"questionnaireId": "<an active version>"}.
func (h *HTTPHandler) request(w http.ResponseWriter, r *http.Request) {
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
	out, err := h.svc.Request(r.Context(), c, r.PathValue("id"), *req.QuestionnaireID)
	h.write(w, r, http.StatusCreated, out, err)
}

// saveAnswers replaces the saved answers; If-Match is the version they were read at.
func (h *HTTPHandler) saveAnswers(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, okVersion := ifMatch(r)
	var req struct {
		Answers *[]Answer `json:"answers"`
	}
	if !okVersion || !decode(w, r, &req) || req.Answers == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.SaveAnswers(r.Context(), c, r.PathValue("id"), *req.Answers, version)
	h.write(w, r, http.StatusOK, out, err)
}

// submit submits the answers as they were read: If-Match is that version.
func (h *HTTPHandler) submit(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.Submit(r.Context(), c, r.PathValue("id"), version)
	h.write(w, r, http.StatusOK, out, err)
}

func (h *HTTPHandler) withdraw(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Withdraw(r.Context(), c, r.PathValue("id"))
	h.write(w, r, http.StatusOK, out, err)
}

// signedIn is the request's caller, or false after answering 401 or 500.
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

func (h *HTTPHandler) write(w http.ResponseWriter, r *http.Request, status int, out Response, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	answers := out.Answers
	if answers == nil {
		answers = []Answer{}
	}
	body := responseJSON{
		ID: out.ID, AppointmentID: out.AppointmentID, QuestionnaireID: out.QuestionnaireID, Status: out.Status, Source: out.Source,
		Answers: answers, DueAt: instant(out.DueAt), Authored: instant(out.AuthoredAt), LastUpdated: instant(out.LastUpdated),
		VersionID: out.VersionID,
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(out.VersionID), 10)))
	httpx.JSON(w, status, body)
}

// instant is t in UTC as RFC 3339, "" for the zero time.
func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
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
	{ErrAlreadyRequested, "already_requested", http.StatusConflict},
	{ErrNotActive, "not_active", http.StatusConflict},
	{ErrAppointmentStarted, "appointment_started", http.StatusConflict},
	{ErrAppointmentClosed, "appointment_closed", http.StatusConflict},
	{ErrConsentRequired, "consent_required", http.StatusConflict},
	{ErrSubmitted, "submitted", http.StatusConflict},
	{ErrWithdrawn, "withdrawn", http.StatusConflict},
	{ErrPastDue, "past_due", http.StatusConflict},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if invalid, ok := errors.AsType[*AnswersError](err); ok {
		ids := invalid.LinkIDs
		if ids == nil {
			ids = []string{}
		}
		httpx.JSON(w, http.StatusUnprocessableEntity, struct {
			Error   string   `json:"error"`
			LinkIDs []string `json:"linkIds"`
		}{"answers_invalid", ids})
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
