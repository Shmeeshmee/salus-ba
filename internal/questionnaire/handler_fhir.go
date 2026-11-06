package questionnaire

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The help text's item control (SPEC.md §2 "Item model"): the R4 core extension and code.
const (
	extensionItemControl = "http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl"
	systemItemControl    = "http://hl7.org/fhir/questionnaire-item-control"
)

// Reader reads and searches a clinic's questionnaires for a caller (Service).
type Reader interface {
	Read(ctx context.Context, c Caller, id string) (Questionnaire, error)
	Search(ctx context.Context, c Caller, f Filter) ([]Questionnaire, error)
}

// Handler serves the Questionnaire read and search of the FHIR API (api/fhir/CapabilityStatement.json), to
// staff of the clinic. There is no create or update: writes go through the JSON API.
type Handler struct {
	svc Reader
	// caller is who is signed in for the request, with their clinic role, and false for nobody (internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4).
func NewHandler(svc Reader, caller func(ctx context.Context) (Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the Questionnaire routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Questionnaire/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/Questionnaire", h.search)
}

// Resource is a Questionnaire as FHIR R4 JSON.
type Resource struct {
	ResourceType string     `json:"resourceType"`
	ID           string     `json:"id"`
	Meta         fhir.Meta  `json:"meta"`
	URL          string     `json:"url"`
	Version      string     `json:"version"`
	Title        string     `json:"title"`
	Status       Status     `json:"status"`
	SubjectType  []string   `json:"subjectType"`
	Date         string     `json:"date"`
	Description  string     `json:"description,omitempty"`
	Item         []fhirItem `json:"item,omitempty"`
}

// fhirItem is a Questionnaire.item. Help text is a child display item with the help item control.
type fhirItem struct {
	Extension    []itemControl `json:"extension,omitempty"`
	LinkID       string        `json:"linkId"`
	Text         string        `json:"text"`
	Type         string        `json:"type"`
	EnableWhen   []enableWhen  `json:"enableWhen,omitempty"`
	AnswerOption []Option      `json:"answerOption,omitempty"`
	Item         []fhirItem    `json:"item,omitempty"`
	Required     bool          `json:"required,omitempty"`
	Repeats      bool          `json:"repeats,omitempty"`
}

type enableWhen struct {
	AnswerBoolean *bool  `json:"answerBoolean,omitempty"`
	Question      string `json:"question"`
	Operator      string `json:"operator"`
	AnswerString  string `json:"answerString,omitempty"`
}

type itemControl struct {
	URL                  string               `json:"url"`
	ValueCodeableConcept fhir.CodeableConcept `json:"valueCodeableConcept"`
}

// ToFHIR is the version q as a FHIR Questionnaire; a QuestionnaireResponse search includes it.
func ToFHIR(q Questionnaire) Resource {
	return Resource{
		ResourceType: "Questionnaire", ID: q.ID, Meta: fhir.NewMeta(q.VersionID, q.LastUpdated),
		URL: CanonicalURL(q.SeriesID), Version: strconv.FormatInt(int64(q.Version), 10), Title: q.Title, Status: q.Status,
		SubjectType: []string{"Patient"}, Date: q.LastUpdated.UTC().Format(time.RFC3339), Description: q.Description,
		Item: toFHIRItems(q.Items),
	}
}

func toFHIRItems(items []Item) []fhirItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]fhirItem, 0, len(items))
	for _, it := range items {
		f := fhirItem{
			LinkID: it.LinkID, Text: it.Text, Type: string(it.Type), AnswerOption: it.Options, Item: toFHIRItems(it.Items),
			Required: it.Required, Repeats: it.Repeats,
		}
		if c := it.EnableWhen; c != nil {
			f.EnableWhen = []enableWhen{{Question: c.Question, Operator: "=", AnswerBoolean: c.Boolean, AnswerString: c.String}}
		}
		if it.Help != "" {
			// "." is never in a linkId of ours, so the help item's cannot clash with another.
			f.Item = append(f.Item, fhirItem{LinkID: it.LinkID + ".help", Text: it.Help, Type: string(ItemDisplay), Extension: []itemControl{{
				URL:                  extensionItemControl,
				ValueCodeableConcept: fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemItemControl, Code: "help", Display: "Help-Button"}}},
			}}})
		}
		out = append(out, f)
	}
	return out
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden): // a patient reads none of them yet
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(q.VersionID), 10)))
		fhir.Write(w, http.StatusOK, ToFHIR(q))
	}
}

// search answers Questionnaire?[url=…][&version=…][&status=…].
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	q, f, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported", "Search by url (a questionnaire of this server), version and status, each once.")
		return
	}
	all, err := h.svc.Search(r.Context(), c, f)
	switch {
	case errors.Is(err, ErrForbidden):
		fhir.WriteError(w, http.StatusForbidden, "forbidden", "Questionnaires are searched by staff of the clinic.")
		return
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	for _, v := range all {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Questionnaire/" + v.ID, Resource: ToFHIR(v)})
	}
	self := h.base + "/Questionnaire"
	if len(q) > 0 {
		self += "?" + q.Encode()
	}
	w.Header().Set("Cache-Control", "no-store")
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(self, matches, nil))
}

// searchQuery reads the search into a filter, and false for an unknown or repeated parameter or a value it
// cannot read.
func searchQuery(raw string) (url.Values, Filter, bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return nil, Filter{}, false
	}
	var f Filter
	for key, values := range q {
		if len(values) != 1 {
			return nil, Filter{}, false
		}
		v, ok := values[0], false
		switch key {
		case "url":
			f.SeriesID, ok = strings.CutPrefix(v, canonicalBase)
			ok = ok && isUUID(f.SeriesID)
		case "version":
			n, err := strconv.ParseInt(v, 10, 32)
			f.Version, ok = int32(n), err == nil && n > 0
		case "status":
			f.Status = Status(v)
			ok = f.Status == StatusDraft || f.Status == StatusActive || f.Status == StatusRetired
		}
		if !ok {
			return nil, Filter{}, false
		}
	}
	return q, f, true
}

// isUUID tells whether s is a UUID in its 8-4-4-4-12 hexadecimal form, as every id of this server is.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F'):
			return false
		}
	}
	return true
}

// signedIn returns the caller, or answers 401 (nobody signed in) or 500 and false.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	case !ok:
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok && err == nil
}
