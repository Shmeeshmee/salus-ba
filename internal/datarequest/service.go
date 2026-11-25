package datarequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Profiles reads the signed-in patient's own profile, audited (patient.Service).
type Profiles interface {
	Me(ctx context.Context, c patient.Caller) (patient.Patient, error)
}

// Consents lists the signed-in patient's consents, audited (consent.Service): a clinic they gave one is a clinic where
// they have a record.
type Consents interface {
	Mine(ctx context.Context, c consent.Caller) ([]consent.Consent, error)
}

// Clinics reads a clinic, for its time zone (organization.Service).
type Clinics interface {
	Clinic(ctx context.Context, organizationID string) (organization.Organization, error)
}

// Objections marks a profile as objecting to the analytics, in the caller's transaction and scope (patient.Service).
type Objections interface {
	ObjectToAnalytics(ctx context.Context, patientID string) error
}

// Subjects reads, to the clinic's admin, the patient a request of the clinic names, audited, in the caller's
// transaction (patient.Service).
type Subjects interface {
	RequestSubject(ctx context.Context, c patient.Caller, id string) (patient.Patient, error)
}

// Ports are what the Service consumes.
type Ports struct {
	Repo     Repository
	Clock    clock.Clock
	Tx       Transactor
	Audit    Auditor
	Profiles Profiles
	Consents Consents
	Clinics  Clinics
	Jobs     Jobs        // the deadline warnings (the worker's)
	Export   ExportPorts // the patient's export (D2, service_export.go)
	// Objections: an objection marks the profile as it is filed (S9).
	Objections Objections
	// Subjects: the clinic's queue and detail name each request's patient (walk finding 3).
	Subjects Subjects
}

// Service is the requests' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// File files the patient's request for its target (S1, S2): their account, or a clinic where they have a record (one
// they gave a consent, active or not). It is received now and due DueMonths later in the answerer's time zone (S3).
// An access or portability request queues its export in the same transaction (S4). ErrInvalid for an unknown kind or
// target, ErrNotFound for a clinic where they have no record, ErrExportInProgress (nothing filed) while an export of
// theirs is still queued, ErrForbidden, audited, without an account. Audited as the patient's own request.
func (s *Service) File(ctx context.Context, p Patient, f Filing) (Request, error) {
	if p.refused() {
		return Request{}, s.refuse(ctx, p.AccountID, p.OrganizationID, auditevent.CodeCreate, auditevent.CodePATRQT)
	}
	if !f.Kind.valid() || !f.valid() {
		return Request{}, ErrInvalid
	}
	r, err := s.filing(ctx, p, f)
	if err != nil {
		return Request{}, err
	}
	var refused bool
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetAccount(ctx, p.AccountID); err != nil {
			return err
		}
		var err error
		if r, refused, err = s.insert(ctx, r, p.AccountID, auditevent.CodePATRQT); err != nil || refused {
			return err
		}
		if err := s.p.Audit.Record(ctx, event(p.AccountID, r, auditevent.CodeCreate, auditevent.CodePATRQT, 1)); err != nil {
			return err
		}
		if err := s.objected(ctx, r); err != nil {
			return err
		}
		return s.QueueExport(ctx, r) // an access or portability request's export, in this transaction (D2)
	})
	switch {
	case err != nil:
		return Request{}, wrap("file", err)
	case refused:
		return Request{}, ErrNotFound
	}
	return r, nil
}

// insert stores r in the transaction in ctx. A filing row-level security refuses (ErrNotFound: a profile or clinic out
// of the scope) is audited there as account's refused create, naming nothing it did not reach, and the transaction
// commits that audit: refused is true and err nil (BG4).
func (s *Service) insert(ctx context.Context, r Request, account, purpose string) (out Request, refused bool, err error) {
	out, err = s.p.Repo.Insert(ctx, r)
	if !errors.Is(err, ErrNotFound) {
		return out, false, err
	}
	e := event(account, Request{OrganizationID: r.OrganizationID}, auditevent.CodeCreate, purpose, 0)
	e.Outcome = auditevent.OutcomeMinorFailure
	return Request{}, true, s.p.Audit.Record(ctx, e)
}

