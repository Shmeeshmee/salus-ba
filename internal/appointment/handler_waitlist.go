package appointment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Waitlist is what the waitlist's handler calls (Service).
type Waitlist interface {
	Join(ctx context.Context, c Caller, in NewEntry) (WaitlistEntry, error)
	MyWaitlist(ctx context.Context, c Caller) ([]WaitlistEntry, error)
	ClinicWaitlist(ctx context.Context, c Caller, f WaitlistFilter) ([]ClinicEntry, error)
	Withdraw(ctx context.Context, c Caller, id string) error
	SetPriority(ctx context.Context, c Caller, id string, priority int) (ClinicEntry, error)
	Accept(ctx context.Context, c Caller, id string) (Appointment, error)
	Decline(ctx context.Context, c Caller, id string) error
}

// WaitlistHandler serves the waitlist's routes of the JSON API (api/openapi.yaml, tag waitlist, SAL-35). Errors are
// {"error": "<code>"}, as the appointment routes'.
type WaitlistHandler struct {
	svc Waitlist
	h   *HTTPHandler // its signedIn and writeError
}

// NewWaitlistHandler returns the handler on svc.
func NewWaitlistHandler(svc Waitlist, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *WaitlistHandler {
	return &WaitlistHandler{svc: svc, h: &HTTPHandler{caller: caller, log: log}}
}

// Routes declares the waitlist's routes on mux.
func (h *WaitlistHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /waitlist", h.join)
	mux.HandleFunc("GET /me/waitlist", h.mine)
	mux.HandleFunc("DELETE /waitlist/{id}", h.withdraw)
	mux.HandleFunc("POST /waitlist/offers/{id}/accept", h.accept)
	mux.HandleFunc("POST /waitlist/offers/{id}/decline", h.decline)
	mux.HandleFunc("GET /clinic/waitlist", h.clinic)
	mux.HandleFunc("PUT /clinic/waitlist/{id}/priority", h.priority)
}

// The waitlist's error codes beside the appointment routes' (errorCodes).
var waitlistCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalidEntry, "invalid_request", http.StatusBadRequest},
	{ErrInvalidPriority, "invalid_request", http.StatusBadRequest},
	{ErrWaitlistOff, "waitlist_off", http.StatusConflict},
	{ErrTooManyEntries, "too_many_entries", http.StatusConflict},
	{ErrOfferExpired, "offer_expired", http.StatusConflict},
}

func (h *WaitlistHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range waitlistCodes {
		if errors.Is(err, c.err) {
			writeCode(w, c.status, c.code)
			return
		}
	}
	h.h.writeError(w, r, err)
}

// timesJSON is the WaitlistTimes schema: HH:MM in the clinic's zone.
type timesJSON struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// offerJSON is the WaitlistOffer schema.
type offerJSON struct {
	SlotStart time.Time `json:"slot_start"`
	SlotEnd   time.Time `json:"slot_end"`
	ExpiresAt time.Time `json:"expires_at"`
	ID        string    `json:"id"`
}

// entryJSON is the WaitlistEntry schema.
type entryJSON struct {
	CreatedAt          time.Time      `json:"created_at"`
	Times              *timesJSON     `json:"times,omitempty"`
	Offer              *offerJSON     `json:"offer,omitempty"`
	ID                 string         `json:"id"`
	OrganizationID     string         `json:"organization_id"`
	PractitionerRoleID string         `json:"practitioner_role_id,omitempty"`
	Specialty          string         `json:"specialty,omitempty"`
	Earliest           string         `json:"earliest"`
	Latest             string         `json:"latest"`
	Status             WaitlistStatus `json:"status"`
}

// clinicEntryJSON is the ClinicWaitlistEntry schema.
type clinicEntryJSON struct {
	PatientID   string    `json:"patient_id"`
	PatientName string    `json:"patient_name,omitempty"`
	Entry       entryJSON `json:"entry"`
	Priority    int       `json:"priority"`
}

