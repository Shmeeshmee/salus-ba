package consent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The code systems of a Consent (docs/booking/SPEC.md "Consent shape").
const (
	systemConsentScope      = "http://terminology.hl7.org/CodeSystem/consentscope"
	systemLOINC             = "http://loinc.org"
	systemActCode           = "http://terminology.hl7.org/CodeSystem/v3-ActCode"
	systemActReason         = "http://terminology.hl7.org/CodeSystem/v3-ActReason"
	systemParticipationType = "http://terminology.hl7.org/CodeSystem/v3-ParticipationType"
)

// Reader reads a consent for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Consent, error)
}

// FHIRHandler serves the Consent read of the FHIR API (api/fhir/CapabilityStatement.json): the patient and
// staff of the consent's clinic; anyone else gets 404.
type FHIRHandler struct {
	svc Reader
	// caller is who is signed in for the request, with a patient's profile id, and false for nobody (in
	// internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewFHIRHandler returns the handler on svc.
func NewFHIRHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *FHIRHandler {
	return &FHIRHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the Consent routes on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Consent/{id}", h.read)
}

// resource is a Consent as FHIR R4 JSON (docs/booking/SPEC.md "Consent shape").
type resource struct {
	ResourceType string                 `json:"resourceType"`
	ID           string                 `json:"id"`
	Status       Status                 `json:"status"`
	DateTime     string                 `json:"dateTime"`
	Patient      fhir.Reference         `json:"patient"`
	Scope        fhir.CodeableConcept   `json:"scope"`
	PolicyRule   fhir.CodeableConcept   `json:"policyRule"`
	Category     []fhir.CodeableConcept `json:"category"`
	Performer    []fhir.Reference       `json:"performer"`
	Organization []fhir.Reference       `json:"organization"`
	Meta         fhir.Meta              `json:"meta"`
	Provision    provision              `json:"provision"`
}

// provision permits the clinic (IRCP, the information recipient) to use the data for treatment.
type provision struct {
	Type    string        `json:"type"`
	Actor   []actor       `json:"actor"`
	Purpose []fhir.Coding `json:"purpose"`
}

type actor struct {
	Reference fhir.Reference       `json:"reference"`
	Role      fhir.CodeableConcept `json:"role"`
}

func concept(system, code string) fhir.CodeableConcept {
	return fhir.CodeableConcept{Coding: []fhir.Coding{{System: system, Code: code}}}
}

func toFHIR(c Consent) resource {
	patient := fhir.Reference{Reference: "Patient/" + c.PatientID}
	clinic := fhir.Reference{Reference: "Organization/" + c.OrganizationID}
	return resource{
		ResourceType: "Consent", ID: c.ID, Status: c.Status, Meta: fhir.NewMeta(c.VersionID, c.LastUpdated),
		DateTime: c.DateTime.UTC().Format(time.RFC3339),
		Scope:    concept(systemConsentScope, "patient-privacy"),
		Category: []fhir.CodeableConcept{concept(systemLOINC, "59284-0")},
		Patient:  patient, Performer: []fhir.Reference{patient}, Organization: []fhir.Reference{clinic},
		PolicyRule: concept(systemActCode, "OPTIN"),
		Provision: provision{
			Type:    "permit",
			Actor:   []actor{{Role: concept(systemParticipationType, "IRCP"), Reference: clinic}},
			Purpose: []fhir.Coding{{System: systemActReason, Code: "TREAT"}},
		},
	}
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
	out, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		res := toFHIR(out)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(res.Meta.VersionID))
		fhir.Write(w, http.StatusOK, res)
	}
}