// objected marks r's profile as objecting to the analytics when r is an objection (S9: from the request on), in the
// filing's transaction and scope.
func (s *Service) objected(ctx context.Context, r Request) error {
	if r.Kind != KindObjection || r.PatientID == "" {
		return nil
	}
	return s.p.Objections.ObjectToAnalytics(ctx, r.PatientID)
}

// filing is the request p files with f, received now and due in the answerer's time zone, after checking that a clinic
// target is one of p's clinics. Its lookups run before the request's own transaction: each sets its own scope.
func (s *Service) filing(ctx context.Context, p Patient, f Filing) (Request, error) {
	profile, err := s.p.Profiles.Me(ctx, patient.Caller{AccountID: p.AccountID})
	switch {
	case errors.Is(err, patient.ErrNotFound):
	case err != nil:
		return Request{}, fmt.Errorf("datarequest: file: %w", err)
	}
	if f.Target == TargetClinic {
		if profile.ID == "" {
			return Request{}, ErrNotFound
		}
		ok, err := s.hasRecord(ctx, p.AccountID, profile.ID, f.OrganizationID)
		switch {
		case err != nil:
			return Request{}, err
		case !ok:
			return Request{}, ErrNotFound
		}
	}
	due, err := s.dueAt(ctx, f.OrganizationID)
	if err != nil {
		return Request{}, err
	}
	return Request{
		AccountID: p.AccountID, PatientID: profile.ID, OrganizationID: f.OrganizationID, FiledBy: p.AccountID,
		Kind: f.Kind, Status: StatusReceived, ReceivedAt: s.p.Clock.Now(), DueAt: due, Version: 1,
	}, nil
}

// hasRecord reports whether the patient gave the clinic org a consent, active or not.
func (s *Service) hasRecord(ctx context.Context, accountID, patientID, org string) (bool, error) {
	all, err := s.p.Consents.Mine(ctx, consent.Caller{AccountID: accountID, PatientID: patientID})
	if err != nil {
		return false, fmt.Errorf("datarequest: file: %w", err)
	}
	for _, c := range all {
		if c.OrganizationID == org && c.PatientID == patientID {
			return true, nil
		}
	}
	return false, nil
}

// Mine returns the patient's requests, newest first, audited as one search. ErrForbidden, audited, without an account.
func (s *Service) Mine(ctx context.Context, p Patient) ([]Request, error) {
	if p.refused() {
		return nil, s.refuse(ctx, p.AccountID, p.OrganizationID, auditevent.CodeSearchType, auditevent.CodePATRQT)
	}
	out := []Request{}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetAccount(ctx, p.AccountID); err != nil {
			return err
		}
		all, err := s.p.Repo.ForAccount(ctx, p.AccountID)
		if err != nil {
			return err
		}
		out = append(out, all...)
		return s.p.Audit.Record(ctx, event(p.AccountID, Request{}, auditevent.CodeSearchType, auditevent.CodePATRQT, int64(len(out))))
	})
	if err != nil {
		return nil, fmt.Errorf("datarequest: mine: %w", err)
	}
	return out, nil
}

// Withdraw withdraws the patient's open request id at version; it keeps its history. ErrNotFound for none of theirs,
// ErrClosed once answered or withdrawn, ErrVersionConflict for another version. Audited.
func (s *Service) Withdraw(ctx context.Context, p Patient, id string, version int) (Request, error) {
	if p.refused() {
		return Request{}, s.refuse(ctx, p.AccountID, p.OrganizationID, auditevent.CodeUpdate, auditevent.CodePATRQT)
	}
	var out Request
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetAccount(ctx, p.AccountID); err != nil {
			return err
		}
		r, err := s.open(ctx, id, version)
		if err != nil {
			return err
		}
		if r.AccountID != p.AccountID { // not on row-level security alone
			return ErrNotFound
		}
		r.Status = StatusWithdrawn
		if out, err = s.update(ctx, r, p.AccountID); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(p.AccountID, out, auditevent.CodeUpdate, auditevent.CodePATRQT, 1))
	})
	if err != nil {
		return Request{}, wrap("withdraw", err)
	}
	return out, nil
}

