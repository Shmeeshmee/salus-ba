package invoice

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// text is everything the document prints, a line each.
func printed(d pdfdoc.Document) string {
	lines := append([]string{d.Title}, d.Meta...)
	for _, s := range d.Sections {
		lines = slices.Concat(lines, []string{s.Heading}, s.Paragraphs, s.Items)
	}
	return strings.Join(lines, "\n")
}

func sample() Invoice {
	vat := int32(1700)
	return Invoice{
		IssuedAt: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC), ID: "inv-1", OrganizationID: "org-1", PatientID: "patient-a",
		SellerLegalName: "Synthetic clinic d.o.o.", SellerTaxID: "4200000000000", SellerAddress: "Placeholder street 1, 71000 Sarajevo",
		Kind: KindReceipt, PaymentMethod: PaymentCash, Currency: "BAM", Number: 12, TotalMinor: 10670, FiscalStatus: FiscalPending,
		Lines: []Line{
			{Name: "Placeholder examination", Quantity: 2, UnitPriceMinor: 5000, DiscountMinor: 500, TotalMinor: 9500},
			{Name: "Placeholder laboratory item", Quantity: 1, UnitPriceMinor: 1170, TotalMinor: 1170, VATRateBP: &vat},
		},
	}
}

var clinic = organization.Organization{Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo", DefaultLanguage: "en"}

// The PDF's text: the letterhead (the clinic's name, the seller's legal name, tax id and address at issue), the
// number, the time in the clinic's zone, the lines, the total; no fiscal data before the report, and no patient.
func TestDocument_LetterheadLinesAndTotal(t *testing.T) {
	got := printed(document(sample(), clinic, 0))
	for _, want := range []string{
		"Synthetic clinic\nSynthetic clinic d.o.o.\nTax ID (JIB): 4200000000000\nPlaceholder street 1, 71000 Sarajevo",
		"Receipt no. 12", "Issued: 21.11.2026 10:00", "Payment: Cash",
		"Placeholder examination: 2 × 50.00, discount 5.00, VAT exempt = 95.00 BAM",
		"Placeholder laboratory item: 1 × 11.70, VAT 17.00 % = 11.70 BAM",
		"Total\n106.70 BAM", "Not yet reported to the tax authority.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the receipt lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "patient-a") || strings.Contains(got, "Fiscal number") {
		t.Errorf("the receipt names the patient or a fiscal number it does not have:\n%s", got)
	}
}

// Once fiscalized the receipt prints the fiscal number, the verification code and the QR payload; an invoice names
// the company; a reversal names the receipt it reverses, with negative amounts.
func TestDocument_FiscalDataCompanyAndReversal(t *testing.T) {
	inv := sample()
	inv.Kind, inv.PatientID, inv.CompanyName, inv.CompanyTaxID = KindInvoice, "", "Synthetic company d.o.o.", "4200000000001"
	inv.FiscalStatus, inv.FiscalID, inv.FiscalCode, inv.FiscalQR = FiscalFiscalized, "TEST-1", "code-1", "TEST:qr-1"
	got := printed(document(inv, clinic, 0))
	for _, want := range []string{"Invoice no. 12", "Buyer: Synthetic company d.o.o., Tax ID (JIB) 4200000000001",
		"Fiscal number: TEST-1", "Verification code: code-1", "QR: TEST:qr-1"} {
		if !strings.Contains(got, want) {
			t.Errorf("the invoice lacks %q:\n%s", want, got)
		}
	}
	rev := reversal(sample(), PaymentCard)
	rev.Number = 13
	got = printed(document(rev, clinic, 12))
	for _, want := range []string{"Receipt no. 13, reversal of receipt no. 12", "Placeholder examination: -2 × 50.00, discount 5.00, VAT exempt = -95.00 BAM", "Total\n-106.70 BAM"} {
		if !strings.Contains(got, want) {
			t.Errorf("the reversal lacks %q:\n%s", want, got)
		}
	}
}

// Every staff language has every label of the English catalog and no other, renders with the bundled fonts (Cyrillic
// too), and an unknown language falls back to Bosnian.
func TestDocument_EveryStaffLanguageRenders(t *testing.T) {
	en := labelsOf("en")
	for _, language := range []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de"} {
		c := clinic
		c.DefaultLanguage = language
		d := document(sample(), c, 0)
		l := labelsOf(language)
		if len(l.text) != len(en.text) {
			t.Errorf("%s: %d labels, want %d", language, len(l.text), len(en.text))
		}
		for key := range en.text {
			if l.text[key] == "" {
				t.Errorf("%s: the label %s is missing", language, key)
			}
		}
		if b, err := pdfdoc.Render(d); err != nil || !strings.HasPrefix(string(b), "%PDF-") {
			t.Errorf("%s: Render() = %v", language, err)
		}
	}
	if got := printed(document(sample(), organization.Organization{DefaultLanguage: "de"}, 0)); !strings.Contains(got, "Račun br. 12") {
		t.Errorf("an unknown language is not Bosnian:\n%s", got)
	}
}

