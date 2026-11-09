package composition

import (
	"context"
	"encoding/base64"
	"errors"
	"html"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The code systems of the FHIR resources.
const (
	systemListEmptyReason   = "http://terminology.hl7.org/CodeSystem/list-empty-reason"
	systemDocumentCompl     = "http://terminology.hl7.org/CodeSystem/v3-DocumentCompletion"
	systemProvenanceAgent   = "http://terminology.hl7.org/CodeSystem/provenance-participant-type"
	xhtmlNamespace          = "http://www.w3.org/1999/xhtml"
	attachmentContentType   = "application/pdf"
	documentReferenceStatus = "current"
)

// Reader reads a document's resources for a caller (Service).
type Reader interface {
	Composition(ctx context.Context, c appointment.Caller, id string) (Document, error)
	DocumentReference(ctx context.Context, c appointment.Caller, id string) (Document, error)
	Provenance(ctx context.Context, c appointment.Caller, id string) (Document, error)
}

// FHIRHandler serves the Composition, DocumentReference and Provenance reads of the FHIR API
// (api/fhir/CapabilityStatement.json). No create or update: drafts are written and signed through the JSON API.
type FHIRHandler struct {
	svc Reader
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
	api    string // the API's absolute base URL: the FHIR base without /fhir/R4, for the PDF's url
}

// NewFHIRHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewFHIRHandler(svc Reader, caller func(ctx context.Context) (appointment.Caller, bool, error), base string, log *slog.Logger) *FHIRHandler {
	return &FHIRHandler{svc: svc, caller: caller, api: strings.TrimSuffix(base, "/fhir/R4"), log: log}
}

// Routes declares the routes on mux.
func (h *FHIRHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Composition/{id}", h.read(h.svc.Composition, compositionOf))
	mux.HandleFunc("GET /fhir/R4/DocumentReference/{id}", h.read(h.svc.DocumentReference, h.documentReferenceOf))
	mux.HandleFunc("GET /fhir/R4/Provenance/{id}", h.read(h.svc.Provenance, provenanceOf))
}

type compositionJSON struct {
	ResourceType string               `json:"resourceType"`
	ID           string               `json:"id"`
	Meta         fhir.Meta            `json:"meta"`
	Language     string               `json:"language"`
	Status       Status               `json:"status"`
	Type         fhir.CodeableConcept `json:"type"`
	Subject      fhir.Reference       `json:"subject"`
	Encounter    fhir.Reference       `json:"encounter"`
	Date         string               `json:"date"`
	Author       []fhir.Reference     `json:"author"`
	Title        string               `json:"title"`
	Attester     []attester           `json:"attester,omitempty"`
	Custodian    fhir.Reference       `json:"custodian"`
	Section      []section            `json:"section"`
}

type attester struct {
	Mode  string         `json:"mode"`
	Time  string         `json:"time"`
	Party fhir.Reference `json:"party"`
}

type section struct {
	Title       string                `json:"title"`
	Text        *narrative            `json:"text,omitempty"`
	EmptyReason *fhir.CodeableConcept `json:"emptyReason,omitempty"`
}

type narrative struct {
	Status string `json:"status"`
	Div    string `json:"div"`
}

func instant(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func documentType(code string) fhir.CodeableConcept {
	return fhir.CodeableConcept{Coding: []fhir.Coding{{System: SystemLOINC, Code: code}}}
}

// compositionOf is the draft as an R4 Composition: each field a section, its value the narrative (a list as
// items, a coded field's ICD-10 code as text), a missing one with an emptyReason. A final one is attested by
// its signer.
func compositionOf(doc Document) (any, int32) {
	d, e := doc.Draft, doc.Encounter
	author, date := e.PractitionerRoleID, d.UpdatedAt
	out := compositionJSON{
		ResourceType: "Composition", ID: d.ID, Meta: fhir.NewMeta(d.VersionID, d.UpdatedAt), Language: d.Language, Status: d.Status,
		Type: documentType(d.TypeCode), Subject: fhir.Reference{Reference: "Patient/" + e.PatientID},
		Encounter: fhir.Reference{Reference: "Encounter/" + d.EncounterID}, Title: d.Title,
		Custodian: fhir.Reference{Reference: "Organization/" + e.OrganizationID},
	}
	if d.Status == StatusFinal {
		author, date = d.SignerRoleID, d.SignedAt
		out.Attester = []attester{{Mode: "legal", Time: instant(d.SignedAt), Party: fhir.Reference{Reference: "PractitionerRole/" + d.SignerRoleID}}}
	}
	out.Author, out.Date = []fhir.Reference{{Reference: "PractitionerRole/" + author}}, instant(date)
	for _, f := range d.Fields {
		s := section{Title: f.Label}
		s.Text = &narrative{Status: "generated", Div: div(f)}
		if f.Missing { // cmp-1 wants text even then: a dash
			s.Text.Status = "empty"
			s.EmptyReason = &fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemListEmptyReason, Code: "unavailable"}}}
		}
		out.Section = append(out.Section, s)
	}
	return out, d.VersionID
}

