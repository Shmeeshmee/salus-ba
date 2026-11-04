package practitionerrole

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// includePractitioner is the one _include this server supports.
const includePractitioner = "PractitionerRole:practitioner"

// Roles reads and searches practitioner roles for a caller (Service).
type Roles interface {
	Read(ctx context.Context, c Caller, id string) (PractitionerRole, error)
	Search(ctx context.Context, c Caller, org string) ([]PractitionerRole, error)
}

// Practitioners reads a practitioner for a caller (practitioner.Service): each one a search includes is read,
// and audited, as a read of its own.
type Practitioners interface {
	Read(ctx context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error)
}

// Handler serves the PractitionerRole read and search of the FHIR API (api/fhir/CapabilityStatement.json).
type Handler struct {
	roles  Roles
	people Practitioners
	// caller is who is signed in for the request, and false for nobody (from the session, in internal/api).
	caller func(ctx context.Context) (Caller, bool)
	log    *slog.Logger
	base   string // the FHIR API's absolute base URL, for a Bundle's links (FHIR_BASE_URL)
}

// NewHandler returns the handler on roles, reading included practitioners from people. base is the FHIR
// API's absolute base URL (https://…/fhir/R4).
func NewHandler(roles Roles, people Practitioners, caller func(ctx context.Context) (Caller, bool), base string, log *slog.Logger) *Handler {
	return &Handler{roles: roles, people: people, caller: caller, base: base, log: log}
}

// Routes declares the PractitionerRole routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/PractitionerRole/{id}", h.read)
	mux.HandleFunc("GET /fhir/R4/PractitionerRole", h.search)
}

// resource is a PractitionerRole as FHIR R4 JSON: only the elements SPEC.md "Directory model" names.
type resource struct {
	ResourceType string         `json:"resourceType"`
	ID           string         `json:"id"`
	Practitioner fhir.Reference `json:"practitioner"`
	Organization fhir.Reference `json:"organization"`
	Meta         fhir.Meta      `json:"meta"`
	Active       bool           `json:"active"`
}

func toFHIR(p PractitionerRole) resource {
	return resource{
		ResourceType: "PractitionerRole", ID: p.ID, Meta: fhir.NewMeta(p.VersionID, p.LastUpdated), Active: p.Active,
		Practitioner: fhir.Reference{Reference: "Practitioner/" + p.PractitionerID},
		Organization: fhir.Reference{Reference: "Organization/" + p.OrganizationID},
	}
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	p, err := h.roles.Read(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	res := toFHIR(p)
	w.Header().Set("ETag", fhir.ETag(res.Meta.VersionID))
	fhir.Write(w, http.StatusOK, res)
}

// search answers PractitionerRole?organization=<id>[&_include=PractitionerRole:practitioner] with a
// searchset Bundle.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	org, include, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by one organization, optionally with _include=PractitionerRole:practitioner.")
		return
	}
	roles, err := h.roles.Search(r.Context(), c, org)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	matches := make([]fhir.BundleEntry, 0, len(roles))
	for _, p := range roles {
		matches = append(matches, fhir.BundleEntry{FullURL: h.base + "/PractitionerRole/" + p.ID, Resource: toFHIR(p)})
	}
	self := h.base + "/PractitionerRole?organization=" + url.QueryEscape(org)
	var included []fhir.BundleEntry
	if include {
		self += "&_include=" + includePractitioner
		if included, err = h.practitioners(r.Context(), c, roles); err != nil {
			fhir.InternalError(w, r, h.log, err)
			return
		}
	}
	fhir.Write(w, http.StatusOK, fhir.NewSearchset(self, matches, included))
}

// practitioners reads the practitioner of each role for c, leaving out one c may not see.
func (h *Handler) practitioners(ctx context.Context, c Caller, roles []PractitionerRole) ([]fhir.BundleEntry, error) {
	pc := practitioner.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID}
	var out []fhir.BundleEntry
	for _, role := range roles {
		p, err := h.people.Read(ctx, pc, role.PractitionerID)
		switch {
		case errors.Is(err, practitioner.ErrNotFound):
		case err != nil:
			return nil, err
		default:
			out = append(out, fhir.BundleEntry{FullURL: h.base + "/Practitioner/" + p.ID, Resource: practitioner.ToFHIR(p)})
		}
	}
	return out, nil
}

// searchQuery returns the clinic a search names (organization=<id> or Organization/<id>, once) and whether
// it includes the practitioners, or false for any other search.
func searchQuery(raw string) (org string, include, ok bool) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return "", false, false
	}
	for key, values := range q {
		switch {
		case len(values) != 1:
			return "", false, false
		case key == "organization":
			org = strings.TrimPrefix(values[0], "Organization/")
		case key == "_include" && values[0] == includePractitioner:
			include = true
		default:
			return "", false, false
		}
	}
	if org == "" || strings.ContainsAny(org, ",/") {
		return "", false, false
	}
	return org, include, true
}

// signedIn returns the caller, or answers 401 and false when nobody is signed in.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok := h.caller(r.Context())
	if !ok {
		fhir.WriteError(w, http.StatusUnauthorized, "login", "Sign in to use this resource.")
	}
	return c, ok
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		fhir.WriteError(w, http.StatusNotFound, "not-found", "Resource not found.")
		return
	}
	fhir.InternalError(w, r, h.log, err)
}
