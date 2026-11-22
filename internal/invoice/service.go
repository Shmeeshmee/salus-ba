package invoice

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// Prices prices a receipt's picks from the clinic's price list, in the caller's transaction
// (chargeitemdefinition.Service).
type Prices interface {
	Lines(ctx context.Context, org string, picks []chargeitemdefinition.Pick) ([]chargeitemdefinition.Line, error)
}

// Sellers gives the clinic's legal name, tax id and address as they are now, in the caller's transaction and scope
// (the organization domain's billing details and address).
type Sellers interface {
	Seller(ctx context.Context, org string) (Seller, error)
}

// Clinics gives a clinic's name, time zone and default language for the receipt's PDF, in the clinic's own scope and
// outside the receipt's transaction (organization.Service).
type Clinics interface {
	Clinic(ctx context.Context, org string) (organization.Organization, error)
}

// Visits gives a visit of the scoped clinic and its practitioner, in the caller's transaction, unaudited
// (appointment.Service): the receipt's own event covers the read.
type Visits interface {
	Lookup(ctx context.Context, id string) (appointment.Appointment, error)
	LookupPractitioner(ctx context.Context, id string) (string, error)
}

// Consents gives the state of a patient's consents with the clinic, in the caller's transaction and scope
// (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, org string) (access.Consent, error)
}

// Care gives a patient's visits in the clinic, in the caller's transaction and scope (appointment.Facts).
type Care interface {
	Visits(ctx context.Context, patientID, org string) ([]access.Visit, error)
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Draft is a receipt to issue: to a patient, a company or neither, for a visit or on its own, its picks from the
// price list and how it was paid. A receipt for a visit that names no company is the visit's patient's.
type Draft struct {
	PatientID     string
	CompanyName   string
	CompanyTaxID  string
	AppointmentID string
	Kind          Kind
	PaymentMethod PaymentMethod
	Picks         []chargeitemdefinition.Pick
}

// Ports are the Service's dependencies. The fiscal job (Fiscalize, GiveUp) needs Repo, Reporter, Audit and Tx only.
type Ports struct {
	Repo     Repository
	Prices   Prices
	Reporter fiscal.Reporter
	Sellers  Sellers
	Clinics  Clinics
	Visits   Visits
	Consents Consents
	Care     Care
	Jobs     Jobs
	Audit    Auditor
	Tx       Transactor
	Clock    clock.Clock
}

// Service holds the receipts' rules: the clinic's admin issues, reverses, lists and reports receipts, the
// practitioner of a visit issues its receipt and reads the receipts they issued (the reception role waits for Q1), a
// patient sees their own; every access is audited, also when it is refused.
type Service struct{ p Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{p: p} }

var taxID = regexp.MustCompile(`^[0-9]{12,13}$`)

// Issue numbers, stores and enqueues the fiscal report of a receipt (POST /clinic/receipts). ErrInvalid for a draft
// that is no receipt, a line the price list does not price, a patient who is not the visit's, or, on its own, a
// patient the clinic does not treat; ErrForbidden to anyone but the clinic's admin and the practitioner of the visit;
// ErrNotFound for a visit the clinic does not have; ErrSellerIncomplete while the clinic's billing details are.
func (s *Service) Issue(ctx context.Context, c Caller, d Draft) (Invoice, error) {
	if !validDraft(d) {
		return Invoice{}, ErrInvalid
	}
	if c.OrganizationID == "" || (!c.Admin && (c.PractitionerID == "" || d.AppointmentID == "")) {
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		inv, err := s.draft(ctx, c, d)
		if err != nil {
			return err
		}
		out, err = s.store(ctx, c, inv)
		return err
	})
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotFound):
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, err)
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrSellerIncomplete):
		return Invoice{}, err
	case err != nil:
		return Invoice{}, fmt.Errorf("invoice: issue: %w", err)
	}
	return out, nil
}

