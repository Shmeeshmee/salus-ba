package organization

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

const maxEmail = 254 // characters

// taxID is the JIB (13 digits) or the IDB/PDV number (12), as the column's check (migration 00059).
var taxID = regexp.MustCompile(`^[0-9]{12,13}$`)

// BillingDetails are the clinic's details on Salus's invoices to it (docs/plans/clinic-cms.md, package D): its legal
// name, tax id, whether it is registered for VAT, and the e-mail the invoices go to. Each is empty until the admin
// states it. Not personal data.
type BillingDetails struct {
	LegalName     string
	TaxID         string
	BillingEmail  string
	VATRegistered bool
}

// BillingRepository stores the clinics' billing details (columns of organization, migration 00059), row-level secured
// like the clinics: every method sees only the clinic set by SetOrganization.
type BillingRepository interface {
	SetOrganization(ctx context.Context, id string) error
	// BillingDetails returns the clinic's details, or ErrNotFound.
	BillingDetails(ctx context.Context, org string) (BillingDetails, error)
	// SetBillingDetails replaces the clinic's details, or returns ErrNotFound.
	SetBillingDetails(ctx context.Context, org string, d BillingDetails) error
}

// BillingPorts are what Billing runs on.
type BillingPorts struct {
	Repo  BillingRepository
	Audit Auditor
	Tx    Transactor
}

// Billing holds the clinic's billing details: only its admin reads and changes them; a change is audited, also when
// refused.
type Billing struct{ BillingPorts }

// NewBilling returns Billing on its ports.
func NewBilling(p BillingPorts) *Billing { return &Billing{BillingPorts: p} }

// Details returns the billing details of the admin's clinic. ErrForbidden for its other staff, ErrNotFound for anyone
// else. Not audited: nothing in them is personal data.
func (s *Billing) Details(ctx context.Context, c Caller) (BillingDetails, error) {
	if err := adminOf(c); err != nil {
		return BillingDetails{}, err
	}
	var out BillingDetails
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.BillingDetails(ctx, c.OrganizationID)
		return err
	})
	if err != nil {
		return BillingDetails{}, billingError(err)
	}
	return out, nil
}

// ForClinic returns the billing details of the clinic org for a document it issues itself, such as the seller of its
// receipts (clinic-cms C): read in the clinic's scope, set on the transaction in ctx (joining one already there), for
// whoever of its staff issues; unaudited, as Details. ErrNotFound for no clinic.
func (s *Billing) ForClinic(ctx context.Context, org string) (BillingDetails, error) {
	if org == "" {
		return BillingDetails{}, ErrNotFound
	}
	var out BillingDetails
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.BillingDetails(ctx, org)
		return err
	})
	if err != nil {
		return BillingDetails{}, billingError(err)
	}
	return out, nil
}

// SetDetails replaces the billing details of the admin's clinic with in (trimmed), audited as an update of the
// clinic; refusals as Details', audited too. ErrInvalidBillingDetails for details that are malformed.
func (s *Billing) SetDetails(ctx context.Context, c Caller, in BillingDetails) (BillingDetails, error) {
	if err := adminOf(c); err != nil {
		if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			return s.Audit.Record(ctx, settingsEvent(c, auditevent.OutcomeMinorFailure))
		}); aerr != nil {
			return BillingDetails{}, billingError(aerr)
		}
		return BillingDetails{}, err
	}
	in.LegalName, in.TaxID, in.BillingEmail = strings.TrimSpace(in.LegalName), strings.TrimSpace(in.TaxID), strings.TrimSpace(in.BillingEmail)
	if !validBillingDetails(in) {
		return BillingDetails{}, ErrInvalidBillingDetails
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := s.Repo.SetBillingDetails(ctx, c.OrganizationID, in); err != nil {
			return err
		}
		return s.Audit.Record(ctx, settingsEvent(c, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return BillingDetails{}, billingError(err)
	}
	return in, nil
}

// validBillingDetails tells whether each stated detail is well formed; an empty one is not stated.
func validBillingDetails(d BillingDetails) bool {
	if utf8.RuneCountInString(d.LegalName) > maxName || (d.TaxID != "" && !taxID.MatchString(d.TaxID)) {
		return false
	}
	if d.BillingEmail == "" {
		return true
	}
	a, err := mail.ParseAddress(d.BillingEmail)
	return err == nil && a.Address == d.BillingEmail && len(d.BillingEmail) <= maxEmail
}

// billingError passes ErrNotFound and names the operation on any other error.
func billingError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("organization: billing details: %w", err)
}
