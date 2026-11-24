package datarequest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// AgentExport names the agent of the build job's own audit events (AuditEvent.agent.name): the job, which is no
// account. The reads it makes through the other services are audited by them, as the patient's own request.
const AgentExport = "data_request_export_build"

// ExportTransactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type ExportTransactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExportAuditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type ExportAuditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Records builds the patient's own record as a FHIR Bundle, from every clinic for an empty org and from the clinic org
// otherwise, each read audited as the patient's; nil for an account with no profile (clinicexport.OwnRecords).
type Records interface {
	Bundle(ctx context.Context, account, org string) ([]byte, error)
}

// Accounts reads the patient's account, audited (auth.Service).
type Accounts interface {
	Me(ctx context.Context, id string) (auth.Account, error)
}

// ExportPorts are what the patient's export works with (D2). The API needs Repo, Audit, Tx, Jobs, Store and KEK; the
// worker's build needs them all.
type ExportPorts struct {
	Repo     ExportRepository
	Audit    ExportAuditor
	Tx       ExportTransactor
	Jobs     ExportJobs
	Store    Store
	KEK      *envelope.KEK
	Records  Records
	Accounts Accounts
}

// StoredExport is the export of one access or portability request as it is stored, with its request's kind and
// target. WrappedKey and KEKID are the archive's data key, wrapped by the key-encryption key and bound to the request's
// id; Size is the archive's plaintext bytes. Those three and ReadyAt and ExpiresAt are set once it is ready.
type StoredExport struct {
	RequestedAt    time.Time
	ReadyAt        time.Time
	ExpiresAt      time.Time
	RequestID      string
	AccountID      string
	OrganizationID string // the request's clinic target; empty for the account
	Status         ExportStatus
	Kind           Kind
	KEKID          string
	WrappedKey     []byte
	Size           int64
}

// at is the export as it reads at now: a ready one past its expiry is expired.
func (x StoredExport) at(now time.Time) Export {
	out := Export{RequestID: x.RequestID, Status: x.Status}
	if x.Status == ExportReady {
		out.ExpiresAt = x.ExpiresAt
		if !now.Before(x.ExpiresAt) {
			out.Status = ExportExpired
		}
	}
	return out
}

// ExportArgs is the job that builds a patient's export: opaque ids only.
type ExportArgs struct {
	RequestID string `json:"data_request_id"`
	AccountID string `json:"account_id"`
}

// Kind names the job in the river_job table.
func (ExportArgs) Kind() string { return "data_request_export_build" }

// ExportObjectKey is where the archive of the request id's export is stored.
func ExportObjectKey(requestID string) string { return "datarequest/" + requestID }

// exports reports whether a request of kind k is served by the export (S2).
func exports(k Kind) bool { return k == KindAccess || k == KindPortability }

// QueueExport queues the export of r, an access or portability request the patient filed in the app, and its build,
// in the transaction in ctx (joined) and in the patient's scope; File calls it as it files such a request (D1). Nothing
// for another kind or a request a clinic filed for a patient it registered (S10: its hand-over is the admin's).
// ErrExportInProgress while an export of the patient's is still queued.
func (s *Service) QueueExport(ctx context.Context, r Request) error {
	if !exports(r.Kind) || r.AccountID == "" {
		return nil
	}
	x := s.p.Export
	err := x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, r.AccountID); err != nil {
			return err
		}
		inserted, err := x.Repo.InsertExport(ctx, StoredExport{RequestID: r.ID, AccountID: r.AccountID, RequestedAt: s.p.Clock.Now()})
		if err != nil {
			return err
		}
		if !inserted {
			return ErrExportInProgress
		}
		if err := x.Jobs.EnqueueExport(ctx, ExportArgs{RequestID: r.ID, AccountID: r.AccountID}); err != nil {
			return err
		}
		return x.Audit.Record(ctx, exportEvent(r.AccountID, auditevent.CodeCreate, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrExportInProgress):
		return ErrExportInProgress
	case err != nil:
		return fmt.Errorf("datarequest: queue export: %w", err)
	}
	return nil
}

