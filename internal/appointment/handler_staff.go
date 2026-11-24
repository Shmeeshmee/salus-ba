package appointment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// dateLayout is a calendar date, as the desk's day and a birth date travel.
const dateLayout = "2006-01-02"

// StaffAppointments is what the staff handler calls (Service).
type StaffAppointments interface {
	BookForPatient(ctx context.Context, c Caller, b StaffBooking) (Appointment, error)
	CancelForPatient(ctx context.Context, c Caller, id string, version int32) (Appointment, error)
	MoveForPatient(ctx context.Context, c Caller, id, slotID string, version int32) (Appointment, error)
	Arrive(ctx context.Context, c Caller, id string) (Appointment, error)
	DeskDay(ctx context.Context, c Caller, day time.Time) (Desk, error)
}

// StaffHandler serves the bookings staff make for patients and the desk's day in the JSON API (package F2 of
// docs/clinic-cms). Errors as HTTPHandler's.
type StaffHandler struct {
	svc StaffAppointments
	HTTPHandler
}

// NewStaffHandler returns the handler on svc. The desk's day without a date is today in the clinic's zone, which the
// service knows.
func NewStaffHandler(svc StaffAppointments, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *StaffHandler {
	return &StaffHandler{svc: svc, HTTPHandler: HTTPHandler{caller: caller, log: log}}
}

// Routes declares the staff's appointment routes on mux.
func (h *StaffHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /clinic/appointments", h.book)
	mux.HandleFunc("POST /clinic/appointments/{id}/cancel", h.cancel)
	mux.HandleFunc("POST /clinic/appointments/{id}/reschedule", h.move)
	mux.HandleFunc("POST /clinic/appointments/{id}/arrive", h.arrive)
	mux.HandleFunc("GET /clinic/desk/today", h.desk)
}

func (h *StaffHandler) book(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var req struct {
		PatientID string `json:"patientId"`
		SlotID    string `json:"slotId"`
		ServiceID string `json:"serviceId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.PatientID == "" || req.SlotID == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.BookForPatient(r.Context(), c, StaffBooking(req))
	h.reply(w, r, http.StatusCreated, out, err)
}

func (h *StaffHandler) cancel(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	out, err := h.svc.CancelForPatient(r.Context(), c, r.PathValue("id"), version)
	h.reply(w, r, http.StatusOK, out, err)
}

// move is the desk's move of a visit (docs/clinic-cms WAVE4 M1): the slot from the body, the version from If-Match.
func (h *StaffHandler) move(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	var req struct {
		SlotID string `json:"slotId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.SlotID == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	out, err := h.svc.MoveForPatient(r.Context(), c, r.PathValue("id"), req.SlotID, version)
	h.reply(w, r, http.StatusOK, out, err)
}

func (h *StaffHandler) arrive(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := h.svc.Arrive(r.Context(), c, r.PathValue("id"))
	h.reply(w, r, http.StatusOK, out, err)
}

// deskPatientJSON is the ClinicPatient schema of api/openapi.yaml.
type deskPatientJSON struct {
	LastUpdated        time.Time `json:"lastUpdated"`
	ID                 string    `json:"id"`
	GivenName          string    `json:"givenName"`
	FamilyName         string    `json:"familyName"`
	BirthDate          string    `json:"birthDate"`
	Phone              string    `json:"phone,omitempty"`
	RegisteredByClinic bool      `json:"registeredByClinic"`
	ConsentActive      bool      `json:"consentActive"`
	VersionID          int32     `json:"versionId"`
}

// deskVisitJSON is the DeskVisit schema.
type deskVisitJSON struct {
	ServiceName string            `json:"serviceName,omitempty"`
	ReminderOff string            `json:"reminderOff,omitempty"`
	Patient     deskPatientJSON   `json:"patient"`
	Receipts    []deskReceiptJSON `json:"receipts"`
	Appointment appointmentJSON   `json:"appointment"`
	Reminder    bool              `json:"reminder"`
}

// deskReceiptJSON is the DeskReceipt schema.
type deskReceiptJSON struct {
	ID       string `json:"id"`
	Number   int64  `json:"number"`
	Reversed bool   `json:"reversed"`
}

func (h *StaffHandler) desk(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var day time.Time // none: today in the clinic's zone, the service's to say
	if d := r.URL.Query().Get("date"); d != "" {
		var err error
		if day, err = time.Parse(dateLayout, d); err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	desk, err := h.svc.DeskDay(r.Context(), c, day)
	if err != nil {
		h.writeStaffError(w, r, err)
		return
	}
	out := struct {
		Date   string          `json:"date"`
		Visits []deskVisitJSON `json:"visits"`
	}{Date: desk.Day.Format(dateLayout), Visits: make([]deskVisitJSON, 0, len(desk.Visits))}
	for _, v := range desk.Visits {
		p := v.Patient
		receipts := make([]deskReceiptJSON, 0, len(v.Receipts))
		for _, rc := range v.Receipts {
			receipts = append(receipts, deskReceiptJSON(rc))
		}
		out.Visits = append(out.Visits, deskVisitJSON{ServiceName: v.ServiceName, Receipts: receipts, Appointment: toJSON(v.Appointment), Reminder: v.Reminder, ReminderOff: v.ReminderOff, Patient: deskPatientJSON{
			ID: p.ID, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate.Format(dateLayout), Phone: p.Phone,
			RegisteredByClinic: p.RegisteredBy != "", ConsentActive: p.ConsentActive, VersionID: p.VersionID, LastUpdated: p.LastUpdated.UTC(),
		}})
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

// ifMatch is the version the If-Match header names, or answers 400 if_match_required.
func ifMatch(w http.ResponseWriter, r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	n, err := strconv.ParseInt(v, 10, 32)
	if !ok || err != nil {
		writeCode(w, http.StatusBadRequest, "if_match_required")
		return 0, false
	}
	return int32(n), true
}

// reply sends out with its ETag, never to be cached, or the error.
func (h *StaffHandler) reply(w http.ResponseWriter, r *http.Request, status int, out Appointment, err error) {
	if err != nil {
		h.writeStaffError(w, r, err)
		return
	}
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(out.VersionID), 10)))
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, toJSON(out))
}

// writeStaffError answers the staff routes' own codes (api/openapi.yaml), then the appointment routes'.
func (h *StaffHandler) writeStaffError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrConsentRequired):
		writeCode(w, http.StatusForbidden, "consent_required")
	case errors.Is(err, ErrVersionConflict):
		writeCode(w, http.StatusConflict, "version_conflict")
	case errors.Is(err, ErrNotInPerson):
		writeCode(w, http.StatusConflict, "not_in_person")
	default:
		h.writeError(w, r, err)
	}
}
