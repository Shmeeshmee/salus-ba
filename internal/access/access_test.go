package access_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Synthetic ids only.
const (
	patientA = "d1000000-0000-4000-8000-0000000000d1"
	patientB = "d2000000-0000-4000-8000-0000000000d2"
	clinicA  = "a0000000-0000-4000-8000-00000000000a"
	clinicB  = "b0000000-0000-4000-8000-00000000000b"
	roleA1   = "e1000000-0000-4000-8000-0000000000e1" // the viewing practitioner's role in clinic A
	roleA2   = "e2000000-0000-4000-8000-0000000000e2" // another practitioner's role in clinic A
)

var revokedAt = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

// The dimensions of SPEC.md "Access rule": viewer × consent × appointment × clinic.
var (
	viewers = map[string]access.Viewer{
		"patient":       {Role: access.RolePatient, Patient: patientA},
		"other patient": {Role: access.RolePatient, Patient: patientB},
		"practitioner":  {Role: access.RolePractitioner, Clinic: clinicA, PractitionerRole: roleA1},
		"admin":         {Role: access.RoleAdmin, Clinic: clinicA},
	}
	consents = map[string]access.Consent{
		"active":  {Active: true},
		"revoked": {RevokedAt: revokedAt},
		"never":   {},
	}
	// The patient's appointment in the facts' clinic; "none" has none. Past and future are relative to the
	// revocation; "other practitioner" is a past appointment with another practitioner of the clinic.
	visits = map[string][]access.Visit{
		"none":               nil,
		"past":               {{PractitionerRole: roleA1, Start: revokedAt.Add(-48 * time.Hour)}},
		"future":             {{PractitionerRole: roleA1, Start: revokedAt.Add(48 * time.Hour)}},
		"other practitioner": {{PractitionerRole: roleA2, Start: revokedAt.Add(-48 * time.Hour)}},
	}
	clinics = map[string]string{"match": clinicA, "other clinic": clinicB}
)

// each calls fn with every combination, keyed "viewer/consent/appointment/clinic".
func each(fn func(key string, v access.Viewer, f access.Facts)) {
	for vn, v := range viewers {
		for cn, c := range consents {
			for an, a := range visits {
				for kn, k := range clinics {
					fn(fmt.Sprintf("%s/%s/%s/%s", vn, cn, an, kn), v, access.Facts{Patient: patientA, Clinic: k, Consent: c, Visits: a})
				}
			}
		}
	}
}

// selfFull is the patient on themselves: Full in every case.
func selfFull(want map[string]access.Level) map[string]access.Level {
	for cn := range consents {
		for an := range visits {
			for kn := range clinics {
				want[fmt.Sprintf("patient/%s/%s/%s", cn, an, kn)] = access.Full
			}
		}
	}
	return want
}

// check compares got with want for every combination, None where want has no entry, and fails on an entry
// of want that is no combination (a typo would otherwise pass as None).
func check(t *testing.T, name string, want map[string]access.Level, got func(key string, v access.Viewer, f access.Facts) (access.Level, bool)) {
	t.Helper()
	seen := map[string]bool{}
	each(func(key string, v access.Viewer, f access.Facts) {
		level, ok := got(key, v, f)
		if !ok {
			return
		}
		seen[key] = true
		if level != want[key] {
			t.Errorf("%s(%s) = %v, want %v", name, key, level, want[key])
		}
	})
	for key := range want {
		if !seen[key] {
			t.Errorf("expectation %q is no combination", key)
		}
	}
}

func TestProfile_OverEveryRoleConsentAppointmentAndClinic(t *testing.T) {
	want := selfFull(map[string]access.Level{
		// With active consent: the practitioner with an appointment with them, any time; the admin the name.
		"practitioner/active/past/match":        access.Full,
		"practitioner/active/future/match":      access.Full,
		"admin/active/none/match":               access.Name,
		"admin/active/past/match":               access.Name,
		"admin/active/future/match":             access.Name,
		"admin/active/other practitioner/match": access.Name,
		// After a revocation: the name to the practitioner of a past appointment and to the clinic's admin.
		"practitioner/revoked/past/match":        access.Name,
		"admin/revoked/past/match":               access.Name,
		"admin/revoked/other practitioner/match": access.Name,
	})
	check(t, "Profile", want, func(_ string, v access.Viewer, f access.Facts) (access.Level, bool) {
		return access.Profile(v, f), true
	})
}

