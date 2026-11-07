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

// The help text's item control (SPEC.md §2 "Item model"): the R4 core extension and code. An attachment
// item's accepted content types and number of files (docs/attachments/SPEC.md §2 "Item model"): R4 core
// extensions too.
const (
	extensionItemControl = "http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl"
	systemItemControl    = "http://hl7.org/fhir/questionnaire-item-control"
	extensionMimeType    = "http://hl7.org/fhir/StructureDefinition/mimeType"
	extensionMaxOccurs   = "http://hl7.org/fhir/StructureDefinition/questionnaire-maxOccurs"
	// extensionTranslation carries a text's translation into one language (docs/i18n/SPEC.md §3 "FHIR").
	extensionTranslation = "http://hl7.org/fhir/StructureDefinition/translation"
)

// mimeTypes are the content types of each kind of file, as the server detects them.
var mimeTypes = map[FileKind][]string{
	FilePhoto:    {"image/jpeg", "image/png"},
	FileDocument: {"application/pdf"},
	FileVideo:    {"video/mp4"},
}

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

// Resource is a Questionnaire as FHIR R4 JSON. Language is the original language; the _-prefixed elements
// carry the translations of the text beside them.
type Resource struct {
	ResourceType   string     `json:"resourceType"`
	ID             string     `json:"id"`
	Meta           fhir.Meta  `json:"meta"`
	Language       string     `json:"language,omitempty"`
	URL            string     `json:"url"`
	Version        string     `json:"version"`
	Title          string     `json:"title"`
	TitleExt       *element   `json:"_title,omitempty"`
	Status         Status     `json:"status"`
	SubjectType    []string   `json:"subjectType"`
	Date           string     `json:"date"`
	Description    string     `json:"description,omitempty"`
	DescriptionExt *element   `json:"_description,omitempty"`
	Item           []fhirItem `json:"item,omitempty"`
}

// fhirItem is a Questionnaire.item. Help text is a child display item with the help item control.
type fhirItem struct {
	Extension    []extension  `json:"extension,omitempty"`
	LinkID       string       `json:"linkId"`
	Text         string       `json:"text"`
	TextExt      *element     `json:"_text,omitempty"`
	Type         string       `json:"type"`
	EnableWhen   []enableWhen `json:"enableWhen,omitempty"`
	AnswerOption []fhirOption `json:"answerOption,omitempty"`
	Item         []fhirItem   `json:"item,omitempty"`
	Required     bool         `json:"required,omitempty"`
	Repeats      bool         `json:"repeats,omitempty"`
}

// fhirOption is a Questionnaire.item.answerOption with its translations.
type fhirOption struct {
	ValueStringExt *element `json:"_valueString,omitempty"`
	ValueString    string   `json:"valueString"`
}

// element is the extensions of a primitive element in FHIR JSON (_title, _text, …): its translations.
type element struct {
	Extension []translation `json:"extension"`
}

// translation is one translation extension: its parts, lang and content, are extensions too.
type translation struct {
	URL       string            `json:"url"`
	Extension []translationPart `json:"extension"`
}

type translationPart struct {
	URL         string `json:"url"`
	ValueCode   string `json:"valueCode,omitempty"`
	ValueString string `json:"valueString,omitempty"`
}

type enableWhen struct {
	AnswerBoolean *bool  `json:"answerBoolean,omitempty"`
	Question      string `json:"question"`
	Operator      string `json:"operator"`
	AnswerString  string `json:"answerString,omitempty"`
}

// extension is an item's extension: the help item control, an accepted content type or the number of files.
type extension struct {
	URL                  string                `json:"url"`
	ValueCodeableConcept *fhir.CodeableConcept `json:"valueCodeableConcept,omitempty"`
	ValueCode            string                `json:"valueCode,omitempty"`
	ValueInteger         int32                 `json:"valueInteger,omitempty"` // maxFiles, 2 to 10
}

