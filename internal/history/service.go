package history

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// The ports of the history shared by the tickets that build it (docs/plans/patient-history.md "Contract"): each is
// implemented by the owning domain's service with this exact method, which applies access.ProfileEntry and
// access.Attributed and audits every entry it returns. The service methods of the history and their other ports come
// with SAL-39 (the patient's timeline) and SAL-40 (the practitioner's view and summary).

// Allergies gives the allergies of the patient the caller may see, entered-in-error ones left out
// (allergyintolerance.Service.Seen, SAL-41).
type Allergies interface {
	Seen(ctx context.Context, c appointment.Caller, patientID string) ([]allergyintolerance.Seen, error)
}

// Medications gives the medications of the patient the caller may see, entered-in-error ones left out
// (medicationstatement.Service.Seen, SAL-41).
type Medications interface {
	Seen(ctx context.Context, c appointment.Caller, patientID string) ([]medicationstatement.Seen, error)
}

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Encounters gives the visits of the patient a practitioner may see in their history (access.HistoryVisit), audits
// the ones a view shows, and the encounter of an appointment for the summary (encounter.Service).
type Encounters interface {
	History(ctx context.Context, c appointment.Caller, patientID string) ([]encounter.Visit, error)
	Disclosed(ctx context.Context, c appointment.Caller, ids []string) error
	Search(ctx context.Context, c appointment.Caller, f encounter.Filter) ([]encounter.Encounter, error)
}

// Compositions gives the signed documents of the visits, audits the ones a view shows, and the drafts of the
// appointment's encounter for the summary (composition.Service).
type Compositions interface {
	History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]composition.Document, error)
	Disclosed(ctx context.Context, c appointment.Caller, ids []string) error
	Drafts(ctx context.Context, c appointment.Caller, encounterID string) ([]composition.Draft, error)
}

// Responses gives the completed questionnaire responses of the visits with their scanned files, audits the ones a
// view shows and their files, and the responses of the appointment for the summary (questionnaireresponse.Service).
type Responses interface {
	History(ctx context.Context, c appointment.Caller, visits []encounter.Visit) ([]questionnaireresponse.Response, error)
	Disclosed(ctx context.Context, c appointment.Caller, ids []string) error
	DisclosedFiles(ctx context.Context, c appointment.Caller, ids []string) error
	Search(ctx context.Context, c appointment.Caller, f questionnaireresponse.Filter) ([]questionnaireresponse.Seen, error)
}

// Questionnaires gives the titles of the versions responses answer, in one read, each audited as the caller's read
// (questionnaire.Service).
type Questionnaires interface {
	Titles(ctx context.Context, c questionnaire.Caller, ids []string) (map[string]string, error)
}

// Conditions gives the coded fields (key and code) of the signed documents whose Condition is confirmed, by document:
// the caller's clinic's own, and a shared visit's (condition.Reads). Part of the documents, which are audited.
type Conditions interface {
	Confirmed(ctx context.Context, c appointment.Caller, documentIDs []string) (map[string][]condition.FieldCode, error)
}

// Clinics gives the names of the clinics a clinic's scope shows (organization.Service).
type Clinics interface {
	Names(ctx context.Context, org string, ids []string) (map[string]string, error)
}

// Roles gives the specialties of the practitioner roles a clinic's scope shows (practitionerrole.Service).
type Roles interface {
	Specialties(ctx context.Context, org string, ids []string) (map[string]string, error)
}

// Appointments gives the caller's attendance of an appointment, unaudited (appointment.Service): the summary is for
// one of the practitioner's own.
type Appointments interface {
	Attendance(ctx context.Context, c appointment.Caller, id string) (appointment.Attendance, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Tx             Transactor
	Audit          Auditor
	Encounters     Encounters
	Compositions   Compositions
	Responses      Responses
	Questionnaires Questionnaires
	Clinics        Clinics
	Roles          Roles
	Appointments   Appointments
	Allergies      Allergies
	Medications    Medications
	Conditions     Conditions
}

// Service builds the history views. It owns no table: every item comes from its owning domain's service.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }
