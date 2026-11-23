package invoice_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
)

// Treatment quotes are the wave 3 contract (docs/clinic-cms, 3.0): refused until package X1 builds them.
func TestQuotes_RefuseUntilPackageX1(t *testing.T) {
	t.Parallel()
	var s invoice.Service
	staff := invoice.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	patient := invoice.Caller{AccountID: "b", PatientID: "p"}
	ctx := t.Context()
	_, send := s.SendQuote(ctx, staff, invoice.QuoteDraft{PatientID: "p", Picks: []chargeitemdefinition.Pick{{DefinitionID: "d", Quantity: 1}}})
	_, list := s.Quotes(ctx, staff)
	_, read := s.ReadQuote(ctx, staff, "q")
	_, mine := s.MyQuotes(ctx, patient)
	_, pdf := s.QuotePDF(ctx, patient, "q")
	for name, err := range map[string]error{"SendQuote": send, "Quotes": list, "ReadQuote": read, "MyQuotes": mine, "QuotePDF": pdf} {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s = %v, want errors.ErrUnsupported", name, err)
		}
	}
}
