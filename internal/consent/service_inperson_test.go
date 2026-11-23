package consent_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
)

// A care consent given in person and recorded by staff is the wave 3 contract (docs/clinic-cms, 3.0): refused
// until package F2 builds it.
func TestRecordInPerson_RefusesUntilPackageF2(t *testing.T) {
	t.Parallel()
	var s consent.Service
	c := consent.Caller{AccountID: "a", OrganizationID: "o"}
	if _, err := s.RecordInPerson(t.Context(), c, "p"); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("RecordInPerson = %v, want errors.ErrUnsupported", err)
	}
}