// div is the field's value as XHTML: a list as items, other values a paragraph per line, a missing one a dash.
func div(f DraftField) string {
	lines, tag, open, end := strings.Split(f.Value, "\n"), "p", "", ""
	switch {
	case f.Missing:
		lines = []string{"—"}
	case f.Type == FieldList:
		tag, open, end = "li", "<ul>", "</ul>"
	}
	var b strings.Builder
	b.WriteString(`<div xmlns="` + xhtmlNamespace + `">` + open)
	for _, line := range lines {
		b.WriteString("<" + tag + ">" + html.EscapeString(line) + "</" + tag + ">")
	}
	b.WriteString(end + "</div>")
	return b.String()
}

type documentReferenceJSON struct {
	ResourceType  string               `json:"resourceType"`
	ID            string               `json:"id"`
	Meta          fhir.Meta            `json:"meta"`
	Status        string               `json:"status"`
	DocStatus     string               `json:"docStatus"`
	Type          fhir.CodeableConcept `json:"type"`
	Subject       fhir.Reference       `json:"subject"`
	Date          string               `json:"date"`
	Author        []fhir.Reference     `json:"author"`
	Authenticator fhir.Reference       `json:"authenticator"`
	Custodian     fhir.Reference       `json:"custodian"`
	Content       []content            `json:"content"`
	Context       docContext           `json:"context"`
}

type content struct {
	Attachment attachment `json:"attachment"`
}

type attachment struct {
	ContentType string `json:"contentType"`
	Language    string `json:"language"`
	URL         string `json:"url"`
	Hash        string `json:"hash"`
	Title       string `json:"title"`
	Creation    string `json:"creation"`
	Size        int64  `json:"size"`
}

type docContext struct {
	Encounter []fhir.Reference `json:"encounter"`
	Related   []fhir.Reference `json:"related"`
}

// documentReferenceOf is the signed draft's PDF as an R4 DocumentReference, written once (version 1).
func (h *FHIRHandler) documentReferenceOf(doc Document) (any, int32) {
	d, e := doc.Draft, doc.Encounter
	signer := fhir.Reference{Reference: "PractitionerRole/" + d.SignerRoleID}
	return documentReferenceJSON{
		ResourceType: "DocumentReference", ID: d.DocumentReferenceID, Meta: fhir.NewMeta(1, d.SignedAt),
		Status: documentReferenceStatus, DocStatus: string(StatusFinal), Type: documentType(d.TypeCode),
		Subject: fhir.Reference{Reference: "Patient/" + e.PatientID}, Date: instant(d.SignedAt),
		Author: []fhir.Reference{signer}, Authenticator: signer, Custodian: fhir.Reference{Reference: "Organization/" + e.OrganizationID},
		Content: []content{{Attachment: attachment{
			ContentType: attachmentContentType, Language: d.Language, URL: h.api + "/document-references/" + d.DocumentReferenceID + "/content",
			Size: d.PDF.Size, Hash: base64.StdEncoding.EncodeToString(d.PDF.SHA1), Title: d.Title, Creation: instant(d.SignedAt),
		}}},
		Context: docContext{
			Encounter: []fhir.Reference{{Reference: "Encounter/" + d.EncounterID}},
			Related:   []fhir.Reference{{Reference: "Composition/" + d.ID}},
		},
	}, 1
}

type provenanceJSON struct {
	ResourceType string               `json:"resourceType"`
	ID           string               `json:"id"`
	Meta         fhir.Meta            `json:"meta"`
	Target       []fhir.Reference     `json:"target"`
	Recorded     string               `json:"recorded"`
	Activity     fhir.CodeableConcept `json:"activity"`
	Agent        []agent              `json:"agent"`
}

type agent struct {
	Type fhir.CodeableConcept `json:"type"`
	Who  fhir.Reference       `json:"who"`
}

// provenanceOf is the signature as an R4 Provenance: the signer legally authenticated the Composition's signed
// version and its DocumentReference, written once (version 1).
func provenanceOf(doc Document) (any, int32) {
	d := doc.Draft
	return provenanceJSON{
		ResourceType: "Provenance", ID: d.ProvenanceID, Meta: fhir.NewMeta(1, d.SignedAt),
		Target: []fhir.Reference{
			{Reference: "Composition/" + d.ID + "/_history/" + strconv.FormatInt(int64(d.VersionID), 10)},
			{Reference: "DocumentReference/" + d.DocumentReferenceID},
		},
		Recorded: instant(d.SignedAt),
		Activity: fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemDocumentCompl, Code: "LA", Display: "legally authenticated"}}},
		Agent: []agent{{
			Type: fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemProvenanceAgent, Code: "author", Display: "Author"}}},
			Who:  fhir.Reference{Reference: "PractitionerRole/" + d.SignerRoleID},
		}},
	}, 1
}

// read serves a read of a resource found by get and mapped by to.
func (h *FHIRHandler) read(get func(ctx context.Context, c appointment.Caller, id string) (Document, error), to func(Document) (any, int32)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok, err := h.caller(r.Context())
		switch {
		case err != nil:
			fhir.InternalError(w, r, h.log, err)
			return
		case !ok:
			fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
			return
		}
		doc, err := get(r.Context(), c, r.PathValue("id"))
		switch {
		case errors.Is(err, ErrNotFound):
			fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
			return
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrConsentRequired):
			fhir.WriteError(w, http.StatusForbidden, "forbidden", "Only the encounter's practitioner reads its documents, while the patient's consent is active.")
			return
		case err != nil:
			fhir.InternalError(w, r, h.log, err)
			return
		}
		resource, version := to(doc)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(version), 10)))
		fhir.Write(w, http.StatusOK, resource)
	}
}