// Export returns where the export of the patient's access or portability request id stands, audited as a read of the
// patient's request. ErrNotFound for a request that is not theirs or has no export.
func (s *Service) Export(ctx context.Context, p Patient, id string) (Export, error) {
	x, err := s.ownExport(ctx, p, id, false)
	if err != nil {
		return Export{}, err
	}
	return x.at(s.p.Clock.Now()), nil
}

// ExportArchive returns the plaintext archive of the patient's ready export of request id and its size, decrypted as
// it is read, audited as the patient's read. ErrNotFound for a request that is not theirs, an export not ready, or one
// expired. The caller closes the reader.
func (s *Service) ExportArchive(ctx context.Context, p Patient, id string) (io.ReadCloser, int64, error) {
	x, err := s.ownExport(ctx, p, id, true)
	if err != nil {
		return nil, 0, err
	}
	r, err := s.openExport(ctx, x)
	if err != nil {
		return nil, 0, fmt.Errorf("datarequest: export archive: %w", err)
	}
	return r, x.Size, nil
}

// ownExport reads the patient's export of request id in their scope and audits the read; ready asks for one that can
// be downloaded now.
func (s *Service) ownExport(ctx context.Context, p Patient, id string, ready bool) (StoredExport, error) {
	if p.AccountID == "" {
		return StoredExport{}, ErrNotFound
	}
	x := s.p.Export
	var out StoredExport
	err := x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, p.AccountID); err != nil {
			return err
		}
		var err error
		if out, err = x.Repo.ExportOf(ctx, id); err != nil {
			return err
		}
		if ready && out.at(s.p.Clock.Now()).Status != ExportReady {
			return ErrNotFound
		}
		return x.Audit.Record(ctx, exportEvent(p.AccountID, auditevent.CodeRead, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return StoredExport{}, ErrNotFound
	case err != nil:
		return StoredExport{}, fmt.Errorf("datarequest: export: %w", err)
	}
	return out, nil
}

// openExport reads the ready export's archive from the store and decrypts it as it is read.
func (s *Service) openExport(ctx context.Context, x StoredExport) (io.ReadCloser, error) {
	dek, err := s.p.Export.KEK.Unwrap(x.WrappedKey, x.RequestID)
	if err != nil {
		return nil, err
	}
	src, err := s.p.Export.Store.GetRange(ctx, ExportObjectKey(x.RequestID), 0, envelope.CiphertextSize(x.Size)-1)
	if err != nil {
		return nil, err
	}
	plain, err := envelope.Open(src, dek, x.RequestID)
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

// FailExport marks the queued export failed: the build job gave up. The request stays open for its answerer.
func (s *Service) FailExport(ctx context.Context, args ExportArgs) error {
	x := s.p.Export
	err := x.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := x.Repo.SetAccount(ctx, args.AccountID); err != nil {
			return err
		}
		ok, err := x.Repo.MarkExportFailed(ctx, args.RequestID)
		if err != nil || !ok {
			return err
		}
		return x.Audit.Record(ctx, exportJobEvent(auditevent.OutcomeSeriousFailure))
	})
	if err != nil {
		return fmt.Errorf("datarequest: fail export: %w", err)
	}
	return nil
}

// exportEvent is the audit event of the patient's interaction with their export: the table, as an export is no FHIR
// resource, for the patient's own request.
func exportEvent(account, interaction string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodePATRQT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: account,
		EntityName:     "data_request_export",
		EntityCount:    1,
	}
}

// exportJobEvent is the audit event of the build job's own write on the patient's export.
func exportJobEvent(outcome auditevent.Outcome) auditevent.AuditEvent {
	e := exportEvent("", auditevent.CodeUpdate, outcome)
	e.AgentName = AgentExport
	return e
}
