//go:build e2e

package e2e_test

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The e-mail subjects of docs/booking/SPEC.md "Jobs".
const (
	bookedSubject    = "Your appointment is booked"
	cancelledSubject = "Your appointment is cancelled"
)

// TestBooking_PatientBooksAndPractitionerCancels: a patient registers, completes the profile, finds the listed
// clinic, gets consent_required, consents and books, and the confirmation arrives; the practitioner finds the
// appointment and reads the profile, then cancels; the patient sees it cancelled and gets the e-mail.
func TestBooking_PatientBooksAndPractitionerCancels(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)

	practitionerSees(t, clinic, appt, p)
	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/cancel", nil, nil)

	got := p.appointment(appt.ID)
	if got.Status != "cancelled" || got.reason() != "prov" {
		t.Errorf("the patient's Appointment/%s: status %q, reason %q; want cancelled by the practitioner (prov)",
			appt.ID, got.Status, got.reason())
	}
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
}

// TestBooking_RevokedConsentEndsAccessAndCancelsTheFutureAppointment: after a booking, the patient revokes the
// consent; the practitioner's Patient read is 404 at once, and the consent_revoked job cancels the future
// appointment (waited for as a condition) and e-mails the patient.
func TestBooking_RevokedConsentEndsAccessAndCancelsTheFutureAppointment(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	practitionerSees(t, clinic, appt, p)

	var consents struct {
		Consents []struct{ ID, OrganizationID, Status string }
	}
	p.must(http.StatusOK, http.MethodGet, "/me/consents", nil, &consents)
	if len(consents.Consents) != 1 || consents.Consents[0].OrganizationID != clinic.id || consents.Consents[0].Status != "active" {
		t.Fatalf("GET /me/consents = %+v, want the one active consent with the clinic", consents.Consents)
	}
	p.must(http.StatusOK, http.MethodPost, "/consents/"+consents.Consents[0].ID+"/revoke", nil, nil)

	if got := clinic.practitioner.do(http.MethodGet, "/fhir/R4/Patient/"+p.patientID, nil, nil); got != http.StatusNotFound {
		t.Errorf("the practitioner's Patient read right after the revocation = %d, want 404", got)
	}
	eventually(t, "the consent_revoked job cancels the future appointment", func() bool {
		return p.appointment(appt.ID).Status == "cancelled"
	})
	if got := p.appointment(appt.ID); got.reason() != "pat" {
		t.Errorf("the cancelled appointment's reason = %q, want pat (the patient revoked)", got.reason())
	}
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
}

// bookableClinic is a listed clinic made with salusctl, with one practitioner whose availability is set and
// whose slots are generated.
type bookableClinic struct {
	practitioner     *api
	admin            *api           // the clinic's admin, on the session their enrolment opened
	totp             *authenticator // the practitioner's, for the web app's sign-in
	id               string
	name             string
	practitionerID   string
	practitionerAddr string
	practitionerName string
	scheduleID       string
}

// newBookableClinic creates a clinic with salusctl; its admin lists it through the FHIR update and invites a
// practitioner, who sets 30-minute slots every day and waits for the slots_generate job's free slots. Both stay
// on the sessions their enrolment opened, so the journey logs in only its patient.
func newBookableClinic(t *testing.T) *bookableClinic {
	t.Helper()
	waitReady(t)
	adminAddr, practitionerAddr := email(t, "admin"), email(t, "practitioner")
	c := &bookableClinic{name: "E2E synthetic clinic " + suffix(t), practitionerAddr: practitionerAddr, practitionerName: "Bo Syntheticson"}
	c.id = field(t, salusctl(t, "create-clinic", "-name", c.name, "-admin-email", adminAddr), "organization")
	admin, _ := enrolFromInvite(t, adminAddr)
	c.admin = admin

	var org map[string]any
	admin.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization/"+c.id, nil, &org)
	meta, _ := org["meta"].(map[string]any)
	version, _ := meta["versionId"].(string)
	org["extension"] = []map[string]any{{
		"url": "https://salus.invalid/fhir/StructureDefinition/organization-listed", "valueBoolean": true,
	}}
	admin.mustWith(http.Header{"If-Match": {`W/"` + version + `"`}}, http.StatusOK, http.MethodPut, "/fhir/R4/Organization/"+c.id, org, nil)

	admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": practitionerAddr, "role": "practitioner", "givenName": "Bo", "familyName": "Syntheticson",
	}, nil)
	c.practitioner, c.totp = enrolFromInvite(t, practitionerAddr)
	c.practitionerID = c.practitioner.me().PractitionerID
	if c.practitionerID == "" {
		t.Fatal("GET /auth/me of the practitioner holds no practitionerId")
	}

	rules := make([]map[string]any, 0, 7)
	for day := 1; day <= 7; day++ {
		rules = append(rules, map[string]any{"day": day, "start": "00:00", "end": "23:30"})
	}
	var schedule struct{ ID string }
	c.practitioner.mustWith(http.Header{"If-Match": {`W/"0"`}}, http.StatusOK, http.MethodPut,
		"/practitioner-roles/"+publicRole(t, c.id)+"/availability", map[string]any{"slotMinutes": 30, "availability": rules}, &schedule)
	c.scheduleID = schedule.ID
	eventually(t, "the slots_generate job makes the schedule's free slots", func() bool {
		return len(freeSlots(t, newAPI(t), c.scheduleID)) > 0
	})
	return c
}

