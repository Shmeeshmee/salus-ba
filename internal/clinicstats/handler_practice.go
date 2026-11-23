package clinicstats

import (
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// practiceRowJSON is an item of the PracticeReport schema.
type practiceRowJSON struct {
	PractitionerRole string             `json:"practitionerRole"`
	Mode             schedule.VisitMode `json:"visitMode"`
	MinutesOffered   int64              `json:"minutesOffered"`
	Visits           int64              `json:"visits"`
	ReceiptedMinor   int64              `json:"receiptedMinor"`
}

// practiceJSON is the PracticeReport schema. Prices are in BAM only (docs/clinic-cms/PROGRESS.md, wave 2 decisions).
type practiceJSON struct {
	From     time.Time         `json:"from"`
	To       time.Time         `json:"to"`
	Currency string            `json:"currency"`
	Rows     []practiceRowJSON `json:"rows"`
}

// practice answers GET /clinic/reports/practice?from=&to= (RFC 3339 times of a half-open range, as the receipts
// report's).
func (h *HTTPHandler) practice(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	from, errFrom := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, errTo := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if errFrom != nil || errTo != nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	rows, err := h.svc.Practice(r.Context(), c, PracticeQuery{From: from, To: to})
	if h.failed(w, r, err) {
		return
	}
	out := practiceJSON{From: from.UTC(), To: to.UTC(), Currency: "BAM", Rows: make([]practiceRowJSON, len(rows))}
	for i, p := range rows {
		out.Rows[i] = practiceRowJSON{
			PractitionerRole: p.PractitionerRoleID, Mode: p.Mode, MinutesOffered: p.MinutesOffered, Visits: p.Visits, ReceiptedMinor: p.ReceiptedMinor,
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.JSON(w, http.StatusOK, out)
}
