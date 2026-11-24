package chargeitemdefinition

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxImportBody bounds an uploaded price list: a thousand rows fit many times over.
const maxImportBody = 2 << 20

// importServiceJSON is the SaveServiceRequest schema of a valid row.
type importServiceJSON struct {
	VATRateBP       *int32               `json:"vatRateBp,omitempty"`
	Name            string               `json:"name"`
	Code            string               `json:"code,omitempty"`
	Currency        string               `json:"currency"`
	Modes           []schedule.VisitMode `json:"modes"`
	Translations    []translationJSON    `json:"translations"`
	PriceMinor      int64                `json:"priceMinor"`
	DurationMinutes int32                `json:"durationMinutes,omitempty"`
	Active          bool                 `json:"active"`
}

// importRowJSON is the ServiceImportRow schema: a row with errors has no service.
type importRowJSON struct {
	Service *importServiceJSON `json:"service,omitempty"`
	Errors  []string           `json:"errors"`
	Line    int                `json:"line"`
}

// importJSON is the ServiceImport schema.
type importJSON struct {
	Rows  []importRowJSON `json:"rows"`
	Saved bool            `json:"saved"`
}

// importFile serves POST /clinic/services/import: the multipart part "file", previewed or, with save=true, saved.
func (h *Handler) importFile(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	save := false
	if v := r.URL.Query().Get("save"); v != "" {
		var err error
		if save, err = strconv.ParseBool(v); err != nil {
			writeCode(w, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	file, data, status := httpx.FormFile(w, r, "file", maxImportBody)
	if status != 0 {
		code := "invalid_request"
		if status == http.StatusRequestEntityTooLarge {
			code = "too_large"
		}
		writeCode(w, status, code)
		return
	}
	rows, err := h.svc.Import(r.Context(), c, ImportFile{Name: file, Data: data}, save)
	if errors.Is(err, ErrInvalidImport) {
		writeCode(w, http.StatusBadRequest, "invalid_import")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := importJSON{Rows: make([]importRowJSON, len(rows)), Saved: save}
	for i, row := range rows {
		out.Rows[i] = importRowJSON{Line: row.Line, Errors: row.Errors}
		if len(row.Errors) > 0 {
			out.Saved = false
			continue
		}
		d := row.Service
		out.Rows[i].Errors = []string{}
		out.Rows[i].Service = &importServiceJSON{
			VATRateBP: d.VATRateBP, Name: d.Name, Code: d.Code, Currency: d.Currency, Modes: d.Modes,
			Translations: []translationJSON{}, PriceMinor: d.PriceMinor, DurationMinutes: d.DurationMinutes, Active: d.Active,
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}
