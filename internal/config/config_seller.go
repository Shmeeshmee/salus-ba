package config

import (
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

// LoadSeller reads the seller; a value not set is its [SELLER] placeholder, printed as such on the invoice until the
// owner sets it (Q3).
func LoadSeller() Seller {
	get := func(name, placeholder string) string {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
		return placeholder
	}
	return Seller{
		Name: get("SELLER_NAME", "[SELLER]"), TaxID: get("SELLER_TAX_ID", "[SELLER TAX ID]"),
		Address: get("SELLER_ADDRESS", "[SELLER ADDRESS]"), Account: get("SELLER_ACCOUNT", "[SELLER ACCOUNT]"),
	}
}
