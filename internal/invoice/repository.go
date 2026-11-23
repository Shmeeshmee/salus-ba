package invoice

import (
	"context"

	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// Repository stores receipts. Every read sees only the clinic set by SetOrganization, or the patient's own after
// SetPatient (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id's own receipts.
	SetPatient(ctx context.Context, id string) error
	// ByID returns the receipt with its lines, or ErrNotFound (also outside the scope, or for an id that is no UUID).
	ByID(ctx context.Context, id string) (Invoice, error)
	// Locked is ByID with the receipt locked until the transaction ends, without ReversedByID.
	Locked(ctx context.Context, id string) (Invoice, error)
	// ByIdempotencyKey returns the clinic org's receipt issued with the key, with its lines, or ErrNotFound (also for
	// a key that is no UUID).
	ByIdempotencyKey(ctx context.Context, org, key string) (Invoice, error)
	// Issued returns the clinic org's receipts issued in the period, newest first, with their lines.
	Issued(ctx context.Context, org string, p Period) ([]Invoice, error)
	// OfPatient returns the patient's receipts, newest first, with their lines.
	OfPatient(ctx context.Context, patientID string) ([]Invoice, error)
	// NextNumber returns the clinic org's next receipt number and keeps its sequence locked until the transaction
	// ends: a transaction that rolls back gives the number back.
	NextNumber(ctx context.Context, org string) (int64, error)
	// Insert stores the receipt and its lines, in their order, as pending and returns it with their ids. ErrAlreadyReversed for a
	// second reversal of one receipt; ErrInvalid for a line the clinic's price list does not hold.
	Insert(ctx context.Context, inv Invoice) (Invoice, error)
	// SetFiscalized stores the authority's answer on a pending or failed receipt.
	SetFiscalized(ctx context.Context, id string, f fiscal.Fiscalized) error
	// SetFailed marks a pending receipt's report failed; any other receipt is left as it is.
	SetFailed(ctx context.Context, id string) error

	// InsertQuote stores the quote and its lines, in their order, and returns it with their ids. It takes no number
	// and touches no receipt. ErrInvalid for a line the clinic's price list does not hold or a visit not the patient's
	// at the clinic; ErrNotFound for a patient without the clinic's active care consent (row-level security).
	InsertQuote(ctx context.Context, q Quote) (Quote, error)
	// QuoteByID returns the quote with its lines, or ErrNotFound (also outside the scope, or for an id that is no UUID).
	QuoteByID(ctx context.Context, id string) (Quote, error)
	// QuotesOf returns the clinic org's quotes, newest first, with their lines.
	QuotesOf(ctx context.Context, org string) ([]Quote, error)
	// QuotesOfPatient returns the patient's quotes, newest first, with their lines.
	QuotesOfPatient(ctx context.Context, patientID string) ([]Quote, error)
}

// Jobs enqueues the receipt's fiscal report in the transaction in ctx (River's InsertTx): invoice_fiscalize, opaque
// ids only.
type Jobs interface {
	EnqueueFiscalize(ctx context.Context, invoiceID, org string) error
	// EnqueueQuoteEmail enqueues quote_email on the email queue in the transaction in ctx: the patient of the visit
	// appointmentID hears that a quote is in the app.
	EnqueueQuoteEmail(ctx context.Context, appointmentID, org string) error
}
