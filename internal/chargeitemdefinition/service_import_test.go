package chargeitemdefinition_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
)

// Importing a price list from a file is the wave 3 contract (docs/clinic-cms, 3.0): refused until package O2
// builds it.
func TestImport_RefusesUntilPackageO2(t *testing.T) {
	t.Parallel()
	var s chargeitemdefinition.Service
	c := chargeitemdefinition.Caller{AccountID: "a", OrganizationID: "o", Admin: true}
	f := chargeitemdefinition.ImportFile{Name: "services.csv", Data: []byte("name;price\n")}
	if _, err := s.Import(t.Context(), c, f, false); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Import = %v, want errors.ErrUnsupported", err)
	}
}
