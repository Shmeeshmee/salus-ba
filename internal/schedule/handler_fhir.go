package schedule

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Public reads the schedules of the public directory, for anyone (Service).
type Public interface {
	Read(ctx context.Context, id string) (Schedule, error)
	ByActor(ctx context.Context, roleID string) (Schedule, error)
}

// Handler serves the Schedule read and search of the FHIR API (api/fhir/CapabilityStatement.json): the
// schedules of active roles of listed clinics, to anyone, signed in or not, unaudited (SPEC.md "Public
// browse").
type Handler struct {
	svc  Public
	log  *slog.Logger
	base string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Public, base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, base: base, log: log}
}

// Routes declares the Schedule routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Schedule/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/Schedule", h.search)
}

// resource is a Schedule as FHIR R4 JSON: its actor, the practitioner role. The weekly rules are not FHIR
// elements; the slots are what a client books.
type resource struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id"`
	Meta         fhir.Meta        `json:"meta"`
	Actor        []fhir.Reference `json:"actor"`
	Active       bool             `json:"active"`
}

func toFHIR(s Schedule) resource {
	return resource{
		ResourceType: "Schedule", ID: s.ID, Active: true, Meta: fhir.NewMeta(s.VersionID, s.LastUpdated),
		Actor: []fhir.Reference{{Reference: "PractitionerRole/" + s.PractitionerRoleID}},
	}
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Read(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(s.VersionID), 10)))
		fhir.Write(w, http.StatusOK, toFHIR(s))
	}
}

// search answers Schedule?actor=PractitionerRole/<id> with the role's schedule, or none.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	role, ok := "", err == nil && len(q) == 1 && len(q["actor"]) == 1
	if ok {
		role, ok = fhir.ReferenceID(q.Get("actor"), "PractitionerRole")
	}
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported", "Search by one actor, a PractitionerRole.")
		return
	}
	var matches []fhir.BundleEntry
	s, err := h.svc.ByActor(r.Context(), role)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return
	default:
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Schedule/" + s.ID, Resource: toFHIR(s)})
	}
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(h.base+"/Schedule?actor=PractitionerRole/"+url.QueryEscape(role), matches, nil))
}
