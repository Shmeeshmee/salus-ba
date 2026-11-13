//go:build e2e

package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	patientdomain "github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const rescheduledSubject = "Your appointment has a new time"

// TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp: a patient books two appointments more
// than a day ahead. A one-time link of the first (issued as the notifier's reminder job issues it) shows the clinic and
// the time however often it is fetched and changes nothing; its POST confirms, and the page then shows only that
// outcome. The patient moves the first appointment in the app ("open the app"), gets the e-mail with the new time, and
// the old link no longer works. The second appointment's link cancels it, on the second tap with the box ticked, and
// the cancellation e-mail arrives.
//
// Until SAL-33 merges, the links are issued here through appointment.Service.IssueLink; once its reminder e-mail
// carries the link, this journey starts from that e-mail in Mailpit.
func TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	slots := slotsBetween(t, p.api, clinic.scheduleID, 26*time.Hour, 30*time.Hour)
	if len(slots) < 3 {
		t.Fatalf("%d free slots between 26 and 30 hours ahead, want 3", len(slots))
	}
	var first, second struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slots[0]}, &first)
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slots[1]}, &second)

	link := issueLink(t, clinic.id, first.ID)
	for range 3 { // a mail scanner's prefetch, a link preview, the patient
		if code, page := linkPage(t, link); code != http.StatusOK || !strings.Contains(page, clinic.name) || !strings.Contains(page, `value="confirm"`) {
			t.Fatalf("GET the link = %d, want the page with the clinic and the confirm button:\n%s", code, page)
		}
	}
	if got := participantStatus(p.appointment(first.ID)); got != "tentative" {
		t.Errorf("after the GETs the patient participant is %q, want tentative (nothing changed)", got)
	}
	if code := postLink(t, link, url.Values{"action": {"confirm"}}, true); code != http.StatusSeeOther {
		t.Fatalf("POST confirm = %d, want 303", code)
	}
	if _, page := linkPage(t, link); !strings.Contains(page, "You confirmed this appointment.") || strings.Contains(page, "<form") {
		t.Errorf("the used link's page is not its outcome only:\n%s", page)
	}
	if got := participantStatus(p.appointment(first.ID)); got != "accepted" {
		t.Errorf("after the confirmation the patient participant is %q, want accepted", got)
	}
	if code := postLink(t, link, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, true); code != http.StatusSeeOther ||
		p.appointment(first.ID).Status != "booked" {
		t.Errorf("a second POST of the used link = %d, and changed the appointment", code)
	}

	// "To choose another time, open the Salus app": the app moves it to another slot of the role.
	var moved struct {
		Start           string
		RescheduleCount int
	}
	p.must(http.StatusOK, http.MethodPost, "/appointments/"+first.ID+"/reschedule", map[string]string{"slotId": slots[2]}, &moved)
	if moved.RescheduleCount != 1 {
		t.Errorf("the rescheduled appointment's count = %d, want 1", moved.RescheduleCount)
	}
	checkAppointmentMail(t, p.addr, rescheduledSubject, clinic.name)
	if _, page := linkPage(t, issueLink(t, clinic.id, first.ID)); !strings.Contains(page, `value="cancel"`) {
		t.Errorf("a link issued after the reschedule does not offer the cancel:\n%s", page)
	}

	cancel := issueLink(t, clinic.id, second.ID)
	if code := postLink(t, cancel, url.Values{"action": {"cancel"}}, true); code != http.StatusBadRequest || p.appointment(second.ID).Status != "booked" {
		t.Fatalf("POST cancel without the ticked box = %d, want 400 and nothing changed", code)
	}
	if code := postLink(t, cancel, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, false); code != http.StatusForbidden {
		t.Errorf("POST without Origin = %d, want 403", code)
	}
	if code := postLink(t, cancel, url.Values{"action": {"cancel"}, "confirm_cancel": {"yes"}}, true); code != http.StatusSeeOther {
		t.Fatalf("POST cancel = %d, want 303", code)
	}
	if got := p.appointment(second.ID); got.Status != "cancelled" || got.reason() != "pat" {
		t.Errorf("the appointment cancelled through the link: %q (%q), want cancelled by the patient", got.Status, got.reason())
	}
	if _, page := linkPage(t, cancel); !strings.Contains(page, "You cancelled this appointment.") {
		t.Errorf("the cancel link's page is not its outcome:\n%s", page)
	}
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
	var changes struct {
		Changes []struct{ Kind, Via, ToStatus string }
	}
	p.must(http.StatusOK, http.MethodGet, "/appointments/"+second.ID+"/changes", nil, &changes)
	if n := len(changes.Changes); n != 1 || changes.Changes[0].Via != "link" {
		t.Errorf("the cancelled appointment's history %+v, want its cancellation via the link", changes.Changes)
	}
}

// issueLink issues a one-time e-mail link of the appointment, in English, as the reminder job does (connected as the
// owner); it returns its path, /a/<token>.
func issueLink(t *testing.T, org, appointmentID string) string {
	t.Helper()
	pool, err := postgres.Open(t.Context(), env(t, "DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx := postgres.NewTransactor(pool)
	svc := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{}),
		Tx: tx, Clock: clock.System{}, Patients: ownerPatients{pool: pool}, Tokens: appointment.NewPostgresTokenRepository(pool),
		LinkURL: "https://api.salus.invalid",
	})
	u, err := svc.IssueLink(t.Context(), "appointment_reminder", appointment.Link{
		AppointmentID: appointmentID, OrganizationID: org, Language: "en", Channel: appointment.ChannelEmail,
	})
	if err != nil {
		t.Fatalf("IssueLink() = %v", err)
	}
	return strings.TrimPrefix(u, "https://api.salus.invalid")
}

// ownerPatients gives IssueLink a patient's account, read as the owner.
type ownerPatients struct{ pool postgres.Querier }

func (o ownerPatients) Lookup(ctx context.Context, id string) (patientdomain.Patient, error) {
	p := patientdomain.Patient{ID: id}
	err := o.pool.QueryRow(ctx, "SELECT account_id::text FROM patient WHERE id = $1", id).Scan(&p.AccountID)
	return p, err
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
