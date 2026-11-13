package organization

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// AppointmentRules is what the appointment settings' handler calls (Appointments).
type AppointmentRules interface {
	Settings(ctx context.Context, c Caller) (AppointmentSettings, error)
	SetSettings(ctx context.Context, c Caller, in AppointmentSettings) (AppointmentSettings, error)
}

// AppointmentsHandler serves the clinic's appointment settings (api/openapi.yaml, tag appointments). Errors are
// {"error": "<code>"}.
type AppointmentsHandler struct {
	svc    AppointmentRules
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewAppointmentsHandler returns the handler on svc; caller resolves the signed-in session (with whether it is the
// clinic's admin).
func NewAppointmentsHandler(svc AppointmentRules, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *AppointmentsHandler {
	return &AppointmentsHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *AppointmentsHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/appointment-settings", h.settings)
	mux.HandleFunc("PUT /clinic/appointment-settings", h.setSettings)
}

// appointmentSettingsJSON is the AppointmentSettings schema as a request: every field required.
type appointmentSettingsJSON struct {
	ReminderOffsetsMinutes *[]int `json:"reminder_offsets_minutes"`
	SMSReminders           *bool  `json:"sms_reminders"`
	RescheduleNoticeHours  *int   `json:"reschedule_notice_hours"`
	RescheduleMax          *int   `json:"reschedule_max"`
	WaitlistEnabled        *bool  `json:"waitlist_enabled"`
	WaitlistHoldMinutes    *int   `json:"waitlist_hold_minutes"`
	NoShowFlagThreshold    *int   `json:"noshow_flag_threshold"`
	NoShowFlagWindowDays   *int   `json:"noshow_flag_window_days"`
}

func (in appointmentSettingsJSON) settings() (AppointmentSettings, bool) {
	if in.ReminderOffsetsMinutes == nil || in.SMSReminders == nil || in.RescheduleNoticeHours == nil || in.RescheduleMax == nil ||
		in.WaitlistEnabled == nil || in.WaitlistHoldMinutes == nil || in.NoShowFlagThreshold == nil || in.NoShowFlagWindowDays == nil {
		return AppointmentSettings{}, false
	}
	return AppointmentSettings{
		ReminderOffsetsMinutes: *in.ReminderOffsetsMinutes, SMSReminders: *in.SMSReminders, RescheduleNoticeHours: *in.RescheduleNoticeHours,
		RescheduleMax: *in.RescheduleMax, WaitlistEnabled: *in.WaitlistEnabled, WaitlistHoldMinutes: *in.WaitlistHoldMinutes,
		NoShowFlagThreshold: *in.NoShowFlagThreshold, NoShowFlagWindowDays: *in.NoShowFlagWindowDays,
	}, true
}

func (h *AppointmentsHandler) settings(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Settings(r.Context(), c)
	h.write(w, r, s, err)
}

func (h *AppointmentsHandler) setSettings(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	var in appointmentSettingsJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSettingsBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	set, ok := in.settings()
	version, okVersion := settingsVersion(r)
	if !ok || !okVersion {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	set.Version = version
	s, err := h.svc.SetSettings(r.Context(), c, set)
	h.write(w, r, s, err)
}

func (h *AppointmentsHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
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

func (h *AppointmentsHandler) write(w http.ResponseWriter, r *http.Request, v AppointmentSettings, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidAppointmentSettings):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, ErrSMSUnavailable):
		writeCode(w, http.StatusConflict, "sms_unavailable")
	case errors.Is(err, ErrSettingsChanged):
		writeCode(w, http.StatusPreconditionFailed, "version_conflict")
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	default:
		if v.ReminderOffsetsMinutes == nil {
			v.ReminderOffsetsMinutes = []int{}
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fhir.ETag(strconv.Itoa(v.Version)))
		httpx.JSON(w, http.StatusOK, v)
	}
}

// settingsVersion is the version the If-Match header names (W/"<n>" or "<n>", 0 for none stored), and false for
// none or another value.
func settingsVersion(r *http.Request) (int, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 32)
	return int(n), err == nil && n >= 0
}
