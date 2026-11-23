package invoice

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
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
	number := strconv.FormatInt(inv.Number, 10)
	heading := l.get("receipt", "number", number)
	switch {
	case inv.ReversesID != "":
		heading = l.get("reversal", "number", number, "reversed", strconv.FormatInt(reversed, 10))
	case inv.Kind == KindInvoice:
		heading = l.get("invoice", "number", number)
	}
	facts := []string{
		l.get("issued") + ": " + inv.IssuedAt.In(zone).Format("02.01.2006 15:04"),
		l.get("payment") + ": " + l.method(inv.PaymentMethod),
	}
	if inv.CompanyName != "" {
		buyer := l.get("buyer") + ": " + inv.CompanyName
		if inv.CompanyTaxID != "" {
			buyer += ", " + l.get("tax_id") + " " + inv.CompanyTaxID
		}
		facts = append(facts, buyer)
	}
	items := make([]string, len(inv.Lines))
	for i, line := range inv.Lines {
		items[i] = l.line(line, inv.Currency)
	}
	fiscalData := []string{l.get("unreported")}
	if inv.FiscalStatus == FiscalFiscalized {
		fiscalData = []string{l.get("fiscal_id") + ": " + inv.FiscalID, l.get("fiscal_code") + ": " + inv.FiscalCode, "QR: " + inv.FiscalQR}
	}
	return pdfdoc.Document{
		Created: inv.IssuedAt,
		Title:   clinic.Name,
		Meta:    []string{inv.SellerLegalName, l.get("tax_id") + ": " + inv.SellerTaxID, inv.SellerAddress},
		Sections: []pdfdoc.Section{
			{Heading: heading, Paragraphs: facts},
			{Heading: l.get("items"), Items: items},
			{Heading: l.get("total"), Paragraphs: []string{l.money(inv.TotalMinor) + " " + inv.Currency}},
			{Heading: l.get("fiscal"), Paragraphs: fiscalData},
		},
	}
}

// labelsFS holds the receipt's printed words, templates/pdf/<language>.json in each of the five staff languages: the
// receipt is the clinic's document, printed in the clinic's default language. ICU messages with simple placeholders
// only ({number}, {reversed}), checked by make i18n-check; the report's CSV headers are there too.
//
//go:embed templates/pdf
var labelsFS embed.FS

// staffLanguages are the catalog's languages; the first is the fallback.
var staffLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// labels are the receipt's printed words in one language.
type labels struct {
	text  map[string]string
	comma bool // a decimal comma
}

// catalog is read once from the binary: a broken file is a programming error the tests catch, so it panics.
var catalog = func() map[string]labels {
	out := map[string]labels{}
	for _, lang := range staffLanguages {
		b, err := fs.ReadFile(labelsFS, "templates/pdf/"+lang+".json")
		if err != nil {
			panic(err)
		}
		var text map[string]string
		if err := json.Unmarshal(b, &text); err != nil {
			panic(fmt.Sprintf("invoice: templates/pdf/%s.json: %v", lang, err))
		}
		out[lang] = labels{text: text, comma: lang != "en"}
	}
	return out
}()

// labelsOf returns the labels of the staff language, Bosnian (the clinics' default) for any other.
func labelsOf(language string) labels {
	if l, ok := catalog[language]; ok {
		return l
	}
	return catalog[staffLanguages[0]]
}

// get returns the label key with its {name} placeholders filled from pairs of name and value.
func (l labels) get(key string, pairs ...string) string {
	s := l.text[key]
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, "{"+pairs[i]+"}", pairs[i+1])
	}
	return s
}

// method is the payment method's name.
func (l labels) method(m PaymentMethod) string { return l.get("method_" + string(m)) }

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
		s += ", " + l.get("discount") + " " + l.money(line.DiscountMinor)
	}
	if line.VATRateBP == nil {
		s += ", " + l.get("exempt")
	} else {
		s += fmt.Sprintf(", %s %s %%", l.get("vat"), l.money(int64(*line.VATRateBP)))
	}
	return s + " = " + l.money(line.TotalMinor) + " " + currency
}
