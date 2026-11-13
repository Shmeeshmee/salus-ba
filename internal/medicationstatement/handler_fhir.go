package medicationstatement

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

// The code systems and the extension of a MedicationStatement.
const (
	systemATC  = "http://www.whocc.no/atc"
	systemUCUM = "http://unitsofmeasure.org"
	// extensionConfirmed marks a statement a practitioner confirmed: MedicationStatement has no verification status
	// of its own. Who confirmed is never in it.
	extensionConfirmed = "https://salus.invalid/fhir/StructureDefinition/medicationstatement-confirmed"
)

// resource is a MedicationStatement as FHIR R4 JSON, with the elements docs/plans/patient-history.md "FHIR mapping"
// names and no others.
type resource struct {
	Context                   *fhir.Reference      `json:"context,omitempty"`
	InformationSource         *fhir.Reference      `json:"informationSource,omitempty"`
	ResourceType              string               `json:"resourceType"`
	ID                        string               `json:"id"`
	Status                    Status               `json:"status"`
	DateAsserted              string               `json:"dateAsserted"`
	Subject                   fhir.Reference       `json:"subject"`
	Meta                      fhir.Meta            `json:"meta"`
	MedicationCodeableConcept fhir.CodeableConcept `json:"medicationCodeableConcept"`
	Extension                 []fhir.Extension     `json:"extension,omitempty"`
	Dosage                    []dosage             `json:"dosage,omitempty"`
}

type dosage struct {
	Timing          *timing       `json:"timing,omitempty"`
	AsNeededBoolean *bool         `json:"asNeededBoolean,omitempty"`
	DoseAndRate     []doseAndRate `json:"doseAndRate,omitempty"`
}

type timing struct {
	Repeat repeat `json:"repeat"`
}

type repeat struct {
	PeriodUnit string  `json:"periodUnit"`
	Period     float64 `json:"period"`
	Frequency  int32   `json:"frequency"`
}

type doseAndRate struct {
	DoseQuantity quantity `json:"doseQuantity"`
}

type quantity struct {
	Unit   string  `json:"unit"`
	System string  `json:"system"`
	Code   string  `json:"code"`
	Value  float64 `json:"value"`
}

// toFHIR is m as a FHIR MedicationStatement. attributed (access.Attributed) shows where a clinic's entry comes from:
// its encounter (context) and its recorder (informationSource); without it neither is sent. The patient's own entry
// has the patient as its information source. A confirmed entry carries the confirmed extension. The dosage has the
// dose as a UCUM quantity and the timing as frequency per period, each only when given, and asNeeded when set.
func toFHIR(m MedicationStatement, attributed bool) resource {
	subject := fhir.Reference{Reference: "Patient/" + m.PatientID}
	res := resource{
		ResourceType: "MedicationStatement", ID: m.ID, Meta: fhir.NewMeta(m.VersionID, m.LastUpdated), Subject: subject,
		Status: m.Status, DateAsserted: m.DateAsserted.UTC().Format(time.RFC3339),
		MedicationCodeableConcept: fhir.CodeableConcept{Text: m.Code.Text},
	}
	if m.Code.ATC != "" {
		res.MedicationCodeableConcept.Coding = []fhir.Coding{{System: systemATC, Code: m.Code.ATC}}
	}
	if !m.Confirmation.At.IsZero() {
		confirmed := true
		res.Extension = []fhir.Extension{{URL: extensionConfirmed, ValueBoolean: &confirmed}}
	}
	if d, ok := toDosage(m.Dosage); ok {
		res.Dosage = []dosage{d}
	}
	switch {
	case m.Source.Patient():
		res.InformationSource = &subject
	case attributed:
		res.Context = &fhir.Reference{Reference: "Encounter/" + m.Source.EncounterID}
		res.InformationSource = &fhir.Reference{Reference: "PractitionerRole/" + m.Source.RecorderRoleID}
	}
	return res
}

// toDosage is d as a FHIR Dosage, and false when it says nothing.
func toDosage(d Dosage) (dosage, bool) {
	var out dosage
	if d.DoseUnit != "" {
		out.DoseAndRate = []doseAndRate{{DoseQuantity: quantity{Value: d.DoseValue, Unit: d.DoseUnit, System: systemUCUM, Code: d.DoseUnit}}}
	}
	if d.PeriodUnit != "" {
		out.Timing = &timing{Repeat: repeat{Frequency: d.Frequency, Period: d.Period, PeriodUnit: d.PeriodUnit}}
	}
	if d.AsNeeded {
		asNeeded := true
		out.AsNeededBoolean = &asNeeded
	}
	return out, out.DoseAndRate != nil || out.Timing != nil || out.AsNeededBoolean != nil
}

// Reader reads and searches the medications for a caller (Service).
type Reader interface {
	Get(ctx context.Context, c Caller, id string) (Seen, error)
	Seen(ctx context.Context, c Caller, patientID string) ([]Seen, error)
	Provenance(ctx context.Context, c Caller, id string) (Provenance, bool, error)
}

// FHIRHandler serves the MedicationStatement read and search by patient of the FHIR API
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

// Routes declares the MedicationStatement routes on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/MedicationStatement/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/MedicationStatement", h.search)
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
	fhir.Write(w, http.StatusOK, toFHIR(a.MedicationStatement, a.Attributed))
}

// search answers MedicationStatement?patient=…, the one search there is.
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
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/MedicationStatement/" + a.ID, Resource: toFHIR(a.MedicationStatement, a.Attributed)})
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(h.base+"/MedicationStatement?"+q.Encode(), matches, nil))
}

// Provenance answers GET /fhir/R4/Provenance/{id} and reports true when the id is a medication's Provenance in the
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
	fhir.Write(w, http.StatusOK, fhir.NewWriteProvenance(fhir.Change{
		Recorded: p.Recorded, ID: p.ID, Resource: "MedicationStatement/" + p.EntryID, Activity: p.Activity, AgentType: p.AgentType,
		Who: "PractitionerRole/" + p.RoleID, OnBehalfOf: "Organization/" + p.OrganizationID, Version: p.Version,
	}))
	return true
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
