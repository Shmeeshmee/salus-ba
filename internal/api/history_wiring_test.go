package api_test

import (
	"os"
	"strings"
	"testing"
)

// The summary reads the allergies and medications through SAL-41's services: the stand-ins that failed closed
// until they were on main (history.go) are gone, and the history is built on the profile's services.
func TestHistory_ReadsTheProfileThroughSAL41sServices(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("history.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, stale := range []string{"noAllergies", "noMedications", "errProfileNotWired"} {
		if strings.Contains(src, stale) {
			t.Errorf("history.go still has the stand-in %s", stale)
		}
	}
	if !strings.Contains(src, "Allergies: p.allergies") || !strings.Contains(src, "Medications: p.medications") {
		t.Error("the history is not built on the profile's allergy and medication services")
	}
}
