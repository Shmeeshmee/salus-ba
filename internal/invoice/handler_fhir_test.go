package invoice_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/invoice"
)

func TestFHIR_WritesAReceiptAsAnInvoiceIssuedOrCancelled(t *testing.T) {
	t.Parallel()
	issued := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		want string
		in   invoice.Invoice
	}{
		"a patient's receipt": {
			in: invoice.Invoice{ID: "i1", OrganizationID: "o1", PatientID: "p1", Number: 7, IssuedAt: issued, TotalMinor: 5005, Currency: "BAM"},
			want: `{"resourceType":"Invoice","id":"i1","status":"issued","date":"2026-11-20T08:00:00Z","meta":{"lastUpdated":"2026-11-20T08:00:00Z","versionId":"1"},` +
				`"subject":{"reference":"Patient/p1"},"issuer":{"reference":"Organization/o1"},"totalGross":{"value":50.05,"currency":"BAM"},"identifier":[{"value":"7"}]}`,
		},
		"a reversed one, to no patient": {
			in: invoice.Invoice{ID: "i2", OrganizationID: "o1", ReversedByID: "i3", Number: 8, IssuedAt: issued, TotalMinor: -90, Currency: "EUR"},
			want: `{"resourceType":"Invoice","id":"i2","status":"cancelled","date":"2026-11-20T08:00:00Z","meta":{"lastUpdated":"2026-11-20T08:00:00Z","versionId":"1"},` +
				`"issuer":{"reference":"Organization/o1"},"totalGross":{"value":-0.90,"currency":"EUR"},"identifier":[{"value":"8"}]}`,
		},
	} {
		got, err := json.Marshal(invoice.FHIR(tc.in))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: %s, %v\nwant %s", name, got, err, tc.want)
		}
	}
}
