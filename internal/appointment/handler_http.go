package appointment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body; a booking or a status fits many times over.
const maxBody = 1 << 10

// Appointments is what the handler calls (Service).
type Appointments interface {
	Book(ctx context.Context, c Caller, slotID string) (Appointment, error)
	Cancel(ctx context.Context, c Caller, id string) (Appointment, error)
	SetStatus(ctx context.Context, c Caller, id string, status Status) (Appointment, error)
	Changes(ctx context.Context, c Caller, id string) ([]Change, error)
}

// HTTPHandler serves the appointment actions of the JSON API (api/openapi.yaml); reads go through FHIR.
// Errors are {"error": "<code>"} with a fixed code per case; internal error text goes to the log only.
type HTTPHandler struct {
	svc Appointments
	// caller is who is signed in for the request, and false for nobody (from the session, the patient's
	// profile and the staff account's membership, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Appointments, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the appointment routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /appointments", h.book)
	mux.HandleFunc("POST /appointments/{id}/cancel", h.cancel)
	mux.HandleFunc("POST /appointments/{id}/status", h.setStatus)
	mux.HandleFunc("GET /appointments/{id}/changes", h.changes)
}

// appointmentJSON is the Appointment schema of api/openapi.yaml: no patient, no canceller.
type appointmentJSON struct {
	Start              time.Time  `json:"start"`
	End                time.Time  `json:"end"`
	CancelledAt        *time.Time `json:"cancelledAt,omitempty"`
	ConfirmedAt        *time.Time `json:"confirmedAt,omitempty"`
	ID                 string     `json:"id"`
	OrganizationID     string     `json:"organizationId"`
	PractitionerRoleID string     `json:"practitionerRoleId"`
	SlotID             string     `json:"slotId"`
	Status             Status     `json:"status"`
	Reason             Reason     `json:"cancellationReason,omitempty"`
	ConfirmedVia       Via        `json:"confirmedVia,omitempty"`
	BookedVia          BookedVia  `json:"bookedVia"`
	VersionID          int32      `json:"versionId"`
	RescheduleCount    int32      `json:"rescheduleCount"`
}

func toJSON(a Appointment) appointmentJSON {
	out := appointmentJSON{
		Start: a.Start.UTC(), End: a.End.UTC(), ID: a.ID, OrganizationID: a.OrganizationID, PractitionerRoleID: a.PractitionerRoleID,
		SlotID: a.SlotID, Status: a.Status, Reason: a.Reason, VersionID: a.VersionID,
		ConfirmedVia: a.ConfirmedVia, BookedVia: bookedVia(a.BookedVia), RescheduleCount: a.RescheduleCount,
	}
	if !a.CancelledAt.IsZero() {
		at := a.CancelledAt.UTC()
		out.CancelledAt = &at
	}
	if !a.ConfirmedAt.IsZero() {
		at := a.ConfirmedAt.UTC()
		out.ConfirmedAt = &at
	}
	return out
}

// changeJSON is the AppointmentChange schema of api/openapi.yaml: the period and status fields of the kind, never
// the account, agent or slots.
type changeJSON struct {
	At         time.Time  `json:"at"`
	FromStart  *time.Time `json:"from_start,omitempty"`
	FromEnd    *time.Time `json:"from_end,omitempty"`
	ToStart    *time.Time `json:"to_start,omitempty"`
	ToEnd      *time.Time `json:"to_end,omitempty"`
	Kind       ChangeKind `json:"kind"`
	Via        Via        `json:"via"`
	FromStatus Status     `json:"from_status,omitempty"`
	ToStatus   Status     `json:"to_status,omitempty"`
}

// changeListJSON is the AppointmentChangeList schema.
type changeListJSON struct {
	Changes []changeJSON `json:"changes"`
}

func toChangeJSON(c Change) changeJSON {
	out := changeJSON{At: c.At.UTC(), Kind: c.Kind, Via: c.Via, FromStatus: c.FromStatus, ToStatus: c.ToStatus}
	for _, t := range []struct {
		dst **time.Time
		v   time.Time
	}{{&out.FromStart, c.FromStart}, {&out.FromEnd, c.FromEnd}, {&out.ToStart, c.ToStart}, {&out.ToEnd, c.ToEnd}} {
		if !t.v.IsZero() {
			u := t.v.UTC()
			*t.dst = &u
		}
	}
	return out
}

func (h *HTTPHandler) changes(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.Changes(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := changeListJSON{Changes: make([]changeJSON, 0, len(all))}
	for _, ch := range all {
		out.Changes = append(out.Changes, toChangeJSON(ch))
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *HTTPHandler) book(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		SlotID string `json:"slotId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.SlotID == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.Book(r.Context(), c, req.SlotID)
	h.answer(w, r, http.StatusCreated, out, err)
}

func (h *HTTPHandler) cancel(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Cancel(r.Context(), c, r.PathValue("id"))
	h.answer(w, r, http.StatusOK, out, err)
}

func (h *HTTPHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		Status Status `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.SetStatus(r.Context(), c, r.PathValue("id"), req.Status)
	h.answer(w, r, http.StatusOK, out, err)
}

// answer sends out with status, never to be cached (whom a patient consults is health data), or the error.
func (h *HTTPHandler) answer(w http.ResponseWriter, r *http.Request, status int, out Appointment, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, toJSON(out))
}

// signedIn returns the caller, or answers 401 for nobody (500 when the caller cannot be resolved).
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

// decodeJSON reads one JSON object with known fields only, or answers 400 invalid_request.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrProfileRequired, "profile_required", http.StatusConflict},
	{ErrConsentRequired, "consent_required", http.StatusConflict},
	{ErrSlotUnavailable, "slot_unavailable", http.StatusConflict},
	{ErrCancellationWindowClosed, "cancellation_window_closed", http.StatusConflict},
	{ErrClosed, "appointment_closed", http.StatusConflict},
	{ErrNotStarted, "appointment_not_started", http.StatusConflict},
	{ErrInvalidStatus, "invalid_status", http.StatusBadRequest},
	{ErrRescheduleWindowClosed, "reschedule_window_closed", http.StatusConflict},
	{ErrRescheduleLimit, "reschedule_limit", http.StatusConflict},
	{ErrNotReschedulable, "not_reschedulable", http.StatusConflict},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
