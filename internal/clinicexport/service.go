package clinicexport

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// AgentBuild names the agent of the build job's own audit events (AuditEvent.agent.name): the job, which is no account.
// The reads it makes through the other services are audited by them, as the admin who asked.
const AgentBuild = "clinic_export_build"

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// PriceList reads the clinic's whole price list (chargeitemdefinition.Service).
type PriceList interface {
	OfClinic(ctx context.Context, org string) ([]chargeitemdefinition.Definition, error)
}

// Roles reads the clinic's practitioner roles (practitionerrole.Service).
type Roles interface {
	Search(ctx context.Context, c practitionerrole.Caller, org string) ([]practitionerrole.PractitionerRole, error)
}

// Practitioners reads a practitioner of the caller's clinic (practitioner.Service).
type Practitioners interface {
	Read(ctx context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error)
}

// Appointments reads the clinic's appointments as its record holds them, the visits a withdrawal of consent cancelled
// included (docs/clinic-cms/WAVE5.md S11), each one audited (appointment.Service).
type Appointments interface {
	ClinicRecord(ctx context.Context, c appointment.Caller, f appointment.Filter) ([]appointment.Seen, error)
}

// Receipts lists the clinic's receipts, and those it issued to one patient, audited (invoice.Service).
type Receipts interface {
	List(ctx context.Context, c invoice.Caller, p invoice.Period) ([]invoice.Invoice, error)
	OfClinicPatient(ctx context.Context, c invoice.Caller, patientID string) ([]invoice.Invoice, error)
}

// Quotes lists the clinic's quotes, audited (invoice.Service): with no appointment id, every quote to its admin.
type Quotes interface {
	Quotes(ctx context.Context, c invoice.Caller, appointmentID string) ([]invoice.Quote, error)
}

// Patients lists the patients the clinic registered and reads one's record (profile and consents) as its admin,
// each read audited with its purpose (patient.Service).
type Patients interface {
	Registered(ctx context.Context, c patient.Caller) ([]string, error)
	ClinicRecord(ctx context.Context, c patient.Caller, id, purpose string) (patient.ClinicRecord, error)
}

// Ports are what the Service works with.
type Ports struct {
	Repo          Repository
	Audit         Auditor
	Tx            Transactor
	Clock         clock.Clock
	Jobs          Jobs
	Store         Store
	KEK           *envelope.KEK
	PriceList     PriceList
	Roles         Roles
	Practitioners Practitioners
	Appointments  Appointments
	Receipts      Receipts
	Quotes        Quotes
	Patients      Patients
}

// Service holds the export's rules: only the clinic's admin asks for, reads and downloads its exports; one is queued
// at a time; the archive is built by a job, encrypted, and downloadable until the export expires. Asking, the build
// and every download are audited.
type Service struct {
	p Ports
}

// NewService returns a Service on its ports. The API's routes need the sources of a patient's record (Patients,
// Appointments, Receipts) for its hand-over; the worker's build needs them all.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Request queues an export of the admin's clinic and its build, or returns the one already queued. ErrForbidden,
// audited, to anyone but the clinic's admin.
func (s *Service) Request(ctx context.Context, c Caller) (Export, error) {
	if !c.admin() {
		return Export{}, s.refuse(ctx, c, auditevent.CodeCreate)
	}
	now := s.p.Clock.Now()
	var out Export
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		e, inserted, err := s.p.Repo.Insert(ctx, Export{
			OrganizationID: c.OrganizationID, RequestedBy: c.AccountID, RequestedAt: now, ExpiresAt: now.Add(Window),
		})
		if err != nil {
			return err
		}
		if !inserted {
			out, err = s.p.Repo.Queued(ctx, c.OrganizationID)
			return err
		}
		out = e
		if err := s.p.Jobs.EnqueueBuild(ctx, BuildArgs{ExportID: e.ID, OrganizationID: e.OrganizationID}); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(c.AccountID, c.OrganizationID, auditevent.CodeCreate, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return Export{}, fmt.Errorf("clinicexport: request: %w", err)
	}
	return out, nil
}

