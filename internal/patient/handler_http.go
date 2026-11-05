package patient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a JSON request body; a profile fits many times over.
const maxBody = 4 << 10

// dateLayout is a FHIR date with day precision, as birthDate travels.
const dateLayout = "2006-01-02"

// Profiles is what the handler calls (Service).
type Profiles interface {
	Me(ctx context.Context, c Caller) (Patient, error)
	SaveMe(ctx context.Context, c Caller, in Patient) (Patient, error)
}

// Handler serves the patient's own profile in the JSON API (api/openapi.yaml). Errors are
// {"error": "<code>"} with a fixed code per case; internal error text goes to the log only, and no profile
// field ever does.
type Handler struct {
	svc Profiles
	// caller is who is signed in for the request, and false for nobody (from the session, in internal/api).
	caller func(ctx context.Context) (Caller, bool)
	log    *slog.Logger
}

// NewHandler returns the handler on svc.
func NewHandler(svc Profiles, caller func(ctx context.Context) (Caller, bool), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the profile routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /me/patient", h.me)
	mux.HandleFunc("PUT /me/patient", h.saveMe)
}

// profileJSON is the Patient schema of api/openapi.yaml.
type profileJSON struct {
	LastUpdated time.Time `json:"lastUpdated"`
	ID          string    `json:"id"`
	GivenName   string    `json:"givenName"`
	FamilyName  string    `json:"familyName"`
	BirthDate   string    `json:"birthDate"`
	Phone       string    `json:"phone,omitempty"`
	VersionID   int32     `json:"versionId"`
}

// saveRequest is the SavePatientRequest schema: no versionId creates the profile.
type saveRequest struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	BirthDate  string `json:"birthDate"`
	Phone      string `json:"phone"`
	VersionID  int32  `json:"versionId"`
}

func toJSON(p Patient) profileJSON {
	return profileJSON{
		ID: p.ID, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate.Format(dateLayout),
		Phone: p.Phone, VersionID: p.VersionID, LastUpdated: p.LastUpdated.UTC(),
	}
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	c, ok := h.caller(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	p, err := h.svc.Me(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeProfile(w, http.StatusOK, p)
}

func (h *Handler) saveMe(w http.ResponseWriter, r *http.Request) {
	c, ok := h.caller(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req saveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	born, err := time.Parse(dateLayout, req.BirthDate)
	if err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_birth_date")
		return
	}
	p, err := h.svc.SaveMe(r.Context(), c, Patient{
		GivenName: req.GivenName, FamilyName: req.FamilyName, BirthDate: born, Phone: req.Phone, VersionID: req.VersionID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if req.VersionID == 0 {
		status = http.StatusCreated
	}
	writeProfile(w, status, p)
}

// writeProfile sends p, never to be cached: it is health data.
func writeProfile(w http.ResponseWriter, status int, p Patient) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, toJSON(p))
}

// decodeJSON reads one JSON object with known fields only, or answers 400 invalid_request.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrConflict, "version_conflict", http.StatusConflict},
	{ErrInvalidName, "invalid_name", http.StatusBadRequest},
	{ErrInvalidBirthDate, "invalid_birth_date", http.StatusBadRequest},
	{ErrInvalidPhone, "invalid_phone", http.StatusBadRequest},
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			writeCode(w, c.status, c.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
