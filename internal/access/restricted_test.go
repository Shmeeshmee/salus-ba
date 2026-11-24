package access_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

// A restricted record (docs/clinic-cms/WAVE5.md S7) is hidden from the clinic's staff, practitioners, the admin and
// the desk alike, whatever the consent and the appointments: every rule of the care record and of the chat refuses
// them. Only ClinicRecord, the admin's audited opening with a stated purpose, still reads it.
func TestRestricted_EveryStaffRuleRefusesARestrictedRecord(t *testing.T) {
	t.Parallel()
	guest := access.Viewer{Role: access.RolePractitioner, Clinic: clinicB, PractitionerRole: roleA2}
	for _, v := range []access.Viewer{viewers["practitioner"], viewers["admin"], receptionist, guest} {
		for cn, c := range consents {
			for an, vs := range visits {
				c.Restricted, c.Transcription, c.History = true, true, true
				f := access.Facts{Patient: patientA, Clinic: clinicA, Consent: c, Visits: vs}
				key := fmt.Sprintf("%+v/%s/%s", v, cn, an)
				if !access.Restricted(v, f) && v.Clinic == clinicA {
					t.Errorf("Restricted(%s) = false, want true", key)
				}
				checkRefused(t, key, v, f)
			}
		}
	}
}

func checkRefused(t *testing.T, key string, v access.Viewer, f access.Facts) {
	t.Helper()
	a := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(-48 * time.Hour)}
	r := access.Record{Clinic: clinicA, PractitionerRole: roleA1}
	levels := map[string]access.Level{
		"Profile": access.Profile(v, f), "Appointment": access.Appointment(v, f, a),
		"QuestionnaireResponse": access.QuestionnaireResponse(v, f, a, true),
	}
	for name, got := range levels {
		if got != access.None {
			t.Errorf("%s(%s) = %v, want None", name, key, got)
		}
	}
	rules := map[string]bool{
		"RecordsOutcome": access.RecordsOutcome(v, f, a), "CorrectsOutcome": access.CorrectsOutcome(v, f, a),
		"RequestsQuestionnaire": access.RequestsQuestionnaire(v, f, a), "ConsultationRecord": access.ConsultationRecord(v, f, a),
		"SignedDocument": access.SignedDocument(v, f, a, true), "TranscriptCopy": access.TranscriptCopy(v, f, a, true),
		"OpensChat": access.OpensChat(v, f, roleA1), "ReadsChat": access.ReadsChat(v, f, true), "WritesChat": access.WritesChat(v, f, true),
		"GuestChat": access.GuestChat(v, f, true), "ManagesChat": access.ManagesChat(v, f, roleA1),
		"OpensChatThread": access.OpensChatThread(v, f, true, true), "HistoryVisit": access.HistoryVisit(v, f, r),
		"ProfileEntry": access.ProfileEntry(v, f, true, r), "Attributed": access.Attributed(v, f, clinicA),
	}
	for name, got := range rules {
		if got {
			t.Errorf("%s(%s) = true, want false", name, key)
		}
	}
}

// The restriction takes nothing from the patient themselves (their own record, Art. 15) and nothing at another clinic.
func TestRestricted_ThePatientAndAnotherClinicAreUnchanged(t *testing.T) {
	t.Parallel()
	a := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(48 * time.Hour)}
	restricted := access.Facts{Patient: patientA, Clinic: clinicA, Consent: access.Consent{Active: true, Restricted: true}, Visits: []access.Visit{a}}
	if got := access.Profile(viewers["patient"], restricted); got != access.Full {
		t.Errorf("Profile(patient) = %v, want Full", got)
	}
	if access.Restricted(viewers["patient"], restricted) {
		t.Error("Restricted(patient) = true, want false")
	}
	open := restricted
	open.Consent.Restricted = false
	if got := access.Profile(viewers["practitioner"], open); got != access.Full {
		t.Errorf("Profile(practitioner, unrestricted) = %v, want Full", got)
	}
}

// The admin's ClinicRecord (the hand-over, opened with a stated purpose and audited by its service) reads a restricted
// record as it reads an unrestricted one; a practitioner still nothing.
func TestRestricted_ClinicRecordOpensForTheAdminOnly(t *testing.T) {
	t.Parallel()
	for cn, c := range consents {
		a := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(-48 * time.Hour)}
		f := access.Facts{Patient: patientA, Clinic: clinicA, Consent: c, Visits: []access.Visit{a}}
		want := access.ClinicRecord(viewers["admin"], f, a)
		f.Consent.Restricted = true
		if got := access.ClinicRecord(viewers["admin"], f, a); got != want || got == access.None {
			t.Errorf("ClinicRecord(admin, %s, restricted) = %v, want %v", cn, got, want)
		}
		if got := access.ClinicRecord(viewers["practitioner"], f, a); got != access.None {
			t.Errorf("ClinicRecord(practitioner, %s, restricted) = %v, want None", cn, got)
		}
	}
}
