package clinicexport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// since is before the first receipt any clinic issued: the receipts' period starts there.
var since = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Build builds the archive of the queued export args.ExportID: the clinic's data read as the admin who asked (each
// read audited by the service that owns it), written as CSV files into one zip, encrypted with a data key of its own
// and stored, then the export is marked ready. An export that is not queued (built, failed, gone) is left alone.
func (s *Service) Build(ctx context.Context, args BuildArgs) error {
	var x Export
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, args.OrganizationID); err != nil {
			return err
		}
		var err error
		x, err = s.p.Repo.ByID(ctx, args.ExportID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("clinicexport: build: %w", err)
	case x.Status != StatusQueued:
		return nil
	}
	if ok, err := s.p.Admins.IsClinicAdmin(ctx, auth.Session{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID}); err != nil {
		return fmt.Errorf("clinicexport: build: %w", err)
	} else if !ok {
		return s.refuseBuild(ctx, x)
	}
	archive, err := s.archive(ctx, x)
	if err != nil {
		return fmt.Errorf("clinicexport: build: %w", err)
	}
	if err := s.seal(ctx, &x, archive); err != nil {
		return fmt.Errorf("clinicexport: build: %w", err)
	}
	x.ReadyAt = s.p.Clock.Now()
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, x.OrganizationID); err != nil {
			return err
		}
		ok, err := s.p.Repo.MarkReady(ctx, x)
		if err != nil || !ok {
			return err
		}
		return s.p.Audit.Record(ctx, jobEvent(x.OrganizationID, auditevent.CodeUpdate, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return fmt.Errorf("clinicexport: build: %w", err)
	}
	return nil
}

// refuseBuild marks the export x failed without reading anything of the clinic: the admin who asked is no longer its
// admin. Audited as the job's refused write.
func (s *Service) refuseBuild(ctx context.Context, x Export) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, x.OrganizationID); err != nil {
			return err
		}
		ok, err := s.p.Repo.MarkFailed(ctx, x.ID)
		if err != nil || !ok {
			return err
		}
		return s.p.Audit.Record(ctx, jobEvent(x.OrganizationID, auditevent.CodeUpdate, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("clinicexport: build: %w", err)
	}
	return nil
}

// seal encrypts the archive under a new data key bound to the export's id, stores it outside any transaction, and
// sets what the row keeps of it on x.
func (s *Service) seal(ctx context.Context, x *Export, archive []byte) error {
	dek, err := envelope.NewDataKey()
	if err != nil {
		return err
	}
	if x.WrappedKey, err = s.p.KEK.Wrap(dek, x.ID); err != nil {
		return err
	}
	x.KEKID, x.Size = s.p.KEK.ID(), int64(len(archive))
	var sealed bytes.Buffer
	w, err := envelope.NewWriter(&sealed, dek, x.ID)
	if err != nil {
		return err
	}
	if _, err := w.Write(archive); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return s.p.Store.Put(ctx, ObjectKey(x.ID), &sealed, int64(sealed.Len()))
}

// archive reads the clinic's data and writes it as one zip: the CSV files, each registered patient's record as a FHIR
// Bundle (patients/<id>.json, the same file the hand-over gives), and manifest.txt last, which says what is in it and
// what is not.
func (s *Service) archive(ctx context.Context, x Export) ([]byte, error) {
	files := []struct {
		read func(context.Context, Export) ([][]string, error)
		name string
	}{
		{name: "prices.csv", read: s.prices},
		{name: "staff.csv", read: s.staff},
		{name: "appointments.csv", read: s.appointments},
		{name: "receipts.csv", read: s.receipts},
		{name: "quotes.csv", read: s.quotes},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		rows, err := f.read(ctx, x)
		if err != nil {
			return nil, err
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: x.RequestedAt})
		if err != nil {
			return nil, err
		}
		cw := csv.NewWriter(w)
		for _, row := range rows {
			if err := cw.Write(cells(row)); err != nil {
				return nil, err
			}
		}
		if cw.Flush(); cw.Error() != nil {
			return nil, cw.Error()
		}
	}
	patients, err := s.patients(ctx, x, zw)
	if err != nil {
		return nil, err
	}
	restricted, err := s.p.Patients.RestrictedRecords(ctx, patient.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true})
	if err != nil {
		return nil, err
	}
	if err := write(zw, x, "manifest.txt", []byte(manifest(patients, restricted))); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// patients writes the record of each patient the clinic registered into zw, read through the hand-over's builder as
