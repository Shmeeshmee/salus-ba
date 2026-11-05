package appointment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Reader reads and searches appointments for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Seen, error)
	Search(ctx context.Context, c Caller, f Filter) ([]Seen, error)
}

// Handler serves the Appointment read and search of the FHIR API (api/fhir/CapabilityStatement.json), each
// appointment as access lets the caller see it (SPEC.md "Access rule", "FHIR reads and searches").
type Handler struct {
	svc Reader
	// caller is who is signed in for the request, with a patient's profile id or a staff account's role in its
	// clinic, and false for nobody (in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the Appointment routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Appointment/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/Appointment", h.search)
}

// resource is an Appointment as FHIR R4 JSON. At the name level (a past appointment after a revocation) the
// service leaves out the slot, the created time and the cancellation details, and so does the resource.
type resource struct {
	ResourceType      string                `json:"resourceType"`
	ID                string                `json:"id"`
	Status            Status                `json:"status"`
	Start             string                `json:"start"`
	End               string                `json:"end"`
	Created           string                `json:"created,omitempty"`
	CancelationReason *fhir.CodeableConcept `json:"cancelationReason,omitempty"`
	Meta              fhir.Meta             `json:"meta"`
	Slot              []fhir.Reference      `json:"slot,omitempty"`
	Participant       []participant         `json:"participant"`
}

// participant is the patient, the practitioner or the practitioner role, each accepted.
type participant struct {
	Status string         `json:"status"`
	Actor  fhir.Reference `json:"actor"`
}

func instant(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func toFHIR(s Seen) resource {
	a := s.Appointment
	patient := fhir.Reference{Reference: "Patient/" + a.PatientID}
	if s.Patient.Family != "" { // the name only where Profile allows it
		patient.Display = s.Patient.Given + " " + s.Patient.Family
	}
	res := resource{
		ResourceType: "Appointment", ID: a.ID, Status: a.Status, Meta: fhir.NewMeta(a.VersionID, a.LastUpdated),
		Start: instant(a.Start), End: instant(a.End), Participant: []participant{{Status: "accepted", Actor: patient}},
	}
	if s.PractitionerID != "" {
		res.Participant = append(res.Participant, participant{Status: "accepted", Actor: fhir.Reference{Reference: "Practitioner/" + s.PractitionerID}})
	}
	res.Participant = append(res.Participant, participant{Status: "accepted", Actor: fhir.Reference{Reference: "PractitionerRole/" + a.PractitionerRoleID}})
	if !a.Created.IsZero() {
		res.Created = instant(a.Created)
	}
	if a.SlotID != "" {
		res.Slot = []fhir.Reference{{Reference: "Slot/" + a.SlotID}}
	}
	if a.Reason != "" {
		res.CancelationReason = &fhir.CodeableConcept{Coding: []fhir.Coding{{System: SystemCancellationReason, Code: string(a.Reason)}}}
	}
	return res
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(s.Appointment.VersionID), 10)))
	fhir.Write(w, http.StatusOK, toFHIR(s))
}

// search answers Appointment?[patient=…][&practitioner=…][&date=ge…][&date=lt…][&status=…].
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, f, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by patient, practitioner, date (ge and lt, instants with an offset) and status, each once.")
		return
	}
	all, err := h.svc.Search(r.Context(), c, f)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	for _, s := range all {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Appointment/" + s.Appointment.ID, Resource: toFHIR(s)})
	}
	self := h.base + "/Appointment"
	if len(q) > 0 {
		self += "?" + q.Encode()
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(self, matches, nil))
}

// searchQuery reads the search into a filter, and false for an unknown or repeated parameter or a value it
// cannot read.
func searchQuery(raw string) (url.Values, Filter, bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return nil, Filter{}, false
	}
	var f Filter
	for key, values := range q {
		ok := len(values) == 1
		switch key {
		case "patient":
			f.PatientID, ok = fhir.ReferenceID(values[0], "Patient")
			ok = ok && len(values) == 1
		case "practitioner":
			f.PractitionerID, ok = fhir.ReferenceID(values[0], "Practitioner")
			ok = ok && len(values) == 1
		case "status":
			f.Status = Status(values[0])
			ok = ok && (f.Status == StatusBooked || f.Status == StatusCancelled || f.Status == StatusFulfilled || f.Status == StatusNoShow)
		case "date":
			f.From, f.To, ok = fhir.DateRange(values)
		default:
			ok = false
		}
		if !ok {
			return nil, Filter{}, false
		}
	}
	return q, f, true
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	case !ok:
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok && err == nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case errors.Is(err, ErrForbidden):
		fhir.WriteError(w, http.StatusForbidden, "forbidden", "Search your own appointments only.")
	default:
		fhir.InternalError(w, r, h.log, err)
	}
}
