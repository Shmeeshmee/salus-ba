package diagnosticreport

import (
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// A lab result is not served over FHIR; it is written as a FHIR R4 DiagnosticReport only inside the patient's own
// export (record.json, docs/clinic-cms/WAVE5.md D7), with its values as contained Observations (they have no life of
// their own outside the report). Only what the report holds: no Encounter, no specimen, no PDF yet.

const (
	systemLOINC          = "http://loinc.org"
	systemUCUM           = "http://unitsofmeasure.org"
	systemInterpretation = "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"
)

type fhirReport struct {
	ResourceType      string               `json:"resourceType"`
	ID                string               `json:"id"`
	Meta              fhir.Meta            `json:"meta"`
	Contained         []fhirObservation    `json:"contained,omitempty"`
	Status            Status               `json:"status"`
	Code              fhir.CodeableConcept `json:"code"`
	Subject           fhir.Reference       `json:"subject"`
	EffectiveDateTime string               `json:"effectiveDateTime,omitempty"`
	Issued            *time.Time           `json:"issued,omitempty"`
	Performer         []fhir.Reference     `json:"performer"`
	Result            []fhir.Reference     `json:"result,omitempty"`
}

type fhirObservation struct {
	ValueQuantity  *quantity              `json:"valueQuantity,omitempty"`
	ResourceType   string                 `json:"resourceType"`
	ID             string                 `json:"id"`
	Status         Status                 `json:"status"`
	Code           fhir.CodeableConcept   `json:"code"`
	Subject        fhir.Reference         `json:"subject"`
	ValueString    string                 `json:"valueString,omitempty"`
	Interpretation []fhir.CodeableConcept `json:"interpretation,omitempty"`
	ReferenceRange []rangeText            `json:"referenceRange,omitempty"`
}

type quantity struct {
	Unit   string  `json:"unit,omitempty"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
	Value  float64 `json:"value"`
}

type rangeText struct {
	Text string `json:"text"`
}

// FHIR is the report r as a FHIR R4 DiagnosticReport: its values contained, the clinic its performer. Last updated
// when released (a released report never changes after), else when attached.
func FHIR(r Report) any {
	at := r.CreatedAt
	if r.ReleasedAt != nil {
		at = *r.ReleasedAt
	}
	subject := fhir.Reference{Reference: "Patient/" + r.PatientID}
	out := fhirReport{
		ResourceType: "DiagnosticReport", ID: r.ID, Meta: fhir.NewMeta(int32(r.VersionID), at), Status: r.Status, //nolint:gosec // a version is small
		Code: concept(r.Title, r.LOINC), Subject: subject, Performer: []fhir.Reference{{Reference: "Organization/" + r.OrganizationID}},
	}
	if r.EffectiveDate != nil {
		out.EffectiveDateTime = r.EffectiveDate.Format(time.DateOnly)
	}
	if r.ReleasedAt != nil {
		issued := r.ReleasedAt.UTC()
		out.Issued = &issued
	}
	for i, v := range r.Results {
		id := "result-" + strconv.Itoa(i+1)
		o := fhirObservation{ResourceType: "Observation", ID: id, Status: r.Status, Code: concept(v.Test, v.LOINC), Subject: subject, ValueString: v.ValueText}
		if v.Value != nil {
			o.ValueQuantity = &quantity{Value: *v.Value}
			if v.Unit != "" {
				o.ValueQuantity.Unit, o.ValueQuantity.System, o.ValueQuantity.Code = v.Unit, systemUCUM, v.Unit
			}
		}
		if v.Interpretation != "" {
			o.Interpretation = []fhir.CodeableConcept{{Coding: []fhir.Coding{{System: systemInterpretation, Code: string(v.Interpretation)}}}}
		}
		if v.ReferenceRange != "" {
			o.ReferenceRange = []rangeText{{Text: v.ReferenceRange}}
		}
		out.Contained = append(out.Contained, o)
		out.Result = append(out.Result, fhir.Reference{Reference: "#" + id})
	}
	return out
}

// concept is a name as the lab prints it, with its LOINC coding when known.
func concept(text, code string) fhir.CodeableConcept {
	c := fhir.CodeableConcept{Text: text}
	if code != "" {
		c.Coding = []fhir.Coding{{System: systemLOINC, Code: code}}
	}
	return c
}
