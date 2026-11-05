package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body: a week of short windows fits many times over.
const maxBody = 64 << 10

// Availabilities reads and edits a role's availability for a caller (Service).
type Availabilities interface {
	Availability(ctx context.Context, c Caller, roleID string) (Schedule, error)
	SaveAvailability(ctx context.Context, c Caller, roleID string, in Availability, version int32) (Schedule, error)
}

// HTTPHandler serves the availability actions of the JSON API (api/openapi.yaml). Errors are
// {"error": "<code>"} with a fixed code per case; internal error text goes to the log only.
type HTTPHandler struct {
	svc Availabilities
	// caller is who is signed in for the request, with their clinic role, and false for nobody (from the
	// session and the account, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Availabilities, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the JSON availability routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /practitioner-roles/{id}/availability", h.availability)
	mux.HandleFunc("PUT /practitioner-roles/{id}/availability", h.saveAvailability)
}

// ruleBody is a Rule in the AvailabilityRule schema.
type ruleBody struct {
	Day   *int    `json:"day"`
	Start *string `json:"start"`
	End   *string `json:"end"`
}

// availabilityJSON is the Availability schema.
type availabilityJSON struct {
	ID                 string     `json:"id,omitempty"`
	PractitionerRoleID string     `json:"practitionerRoleId"`
	Rules              []ruleJSON `json:"availability"`
	SlotMinutes        int32      `json:"slotMinutes,omitempty"`
	VersionID          int32      `json:"versionId"`
}

func (h *HTTPHandler) availability(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Availability(r.Context(), c, r.PathValue("id"))
	h.write(w, r, s, err)
}

// saveAvailability sets the role's availability: the body is {"slotMinutes": n, "availability": [rules]},
// If-Match the version the availability was read at (W/"0" before the first edit).
func (h *HTTPHandler) saveAvailability(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, okVersion := ifMatch(r)
	in, okBody := decodeAvailability(w, r)
	if !okVersion || !okBody {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s, err := h.svc.SaveAvailability(r.Context(), c, r.PathValue("id"), in, version)
	h.write(w, r, s, err)
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

func (h *HTTPHandler) write(w http.ResponseWriter, r *http.Request, s Schedule, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := availabilityJSON{ID: s.ID, PractitionerRoleID: s.PractitionerRoleID, Rules: make([]ruleJSON, 0, len(s.Rules)),
		SlotMinutes: s.SlotMinutes, VersionID: s.VersionID}
	for _, rule := range s.Rules {
		out.Rules = append(out.Rules, ruleJSON(rule))
	}
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(s.VersionID), 10)))
	httpx.JSON(w, http.StatusOK, out)
}

// decodeAvailability reads one SaveAvailabilityRequest with known fields only, every field given, and false
// when it cannot.
func decodeAvailability(w http.ResponseWriter, r *http.Request) (Availability, bool) {
	var req struct {
		SlotMinutes *int32      `json:"slotMinutes"`
		Rules       *[]ruleBody `json:"availability"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil || dec.Decode(&struct{}{}) != io.EOF || req.SlotMinutes == nil || req.Rules == nil {
		return Availability{}, false
	}
	in := Availability{SlotMinutes: *req.SlotMinutes, Rules: make([]Rule, 0, len(*req.Rules))}
	for _, rule := range *req.Rules {
		if rule.Day == nil || rule.Start == nil || rule.End == nil {
			return Availability{}, false
		}
		in.Rules = append(in.Rules, Rule{Day: *rule.Day, Start: *rule.Start, End: *rule.End})
	}
	return in, true
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
	{ErrInvalidSlotMinutes, "invalid_slot_minutes", http.StatusBadRequest},
	{ErrInvalidAvailability, "invalid_availability", http.StatusBadRequest},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrConflict, "version_conflict", http.StatusPreconditionFailed},
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