// A company's invoice is headed "Račun" in Croatian and "Faktura" in Bosnian and Serbian.
func TestDocument_ACompanysInvoiceInCroatianIsARacun(t *testing.T) {
	inv := sample()
	inv.Kind, inv.PatientID, inv.CompanyName = KindInvoice, "", "Synthetic company d.o.o."
	for language, want := range map[string]string{"hr": "Račun br. 12", "bs": "Faktura br. 12", "sr-Latn": "Faktura br. 12", "sr-Cyrl": "Фактура бр. 12"} {
		c := clinic
		c.DefaultLanguage = language
		if got := printed(document(inv, c, 0)); !strings.Contains(got, want) {
			t.Errorf("%s: the invoice lacks %q:\n%s", language, want, got)
		}
	}
}

// A quote's PDF: the clinic's name, "Quote", the date of issue and the validity in the clinic's zone, the lines, the
// total, the note, and that it is no receipt and nothing was charged. No number, no fiscal data, no patient.
func TestQuoteDocument_SaysItIsNoReceipt(t *testing.T) {
	inv := sample()
	q := Quote{
		IssuedAt: inv.IssuedAt, ValidUntil: inv.IssuedAt.Add(30 * 24 * time.Hour), ID: "quote-1", OrganizationID: "org-1",
		PatientID: "patient-a", Note: "Placeholder note", Currency: "BAM", Lines: inv.Lines, TotalMinor: inv.TotalMinor,
	}
	got := printed(quoteDocument(q, clinic))
	for _, want := range []string{
		"Synthetic clinic", "Quote", "Issued: 21.11.2026 10:00", "Valid until: 21.12.2026",
		"Placeholder examination: 2 × 50.00, discount 5.00, VAT exempt = 95.00 BAM", "Total\n106.70 BAM",
		"Placeholder note", "This is a quote, not a receipt. Nothing has been charged.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the quote lacks %q:\n%s", want, got)
		}
	}
	for _, not := range []string{"patient-a", "Receipt no.", "Fiscal", "tax authority", "Payment"} {
		if strings.Contains(got, not) {
			t.Errorf("the quote prints %q:\n%s", not, got)
		}
	}
}

// A receipt of an in-person visit names where the service was given (docs/clinic-cms/WAVE6.md U2): the visit's
// location, below the payment; the seller stays the clinic's registered details (the owner, 2026-11-25: the seller's
// address is a fiscal matter). A reversal keeps it; a receipt without a place prints no line.
func TestDocument_ThePlaceOfServiceOfAnInPersonVisit(t *testing.T) {
	inv := sample()
	inv.PlaceOfService = "Synthetic annex, Synthetic annex 2, 88000 Mostar"
	got := printed(document(inv, clinic, 0))
	if !strings.Contains(got, "Payment: Cash\nPlace of service: Synthetic annex, Synthetic annex 2, 88000 Mostar") ||
		!strings.Contains(got, "Synthetic clinic d.o.o.\nTax ID (JIB): 4200000000000\nPlaceholder street 1, 71000 Sarajevo") {
		t.Errorf("the receipt lacks the place of service below the payment, or the seller's details:\n%s", got)
	}
	if rev := reversal(inv, PaymentCard); rev.PlaceOfService != inv.PlaceOfService {
		t.Errorf("the reversal's place %q, want the receipt's", rev.PlaceOfService)
	}
	if got := printed(document(sample(), clinic, 0)); strings.Contains(got, "Place of service") {
		t.Errorf("a receipt without a place prints one:\n%s", got)
	}
}
