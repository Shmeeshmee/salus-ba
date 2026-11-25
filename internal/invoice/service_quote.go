package invoice

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// QuoteDraft is a quote as staff write it at a visit: lines picked from the clinic's price list and a note.
type QuoteDraft struct {
	PatientID     string
	AppointmentID string
	Note          string
	Picks         []chargeitemdefinition.Pick
}

// quoteValidity is how long a quote is valid (W4 of docs/clinic-cms).
const quoteValidity = 30 * 24 * time.Hour

// maxNote bounds a quote's note, in characters (SendQuoteRequest.note).
const maxNote = 2000

// SendQuote stores d as a quote, valid 30 days, and enqueues the patient's e-mail in the same transaction
// (POST /clinic/quotes). A quote is no receipt: it takes no number, is never reported and charges nothing. By the
// clinic's admin and the practitioner of the visit. ErrInvalid for a draft that is no quote or a line the price list
// does not price; ErrForbidden to anyone else; ErrNotFound for a visit the clinic does not have, a patient not the
// visit's, a patient without the clinic's active care consent, or one whose record the clinic restricts (S7). Audited,
// also when refused.
func (s *Service) SendQuote(ctx context.Context, c Caller, d QuoteDraft) (Quote, error) {
	if !validQuote(d) {
		return Quote{}, ErrInvalid
	}
	if c.OrganizationID == "" || (!c.Admin && c.PractitionerID == "") {
		return Quote{}, s.refuse(ctx, c, auditevent.CodeCreate, ErrForbidden)
	}
	var out Quote
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		q, err := s.quote(ctx, c, d)
		if err != nil {
			return err
		}
		if out, err = s.p.Repo.InsertQuote(ctx, q); err != nil {
			return err
		}
		if err := s.quoteEmail(ctx, out); err != nil {
			return err
		}
		return s.p.Audit.Record(ctx, quoteEvent(c, auditevent.CodeCreate, out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotFound):
		return Quote{}, s.refuse(ctx, c, auditevent.CodeCreate, err)
	case errors.Is(err, ErrInvalid):
		return Quote{}, err
	case err != nil:
		return Quote{}, fmt.Errorf("invoice: send quote: %w", err)
	}
	return s.stated(out), nil
}

// quoteEmail enqueues the e-mail of the quote q in its transaction, unless the visit's patient has no account (a
// profile the clinic registered, docs/clinic-cms W1: no e-mail and no push).
func (s *Service) quoteEmail(ctx context.Context, q Quote) error {
	ok, err := s.p.Visits.PatientHasAccount(ctx, q.AppointmentID)
	if err != nil || !ok {
		return err
	}
	return s.p.Jobs.EnqueueQuoteEmail(ctx, q.AppointmentID, q.OrganizationID)
}

// validQuote reports whether d could be a quote: a patient and a visit, lines as a receipt's, a note within bounds.
func validQuote(d QuoteDraft) bool {
	if d.PatientID == "" || d.AppointmentID == "" || len(d.Picks) == 0 || len(d.Picks) > maxLines || utf8.RuneCountInString(d.Note) > maxNote {
		return false
	}
	for _, p := range d.Picks {
		if p.DefinitionID == "" || p.Quantity <= 0 || p.Quantity > maxQuantity || p.DiscountMinor < 0 {
			return false
		}
	}
	return true
}

