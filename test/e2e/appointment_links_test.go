//go:build e2e

package e2e_test

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

const rescheduledSubject = "Your appointment has a new time"

// TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp: a patient books two appointments more
// than a day ahead, each with a reminder due within the minute (the clinic's offset just under the time to its start),
// and takes each reminder's one-time link from the e-mail in Mailpit. The first link's page shows the clinic and the
// time however often it is fetched and changes nothing; its POST confirms, and the page then shows only that outcome.
// The patient moves the first appointment in the app ("open the app") and gets the e-mail with the new time. The
// second link cancels its appointment, on the second tap with the box ticked, and the cancellation e-mail arrives.
func TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	slots := slotsBetween(t, p.api, clinic.scheduleID, 26*time.Hour, 30*time.Hour)
	if len(slots) < 3 {
		t.Fatalf("%d free slots between 26 and 30 hours ahead, want 3", len(slots))
	}
	firstID, link := bookWithReminder(t, clinic, p, slots[0])
	secondID, cancel := bookWithReminder(t, clinic, p, slots[1])

	for range 3 { // a mail scanner's prefetch, a link preview, the patient
		if code, page := linkPage(t, link); code != http.StatusOK || !strings.Contains(page, clinic.name) || !strings.Contains(page, `value="confirm"`) {
			t.Fatalf("GET the link = %d, want the page with the clinic and the confirm button:\n%s", code, page)
		}
	}
	if got := participantStatus(p.appointment(firstID)); got != "tentative" {
		t.Errorf("after the GETs the patient participant is %q, want tentative (nothing changed)", got)
	}
	if code := postLink(t, link, url.Values{"action": {"confirm"}}, true); code != http.StatusSeeOther {
		t.Fatalf("POST confirm = %d, want 303", code)
	}
	if _, page := linkPage(t, link); !strings.Contains(page, "You confirmed this appointment.") || strings.Contains(page, "<form") {
		t.Errorf("the used link's page is not its outcome only:\n%s", page)
	}
	if got := participantStatus(p.appointment(firstID)); got != "accepted" {
		t.Errorf("after the confirmation the patient participant is %q, want accepted", got)
	}
	if code := postLink(t, link, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, true); code != http.StatusSeeOther ||
		p.appointment(firstID).Status != "booked" {
		t.Errorf("a second POST of the used link = %d, and changed the appointment", code)
	}

	// "To choose another time, open the Salus app": the app moves it to another slot of the role.
	var moved struct{ RescheduleCount int }
	p.must(http.StatusOK, http.MethodPost, "/appointments/"+firstID+"/reschedule", map[string]string{"slotId": slots[2]}, &moved)
	if moved.RescheduleCount != 1 {
		t.Errorf("the rescheduled appointment's count = %d, want 1", moved.RescheduleCount)
	}
	checkAppointmentMail(t, p.addr, rescheduledSubject, clinic.name)

	if code := postLink(t, cancel, url.Values{"action": {"cancel"}}, true); code != http.StatusBadRequest || p.appointment(secondID).Status != "booked" {
		t.Fatalf("POST cancel without the ticked box = %d, want 400 and nothing changed", code)
	}
	if code := postLink(t, cancel, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, false); code != http.StatusForbidden {
		t.Errorf("POST without Origin = %d, want 403", code)
	}
	if code := postLink(t, cancel, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, true); code != http.StatusSeeOther {
		t.Fatalf("POST cancel = %d, want 303", code)
	}
	if got := p.appointment(secondID); got.Status != "cancelled" || got.reason() != "pat" {
		t.Errorf("the appointment cancelled through the link: %q (%q), want cancelled by the patient", got.Status, got.reason())
	}
	if _, page := linkPage(t, cancel); !strings.Contains(page, "You cancelled this appointment.") {
		t.Errorf("the cancel link's page is not its outcome:\n%s", page)
	}
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
	var changes struct {
		Changes []struct{ Kind, Via, ToStatus string }
	}
	p.must(http.StatusOK, http.MethodGet, "/appointments/"+secondID+"/changes", nil, &changes)
	if n := len(changes.Changes); n != 1 || changes.Changes[0].Via != "link" {
		t.Errorf("the cancelled appointment's history %+v, want its cancellation via the link", changes.Changes)
	}
}

// linkInMail is the path of the one-time link in a reminder e-mail.
var linkInMail = regexp.MustCompile(`/a/[A-Za-z0-9_-]{43}`)

// bookWithReminder sets the clinic's one reminder just under the time to the slot's start (due within a minute), books
// the slot for p and returns the appointment's id and the path of the one-time link in its reminder e-mail, /a/<token>.
func bookWithReminder(t *testing.T, clinic *bookableClinic, p *patient, slot string) (string, string) {
	t.Helper()
	start := slotStart(t, p.api, clinic.scheduleID, slot)
	offset := int(start.Sub(clock.System{}.Now().Add(5*time.Second)) / time.Minute)
	clinic.admin.changeAppointmentSettings(func(settings map[string]any) { settings["reminder_offsets_minutes"] = []int{offset} })
	var booked struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot}, &booked)
	link := linkInMail.FindString(mailTextWithin(t, p.addr, reminderSubject, 2*time.Minute))
	if link == "" {
		t.Fatal("the reminder e-mail carries no one-time link")
	}
	return booked.ID, link
}

// slotStart is the start of the schedule's free slot id, from the slot search (Slot has no read).
func slotStart(t *testing.T, c *api, schedule, id string) time.Time {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + schedule}, "status": {"free"},
		"start": {"ge" + now.Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
	}
	var found struct {
		Entry []struct{ Resource struct{ ID, Start string } }
	}
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &found)
	for _, e := range found.Entry {
		if e.Resource.ID == id {
			at, err := time.Parse(time.RFC3339, e.Resource.Start)
			if err != nil {
				t.Fatal(err)
			}
			return at
		}
	}
	t.Fatalf("the free slot %s is not in the schedule's search", id)
	return time.Time{}
}

// linkPage fetches the link's page without a session, as a mail client opens it.
func linkPage(t *testing.T, path string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env(t, "E2E_BASE_URL")+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return linkDo(t, req)
}

// postLink submits the page's form, with the page's own Origin when origin is set.
func postLink(t *testing.T, path string, form url.Values, origin bool) int {
	t.Helper()
	base := env(t, "E2E_BASE_URL")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, base+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin {
		req.Header.Set("Origin", base)
	}
	code, _ := linkDo(t, req)
	return code
}

func linkDo(t *testing.T, req *http.Request) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s %s: Cache-Control %q, want no-store", req.Method, req.URL.Path[:3], resp.Header.Get("Cache-Control"))
	}
	return resp.StatusCode, string(b)
}

// slotsBetween returns the schedule's free slots starting between from and to from now.
func slotsBetween(t *testing.T, c *api, schedule string, from, to time.Duration) []string {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + schedule}, "status": {"free"},
		"start": {"ge" + now.Add(from).Format(time.RFC3339), "lt" + now.Add(to).Format(time.RFC3339)},
	}
	var slots bundle
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	return slots.ids("Slot")
}

// participantStatus is the status of the appointment's patient participant (tentative until confirmed, ADR 0015).
func participantStatus(a appointmentResource) string {
	for _, p := range a.Participant {
		if strings.HasPrefix(p.Actor.Reference, "Patient/") {
			return p.Status
		}
	}
	return ""
}
