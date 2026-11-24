// Package datarequest is the data subject's requests (GDPR Art. 12 to 21; docs/clinic-cms/WAVE5.md): a patient, or a
// clinic for a patient it registered, files a request of a kind for a target (the patient's Salus account or their
// record at one clinic); the target's answerer answers it by its legal deadline. It also closes a patient's account and
// serves the patient's own export. Package 5.0 is its contract: the services refuse until D1 to D3 build them.
package datarequest

import "time"

// Kind is what the patient asks for (S2). Access and portability are served by the same export.
type Kind string

// The kinds (data_request.kind).
const (
	KindAccess        Kind = "access"
	KindPortability   Kind = "portability"
	KindRectification Kind = "rectification"
	KindRestriction   Kind = "restriction"
	KindObjection     Kind = "objection"
	KindErasure       Kind = "erasure"
)

// Target is who answers a request (S1): Salus, through the operator, for the account; the clinic's admin for the
// patient's record at that clinic.
type Target string

// The targets. A clinic target names the clinic in OrganizationID.
const (
	TargetAccount Target = "account"
	TargetClinic  Target = "clinic"
)

// Status is where a request stands. It moves forward only; fulfilled, refused and withdrawn are final.
type Status string

// The statuses (data_request.status).
const (
	StatusReceived   Status = "received"
	StatusInProgress Status = "in_progress"
	StatusFulfilled  Status = "fulfilled"
	StatusRefused    Status = "refused"
	StatusWithdrawn  Status = "withdrawn"
)

// AnswerCode is the answer told to the patient: a code, never free text about health.
type AnswerCode string

// The answers. Done and DonePartlyKept fulfil a request; the others refuse it.
const (
	AnswerDone                AnswerCode = "done"
	AnswerDonePartlyKept      AnswerCode = "done_partly_kept"     // done; what the law keeps is kept (S6)
	AnswerIdentityUnverified  AnswerCode = "identity_unverified"  // Art. 12(6)
	AnswerManifestlyUnfounded AnswerCode = "manifestly_unfounded" // Art. 12(5)
	AnswerExcessive           AnswerCode = "excessive"            // Art. 12(5)
	AnswerRetentionLaw        AnswerCode = "retention_law"        // Art. 17(3)(b), (c): the medical-record retention
	AnswerNoData              AnswerCode = "no_data"
)

// ExtensionReason is why a request's deadline was extended, told to the patient (Art. 12(3)).
type ExtensionReason string

// The reasons.
const (
	ExtensionComplex      ExtensionReason = "complex"
	ExtensionManyRequests ExtensionReason = "many_requests"
)

// The deadlines (S3, S4, S5, S6).
const (
	DueMonths       = 1                   // a request is due one calendar month after receipt, in the answerer's time zone
	ExtensionMonths = 2                   // an extension adds at most two months, once
	EraseAfter      = 21 * 24 * time.Hour // a closed account is erased this long after closure, inside the one month of its erasure request; the operator may reopen it before (owner, 2026-11-24)
	ExportWindow    = 7 * 24 * time.Hour  // the patient's export can be downloaded this long after it is ready
)

// Request is one data subject's request. AccountID is the patient's own account when they filed it in the app, empty
// for one a clinic filed for a patient it registered (S10); PatientID is the patient's profile, empty only for an
// account target of an account without one.
type Request struct {
	ReceivedAt      time.Time
	DueAt           time.Time
	ExtendedUntil   time.Time // zero when not extended
	AnsweredAt      time.Time // zero until fulfilled or refused
	ID              string
	AccountID       string
	PatientID       string
	OrganizationID  string // the clinic target; empty for the account
	FiledBy         string // the account that filed it: the patient's, or the clinic's staff for S10
	Kind            Kind
	Status          Status
	ExtensionReason ExtensionReason
	AnswerCode      AnswerCode
	Version         int
}

// Target is who answers the request.
func (r Request) Target() Target {
	if r.OrganizationID == "" {
		return TargetAccount
	}
	return TargetClinic
}

// Change is one entry of a request's append-only history of status changes. ChangedBy is empty for a job or the
// operator.
type Change struct {
	ChangedAt time.Time
	ID        string
	RequestID string
	ChangedBy string
	Status    Status
}

// Deadline is an open request as the deadline job sees it (D1): with when its last warning was queued, zero for none.
type Deadline struct {
	WarnedAt time.Time
	Request  Request
}

// Deadline returns the request's deadline: its extension's, or its due date.
func (r Request) Deadline() time.Time {
	if r.ExtendedUntil.IsZero() {
		return r.DueAt
	}
	return r.ExtendedUntil
}

// overdue reports whether the request's due date has passed at now: it can no longer be extended (S12).
func (r Request) overdue(now time.Time) bool { return now.After(r.DueAt) }

// The warnings before a deadline (S3): the answerer is warned this many days before it.
const (
	WarnDays     = 7
	LastWarnDays = 1
)

// DeadlinesArgs is the scheduled job that queues the deadline warnings of every open request (S3).
type DeadlinesArgs struct{}

// Kind names the job in the river_job table.
func (DeadlinesArgs) Kind() string { return "data_request_deadlines" }

// WarningArgs is one deadline warning to a request's answerer, queued by the deadline job and sent by package D5: the
// request's opaque id and how many days before the deadline it warns (WarnDays or LastWarnDays).
type WarningArgs struct {
	RequestID string `json:"request_id"`
	Days      int    `json:"days"`
}

// Kind names the job in the river_job table.
func (WarningArgs) Kind() string { return "data_request_deadline_warning" }

// Filing is what a patient files: the kind and the target, with the clinic for a clinic target.
type Filing struct {
	Kind           Kind
	Target         Target
	OrganizationID string
}

// Answer is the answerer's: fulfilled or refused, with its code, at the version read.
type Answer struct {
	Status  Status
	Code    AnswerCode
	Version int
}

// ExportStatus is where the patient's export stands (S4).
type ExportStatus string

// The export statuses. Expired is never stored: a ready export past its window reads as it.
const (
	ExportQueued  ExportStatus = "queued"
	ExportReady   ExportStatus = "ready"
	ExportFailed  ExportStatus = "failed"
	ExportExpired ExportStatus = "expired"
)

// Export is the patient's export for an access or portability request.
type Export struct {
	ExpiresAt time.Time // zero until ready
	RequestID string
	Status    ExportStatus
}

// Patient is the signed-in patient: their account.
type Patient struct {
	AccountID string
}

// Staff is the signed-in staff account, its clinic and whether it is the clinic's admin or a receptionist.
type Staff struct {
	AccountID      string
	OrganizationID string
	Admin          bool
	Reception      bool // the front desk: files on behalf of a patient the clinic registered (S10)
}
