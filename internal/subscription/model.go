// Package subscription is what a clinic owes Salus (docs/plans/clinic-cms.md, package D; ADR 0016): its plan, priced
// per active practitioner role per month, a 30-day trial from its creation, its status, and the monthly invoice the
// scheduler issues for the month ahead. Plain JSON, not FHIR; nothing clinical and no personal data. The owner edits
// the plans and marks invoices paid in salusctl (Q2, Q3). Past due: a banner at 14 days, new bookings stop at 30; care
// never stops (Q4).
package subscription

import "time"

// Plan is a membership plan.
type Plan struct {
	Code              string // basic, pro
	Name              string
	Currency          string // ISO 4217, BAM
	Features          []string
	PricePerSeatMinor int64 // per active practitioner role per month
	Active            bool
}

// The membership's terms (docs/plans/clinic-cms.md Q2 to Q4, the defaults applied 2026-11-21).
const (
	DefaultPlan  = "basic"             // the plan of a new clinic's trial
	TrialLength  = 30 * 24 * time.Hour // from the clinic's creation
	PaymentTerm  = 14 * 24 * time.Hour // an invoice is due this long after it is issued
	SuspendAfter = 30 * 24 * time.Hour // past due this long, new bookings stop; care never does
)

// Status is where a clinic's membership stands. It moves only forward, through the monthly job or salusctl; the one
// way back is a payment: past_due or suspended returns to active when no invoice is past due any more.
type Status string

// The statuses.
const (
	StatusTrial     Status = "trial"
	StatusActive    Status = "active"
	StatusPastDue   Status = "past_due"
	StatusSuspended Status = "suspended"
	StatusCancelled Status = "cancelled"
)

// rank orders the statuses: a move is forward when it goes to a higher rank.
var rank = map[Status]int{StatusTrial: 1, StatusActive: 2, StatusPastDue: 3, StatusSuspended: 4, StatusCancelled: 5}

// Subscription is a clinic's membership. Current fills Plan and PastDueSince (the due date of the oldest open invoice
// past due, zero for none): the admin's banner reads it.
type Subscription struct {
	PastDueSince       time.Time
	TrialEndsAt        time.Time // zero without a trial
	CurrentPeriodStart time.Time // zero before the first invoice
	CurrentPeriodEnd   time.Time
	LastUpdated        time.Time
	OrganizationID     string
	PlanCode           string
	Status             Status
	Plan               Plan
}

// InvoiceStatus is where a monthly invoice stands.
type InvoiceStatus string

// The invoice statuses.
const (
	InvoiceOpen InvoiceStatus = "open"
	InvoicePaid InvoiceStatus = "paid"
	InvoiceVoid InvoiceStatus = "void"
)

// Invoice is Salus's monthly invoice to a clinic for the month [PeriodStart, PeriodEnd).
type Invoice struct {
	PeriodStart    time.Time
	PeriodEnd      time.Time
	IssuedAt       time.Time
	DueAt          time.Time
	PaidAt         time.Time // zero until paid
	ID             string
	OrganizationID string
	Currency       string
	Status         InvoiceStatus
	Number         int64 // in Salus's sequence
	AmountMinor    int64
	Seats          int32 // active practitioner roles on the 1st
}

// Caller is who asks: the signed-in account, its clinic and whether it is the clinic's admin.
type Caller struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}

// MonthlyArgs is the subscription_monthly job: issue the invoice of one clinic for the month that starts at Month, or
// of every clinic when OrganizationID is empty (the scheduler's run on the 1st). Opaque ids only.
type MonthlyArgs struct {
	Month          time.Time `json:"month"`
	OrganizationID string    `json:"organization_id,omitempty"`
}

// Kind names the job in river_job.
func (MonthlyArgs) Kind() string { return "subscription_monthly" }
