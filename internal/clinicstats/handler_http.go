package clinicstats

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Stats is what the HTTPHandler calls: Service.
type Stats interface {
	Appointments(ctx context.Context, c Caller, q AppointmentQuery) (AppointmentFigures, error)
	Diagnoses(ctx context.Context, c Caller, q DiagnosisQuery) ([]Count, error)
}

// HTTPHandler serves the clinic's figures on the JSON API (api/openapi.yaml, docs/plans/analytics.md §8).
type HTTPHandler struct {
	svc    Stats
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the HTTPHandler; caller gives who is signed in, false for nobody.
func NewHTTPHandler(svc Stats, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the figures' routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /analytics/clinic/appointments", h.appointments)
	mux.HandleFunc("GET /analytics/clinic/diagnoses", h.diagnoses)
}

// periodJSON, heatJSON, practitionerJSON and countJSON are the ClinicAppointments and ClinicDiagnoses schemas' items.
type periodJSON struct {
	Period    string `json:"period"`
	Booked    int64  `json:"booked"`
	Fulfilled int64  `json:"fulfilled"`
	NoShow    int64  `json:"noshow"`
	Cancelled int64  `json:"cancelled"`
}

type heatJSON struct {
	Weekday int32 `json:"weekday"`
	Hour    int32 `json:"hour"`
	Count   int64 `json:"count"`
}

type practitionerJSON struct {
	PractitionerRole string `json:"practitionerRole"`
	Fulfilled        int64  `json:"fulfilled"`
}

type countJSON struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// appointmentsJSON is the ClinicAppointments schema; practitioners only for an admin.
type appointmentsJSON struct {
	Practitioners *[]practitionerJSON `json:"practitioners,omitempty"`
	Series        []periodJSON        `json:"series"`
	Heatmap       []heatJSON          `json:"heatmap"`
}

// appointments answers GET /analytics/clinic/appointments?from=&to=&practitionerRole=&specialty=.
func (h *HTTPHandler) appointments(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v := r.URL.Query()
	from, to, ok := dates(v)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	f, err := h.svc.Appointments(r.Context(), c, AppointmentQuery{
		From: from, To: to, PractitionerRoleID: v.Get("practitionerRole"), Specialty: v.Get("specialty"),
	})
	if h.failed(w, r, err) {
		return
	}
	out := appointmentsJSON{Series: make([]periodJSON, len(f.Series)), Heatmap: make([]heatJSON, len(f.Heatmap))}
	for i, p := range f.Series {
		out.Series[i] = periodJSON(p)
	}
	for i, c := range f.Heatmap {
		out.Heatmap[i] = heatJSON{Weekday: c.Weekday, Hour: c.Hour, Count: c.Count}
	}
	if f.PerPractitioner {
		list := make([]practitionerJSON, len(f.Practitioners))
		for i, p := range f.Practitioners {
			list[i] = practitionerJSON{PractitionerRole: p.PractitionerRoleID, Fulfilled: p.Fulfilled}
		}
		out.Practitioners = &list
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.JSON(w, http.StatusOK, out)
}

// diagnoses answers GET /analytics/clinic/diagnoses?from=&to=&by=&group=&subgroup=&specialty=&practitionerRole=.
func (h *HTTPHandler) diagnoses(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v := r.URL.Query()
	from, to, ok := dates(v)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	rows, err := h.svc.Diagnoses(r.Context(), c, DiagnosisQuery{
		From: from, To: to, By: Breakdown(v.Get("by")), Group: v.Get("group"), Subgroup: v.Get("subgroup"),
		Specialty: v.Get("specialty"), PractitionerRoleID: v.Get("practitionerRole"),
	})
	if h.failed(w, r, err) {
		return
	}
	out := make([]countJSON, len(rows))
	for i, c := range rows {
		out[i] = countJSON(c)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.JSON(w, http.StatusOK, map[string][]countJSON{"rows": out})
}

// dates parses the from and to dates (YYYY-MM-DD); false when either is malformed. A missing one is zero, which
// the service refuses.
func dates(v url.Values) (from, to time.Time, ok bool) {
	parse := func(s string) (time.Time, bool) {
		if s == "" {
			return time.Time{}, true
		}
		t, err := time.Parse(time.DateOnly, s)
		return t, err == nil
	}
	from, okFrom := parse(v.Get("from"))
	to, okTo := parse(v.Get("to"))
	return from, to, okFrom && okTo
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

// failed writes the error's status and reports whether there was one.
func (h *HTTPHandler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidQuery):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
