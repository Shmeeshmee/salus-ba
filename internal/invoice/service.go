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
	"github.com/Shmeeshmee/salus/internal/schedule"
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
// (appointment.Service): the receipt's own event covers the read. PatientHasAccount tells whether the visit's patient
// has a Salus account (a profile the clinic registered has none and gets no e-mail, docs/clinic-cms W1).
type Visits interface {
	Lookup(ctx context.Context, id string) (appointment.Appointment, error)
	LookupPractitioner(ctx context.Context, id string) (string, error)
	PatientHasAccount(ctx context.Context, id string) (bool, error)
}

// Sites names a clinic's location with its address on one line (location.Service.Place), "" for one it does not know:
// the place of service of an in-person visit's receipt (docs/clinic-cms/WAVE6.md U2).
type Sites interface {
	Place(ctx context.Context, org, id string) (string, error)
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
// price list and how it was paid. A receipt for a visit that names no company is the visit's patient's. A draft with
// an IdempotencyKey (a UUID the receipt dialog makes) is issued once: issued again, it answers the first receipt.
type Draft struct {
	IdempotencyKey string
	PatientID      string
	CompanyName    string
	CompanyTaxID   string
	AppointmentID  string
	Kind           Kind
	PaymentMethod  PaymentMethod
	Picks          []chargeitemdefinition.Pick
}

// Ports are the Service's dependencies. The fiscal job (Fiscalize, GiveUp) needs Repo, Reporter, Audit and Tx only.
type Ports struct {
	Repo     Repository
	Prices   Prices
	Reporter fiscal.Reporter
	Sellers  Sellers
	Clinics  Clinics
	Visits   Visits
	Sites    Sites // nil: no place of service on a receipt
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
// patient the clinic does not treat; ErrForbidden to anyone but the clinic's admin, its receptionists and the
// practitioner of the visit;
// ErrNotFound for a visit the clinic does not have; ErrSellerIncomplete while the clinic's billing details are.
// With an idempotency key already issued in the clinic: that receipt, audited as a read, or ErrInvalid when another
// account issued it.
func (s *Service) Issue(ctx context.Context, c Caller, d Draft) (Invoice, error) {
	if !validDraft(d) {
		return Invoice{}, ErrInvalid
	}
	if c.OrganizationID == "" || (!c.issuesAny() && (c.PractitionerID == "" || d.AppointmentID == "")) {
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	var out Invoice
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var found bool
		var err error
		if out, found, err = s.issued(ctx, c, d.IdempotencyKey); found || err != nil {
			return err
		}
		inv, err := s.draft(ctx, c, d)
		if err != nil {
			return err
		}
		inv.IdempotencyKey = d.IdempotencyKey
		out, err = s.store(ctx, c, inv)
		return err
	})
	if errors.Is(err, errIssuedMeanwhile) { // the same key, issued while this one waited for the number
		err = s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
			if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
				return err
			}
			var err error
			out, _, err = s.issued(ctx, c, d.IdempotencyKey)
			return err
		})
	}
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
// maxLines and maxQuantity bound a receipt so its total fits an int64 with room to spare: with the price list's cap of
// 1,000,000,000 minor units a line is at most 1e13, a receipt at most 1e15.
const (
	maxLines    = 100
	maxQuantity = 10000
)

