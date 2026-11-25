package careplan

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Plans is what the HTTPHandler calls: Service.
type Plans interface {
	Write(ctx context.Context, c Staff, d Draft) (Plan, error)
	Change(ctx context.Context, c Staff, id string, d Draft) (Plan, error)
	Book(ctx context.Context, c Staff, id string, b Booking) (Plan, error)
	Cancel(ctx context.Context, c Staff, id string, version int) (Plan, error)
	Read(ctx context.Context, c Staff, id string) (Plan, error)
	ForPatient(ctx context.Context, c Staff, patientID string) ([]Plan, error)
	Mine(ctx context.Context, c Patient) ([]Plan, error)
}

// Caller is the signed-in session as the plans' routes see it: staff of a clinic (OrganizationID set), or a patient.
type Caller struct {
	Patient Patient
	Staff   Staff
}

// HTTPHandler serves the treatment plans' JSON routes (api/openapi.yaml, tag care-plans).
type HTTPHandler struct {
	svc    Plans
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the HTTPHandler; caller gives who is signed in, false for nobody.
func NewHTTPHandler(svc Plans, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the plans' routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /clinic/care-plans", h.staff(h.write))
	mux.HandleFunc("GET /clinic/patients/{id}/care-plans", h.staff(h.forPatient))
	mux.HandleFunc("GET /clinic/care-plans/{id}", h.staff(h.read))
	mux.HandleFunc("PUT /clinic/care-plans/{id}", h.staff(h.change))
	mux.HandleFunc("POST /clinic/care-plans/{id}/book", h.staff(h.book))
	mux.HandleFunc("POST /clinic/care-plans/{id}/cancel", h.staff(h.cancel))
	mux.HandleFunc("GET /me/care-plans", h.mine)
}

// planJSON is the CarePlan schema: Plan with its start as a date.
type planJSON struct {
	BookedAt       *time.Time `json:"bookedAt,omitempty"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	PatientID      string     `json:"patientId"`
	Status         Status     `json:"status"`
	Title          string     `json:"title"`
	QuoteID        string     `json:"quoteId,omitempty"`
	StartsOn       string     `json:"startsOn,omitempty"`
	Steps          []Step     `json:"steps"`
	VersionID      int        `json:"versionId"`
}

func toJSON(p Plan) planJSON {
	out := planJSON{
		BookedAt: p.BookedAt, RevokedAt: p.RevokedAt, CreatedAt: p.CreatedAt, ID: p.ID, OrganizationID: p.OrganizationID,
		PatientID: p.PatientID, Status: p.Status, Title: p.Title, QuoteID: p.QuoteID, Steps: p.Steps, VersionID: p.VersionID,
	}
	if p.StartsOn != nil {
		out.StartsOn = p.StartsOn.Format(time.DateOnly)
	}
	if out.Steps == nil {
		out.Steps = []Step{}
	}
	return out
}

func listJSON(list []Plan) []planJSON {
	out := make([]planJSON, len(list))
	for i, p := range list {
		out[i] = toJSON(p)
	}
	return out
}

// saveJSON and stepJSON are SaveCarePlanRequest; bookJSON is BookCarePlanRequest.
type saveJSON struct {
	PatientID string     `json:"patientId"`
	Title     string     `json:"title"`
	QuoteID   string     `json:"quoteId"`
	Steps     []stepJSON `json:"steps"`
}

type stepJSON struct {
	ServiceID string `json:"serviceId"`
	VisitMode string `json:"visitMode"`
	DayOffset int    `json:"dayOffset"`
}

type bookJSON struct {
	StartsOn string   `json:"startsOn"`
	SlotIDs  []string `json:"slotIds"`
}

var uuidText = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// draft reads a SaveCarePlanRequest: 400 invalid_request for one that is malformed or names an id that is no UUID.
func draft(w http.ResponseWriter, r *http.Request, isNew bool) (Draft, bool) {
	var in saveJSON
	if !decodeJSON(w, r, &in) {
		return Draft{}, false
	}
	ok := (!isNew || uuidText.MatchString(in.PatientID)) && (in.QuoteID == "" || uuidText.MatchString(in.QuoteID))
	d := Draft{Title: in.Title, QuoteID: in.QuoteID}
	if isNew { // ignored on a change
		d.PatientID = in.PatientID
	}
	for _, s := range in.Steps {
		ok = ok && uuidText.MatchString(s.ServiceID)
		d.Steps = append(d.Steps, Step{ServiceID: s.ServiceID, VisitMode: s.VisitMode, DayOffset: s.DayOffset})
	}
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
	}
	return d, ok
}

// write answers POST /clinic/care-plans.
func (h *HTTPHandler) write(w http.ResponseWriter, r *http.Request, c Staff) {
	var d Draft
	if c.Clinical() { // the service refuses anyone else, audited, before the request is read
		var ok bool
		if d, ok = draft(w, r, true); !ok {
			return
		}
	}
	p, err := h.svc.Write(r.Context(), c, d)
	h.answer(w, r, http.StatusCreated, p, err)
}

// change answers PUT /clinic/care-plans/{id}.
func (h *HTTPHandler) change(w http.ResponseWriter, r *http.Request, c Staff) {
	version, ok := h.version(w, r, c)
	if !ok {
		return
	}
	var d Draft
	if c.Clinical() {
		if d, ok = draft(w, r, false); !ok {
			return
		}
	}
	d.Version = version
	p, err := h.svc.Change(r.Context(), c, r.PathValue("id"), d)
	h.answer(w, r, http.StatusOK, p, err)
}

// book answers POST /clinic/care-plans/{id}/book.
func (h *HTTPHandler) book(w http.ResponseWriter, r *http.Request, c Staff) {
	version, ok := h.version(w, r, c)
	if !ok {
		return
	}
	b := Booking{Version: version}
	if c.Clinical() {
		var in bookJSON
		if !decodeJSON(w, r, &in) {
			return
		}
		day, err := time.Parse(time.DateOnly, in.StartsOn)
		valid := err == nil && len(in.SlotIDs) > 0
		for _, id := range in.SlotIDs {
			valid = valid && uuidText.MatchString(id)
		}
		if !valid {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		b.StartsOn, b.SlotIDs = day, in.SlotIDs
	}
	p, err := h.svc.Book(r.Context(), c, r.PathValue("id"), b)
	h.answer(w, r, http.StatusOK, p, err)
}

// cancel answers POST /clinic/care-plans/{id}/cancel.
func (h *HTTPHandler) cancel(w http.ResponseWriter, r *http.Request, c Staff) {
	version, ok := h.version(w, r, c)
	if !ok {
		return
	}
	p, err := h.svc.Cancel(r.Context(), c, r.PathValue("id"), version)
	h.answer(w, r, http.StatusOK, p, err)
}

// read answers GET /clinic/care-plans/{id}.
func (h *HTTPHandler) read(w http.ResponseWriter, r *http.Request, c Staff) {
	p, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(p))
}

// forPatient answers GET /clinic/patients/{id}/care-plans.
func (h *HTTPHandler) forPatient(w http.ResponseWriter, r *http.Request, c Staff) {
	list, err := h.svc.ForPatient(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

// mine answers GET /me/care-plans: a patient's own booked plans; staff and an account without a profile have none.
func (h *HTTPHandler) mine(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if c.Staff.OrganizationID != "" || c.Patient.PatientID == "" {
		writeCode(w, http.StatusForbidden, "profile_required")
		return
	}
	list, err := h.svc.Mine(r.Context(), c.Patient)
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

// version reads If-Match: 400 invalid_request without one, for a caller the service would not refuse anyway.
func (h *HTTPHandler) version(w http.ResponseWriter, r *http.Request, c Staff) (int, bool) {
	v, matched := fhir.IfMatch(r.Header.Get("If-Match"))
	n, err := strconv.Atoi(v)
	if (!matched || err != nil) && c.Clinical() {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return 0, false
	}
	return n, true
}

// answer writes p with its ETag, or err.
func (h *HTTPHandler) answer(w http.ResponseWriter, r *http.Request, status int, p Plan, err error) {
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("ETag", `W/"`+strconv.Itoa(p.VersionID)+`"`)
	httpx.JSON(w, status, toJSON(p))
}

// staff runs next for a signed-in staff account; a patient is answered not_found.
func (h *HTTPHandler) staff(next func(http.ResponseWriter, *http.Request, Staff)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := h.signedIn(w, r)
		switch {
		case !ok:
		case c.Staff.OrganizationID == "":
			writeCode(w, http.StatusNotFound, "not_found")
		default:
			next(w, r, c.Staff)
		}
	}
}

func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	}
	return c, ok && err == nil
}

// maxBody bounds a request body: 50 steps fit well within it.
const maxBody = 64 << 10

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
	{ErrInvalid, "invalid_care_plan", http.StatusBadRequest},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrNotDraft, "care_plan_not_draft", http.StatusConflict},
	{ErrEnded, "care_plan_ended", http.StatusConflict},
	{ErrSlotUnavailable, "slot_unavailable", http.StatusConflict},
	{ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
}

func (h *HTTPHandler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	for _, e := range errorCodes {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return true
		}
	}
	httpx.InternalError(w, r, h.log, err)
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