func toEntryJSON(e WaitlistEntry) entryJSON {
	out := entryJSON{
		CreatedAt: e.Created.UTC(), ID: e.ID, OrganizationID: e.OrganizationID, PractitionerRoleID: e.PractitionerRoleID,
		Specialty: e.Specialty, Earliest: e.Earliest.Format(time.DateOnly), Latest: e.Latest.Format(time.DateOnly), Status: e.Status,
	}
	if !e.Times.Any() {
		out.Times = &timesJSON{Start: hhmm(e.Times.Start), End: hhmm(e.Times.End)}
	}
	if e.Offer.ID != "" {
		out.Offer = &offerJSON{ID: e.Offer.ID, SlotStart: e.Offer.Start.UTC(), SlotEnd: e.Offer.End.UTC(), ExpiresAt: e.Offer.ExpiresAt.UTC()}
	}
	return out
}

func toClinicEntryJSON(e ClinicEntry) clinicEntryJSON {
	out := clinicEntryJSON{PatientID: e.Entry.PatientID, Entry: toEntryJSON(e.Entry), Priority: e.Entry.Priority}
	if e.Named {
		out.PatientName = strings.TrimSpace(e.Patient.Given + " " + e.Patient.Family)
	}
	return out
}

// hhmm is minutes after midnight as HH:MM (24:00 for the end of the day).
func hhmm(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

// parseHHMM parses HH:MM (00:00 to 24:00) into minutes after midnight.
func parseHHMM(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	switch {
	case s == "24:00":
		return minutesPerDay, true
	case err != nil || len(s) != len("15:04"):
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func (h *WaitlistHandler) join(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		Times              *timesJSON `json:"times"`
		OrganizationID     string     `json:"organization_id"`
		PractitionerRoleID string     `json:"practitioner_role_id"`
		Specialty          string     `json:"specialty"`
		Earliest           string     `json:"earliest"`
		Latest             string     `json:"latest"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	in, ok := newEntry(req.OrganizationID, req.PractitionerRoleID, req.Specialty, req.Earliest, req.Latest, req.Times)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.Join(r.Context(), c, in)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusCreated, toEntryJSON(out))
}

// newEntry parses a request to join, or reports false for one that does not parse.
func newEntry(org, role, specialty, earliest, latest string, times *timesJSON) (NewEntry, bool) {
	from, ferr := time.Parse(time.DateOnly, earliest)
	to, terr := time.Parse(time.DateOnly, latest)
	if org == "" || ferr != nil || terr != nil {
		return NewEntry{}, false
	}
	in := NewEntry{OrganizationID: org, PractitionerRoleID: role, Specialty: specialty, Earliest: from, Latest: to}
	if times != nil {
		start, sok := parseHHMM(times.Start)
		end, eok := parseHHMM(times.End)
		if !sok || !eok || end == 0 {
			return NewEntry{}, false
		}
		in.Times = Times{Start: start, End: end}
	}
	return in, true
}

func (h *WaitlistHandler) mine(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.MyWaitlist(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := struct {
		Entries []entryJSON `json:"entries"`
	}{Entries: make([]entryJSON, 0, len(all))}
	for _, e := range all {
		out.Entries = append(out.Entries, toEntryJSON(e))
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *WaitlistHandler) clinic(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := WaitlistFilter{PractitionerRoleID: q.Get("practitioner_role"), Status: WaitlistStatus(q.Get("status"))}
	switch f.Status {
	case "", WaitlistActive, WaitlistBooked, WaitlistWithdrawn, WaitlistExpired:
	default:
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	all, err := h.svc.ClinicWaitlist(r.Context(), c, f)
	if errors.Is(err, ErrForbidden) {
		writeCode(w, http.StatusNotFound, "not_found") // the ClinicNotFound response: a patient has no clinic
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := struct {
		Entries []clinicEntryJSON `json:"entries"`
	}{Entries: make([]clinicEntryJSON, 0, len(all))}
	for _, e := range all {
		out.Entries = append(out.Entries, toClinicEntryJSON(e))
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *WaitlistHandler) priority(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		Priority *int `json:"priority"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Priority == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.SetPriority(r.Context(), c, r.PathValue("id"), *req.Priority)
	if errors.Is(err, ErrForbidden) {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toClinicEntryJSON(out))
}

func (h *WaitlistHandler) withdraw(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.Withdraw(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WaitlistHandler) accept(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Accept(r.Context(), c, r.PathValue("id"))
	if errors.Is(err, ErrForbidden) {
		writeCode(w, http.StatusNotFound, "not_found") // staff have no offers
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusCreated, toJSON(out))
}

func (h *WaitlistHandler) decline(w http.ResponseWriter, r *http.Request) {
	c, ok := h.h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.Decline(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