// validDraft reports whether d could be a receipt: a known kind and payment method, lines of a positive quantity and
// no negative discount, an invoice exactly when a company is named, a company's tax id only with its name, and
// never a patient and a company together.
func validDraft(d Draft) bool {
	if (d.Kind != KindReceipt && d.Kind != KindInvoice) || !validMethod(d.PaymentMethod) || len(d.Picks) == 0 {
		return false
	}
	for _, p := range d.Picks {
		if p.DefinitionID == "" || p.Quantity <= 0 || p.DiscountMinor < 0 {
			return false
		}
	}
	if (d.Kind == KindInvoice) != (d.CompanyName != "") || (d.CompanyName != "" && d.PatientID != "") {
		return false
	}
	return d.CompanyTaxID == "" || (d.CompanyName != "" && taxID.MatchString(d.CompanyTaxID))
}

func validMethod(m PaymentMethod) bool {
	return m == PaymentCash || m == PaymentCard || m == PaymentTransfer
}

// draft is the receipt d asks for, without its number: the buyer checked, the seller and the lines copied.
func (s *Service) draft(ctx context.Context, c Caller, d Draft) (Invoice, error) {
	patientID, err := s.buyer(ctx, c, d)
	if err != nil {
		return Invoice{}, err
	}
	seller, err := s.p.Sellers.Seller(ctx, c.OrganizationID)
	if err != nil {
		return Invoice{}, err
	}
	if seller.LegalName == "" || !taxID.MatchString(seller.TaxID) || seller.Address == "" {
		return Invoice{}, ErrSellerIncomplete
	}
	priced, err := s.p.Prices.Lines(ctx, c.OrganizationID, d.Picks)
	switch {
	case errors.Is(err, chargeitemdefinition.ErrNotFound), errors.Is(err, chargeitemdefinition.ErrInvalid):
		return Invoice{}, ErrInvalid
	case err != nil:
		return Invoice{}, err
	case len(priced) != len(d.Picks):
		return Invoice{}, ErrInvalid
	}
	inv := Invoice{
		OrganizationID: c.OrganizationID, SellerLegalName: seller.LegalName, SellerTaxID: seller.TaxID, SellerAddress: seller.Address,
		PatientID: patientID, CompanyName: d.CompanyName, CompanyTaxID: d.CompanyTaxID, AppointmentID: d.AppointmentID,
		Kind: d.Kind, PaymentMethod: d.PaymentMethod, Currency: priced[0].Currency,
	}
	for _, l := range priced {
		amount := int64(l.Quantity) * l.UnitPriceMinor
		if l.Currency != inv.Currency || l.Quantity <= 0 || l.DiscountMinor < 0 || l.DiscountMinor > amount || l.TotalMinor != amount-l.DiscountMinor {
			return Invoice{}, ErrInvalid
		}
		inv.Lines = append(inv.Lines, Line{
			VATRateBP: l.VATRateBP, ChargeItemDefinitionID: l.DefinitionID, Name: l.Name, UnitPriceMinor: l.UnitPriceMinor,
			DiscountMinor: l.DiscountMinor, TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		})
		inv.TotalMinor += l.TotalMinor
	}
	return inv, nil
}

// buyer returns the patient the receipt names. For a visit: the visit's patient unless a company pays, and only to
// the clinic's admin or the visit's practitioner. On its own: only a patient the clinic treats (an active care
// consent, or a visit), because the patient sees the receipt.
func (s *Service) buyer(ctx context.Context, c Caller, d Draft) (string, error) {
	if d.AppointmentID == "" {
		if d.PatientID == "" {
			return "", nil
		}
		return d.PatientID, s.treats(ctx, c.OrganizationID, d.PatientID)
	}
	a, err := s.p.Visits.Lookup(ctx, d.AppointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return "", ErrNotFound
	case err != nil:
		return "", err
	case a.OrganizationID != c.OrganizationID:
		return "", ErrNotFound
	}
	if !c.Admin {
		practitioner, err := s.p.Visits.LookupPractitioner(ctx, d.AppointmentID)
		if err != nil {
			return "", err
		}
		if practitioner != c.PractitionerID {
			return "", ErrForbidden
		}
	}
	switch {
	case d.PatientID != "" && d.PatientID != a.PatientID:
		return "", ErrInvalid
	case d.CompanyName != "":
		return "", nil
	}
	return a.PatientID, nil
}

