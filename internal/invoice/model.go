// Package invoice is the receipts and invoices a clinic issues: FHIR R4 Invoice (docs/plans/clinic-cms.md, package C;
// ADR 0016). A receipt is issued by staff for a visit or on its own, from price-list lines (internal/chargeitemdefinition)
// with quantity and discount, paid in cash, by card or by transfer, numbered in the clinic's gapless sequence, and
// reported to the tax authority through pkg/fiscal by a River job after its transaction. An issued receipt never
// changes; a mistake is a reversing receipt. A receipt to a patient is health data (Art. 9); every access is audited.
//
// Step 0 holds the contract: the model, the ports and the service's methods, each refused with errors.ErrUnsupported
// until package C builds it.
package invoice

import "time"

// Kind is a receipt to a person or an invoice to a company.
type Kind string

// The kinds.
const (
	KindReceipt Kind = "receipt"
	KindInvoice Kind = "invoice"
)

// PaymentMethod is how it was paid.
type PaymentMethod string

// The payment methods.
const (
	PaymentCash     PaymentMethod = "cash"
	PaymentCard     PaymentMethod = "card"
	PaymentTransfer PaymentMethod = "transfer"
)

// FiscalStatus is where its report to the tax authority stands: pending until the job fiscalizes it, failed when the
// job gave up (the clinic sees it and the report is retried), fiscalized once, for good.
type FiscalStatus string

// The fiscal statuses.
const (
	FiscalPending    FiscalStatus = "pending"
	FiscalFiscalized FiscalStatus = "fiscalized"
	FiscalFailed     FiscalStatus = "failed"
)

// Invoice is an issued receipt or invoice with its lines. A patient, a company or neither (an anonymous cash sale).
type Invoice struct {
	IssuedAt          time.Time
	FiscalizedAt      time.Time // zero until fiscalized
	ID                string
	OrganizationID    string
	SellerLegalName   string // the clinic's, copied at issue
	SellerTaxID       string
	SellerAddress     string
	PatientID         string // empty unless to a patient
	CompanyName       string // empty unless to a company
	CompanyTaxID      string
	AppointmentID     string // the visit; empty for one on its own
	ReversesID        string // the receipt it reverses; empty unless a reversal
	IssuedByAccountID string
	Currency          string
	FiscalID          string
	FiscalCode        string
	FiscalQR          string // the QR payload printed on it
	Kind              Kind
	PaymentMethod     PaymentMethod
	FiscalStatus      FiscalStatus
	Lines             []Line
	Number            int64 // in the clinic's sequence
	TotalMinor        int64
}

// Line is one line of an issued receipt: what the price list said at issue, copied.
type Line struct {
	VATRateBP              *int32 // nil for exempt
	ID                     string
	ChargeItemDefinitionID string // empty for an item off the price list
	Name                   string
	UnitPriceMinor         int64
	DiscountMinor          int64
	TotalMinor             int64
	Quantity               int32
}

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and the
// practitioner they are; for a patient, their profile.
type Caller struct {
	AccountID      string
	OrganizationID string // empty for a patient
	PractitionerID string
	PatientID      string // empty for staff
	Admin          bool
}

// Period is a report's half-open range [From, To).
type Period struct {
	From time.Time
	To   time.Time
}

// FiscalizeArgs is the invoice_fiscalize job: report one receipt to the tax authority, after the transaction that
// issued it. Opaque ids only.
type FiscalizeArgs struct {
	InvoiceID      string `json:"invoice_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (FiscalizeArgs) Kind() string { return "invoice_fiscalize" }
