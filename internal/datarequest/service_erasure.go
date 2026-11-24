package datarequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Records says at which clinics a patient has a record (consent.Service): every clinic they ever gave a consent to, as
// a booking or a clinic's registration needs one.
type Records interface {
	ClinicsOf(ctx context.Context, patientID string) ([]string, error)
}

// Restrictions restricts the record a clinic keeps of an erased account's profile (patient.Service).
type Restrictions interface {
	RestrictForErasure(ctx context.Context, patientID, organizationID string) error
}

// ErasurePorts are what Erasure runs on.
type ErasurePorts struct {
	Tx           Transactor
	Repo         Repository
	Scope        Scope
	Accounts     Accounts
	Records      Records
	Restrictions Restrictions
	Devices      Devices
	Clock        clock.Clock
}

// Erasure erases the closed accounts whose erasure date has come (docs/clinic-cms/WAVE5.md S6): the retention applier
// of account.erase_after, run daily by its job.
type Erasure struct {
	p ErasurePorts
}

// NewErasure builds an Erasure on p.
func NewErasure(p ErasurePorts) *Erasure {
	return &Erasure{p: p}
}

// Erased is the record of one account's erasure: its request, answered, and the clinics that keep the patient's
// record, restricted, under the medical-record retention (each kept table's rule in internal/retention).
type Erased struct {
	AccountID string
	RequestID string
	Kept      []string // the clinics; empty when the patient had no record anywhere and was erased whole
}

// Run erases every account due, one transaction each, and returns what it did. Per account: the record each clinic
// keeps is restricted there, the push devices are removed, the account's login, password, MFA and languages are
// erased, and its erasure request is fulfilled: done when nothing was kept, done_partly_kept otherwise. An erased
// account is no longer due, so a second run erases nothing more. An account due without its erasure request stops the
// run with ErrNotFound: what to keep would be unknown.
func (e *Erasure) Run(ctx context.Context) ([]Erased, error) {
	ids, err := e.p.Accounts.DueForErasure(ctx)
	if err != nil {
		return nil, fmt.Errorf("datarequest: erasure: %w", err)
	}
	var out []Erased
	for _, id := range ids {
		done, err := e.erase(ctx, id)
		if err != nil {
			return out, fmt.Errorf("datarequest: erasure: %w", err)
		}
		out = append(out, done)
	}
	return out, nil
}

// erase erases the account id. The clinics that keep a record are read first, on their own (the consents' scope is
// not the erasure scope); then, in one transaction, the request is answered in the erasure scope before the
// restrictions set each clinic's.
func (e *Erasure) erase(ctx context.Context, accountID string) (Erased, error) {
	var r Request
	err := e.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		r, err = openErasure(ctx, e.p.Scope, e.p.Repo, accountID)
		return err
	})
	if err != nil {
		return Erased{}, err
	}
	var kept []string
	if r.PatientID != "" {
		if kept, err = e.p.Records.ClinicsOf(ctx, r.PatientID); err != nil {
			return Erased{}, err
		}
	}
	err = e.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := e.p.Scope.SetErasure(ctx); err != nil {
			return err
		}
		r.Status, r.AnswerCode, r.AnsweredAt, r.Version = StatusFulfilled, AnswerDone, e.p.Clock.Now(), r.Version+1
		if len(kept) > 0 {
			r.AnswerCode = AnswerDonePartlyKept
		}
		if ok, err := e.p.Repo.Update(ctx, r, ""); err != nil || !ok {
			return errors.Join(err, ErrVersionConflict)
		}
		if _, err := e.p.Devices.RemoveDevices(ctx, accountID); err != nil {
			return err
		}
		if err := e.p.Accounts.Erase(ctx, accountID); err != nil {
			return err
		}
		for _, org := range kept {
			if err := e.p.Restrictions.RestrictForErasure(ctx, r.PatientID, org); err != nil {
				return err
			}
		}
		return nil
	})
	return Erased{AccountID: accountID, RequestID: r.ID, Kept: kept}, err
}
