package diagnosticreport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// uuidText is a UUID in its text form.
var uuidText = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// maxBody is the largest request body read: a report of MaxResults values.
const maxBody = 64 << 10

// Reports is what the HTTPHandler calls: Service.
type Reports interface {
	Attach(ctx context.Context, c Staff, appointmentID string, d Draft) (Report, error)
	Change(ctx context.Context, c Staff, id string, d Draft) (Report, error)
	Release(ctx context.Context, c Staff, id string, version int) (Report, error)
	ForVisit(ctx context.Context, c Staff, appointmentID string) ([]Report, error)
	Read(ctx context.Context, c Staff, id string) (Report, error)
	Mine(ctx context.Context, c Patient) ([]Report, error)
	MyReport(ctx context.Context, c Patient, id string) (Report, error)
}

// Session is who is signed in: staff of a clinic (OrganizationID set) or a patient account.
type Session struct {
	Patient Patient
	Staff   Staff
}

// HTTPHandler serves the lab results on the JSON API (api/openapi.yaml, tag lab-results).
type HTTPHandler struct {
	svc     Reports
	session func(ctx context.Context) (Session, bool, error)
	log     *slog.Logger
}

// NewHTTPHandler returns the HTTPHandler; session gives who is signed in, false for nobody.
func NewHTTPHandler(svc Reports, session func(ctx context.Context) (Session, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, session: session, log: log}
}

// Routes declares the lab results' routes on mux. The PDF's routes are not served yet (PROGRESS.md, LB1).
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/appointments/{id}/diagnostic-reports", h.staff(h.forVisit))
	mux.HandleFunc("POST /clinic/appointments/{id}/diagnostic-reports", h.staff(h.attach))
	mux.HandleFunc("GET /clinic/diagnostic-reports/{id}", h.staff(h.read))
	mux.HandleFunc("PUT /clinic/diagnostic-reports/{id}", h.staff(h.change))
	mux.HandleFunc("POST /clinic/diagnostic-reports/{id}/release", h.staff(h.release))
	mux.HandleFunc("GET /me/diagnostic-reports", h.mine)
	mux.HandleFunc("GET /me/diagnostic-reports/{id}", h.myReport)
}

// reportJSON, resultJSON and saveJSON are the DiagnosticReport, DiagnosticReportResult and
// SaveDiagnosticReportRequest schemas.
type resultJSON struct {
	Value          *float64       `json:"value,omitempty"`
	Test           string         `json:"test"`
	LOINC          string         `json:"loinc,omitempty"`
	Unit           string         `json:"unit,omitempty"`
	ValueText      string         `json:"valueText,omitempty"`
	ReferenceRange string         `json:"referenceRange,omitempty"`
	Interpretation Interpretation `json:"interpretation,omitempty"`
}

