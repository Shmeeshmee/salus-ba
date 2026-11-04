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

// Roles reads and searches practitioner roles for a caller, and the public directory's for anyone (Service).
type Roles interface {
	Read(ctx context.Context, c Caller, id string) (PractitionerRole, error)
	Search(ctx context.Context, c Caller, org string) ([]PractitionerRole, error)
	PublicByOrganization(ctx context.Context, org string) ([]PractitionerRole, error)
}

// Practitioners reads a practitioner for a caller (practitioner.Service): each one a staff search includes is
// read, and audited, as a read of its own; a public search includes their public names.
type Practitioners interface {
	Read(ctx context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error)
	PublicNames(ctx context.Context, ids []string) ([]practitioner.Practitioner, error)
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

// resource is a PractitionerRole as FHIR R4 JSON: only the elements the accounts SPEC "Directory model" and
// the booking SPEC "Specialty" name.
type resource struct {
	ResourceType string                 `json:"resourceType"`
	ID           string                 `json:"id"`
	Practitioner fhir.Reference         `json:"practitioner"`
	Organization fhir.Reference         `json:"organization"`
	Meta         fhir.Meta              `json:"meta"`
	Specialty    []fhir.CodeableConcept `json:"specialty,omitempty"`
	Active       bool                   `json:"active"`
}

func toFHIR(p PractitionerRole) resource {
	res := resource{
		ResourceType: "PractitionerRole", ID: p.ID, Meta: fhir.NewMeta(p.VersionID, p.LastUpdated), Active: p.Active,
		Practitioner: fhir.Reference{Reference: "Practitioner/" + p.PractitionerID},
		Organization: fhir.Reference{Reference: "Organization/" + p.OrganizationID},
	}
	if p.SpecialtyCode != "" {
		sp, _ := SpecialtyByCode(p.SpecialtyCode) // the service stores listed codes only
		res.Specialty = []fhir.CodeableConcept{{
			Coding: []fhir.Coding{{System: SystemSNOMED, Code: p.SpecialtyCode, Display: sp.Display}}, Text: sp.Text,
		}}
	}
	return res
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
// searchset Bundle: to staff of the clinic its roles, audited; to anyone else, signed in or not, the public
// directory's (the active roles of a listed clinic, none for another).
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	org, include, ok := searchQuery(r.URL.RawQuery)
	if !ok {
		fhir.WriteError(w, http.StatusBadRequest, "not-supported",
			"Search by one organization, optionally with _include=PractitionerRole:practitioner.")
		return
	}
	c, signedIn := h.caller(r.Context())
	staff := signedIn && c.OrganizationID == org
	var roles []PractitionerRole
	var err error
	if staff {
		roles, err = h.roles.Search(r.Context(), c, org)
	} else {
		roles, err = h.roles.PublicByOrganization(r.Context(), org)
	}
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
		if staff {
			included, err = h.practitioners(r.Context(), c, roles)
		} else {
			included, err = h.publicPractitioners(r.Context(), roles)
		}
		if err != nil {
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

// publicPractitioners gives the public names of the roles' practitioners.
func (h *Handler) publicPractitioners(ctx context.Context, roles []PractitionerRole) ([]fhir.BundleEntry, error) {
	if len(roles) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.PractitionerID)
	}
	people, err := h.people.PublicNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]fhir.BundleEntry, 0, len(people))
	for _, p := range people {
		out = append(out, fhir.BundleEntry{FullURL: h.base + "/Practitioner/" + p.ID, Resource: practitioner.ToFHIR(p)})
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
