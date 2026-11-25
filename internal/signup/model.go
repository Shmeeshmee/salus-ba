// Package signup is a clinic's self-signup and its setup wizard (package O1 of docs/clinic-cms, W3): anyone may sign
// a clinic up with a verified e-mail; on verification the clinic is created unlisted, its admin invited and the trial
// started; the operator approves it for the public directory (salusctl). The admin's e-mail is personal data.
package signup

import "time"

// Signup is one clinic's request to sign up. The verification token is sent once and never stored, only its SHA-256.
type Signup struct {
	CreatedAt      time.Time
	VerifiedAt     time.Time // zero until the e-mail is verified
	ApprovedAt     time.Time // zero until the operator lists the clinic
	TokenIssuedAt  time.Time // when the latest verification e-mail's token was issued: its link's 72 hours start here
	ID             string
	Email          string // the admin's: trimmed and lower-cased; empty once the admin joined (U8)
	ClinicName     string
	Language       string // a BCP 47 tag of the language enum: the e-mail's and the clinic's default
	OrganizationID string // created on verification; empty before
}

// Request is what the public signup form sends.
type Request struct {
	Email      string
	ClinicName string
	Language   string
}

// Step is one step of the setup wizard at the admin's first sign-in.
type Step string

// The steps, in the wizard's order.
const (
	StepDetails      Step = "details"      // the clinic's details and address
	StepBilling      Step = "billing"      // the billing details
	StepHours        Step = "hours"        // working hours
	StepPractitioner Step = "practitioner" // the first practitioner invite
	StepServices     Step = "services"     // the price list, or its import
)

// StepState is where a step stands: each is skippable and the wizard resumable.
type StepState string

// The step states.
const (
	StepOpen    StepState = "open"
	StepDone    StepState = "done"
	StepSkipped StepState = "skipped"
)

// Setup is the wizard's progress for one clinic.
type Setup struct {
	Steps map[Step]StepState
}

// Caller is who asks for the wizard: the signed-in staff account, its clinic and whether it is the clinic's admin.
type Caller struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}

// Created is a clinic created by CreateClinic: the clinic, its admin's invite and whether its trial started (none while
// the owner has not set the default plan).
type Created struct {
	OrganizationID string
	InviteID       string
	Trial          bool
}

// Verification is what the verification e-mail needs: the address, the clinic's name, the language and the token of
// the link, issued anew for each e-mail.
type Verification struct {
	Email      string
	ClinicName string
	Language   string
	Token      string
}

// EmailArgs is the signup_email job, on the notifier's "email" queue: the verification e-mail of a signup. Its argument
// is an opaque id only; the job loads the rest.
type EmailArgs struct {
	SignupID string `json:"signup_id"`
}

// Kind names the job in river_job.
func (EmailArgs) Kind() string { return "signup_email" }
