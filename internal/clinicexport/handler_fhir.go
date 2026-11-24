package clinicexport

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// collection is a FHIR R4 Bundle of type collection: the resources of a record handed over, each at its urn:uuid.
type collection struct {
	ResourceType string            `json:"resourceType"`
	Type         string            `json:"type"`
	Timestamp    string            `json:"timestamp"`
	Entry        []collectionEntry `json:"entry"`
}

type collectionEntry struct {
	Resource any    `json:"resource"`
	FullURL  string `json:"fullUrl"`
}

// bundleJSON is the record r as a FHIR R4 Bundle of type collection: the Patient, the Consents, the Appointments and
// the receipts as Invoices, timestamped at the latest change among them, as JSON with a final newline. The hand-over's download and the export's
// patients/<id>.json are this same file.
func bundleJSON(r Record) ([]byte, error) { return ownBundleJSON(r, clinical{}) }

// ownBundleJSON is bundleJSON with the patient's own clinical content after the Appointments (D7): the Encounters,
// the signed documents as Compositions and the submitted QuestionnaireResponses, each attachment naming its file in
// the archive (files/<Binary id>.<ext>).
func ownBundleJSON(r Record, c clinical) ([]byte, error) {
	p := r.Patient.Patient
	latest := p.LastUpdated
	b := collection{ResourceType: "Bundle", Type: "collection",
		Entry: []collectionEntry{{FullURL: "urn:uuid:" + p.ID, Resource: patient.FHIR(p)}}}
	add := func(id string, at time.Time, res any) {
		b.Entry = append(b.Entry, collectionEntry{FullURL: "urn:uuid:" + id, Resource: res})
		if at.After(latest) {
			latest = at
		}
	}
	for _, x := range r.Patient.Consents {
		add(x.ID, x.LastUpdated, consent.FHIR(x))
	}
	for _, v := range r.Visits {
		add(v.Appointment.ID, v.Appointment.LastUpdated, appointment.FHIR(v))
	}
	for _, e := range c.encounters {
		add(e.ID, e.LastUpdated, encounter.FHIR(e))
	}
	for _, d := range c.documents {
		add(d.Draft.ID, d.Draft.UpdatedAt, composition.CompositionFHIR(d))
	}
	for _, a := range c.answers {
		names := map[string]string{}
		for _, f := range a.seen.Response.Attachments {
			names[f.BinaryID] = fileName(f.BinaryID, f.ContentType)
		}
		add(a.seen.Response.ID, a.seen.Response.LastUpdated,
			questionnaireresponse.FHIR(a.seen, a.version, func(id string) string { return names[id] }))
	}
	for _, i := range r.Receipts {
		add(i.ID, i.IssuedAt, invoice.FHIR(i))
	}
	b.Timestamp = latest.UTC().Format(time.RFC3339)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf) // as fhir.Write: URLs keep their &
	enc.SetEscapeHTML(false)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
