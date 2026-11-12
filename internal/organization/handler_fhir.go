package organization

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

const maxBody = 64 << 10 // bytes of an update's body

// The extensions of an Organization: whether it is listed in the public directory, its IANA time zone, its
// default language and whether its patients get a copy of their transcripts
// (api/fhir/StructureDefinition-organization-listed.json, -organization-timezone.json,
// -organization-default-language.json, -organization-patient-transcript-copy.json).
const (
	extensionListed          = "https://salus.invalid/fhir/StructureDefinition/organization-listed"
	extensionTimeZone        = "https://salus.invalid/fhir/StructureDefinition/organization-timezone"
	extensionDefaultLanguage = "https://salus.invalid/fhir/StructureDefinition/organization-default-language"
	extensionTranscriptCopy  = "https://salus.invalid/fhir/StructureDefinition/organization-patient-transcript-copy"
	// The clinic's contribution to the platform figures (docs/plans/analytics.md O6), and the city of its address
	// as a Wikidata id (a city's name is not unique): -organization-analytics-contributes.json, -organization-city.json.
	extensionContributes = "https://salus.invalid/fhir/StructureDefinition/organization-analytics-contributes"
	extensionCity        = "https://salus.invalid/fhir/StructureDefinition/organization-city"
)

// Clinics reads and changes a clinic for a caller, and lists the listed ones for anyone (Service).
type Clinics interface {
	Read(ctx context.Context, c Caller, id string) (Organization, error)
	Update(ctx context.Context, c Caller, id string, ch Changes, version int32) (Organization, error)
	ListListed(ctx context.Context) ([]Organization, error)
}

// Handler serves the Organization read, update and search of the FHIR API (api/fhir/CapabilityStatement.json).
type Handler struct {
	svc Clinics
	// caller is who is signed in for the request, and false for nobody (from the session and the account's
	// role, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL (FHIR_BASE_URL)
}

// NewHandler returns the handler on svc. base is the FHIR API's absolute base URL (https://…/fhir/R4), for a
// Bundle's links.
func NewHandler(svc Clinics, caller func(ctx context.Context) (Caller, bool, error), base string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, base: base, log: log}
}

// Routes declares the Organization routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Organization/{id}", h.read)
	mux.HandleFunc("PUT /fhir/R4/Organization/{id}", h.update)
	mux.HandleFunc("GET /fhir/R4/Organization", h.search)
}

// resource is an Organization as FHIR R4 JSON: only the elements the accounts SPEC "Directory model" and the
// booking SPEC "Listed in FHIR" name.
type resource struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Meta         fhir.Meta        `json:"meta"`
	Extension    []fhir.Extension `json:"extension"`
	Address      []address        `json:"address,omitempty"`
	Active       bool             `json:"active"`
}

// address is the clinic's location (docs/plans/analytics.md O1): its leaf region's ISO 3166-2 code as the state,
// its city's name in Latin script, and the city's Wikidata id in the organization-city extension.
type address struct {
	City      string           `json:"city,omitempty"`
	State     string           `json:"state"`
	Extension []fhir.Extension `json:"extension"`
}

// toFHIR is the clinic as its staff read it; toDirectory as the public directory shows it, without its location
// and its contribution to the platform figures.
func toFHIR(o Organization) resource {
	r := toDirectory(o)
	r.Extension = append(r.Extension, fhir.Extension{URL: extensionContributes, ValueBoolean: &o.AnalyticsContributes})
	if o.City != "" {
		r.Address = []address{{State: o.Region, City: o.CityName, Extension: []fhir.Extension{{URL: extensionCity, ValueCode: o.City}}}}
	}
	return r
}

func toDirectory(o Organization) resource {
	return resource{
		ResourceType: "Organization", ID: o.ID, Name: o.Name, Active: o.Active,
		Meta: fhir.NewMeta(o.VersionID, o.LastUpdated),
		Extension: []fhir.Extension{
			{URL: extensionListed, ValueBoolean: &o.Listed},
			{URL: extensionTimeZone, ValueCode: o.TimeZone},
			{URL: extensionDefaultLanguage, ValueCode: o.DefaultLanguage},
			{URL: extensionTranscriptCopy, ValueBoolean: &o.PatientTranscriptCopy},
		},
	}
}

// search answers Organization?listed=true, the public directory, for anyone, signed in or not.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 1 || len(q["listed"]) != 1 || q.Get("listed") != "true" {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported", "Search with listed=true only.")
		return
	}
	all, err := h.svc.ListListed(r.Context())
	if err != nil {
		fhir.InternalError(w, r, h.log, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(all))
	for _, o := range all {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/Organization/" + o.ID, Resource: toDirectory(o)})
	}
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(h.base+"/Organization?listed=true", matches, nil))
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	o, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	h.write(w, r, o, err)
}

