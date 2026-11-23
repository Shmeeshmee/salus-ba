package invoice

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shmeeshmee/salus/internal/invoice/invoicedb"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type postgresRepository struct {
	db postgres.Querier
}

// NewPostgresRepository returns the Repository on db (salus_app), using the transaction in the context when there is
// one: row-level security shows only the clinic SetOrganization set, or the patient's own after SetPatient.
func NewPostgresRepository(db postgres.Querier) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) q(ctx context.Context) *invoicedb.Queries {
	return invoicedb.New(postgres.Conn(ctx, r.db))
}

func (r *postgresRepository) SetOrganization(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.organization_id", id)
}

func (r *postgresRepository) SetPatient(ctx context.Context, id string) error {
	return postgres.SetLocal(ctx, "app.patient_id", id)
}

func (r *postgresRepository) ByID(ctx context.Context, id string) (Invoice, error) {
	u, ok := parse(id)
	if !ok {
		return Invoice{}, ErrNotFound
	}
	row, err := r.q(ctx).InvoiceByID(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Invoice{}, ErrNotFound
	case err != nil:
		return Invoice{}, err
	}
	out, err := r.withLines(ctx, []Invoice{fromRow(row.Invoice, row.ReversedByID)})
	if err != nil {
		return Invoice{}, err
	}
	return out[0], nil
}

func (r *postgresRepository) Locked(ctx context.Context, id string) (Invoice, error) {
	u, ok := parse(id)
	if !ok {
		return Invoice{}, ErrNotFound
	}
	row, err := r.q(ctx).InvoiceLocked(ctx, u)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Invoice{}, ErrNotFound
	case err != nil:
		return Invoice{}, err
	}
	out, err := r.withLines(ctx, []Invoice{fromRow(row, pgtype.UUID{})})
	if err != nil {
		return Invoice{}, err
	}
	return out[0], nil
}

func (r *postgresRepository) ByIdempotencyKey(ctx context.Context, org, key string) (Invoice, error) {
	o, okOrg := parse(org)
	k, okKey := parse(key)
	if !okOrg || !okKey {
		return Invoice{}, ErrNotFound
	}
	row, err := r.q(ctx).InvoiceByIdempotencyKey(ctx, invoicedb.InvoiceByIdempotencyKeyParams{OrganizationID: o, IdempotencyKey: k})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Invoice{}, ErrNotFound
	case err != nil:
		return Invoice{}, err
	}
	out, err := r.withLines(ctx, []Invoice{fromRow(row.Invoice, row.ReversedByID)})
	if err != nil {
		return Invoice{}, err
	}
	return out[0], nil
}

func (r *postgresRepository) Issued(ctx context.Context, org string, p Period) ([]Invoice, error) {
	u, ok := parse(org)
	if !ok {
		return nil, nil
	}
	rows, err := r.q(ctx).InvoicesIssued(ctx, invoicedb.InvoicesIssuedParams{OrganizationID: u, FromTime: timestamp(p.From), ToTime: timestamp(p.To)})
	if err != nil {
		return nil, err
	}
	out := make([]Invoice, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row.Invoice, row.ReversedByID)
	}
	return r.withLines(ctx, out)
}

func (r *postgresRepository) OfPatient(ctx context.Context, patientID string) ([]Invoice, error) {
	u, ok := parse(patientID)
	if !ok {
		return nil, nil
	}
	rows, err := r.q(ctx).InvoicesOfPatient(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Invoice, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row.Invoice, row.ReversedByID)
	}
	return r.withLines(ctx, out)
}

// withLines returns the receipts with their lines, read in one query after the receipts' rows were collected.
func (r *postgresRepository) withLines(ctx context.Context, list []Invoice) ([]Invoice, error) {
	if len(list) == 0 {
		return list, nil
	}
	ids, at := make([]pgtype.UUID, len(list)), make(map[string]int, len(list))
	for i, inv := range list {
		ids[i], _ = parse(inv.ID)
		at[inv.ID] = i
	}
	lines, err := r.q(ctx).InvoiceLines(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		inv := &list[at[l.InvoiceID.String()]]
		line := Line{
			ID: l.ID.String(), ChargeItemDefinitionID: text(l.ChargeItemDefinitionID), Name: l.Name, UnitPriceMinor: l.UnitPriceMinor,
			DiscountMinor: l.DiscountMinor, TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		}
		if l.VatRateBp.Valid {
			line.VATRateBP = &l.VatRateBp.Int32
		}
		inv.Lines = append(inv.Lines, line)
	}
	return list, nil
}

func (r *postgresRepository) NextNumber(ctx context.Context, org string) (int64, error) {
	u, err := must(org)
	if err != nil {
		return 0, err
	}
	return r.q(ctx).NextInvoiceNumber(ctx, u)
}

