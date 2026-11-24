package patient

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxImportBody bounds an uploaded patient list: 5 000 rows fit many times over.
const maxImportBody = 2 << 20

// importedPatientJSON is the ImportedPatient schema: the profile of a valid row, with its id once saved.
type importedPatientJSON struct {
	ID         string `json:"id,omitempty"`
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	BirthDate  string `json:"birthDate"`
	Phone      string `json:"phone,omitempty"`
}

// importRowJSON is the PatientImportRow schema: a row with errors has no patient.
type importRowJSON struct {
	Patient   *importedPatientJSON `json:"patient,omitempty"`
	Errors    []string             `json:"errors"`
	Line      int                  `json:"line"`
	Duplicate bool                 `json:"duplicate"`
}

// importJSON is the PatientImport schema.
type importJSON struct {
	Rows  []importRowJSON `json:"rows"`
	Saved bool            `json:"saved"`
}

// importFile serves POST /clinic/patients/import: the multipart part "file", previewed or, with save=true, saved.
func (h *ClinicHandler) importFile(w http.ResponseWriter, r *http.Request) {
	c, ok := h.staff(w, r)
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
	_, data, status := httpx.FormFile(w, r, "file", maxImportBody)
	if status != 0 {
		code := "invalid_request"
		if status == http.StatusRequestEntityTooLarge {
			code = "too_large"
		}
		writeCode(w, status, code)
		return
	}
	rows, err := h.svc.Import(r.Context(), c, data, save)
	if errors.Is(err, ErrInvalidImport) {
		writeCode(w, http.StatusBadRequest, "invalid_import")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := importJSON{Rows: make([]importRowJSON, len(rows)), Saved: save}
	for i, row := range rows {
		out.Rows[i] = importRowJSON{Line: row.Line, Errors: row.Errors, Duplicate: row.Duplicate}
		if len(row.Errors) > 0 {
			out.Saved = false
			continue
		}
		p := row.Patient
		out.Rows[i].Errors = []string{}
		out.Rows[i].Patient = &importedPatientJSON{ID: p.ID, GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate.Format(dateLayout), Phone: p.Phone}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}
