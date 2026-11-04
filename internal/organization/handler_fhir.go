package organization

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

const maxBody = 64 << 10 // bytes of an update's body

// Clinics reads and renames a clinic for a caller (Service).
type Clinics interface {
	Read(ctx context.Context, c Caller, id string) (Organization, error)
	Update(ctx context.Context, c Caller, id string, ch Changes, version int32) (Organization, error)
}

// Handler serves the Organization read and update of the FHIR API (api/fhir/CapabilityStatement.json).
type Handler struct {
	svc Clinics
	// caller is who is signed in for the request, and false for nobody (from the session and the account's
	// role, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Clinics, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the Organization routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /fhir/R4/Organization/{id}", h.read)
	mux.HandleFunc("PUT /fhir/R4/Organization/{id}", h.update)
}

// resource is an Organization as FHIR R4 JSON: only the elements SPEC.md "Directory model" names.
type resource struct {
	ResourceType string    `json:"resourceType"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Meta         fhir.Meta `json:"meta"`
	Active       bool      `json:"active"`
}

func toFHIR(o Organization) resource {
	return resource{
		ResourceType: "Organization", ID: o.ID, Name: o.Name, Active: o.Active,
		Meta: fhir.NewMeta(o.VersionID, o.LastUpdated),
	}
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	o, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	h.write(w, r, o, err)
}

// update renames the clinic: the body is the Organization with the new name, If-Match the version it was
// read at. Only the name changes; the other elements of the body are ignored.
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
	var body struct {
		ResourceType string `json:"resourceType"`
		ID           string `json:"id"`
		Name         string `json:"name"`
	}
	id := r.PathValue("id")
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&body)
	if err != nil || body.ResourceType != "Organization" || body.ID != id {
		fhir.WriteError(w, http.StatusBadRequest, "invalid", "Send this Organization, with its id, as FHIR JSON.")
		return
	}
	o, err := h.svc.Update(r.Context(), c, id, Changes{Name: body.Name}, version)
	h.write(w, r, o, err)
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
	case err != nil:
		fhir.InternalError(w, r, h.log, err)
	default:
		res := toFHIR(o)
		w.Header().Set("ETag", fhir.ETag(res.Meta.VersionID))
		fhir.Write(w, http.StatusOK, res)
	}
}
