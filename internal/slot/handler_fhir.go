package slot

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Searcher finds the free slots of the public directory, for anyone, and of the caller's own clinic org for its staff
// (empty org for anyone else) whether or not it is listed (Service).
type Searcher interface {
	SearchFor(ctx context.Context, org, scheduleID string, from, to time.Time) ([]Slot, error)
}

// Handler serves the Slot search of the FHIR API (api/fhir/CapabilityStatement.json): the free future slots
// of a schedule of the public directory, to anyone, signed in or not, and of its own clinic's schedules to its staff
// (the front desk, docs/clinic-cms F3); unaudited (SPEC.md "FHIR reads and searches").
type Handler struct {
	svc    Searcher
	clinic func(ctx context.Context) string // the signed-in staff's clinic; empty for anyone else
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. clinic gives the signed-in staff's clinic (empty for anyone else); base is
// the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Searcher, clinic func(ctx context.Context) string, base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, clinic: clinic, base: base, log: log}
}

// Routes declares the Slot routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Slot", h.search)
}

// ExtensionVisitMode is the extension of a Slot and an Appointment that says how the patient attends, in-person or
// video (api/fhir/StructureDefinition-visit-mode.json, ADR 0016): R4 has no element for it, R5's
// Appointment.virtualService means the same.
const ExtensionVisitMode = "https://salus.invalid/fhir/StructureDefinition/visit-mode"

// resource is a Slot as FHIR R4 JSON.
type resource struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id"`
	Status       Status           `json:"status"`
	Start        string           `json:"start"`
	End          string           `json:"end"`
	Schedule     fhir.Reference   `json:"schedule"`
	Meta         fhir.Meta        `json:"meta"`
	Extension    []fhir.Extension `json:"extension"`
}

func toFHIR(s Slot) resource {
	return resource{
		ResourceType: "Slot", ID: s.ID, Status: s.Status, Meta: fhir.NewMeta(s.VersionID, s.LastUpdated),
		Extension: []fhir.Extension{{URL: ExtensionVisitMode, ValueCode: string(s.Mode)}},
		Start:     s.Start.UTC().Format(time.RFC3339), End: s.End.UTC().Format(time.RFC3339),
		Schedule: fhir.Reference{Reference: "Schedule/" + s.ScheduleID},
	}
}

// search answers Slot?schedule=<id>&status=free&start=lt<instant>[&start=ge<instant>]: the free slots of the
// schedule that start in the range and after now, by start.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	scheduleID, from, to, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by one schedule, status=free and start=lt<instant>, optionally start=ge<instant>.")
		return
	}
	all, err := h.svc.SearchFor(r.Context(), h.clinic(r.Context()), scheduleID, from, to)
	if err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	for _, s := range all {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Slot/" + s.ID, Resource: toFHIR(s)})
	}
	self := h.base + "/Slot?schedule=Schedule/" + url.QueryEscape(scheduleID) + "&status=free"
	if !from.IsZero() {
		self += "&start=ge" + url.QueryEscape(from.UTC().Format(time.RFC3339))
	}
	self += "&start=lt" + url.QueryEscape(to.UTC().Format(time.RFC3339))
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(self, matches, nil))
}

// searchQuery reads the search, and false for any other.
func searchQuery(raw string) (scheduleID string, from, to time.Time, ok bool) {
	q, err := url.ParseQuery(raw)
	if err != nil || len(q) != 3 || len(q["schedule"]) != 1 || len(q["status"]) != 1 || q.Get("status") != string(StatusFree) {
		return "", time.Time{}, time.Time{}, false
	}
	if scheduleID, ok = fhir.ReferenceID(q.Get("schedule"), "Schedule"); !ok {
		return "", time.Time{}, time.Time{}, false
	}
	from, to, ok = fhir.DateRange(q["start"])
	return scheduleID, from, to, ok && !to.IsZero()
}
