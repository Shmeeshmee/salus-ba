package location

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a location as sent: a few short texts.
const maxBody = 8 << 10

// Locations is what the handler calls (Service).
type Locations interface {
	Locations(ctx context.Context, st Staff) ([]Location, error)
	SaveLocation(ctx context.Context, st Staff, l Location) (Location, error)
	PublicLocations(ctx context.Context, org string, served organization.Served) ([]Location, error)
}

// Handler serves the clinic's locations (api/openapi.yaml, tag locations). Errors are {"error": "<code>"}.
type Handler struct {
	svc    Locations
	caller func(ctx context.Context) (Staff, bool, error)
	served organization.Served // the clinics the signed-in caller keeps once their membership stops; nil: none
	log    *slog.Logger
}

// NewHandler returns the handler on svc; caller resolves the signed-in staff (with whether it is the clinic's admin).
func NewHandler(svc Locations, caller func(ctx context.Context) (Staff, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// WithServed returns h with the public locations read for the clinics served gives too once their membership stops.
func (h *Handler) WithServed(served organization.Served) *Handler {
	out := *h
	out.served = served
	return &out
}

// Routes declares the routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/locations", h.list)
	mux.HandleFunc("POST /clinic/locations", h.create)
	mux.HandleFunc("PUT /clinic/locations/{id}", h.update)
	mux.HandleFunc("GET /organizations/{id}/locations", h.public)
}

// saveJSON is the SaveLocationRequest schema: the pointers tell a required field left out.
type saveJSON struct {
	Name        *string `json:"name"`
	Active      *bool   `json:"active"`
	AddressLine string  `json:"addressLine"`
	PostalCode  string  `json:"postalCode"`
	CityID      string  `json:"cityId"`
	Phone       string  `json:"phone"`
	TimeZone    string  `json:"timeZone"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Locations(r.Context(), st)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, nonNil(out))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	l, ok := decode(w, r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.SaveLocation(r.Context(), st, l)
	h.stored(w, r, http.StatusCreated, out, err)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	l, ok := decode(w, r)
	tag, okTag := fhir.IfMatch(r.Header.Get("If-Match"))
	version, err := strconv.Atoi(tag)
	if !ok || !okTag || err != nil || version < 1 {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	l.ID, l.VersionID = r.PathValue("id"), version
	out, err := h.svc.SaveLocation(r.Context(), st, l)
	h.stored(w, r, http.StatusOK, out, err)
}

func (h *Handler) public(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.PublicLocations(r.Context(), r.PathValue("id"), h.served)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, nonNil(out))
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Staff, bool) {
	st, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Staff{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Staff{}, false
	}
	return st, true
}

// stored writes a saved location with its version as the ETag, or the error.
func (h *Handler) stored(w http.ResponseWriter, r *http.Request, status int, l Location, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(l.VersionID)))
	httpx.JSON(w, status, l)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_location")
	case errors.Is(err, ErrVersionConflict):
		writeCode(w, http.StatusPreconditionFailed, "version_conflict")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

// decode reads a SaveLocationRequest, and is false for a body that is not one.
func decode(w http.ResponseWriter, r *http.Request) (Location, bool) {
	var in saveJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.Name == nil || in.Active == nil {
		return Location{}, false
	}
	return Location{
		Name: *in.Name, Active: *in.Active, AddressLine: in.AddressLine, PostalCode: in.PostalCode, CityID: in.CityID,
		Phone: in.Phone, TimeZone: in.TimeZone,
	}, true
}

// nonNil is ls, or an empty list for none: the schema is an array.
func nonNil(ls []Location) []Location {
	if ls == nil {
		return []Location{}
	}
	return ls
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
