package subscription

import (
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// The membership invoice names the seller, the clinic as the buyer, the month, the seats, the amount and how to pay,
// in the clinic's language and its zone.
func TestDocument_LaysOutTheMembershipInvoice(t *testing.T) {
	sarajevo, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	inv := Invoice{
		PeriodStart: time.Date(2026, 12, 1, 0, 0, 0, 0, sarajevo), PeriodEnd: time.Date(2027, 1, 1, 0, 0, 0, 0, sarajevo),
		IssuedAt: time.Date(2026, 12, 1, 1, 0, 0, 0, time.UTC), DueAt: time.Date(2026, 12, 15, 1, 0, 0, 0, time.UTC),
		Currency: "BAM", Number: 7, AmountMinor: 30000, Seats: 3, Status: InvoiceOpen,
	}
	clinic := organization.Organization{Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo", DefaultLanguage: "en", AddressLine: "Synthetic street 1", PostalCode: "71000"}
	buyer := organization.BillingDetails{LegalName: "Synthetic clinic d.o.o.", TaxID: "4200000000001"}
	seller := Seller{Name: "[SELLER]", TaxID: "[SELLER TAX ID]", Address: "[SELLER ADDRESS]", Account: "[SELLER ACCOUNT]"}

	d := document(inv, clinic, buyer, seller)
	text := d.Title + "\n" + strings.Join(d.Meta, "\n")
	for _, s := range d.Sections {
		text += "\n" + s.Heading + "\n" + strings.Join(s.Paragraphs, "\n") + "\n" + strings.Join(s.Items, "\n")
	}
	for _, want := range []string{
		"[SELLER]", "[SELLER TAX ID]", "[SELLER ADDRESS]", "Invoice no. 7", "Issued: 01.12.2026", "Due: 15.12.2026",
		"Synthetic clinic d.o.o.", "4200000000001", "Synthetic street 1, 71000",
		"Membership 01.12.2026 to 31.12.2026, active practitioners: 3 × 100.00 = 300.00 BAM", "300.00 BAM", "[SELLER ACCOUNT]", "reference 7",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the invoice lacks %q:\n%s", want, text)
		}
	}

	inv.PaidAt, inv.Status = time.Date(2026, 12, 10, 9, 0, 0, 0, time.UTC), InvoicePaid
	clinic.DefaultLanguage = "de" // not a staff language: Bosnian
	d = document(inv, clinic, organization.BillingDetails{}, seller)
	text = d.Title
	for _, s := range d.Sections {
		text += "\n" + s.Heading + "\n" + strings.Join(s.Paragraphs, "\n") + "\n" + strings.Join(s.Items, "\n")
	}
	for _, want := range []string{"Račun br. 7", "Synthetic clinic", "300,00 BAM", "Plaćeno: 10.12.2026"} {
		if !strings.Contains(text, want) {
			t.Errorf("the paid invoice in Bosnian lacks %q:\n%s", want, text)
		}
	}
}
