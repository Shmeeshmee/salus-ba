package config_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// The seller of the membership invoices is the owner's company (Q3): [SELLER] placeholders until it is set.
func TestLoadSeller_IsThePlaceholdersUntilSet(t *testing.T) {
	for _, name := range []string{"SELLER_NAME", "SELLER_TAX_ID", "SELLER_ADDRESS", "SELLER_ACCOUNT"} {
		t.Setenv(name, "")
	}
	want := config.Seller{Name: "[SELLER]", TaxID: "[SELLER TAX ID]", Address: "[SELLER ADDRESS]", Account: "[SELLER ACCOUNT]"}
	if got := config.LoadSeller(); got != want {
		t.Errorf("LoadSeller() = %+v, want %+v", got, want)
	}
	t.Setenv("SELLER_NAME", " Synthetic seller d.o.o. ")
	t.Setenv("SELLER_TAX_ID", "4200000000002")
	t.Setenv("SELLER_ADDRESS", "Synthetic street 2, 71000 Sarajevo")
	t.Setenv("SELLER_ACCOUNT", "BA391290079401028494")
	want = config.Seller{Name: "Synthetic seller d.o.o.", TaxID: "4200000000002", Address: "Synthetic street 2, 71000 Sarajevo", Account: "BA391290079401028494"}
	if got := config.LoadSeller(); got != want {
		t.Errorf("LoadSeller() = %+v, want %+v", got, want)
	}
}
