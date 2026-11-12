package trends

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// View is what the Handler calls: Service.
type View interface {
	Overview(ctx context.Context, c Caller) (Overview, error)
	Cells(ctx context.Context, c Caller, q Query) (Breakdown, error)
	Heatmap(ctx context.Context, c Caller, quarter, region string) ([]HeatCell, error)
	Rises(ctx context.Context, c Caller, region string, weeks int) ([]Flag, error)
}

// Handler serves the All of Salus view of the JSON API (docs/plans/analytics.md §8). Its routes are mounted only
// when ANALYTICS_PLATFORM_ENABLED is on (§2a).
type Handler struct {
	svc    View
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the Handler; caller gives who is signed in, false for nobody.
func NewHandler(svc View, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the platform view's routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /analytics/platform/months", h.months)
	mux.HandleFunc("GET /analytics/platform/cells", h.cells)
	mux.HandleFunc("GET /analytics/platform/heatmap", h.heatmap)
	mux.HandleFunc("GET /analytics/platform/rises", h.rises)
}

// The response shapes (api/openapi.yaml): codes and counts only, no identifier (A1).
type (
	overviewJSON struct {
		Months   []string `json:"months"`
		Quarters []string `json:"quarters"`
		MinCount int      `json:"minCount"`
	}
	countJSON struct {
		Count *int   `json:"count"`
		Key   string `json:"key"`
	}
	breakdownJSON struct {
		Rows    []countJSON `json:"rows"`
		Partial bool        `json:"partial"`
	}
	heatCellJSON struct {
		Count   *int `json:"count"`
		Weekday int  `json:"weekday"`
		Hour    int  `json:"hour"`
	}
	riseJSON struct {
		Week   string `json:"week"`
		Region string `json:"region"`
		Group  string `json:"group"`
	}
)

// cellDimensions are the query parameters that fix a dimension of a cells query.
var cellDimensions = []string{DimNameEntity, DimNameRegion, DimNameCity, DimNameGroup, DimNameSubgroup, DimNameCode, DimNameSpecialty}

func (h *Handler) months(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	o, err := h.svc.Overview(r.Context(), c)
	if h.failed(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, overviewJSON{Months: nonNil(o.Months), Quarters: nonNil(o.Quarters), MinCount: o.MinCount})
}

// cells answers GET /analytics/platform/cells?table=&from=&to=&by=&<dimension>=…; any other parameter, or one given
// twice, is a 400.
func (h *Handler) cells(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v, err := url.ParseQuery(r.URL.RawQuery)
	q := Query{Table: v.Get("table"), From: v.Get("from"), To: v.Get("to"), By: v.Get("by"), Fixed: map[string]string{}}
	for _, d := range cellDimensions {
		if v.Has(d) {
			q.Fixed[d] = v.Get(d)
		}
	}
	if err != nil || !only(v, append([]string{"table", "from", "to", "by"}, cellDimensions...)...) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	b, err := h.svc.Cells(r.Context(), c, q)
	if h.failed(w, r, err) {
		return
	}
	out := breakdownJSON{Rows: make([]countJSON, len(b.Rows)), Partial: b.Partial}
	for i, row := range b.Rows {
		out.Rows[i] = countJSON(row)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) heatmap(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || !only(v, "quarter", "region") {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	cells, err := h.svc.Heatmap(r.Context(), c, v.Get("quarter"), v.Get("region"))
	if h.failed(w, r, err) {
		return
	}
	out := make([]heatCellJSON, len(cells))
	for i, cell := range cells {
		out[i] = heatCellJSON(cell)
	}
	httpx.JSON(w, http.StatusOK, map[string][]heatCellJSON{"cells": out})
}

func (h *Handler) rises(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	v, err := url.ParseQuery(r.URL.RawQuery)
	weeks := 8
	if v.Has("weeks") {
		if weeks, err = strconv.Atoi(v.Get("weeks")); err != nil {
			weeks = 0
		}
	}
	if err != nil || !only(v, "region", "weeks") {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	flags, err := h.svc.Rises(r.Context(), c, v.Get("region"), weeks)
	if h.failed(w, r, err) {
		return
	}
	out := make([]riseJSON, len(flags))
	for i, f := range flags {
		out[i] = riseJSON(f)
	}
	httpx.JSON(w, http.StatusOK, map[string][]riseJSON{"rises": out})
}

// only reports whether v holds no parameter but names, each at most once.
func only(v url.Values, names ...string) bool {
	allowed := map[string]bool{}
	for _, n := range names {
		allowed[n] = true
	}
	for k, vs := range v {
		if !allowed[k] || len(vs) != 1 {
			return false
		}
	}
	return true
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	}
	return c, ok && err == nil
}

// failed writes the error's status and code and reports whether there was one.
func (h *Handler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotContributing):
		writeCode(w, http.StatusForbidden, "analytics_not_contributing")
	case errors.Is(err, ErrInvalidQuery):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
	return true
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