// update changes the clinic: the body is the Organization with its name and, when given, the listed,
// time-zone, default-language and patient-transcript-copy extensions; If-Match the version it was read at. An extension left out keeps its value; the
// other elements of the body are ignored.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(r)
	if !ok {
		fhir.WriteError(w, http.StatusPreconditionFailed, "conflict", "Send If-Match with the version you read.")
		return
	}
	var body updateBody
	id := r.PathValue("id")
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&body)
	ch, valid := body.changes()
	if err != nil || !valid || body.ResourceType != "Organization" || body.ID != id {
		fhir.WriteError(w, http.StatusBadRequest, "invalid", "Send this Organization, with its id, as FHIR JSON.")
		return
	}
	o, err := h.svc.Update(r.Context(), c, id, ch, version)
	h.write(w, r, o, err)
}

// updateBody is the part of an Organization an update reads.
type updateBody struct {
	ResourceType string          `json:"resourceType"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Extension    []bodyExtension `json:"extension"`
	Address      []struct {
		State     *string         `json:"state"`
		Extension []bodyExtension `json:"extension"`
	} `json:"address"`
}

// bodyExtension is an extension of an update's body.
type bodyExtension struct {
	ValueBoolean *bool   `json:"valueBoolean"`
	ValueCode    *string `json:"valueCode"`
	URL          string  `json:"url"`
}

// changes are what the body sets, and false when one of our extensions comes twice or without its value, or the
// address is not one with a state and one city.
func (b updateBody) changes() (Changes, bool) {
	ch := Changes{Name: b.Name}
	if !b.location(&ch) {
		return Changes{}, false
	}
	for _, e := range b.Extension {
		switch {
		case e.URL == extensionListed && ch.Listed == nil && e.ValueBoolean != nil && e.ValueCode == nil:
			ch.Listed = e.ValueBoolean
		case e.URL == extensionTimeZone && ch.TimeZone == nil && code(e.ValueCode, e.ValueBoolean):
			ch.TimeZone = e.ValueCode
		case e.URL == extensionDefaultLanguage && ch.DefaultLanguage == nil && code(e.ValueCode, e.ValueBoolean):
			ch.DefaultLanguage = e.ValueCode
		case e.URL == extensionTranscriptCopy && ch.PatientTranscriptCopy == nil && e.ValueBoolean != nil && e.ValueCode == nil:
			ch.PatientTranscriptCopy = e.ValueBoolean
		case e.URL == extensionContributes && ch.AnalyticsContributes == nil && e.ValueBoolean != nil && e.ValueCode == nil:
			ch.AnalyticsContributes = e.ValueBoolean
		case e.URL == extensionListed || e.URL == extensionTimeZone || e.URL == extensionDefaultLanguage || e.URL == extensionTranscriptCopy ||
			e.URL == extensionContributes:
			return Changes{}, false
		}
	}
	return ch, true
}

// location sets the region and the city of ch from the body's address, when it has one: one address with a state
// and one organization-city extension; false for any other address. Its city name is ignored.
func (b updateBody) location(ch *Changes) bool {
	switch {
	case len(b.Address) == 0:
		return true
	case len(b.Address) > 1 || !code(b.Address[0].State, nil):
		return false
	}
	for _, e := range b.Address[0].Extension {
		switch {
		case e.URL == extensionCity && ch.City == nil && code(e.ValueCode, e.ValueBoolean):
			ch.City = e.ValueCode
		case e.URL == extensionCity:
			return false
		}
	}
	ch.Region = b.Address[0].State
	return ch.City != nil
}

// code reports whether an extension's value is a code that is not empty, and nothing else.
func code(valueCode *string, valueBoolean *bool) bool {
	return valueCode != nil && *valueCode != "" && valueBoolean == nil
}

// ifMatch is the version the If-Match header names, and false for none.
func ifMatch(r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 32)
	return int32(n), err == nil
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

// write sends the clinic with its ETag, or the OperationOutcome for err.
func (h *Handler) write(w http.ResponseWriter, r *http.Request, o Organization, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
	case errors.Is(err, ErrConflict):
		fhir.WriteError(w, http.StatusPreconditionFailed, "conflict", "The resource changed since you read it. Read it again.")
	case errors.Is(err, ErrForbidden):
		fhir.WriteError(w, http.StatusForbidden, "forbidden", "Only an admin of the clinic may change it.")
	case errors.Is(err, ErrInvalidName):
		fhir.WriteError(w, http.StatusUnprocessableEntity, "business-rule", "The name must be 1 to 200 characters.")
	case errors.Is(err, ErrInvalidTimeZone):
		fhir.WriteError(w, http.StatusBadRequest, "value", "The time zone must be an IANA time zone name.")
	case errors.Is(err, ErrInvalidDefaultLanguage):
		fhir.WriteError(w, http.StatusBadRequest, "value", "The default language must be bs, hr, sr-Latn, sr-Cyrl or en.")
	case errors.Is(err, ErrInvalidLocation):
		fhir.WriteError(w, http.StatusBadRequest, "code-invalid", "The city must be a city of the region given as the state.")
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		res := toFHIR(o)
		w.Header().Set("ETag", fhir.ETag(res.Meta.VersionID))
		fhir.Write(w, http.StatusOK, res)
	}
}
