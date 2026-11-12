package consent

import (
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The bounds of a selection (api/openapi.yaml SetHistorySharingRequest): at most maxShared visits, a UUID each.
const (
	maxShared      = 500
	maxSharingBody = 32 << 10
)

func (h *Handler) candidates(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Candidates(r.Context(), c, r.PathValue("organizationId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if out == nil {
		out = []Candidate{}
	}
	write(w, http.StatusOK, struct {
		Candidates []Candidate `json:"candidates"`
	}{out})
}

func (h *Handler) setSharing(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var version int32
	if header := r.Header.Get("If-Match"); header != "" {
		v, ok := fhir.IfMatch(header)
		n, err := strconv.ParseInt(v, 10, 32)
		if !ok || err != nil || n < 1 {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		version = int32(n)
	}
	var in SetSharing
	if !decodeJSONUpTo(w, r, &in, maxSharingBody) {
		return
	}
	if in.EncounterIDs == nil || len(in.EncounterIDs) > maxShared {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.SetSharing(r.Context(), c, r.PathValue("organizationId"), in, version)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if out.VersionID != "" {
		w.Header().Set("ETag", fhir.ETag(out.VersionID))
	}
	write(w, http.StatusOK, out)
}

func (h *Handler) stopSharing(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.StopSharing(r.Context(), c, r.PathValue("organizationId")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) mySharing(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.MySharing(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if out == nil {
		out = []Sharing{}
	}
	write(w, http.StatusOK, struct {
		Sharing []Sharing `json:"sharing"`
	}{out})
}
