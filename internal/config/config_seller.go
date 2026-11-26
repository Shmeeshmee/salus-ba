package config

import (
	"errors"
	"os"
	"strings"
)

// Seller is who issues Salus's membership invoices to the clinics (docs/plans/clinic-cms.md Q3, package I): the
// owner's company and the bank account the clinics pay into. Not secret, and not personal data.
type Seller struct {
	Name    string // SELLER_NAME
	TaxID   string // SELLER_TAX_ID (JIB/IDB)
	Address string // SELLER_ADDRESS, on one line
	Account string // SELLER_ACCOUNT, the IBAN of the transfer
}

// LoadSeller reads the seller, for the membership invoices' PDF: cmd/api and cmd/notifier call it. In dev a value not
// set is its [SELLER] placeholder, printed as such on the invoice. Outside SALUS_ENV=dev each of the four is required
// and may not be a placeholder (isPlaceholder), so no invoice of a real clinic prints one. Errors name the variable,
// never a value.
func LoadSeller() (Seller, error) {
	s := Seller{
		Name: strings.TrimSpace(os.Getenv("SELLER_NAME")), TaxID: strings.TrimSpace(os.Getenv("SELLER_TAX_ID")),
		Address: strings.TrimSpace(os.Getenv("SELLER_ADDRESS")), Account: strings.TrimSpace(os.Getenv("SELLER_ACCOUNT")),
	}
	dev := os.Getenv("SALUS_ENV") == EnvDev
	for _, v := range []struct {
		value       *string
		name, blank string
	}{
		{&s.Name, "SELLER_NAME", "[SELLER]"}, {&s.TaxID, "SELLER_TAX_ID", "[SELLER TAX ID]"},
		{&s.Address, "SELLER_ADDRESS", "[SELLER ADDRESS]"}, {&s.Account, "SELLER_ACCOUNT", "[SELLER ACCOUNT]"},
	} {
		switch {
		case !isPlaceholder(*v.value):
		case !dev:
			return Seller{}, errors.New("config: " + v.name + " is required outside SALUS_ENV=dev (it is unset or still a [placeholder])")
		case *v.value == "":
			*v.value = v.blank
		}
	}
	return s, nil
}

// isPlaceholder reports whether a setting is unset or still a placeholder in brackets, as configs/.env.example and
// deployments/production/.env.production.example write them ([OPERATOR EMAIL], [SELLER]).
func isPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")
}
