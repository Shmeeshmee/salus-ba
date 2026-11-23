package access_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

var receptionist = access.Viewer{Role: access.RoleReception, Clinic: clinicA}

// eachForReception calls fn with the receptionist of clinic A and every consent × appointment × clinic.
func eachForReception(fn func(key string, f access.Facts)) {
	for cn, c := range consents {
		for an, a := range visits {
			for kn, k := range clinics {
				fn(fmt.Sprintf("%s/%s/%s", cn, an, kn), access.Facts{Patient: patientA, Clinic: k, Consent: c, Visits: a})
			}
		}
	}
}

// A receptionist sees no clinical content (docs/clinic-cms W2): every rule of the care record refuses them, whatever
// the consent, the appointments and the clinic, also a receptionist who is (wrongly) given a practitioner role.
func TestReception_EveryClinicalRuleRefusesAReceptionist(t *testing.T) {
	t.Parallel()
	withRole := receptionist
	withRole.PractitionerRole = roleA1
	for _, v := range []access.Viewer{receptionist, withRole} {
		eachForReception(func(key string, f access.Facts) {
			a := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(-48 * time.Hour)}
			levels := map[string]access.Level{
				"Profile": access.Profile(v, f), "Appointment": access.Appointment(v, f, a),
				"QuestionnaireResponse": access.QuestionnaireResponse(v, f, a, true),
			}
			for name, got := range levels {
				if got != access.None {
					t.Errorf("%s(%+v, %s) = %v, want None", name, v, key, got)
				}
			}
			rules := map[string]bool{
				"RecordsOutcome": access.RecordsOutcome(v, f, a), "CorrectsOutcome": access.CorrectsOutcome(v, f, a),
				"RequestsQuestionnaire": access.RequestsQuestionnaire(v, f, a), "ConsultationRecord": access.ConsultationRecord(v, f, a),
				"SignedDocument": access.SignedDocument(v, f, a, true), "TranscriptCopy": access.TranscriptCopy(v, f, a, true),
				"OpensChat": access.OpensChat(v, f, roleA1), "ReadsChat": access.ReadsChat(v, f, true), "WritesChat": access.WritesChat(v, f, true),
				"GuestChat": access.GuestChat(v, f, true), "ManagesChat": access.ManagesChat(v, f, roleA1),
				"OpensChatThread": access.OpensChatThread(v, f, true, true), "GrantsChatGuest": access.GrantsChatGuest(v, f),
			}
			for name, got := range rules {
				if got {
					t.Errorf("%s(%+v, %s) = true, want false", name, v, key)
				}
			}
		})
	}
}
