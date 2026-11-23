package subscription

import (
	"context"
	"fmt"
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
	facts := []string{l.issued + ": " + date(inv.IssuedAt), l.due + ": " + date(inv.DueAt)}
	if !inv.PaidAt.IsZero() {
		facts = append(facts, l.paid+": "+date(inv.PaidAt))
	}
	name := buyer.LegalName
	if name == "" {
		name = clinic.Name
	}
	buyerLines := []string{name}
	if buyer.TaxID != "" {
		buyerLines = append(buyerLines, l.taxID+": "+buyer.TaxID)
	}
	if clinic.AddressLine != "" {
		buyerLines = append(buyerLines, strings.TrimSuffix(clinic.AddressLine+", "+clinic.PostalCode, ", "))
	}
	var unit int64
	if inv.Seats > 0 {
		unit = inv.AmountMinor / int64(inv.Seats)
	}
	last := inv.PeriodEnd.AddDate(0, 0, -1) // the period ends at the next month's first midnight
	item := fmt.Sprintf(l.item, date(inv.PeriodStart), date(last)) +
		fmt.Sprintf(": %d × %s = %s %s", inv.Seats, l.money(unit), l.money(inv.AmountMinor), inv.Currency)
	return pdfdoc.Document{
		Created: inv.IssuedAt,
		Title:   seller.Name,
		Meta:    []string{l.taxID + ": " + seller.TaxID, seller.Address},
		Sections: []pdfdoc.Section{
			{Heading: fmt.Sprintf(l.invoice, inv.Number), Paragraphs: facts},
			{Heading: l.buyer, Paragraphs: buyerLines},
			{Heading: l.items, Items: []string{item}},
			{Heading: l.total, Paragraphs: []string{l.money(inv.AmountMinor) + " " + inv.Currency}},
			{Heading: l.payment, Paragraphs: []string{fmt.Sprintf(l.transfer, seller.Account, inv.Number)}},
		},
	}
}

// languageOf is the staff language the invoice is written in: the clinic's default, Bosnian for any other.
func languageOf(language string) string {
	if _, ok := documentLabels[language]; ok {
		return language
	}
	return "bs"
}

// labels are the invoice's printed words in one staff language. The invoice is Salus's document to a clinic, read by
// its staff; nothing else prints them.
type labels struct {
	invoice, item, transfer string // formats: the number; the month's first and last day; the account and the number
	issued, due, paid       string
	taxID, buyer, items     string
	total, payment          string
	comma                   bool // a decimal comma
}

var documentLabels = map[string]labels{
	"en": {
		invoice: "Invoice no. %d", item: "Membership %s to %s, active practitioners", transfer: "Bank transfer to account %s, reference %d.",
		issued: "Issued", due: "Due", paid: "Paid", taxID: "Tax ID (JIB)", buyer: "Buyer", items: "Items", total: "Total", payment: "Payment",
	},
	"bs": {
		invoice: "Račun br. %d", item: "Članarina %s do %s, aktivni ljekari", transfer: "Uplata na račun %s, poziv na broj %d.",
		issued: "Izdato", due: "Rok plaćanja", paid: "Plaćeno", taxID: "JIB", buyer: "Kupac", items: "Stavke", total: "Ukupno", payment: "Plaćanje",
		comma: true,
	},
	"hr": {
		invoice: "Račun br. %d", item: "Članarina %s do %s, aktivni liječnici", transfer: "Uplata na račun %s, poziv na broj %d.",
		issued: "Izdano", due: "Rok plaćanja", paid: "Plaćeno", taxID: "JIB", buyer: "Kupac", items: "Stavke", total: "Ukupno", payment: "Plaćanje",
		comma: true,
	},
	"sr-Latn": {
		invoice: "Račun br. %d", item: "Članarina %s do %s, aktivni ljekari", transfer: "Uplata na račun %s, poziv na broj %d.",
		issued: "Izdato", due: "Rok plaćanja", paid: "Plaćeno", taxID: "JIB", buyer: "Kupac", items: "Stavke", total: "Ukupno", payment: "Plaćanje",
		comma: true,
	},
	"sr-Cyrl": {
		invoice: "Рачун бр. %d", item: "Чланарина %s до %s, активни љекари", transfer: "Уплата на рачун %s, позив на број %d.",
		issued: "Издато", due: "Рок плаћања", paid: "Плаћено", taxID: "ЈИБ", buyer: "Купац", items: "Ставке", total: "Укупно", payment: "Плаћање",
		comma: true,
	},
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
