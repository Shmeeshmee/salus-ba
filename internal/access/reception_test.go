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

// Desk is the desk's view of a patient (name, birth date, phone, the appointments with their service and visit
// type): the clinic's receptionist and admin with an active care consent, or, after a revocation, while the patient
// has an appointment that started before it (as Profile's Name); nobody else, and never another clinic's staff.
func TestDesk_GivesTheClinicsFrontDeskThePatientWithAConsentOrAPastVisit(t *testing.T) {
	t.Parallel()
	viewersForDesk := map[string]access.Viewer{"receptionist": receptionist, "admin": viewers["admin"]}
	for vn, v := range viewersForDesk {
		eachForReception(func(key string, f access.Facts) {
			want := f.Clinic == clinicA && (f.Consent.Active || (f.Consent.RevokedAt.Equal(revokedAt) && hasPast(f)))
			if got := access.Desk(v, f); got != want {
				t.Errorf("Desk(%s, %s) = %v, want %v", vn, key, got, want)
			}
		})
	}
	for _, vn := range []string{"patient", "other patient", "practitioner"} {
		eachForReception(func(key string, f access.Facts) {
			if access.Desk(viewers[vn], f) {
				t.Errorf("Desk(%s, %s) = true, want false: the desk's view is the front desk's", vn, key)
			}
		})
	}
	if access.Desk(access.Viewer{Role: access.RoleReception}, access.Facts{Patient: patientA, Consent: access.Consent{Active: true}}) {
		t.Error("Desk() without a clinic = true, want false")
	}
}

func hasPast(f access.Facts) bool {
	for _, a := range f.Visits {
		if a.Start.Before(f.Consent.RevokedAt) {
			return true
		}
	}
	return false
}
