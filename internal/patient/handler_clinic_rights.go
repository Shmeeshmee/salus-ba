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
func (h *ClinicHandler) rightsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /clinic/patients/{id}", h.correct)
	mux.HandleFunc("POST /clinic/patients/{id}/consent-withdrawal", h.withdraw)
	mux.HandleFunc("GET /clinic/patients/{id}/record", h.record)
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

// collection is a FHIR R4 Bundle of type collection: the resources of a record handed over, each at its urn:uuid.
type collection struct {
	ResourceType string            `json:"resourceType"`
	Type         string            `json:"type"`
	Timestamp    string            `json:"timestamp"`
	Entry        []collectionEntry `json:"entry"`
}

type collectionEntry struct {
	Resource any    `json:"resource"`
	FullURL  string `json:"fullUrl"`
}

func (h *ClinicHandler) record(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	rec, err := h.svc.ClinicRecord(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	b := collection{ResourceType: "Bundle", Type: "collection", Timestamp: rec.Patient.LastUpdated.UTC().Format(time.RFC3339),
		Entry: []collectionEntry{{FullURL: "urn:uuid:" + rec.Patient.ID, Resource: toFHIR(rec.Patient)}}}
	for _, x := range rec.Consents {
		b.Entry = append(b.Entry, collectionEntry{FullURL: "urn:uuid:" + x.ID, Resource: consent.FHIR(x)})
		if x.LastUpdated.After(rec.Patient.LastUpdated) {
			b.Timestamp = x.LastUpdated.UTC().Format(time.RFC3339)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="patient-`+rec.Patient.ID+`.json"`)
	fhir.Write(w, http.StatusOK, b)
}

// rightsError writes the rights routes' own refusals, and reports whether err was one.
func rightsError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, consent.ErrInactive) {
		writeCode(w, http.StatusConflict, "consent_inactive")
		return true
	}
	return false
}