func (r *postgresRepository) Insert(ctx context.Context, inv Invoice) (Invoice, error) {
	p := invoicedb.InsertInvoiceParams{
		SellerLegalName: inv.SellerLegalName, SellerTaxID: inv.SellerTaxID, SellerAddress: inv.SellerAddress,
		CompanyName: optional(inv.CompanyName), CompanyTaxID: optional(inv.CompanyTaxID), Kind: string(inv.Kind),
		Number: inv.Number, IssuedAt: timestamp(inv.IssuedAt), PaymentMethod: string(inv.PaymentMethod),
		TotalMinor: inv.TotalMinor, Currency: inv.Currency,
	}
	for _, f := range []struct {
		to       *pgtype.UUID
		id       string
		required bool
	}{
		{&p.OrganizationID, inv.OrganizationID, true}, {&p.IssuedByAccountID, inv.IssuedByAccountID, true},
		{&p.PatientID, inv.PatientID, false}, {&p.AppointmentID, inv.AppointmentID, false}, {&p.ReversesID, inv.ReversesID, false},
		{&p.IdempotencyKey, inv.IdempotencyKey, false},
	} {
		if f.id == "" && !f.required {
			continue
		}
		u, ok := parse(f.id)
		if !ok {
			return Invoice{}, ErrInvalid // an id that is no UUID names nothing
		}
		*f.to = u
	}
	q := r.q(ctx)
	id, err := q.InsertInvoice(ctx, p)
	if err != nil {
		return Invoice{}, insertError(err)
	}
	inv.ID, inv.FiscalStatus = id.String(), FiscalPending
	inv.Lines = append([]Line(nil), inv.Lines...)
	for i, l := range inv.Lines {
		lp := invoicedb.InsertInvoiceLineParams{
			InvoiceID: id, Position: int32(i), Name: l.Name, Quantity: l.Quantity, UnitPriceMinor: l.UnitPriceMinor, DiscountMinor: l.DiscountMinor,
			TotalMinor: l.TotalMinor,
		}
		if l.ChargeItemDefinitionID != "" {
			u, ok := parse(l.ChargeItemDefinitionID)
			if !ok {
				return Invoice{}, ErrInvalid
			}
			lp.ChargeItemDefinitionID = u
		}
		if l.VATRateBP != nil {
			lp.VatRateBp = pgtype.Int4{Int32: *l.VATRateBP, Valid: true}
		}
		lineID, err := q.InsertInvoiceLine(ctx, lp)
		if err != nil {
			return Invoice{}, insertError(err)
		}
		inv.Lines[i].ID = lineID.String()
	}
	return inv, nil
}

// insertError maps the schema's refusals: a second reversal of one receipt, and a line or a visit that is not the
// clinic's. The messages of the constraints carry no receipt's content.
func insertError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch {
	case pgErr.ConstraintName == "invoice_reverses_key":
		return ErrAlreadyReversed
	case pgErr.Code == "23503", pgErr.Code == "23514": // a foreign key, a check or the line's guard
		return ErrInvalid
	}
	return err
}

func (r *postgresRepository) SetFiscalized(ctx context.Context, id string, f fiscal.Fiscalized) error {
	u, err := must(id)
	if err != nil {
		return err
	}
	n, err := r.q(ctx).SetInvoiceFiscalized(ctx, invoicedb.SetInvoiceFiscalizedParams{
		ID: u, FiscalID: optional(f.ID), FiscalCode: optional(f.Code), FiscalQr: optional(f.QR), FiscalizedAt: timestamp(f.At),
	})
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func (r *postgresRepository) SetFailed(ctx context.Context, id string) error {
	u, err := must(id)
	if err != nil {
		return err
	}
	_, err = r.q(ctx).SetInvoiceFailed(ctx, u)
	return err
}

func fromRow(i invoicedb.Invoice, reversedBy pgtype.UUID) Invoice {
	return Invoice{
		IssuedAt: i.IssuedAt.Time, FiscalizedAt: i.FiscalizedAt.Time, ID: i.ID.String(), OrganizationID: i.OrganizationID.String(),
		SellerLegalName: i.SellerLegalName, SellerTaxID: i.SellerTaxID, SellerAddress: i.SellerAddress,
		PatientID: text(i.PatientID), CompanyName: i.CompanyName.String, CompanyTaxID: i.CompanyTaxID.String,
		AppointmentID: text(i.AppointmentID), ReversesID: text(i.ReversesID), ReversedByID: text(reversedBy),
		IssuedByAccountID: i.IssuedByAccountID.String(), Currency: i.Currency, FiscalID: i.FiscalID.String,
		FiscalCode: i.FiscalCode.String, FiscalQR: i.FiscalQr.String, IdempotencyKey: text(i.IdempotencyKey), Kind: Kind(i.Kind),
		PaymentMethod: PaymentMethod(i.PaymentMethod), FiscalStatus: FiscalStatus(i.FiscalStatus), Number: i.Number,
		TotalMinor: i.TotalMinor,
	}
}

// parse reports whether s is a UUID, and returns it.
func parse(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	return u, u.Scan(s) == nil && u.Valid
}

// must is parse for an id the service made or read: one that is no UUID is a fault.
func must(s string) (pgtype.UUID, error) {
	u, ok := parse(s)
	if !ok {
		return u, errors.New("invoice: an id that is no UUID")
	}
	return u, nil
}

// text is the UUID as text, empty for NULL.
func text(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func optional(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
