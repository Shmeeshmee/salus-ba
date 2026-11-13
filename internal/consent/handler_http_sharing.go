package consent

import (
	"net/http"
	"strconv"
	"strings"

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
	if out.Candidates == nil {
		out.Candidates = []Candidate{}
	}
	write(w, http.StatusOK, out)
}

func (h *Handler) setSharing(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var version string
	if header := r.Header.Get("If-Match"); header != "" {
		v, ok := fhir.IfMatch(header)
		if !ok || !validVersion(v) {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		version = v
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

// validVersion reports whether v has the shape of a sharing's version (Sharing.VersionID): a consent id and a
// version number from 1.
func validVersion(v string) bool {
	id, n, ok := strings.Cut(v, ".")
	version, err := strconv.ParseInt(n, 10, 32)
	return ok && err == nil && version >= 1 && fhir.IsUUID(id)
}
