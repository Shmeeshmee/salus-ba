package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

var sellerVars = []string{"SELLER_NAME", "SELLER_TAX_ID", "SELLER_ADDRESS", "SELLER_ACCOUNT"}

// The seller of the membership invoices is the owner's company (Q3). In dev the [SELLER] placeholders stand in until it
// is set.
func TestLoadSeller_IsThePlaceholdersUntilSetInDev(t *testing.T) {
	t.Setenv("SALUS_ENV", config.EnvDev)
	for _, name := range sellerVars {
		t.Setenv(name, "")
	}
	want := config.Seller{Name: "[SELLER]", TaxID: "[SELLER TAX ID]", Address: "[SELLER ADDRESS]", Account: "[SELLER ACCOUNT]"}
	if got, err := config.LoadSeller(); err != nil || got != want {
		t.Errorf("LoadSeller() = %+v, %v; want %+v", got, err, want)
	}
	setSyntheticSeller(t)
	want = config.Seller{Name: "Synthetic seller d.o.o.", TaxID: "4200000000002", Address: "Synthetic street 2, 71000 Sarajevo", Account: "BA391290079401028494"}
	if got, err := config.LoadSeller(); err != nil || got != want {
		t.Errorf("LoadSeller() = %+v, %v; want %+v", got, err, want)
	}
}

// Outside dev a membership invoice may not print a placeholder: each of the four, unset or still in brackets, refuses
// the start-up, naming the variable and never the value.
func TestLoadSeller_RefusesAPlaceholderOutsideDev(t *testing.T) {
	t.Setenv("SALUS_ENV", "production")
	setSyntheticSeller(t)
	if got, err := config.LoadSeller(); err != nil || got.Name != "Synthetic seller d.o.o." {
		t.Fatalf("every value set: %+v, %v", got, err)
	}
	for _, name := range sellerVars {
		for _, value := range []string{"", "  ", "[SELLER]", "[" + name + "]", "[to be decided]"} {
			setSyntheticSeller(t)
			t.Setenv(name, value)
			_, err := config.LoadSeller()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("%s=%q outside dev: error = %v, want one naming the variable", name, value, err)
				continue
			}
			if strings.TrimSpace(value) != "" && strings.Contains(err.Error(), value) {
				t.Errorf("%s=%q: the error %q holds the value", name, value, err)
			}
		}
	}
}

func setSyntheticSeller(t *testing.T) {
	t.Helper()
	t.Setenv("SELLER_NAME", " Synthetic seller d.o.o. ")
	t.Setenv("SELLER_TAX_ID", "4200000000002")
	t.Setenv("SELLER_ADDRESS", "Synthetic street 2, 71000 Sarajevo")
	t.Setenv("SELLER_ACCOUNT", "BA391290079401028494")
}
