package questionnaireresponse

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The appointment extension (api/fhir/StructureDefinition-questionnaireresponse-appointment.json) and the one
// _include this server supports.
const (
	extensionAppointment  = "https://salus.invalid/fhir/StructureDefinition/questionnaireresponse-appointment"
	includeQuestionnaire  = "QuestionnaireResponse:questionnaire"
	unsupportedSearchText = "Search by appointment, patient and status, each once, optionally with _include=QuestionnaireResponse:questionnaire."
)

// Reader reads and searches responses for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c appointment.Caller, id string) (Seen, error)
	Search(ctx context.Context, c appointment.Caller, f Filter) ([]Seen, error)
}

// Versions reads a questionnaire version for a caller, audited as their read of it (questionnaire.Service):
// staff of the clinic any of its versions, a patient those their own responses reference.
type Versions interface {
	Read(ctx context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error)
}

// Handler serves the QuestionnaireResponse read and search of the FHIR API (api/fhir/CapabilityStatement.json).
// There is no create or update: requests and answers go through the JSON API. Each response is written against
// the version it answers, read once per request: its canonical url and version, and its sections to nest the
// answers in.
type Handler struct {
	svc      Reader
	versions Versions
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
	api    string // the api's public base: base without /fhir/R4, for the files' download URLs
}

// NewHandler returns the handler on svc, reading the versions from versions. base is the FHIR API's absolute
// base URL (https://…/fhir/R4); the api's own base, for the download URL of a valueAttachment, is base without
// /fhir/R4 (config.Config.APIBaseURL).
func NewHandler(svc Reader, versions Versions, caller func(ctx context.Context) (appointment.Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, versions: versions, caller: caller, base: base, api: strings.TrimSuffix(base, "/fhir/R4"), log: log}
}

// Routes declares the QuestionnaireResponse routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/QuestionnaireResponse/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/QuestionnaireResponse", h.search)
}

// resource is a QuestionnaireResponse as FHIR R4 JSON. A Name view carries no item.
type resource struct {
	ResourceType  string         `json:"resourceType"`
	ID            string         `json:"id"`
	Meta          fhir.Meta      `json:"meta"`
	Extension     []extension    `json:"extension"`
	Questionnaire string         `json:"questionnaire"`
	Status        Status         `json:"status"`
	Subject       fhir.Reference `json:"subject"`
	Authored      string         `json:"authored,omitempty"`
	Author        fhir.Reference `json:"author"`
	Item          []item         `json:"item,omitempty"`
}

type extension struct {
	URL            string         `json:"url"`
	ValueReference fhir.Reference `json:"valueReference"`
}

// item is a QuestionnaireResponse.item: a section with its answered items, or a question with its answers.
type item struct {
	LinkID string   `json:"linkId"`
	Answer []answer `json:"answer,omitempty"`
	Item   []item   `json:"item,omitempty"`
}

// answer is one QuestionnaireResponse.item.answer: a Value (its JSON is FHIR's value[x]) or a file.
type answer struct {
	Attachment *attachment `json:"valueAttachment,omitempty"`
	Value
}

