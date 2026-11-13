package access_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Q6 (docs/plans/appointments.md): a no-show and an attendance are corrected by whoever records the outcome (the
// appointment's practitioner) and by the clinic's admin, with an active consent or, after a revocation, for an
// appointment that started before it (the care record); nobody else. The 30 days are the service's check.
func TestCorrectsOutcome_OverEveryRoleConsentAppointmentAndClinic(t *testing.T) {
	want := map[string]bool{
		"practitioner/active/past/match":         true,
		"practitioner/active/future/match":       true,
		"practitioner/revoked/past/match":        true,
		"admin/active/past/match":                true,
		"admin/active/future/match":              true,
		"admin/active/other practitioner/match":  true,
		"admin/revoked/past/match":               true,
		"admin/revoked/other practitioner/match": true,
	}
	seen := map[string]bool{}
	each(func(key string, v access.Viewer, f access.Facts) {
		if len(f.Visits) == 0 {
			return
		}
		seen[key] = true
		if got := access.CorrectsOutcome(v, f, f.Visits[0]); got != want[key] {
			t.Errorf("CorrectsOutcome(%s) = %v, want %v", key, got, want[key])
		}
	})
	for key := range want {
		if !seen[key] {
			t.Errorf("expectation %q is no combination", key)
		}
	}
}
