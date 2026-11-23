package chargeitemdefinition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a service as sent: a name, a code and ten translated names.
const maxBody = 16 << 10

// PriceList is what the handler calls (Service).
type PriceList interface {
	List(ctx context.Context, c Caller) ([]Definition, error)
	Create(ctx context.Context, c Caller, d Definition) (Definition, error)
	Update(ctx context.Context, c Caller, d Definition) (Definition, error)
	Offered(ctx context.Context, org string, mode schedule.VisitMode, language string) ([]Definition, error)
	Import(ctx context.Context, c Caller, f ImportFile, save bool) ([]ImportRow, error)
}

// Handler serves the price list (api/openapi.yaml, tag billing). Errors are {"error": "<code>"}.
type Handler struct {
	svc    PriceList
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc; caller resolves the signed-in session (with whether it is the clinic's
// admin).
func NewHandler(svc PriceList, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/services", h.list)
	mux.HandleFunc("POST /clinic/services", h.create)
	mux.HandleFunc("POST /clinic/services/import", h.importFile)
	mux.HandleFunc("PUT /clinic/services/{id}", h.update)
	mux.HandleFunc("GET /organizations/{id}/services", h.offered)
}

// translationJSON is the ServiceTranslation schema.
type translationJSON struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

// serviceJSON is the Service schema.
type serviceJSON struct {
	VATRateBP       *int32               `json:"vatRateBp,omitempty"`
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Code            string               `json:"code,omitempty"`
	Currency        string               `json:"currency"`
	Modes           []schedule.VisitMode `json:"modes"`
	Translations    []translationJSON    `json:"translations"`
	PriceMinor      int64                `json:"priceMinor"`
	DurationMinutes int32                `json:"durationMinutes,omitempty"`
	VersionID       int32                `json:"versionId"`
	Active          bool                 `json:"active"`
}

// offeredJSON is the OfferedService schema: what a patient sees at booking.
type offeredJSON struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Currency        string `json:"currency"`
	PriceMinor      int64  `json:"priceMinor"`
	DurationMinutes int32  `json:"durationMinutes,omitempty"`
}

// saveJSON is the SaveServiceRequest schema: the pointers tell a required field left out.
type saveJSON struct {
	Name            *string               `json:"name"`
	PriceMinor      *int64                `json:"priceMinor"`
	VATRateBP       *int32                `json:"vatRateBp"`
	Modes           *[]schedule.VisitMode `json:"modes"`
	Active          *bool                 `json:"active"`
	Translations    *[]translationJSON    `json:"translations"`
	Code            string                `json:"code"`
	Currency        string                `json:"currency"`
	DurationMinutes int32                 `json:"durationMinutes"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.List(r.Context(), c)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]serviceJSON, len(all))
	for i, d := range all {
		out[i] = toJSON(d)
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	d, ok := decode(w, r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.Create(r.Context(), c, d)
	h.stored(w, r, http.StatusCreated, out, err)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	d, ok := decode(w, r)
	tag, okTag := fhir.IfMatch(r.Header.Get("If-Match"))
	version, err := strconv.ParseInt(tag, 10, 32)
	if !ok || !okTag || err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	d.ID, d.VersionID = r.PathValue("id"), int32(version)
	out, err := h.svc.Update(r.Context(), c, d)
	h.stored(w, r, http.StatusOK, out, err)
}

func (h *Handler) offered(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	all, err := h.svc.Offered(r.Context(), r.PathValue("id"), schedule.VisitMode(q.Get("mode")), q.Get("language"))
	switch {
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	case err != nil:
		h.fail(w, r, err)
	default:
		out := make([]offeredJSON, len(all))
		for i, d := range all {
			out[i] = offeredJSON{ID: d.ID, Name: d.Name, PriceMinor: d.PriceMinor, Currency: d.Currency, DurationMinutes: d.DurationMinutes}
		}
		httpx.JSON(w, http.StatusOK, out)
	}
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Caller{}, false
	}
	return c, true
}

// stored writes a saved service with its version as the ETag, or the error.
func (h *Handler) stored(w http.ResponseWriter, r *http.Request, status int, d Definition, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(int(d.VersionID))))
	httpx.JSON(w, status, toJSON(d))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_service")
	case errors.Is(err, ErrConflict):
		writeCode(w, http.StatusPreconditionFailed, "version_conflict")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

// decode reads a SaveServiceRequest, and is false for a body that is not one.
func decode(w http.ResponseWriter, r *http.Request) (Definition, bool) {
	var in saveJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF ||
		in.Name == nil || in.PriceMinor == nil || in.Modes == nil || in.Active == nil || in.Translations == nil {
		return Definition{}, false
	}
	d := Definition{
		Name: *in.Name, Code: in.Code, PriceMinor: *in.PriceMinor, Currency: in.Currency, DurationMinutes: in.DurationMinutes,
		VATRateBP: in.VATRateBP, Modes: *in.Modes, Active: *in.Active,
	}
	for _, t := range *in.Translations {
		d.Translations = append(d.Translations, Translation(t))
	}
	return d, true
}

func toJSON(d Definition) serviceJSON {
	out := serviceJSON{
		ID: d.ID, Name: d.Name, Code: d.Code, PriceMinor: d.PriceMinor, Currency: d.Currency, DurationMinutes: d.DurationMinutes,
		VATRateBP: d.VATRateBP, Modes: d.Modes, Active: d.Active, Translations: make([]translationJSON, len(d.Translations)),
		VersionID: d.VersionID,
	}
	for i, t := range d.Translations {
		out.Translations[i] = translationJSON(t)
	}
	return out
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