// publicRole finds the clinic's one practitioner role as anyone does: the clinic among the listed ones, then
// its public role search.
func publicRole(t *testing.T, clinic string) string {
	t.Helper()
	anonymous := newAPI(t)
	var listed bundle
	anonymous.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization?listed=true", nil, &listed)
	if !listed.has("Organization", clinic) {
		t.Fatalf("Organization?listed=true does not hold the clinic %s", clinic)
	}
	var roles bundle
	q := url.Values{"organization": {clinic}, "_include": {"PractitionerRole:practitioner"}}
	anonymous.must(http.StatusOK, http.MethodGet, "/fhir/R4/PractitionerRole?"+q.Encode(), nil, &roles)
	ids := roles.ids("PractitionerRole")
	if len(ids) != 1 {
		t.Fatalf("the clinic's public role search holds %d roles, want 1", len(ids))
	}
	return ids[0]
}

// freeSlots returns the schedule's free slots that start between one hour and three days from now.
func freeSlots(t *testing.T, c *api, schedule string) []string {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + schedule}, "status": {"free"},
		"start": {"ge" + now.Add(time.Hour).Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
	}
	var slots bundle
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	return slots.ids("Slot")
}

// patient is a signed-in patient with a profile.
type patient struct {
	*api
	addr      string
	patientID string
	name      string
}

// newPatientWithProfile registers a patient whose phone is in English, signs in and completes the profile with
// synthetic values. The app matches the phone's language and sends it (docs/i18n SPEC §3), so the e-mails these
// journeys read are the English ones.
func newPatientWithProfile(t *testing.T) *patient {
	t.Helper()
	return newPatientIn(t, "en")
}

// newPatientIn is newPatientWithProfile for a phone whose language the app matched to language.
func newPatientIn(t *testing.T, language string) *patient {
	t.Helper()
	p := &patient{api: newAPI(t), addr: email(t, "patient")}
	p.must(http.StatusCreated, http.MethodPost, "/auth/register", map[string]string{"email": p.addr, "password": password, "detectedLanguage": language}, nil)
	p.bearer = p.login(p.addr).Token
	if got := p.do(http.MethodGet, "/me/patient", nil, nil); got != http.StatusNotFound {
		t.Fatalf("GET /me/patient before the profile = %d, want 404", got)
	}
	var profile struct{ ID string }
	p.must(http.StatusCreated, http.MethodPut, "/me/patient", map[string]string{
		"givenName": "Ena", "familyName": "Syntheticova", "birthDate": "1990-05-17", "phone": "+38760000001",
	}, &profile)
	p.patientID, p.name = profile.ID, "Ena Syntheticova"
	return p
}

