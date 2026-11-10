//go:build e2e

package e2e_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// patientOpensTheSignedDocument is the patient's side of a signed document (SAL-17, docs/consultation/SPEC.md
// "Patient documents"), as the mobile app reads it: the e-mail that it is ready names the clinic and no document;
// the encounter's search finds the one signed document with the template's title; its Composition is final; its
// PDF downloads.
func patientOpensTheSignedDocument(t *testing.T, clinic *bookableClinic, p *patient, appointmentID, title string) {
	t.Helper()
	text := mailText(t, p.addr, "A document from your appointment is ready")
	if !strings.Contains(text, clinic.name) || !strings.Contains(text, "/appointments") || strings.Contains(text, title) {
		t.Errorf("the document e-mail does not name the clinic, link the app, or names the document:\n%s", text)
	}
	var found struct {
		Entry []struct {
			Resource struct {
				ID      string
				Content []struct{ Attachment struct{ Title, URL string } }
				Context struct{ Related []struct{ Reference string } }
			}
		}
	}
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/DocumentReference?encounter="+encounterOf(t, p, appointmentID).ID, nil, &found)
	if len(found.Entry) != 1 || len(found.Entry[0].Resource.Content) != 1 || found.Entry[0].Resource.Content[0].Attachment.Title != title ||
		len(found.Entry[0].Resource.Context.Related) != 1 {
		t.Fatalf("the patient's documents = %+v, want the one signed document %q", found, title)
	}
	doc := found.Entry[0].Resource
	var composition struct{ Status string }
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/"+doc.Context.Related[0].Reference, nil, &composition)
	if composition.Status != "final" {
		t.Errorf("the patient's Composition is %q, want final", composition.Status)
	}
	status, pdf := p.send(nil, http.MethodGet, "/document-references/"+doc.ID+"/content", nil)
	if status != http.StatusOK || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Errorf("the patient's PDF = %d, %d bytes", status, len(pdf))
	}
}