// attachment is an R4 Attachment (docs/attachments/SPEC.md §2 "FHIR"): url is the api's download endpoint,
// never the bucket; hash the SHA-1, base64.
type attachment struct {
	ContentType string `json:"contentType"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Hash        []byte `json:"hash"`
	Size        int64  `json:"size"`
}

func (h *Handler) toFHIR(s Seen, v questionnaire.Questionnaire) resource {
	r := s.Response
	patient := fhir.Reference{Reference: "Patient/" + r.PatientID}
	res := resource{
		ResourceType: "QuestionnaireResponse", ID: r.ID, Meta: fhir.NewMeta(r.VersionID, r.LastUpdated),
		Extension:     []extension{{URL: extensionAppointment, ValueReference: fhir.Reference{Reference: "Appointment/" + r.AppointmentID}}},
		Questionnaire: questionnaire.CanonicalURL(v.SeriesID) + "|" + strconv.FormatInt(int64(v.Version), 10),
		Status:        r.Status, Subject: patient, Author: patient, Authored: instant(r.AuthoredAt),
	}
	if s.Level == access.Full {
		given := make(map[string][]answer, len(r.Answers)+len(r.Attachments))
		for _, a := range r.Answers {
			for _, value := range a.Values {
				given[a.LinkID] = append(given[a.LinkID], answer{Value: value})
			}
		}
		for _, f := range r.Attachments { // in upload order, a pending one too: only the patient sees one
			given[f.LinkID] = append(given[f.LinkID], answer{Attachment: &attachment{
				ContentType: f.ContentType, URL: h.api + "/binaries/" + f.BinaryID + "/content", Size: f.Size, Hash: f.Hash, Title: f.Title,
			}})
		}
		res.Item = nest(v.Items, given)
	}
	return res
}

// nest gives the answered items of items in document order, a section with its answered items; a section with
// none, an instruction and an unanswered question are left out.
func nest(items []questionnaire.Item, given map[string][]answer) []item {
	var out []item
	for _, it := range items {
		switch {
		case it.Type == questionnaire.ItemGroup:
			if inner := nest(it.Items, given); len(inner) > 0 {
				out = append(out, item{LinkID: it.LinkID, Item: inner})
			}
		case len(given[it.LinkID]) > 0:
			out = append(out, item{LinkID: it.LinkID, Answer: given[it.LinkID]})
		}
	}
	return out
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
		return
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return
	}
	known := map[string]questionnaire.Questionnaire{}
	if err := h.readVersions(r.Context(), c, []Seen{s}, known); err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(s.Response.VersionID), 10)))
	fhir.Write(w, http.StatusOK, h.toFHIR(s, known[s.Response.QuestionnaireID]))
}

// search answers QuestionnaireResponse?[appointment=…][&patient=…][&status=…]
// [&_include=QuestionnaireResponse:questionnaire].
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, f, include, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported", unsupportedSearchText)
		return
	}
	all, err := h.svc.Search(r.Context(), c, f)
	if err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	known := map[string]questionnaire.Questionnaire{}
	if err := h.readVersions(r.Context(), c, all, known); err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	var included []fhir.BundleEntry
	once := map[string]bool{}
	for _, s := range all {
		v := known[s.Response.QuestionnaireID]
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/QuestionnaireResponse/" + s.Response.ID, Resource: h.toFHIR(s, v)})
		if include && !once[v.ID] {
			once[v.ID] = true
			included = append(included, fhir.BundleEntry{FullURL: h.base + "/Questionnaire/" + v.ID, Resource: questionnaire.ToFHIR(v)})
		}
	}
	self := h.base + "/QuestionnaireResponse"
	if len(q) > 0 {
		self += "?" + q.Encode()
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(self, matches, included))
}

// readVersions reads, as c, each version the responses answer that known does not hold yet, once, into known.
func (h *Handler) readVersions(ctx context.Context, c appointment.Caller, all []Seen, known map[string]questionnaire.Questionnaire) error {
	qc := questionnaire.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin}
	for _, s := range all {
		id := s.Response.QuestionnaireID
		if _, ok := known[id]; ok {
			continue
		}
		v, err := h.versions.Read(ctx, qc, id)
		if err != nil {
			return err // a version a response the caller sees answers is always there for them
		}
		known[id] = v
	}
	return nil
}

// searchQuery reads the search into a filter and whether it includes the versions, and false for an unknown or
// repeated parameter or a value it cannot read.
func searchQuery(raw string) (q url.Values, f Filter, include, ok bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return nil, Filter{}, false, false
	}
	for key, values := range q {
		v, ok := values[0], false
		switch key {
		case "appointment":
			f.AppointmentID, ok = fhir.ReferenceID(v, "Appointment")
		case "patient":
			f.PatientID, ok = fhir.ReferenceID(v, "Patient")
		case "status":
			f.Status = Status(v)
			ok = f.Status == StatusInProgress || f.Status == StatusCompleted || f.Status == StatusStopped
		case "_include":
			include, ok = v == includeQuestionnaire, v == includeQuestionnaire
		}
		if !ok || len(values) != 1 {
			return nil, Filter{}, false, false
		}
	}
	return q, f, include, true
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	case !ok:
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok && err == nil
}
