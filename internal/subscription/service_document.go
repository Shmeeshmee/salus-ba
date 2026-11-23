package subscription

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// PDF renders the clinic's invoice id for its admin (GET /clinic/subscription/invoices/{id}/pdf). Refusals as
// Current's; ErrNotFound for an invoice that is not the clinic's. Not audited: nothing in it is personal data.
func (s *Service) PDF(ctx context.Context, c Caller, id string) ([]byte, error) {
	if err := adminOf(c); err != nil {
		return nil, err
	}
	_, pdf, _, err := s.render(ctx, c.OrganizationID, id)
	return pdf, err
}

// SendInvoice sends the clinic org's invoice id to its billing e-mail with the PDF attached, in the clinic's default
// language (the subscription_invoice_email job). A clinic without a billing e-mail gets none: the admin downloads the
// invoice from the membership screen. ErrNotFound for an invoice that is not the clinic's.
func (s *Service) SendInvoice(ctx context.Context, org, id string) error {
	inv, pdf, clinic, err := s.render(ctx, org, id)
	if err != nil {
		return err
	}
	to, err := s.Billing.ForClinic(ctx, org)
	if err != nil {
		return fmt.Errorf("subscription: send invoice: %w", err)
	}
	if to.BillingEmail == "" {
		return nil
	}
	err = s.Mailer.SendMembershipInvoice(ctx, notification.MembershipInvoice{
		To: to.BillingEmail, Language: languageOf(clinic.DefaultLanguage), Clinic: clinic.Name, Number: inv.Number, PDF: pdf,
	})
	if err != nil {
		return fmt.Errorf("subscription: send invoice: %w", err)
	}
	return nil
}

// render reads the clinic org's invoice id, the clinic and its billing details, and lays the invoice out as PDF.
func (s *Service) render(ctx context.Context, org, id string) (Invoice, []byte, organization.Organization, error) {
	var inv Invoice
	var clinic organization.Organization
	var buyer organization.BillingDetails
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		all, err := s.Repo.Invoices(ctx, org) // a clinic has one a month: no query of its own
		if err != nil {
			return err
		}
		found := false
		for _, in := range all {
			if in.ID == id {
				inv, found = in, true
			}
		}
		if !found {
			return ErrNotFound
		}
		if clinic, err = s.Clinics.Clinic(ctx, org); err != nil {
			return err
		}
		buyer, err = s.Billing.ForClinic(ctx, org)
		return err
	})
	if err != nil {
		return Invoice{}, nil, clinic, wrap("invoice pdf", err)
	}
	pdf, err := pdfdoc.Render(document(inv, clinic, buyer, s.Seller))
	if err != nil {
		return Invoice{}, nil, clinic, fmt.Errorf("subscription: invoice pdf: %w", err)
	}
	return inv, pdf, clinic, nil
}

// document lays the invoice out: the seller, the invoice's number and dates, the clinic as the buyer, the month's
// line, the total and how to pay. In the clinic's language (Bosnian for one outside the five staff languages), the
// dates in its zone.
func document(inv Invoice, clinic organization.Organization, buyer organization.BillingDetails, seller Seller) pdfdoc.Document {
	l := documentLabels[languageOf(clinic.DefaultLanguage)]
	zone, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		zone = time.UTC
	}
	date := func(t time.Time) string { return t.In(zone).Format("02.01.2006") }
	facts := []string{l.get("issued") + ": " + date(inv.IssuedAt), l.get("due") + ": " + date(inv.DueAt)}
	if !inv.PaidAt.IsZero() {
		facts = append(facts, l.get("paid")+": "+date(inv.PaidAt))
	}
	name := buyer.LegalName
	if name == "" {
		name = clinic.Name
	}
	buyerLines := []string{name}
	if buyer.TaxID != "" {
		buyerLines = append(buyerLines, l.get("taxId")+": "+buyer.TaxID)
	}
	if clinic.AddressLine != "" {
		buyerLines = append(buyerLines, strings.TrimSuffix(clinic.AddressLine+", "+clinic.PostalCode, ", "))
	}
	var unit int64
	if inv.Seats > 0 {
		unit = inv.AmountMinor / int64(inv.Seats)
	}
	last := inv.PeriodEnd.AddDate(0, 0, -1) // the period ends at the next month's first midnight
	number := strconv.FormatInt(inv.Number, 10)
	item := l.get("item", "from", date(inv.PeriodStart), "to", date(last)) +
		fmt.Sprintf(": %d × %s = %s %s", inv.Seats, l.money(unit), l.money(inv.AmountMinor), inv.Currency)
	return pdfdoc.Document{
		Created: inv.IssuedAt,
		Title:   seller.Name,
		Meta:    []string{l.get("taxId") + ": " + seller.TaxID, seller.Address},
		Sections: []pdfdoc.Section{
			{Heading: l.get("invoice", "number", number), Paragraphs: facts},
			{Heading: l.get("buyer"), Paragraphs: buyerLines},
			{Heading: l.get("items"), Items: []string{item}},
			{Heading: l.get("total"), Paragraphs: []string{l.money(inv.AmountMinor) + " " + inv.Currency}},
			{Heading: l.get("payment"), Paragraphs: []string{l.get("transfer", "account", seller.Account, "number", number)}},
		},
	}
}

// The invoice's printed words, one catalog per staff language (templates/pdf/<language>.json, flat, with {name}
// placeholders only), checked by make i18n-check as the receipt's are (internal/invoice). The invoice is Salus's
// document to a clinic, read by its staff; nothing else prints them.
//
//go:embed templates/pdf
var labelsFS embed.FS

// staffLanguages are the catalog's languages; the first is the fallback.
var staffLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// labels are the invoice's printed words in one staff language.
type labels struct {
	text  map[string]string
	comma bool // a decimal comma
}

// documentLabels is read once from the binary: a broken file is a programming error the tests catch, so it panics.
var documentLabels = func() map[string]labels {
	out := map[string]labels{}
	for _, lang := range staffLanguages {
		b, err := fs.ReadFile(labelsFS, "templates/pdf/"+lang+".json")
		if err != nil {
			panic(err)
		}
		var text map[string]string
		if err := json.Unmarshal(b, &text); err != nil {
			panic(fmt.Sprintf("subscription: templates/pdf/%s.json: %v", lang, err))
		}
		out[lang] = labels{text: text, comma: lang != "en"}
	}
	return out
}()

// languageOf is the staff language the invoice is written in: the clinic's default, Bosnian for any other.
func languageOf(language string) string {
	if _, ok := documentLabels[language]; ok {
		return language
	}
	return staffLanguages[0]
}

// get returns the label key with its {name} placeholders filled from pairs of name and value.
func (l labels) get(key string, pairs ...string) string {
	s := l.text[key]
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, "{"+pairs[i]+"}", pairs[i+1])
	}
	return s
}

// money writes an amount of the minor unit with two decimals: 30000 is 300,00 (300.00 in English).
func (l labels) money(minor int64) string {
	sign, sep := "", "."
	if minor < 0 {
		sign, minor = "-", -minor
	}
	if l.comma {
		sep = ","
	}
	return fmt.Sprintf("%s%d%s%02d", sign, minor/100, sep, minor%100)
}
