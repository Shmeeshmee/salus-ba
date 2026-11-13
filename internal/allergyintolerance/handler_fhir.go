package allergyintolerance

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

// The code systems of an AllergyIntolerance.
const (
	systemClinical     = "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical"
	systemVerification = "http://terminology.hl7.org/CodeSystem/allergyintolerance-verification"
	systemATC          = "http://www.whocc.no/atc"
)

// resource is an AllergyIntolerance as FHIR R4 JSON, with the elements docs/plans/patient-history.md "FHIR mapping"
// names and no others.
type resource struct {
	ClinicalStatus     *fhir.CodeableConcept `json:"clinicalStatus,omitempty"`
	Encounter          *fhir.Reference       `json:"encounter,omitempty"`
	Recorder           *fhir.Reference       `json:"recorder,omitempty"`
	Asserter           *fhir.Reference       `json:"asserter,omitempty"`
	ResourceType       string                `json:"resourceType"`
	ID                 string                `json:"id"`
	RecordedDate       string                `json:"recordedDate"`
	Patient            fhir.Reference        `json:"patient"`
	Meta               fhir.Meta             `json:"meta"`
	Code               fhir.CodeableConcept  `json:"code"`
	VerificationStatus fhir.CodeableConcept  `json:"verificationStatus"`
	Category           []Category            `json:"category,omitempty"`
	Reaction           []reaction            `json:"reaction,omitempty"`
}

type reaction struct {
	Severity      Severity               `json:"severity,omitempty"`
	Manifestation []fhir.CodeableConcept `json:"manifestation"`
}

func concept(system, code string) fhir.CodeableConcept {
	return fhir.CodeableConcept{Coding: []fhir.Coding{{System: system, Code: code}}}
}

// toFHIR is a as a FHIR AllergyIntolerance. attributed (access.Attributed) shows where a clinic's entry comes from:
// its encounter and its recorder; without it neither is sent, so nothing names the clinic. The patient's own entry
// has the patient as its asserter and no recorder. The code is the ATC code when the substance has one, always with
// its text; a typed substance is text only. A mistaken entry has no clinical status (FHIR ait-2).
func toFHIR(a AllergyIntolerance, attributed bool) resource {
	patient := fhir.Reference{Reference: "Patient/" + a.PatientID}
	res := resource{
		ResourceType: "AllergyIntolerance", ID: a.ID, Meta: fhir.NewMeta(a.VersionID, a.LastUpdated), Patient: patient,
		RecordedDate:       a.RecordedDate.UTC().Format(time.RFC3339),
		VerificationStatus: concept(systemVerification, string(a.VerificationStatus)),
		Code:               fhir.CodeableConcept{Text: a.Code.Text},
	}
	if a.VerificationStatus != VerificationEnteredInError {
		c := concept(systemClinical, string(a.ClinicalStatus))
		res.ClinicalStatus = &c
	}
	if a.Code.ATC != "" {
		res.Code.Coding = []fhir.Coding{{System: systemATC, Code: a.Code.ATC}}
	}
	if a.Category != "" {
		res.Category = []Category{a.Category}
	}
	if a.Reaction != "" {
		res.Reaction = []reaction{{Severity: a.Severity, Manifestation: []fhir.CodeableConcept{{Text: a.Reaction}}}}
	}
	switch {
	case a.Source.Patient():
		res.Asserter = &patient
	case attributed:
		res.Encounter = &fhir.Reference{Reference: "Encounter/" + a.Source.EncounterID}
		res.Recorder = &fhir.Reference{Reference: "PractitionerRole/" + a.Source.RecorderRoleID}
	}
	return res
}

// Reader reads and searches the allergies for a caller (Service).
type Reader interface {
	Get(ctx context.Context, c Caller, id string) (Seen, error)
	Seen(ctx context.Context, c Caller, patientID string) ([]Seen, error)
	Provenance(ctx context.Context, c Caller, id string) (Provenance, bool, error)
}

// FHIRHandler serves the AllergyIntolerance read and search by patient of the FHIR API
// (api/fhir/CapabilityStatement.json). The writes are the JSON API's (HTTPHandler).
type FHIRHandler struct {
	svc Reader
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewFHIRHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewFHIRHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), base string, log *slog.Logger) *FHIRHandler {
	return &FHIRHandler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the AllergyIntolerance routes on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/AllergyIntolerance/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/AllergyIntolerance", h.search)
}

func (h *FHIRHandler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Get(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
		return
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(a.VersionID), 10)))
	fhir.Write(w, http.StatusOK, toFHIR(a.AllergyIntolerance, a.Attributed))
}

// search answers AllergyIntolerance?patient=…, the one search there is.
func (h *FHIRHandler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	values := q["patient"]
	var patientID string
	if ok = err == nil && len(q) == 1 && len(values) == 1; ok {
		patientID, ok = fhir.ReferenceID(values[0], "Patient")
	}
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported", "Search by patient, once.")
		return
	}
	all, err := h.svc.Seen(r.Context(), c, patientID)
	if err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	for _, a := range all {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/AllergyIntolerance/" + a.ID, Resource: toFHIR(a.AllergyIntolerance, a.Attributed)})
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(h.base+"/AllergyIntolerance?"+q.Encode(), matches, nil))
}

// Provenance answers GET /fhir/R4/Provenance/{id} and reports true when the id is an allergy's Provenance in the
// caller's scope (or nobody is signed in); otherwise it writes nothing and reports false, for the route's next
// source (internal/api/router).
func (h *FHIRHandler) Provenance(w http.ResponseWriter, r *http.Request) bool {
	c, ok := h.signedIn(w, r)
	if !ok {
		return true
	}
	p, found, err := h.svc.Provenance(r.Context(), c, r.PathValue("id"))
	switch {
	case !found:
		return false
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
		return true
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return true
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag("1"))
	fhir.Write(w, http.StatusOK, provenanceOf(p))
	return true
}

// provenanceOf is p as an R4 Provenance: a clinic's role on behalf of the clinic, or the patient. Recording confirms
// the entry with the recorder's role (00053 checks it), so a creation's agent is its author and its verifier.
func provenanceOf(p Provenance) fhir.Provenance {
	c := fhir.Change{Recorded: p.Recorded, ID: p.ID, Resource: "AllergyIntolerance/" + p.EntryID, Activity: p.Activity,
		AgentTypes: []string{p.AgentType}, Who: "Patient/" + p.PatientID, Version: p.Version}
	if p.PatientID == "" {
		c.Who, c.OnBehalfOf = "PractitionerRole/"+p.RoleID, "Organization/"+p.OrganizationID
	}
	if p.Activity == fhir.ActivityCreate {
		c.AgentTypes = append(c.AgentTypes, fhir.AgentVerifier)
	}
	return fhir.NewWriteProvenance(c)
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *FHIRHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	case !ok:
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok && err == nil
}