// quote is the quote d asks for, unstored: the visit and its patient checked, the lines priced and copied.
func (s *Service) quote(ctx context.Context, c Caller, d QuoteDraft) (Quote, error) {
	a, err := s.p.Visits.Lookup(ctx, d.AppointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return Quote{}, ErrNotFound
	case err != nil:
		return Quote{}, err
	case a.OrganizationID != c.OrganizationID || a.PatientID != d.PatientID:
		return Quote{}, ErrNotFound
	}
	if !c.Admin {
		practitioner, err := s.p.Visits.LookupPractitioner(ctx, d.AppointmentID)
		if err != nil {
			return Quote{}, err
		}
		if practitioner != c.PractitionerID {
			return Quote{}, ErrForbidden
		}
	}
	state, err := s.p.Consents.State(ctx, d.PatientID, c.OrganizationID)
	switch {
	case err != nil:
		return Quote{}, err
	case !state.Active: // the patient sees the quote: only while the clinic has their care consent
		return Quote{}, ErrNotFound
	case state.Restricted: // a quote is no legal obligation: none while the clinic restricts the record (S7)
		return Quote{}, ErrNotFound
	}
	priced, err := s.p.Prices.Lines(ctx, c.OrganizationID, d.Picks)
	switch {
	case errors.Is(err, chargeitemdefinition.ErrNotFound), errors.Is(err, chargeitemdefinition.ErrInvalid):
		return Quote{}, ErrInvalid
	case err != nil:
		return Quote{}, err
	case len(priced) != len(d.Picks):
		return Quote{}, ErrInvalid
	}
	now := s.p.Clock.Now()
	q := Quote{
		IssuedAt: now, ValidUntil: now.Add(quoteValidity), OrganizationID: c.OrganizationID, PatientID: d.PatientID,
		AppointmentID: d.AppointmentID, CreatedByAccountID: c.AccountID, Note: d.Note, Currency: priced[0].Currency,
	}
	for _, l := range priced {
		amount := int64(l.Quantity) * l.UnitPriceMinor
		if l.Currency != q.Currency || l.Quantity <= 0 || l.DiscountMinor < 0 || l.DiscountMinor > amount || l.TotalMinor != amount-l.DiscountMinor {
			return Quote{}, ErrInvalid
		}
		q.Lines = append(q.Lines, Line{
			VATRateBP: l.VATRateBP, ChargeItemDefinitionID: l.DefinitionID, Name: l.Name, UnitPriceMinor: l.UnitPriceMinor,
			DiscountMinor: l.DiscountMinor, TotalMinor: l.TotalMinor, Quantity: l.Quantity,
		})
		q.TotalMinor += l.TotalMinor
	}
	return q, nil
}

// stated is q with its state now: sent while valid, expired after.
func (s *Service) stated(q Quote) Quote {
	q.State = QuoteSent
	if !s.p.Clock.Now().Before(q.ValidUntil) {
		q.State = QuoteExpired
	}
	return q
}

// Quotes lists the clinic's quotes, newest first, with their state (GET /clinic/quotes): every quote to its admin,
// to a practitioner those they sent; with appointmentID only the quotes of that visit. ErrForbidden, audited, to
// anyone else. Audited.
func (s *Service) Quotes(ctx context.Context, c Caller, appointmentID string) ([]Quote, error) {
	if c.OrganizationID == "" || (!c.Admin && c.PractitionerID == "") {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []Quote
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		list, err := s.p.Repo.QuotesOf(ctx, c.OrganizationID)
		if err != nil {
			return err
		}
		for _, q := range list {
			if (c.Admin || q.CreatedByAccountID == c.AccountID) && (appointmentID == "" || q.AppointmentID == appointmentID) {
				out = append(out, s.stated(q))
			}
		}
		return s.p.Audit.Record(ctx, quoteEvent(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: quotes: %w", err)
	}
	return out, nil
}

// ReadQuote returns the quote id of the caller's clinic to its admin and to the practitioner who sent it
// (GET /clinic/quotes/{id}); ErrNotFound to anyone else. Audited, also when refused.
func (s *Service) ReadQuote(ctx context.Context, c Caller, id string) (Quote, error) {
	if c.OrganizationID == "" {
		return Quote{}, s.refuse(ctx, c, auditevent.CodeRead, ErrNotFound)
	}
	var out Quote
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.p.Repo.QuoteByID(ctx, id); err != nil {
			return err
		}
		if out.OrganizationID != c.OrganizationID || (!c.Admin && out.CreatedByAccountID != c.AccountID) {
			return ErrNotFound
		}
		return s.p.Audit.Record(ctx, quoteEvent(c, auditevent.CodeRead, out.ID, auditevent.OutcomeSuccess))
	})
	if err := s.quoteReadError(ctx, c, "read quote", err); err != nil {
		return Quote{}, err
	}
	return s.stated(out), nil
}

// MyQuotes lists the calling patient's quotes from every clinic, newest first (GET /me/quotes). ErrForbidden,
// audited, to staff and to an account without a patient profile.
func (s *Service) MyQuotes(ctx context.Context, c Caller) ([]Quote, error) {
	return s.MyQuotesAt(ctx, c, "")
}

// MyQuotesAt is MyQuotes narrowed to the clinic org when org is set (a clinic-target export, docs/clinic-cms/WAVE6.md
// PX).
func (s *Service) MyQuotesAt(ctx context.Context, c Caller, org string) ([]Quote, error) {
	if c.OrganizationID != "" || c.PatientID == "" {
		return nil, s.refuse(ctx, c, auditevent.CodeSearchType, ErrForbidden)
	}
	var out []Quote
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		list, err := s.p.Repo.QuotesOfPatient(ctx, c.PatientID)
		if err != nil {
			return err
		}
		for _, q := range list {
			if org == "" || q.OrganizationID == org {
				out = append(out, s.stated(q))
			}
		}
		return s.p.Audit.Record(ctx, quoteEvent(c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess))
	})
	if err != nil {
		return nil, fmt.Errorf("invoice: my quotes: %w", err)
	}
	return out, nil
}

