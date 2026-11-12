package condition

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The code systems of a Condition's status and category.
const (
	systemVerification = "http://terminology.hl7.org/CodeSystem/condition-ver-status"
	systemCategory     = "http://terminology.hl7.org/CodeSystem/condition-category"
)

// Reader reads a Condition for a caller (Reads).
type Reader interface {
	Get(ctx context.Context, c appointment.Caller, id string) (Condition, error)
}

// FHIRHandler serves the Condition read of the FHIR API (api/fhir/CapabilityStatement.json). No search, create
// or update: a Condition is written with its document's signature and never changes.
type FHIRHandler struct {
	svc    Reader
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewFHIRHandler returns the handler on svc; caller is who is signed in, false for nobody (internal/api).
func NewFHIRHandler(svc Reader, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *FHIRHandler {
	return &FHIRHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the Condition route on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Condition/{id}", h.read)
}

// resource is a Condition as FHIR R4 JSON: an encounter diagnosis of a signed document (its evidence).
type resource struct {
	ResourceType       string                 `json:"resourceType"`
	ID                 string                 `json:"id"`
	Meta               fhir.Meta              `json:"meta"`
	VerificationStatus fhir.CodeableConcept   `json:"verificationStatus"`
	Category           []fhir.CodeableConcept `json:"category"`
	Code               fhir.CodeableConcept   `json:"code"`
	Subject            fhir.Reference         `json:"subject"`
	Encounter          fhir.Reference         `json:"encounter"`
	RecordedDate       string                 `json:"recordedDate"`
	Recorder           fhir.Reference         `json:"recorder"`
	Evidence           []evidence             `json:"evidence"`
}

type evidence struct {
	Detail []fhir.Reference `json:"detail"`
}

func toFHIR(c Condition) resource {
	return resource{
		ResourceType: "Condition", ID: c.ID, Meta: fhir.NewMeta(c.VersionID, c.LastUpdated),
		VerificationStatus: fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemVerification, Code: c.VerificationStatus}}},
		Category:           []fhir.CodeableConcept{{Coding: []fhir.Coding{{System: systemCategory, Code: "encounter-diagnosis"}}}},
		Code:               fhir.CodeableConcept{Coding: []fhir.Coding{{System: SystemICD10, Code: c.Code}}},
		Subject:            fhir.Reference{Reference: "Patient/" + c.PatientID},
		Encounter:          fhir.Reference{Reference: "Encounter/" + c.EncounterID},
		RecordedDate:       c.RecordedAt.UTC().Format(time.RFC3339),
		Recorder:           fhir.Reference{Reference: "PractitionerRole/" + c.PractitionerRoleID},
		Evidence:           []evidence{{Detail: []fhir.Reference{{Reference: "Composition/" + c.DocumentID}}}},
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
	cond, err := h.svc.Get(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrConsentRequired):
		fhir.WriteError(w, http.StatusForbidden, "forbidden",
			"Only the encounter's practitioner (while the patient's consent is active) and the encounter's patient may read its diagnoses.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(cond.VersionID), 10)))
		fhir.Write(w, http.StatusOK, toFHIR(cond))
	}
}