// FileOnBehalf files a request of kind for the patient the staff's clinic registered, targeting that clinic (S10): by
// its admin or a receptionist, after checking the patient's identity at the desk. The request names no account. That
// the clinic registered the profile is held by row-level security (ErrNotFound otherwise). ErrInvalid for an unknown
// kind; ErrForbidden, audited, to other staff. Audited.
func (s *Service) FileOnBehalf(ctx context.Context, st Staff, patientID string, kind Kind) (Request, error) {
	if st.OrganizationID == "" || st.AccountID == "" || !st.Admin && !st.Reception {
		return Request{}, s.refuse(ctx, st.AccountID, st.OrganizationID, auditevent.CodeCreate, auditevent.CodeHOPERAT)
	}
	if !kind.valid() {
		return Request{}, ErrInvalid
	}
	due, err := s.dueAt(ctx, st.OrganizationID)
	if err != nil {
		return Request{}, err
	}
	r := Request{
		PatientID: patientID, OrganizationID: st.OrganizationID, FiledBy: st.AccountID, Kind: kind,
		Status: StatusReceived, ReceivedAt: s.p.Clock.Now(), DueAt: due, Version: 1,
	}
	var refused bool
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
			return err
		}
		var err error
		if r, refused, err = s.insert(ctx, r, st.AccountID, auditevent.CodeHOPERAT); err != nil || refused {
			return err
		}
		if err := s.p.Audit.Record(ctx, event(st.AccountID, r, auditevent.CodeCreate, auditevent.CodeHOPERAT, 1)); err != nil {
			return err
		}
		return s.objected(ctx, r)
	})
	switch {
	case err != nil:
		return Request{}, wrap("file on behalf", err)
	case refused:
		return Request{}, ErrNotFound
	}
	return r, nil
}

// ClinicRequests returns the requests targeting the admin's clinic, the open ones by deadline first, audited as one
// search. ErrForbidden, audited, to other staff.
func (s *Service) ClinicRequests(ctx context.Context, st Staff) ([]Request, error) {
	if !st.admin() {
		return nil, s.refuse(ctx, st.AccountID, st.OrganizationID, auditevent.CodeSearchType, auditevent.CodeHOPERAT)
	}
	out := []Request{}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
			return err
		}
		all, err := s.p.Repo.ForClinic(ctx, st.OrganizationID)
		if err != nil {
			return err
		}
		for i := range all {
			if all[i].Subject, err = s.subject(ctx, st, all[i]); err != nil {
				return err
			}
		}
		out = append(out, all...)
		return s.p.Audit.Record(ctx, event(st.AccountID, Request{OrganizationID: st.OrganizationID}, auditevent.CodeSearchType, auditevent.CodeHOPERAT, int64(len(out))))
	})
	if err != nil {
		return nil, fmt.Errorf("datarequest: clinic requests: %w", err)
	}
	return out, nil
}

// ClinicRequest returns one request targeting the admin's clinic with its history of status changes, audited.
// ErrNotFound for none of the clinic's; ErrForbidden, audited, to other staff.
func (s *Service) ClinicRequest(ctx context.Context, st Staff, id string) (Request, []Change, error) {
	if !st.admin() {
		return Request{}, nil, s.refuse(ctx, st.AccountID, st.OrganizationID, auditevent.CodeRead, auditevent.CodeHOPERAT)
	}
	var r Request
	var history []Change
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if r, err = s.clinics(ctx, st, id); err != nil {
			return err
		}
		if r.Subject, err = s.subject(ctx, st, r); err != nil {
			return err
		}
		if history, err = s.p.Repo.History(ctx, id); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(st.AccountID, r, auditevent.CodeRead, auditevent.CodeHOPERAT, 1))
	})
	if err != nil {
		return Request{}, nil, wrap("clinic request", err)
	}
	return r, history, nil
}

