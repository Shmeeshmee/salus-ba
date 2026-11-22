package invoice

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// PDF renders a receipt on the clinic's letterhead (GET /clinic/receipts/{id}/pdf, GET /me/receipts/{id}/pdf): the
// clinic's name, the seller's legal name, tax id and address as they were at issue, the lines, the total and, once
// fiscalized, the fiscal number, the verification code and the QR payload. In the clinic's default language, with the
// times in its zone; the patient's is the same document. It names no patient. For whoever may read the receipt (Read;
// a patient their own); audited as that read.
func (s *Service) PDF(ctx context.Context, c Caller, id string) ([]byte, error) {
	var inv Invoice
	var err error
	if c.OrganizationID == "" && c.PatientID != "" {
		inv, err = s.own(ctx, c, id)
	} else {
		inv, err = s.Read(ctx, c, id)
	}
	if err != nil {
		return nil, err
	}
	clinic, err := s.p.Clinics.Clinic(ctx, inv.OrganizationID) // outside the receipt's transaction: its own scope
	if err != nil {
		return nil, fmt.Errorf("invoice: pdf: %w", err)
	}
	var reversed int64
	if inv.ReversesID != "" {
		if reversed, err = s.numberOf(ctx, c, inv); err != nil {
			return nil, fmt.Errorf("invoice: pdf: %w", err)
		}
	}
	b, err := pdfdoc.Render(document(inv, clinic, reversed))
	if err != nil {
		return nil, fmt.Errorf("invoice: pdf: %w", err)
	}
	return b, nil
}

// numberOf returns the number of the receipt the reversal rev reverses, in the scope c read rev in: it is of the same
// clinic and the same buyer. Unaudited: the number is printed on rev, whose read is.
func (s *Service) numberOf(ctx context.Context, c Caller, rev Invoice) (int64, error) {
	var n int64
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if c.OrganizationID == "" {
			err = s.p.Repo.SetPatient(ctx, c.PatientID)
		} else {
			err = s.p.Repo.SetOrganization(ctx, c.OrganizationID)
		}
		if err != nil {
			return err
		}
		orig, err := s.p.Repo.ByID(ctx, rev.ReversesID)
		n = orig.Number
		return err
	})
	return n, err
}

// document lays the receipt out: the letterhead, the receipt's own data, its lines, the total and the fiscal data.
func document(inv Invoice, clinic organization.Organization, reversed int64) pdfdoc.Document {
	l := labelsOf(clinic.DefaultLanguage)
	zone, err := time.LoadLocation(clinic.TimeZone)
	if err != nil {
		zone = time.UTC
	}
	heading := fmt.Sprintf(l.receipt, inv.Number)
	switch {
	case inv.ReversesID != "":
		heading = fmt.Sprintf(l.reversal, inv.Number, reversed)
	case inv.Kind == KindInvoice:
		heading = fmt.Sprintf(l.invoice, inv.Number)
	}
	facts := []string{
		l.issued + ": " + inv.IssuedAt.In(zone).Format("02.01.2006 15:04"),
		l.payment + ": " + l.methods[inv.PaymentMethod],
	}
	if inv.CompanyName != "" {
		buyer := l.buyer + ": " + inv.CompanyName
		if inv.CompanyTaxID != "" {
			buyer += ", " + l.taxID + " " + inv.CompanyTaxID
		}
		facts = append(facts, buyer)
	}
	items := make([]string, len(inv.Lines))
	for i, line := range inv.Lines {
		items[i] = l.line(line, inv.Currency)
	}
	fiscalData := []string{l.unreported}
	if inv.FiscalStatus == FiscalFiscalized {
		fiscalData = []string{l.fiscalID + ": " + inv.FiscalID, l.fiscalCode + ": " + inv.FiscalCode, "QR: " + inv.FiscalQR}
	}
	return pdfdoc.Document{
		Created: inv.IssuedAt,
		Title:   clinic.Name,
		Meta:    []string{inv.SellerLegalName, l.taxID + ": " + inv.SellerTaxID, inv.SellerAddress},
		Sections: []pdfdoc.Section{
			{Heading: heading, Paragraphs: facts},
			{Heading: l.items, Items: items},
			{Heading: l.total, Paragraphs: []string{l.money(inv.TotalMinor) + " " + inv.Currency}},
			{Heading: l.fiscal, Paragraphs: fiscalData},
		},
	}
}

// labels are the receipt's printed words in one language. The receipt is the clinic's document, so it is printed in
// one of the five staff languages (the clinic's default); the catalog is here because nothing else prints them.
type labels struct {
	methods                                     map[PaymentMethod]string
	receipt, invoice, reversal                  string // formats: the number; the reversal's, then the reversed one's
	issued, payment, buyer, taxID, items, total string
	discount, vat, exempt                       string
	fiscal, fiscalID, fiscalCode, unreported    string
	comma                                       bool // a decimal comma
}

