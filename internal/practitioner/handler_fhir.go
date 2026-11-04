package practitioner

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Reader reads a practitioner for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Practitioner, error)
}

// Handler serves the Practitioner read of the FHIR API (api/fhir/CapabilityStatement.json).
type Handler struct {
	svc Reader
	// caller is who is signed in for the request, and false for nobody (from the session, in internal/api).
	caller func(ctx context.Context) (Caller, bool)
	log    *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Reader, caller func(ctx context.Context) (Caller, bool), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the Practitioner routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Practitioner/{id}", h.read)
}

// Resource is a Practitioner as FHIR R4 JSON: only the elements SPEC.md "Directory model" names.
type Resource struct {
	ResourceType string      `json:"resourceType"`
	ID           string      `json:"id"`
	Meta         fhir.Meta   `json:"meta"`
	Name         []HumanName `json:"name"`
	Active       bool        `json:"active"`
}

// HumanName is a FHIR HumanName with one given name.
type HumanName struct {
	Family string   `json:"family"`
	Given  []string `json:"given"`
}

// ToFHIR maps p to its FHIR form (also for a PractitionerRole search that includes it).
func ToFHIR(p Practitioner) Resource {
	return Resource{
		ResourceType: "Practitioner", ID: p.ID, Meta: fhir.NewMeta(p.VersionID, p.LastUpdated), Active: p.Active,
		Name: []HumanName{{Family: p.FamilyName, Given: []string{p.GivenName}}},
	}
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.caller(r.Context())
	if !ok {
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
		return
	}
	p, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		res := ToFHIR(p)
		w.Header().Set("ETag", fhir.ETag(res.Meta.VersionID))
		fhir.Write(w, http.StatusOK, res)
	}
}
