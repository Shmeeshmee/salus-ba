package access_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
)

// A treatment plan (docs/clinic-cms/WAVE6.md U6) is worked with by the clinic's admin for any patient of the clinic (a
// consent with it, active or revoked, or a visit there) and by a practitioner only for a patient they have or had a
// visit with, of any status, whatever the consent; nobody while the record is restricted, and never a receptionist,
// the patient or another clinic's staff.
func TestCarePlan_OverViewerConsentAndVisit(t *testing.T) {
	t.Parallel()
	reception := access.Viewer{Role: access.RoleReception, Clinic: clinicA}
	for _, tc := range []struct {
		name    string
		clinic  string
		visits  []access.Visit
		viewer  access.Viewer
		consent access.Consent
		want    bool
	}{
		{"the admin, an active consent", clinicA, nil, viewers["admin"], consents["active"], true},
		{"the admin, a revoked consent", clinicA, nil, viewers["admin"], consents["revoked"], true},
		{"the admin, a visit and no consent", clinicA, visits["other practitioner"], viewers["admin"], consents["never"], true},
		{"the admin, no patient of the clinic", clinicA, nil, viewers["admin"], consents["never"], false},
		{"the admin of another clinic", clinicB, visits["past"], viewers["admin"], consents["active"], false},
		{"the admin, a restricted record", clinicA, visits["past"], viewers["admin"], access.Consent{Active: true, Restricted: true}, false},
		{"the practitioner of a past visit", clinicA, visits["past"], viewers["practitioner"], consents["active"], true},
		{"the practitioner of a future visit", clinicA, visits["future"], viewers["practitioner"], consents["active"], true},
		{"the practitioner of a visit, after a revocation", clinicA, visits["future"], viewers["practitioner"], consents["revoked"], true},
		{"a practitioner without a visit", clinicA, visits["other practitioner"], viewers["practitioner"], consents["active"], false},
		{"the practitioner, a restricted record", clinicA, visits["past"], viewers["practitioner"], access.Consent{Active: true, Restricted: true}, false},
		{"the practitioner of another clinic", clinicB, visits["past"], viewers["practitioner"], consents["active"], false},
		{"a receptionist", clinicA, visits["past"], reception, consents["active"], false},
		{"the patient", clinicA, visits["past"], viewers["patient"], consents["active"], false},
	} {
		f := access.Facts{Patient: patientA, Clinic: tc.clinic, Consent: tc.consent, Visits: tc.visits}
		if got := access.CarePlan(tc.viewer, f); got != tc.want {
			t.Errorf("%s: CarePlan = %v, want %v", tc.name, got, tc.want)
		}
	}
}
