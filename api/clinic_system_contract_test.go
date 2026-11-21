package api_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/subscription"
)

// The clinic system's enums in api/openapi.yaml are the Go constants (docs/plans/clinic-cms.md, step 0).
func TestClinicSystemContract_EnumsMatchTheGoConstants(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, want := range map[string][]string{
		"VisitMode":                 {string(schedule.ModeInPerson), string(schedule.ModeVideo)},
		"ReceiptKind":               {string(invoice.KindReceipt), string(invoice.KindInvoice)},
		"PaymentMethod":             {string(invoice.PaymentCash), string(invoice.PaymentCard), string(invoice.PaymentTransfer)},
		"FiscalStatus":              {string(invoice.FiscalPending), string(invoice.FiscalFiscalized), string(invoice.FiscalFailed)},
		"SubscriptionStatus":        {string(subscription.StatusTrial), string(subscription.StatusActive), string(subscription.StatusPastDue), string(subscription.StatusSuspended), string(subscription.StatusCancelled)},
		"SubscriptionInvoiceStatus": {string(subscription.InvoiceOpen), string(subscription.InvoicePaid), string(subscription.InvoiceVoid)},
	} {
		if got := s[name].Enum; !slices.Equal(got, want) {
			t.Errorf("%s: enum %v, want %v", name, got, want)
		}
	}
	if got := s["Appointment"].Properties["visitMode"].Ref; got != "#/components/schemas/VisitMode" || !slices.Contains(s["Appointment"].Required, "visitMode") {
		t.Errorf("Appointment.visitMode = %q, want the required VisitMode", got)
	}
	if got := s["AvailabilityRule"].Properties["mode"].Ref; got != "#/components/schemas/VisitMode" {
		t.Errorf("AvailabilityRule.mode = %q, want VisitMode", got)
	}
}

// Every wave-1 route of the clinic system is in the contract and names the package that builds it.
func TestClinicSystemContract_EveryWave1RouteNamesItsPackage(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"` // an operation, or a path's parameters
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for route, want := range map[string]string{
		"get /clinic/services":               "cms-b",
		"post /clinic/services":              "cms-b",
		"put /clinic/services/{id}":          "cms-b",
		"get /organizations/{id}/services":   "cms-b",
		"get /clinic/receipts":               "cms-c",
		"post /clinic/receipts":              "cms-c",
		"get /clinic/receipts/{id}":          "cms-c",
		"post /clinic/receipts/{id}/reverse": "cms-c",
		"get /clinic/receipts/{id}/pdf":      "cms-c",
		"get /clinic/reports/receipts":       "cms-c",
		"get /me/receipts":                   "cms-c",
		"get /me/receipts/{id}/pdf":          "cms-c",
		"get /clinic/subscription":           "cms-d",
		"get /clinic/subscription/invoices":  "cms-d",
		"get /clinic/billing-details":        "cms-d",
		"put /clinic/billing-details":        "cms-d",
	} {
		method, path, _ := strings.Cut(route, " ")
		op, _ := doc.Paths[path][method].(map[string]any)
		if got, _ := op["x-salus-package"].(string); got != want {
			t.Errorf("%s: x-salus-package %q, want %q", route, got, want)
		}
	}
}
