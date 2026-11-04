package patient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Reader reads a profile for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Patient, error)
}

// FHIRHandler serves the Patient read of the FHIR API (api/fhir/CapabilityStatement.json): the patient
// themselves, and a practitioner whom access gives all of the profile; anyone else gets 404.
type FHIRHandler struct {
	svc Reader
	// caller is who is signed in for the request, with a practitioner's role, and false for nobody (from the
	// session and the account, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewFHIRHandler returns the handler on svc.
func NewFHIRHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *FHIRHandler {
	return &FHIRHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the Patient routes on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Patient/{id}", h.read)
}

// resource is a Patient as FHIR R4 JSON: only the elements SPEC.md "Profile" names.
type resource struct {
	ResourceType string         `json:"resourceType"`
	ID           string         `json:"id"`
	BirthDate    string         `json:"birthDate"`
	Meta         fhir.Meta      `json:"meta"`
	Name         []humanName    `json:"name"`
	Telecom      []contactPoint `json:"telecom,omitempty"`
}

type humanName struct {
	Use    string   `json:"use"`
	Family string   `json:"family"`
	Given  []string `json:"given"`
}

type contactPoint struct {
	System string `json:"system"`
	Value  string `json:"value"`
	Use    string `json:"use"`
}

func toFHIR(p Patient) resource {
	res := resource{
		ResourceType: "Patient", ID: p.ID, Meta: fhir.NewMeta(p.VersionID, p.LastUpdated),
		Name:      []humanName{{Use: "official", Family: p.FamilyName, Given: []string{p.GivenName}}},
		BirthDate: p.BirthDate.Format(dateLayout),
	}
	if p.Phone != "" {
		res.Telecom = []contactPoint{{System: "phone", Value: p.Phone, Use: "mobile"}}
	}
	return res
}

func (h *FHIRHandler) read(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return
	case !ok:
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
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(p.VersionID), 10)))
		fhir.Write(w, http.StatusOK, toFHIR(p))
	}
}