// the admin who asked, for the export's purpose; it returns how many.
func (s *Service) patients(ctx context.Context, x Export, zw *zip.Writer) (int, error) {
	ids, err := s.p.Patients.Registered(ctx, patient.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true})
	if err != nil {
		return 0, err
	}
	written := 0
	for _, id := range ids {
		rec, err := s.record(ctx, x.RequestedBy, x.OrganizationID, true, id, patient.PurposeExport)
		if errors.Is(err, patient.ErrNotFound) {
			continue // gone since the list was read
		} else if err != nil {
			return 0, err
		}
		body, err := bundleJSON(rec)
		if err != nil {
			return 0, err
		}
		if err := write(zw, x, "patients/"+id+".json", body); err != nil {
			return 0, err
		}
		written++
	}
	return written, nil
}

// write adds the file name with body to zw.
func write(zw *zip.Writer, x Export, name string, body []byte) error {
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: x.RequestedAt})
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// manifest is the archive's manifest.txt: what each file holds, how many patients' records it has, and what it leaves
// out, and the records the clinic restricts (S7). Plain text in English: the archive is for the clinic's next software
// and whoever sets it up.
func manifest(patients int, restricted []string) string {
	return "Salus clinic data export\n\n" +
		"prices.csv        the price list\n" +
		"staff.csv         the practitioners and their roles\n" +
		"appointments.csv  every appointment, with the patient's id and name\n" +
		"receipts.csv      every receipt and invoice issued\n" +
		"quotes.csv        every treatment quote sent\n" +
		"patients/<id>.json  the record of each patient the clinic registered, as a FHIR R4 Bundle (collection):\n" +
		"                  the profile, the consents given to the clinic, the appointments and the receipts at the clinic\n\n" +
		"Patients' records in this archive: " + strconv.Itoa(patients) + "\n\n" +
		"Not in this archive: clinical documents, questionnaire answers and the patient's history. Who may hand\n" +
		"them over is not decided yet. A patient with their own Salus account has no file in patients/; their\n" +
		"appointments, receipts and quotes at the clinic are in the CSV files.\n\n" +
		"After a patient withdraws their consent, the clinic's screens show only their visits from before the\n" +
		"withdrawal; this archive holds every visit at the clinic: the visits a withdrawal of consent cancelled are in it too,\n" +
		"in appointments.csv and in the patient's record, with the status cancelled.\n" +
		restrictedNote(restricted)
}

// restrictedNote names the patients whose record the clinic restricts (GDPR Art. 18): the clinic's next software must
// keep them restricted. Empty when there are none.
func restrictedNote(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return "\nRestricted records (GDPR Art. 18), by patient id:\n" + strings.Join(ids, "\n") + "\n" +
		"These records are kept but may not be used, except to store them, for legal claims or with the patient's\n" +
		"consent; keep them restricted in the software that takes this archive over.\n"
}

// cells guards each cell against formula injection: one that a spreadsheet would read as a formula (=, +, -, @, a tab
// or a carriage return first) is prefixed with an apostrophe. Numbers are written by the exporter and never start so.
func cells(row []string) []string {
	out := make([]string, len(row))
	for i, c := range row {
		if c != "" && strings.ContainsRune("=+-@\t\r", rune(c[0])) {
			c = "'" + c
		}
		out[i] = c
	}
	return out
}

