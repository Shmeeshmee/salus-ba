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
	// Issued returns the clinic org's receipts issued in the period, newest first, with their lines.
	Issued(ctx context.Context, org string, p Period) ([]Invoice, error)
	// OfPatient returns the patient's receipts, newest first, with their lines.
	OfPatient(ctx context.Context, patientID string) ([]Invoice, error)
	// NextNumber returns the clinic org's next receipt number and keeps its sequence locked until the transaction
	// ends: a transaction that rolls back gives the number back.
	NextNumber(ctx context.Context, org string) (int64, error)
	// Insert stores the receipt and its lines as pending and returns it with their ids. ErrAlreadyReversed for a
	// second reversal of one receipt; ErrInvalid for a line the clinic's price list does not hold.
	Insert(ctx context.Context, inv Invoice) (Invoice, error)
	// SetFiscalized stores the authority's answer on a pending or failed receipt.
	SetFiscalized(ctx context.Context, id string, f fiscal.Fiscalized) error
	// SetFailed marks a pending receipt's report failed; any other receipt is left as it is.
	SetFailed(ctx context.Context, id string) error
}

// Jobs enqueues the receipt's fiscal report in the transaction in ctx (River's InsertTx): invoice_fiscalize, opaque
// ids only.
type Jobs interface {
	EnqueueFiscalize(ctx context.Context, invoiceID, org string) error
}
