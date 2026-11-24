package invoice

import (
	"encoding/json"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// A receipt is not served over FHIR; it is written as a FHIR R4 Invoice only inside the record of a patient the
// clinic registered, which the clinic hands over or exports (docs/clinic-cms wave 4). Only what that record needs:
// the number, whether it stands, the patient, the clinic, the date and the total. Its lines and the fiscal data stay
// on the receipt's PDF.

type resource struct {
	ResourceType string          `json:"resourceType"`
	ID           string          `json:"id"`
	Status       string          `json:"status"`
	Date         string          `json:"date"`
	Meta         fhir.Meta       `json:"meta"`
	Subject      *fhir.Reference `json:"subject,omitempty"`
	Issuer       fhir.Reference  `json:"issuer"`
	TotalGross   money           `json:"totalGross"`
	Identifier   []identifier    `json:"identifier"`
}

type identifier struct {
	Value string `json:"value"`
}

type money struct {
	Value    json.Number `json:"value"`
	Currency string      `json:"currency"`
}

// FHIR is the receipt i as a FHIR R4 Invoice: issued, or cancelled once a reversal reverses it. A receipt never
// changes once issued, so it is at version 1, last updated when it was issued.
func FHIR(i Invoice) any {
	res := resource{
		ResourceType: "Invoice", ID: i.ID, Status: "issued", Date: i.IssuedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Meta: fhir.NewMeta(1, i.IssuedAt), Issuer: fhir.Reference{Reference: "Organization/" + i.OrganizationID},
		TotalGross: money{Value: decimal(i.TotalMinor), Currency: i.Currency},
		Identifier: []identifier{{Value: strconv.FormatInt(i.Number, 10)}},
	}
	if i.ReversedByID != "" {
		res.Status = "cancelled"
	}
	if i.PatientID != "" {
		res.Subject = &fhir.Reference{Reference: "Patient/" + i.PatientID}
	}
	return res
}

// decimal is an amount in minor units (two decimals, as BAM and EUR have) as a FHIR decimal.
func decimal(minor int64) json.Number {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	cents := strconv.FormatInt(minor%100, 10)
	if len(cents) == 1 {
		cents = "0" + cents
	}
	return json.Number(sign + strconv.FormatInt(minor/100, 10) + "." + cents)
}
