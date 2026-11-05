// Package auditevent records who accessed what: FHIR R4 AuditEvent, append-only (docs/accounts/SPEC.md
// "AuditEvent shape"). Every access to an account or a directory record writes one, in the same
// transaction as the access.
package auditevent

import "time"

// Coding is a FHIR Coding: a code from the code system System.
type Coding struct {
	System string
	Code   string
}

// The code systems and codes this feature uses.
const (
	SystemDCM              = "http://dicom.nema.org/resources/ontology/DCM"
	CodeUserAuthentication = "110114" // type: User Authentication
	CodeLogin              = "110122" // subtype: Login
	CodeLogout             = "110123" // subtype: Logout

	SystemAuditEventType = "http://terminology.hl7.org/CodeSystem/audit-event-type"
	CodeRest             = "rest" // type: RESTful Operation

	SystemRestfulInteraction = "http://hl7.org/fhir/restful-interaction"
	CodeRead                 = "read"
	CodeCreate               = "create"
	CodeUpdate               = "update"
	CodeSearchType           = "search-type"
	CodeDelete               = "delete"
	CodeOperation            = "operation" // an operation that is no CRUD interaction (docs/consultation: the join)

	SystemActReason = "http://terminology.hl7.org/CodeSystem/v3-ActReason"
	CodeHOPERAT     = "HOPERAT" // purposeOfEvent: healthcare operations
	CodePATRQT      = "PATRQT"  // purposeOfEvent: the patient's own request (docs/booking/SPEC.md "Audit codes")
	CodeTREAT       = "TREAT"   // purposeOfEvent: treatment, for staff (docs/booking/SPEC.md "Audit codes")
)

// AgentOperator names the agent of an access by the operator command (cmd/salusctl), which is no account.
const AgentOperator = "salusctl"

// Action is AuditEvent.action (FHIR value set audit-event-action).
type Action string

// The actions.
const (
	ActionCreate  Action = "C"
	ActionRead    Action = "R"
	ActionUpdate  Action = "U"
	ActionDelete  Action = "D"
	ActionExecute Action = "E"
)

// Outcome is AuditEvent.outcome (FHIR value set audit-event-outcome).
type Outcome int16

// The outcomes.
const (
	OutcomeSuccess        Outcome = 0
	OutcomeMinorFailure   Outcome = 4
	OutcomeSeriousFailure Outcome = 8
	OutcomeMajorFailure   Outcome = 12
)

// AuditEvent is one access. It holds opaque ids and codes only: no e-mail, name, IP address or user agent.
type AuditEvent struct {
	Recorded time.Time // set by Record
	Type     Coding
	Subtype  Coding
	Purpose  Coding // purposeOfEvent
	// AgentAccountID is the agent, Account/<id>; empty when there is none (a failed login with an unknown
	// e-mail).
	AgentAccountID string
	// AgentName names an agent that is no account (AuditEvent.agent.name): AgentOperator; empty otherwise.
	AgentName       string
	EntityReference string // the entity, <ResourceType>/<id>; empty when there is none
	// EntityName names an entity that is a table, not a resource (AuditEvent.entity.name): the retention job's
	// deletions; empty otherwise.
	EntityName     string
	OrganizationID string // the clinic, when there is one
	RequestID      string // set by Record from the context
	Action         Action
	// EntityCount is how many rows of EntityName the event covers (AuditEvent.entity.detail "rows"); stored
	// with EntityName only.
	EntityCount int64
	Outcome     Outcome
}