// treats returns ErrInvalid unless the clinic org treats the patient: their care consent with it is active, or they
// have a visit there.
func (s *Service) treats(ctx context.Context, org, patientID string) error {
	state, err := s.p.Consents.State(ctx, patientID, org)
	if err != nil || state.Active {
		return err
	}
	visits, err := s.p.Care.Visits(ctx, patientID, org)
	if err != nil {
		return err
	}
	if len(visits) == 0 {
		return ErrInvalid
	}
	return nil
}

// store gives inv the clinic's next number, stores it, enqueues its fiscal report and audits the create, all in the
// transaction in ctx: a receipt that is rolled back leaves no number and no job.
func (s *Service) store(ctx context.Context, c Caller, inv Invoice) (Invoice, error) {
	var err error
	if inv.Number, err = s.p.Repo.NextNumber(ctx, inv.OrganizationID); err != nil {
		return Invoice{}, err
	}
	inv.IssuedAt, inv.IssuedByAccountID = s.p.Clock.Now(), c.AccountID
	out, err := s.p.Repo.Insert(ctx, inv)
	if err != nil {
		return Invoice{}, err
	}
	if err := s.p.Jobs.EnqueueFiscalize(ctx, out.ID, out.OrganizationID); err != nil {
		return Invoice{}, err
	}
	return out, s.p.Audit.Record(ctx, event(c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess))
}

// Reverse issues the receipt that reverses the receipt id, once (POST /clinic/receipts/{id}/reverse): its lines and
// total negated, the seller and the buyer as on the receipt, the clinic's next number, reported like any receipt.
// By the clinic's admin. ErrAlreadyReversed for a receipt reversed already and for a reversal.
func (s *Service) Reverse(ctx context.Context, c Caller, id string, method PaymentMethod) (Invoice, error) {
	if !validMethod(method) {
		return Invoice{}, ErrInvalid
	}
	if c.OrganizationID == "" || !c.Admin {
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		orig, err := s.p.Repo.ByID(ctx, id)
		switch {
		case err != nil:
			return err
		case orig.ReversesID != "" || orig.ReversedByID != "":
			return ErrAlreadyReversed
		}
		out, err = s.store(ctx, c, reversal(orig, method))
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, err)
	case errors.Is(err, ErrAlreadyReversed):
		return Invoice{}, err
	case err != nil:
		return Invoice{}, fmt.Errorf("invoice: reverse: %w", err)
	}
	return out, nil
}

// reversal is the receipt that reverses orig: the same seller, buyer, visit and lines, every quantity and total
// negated (a line's discount is given back).
func reversal(orig Invoice, method PaymentMethod) Invoice {
	rev := orig
	rev.ID, rev.ReversesID, rev.ReversedByID, rev.PaymentMethod, rev.TotalMinor = "", orig.ID, "", method, -orig.TotalMinor
	rev.FiscalStatus, rev.FiscalID, rev.FiscalCode, rev.FiscalQR, rev.FiscalizedAt = FiscalPending, "", "", "", time.Time{}
	rev.Lines = make([]Line, len(orig.Lines))
	for i, l := range orig.Lines {
		l.ID, l.Quantity, l.TotalMinor = "", -l.Quantity, -l.TotalMinor
		rev.Lines[i] = l
	}
	return rev
}

// refuse audits a refused interaction by c in a transaction of its own and returns refusal.
func (s *Service) refuse(ctx context.Context, c Caller, interaction string, refusal error) error {
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.p.Audit.Record(ctx, event(c, interaction, "", auditevent.OutcomeMinorFailure))
	})
	if err != nil {
		return fmt.Errorf("invoice: refuse: %w", err)
	}
	return refusal
}

// event is the AuditEvent of c's interaction with the receipt id (none for a search or a refusal): for staff the
// clinic's operations (v3-ActReason HOPERAT), for a patient their own request (PATRQT).
func event(c Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeSearchType: auditevent.ActionExecute, auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if c.OrganizationID == "" {
		e.Purpose.Code = auditevent.CodePATRQT
	}
	if id != "" {
		e.EntityReference = "Invoice/" + id
	}
	return e
}