// bookFirstTime finds the clinic and a free slot, is refused with consent_required, consents and books. The
// confirmation must arrive in Mailpit.
func (p *patient) bookFirstTime(clinic *bookableClinic) appointmentResource {
	p.t.Helper()
	publicRole(p.t, clinic.id)
	slots := freeSlots(p.t, p.api, clinic.scheduleID)
	if len(slots) == 0 {
		p.t.Fatal("the patient finds no free slot")
	}
	body := map[string]string{"slotId": slots[0]}
	if got, code := p.refusal(http.MethodPost, "/appointments", body); got != http.StatusConflict || code != "consent_required" {
		p.t.Fatalf("POST /appointments without consent = %d %q, want 409 consent_required", got, code)
	}
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var booked struct{ ID, Status string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", body, &booked)
	if booked.Status != "booked" {
		p.t.Fatalf("POST /appointments: status %q, want booked", booked.Status)
	}
	checkAppointmentMail(p.t, p.addr, bookedSubject, clinic.name)
	return p.appointment(booked.ID)
}

// appointment is the patient's FHIR read of their appointment.
func (p *patient) appointment(id string) appointmentResource {
	p.t.Helper()
	var a appointmentResource
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Appointment/"+id, nil, &a)
	return a
}

// practitionerSees: the practitioner's search of their appointments around its start holds it with the
// patient's name, and their Patient read answers the profile.
func practitionerSees(t *testing.T, clinic *bookableClinic, appt appointmentResource, p *patient) {
	t.Helper()
	start, err := time.Parse(time.RFC3339, appt.Start)
	if err != nil {
		t.Fatalf("the appointment's start %q: %v", appt.Start, err)
	}
	q := url.Values{
		"practitioner": {"Practitioner/" + clinic.practitionerID},
		"date":         {"ge" + start.Add(-time.Hour).UTC().Format(time.RFC3339), "lt" + start.Add(time.Hour).UTC().Format(time.RFC3339)},
	}
	var found struct {
		Entry []struct{ Resource appointmentResource }
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/Appointment?"+q.Encode(), nil, &found)
	if len(found.Entry) != 1 || found.Entry[0].Resource.ID != appt.ID || found.Entry[0].Resource.Status != "booked" {
		t.Fatalf("the practitioner's search found %d appointments, want the booked %s", len(found.Entry), appt.ID)
	}
	ref, display := found.Entry[0].Resource.patient()
	if ref != "Patient/"+p.patientID || display != p.name {
		t.Errorf("the practitioner's appointment names patient %q with display set %t, want the patient with the name",
			ref, display != "")
	}

	var profile struct {
		ResourceType, ID, BirthDate string
	}
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/fhir/R4/Patient/"+p.patientID, nil, &profile)
	if profile.ResourceType != "Patient" || profile.ID != p.patientID || profile.BirthDate != "1990-05-17" {
		t.Errorf("the practitioner's Patient read is not the patient's profile (resource %s)", profile.ResourceType)
	}
}

// checkAppointmentMail waits for the e-mail with the subject and checks it names the clinic and links the app.
func checkAppointmentMail(t *testing.T, addr, subject, clinicName string) {
	t.Helper()
	text := mailText(t, addr, subject)
	if !strings.Contains(text, clinicName) || !strings.Contains(text, "/appointments") {
		t.Errorf("the e-mail %q does not name the clinic or link the app", subject)
	}
}

// appointmentResource is the part of a FHIR Appointment the journeys read.
type appointmentResource struct {
	ResourceType, ID, Status, Start string
	CancelationReason               *struct{ Coding []struct{ Code string } }
	Participant                     []struct {
		Actor  struct{ Reference, Display string }
		Status string // the patient's: tentative until confirmed, accepted after (ADR 0015)
	}
}

func (a appointmentResource) reason() string {
	if a.CancelationReason == nil || len(a.CancelationReason.Coding) == 0 {
		return ""
	}
	return a.CancelationReason.Coding[0].Code
}

// patient is the patient participant's reference and display.
func (a appointmentResource) patient() (string, string) {
	for _, p := range a.Participant {
		if strings.HasPrefix(p.Actor.Reference, "Patient/") {
			return p.Actor.Reference, p.Actor.Display
		}
	}
	return "", ""
}

// bundle is a FHIR searchset, read for its resources' types and ids.
type bundle struct {
	Entry []struct {
		Resource struct{ ResourceType, ID string }
	}
}

func (b bundle) ids(resourceType string) []string {
	var ids []string
	for _, e := range b.Entry {
		if e.Resource.ResourceType == resourceType {
			ids = append(ids, e.Resource.ID)
		}
	}
	return ids
}

func (b bundle) has(resourceType, id string) bool {
	for _, got := range b.ids(resourceType) {
		if got == id {
			return true
		}
	}
	return false
}

// eventually waits up to 30 s for cond, checking every 250 ms: jobs run in the worker, so the journeys wait for
// their effect as a condition, never for a fixed time.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	timeout := time.After(30 * time.Second)
	for !cond() {
		select {
		case <-timeout:
			t.Fatalf("%s: not within 30s", what)
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// suffix is a fresh hexadecimal suffix, so reruns against the same stack never collide.
func suffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
