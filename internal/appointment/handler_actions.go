package appointment

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// SelfService is what the actions handler calls (Service).
type SelfService interface {
	Confirm(ctx context.Context, c Caller, id string) (Appointment, error)
	Reschedule(ctx context.Context, c Caller, id, slotID string) (Appointment, error)
	Actions(ctx context.Context, c Caller, id string) (Actions, error)
}

// ActionsHandler serves the patient's confirmation and reschedule of an appointment and what they may do with it
// (api/openapi.yaml, SAL-34), beside HTTPHandler and with its error codes.
type ActionsHandler struct {
	svc SelfService
	h   *HTTPHandler // for signedIn, answer and writeError
}

// NewActionsHandler returns the handler on svc, with the caller of the request as HTTPHandler's.
func NewActionsHandler(svc SelfService, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *ActionsHandler {
	return &ActionsHandler{svc: svc, h: &HTTPHandler{caller: caller, log: log}}
}

// Routes declares the routes on mux.
func (a *ActionsHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /appointments/{id}/confirm", a.confirm)
	mux.HandleFunc("POST /appointments/{id}/reschedule", a.reschedule)
	mux.HandleFunc("GET /appointments/{id}/actions", a.actions)
}

// actionsJSON is the AppointmentActions schema.
type actionsJSON struct {
	CancelUntil     *time.Time `json:"cancel_until,omitempty"`
	RescheduleUntil *time.Time `json:"reschedule_until,omitempty"`
	Confirm         bool       `json:"confirm"`
	Cancel          bool       `json:"cancel"`
	Reschedule      bool       `json:"reschedule"`
	ReschedulesLeft int        `json:"reschedules_left"`
}

func (a *ActionsHandler) confirm(w http.ResponseWriter, r *http.Request) {
	c, ok := a.h.signedIn(w, r)
	if !ok {
		return
	}
	out, err := a.svc.Confirm(r.Context(), c, r.PathValue("id"))
	a.h.answer(w, r, http.StatusOK, out, err)
}

func (a *ActionsHandler) reschedule(w http.ResponseWriter, r *http.Request) {
	c, ok := a.h.signedIn(w, r)
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
	out, err := a.svc.Reschedule(r.Context(), c, r.PathValue("id"), req.SlotID)
	a.h.answer(w, r, http.StatusOK, out, err)
}

func (a *ActionsHandler) actions(w http.ResponseWriter, r *http.Request) {
	c, ok := a.h.signedIn(w, r)
	if !ok {
		return
	}
	got, err := a.svc.Actions(r.Context(), c, r.PathValue("id"))
	if err != nil {
		a.h.writeError(w, r, err)
		return
	}
	out := actionsJSON{Confirm: got.Confirm, Cancel: got.Cancel, Reschedule: got.Reschedule, ReschedulesLeft: got.ReschedulesLeft}
	if got.Cancel {
		at := got.CancelUntil.UTC()
		out.CancelUntil = &at
	}
	if got.Reschedule {
		at := got.RescheduleUntil.UTC()
		out.RescheduleUntil = &at
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}