func (s *Service) prices(ctx context.Context, x Export) ([][]string, error) {
	defs, err := s.p.PriceList.OfClinic(ctx, x.OrganizationID)
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"id", "code", "name", "price_minor", "currency", "vat_rate_bp", "duration_minutes", "modes", "active"}}
	for _, d := range defs {
		vat := ""
		if d.VATRateBP != nil {
			vat = strconv.Itoa(int(*d.VATRateBP))
		}
		modes := make([]string, len(d.Modes))
		for i, m := range d.Modes {
			modes[i] = string(m)
		}
		rows = append(rows, []string{d.ID, d.Code, d.Name, strconv.FormatInt(d.PriceMinor, 10), d.Currency, vat,
			strconv.Itoa(int(d.DurationMinutes)), strings.Join(modes, " "), strconv.FormatBool(d.Active)})
	}
	return rows, nil
}

func (s *Service) staff(ctx context.Context, x Export) ([][]string, error) {
	roles, err := s.p.Roles.Search(ctx, practitionerrole.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true}, x.OrganizationID)
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"practitioner_role_id", "practitioner_id", "given_name", "family_name", "specialty_code", "active"}}
	for _, r := range roles {
		p, err := s.p.Practitioners.Read(ctx, practitioner.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID}, r.PractitionerID)
		if err != nil && !errors.Is(err, practitioner.ErrNotFound) {
			return nil, err
		}
		rows = append(rows, []string{r.ID, r.PractitionerID, p.GivenName, p.FamilyName, r.SpecialtyCode, strconv.FormatBool(r.Active)})
	}
	return rows, nil
}

func (s *Service) appointments(ctx context.Context, x Export) ([][]string, error) {
	seen, err := s.p.Appointments.ClinicRecord(ctx, appointment.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true}, appointment.Filter{})
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"id", "start", "end", "status", "visit_mode", "patient_id", "patient_given_name", "patient_family_name",
		"practitioner_role_id", "service_id"}}
	for _, v := range seen {
		a := v.Appointment
		rows = append(rows, []string{a.ID, stamp(a.Start), stamp(a.End), string(a.Status), string(a.Mode), a.PatientID,
			v.Patient.Given, v.Patient.Family, a.PractitionerRoleID, a.ServiceID})
	}
	return rows, nil
}

func (s *Service) receipts(ctx context.Context, x Export) ([][]string, error) {
	c := invoice.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true}
	all, err := s.p.Receipts.List(ctx, c, invoice.Period{From: since, To: s.p.Clock.Now().Add(time.Minute)})
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"id", "number", "kind", "issued_at", "patient_id", "company_name", "company_tax_id", "appointment_id",
		"reverses_id", "payment_method", "total_minor", "currency", "fiscal_status", "fiscal_id"}}
	for _, i := range all {
		rows = append(rows, []string{i.ID, strconv.FormatInt(i.Number, 10), string(i.Kind), stamp(i.IssuedAt), i.PatientID,
			i.CompanyName, i.CompanyTaxID, i.AppointmentID, i.ReversesID, string(i.PaymentMethod),
			strconv.FormatInt(i.TotalMinor, 10), i.Currency, string(i.FiscalStatus), i.FiscalID})
	}
	return rows, nil
}

func (s *Service) quotes(ctx context.Context, x Export) ([][]string, error) {
	all, err := s.p.Quotes.Quotes(ctx, invoice.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true}, "")
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"id", "issued_at", "valid_until", "state", "patient_id", "appointment_id", "total_minor", "currency"}}
	for _, q := range all {
		rows = append(rows, []string{q.ID, stamp(q.IssuedAt), stamp(q.ValidUntil), string(q.State), q.PatientID, q.AppointmentID,
			strconv.FormatInt(q.TotalMinor, 10), q.Currency})
	}
	return rows, nil
}

// stamp is an instant in UTC, RFC 3339; empty for none.
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