func validDraft(d Draft) bool {
	if (d.Kind != KindReceipt && d.Kind != KindInvoice) || !validMethod(d.PaymentMethod) || len(d.Picks) == 0 || len(d.Picks) > maxLines {
		return false
	}
	for _, p := range d.Picks {
		if p.DefinitionID == "" || p.Quantity <= 0 || p.Quantity > maxQuantity || p.DiscountMinor < 0 {
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
	patientID, visit, err := s.buyer(ctx, c, d)
	if err != nil {
		return Invoice{}, err
	}
	place := ""
	if visit.Mode == schedule.ModeInPerson && visit.LocationID != "" && s.p.Sites != nil {
		if place, err = s.p.Sites.Place(ctx, c.OrganizationID, visit.LocationID); err != nil {
			return Invoice{}, err
		}
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
		PlaceOfService: place, PatientID: patientID, CompanyName: d.CompanyName, CompanyTaxID: d.CompanyTaxID, AppointmentID: d.AppointmentID,
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

// buyer returns the patient the receipt names, and its visit (none for a receipt on its own). For a visit: the visit's patient unless a company pays, and only to
// the clinic's admin, its receptionists or the visit's practitioner. On its own: only a patient the clinic treats (an active care
// consent, or a visit), because the patient sees the receipt.
func (s *Service) buyer(ctx context.Context, c Caller, d Draft) (string, appointment.Appointment, error) {
	if d.AppointmentID == "" {
		if d.PatientID == "" {
			return "", appointment.Appointment{}, nil
		}
		return d.PatientID, appointment.Appointment{}, s.treats(ctx, c.OrganizationID, d.PatientID)
	}
	a, err := s.p.Visits.Lookup(ctx, d.AppointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return "", appointment.Appointment{}, ErrNotFound
	case err != nil:
		return "", appointment.Appointment{}, err
	case a.OrganizationID != c.OrganizationID:
		return "", appointment.Appointment{}, ErrNotFound
	}
	if !c.issuesAny() {
		practitioner, err := s.p.Visits.LookupPractitioner(ctx, d.AppointmentID)
		if err != nil {
			return "", appointment.Appointment{}, err
		}
		if practitioner != c.PractitionerID {
			return "", appointment.Appointment{}, ErrForbidden
		}
	}
	switch {
	case d.PatientID != "" && d.PatientID != a.PatientID:
		return "", appointment.Appointment{}, ErrInvalid
	case d.CompanyName != "":
		return "", a, nil
	}
	return a.PatientID, a, nil
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

// errIssuedMeanwhile rolls back a receipt whose idempotency key was issued by another request while this one waited
// for the clinic's number: the number is given back, and Issue answers the other request's receipt.
var errIssuedMeanwhile = errors.New("invoice: issued meanwhile")

// issued returns the clinic's receipt issued with the key and records its read, false when there is none (or no key).
// ErrInvalid when another account issued it: a key is the request of the one who made it.
func (s *Service) issued(ctx context.Context, c Caller, key string) (Invoice, bool, error) {
	if key == "" {
		return Invoice{}, false, nil
	}
	inv, err := s.p.Repo.ByIdempotencyKey(ctx, c.OrganizationID, key)
	switch {
	case errors.Is(err, ErrNotFound):
		return Invoice{}, false, nil
	case err != nil:
		return Invoice{}, false, err
	case inv.IssuedByAccountID != c.AccountID:
		return Invoice{}, false, ErrInvalid
	}
	return inv, true, s.p.Audit.Record(ctx, event(c, auditevent.CodeRead, inv.ID, auditevent.OutcomeSuccess))
}

// store gives inv the clinic's next number, stores it, enqueues its fiscal report and audits the create, all in the
// transaction in ctx: a receipt that is rolled back leaves no number and no job. The number keeps the clinic's
// sequence locked, so a receipt with an idempotency key looks for the key once more after it: two requests with one
// key issue one receipt (errIssuedMeanwhile).
func (s *Service) store(ctx context.Context, c Caller, inv Invoice) (Invoice, error) {
	var err error
	if inv.Number, err = s.p.Repo.NextNumber(ctx, inv.OrganizationID); err != nil {
		return Invoice{}, err
	}
	if inv.IdempotencyKey != "" {
		switch _, err := s.p.Repo.ByIdempotencyKey(ctx, inv.OrganizationID, inv.IdempotencyKey); {
		case err == nil:
			return Invoice{}, errIssuedMeanwhile
		case !errors.Is(err, ErrNotFound):
			return Invoice{}, err
		}
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
// By the clinic's admin. ErrAlreadyReversed for a receipt reversed already and for a reversal, ErrReportFailed for a
// receipt whose fiscal report failed (a pending one may be reversed: its reversal is reported after it).
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
		case orig.FiscalStatus == FiscalFailed:
			return ErrReportFailed
		}
		out, err = s.store(ctx, c, reversal(orig, method))
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Invoice{}, s.refuse(ctx, c, auditevent.CodeCreate, err)
	case errors.Is(err, ErrAlreadyReversed), errors.Is(err, ErrReportFailed):
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
	rev.IdempotencyKey = ""
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