var catalog = map[string]labels{
	"en": {
		receipt: "Receipt no. %d", invoice: "Invoice no. %d", reversal: "Receipt no. %d, reversal of receipt no. %d",
		issued: "Issued", payment: "Payment", buyer: "Buyer", taxID: "Tax ID (JIB)", items: "Items", total: "Total",
		discount: "discount", vat: "VAT", exempt: "VAT exempt",
		fiscal: "Fiscal data", fiscalID: "Fiscal number", fiscalCode: "Verification code", unreported: "Not yet reported to the tax authority.",
		methods: map[PaymentMethod]string{PaymentCash: "Cash", PaymentCard: "Card", PaymentTransfer: "Bank transfer"},
	},
	"bs": {
		receipt: "Račun br. %d", invoice: "Faktura br. %d", reversal: "Račun br. %d, storno računa br. %d",
		issued: "Izdato", payment: "Način plaćanja", buyer: "Kupac", taxID: "JIB", items: "Stavke", total: "Ukupno",
		discount: "popust", vat: "PDV", exempt: "oslobođeno PDV-a",
		fiscal: "Fiskalni podaci", fiscalID: "Fiskalni broj", fiscalCode: "Kontrolni kod", unreported: "Još nije prijavljeno poreznoj upravi.",
		methods: map[PaymentMethod]string{PaymentCash: "Gotovina", PaymentCard: "Kartica", PaymentTransfer: "Bankovni prijenos"},
		comma:   true,
	},
	"hr": {
		receipt: "Račun br. %d", invoice: "Faktura br. %d", reversal: "Račun br. %d, storno računa br. %d",
		issued: "Izdano", payment: "Način plaćanja", buyer: "Kupac", taxID: "JIB", items: "Stavke", total: "Ukupno",
		discount: "popust", vat: "PDV", exempt: "oslobođeno PDV-a",
		fiscal: "Fiskalni podaci", fiscalID: "Fiskalni broj", fiscalCode: "Kontrolni kod", unreported: "Još nije prijavljeno poreznoj upravi.",
		methods: map[PaymentMethod]string{PaymentCash: "Gotovina", PaymentCard: "Kartica", PaymentTransfer: "Bankovni prijenos"},
		comma:   true,
	},
	"sr-Latn": {
		receipt: "Račun br. %d", invoice: "Faktura br. %d", reversal: "Račun br. %d, storno računa br. %d",
		issued: "Izdato", payment: "Način plaćanja", buyer: "Kupac", taxID: "JIB", items: "Stavke", total: "Ukupno",
		discount: "popust", vat: "PDV", exempt: "oslobođeno PDV-a",
		fiscal: "Fiskalni podaci", fiscalID: "Fiskalni broj", fiscalCode: "Kontrolni kod", unreported: "Još nije prijavljeno poreskoj upravi.",
		methods: map[PaymentMethod]string{PaymentCash: "Gotovina", PaymentCard: "Kartica", PaymentTransfer: "Bankovni prenos"},
		comma:   true,
	},
	"sr-Cyrl": {
		receipt: "Рачун бр. %d", invoice: "Фактура бр. %d", reversal: "Рачун бр. %d, сторно рачуна бр. %d",
		issued: "Издато", payment: "Начин плаћања", buyer: "Купац", taxID: "ЈИБ", items: "Ставке", total: "Укупно",
		discount: "попуст", vat: "ПДВ", exempt: "ослобођено ПДВ-а",
		fiscal: "Фискални подаци", fiscalID: "Фискални број", fiscalCode: "Контролни код", unreported: "Још није пријављено пореској управи.",
		methods: map[PaymentMethod]string{PaymentCash: "Готовина", PaymentCard: "Картица", PaymentTransfer: "Банковни пренос"},
		comma:   true,
	},
}

// labelsOf returns the labels of the staff language, Bosnian (the clinics' default) for any other.
func labelsOf(language string) labels {
	if l, ok := catalog[language]; ok {
		return l
	}
	return catalog["bs"]
}

// money writes an amount of the minor unit with two decimals: 10670 is 106,70 (106.70 in English).
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

// line writes one line: its name, quantity times unit price, the discount when there is one, the VAT treatment and
// the line's total.
func (l labels) line(line Line, currency string) string {
	s := fmt.Sprintf("%s: %d × %s", line.Name, line.Quantity, l.money(line.UnitPriceMinor))
	if line.DiscountMinor != 0 {
		s += ", " + l.discount + " " + l.money(line.DiscountMinor)
	}
	if line.VATRateBP == nil {
		s += ", " + l.exempt
	} else {
		s += fmt.Sprintf(", %s %s %%", l.vat, l.money(int64(*line.VATRateBP)))
	}
	return s + " = " + l.money(line.TotalMinor) + " " + currency
}
