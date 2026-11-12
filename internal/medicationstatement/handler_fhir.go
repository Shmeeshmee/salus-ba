package medicationstatement

import (
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
