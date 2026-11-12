package allergyintolerance

import (
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
