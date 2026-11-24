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
	"github.com/Shmeeshmee/salus/internal/invoice"
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

// archive reads the clinic's data and writes it as one zip of CSV files.
func (s *Service) archive(ctx context.Context, x Export) ([]byte, error) {
	files := []struct {
		read func(context.Context, Export) ([][]string, error)
		name string
	}{
		{name: "prices.csv", read: s.prices},
		{name: "staff.csv", read: s.staff},
		{name: "appointments.csv", read: s.appointments},
		{name: "receipts.csv", read: s.receipts},
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
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	seen, err := s.p.Appointments.Search(ctx, appointment.Caller{AccountID: x.RequestedBy, OrganizationID: x.OrganizationID, Admin: true}, appointment.Filter{})
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

// stamp is an instant in UTC, RFC 3339; empty for none.
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
