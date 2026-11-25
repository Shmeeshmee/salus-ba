package location

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

// maxRoomBody bounds a SaveRoomRequest or SaveDeviceRequest: a name and a flag.
const maxRoomBody = 4 << 10

// RoomService is what the room handler calls (Service).
type RoomService interface {
	Rooms(ctx context.Context, st Staff, locationID string) ([]Room, error)
	Devices(ctx context.Context, st Staff, locationID string) ([]Device, error)
	SaveRoom(ctx context.Context, st Staff, r Room) (Room, error)
	SaveDevice(ctx context.Context, st Staff, d Device) (Device, error)
}

// RoomHandler serves a location's rooms and devices (api/openapi.yaml, tag locations; RM1). Errors are
// {"error": "<code>"}.
type RoomHandler struct {
	svc    RoomService
	caller func(ctx context.Context) (Staff, bool, error)
	log    *slog.Logger
}

// NewRoomHandler returns the handler on svc; caller resolves the signed-in staff (with whether they are the clinic's
// admin).
func NewRoomHandler(svc RoomService, caller func(ctx context.Context) (Staff, bool, error), log *slog.Logger) *RoomHandler {
	return &RoomHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *RoomHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/locations/{id}/rooms", h.list(false))
	mux.HandleFunc("GET /clinic/locations/{id}/devices", h.list(true))
	mux.HandleFunc("POST /clinic/locations/{id}/rooms", h.save(false, false))
	mux.HandleFunc("POST /clinic/locations/{id}/devices", h.save(true, false))
	mux.HandleFunc("PUT /clinic/rooms/{id}", h.save(false, true))
	mux.HandleFunc("PUT /clinic/devices/{id}", h.save(true, true))
}

// saveRoomJSON is the SaveRoomRequest and SaveDeviceRequest schema: the pointers tell a required field left out.
type saveRoomJSON struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

func (h *RoomHandler) list(devices bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, ok := h.signedIn(w, r)
		if !ok {
			return
		}
		var out []Room
		var err error
		if devices {
			var all []Device
			all, err = h.svc.Devices(r.Context(), st, r.PathValue("id"))
			for _, d := range all {
				out = append(out, Room(d))
			}
		} else {
			out, err = h.svc.Rooms(r.Context(), st, r.PathValue("id"))
		}
		if err != nil {
			h.fail(w, r, err)
			return
		}
		if out == nil {
			out = []Room{}
		}
		w.Header().Set("Cache-Control", "no-store")
		httpx.JSON(w, http.StatusOK, out) // a Device has a Room's fields and JSON names
	}
}

// save is POST on a location (update false: a new room or device of {id}) and PUT on a room or device ({id}, at the
// version of If-Match).
func (h *RoomHandler) save(device, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, ok := h.signedIn(w, r)
		if !ok {
			return
		}
		in, ok := decodeRoom(w, r)
		if !ok {
			writeRoomCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
		status := http.StatusCreated
		if in.LocationID = r.PathValue("id"); update {
			tag, okTag := fhir.IfMatch(r.Header.Get("If-Match"))
			version, err := strconv.Atoi(tag)
			if !okTag || err != nil {
				writeRoomCode(w, http.StatusBadRequest, "invalid_request")
				return
			}
			in.ID, in.LocationID, in.VersionID, status = r.PathValue("id"), "", version, http.StatusOK
		}
		var out Room
		var err error
		if device {
			var d Device
			d, err = h.svc.SaveDevice(r.Context(), st, Device(in))
			out = Room(d)
		} else {
			out, err = h.svc.SaveRoom(r.Context(), st, in)
		}
		if err != nil {
			h.fail(w, r, err)
			return
		}
		w.Header().Set("ETag", fhir.ETag(strconv.Itoa(out.VersionID)))
		httpx.JSON(w, status, out)
	}
}

func (h *RoomHandler) signedIn(w http.ResponseWriter, r *http.Request) (Staff, bool) {
	st, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Staff{}, false
	case !ok:
		writeRoomCode(w, http.StatusUnauthorized, "unauthenticated")
		return Staff{}, false
	}
	return st, true
}

func (h *RoomHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeRoomCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeRoomCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalid):
		writeRoomCode(w, http.StatusBadRequest, "invalid_location")
	case errors.Is(err, ErrVersionConflict):
		writeRoomCode(w, http.StatusPreconditionFailed, "version_conflict")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

// decodeRoom reads a SaveRoomRequest, and is false for a body that is not one.
func decodeRoom(w http.ResponseWriter, r *http.Request) (Room, bool) {
	var in saveRoomJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRoomBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || in.Name == nil || in.Active == nil {
		return Room{}, false
	}
	return Room{Name: *in.Name, Active: *in.Active}, true
}

func writeRoomCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