type reportJSON struct {
	ReleasedAt     *time.Time   `json:"releasedAt,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	ID             string       `json:"id"`
	OrganizationID string       `json:"organizationId"`
	PatientID      string       `json:"patientId"`
	AppointmentID  string       `json:"appointmentId"`
	Status         Status       `json:"status"`
	Title          string       `json:"title"`
	LOINC          string       `json:"loinc,omitempty"`
	EffectiveDate  string       `json:"effectiveDate,omitempty"`
	Results        []resultJSON `json:"results"`
	VersionID      int          `json:"versionId"`
	HasPDF         bool         `json:"hasPdf"`
}

type saveJSON struct {
	Results       *[]resultJSON `json:"results"`
	Title         string        `json:"title"`
	LOINC         string        `json:"loinc"`
	EffectiveDate string        `json:"effectiveDate"`
}

// ReportJSON is the report as the API's DiagnosticReport schema serves it.
func ReportJSON(r Report) any { return toJSON(r) }

func toJSON(r Report) reportJSON {
	out := reportJSON{
		ReleasedAt: r.ReleasedAt, CreatedAt: r.CreatedAt, ID: r.ID, OrganizationID: r.OrganizationID, PatientID: r.PatientID,
		AppointmentID: r.AppointmentID, Status: r.Status, Title: r.Title, LOINC: r.LOINC, VersionID: r.VersionID,
		HasPDF: r.HasPDF, Results: make([]resultJSON, len(r.Results)),
	}
	if r.EffectiveDate != nil {
		out.EffectiveDate = r.EffectiveDate.Format(time.DateOnly)
	}
	for i, v := range r.Results {
		out.Results[i] = resultJSON(v)
	}
	return out
}

func listJSON(list []Report) []reportJSON {
	out := make([]reportJSON, len(list))
	for i, r := range list {
		out[i] = toJSON(r)
	}
	return out
}

// staff serves next to a clinic's account; a patient account gets the 404 of a visit it cannot see.
func (h *HTTPHandler) staff(next func(http.ResponseWriter, *http.Request, Staff)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := h.signedIn(w, r)
		switch {
		case !ok:
		case s.Staff.OrganizationID == "" || !uuidText.MatchString(r.PathValue("id")):
			writeCode(w, http.StatusNotFound, "not_found")
		default:
			next(w, r, s.Staff)
		}
	}
}

func (h *HTTPHandler) forVisit(w http.ResponseWriter, r *http.Request, c Staff) {
	list, err := h.svc.ForVisit(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, hidden(err)) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

func (h *HTTPHandler) attach(w http.ResponseWriter, r *http.Request, c Staff) {
	d, ok := decodeDraft(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Attach(r.Context(), c, r.PathValue("id"), d)
	h.stored(w, r, http.StatusCreated, out, err)
}

func (h *HTTPHandler) read(w http.ResponseWriter, r *http.Request, c Staff) {
	out, err := h.svc.Read(r.Context(), c, r.PathValue("id"))
	if h.failed(w, r, hidden(err)) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(out))
}

func (h *HTTPHandler) change(w http.ResponseWriter, r *http.Request, c Staff) {
	d, ok := decodeDraft(w, r)
	if !ok {
		return
	}
	if d.Version, ok = ifMatch(w, r); !ok {
		return
	}
	out, err := h.svc.Change(r.Context(), c, r.PathValue("id"), d)
	h.stored(w, r, http.StatusOK, out, err)
}

func (h *HTTPHandler) release(w http.ResponseWriter, r *http.Request, c Staff) {
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Release(r.Context(), c, r.PathValue("id"), version)
	h.stored(w, r, http.StatusOK, out, err)
}

func (h *HTTPHandler) mine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Mine(r.Context(), s.Patient)
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(list))
}

func (h *HTTPHandler) myReport(w http.ResponseWriter, r *http.Request) {
	s, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if !uuidText.MatchString(r.PathValue("id")) {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	out, err := h.svc.MyReport(r.Context(), s.Patient, r.PathValue("id"))
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(out))
}

// stored answers a write with the report and its ETag, or err.
func (h *HTTPHandler) stored(w http.ResponseWriter, r *http.Request, status int, out Report, err error) {
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(out.VersionID)))
	httpx.JSON(w, status, toJSON(out))
}

// decodeDraft reads a SaveDiagnosticReportRequest; 400 invalid_request for a body that is unreadable, has an unknown
// field, lacks results or has a date that is no date.
func decodeDraft(w http.ResponseWriter, r *http.Request) (Draft, bool) {
	var in saveJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.Results == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return Draft{}, false
	}
	d := Draft{Title: in.Title, LOINC: in.LOINC, Results: make([]Result, len(*in.Results))}
	if in.EffectiveDate != "" {
		day, err := time.Parse(time.DateOnly, in.EffectiveDate)
		if err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return Draft{}, false
		}
		d.EffectiveDate = &day
	}
	for i, v := range *in.Results {
		d.Results[i] = Result(v)
	}
	return d, true
}

// ifMatch returns the version If-Match names, or answers 400 and false.
func ifMatch(w http.ResponseWriter, r *http.Request) (int, bool) {
	tag, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	v, err := strconv.Atoi(tag)
	if !ok || err != nil || v < 1 {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return 0, false
	}
	return v, true
}

func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Session, bool) {
	s, ok, err := h.session(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	}
	return s, ok && err == nil
}

// hidden is err for a read, whose contract answers a receptionist as a stranger: 404, not 403.
func hidden(err error) error {
	if errors.Is(err, ErrForbidden) {
		return ErrNotFound
	}
	return err
}

// errorCodes maps the domain's errors to their status and code (api/openapi.yaml).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalid, "invalid_diagnostic_report", http.StatusBadRequest},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrReleased, "report_released", http.StatusConflict},
	{ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
}

func (h *HTTPHandler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	for _, e := range errorCodes {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return true
		}
	}
	httpx.InternalError(w, r, h.log, err)
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
