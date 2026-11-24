package datarequest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Requests is what the handler calls (Service).
type Requests interface {
	File(ctx context.Context, p Patient, f Filing) (Request, error)
	Mine(ctx context.Context, p Patient) ([]Request, error)
	Withdraw(ctx context.Context, p Patient, id string, version int) (Request, error)
	FileOnBehalf(ctx context.Context, st Staff, patientID string, kind Kind) (Request, error)
	ClinicRequests(ctx context.Context, st Staff) ([]Request, error)
	ClinicRequest(ctx context.Context, st Staff, id string) (Request, []Change, error)
	Answer(ctx context.Context, st Staff, id string, a Answer) (Request, error)
	Extend(ctx context.Context, st Staff, id string, reason ExtensionReason, version int) (Request, error)
}

// Handler serves the patient's and the clinic's request routes (api/openapi.yaml, tag privacy, cms-d1). Errors are
// {"error": "<code>"}.
type Handler struct {
	svc    Requests
	caller func(ctx context.Context) (Staff, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc; caller resolves the signed-in session: staff with their clinic and role, a
// patient as Staff with no clinic.
func NewHandler(svc Requests, caller func(ctx context.Context) (Staff, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /me/data-requests", h.mine)
	mux.HandleFunc("POST /me/data-requests", h.file)
	mux.HandleFunc("POST /me/data-requests/{id}/withdraw", h.withdraw)
	mux.HandleFunc("GET /clinic/data-requests", h.clinicList)
	mux.HandleFunc("GET /clinic/data-requests/{id}", h.clinicRead)
	mux.HandleFunc("POST /clinic/data-requests/{id}/answer", h.answer)
	mux.HandleFunc("POST /clinic/data-requests/{id}/extend", h.extend)
	mux.HandleFunc("POST /clinic/patients/{id}/data-requests", h.fileOnBehalf)
}

// requestJSON is the DataRequest schema. PatientID is on the clinic's reads only; History on the clinic's read of one.
type requestJSON struct {
	ReceivedAt      time.Time       `json:"receivedAt"`
	DueAt           time.Time       `json:"dueAt"`
	ExtendedUntil   *time.Time      `json:"extendedUntil,omitempty"`
	AnsweredAt      *time.Time      `json:"answeredAt,omitempty"`
	ID              string          `json:"id"`
	Kind            Kind            `json:"kind"`
	Target          Target          `json:"target"`
	OrganizationID  string          `json:"organizationId,omitempty"`
	PatientID       string          `json:"patientId,omitempty"`
	Status          Status          `json:"status"`
	ExtensionReason ExtensionReason `json:"extensionReason,omitempty"`
	AnswerCode      AnswerCode      `json:"answerCode,omitempty"`
	History         []changeJSON    `json:"history,omitempty"`
	Version         int             `json:"version"`
	FiledByPatient  bool            `json:"filedByPatient"`
}

// changeJSON is the DataRequestChange schema.
type changeJSON struct {
	ChangedAt time.Time `json:"changedAt"`
	Status    Status    `json:"status"`
}

// toJSON is r as the patient (clinic false) or the clinic sees it.
func toJSON(r Request, clinic bool) requestJSON {
	out := requestJSON{
		ReceivedAt: r.ReceivedAt.UTC(), DueAt: r.DueAt.UTC(), ID: r.ID, Kind: r.Kind, Target: r.Target(),
		OrganizationID: r.OrganizationID, Status: r.Status, ExtensionReason: r.ExtensionReason, AnswerCode: r.AnswerCode,
		Version: r.Version, FiledByPatient: r.AccountID != "",
	}
	if clinic {
		out.PatientID = r.PatientID
	}
	out.ExtendedUntil, out.AnsweredAt = optionalTime(r.ExtendedUntil), optionalTime(r.AnsweredAt)
	return out
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func listJSON(all []Request, clinic bool) []requestJSON {
	out := make([]requestJSON, len(all))
	for i, r := range all {
		out[i] = toJSON(r, clinic)
	}
	return out
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	p, ok := h.patient(w, r)
	if !ok {
		return
	}
	all, err := h.svc.Mine(r.Context(), p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(all, false))
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	p, ok := h.patient(w, r)
	if !ok {
		return
	}
	var in struct {
		Kind           Kind   `json:"kind"`
		Target         Target `json:"target"`
		OrganizationID string `json:"organizationId"`
	}
	if !decode(w, r, &in) {
		return
	}
	out, err := h.svc.File(r.Context(), p, Filing{Kind: in.Kind, Target: in.Target, OrganizationID: in.OrganizationID})
	h.stored(w, r, http.StatusCreated, out, false, err)
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	p, ok := h.patient(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Withdraw(r.Context(), p, r.PathValue("id"), version)
	h.stored(w, r, http.StatusOK, out, false, err)
}

func (h *Handler) clinicList(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	all, err := h.svc.ClinicRequests(r.Context(), st)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, listJSON(all, true))
}

func (h *Handler) clinicRead(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	req, history, err := h.svc.ClinicRequest(r.Context(), st, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := toJSON(req, true)
	out.History = make([]changeJSON, len(history))
	for i, c := range history {
		out.History[i] = changeJSON{ChangedAt: c.ChangedAt.UTC(), Status: c.Status}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(req.Version)))
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if !st.Admin { // refused (and audited) by the service before the precondition and the body are read
		_, err := h.svc.Answer(r.Context(), st, r.PathValue("id"), Answer{})
		h.fail(w, r, err)
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	var in struct {
		Status Status     `json:"status"`
		Code   AnswerCode `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	out, err := h.svc.Answer(r.Context(), st, r.PathValue("id"), Answer{Status: in.Status, Code: in.Code, Version: version})
	h.stored(w, r, http.StatusOK, out, true, err)
}

func (h *Handler) extend(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if !st.Admin { // refused (and audited) by the service before the precondition and the body are read
		_, err := h.svc.Extend(r.Context(), st, r.PathValue("id"), "", 0)
		h.fail(w, r, err)
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	var in struct {
		Reason ExtensionReason `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	out, err := h.svc.Extend(r.Context(), st, r.PathValue("id"), in.Reason, version)
	h.stored(w, r, http.StatusOK, out, true, err)
}

func (h *Handler) fileOnBehalf(w http.ResponseWriter, r *http.Request) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if !st.Admin && !st.Reception { // refused (and audited) by the service before the body is read
		_, err := h.svc.FileOnBehalf(r.Context(), st, r.PathValue("id"), "")
		h.fail(w, r, err)
		return
	}
	var in struct {
		Kind Kind `json:"kind"`
	}
	if !decode(w, r, &in) {
		return
	}
	out, err := h.svc.FileOnBehalf(r.Context(), st, r.PathValue("id"), in.Kind)
	h.stored(w, r, http.StatusCreated, out, true, err)
}

// stored writes a filed or changed request with its version as the ETag, or the error.
func (h *Handler) stored(w http.ResponseWriter, r *http.Request, status int, out Request, clinic bool, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.Itoa(out.Version)))
	httpx.JSON(w, status, toJSON(out, clinic))
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Staff, bool) {
	st, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Staff{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Staff{}, false
	}
	return st, true
}

// patient is the signed-in patient; staff are refused on the patient's own routes.
func (h *Handler) patient(w http.ResponseWriter, r *http.Request) (Patient, bool) {
	st, ok := h.signedIn(w, r)
	if !ok {
		return Patient{}, false
	}
	if st.OrganizationID != "" {
		writeCode(w, http.StatusForbidden, "forbidden")
		return Patient{}, false
	}
	return Patient{AccountID: st.AccountID}, true
}

// ifMatch is the version an If-Match header names; without one (or one that names none) 428 is written.
func ifMatch(w http.ResponseWriter, r *http.Request) (int, bool) {
	tag, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	version, err := strconv.Atoi(tag)
	if !ok || err != nil {
		writeCode(w, http.StatusPreconditionRequired, "if_match_required")
		return 0, false
	}
	return version, true
}

// decode reads the JSON body into v; a body that is not one is 400 invalid_data_request.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(v); err != nil {
		writeCode(w, http.StatusBadRequest, "invalid_data_request")
		return false
	}
	return true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid_data_request")
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrClosed):
		writeCode(w, http.StatusConflict, "data_request_closed")
	case errors.Is(err, ErrAlreadyExtended):
		writeCode(w, http.StatusConflict, "already_extended")
	case errors.Is(err, ErrOverdue):
		writeCode(w, http.StatusConflict, "request_overdue")
	case errors.Is(err, ErrExportInProgress):
		writeCode(w, http.StatusConflict, "export_in_progress")
	case errors.Is(err, ErrVersionConflict):
		writeCode(w, http.StatusPreconditionFailed, "version_conflict")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
