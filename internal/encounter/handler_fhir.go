package encounter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The codes of the consultation's Encounter (SPEC.md §2 "Encounter").
const (
	systemActCode           = "http://terminology.hl7.org/CodeSystem/v3-ActCode"
	systemParticipationType = "http://terminology.hl7.org/CodeSystem/v3-ParticipationType"
)

// Reader reads and searches encounters for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c appointment.Caller, id string) (Encounter, error)
	SearchPage(ctx context.Context, c appointment.Caller, f Filter, count int) (Page, error)
}

// Handler serves the Encounter read and search of the FHIR API (api/fhir/CapabilityStatement.json). There is
// no create or update: the consultation's lifecycle drives the Encounter.
type Handler struct {
	svc Reader
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Reader, caller func(ctx context.Context) (appointment.Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the Encounter routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Encounter/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/Encounter", h.search)
}

// resource is an Encounter as FHIR R4 JSON.
type resource struct {
	ResourceType    string           `json:"resourceType"`
	ID              string           `json:"id"`
	Meta            fhir.Meta        `json:"meta"`
	Status          Status           `json:"status"`
	Class           fhir.Coding      `json:"class"`
	Subject         fhir.Reference   `json:"subject"`
	Period          period           `json:"period"`
	ServiceProvider fhir.Reference   `json:"serviceProvider"`
	Participant     []participant    `json:"participant"`
	Appointment     []fhir.Reference `json:"appointment"`
}

// participant is the practitioner role, the primary performer.
type participant struct {
	Individual fhir.Reference         `json:"individual"`
	Type       []fhir.CodeableConcept `json:"type"`
}

type period struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// FHIR is the encounter as an R4 Encounter, as the FHIR API serves it (the patient's export writes it too, D7).
func FHIR(e Encounter) any { return toFHIR(e) }

func toFHIR(e Encounter) resource {
	return resource{
		ResourceType: "Encounter", ID: e.ID, Meta: fhir.NewMeta(e.VersionID, e.LastUpdated), Status: e.Status,
		Class:   fhir.Coding{System: systemActCode, Code: "VR", Display: "virtual"},
		Subject: fhir.Reference{Reference: "Patient/" + e.PatientID},
		Participant: []participant{{
			Type:       []fhir.CodeableConcept{{Coding: []fhir.Coding{{System: systemParticipationType, Code: "PPRF", Display: "primary performer"}}}},
			Individual: fhir.Reference{Reference: "PractitionerRole/" + e.PractitionerRoleID},
		}},
		Appointment:     []fhir.Reference{{Reference: "Appointment/" + e.AppointmentID}},
		Period:          period{Start: instant(e.PeriodStart), End: instant(e.PeriodEnd)},
		ServiceProvider: fhir.Reference{Reference: "Organization/" + e.OrganizationID},
	}
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	e, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(e.VersionID), 10)))
	fhir.Write(w, http.StatusOK, toFHIR(e))
}

// search answers Encounter?[appointment=…][&patient=…][&practitioner=…][&_count=…][&_cursor=…], a page at a time
// where the service pages it (SearchPage), with the next page's link while more may follow.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, f, count, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by appointment, patient, practitioner and _count (1 to 200), each once, from a next link's _cursor.")
		return
	}
	page, err := h.svc.SearchPage(r.Context(), c, f, count)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(page.Encounters))
	for _, e := range page.Encounters {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Encounter/" + e.ID, Resource: toFHIR(e)})
	}
	var next string
	if page.More {
		last := page.Encounters[len(page.Encounters)-1]
		next = fhir.NextLink(h.base+"/Encounter", q, fhir.NewCursor(last.PeriodStart, last.ID))
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewPage(fhir.SelfLink(h.base+"/Encounter", q), next, f.AfterID == "", matches, nil))
}

// searchQuery reads the search into a filter and the count it asks for (0 for none), and false for an unknown or
// repeated parameter or a value it cannot read.
func searchQuery(raw string) (url.Values, Filter, int, bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return nil, Filter{}, 0, false
	}
	var f Filter
	var count int
	for key, values := range q {
		var id string
		var ok bool
		switch key {
		case "appointment":
			id, ok = fhir.ReferenceID(values[0], "Appointment")
			f.AppointmentID = id
		case "patient":
			id, ok = fhir.ReferenceID(values[0], "Patient")
			f.PatientID = id
		case "practitioner":
			id, ok = fhir.ReferenceID(values[0], "Practitioner")
			f.PractitionerID = id
		case "_count":
			count, ok = fhir.Count(values, appointment.MaxCount)
		case "_cursor":
			f.AfterStart, f.AfterID, ok = fhir.ParseCursor(values[0])
			ok = ok && !f.AfterStart.IsZero()
		}
		if !ok || len(values) != 1 {
			return nil, Filter{}, 0, false
		}
	}
	return q, f, count, true
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
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
		fhir.WriteError(w, http.StatusForbidden, "forbidden", "Search your own encounters only.")
	default:
		fhir.InternalError(w, r, h.log, err)
	}
}