// ToFHIR is the version q as a FHIR Questionnaire, with the translations it carries; a QuestionnaireResponse
// search includes it.
func ToFHIR(q Questionnaire) Resource {
	r := Resource{
		ResourceType: "Questionnaire", ID: q.ID, Meta: fhir.NewMeta(q.VersionID, q.LastUpdated), Language: q.Language,
		URL: CanonicalURL(q.SeriesID), Version: strconv.FormatInt(int64(q.Version), 10), Title: q.Title, Status: q.Status,
		SubjectType: []string{"Patient"}, Date: q.LastUpdated.UTC().Format(time.RFC3339), Description: q.Description,
		TitleExt: translations(q.Translations, func(t Translation) string { return t.Title }),
		Item:     toFHIRItems(q.Items, q.Translations),
	}
	if q.Description != "" {
		r.DescriptionExt = translations(q.Translations, func(t Translation) string { return t.Description })
	}
	return r
}

// translations is the translation extensions of one text, which text gives from each translation: one per
// translation that has it, in their order; nil when none has.
func translations(ts []Translation, text func(Translation) string) *element {
	var out []translation
	for _, t := range ts {
		if s := text(t); !blank(s) {
			out = append(out, translation{URL: extensionTranslation, Extension: []translationPart{
				{URL: "lang", ValueCode: t.Language}, {URL: "content", ValueString: s},
			}})
		}
	}
	if out == nil {
		return nil
	}
	return &element{Extension: out}
}

// answerOptions are the item's answer options with their translations.
func answerOptions(it Item, ts []Translation) []fhirOption {
	if len(it.Options) == 0 {
		return nil
	}
	out := make([]fhirOption, 0, len(it.Options))
	for i, o := range it.Options {
		out = append(out, fhirOption{ValueString: o.Value, ValueStringExt: translations(ts, func(t Translation) string {
			if opts := t.Items[it.LinkID].Options; i < len(opts) {
				return opts[i]
			}
			return ""
		})})
	}
	return out
}

func toFHIRItems(items []Item, ts []Translation) []fhirItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]fhirItem, 0, len(items))
	for _, it := range items {
		f := fhirItem{
			LinkID: it.LinkID, Text: it.Text, Type: string(it.Type), AnswerOption: answerOptions(it, ts), Item: toFHIRItems(it.Items, ts),
			Required: it.Required, Repeats: it.Repeats,
			TextExt: translations(ts, func(t Translation) string { return t.Items[it.LinkID].Text }),
		}
		if it.Type == ItemAttachment {
			f.Extension, f.Repeats = attachmentExtensions(it), it.MaxFiles > 1
		}
		if c := it.EnableWhen; c != nil {
			f.EnableWhen = []enableWhen{{Question: c.Question, Operator: "=", AnswerBoolean: c.Boolean, AnswerString: c.String}}
		}
		if it.Help != "" {
			// "." is never in a linkId of ours, so the help item's cannot clash with another.
			help := translations(ts, func(t Translation) string { return t.Items[it.LinkID].Help })
			f.Item = append(f.Item, fhirItem{LinkID: it.LinkID + ".help", Text: it.Help, TextExt: help, Type: string(ItemDisplay), Extension: []extension{{
				URL:                  extensionItemControl,
				ValueCodeableConcept: &fhir.CodeableConcept{Coding: []fhir.Coding{{System: systemItemControl, Code: "help", Display: "Help-Button"}}},
			}}})
		}
		out = append(out, f)
	}
	return out
}

// attachmentExtensions are a mimeType extension per content type the item accepts, then its maxOccurs. R4
// allows maxOccurs only on an item that repeats, with 2 or more: one file is said by repeats absent.
func attachmentExtensions(it Item) []extension {
	var out []extension
	for _, k := range it.Accept {
		for _, m := range mimeTypes[k] {
			out = append(out, extension{URL: extensionMimeType, ValueCode: m})
		}
	}
	if it.MaxFiles > 1 {
		out = append(out, extension{URL: extensionMaxOccurs, ValueInteger: it.MaxFiles})
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
