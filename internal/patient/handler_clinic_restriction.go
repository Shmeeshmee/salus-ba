package patient

import "net/http"

// restrictionRoutes declares the restriction of a patient's record at the clinic (docs/clinic-cms/WAVE5.md D4, S7).
func (h *ClinicHandler) restrictionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /clinic/patients/{id}/restriction", h.restrict)
	mux.HandleFunc("DELETE /clinic/patients/{id}/restriction", h.lift)
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
