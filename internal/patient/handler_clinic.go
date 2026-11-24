package patient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// ClinicPatients is what the clinic handler calls (Service).
type ClinicPatients interface {
	SearchClinic(ctx context.Context, c Caller, q ClinicSearch) ([]Patient, error)
	Register(ctx context.Context, c Caller, in Patient) (Patient, error)
	Correct(ctx context.Context, c Caller, in Patient) (Patient, error)
	WithdrawConsent(ctx context.Context, c Caller, id string) (consent.Consent, error)
	ClinicRecord(ctx context.Context, c Caller, id string) (ClinicRecord, error)
}

// ClinicHandler serves the clinic's own patients in the JSON API (package F2 of docs/clinic-cms): the staff's search
// and the registration of a patient without an account. Errors as Handler's.
type ClinicHandler struct {
	svc ClinicPatients
	// caller is the staff caller, with its admin flag and practitioner role, false for nobody (internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewClinicHandler returns the handler on svc.
func NewClinicHandler(svc ClinicPatients, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *ClinicHandler {
	return &ClinicHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the clinic's patient routes on mux.
func (h *ClinicHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/patients", h.search)
	mux.HandleFunc("POST /clinic/patients", h.register)
	h.rightsRoutes(mux) // docs/clinic-cms R2
}

// clinicPatientJSON is the ClinicPatient schema of api/openapi.yaml.
type clinicPatientJSON struct {
	LastUpdated        time.Time `json:"lastUpdated"`
	ID                 string    `json:"id"`
	GivenName          string    `json:"givenName"`
	FamilyName         string    `json:"familyName"`
	BirthDate          string    `json:"birthDate"`
	Phone              string    `json:"phone,omitempty"`
	RegisteredByClinic bool      `json:"registeredByClinic"`
	VersionID          int32     `json:"versionId"`
}

func toClinicJSON(p Patient) clinicPatientJSON {
	return clinicPatientJSON{
		ID: p.ID, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate.Format(dateLayout), Phone: p.Phone,
		RegisteredByClinic: p.RegisteredBy != "", VersionID: p.VersionID, LastUpdated: p.LastUpdated.UTC(),
	}
}

// registerRequest is the RegisterPatientRequest schema.
type registerRequest struct {
	ConsentGivenInPerson *bool  `json:"consentGivenInPerson"`
	GivenName            string `json:"givenName"`
	FamilyName           string `json:"familyName"`
	BirthDate            string `json:"birthDate"`
	Phone                string `json:"phone"`
}

func (h *ClinicHandler) staff(w http.ResponseWriter, r *http.Request) (Caller, bool) {
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

func (h *ClinicHandler) search(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	q := ClinicSearch{Name: r.URL.Query().Get("name")}
	if b := r.URL.Query().Get("birthDate"); b != "" {
		var err error
		if q.BirthDate, err = time.Parse(dateLayout, b); err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	if q.Name == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	found, err := h.svc.SearchClinic(r.Context(), c, q)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]clinicPatientJSON, 0, len(found))
	for _, p := range found {
		out = append(out, toClinicJSON(p))
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *ClinicHandler) register(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
	if !ok {
		return
	}
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ConsentGivenInPerson == nil || !*req.ConsentGivenInPerson {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	born, err := time.Parse(dateLayout, req.BirthDate)
	if err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_birth_date")
		return
	}
	p, err := h.svc.Register(r.Context(), c, Patient{GivenName: req.GivenName, FamilyName: req.FamilyName, BirthDate: born, Phone: req.Phone})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(p.VersionID), 10)))
	httpx.JSON(w, http.StatusCreated, toClinicJSON(p))
}

func (h *ClinicHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrInvalidSearch) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if rightsError(w, err) {
		return
	}
	(&Handler{log: h.log}).writeError(w, r, err)
}