func TestAppointment_OverEveryRoleConsentAppointmentAndClinic(t *testing.T) {
	want := selfFull(map[string]access.Level{
		// With active consent: the appointment's practitioner and the clinic's admin.
		"practitioner/active/past/match":        access.Full,
		"practitioner/active/future/match":      access.Full,
		"admin/active/past/match":               access.Full,
		"admin/active/future/match":             access.Full,
		"admin/active/other practitioner/match": access.Full,
		// After a revocation: name, times and status of a past appointment only.
		"practitioner/revoked/past/match":        access.Name,
		"admin/revoked/past/match":               access.Name,
		"admin/revoked/other practitioner/match": access.Name,
	})
	// "none" has no appointment to judge.
	for cn := range consents {
		for kn := range clinics {
			delete(want, fmt.Sprintf("patient/%s/none/%s", cn, kn))
		}
	}
	check(t, "Appointment", want, func(_ string, v access.Viewer, f access.Facts) (access.Level, bool) {
		if len(f.Visits) == 0 {
			return access.None, false
		}
		return access.Appointment(v, f, f.Visits[0]), true
	})
}

// RecordsOutcome: the appointment's practitioner, with an active consent, or after a revocation for a past
// appointment (the care record, Art. 9(2)(h)); never the admin, the patient or anyone of another clinic. That
// the start has passed is the service's check.
func TestRecordsOutcome_OverEveryRoleConsentAppointmentAndClinic(t *testing.T) {
	want := map[string]bool{
		"practitioner/active/past/match":   true,
		"practitioner/active/future/match": true,
		"practitioner/revoked/past/match":  true,
	}
	seen := map[string]bool{}
	each(func(key string, v access.Viewer, f access.Facts) {
		if len(f.Visits) == 0 {
			return
		}
		seen[key] = true
		if got := access.RecordsOutcome(v, f, f.Visits[0]); got != want[key] {
			t.Errorf("RecordsOutcome(%s) = %v, want %v", key, got, want[key])
		}
	})
	for key := range want {
		if !seen[key] {
			t.Errorf("expectation %q is no combination", key)
		}
	}
}

func TestAccess_RefusesIncompleteViewersAndFacts(t *testing.T) {
	past := access.Visit{Start: revokedAt.Add(-time.Hour)}
	tests := []struct {
		name string
		v    access.Viewer
		f    access.Facts
	}{
		{name: "no role", v: access.Viewer{Patient: patientA, Clinic: clinicA}, f: access.Facts{Patient: patientA, Clinic: clinicA, Consent: access.Consent{Active: true}}},
		{name: "a patient without an id on facts without one", v: access.Viewer{Role: access.RolePatient}, f: access.Facts{Clinic: clinicA}},
		{name: "staff without a clinic on facts without one", v: access.Viewer{Role: access.RoleAdmin}, f: access.Facts{Patient: patientA, Consent: access.Consent{Active: true}}},
		{name: "a practitioner without a role on an appointment without one", v: access.Viewer{Role: access.RolePractitioner, Clinic: clinicA},
			f: access.Facts{Patient: patientA, Clinic: clinicA, Consent: access.Consent{Active: true}, Visits: []access.Visit{past}}},
	}
	for _, tt := range tests {
		if got := access.Profile(tt.v, tt.f); got != access.None {
			t.Errorf("%s: Profile = %v, want None", tt.name, got)
		}
		if got := access.Appointment(tt.v, tt.f, past); got != access.None {
			t.Errorf("%s: Appointment = %v, want None", tt.name, got)
		}
	}
}

