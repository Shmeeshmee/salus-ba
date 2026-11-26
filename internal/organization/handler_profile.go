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

// maxProfileBody bounds a profile request: ten descriptions of 2000 characters, up to four bytes each, and the rest.
const maxProfileBody = 96 << 10

// Profiles is what the profile's handler calls (Service).
type Profiles interface {
	Profile(ctx context.Context, id string, served Served) (Profile, error)
	ClinicProfile(ctx context.Context, c Caller) (Profile, error)
	SaveProfile(ctx context.Context, c Caller, p Profile, version int32) (Profile, error)
}

// ProfileHandler serves the clinic's profile for patients abroad (api/openapi.yaml, tag directory; package X2 of
// docs/clinic-cms). Errors are {"error": "<code>"}.
type ProfileHandler struct {
	svc    Profiles
	caller func(ctx context.Context) (Caller, bool, error)
	served Served // the clinics the signed-in caller keeps once their membership stops; nil: none
	log    *slog.Logger
}

// NewProfileHandler returns the handler on svc; caller resolves the signed-in session.
func NewProfileHandler(svc Profiles, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *ProfileHandler {
	return &ProfileHandler{svc: svc, caller: caller, log: log}
}

// WithServed returns h with the public profile read for the clinics served gives too once their membership stops.
func (h *ProfileHandler) WithServed(served Served) *ProfileHandler {
	out := *h
	out.served = served
	return &out
}

// Routes declares the routes on mux.
func (h *ProfileHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/profile", h.clinic)
	mux.HandleFunc("PUT /clinic/profile", h.save)
	mux.HandleFunc("GET /organizations/{id}/profile", h.public)
}

// profileJSON is the ClinicProfile schema; every field is required in a request.
type profileJSON struct {
	Languages             *[]string          `json:"languages"`
	Description           *string            `json:"description"`
	Translations          *[]translationJSON `json:"translations"`
	WelcomesInternational *bool              `json:"welcomesInternational"`
}

type translationJSON struct {
	Language    string `json:"language"`
	Description string `json:"description"`
}

func (h *ProfileHandler) clinic(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	if err != nil || !ok {
		h.write(w, r, Profile{}, err, !ok, true)
		return
	}
	p, err := h.svc.ClinicProfile(r.Context(), c)
	h.write(w, r, p, err, false, true)
}

func (h *ProfileHandler) save(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	if err != nil || !ok {
		h.write(w, r, Profile{}, err, !ok, true)
		return
	}
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	version, perr := strconv.ParseInt(v, 10, 32)
	if !ok || perr != nil {
		writeCode(w, http.StatusBadRequest, "if_match_required")
		return
	}
	var in profileJSON
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProfileBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF ||
		in.Languages == nil || in.Description == nil || in.Translations == nil || in.WelcomesInternational == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	p := Profile{Languages: *in.Languages, Description: *in.Description, WelcomesInternational: *in.WelcomesInternational,
		Translations: make([]Translation, 0, len(*in.Translations))}
	for _, t := range *in.Translations {
		p.Translations = append(p.Translations, Translation(t))
	}
	saved, err := h.svc.SaveProfile(r.Context(), c, p, int32(version))
	h.write(w, r, saved, err, false, true)
}

func (h *ProfileHandler) public(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Profile(r.Context(), r.PathValue("id"), h.served)
	h.write(w, r, p, err, false, false)
}

// write answers p (with its version as the ETag when versioned), or the code of err, or 401 for a caller who is
// signed out.
func (h *ProfileHandler) write(w http.ResponseWriter, r *http.Request, p Profile, err error, signedOut, versioned bool) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidProfile):
		writeCode(w, http.StatusBadRequest, "invalid_profile")
	case errors.Is(err, ErrConflict):
		writeCode(w, http.StatusConflict, "version_conflict")
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case signedOut:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	default:
		langs, ts := p.Languages, make([]translationJSON, 0, len(p.Translations))
		if langs == nil {
			langs = []string{}
		}
		for _, t := range p.Translations {
			ts = append(ts, translationJSON(t))
		}
		w.Header().Set("Cache-Control", "no-store")
		if versioned {
			w.Header().Set("ETag", fhir.ETag(strconv.Itoa(int(p.VersionID))))
		}
		httpx.JSON(w, http.StatusOK, profileJSON{Languages: &langs, Description: &p.Description, Translations: &ts, WelcomesInternational: &p.WelcomesInternational})
	}
}