// Answer fulfils or refuses an open request targeting the admin's clinic, at the version read, with a code that fits
// (done or done_partly_kept fulfil; the others refuse). ErrInvalid for another status or a code that does not fit,
// ErrClosed once answered or withdrawn, ErrVersionConflict; ErrForbidden, audited, to other staff. Audited.
func (s *Service) Answer(ctx context.Context, st Staff, id string, a Answer) (Request, error) {
	if !st.admin() {
		return Request{}, s.refuse(ctx, st.AccountID, st.OrganizationID, auditevent.CodeUpdate, auditevent.CodeHOPERAT)
	}
	if !a.valid() {
		return Request{}, ErrInvalid
	}
	var out Request
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.clinicsOpen(ctx, st, id, a.Version)
		if err != nil {
			return err
		}
		r.Status, r.AnswerCode, r.AnsweredAt = a.Status, a.Code, s.p.Clock.Now()
		if out, err = s.update(ctx, r, st.AccountID); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(st.AccountID, out, auditevent.CodeUpdate, auditevent.CodeHOPERAT, 1))
	})
	if err != nil {
		return Request{}, wrap("answer", err)
	}
	return out, nil
}

// Extend extends an open request targeting the admin's clinic once, by ExtensionMonths after its due date in the
// clinic's time zone, with its reason (Art. 12(3)), before its due date has passed (S12). ErrInvalid for an unknown
// reason, ErrAlreadyExtended, ErrOverdue, ErrClosed, ErrVersionConflict; ErrForbidden, audited, to other staff. Audited.
func (s *Service) Extend(ctx context.Context, st Staff, id string, reason ExtensionReason, version int) (Request, error) {
	if !st.admin() {
		return Request{}, s.refuse(ctx, st.AccountID, st.OrganizationID, auditevent.CodeUpdate, auditevent.CodeHOPERAT)
	}
	if reason != ExtensionComplex && reason != ExtensionManyRequests {
		return Request{}, ErrInvalid
	}
	loc, err := s.location(ctx, st.OrganizationID)
	if err != nil {
		return Request{}, err
	}
	var out Request
	err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.clinicsOpen(ctx, st, id, version)
		if err != nil {
			return err
		}
		switch {
		case !r.ExtendedUntil.IsZero():
			return ErrAlreadyExtended
		case r.overdue(s.p.Clock.Now()):
			return ErrOverdue
		}
		r.ExtendedUntil, r.ExtensionReason = extendedUntil(r.DueAt, loc), reason
		if out, err = s.update(ctx, r, st.AccountID); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, event(st.AccountID, out, auditevent.CodeUpdate, auditevent.CodeHOPERAT, 1))
	})
	if err != nil {
		return Request{}, wrap("extend", err)
	}
	return out, nil
}

// subject reads the patient r names for the clinic's admin st, the read's audit event naming r (AuditRequestID); zero
// for a request without a patient, or one the read does not find. The caller has checked st is the admin.
func (s *Service) subject(ctx context.Context, st Staff, r Request) (Subject, error) {
	if r.PatientID == "" {
		return Subject{}, nil
	}
	c := patient.Caller{AccountID: st.AccountID, OrganizationID: st.OrganizationID, Admin: true}
	p, err := s.p.Subjects.RequestSubject(middleware.WithRequestID(ctx, AuditRequestID(r.ID)), c, r.PatientID)
	switch {
	case errors.Is(err, patient.ErrNotFound):
		return Subject{}, nil
	case err != nil:
		return Subject{}, fmt.Errorf("datarequest: the request's patient: %w", err)
	}
	return Subject{GivenName: p.GivenName, FamilyName: p.FamilyName, BirthDate: p.BirthDate, RegisteredByClinic: p.RegisteredBy == st.OrganizationID}, nil
}

// clinics scopes the transaction in ctx to the admin's clinic and returns its request id, or ErrNotFound.
func (s *Service) clinics(ctx context.Context, st Staff, id string) (Request, error) {
	if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
		return Request{}, err
	}
	r, err := s.p.Repo.ByID(ctx, id)
	if err == nil && r.OrganizationID != st.OrganizationID { // not on row-level security alone
		return Request{}, ErrNotFound
	}
	return r, err
}

