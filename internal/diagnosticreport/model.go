// Package diagnosticreport is the lab results staff attach to a visit (FHIR DiagnosticReport with its Observation
// results and the lab's PDF as a Binary; docs/clinic-cms/WAVE6.md U5). Health data (GDPR Art. 9): every read and write
// is audited, and the patient sees a report only once a practitioner released it.
package diagnosticreport

import "time"

// Status is a report's DiagnosticReport.status (http://hl7.org/fhir/diagnostic-report-status).
type Status string

// The statuses a report takes: attached and not yet released, then released (never changed after).
const (
	StatusPreliminary Status = "preliminary"
	StatusFinal       Status = "final"
)

// Interpretation is a result's flag (http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation).
type Interpretation string

// The flags a result may carry; none is the empty string.
const (
	InterpretationNormal       Interpretation = "N"
	InterpretationLow          Interpretation = "L"
	InterpretationHigh         Interpretation = "H"
	InterpretationCriticalLow  Interpretation = "LL"
	InterpretationCriticalHigh Interpretation = "HH"
	InterpretationAbnormal     Interpretation = "A"
)

// Interpretations are the flags in the contract's order.
var Interpretations = []Interpretation{
	InterpretationNormal, InterpretationLow, InterpretationHigh, InterpretationCriticalLow, InterpretationCriticalHigh,
	InterpretationAbnormal,
}

// Result is one typed-in value (an Observation): a number with its UCUM unit, or a text value.
type Result struct {
	Value          *float64       `json:"value,omitempty"`
	Test           string         `json:"test"`
	LOINC          string         `json:"loinc,omitempty"`
	Unit           string         `json:"unit,omitempty"`      // UCUM, with Value only
	ValueText      string         `json:"valueText,omitempty"` // instead of Value ("negative")
	ReferenceRange string         `json:"referenceRange,omitempty"`
	Interpretation Interpretation `json:"interpretation,omitempty"`
}

// Report is a lab result on a visit.
type Report struct {
	ReleasedAt     *time.Time `json:"releasedAt,omitempty"`
	EffectiveDate  *time.Time `json:"effectiveDate,omitempty"` // the sample's day, as the lab says
	CreatedAt      time.Time  `json:"createdAt"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	PatientID      string     `json:"patientId"`
	AppointmentID  string     `json:"appointmentId"`
	Status         Status     `json:"status"`
	Title          string     `json:"title"`
	LOINC          string     `json:"loinc,omitempty"`
	Results        []Result   `json:"results"`
	VersionID      int        `json:"versionId"`
	HasPDF         bool       `json:"hasPdf"`
}

// Draft is what staff attach or change while a report is not released.
type Draft struct {
	EffectiveDate *time.Time
	Title         string
	LOINC         string
	Results       []Result
	Version       int // 0 when attaching
}

// Staff is the caller of the clinic's routes: its admin or a practitioner; a receptionist is refused (no clinical
// content).
type Staff struct {
	AccountID      string
	OrganizationID string
	Admin          bool
	Practitioner   bool
	Reception      bool
}

// Patient is the caller of the patient's routes.
type Patient struct {
	AccountID string
	PatientID string
}
