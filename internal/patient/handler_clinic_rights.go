package patient

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// rightsRoutes declares the routes of a registered patient's rights exercised through the clinic (docs/clinic-cms R2).
// The record's hand-over (GET /clinic/patients/{id}/record) is served by clinicexport, which builds it with the
// patient's visits and receipts.
func (h *ClinicHandler) rightsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /clinic/patients/{id}", h.correct)
	mux.HandleFunc("POST /clinic/patients/{id}/consent", h.recordConsent)
	mux.HandleFunc("POST /clinic/patients/{id}/consent-withdrawal", h.withdraw)
}

// correctRequest is the CorrectPatientRequest schema.
type correctRequest struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	BirthDate  string `json:"birthDate"`
	Phone      string `json:"phone"`
}

func (h *ClinicHandler) correct(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	v, matched := fhir.IfMatch(r.Header.Get("If-Match"))
	version, err := strconv.ParseInt(v, 10, 32)
	if !matched || err != nil {
		writeCode(w, http.StatusBadRequest, "if_match_required")
		return
	}
	var req correctRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	born, err := time.Parse(dateLayout, req.BirthDate)
	if err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_birth_date")
		return
	}
	p, err := h.svc.Correct(r.Context(), c, Patient{
		ID: r.PathValue("id"), VersionID: int32(version), GivenName: req.GivenName, FamilyName: req.FamilyName, BirthDate: born, Phone: req.Phone,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(p.VersionID), 10)))
	httpx.JSON(w, http.StatusOK, toClinicJSON(p))
}

func (h *ClinicHandler) withdraw(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	if _, err := h.svc.WithdrawConsent(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClinicHandler) recordConsent(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	if _, err := h.svc.RecordConsent(r.Context(), c, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// rightsError writes the rights routes' own refusals, and reports whether err was one.
func rightsError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, consent.ErrInactive) {
		writeCode(w, http.StatusConflict, "consent_inactive")
		return true
	}
	if errors.Is(err, consent.ErrActive) {
		writeCode(w, http.StatusConflict, "consent_active")
		return true
	}
	return false
}