func TestAccess_PastMeansStartedBeforeTheLatestRevocation(t *testing.T) {
	v := viewers["practitioner"]
	tests := []struct {
		start time.Time
		name  string
		want  access.Level
	}{
		{name: "a second before", start: revokedAt.Add(-time.Second), want: access.Name},
		{name: "at the revocation", start: revokedAt, want: access.None},
		{name: "a second after", start: revokedAt.Add(time.Second), want: access.None},
	}
	for _, tt := range tests {
		visit := access.Visit{PractitionerRole: roleA1, Start: tt.start}
		f := access.Facts{Patient: patientA, Clinic: clinicA, Consent: consents["revoked"], Visits: []access.Visit{visit}}
		if got := access.Profile(v, f); got != tt.want {
			t.Errorf("%s: Profile = %v, want %v", tt.name, got, tt.want)
		}
		if got := access.Appointment(v, f, visit); got != tt.want {
			t.Errorf("%s: Appointment = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAccess_GrantingAgainRestoresEverything(t *testing.T) {
	future := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(48 * time.Hour)}
	f := access.Facts{Patient: patientA, Clinic: clinicA, Consent: access.Consent{Active: true, RevokedAt: revokedAt}, Visits: []access.Visit{future}}
	if got := access.Profile(viewers["practitioner"], f); got != access.Full {
		t.Errorf("Profile = %v, want Full", got)
	}
	if got := access.Appointment(viewers["admin"], f, future); got != access.Full {
		t.Errorf("Appointment = %v, want Full", got)
	}
}

func TestLevel_String(t *testing.T) {
	for level, want := range map[access.Level]string{access.None: "None", access.Name: "Name", access.Full: "Full", 7: "Level(7)"} {
		if got := level.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", int(level), got, want)
		}
	}
}

// QuestionnaireResponse (docs/questionnaire/SPEC.md §2 "Access rule"), over every combination and whether the
// response is submitted: the patient everything of their own; with an active consent the appointment's
// practitioner the answers once submitted and the name (that it exists, its status) before, the clinic's admin
// the name; after a revocation, for a past appointment, the practitioner the submitted answers (D9) and the
// admin the name of a submitted one, as row-level security shows only submitted ones then; nothing else.
func TestQuestionnaireResponse_OverEveryRoleConsentAppointmentClinicAndSubmission(t *testing.T) {
	want := map[string]access.Level{
		"practitioner/active/past/match/submitted":   access.Full,
		"practitioner/active/past/match/open":        access.Name,
		"practitioner/active/future/match/submitted": access.Full,
		"practitioner/active/future/match/open":      access.Name,
		// D9: the care record of a past appointment, to its practitioner only.
		"practitioner/revoked/past/match/submitted": access.Full,
	}
	for _, an := range []string{"past", "future", "other practitioner"} {
		for _, sub := range []string{"submitted", "open"} {
			want["admin/active/"+an+"/match/"+sub] = access.Name
		}
	}
	want["admin/revoked/past/match/submitted"] = access.Name
	want["admin/revoked/other practitioner/match/submitted"] = access.Name
	for cn := range consents {
		for _, an := range []string{"past", "future", "other practitioner"} {
			for kn := range clinics {
				for _, sub := range []string{"submitted", "open"} {
					want[fmt.Sprintf("patient/%s/%s/%s/%s", cn, an, kn, sub)] = access.Full
				}
			}
		}
	}

	seen := map[string]bool{}
	each(func(key string, v access.Viewer, f access.Facts) {
		if len(f.Visits) == 0 {
			return // no appointment to judge
		}
		for sub, submitted := range map[string]bool{"submitted": true, "open": false} {
			k := key + "/" + sub
			seen[k] = true
			got := access.QuestionnaireResponse(v, f, f.Visits[0], submitted)
			if got != want[k] {
				t.Errorf("QuestionnaireResponse(%s) = %v, want %v", k, got, want[k])
			}
			// Nobody but the patient and the appointment's practitioner is ever Full.
			own := v.Role == access.RolePatient && v.Patient == f.Patient ||
				v.Role == access.RolePractitioner && v.PractitionerRole == f.Visits[0].PractitionerRole
			if got == access.Full && !own {
				t.Errorf("QuestionnaireResponse(%s) = Full to someone who is neither the patient nor the practitioner", k)
			}
		}
	})
	for key := range want {
		if !seen[key] {
			t.Errorf("expectation %q is no combination", key)
		}
	}
}

// After a revocation the practitioner reads the submitted answers of an appointment that started before it,
// to the second, and nothing of a later one; granting again restores the rule of an active consent.
func TestQuestionnaireResponse_PastMeansStartedBeforeTheLatestRevocation(t *testing.T) {
	v := viewers["practitioner"]
	for _, tt := range []struct {
		start time.Time
		name  string
		want  access.Level
	}{
		{name: "a second before", start: revokedAt.Add(-time.Second), want: access.Full},
		{name: "at the revocation", start: revokedAt, want: access.None},
		{name: "a second after", start: revokedAt.Add(time.Second), want: access.None},
	} {
		visit := access.Visit{PractitionerRole: roleA1, Start: tt.start}
		f := access.Facts{Patient: patientA, Clinic: clinicA, Consent: consents["revoked"], Visits: []access.Visit{visit}}
		if got := access.QuestionnaireResponse(v, f, visit, true); got != tt.want {
			t.Errorf("%s: QuestionnaireResponse = %v, want %v", tt.name, got, tt.want)
		}
	}
	future := access.Visit{PractitionerRole: roleA1, Start: revokedAt.Add(48 * time.Hour)}
	again := access.Facts{Patient: patientA, Clinic: clinicA, Consent: access.Consent{Active: true, RevokedAt: revokedAt}, Visits: []access.Visit{future}}
	if got := access.QuestionnaireResponse(v, again, future, false); got != access.Name {
		t.Errorf("granted again, open: QuestionnaireResponse = %v, want Name", got)
	}
}

// RequestsQuestionnaire: the appointment's practitioner with an active consent, and nobody else; that the
// appointment is booked and ahead is the service's check.
func TestRequestsQuestionnaire_OverEveryRoleConsentAppointmentAndClinic(t *testing.T) {
	want := map[string]bool{
		"practitioner/active/past/match":   true,
		"practitioner/active/future/match": true,
	}
	seen := map[string]bool{}
	each(func(key string, v access.Viewer, f access.Facts) {
		if len(f.Visits) == 0 {
			return
		}
		seen[key] = true
		if got := access.RequestsQuestionnaire(v, f, f.Visits[0]); got != want[key] {
			t.Errorf("RequestsQuestionnaire(%s) = %v, want %v", key, got, want[key])
		}
	})
	for key := range want {
		if !seen[key] {
			t.Errorf("expectation %q is no combination", key)
		}
	}
}