// QuotePDF renders the calling patient's own quote id on the clinic's letterhead, in the clinic's default language
// (GET /me/quotes/{id}/pdf): it says it is a quote, not a receipt. ErrNotFound for any other quote. Audited as the
// read, also when refused.
func (s *Service) QuotePDF(ctx context.Context, c Caller, id string) ([]byte, error) {
	if c.OrganizationID != "" || c.PatientID == "" {
		return nil, s.refuse(ctx, c, auditevent.CodeRead, ErrNotFound)
	}
	var q Quote
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		if q, err = s.p.Repo.QuoteByID(ctx, id); err != nil {
			return err
		}
		if q.PatientID != c.PatientID { // the owner check does not rest on row-level security alone
			return ErrNotFound
		}
		return s.p.Audit.Record(ctx, quoteEvent(c, auditevent.CodeRead, q.ID, auditevent.OutcomeSuccess))
	})
	if err := s.quoteReadError(ctx, c, "quote pdf", err); err != nil {
		return nil, err
	}
	return s.renderQuote(ctx, q)
}

// ClinicQuotePDF renders the quote id of the caller's clinic as QuotePDF does, to those ReadQuote shows it to
// (GET /clinic/quotes/{id}/pdf, package M4); ErrNotFound to anyone else. Audited as the read, also when refused.
func (s *Service) ClinicQuotePDF(ctx context.Context, c Caller, id string) ([]byte, error) {
	q, err := s.ReadQuote(ctx, c, id)
	if err != nil {
		return nil, err
	}
	return s.renderQuote(ctx, q)
}

// renderQuote renders q on its clinic's letterhead, outside the quote's transaction: the clinic is its own scope.
func (s *Service) renderQuote(ctx context.Context, q Quote) ([]byte, error) {
	clinic, err := s.p.Clinics.Clinic(ctx, q.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invoice: quote pdf: %w", err)
	}
	b, err := pdfdoc.Render(quoteDocument(q, clinic))
	if err != nil {
		return nil, fmt.Errorf("invoice: quote pdf: %w", err)
	}
	return b, nil
}

// quoteReadError audits a read of a quote that found nothing c may see and wraps any other failure.
func (s *Service) quoteReadError(ctx context.Context, c Caller, op string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return s.refuse(ctx, c, auditevent.CodeRead, ErrNotFound)
	case err != nil:
		return fmt.Errorf("invoice: %s: %w", op, err)
	}
	return nil
}

// quoteEvent is the AuditEvent of c's interaction with the quote id (none for a list): as a receipt's, on Quote/<id>.
func quoteEvent(c Caller, interaction, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(c, interaction, "", outcome)
	if id != "" {
		e.EntityReference = "Quote/" + id
	}
	return e
}

// quoteDocument lays the quote out: the letterhead, "Quote" with its date, its validity and that it is no receipt, its lines, the total, the
// and the note. In the clinic's default language (the receipt's catalogs), times in its zone.
func quoteDocument(q Quote, clinic organization.Organization) pdfdoc.Document {
	l := labelsOf(clinic.DefaultLanguage)
	zone, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		zone = time.UTC
	}
	items := make([]string, len(q.Lines))
	for i, line := range q.Lines {
		items[i] = l.line(line, q.Currency)
	}
	sections := []pdfdoc.Section{
		{Heading: l.get("quote"), Paragraphs: []string{
			l.get("issued") + ": " + q.IssuedAt.In(zone).Format("02.01.2006 15:04"),
			l.get("quote_valid_until") + ": " + q.ValidUntil.In(zone).Format("02.01.2006"),
			l.get("quote_not_receipt"),
		}},
		{Heading: l.get("items"), Items: items},
		{Heading: l.get("total"), Paragraphs: []string{l.money(q.TotalMinor) + " " + q.Currency}},
	}
	if q.Note != "" {
		sections = append(sections, pdfdoc.Section{Heading: l.get("quote_note"), Paragraphs: []string{q.Note}})
	}
	return pdfdoc.Document{Created: q.IssuedAt, Title: clinic.Name, Sections: sections}
}