// clinicsOpen is clinics for an open request at version.
func (s *Service) clinicsOpen(ctx context.Context, st Staff, id string, version int) (Request, error) {
	if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
		return Request{}, err
	}
	r, err := s.open(ctx, id, version)
	if err == nil && r.OrganizationID != st.OrganizationID {
		return Request{}, ErrNotFound
	}
	return r, err
}

// open returns the request id visible in the scope if it is open and at version: ErrNotFound, ErrClosed or
// ErrVersionConflict otherwise.
func (s *Service) open(ctx context.Context, id string, version int) (Request, error) {
	r, err := s.p.Repo.ByID(ctx, id)
	switch {
	case err != nil:
		return Request{}, err
	case r.closed():
		return Request{}, ErrClosed
	case r.Version != version:
		return Request{}, ErrVersionConflict
	}
	return r, nil
}

// update stores r at its next version, with a history entry by changedBy when its status changed.
func (s *Service) update(ctx context.Context, r Request, changedBy string) (Request, error) {
	r.Version++
	ok, err := s.p.Repo.Update(ctx, r, changedBy)
	switch {
	case err != nil:
		return Request{}, err
	case !ok:
		return Request{}, ErrVersionConflict
	}
	return r, nil
}

// refuse audits a refused interaction of the account, disclosing no id, in a transaction of its own, and returns
// ErrForbidden.
func (s *Service) refuse(ctx context.Context, account, org, interaction, purpose string) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		e := event(account, Request{OrganizationID: org}, interaction, purpose, 0)
		e.Outcome = auditevent.OutcomeMinorFailure
		return s.p.Audit.Record(ctx, e)
	})
	if err != nil {
		return fmt.Errorf("datarequest: refuse: %w", err)
	}
	return ErrForbidden
}

// wrap returns the domain's errors as they are and wraps the others with the operation.
func wrap(op string, err error) error {
	for _, e := range []error{ErrInvalid, ErrNotFound, ErrForbidden, ErrClosed, ErrAlreadyExtended, ErrOverdue, ErrVersionConflict, ErrExportInProgress} {
		if errors.Is(err, e) {
			return e
		}
	}
	return fmt.Errorf("datarequest: %s: %w", op, err)
}

// event is the audit event of an account's interaction with requests: count rows of the table data_request (a request
// is no FHIR resource), about the patient r names when it names one, at r's clinic.
func event(account string, r Request, interaction, purpose string, count int64) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeSearchType: auditevent.ActionRead, auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         action,
		Outcome:        auditevent.OutcomeSuccess,
		AgentAccountID: account,
		OrganizationID: r.OrganizationID,
		EntityName:     entityName,
		EntityCount:    count,
	}
	if r.PatientID != "" {
		e.EntityReference = "Patient/" + r.PatientID
	}
	return e
}

// entityName names the table of the requests in their audit events.
const entityName = "data_request"

func (k Kind) valid() bool {
	switch k {
	case KindAccess, KindPortability, KindRectification, KindRestriction, KindObjection, KindErasure:
		return true
	}
	return false
}

// valid: the account target names no clinic, a clinic target one.
func (f Filing) valid() bool {
	return f.Target == TargetAccount && f.OrganizationID == "" || f.Target == TargetClinic && f.OrganizationID != ""
}

// valid: fulfilled with done or done_partly_kept, refused with one of the refusals.
func (a Answer) valid() bool {
	switch a.Code {
	case AnswerDone, AnswerDonePartlyKept:
		return a.Status == StatusFulfilled
	case AnswerIdentityUnverified, AnswerManifestlyUnfounded, AnswerExcessive, AnswerRetentionLaw, AnswerNoData:
		return a.Status == StatusRefused
	}
	return false
}

func (r Request) closed() bool {
	return r.Status == StatusFulfilled || r.Status == StatusRefused || r.Status == StatusWithdrawn
}

func (st Staff) admin() bool { return st.Admin && st.OrganizationID != "" && st.AccountID != "" }
