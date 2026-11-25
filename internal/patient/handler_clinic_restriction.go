package patient

import (
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// restrictionRoutes declares the restriction of a patient's record at the clinic (docs/clinic-cms/WAVE5.md D4, S7).
func (h *ClinicHandler) restrictionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/patients/{id}/restriction", h.restriction)
	mux.HandleFunc("PUT /clinic/patients/{id}/restriction", h.restrict)
	mux.HandleFunc("DELETE /clinic/patients/{id}/restriction", h.lift)
}

// restrictionJSON is the PatientRestriction schema: since when, and by the clinic or the erasure, while restricted.
type restrictionJSON struct {
	At         *time.Time `json:"restrictedAt,omitempty"`
	By         string     `json:"by,omitempty"`
	Restricted bool       `json:"restricted"`
}

func (h *ClinicHandler) restriction(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	got, err := h.svc.Restriction(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := restrictionJSON{Restricted: got.Restricted()}
	if out.Restricted {
		at := got.At.UTC()
		out.At, out.By = &at, "clinic"
		if got.ByErasure {
			out.By = "erasure"
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

// restrictRequest is the request body of restrictClinicPatient, which may be absent.
type restrictRequest struct {
	DataRequestID string `json:"dataRequestId"`
}

func (h *ClinicHandler) restrict(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	var req restrictRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.Restrict(r.Context(), c, r.PathValue("id"), req.DataRequestID); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClinicHandler) lift(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	if err := h.svc.LiftRestriction(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