// Get returns the export id of the admin's clinic as it stands now. Not audited: it holds no personal data.
// ErrForbidden to anyone but the clinic's admin; ErrNotFound for another clinic's.
func (s *Service) Get(ctx context.Context, c Caller, id string) (Export, error) {
	if !c.admin() {
		return Export{}, ErrForbidden
	}
	var out Export
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.p.Repo.ByID(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Export{}, ErrNotFound
	case err != nil:
		return Export{}, fmt.Errorf("clinicexport: get: %w", err)
	}
	return out.at(s.p.Clock.Now()), nil
}

// Download returns the plaintext archive of the ready export id and its size, decrypted as it is read, audited as the
// admin's read. ErrForbidden, audited, to anyone but the clinic's admin; ErrNotFound for another clinic's export, one
// not ready, or one expired. The caller closes the reader.
func (s *Service) Download(ctx context.Context, c Caller, id string) (io.ReadCloser, int64, error) {
	if !c.admin() {
		return nil, 0, s.refuse(ctx, c, auditevent.CodeRead)
	}
	var x Export
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if x, err = s.p.Repo.ByID(ctx, id); err != nil {
			return err
		}
		if x.at(s.p.Clock.Now()).Status != StatusReady {
			return ErrNotFound
		}
		return s.p.Audit.Record(ctx, event(c.AccountID, c.OrganizationID, auditevent.CodeRead, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, 0, ErrNotFound
	case err != nil:
		return nil, 0, fmt.Errorf("clinicexport: download: %w", err)
	}
	r, err := s.open(ctx, x)
	if err != nil {
		return nil, 0, fmt.Errorf("clinicexport: download: %w", err)
	}
	return r, x.Size, nil
}

// open reads the ready export's archive from the store and decrypts it as it is read.
func (s *Service) open(ctx context.Context, x Export) (io.ReadCloser, error) {
	dek, err := s.p.KEK.Unwrap(x.WrappedKey, x.ID)
	if err != nil {
		return nil, err
	}
	src, err := s.p.Store.GetRange(ctx, ObjectKey(x.ID), 0, envelope.CiphertextSize(x.Size)-1)
	if err != nil {
		return nil, err
	}
	plain, err := envelope.Open(src, dek, x.ID)
	if err != nil {
		_ = src.Close()
		return nil, err
	}
	return readCloser{Reader: plain, Closer: src}, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

// Fail marks the queued export failed: the build job gave up. The admin may then ask again.
func (s *Service) Fail(ctx context.Context, args BuildArgs) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, args.OrganizationID); err != nil {
			return err
		}
		ok, err := s.p.Repo.MarkFailed(ctx, args.ExportID)
		if err != nil || !ok {
			return err
		}
		return s.p.Audit.Record(ctx, jobEvent(args.OrganizationID, auditevent.CodeUpdate, auditevent.OutcomeSeriousFailure))
	})
	if err != nil {
		return fmt.Errorf("clinicexport: fail: %w", err)
	}
	return nil
}

func (c Caller) admin() bool { return c.Admin && c.OrganizationID != "" && c.AccountID != "" }

// refuse audits c's refused interaction and returns ErrForbidden.
func (s *Service) refuse(ctx context.Context, c Caller, interaction string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, event(c.AccountID, c.OrganizationID, interaction, auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("clinicexport: refuse: %w", err)
	}
	return ErrForbidden
}

// event is the audit event of an account's interaction with an export: the table, as an export is no FHIR resource.
func event(account, org, interaction string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: account,
		OrganizationID: org,
		EntityName:     "clinic_export",
		EntityCount:    1,
	}
}

// jobEvent is the audit event of the build job's own write.
func jobEvent(org, interaction string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event("", org, interaction, outcome)
	e.AgentName = AgentBuild
	return e
}
