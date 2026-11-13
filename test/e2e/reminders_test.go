//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// reminderSubject is the reminder e-mail's subject in English.
const reminderSubject = "Reminder of your appointment"

// SAL-33: the clinic's admin sets a reminder just under the time to the first free slot, a patient books it, and the
// notifier's reminder (its dispatch at the due time, then its send) reaches Mailpit with the clinic, the time and a
// one-time link to the api; the patient's settings say how (e-mail only here).
func TestReminders_TheReminderEmailArrivesAtItsTimeWithAOneTimeLink(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	p.must(http.StatusOK, http.MethodPut, "/me/notification-settings", map[string]any{
		"channels": map[string]bool{"push": false, "email": true, "sms": false},
	}, nil)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)

	// The first free slot from an hour ahead, with its start (Slot has a search, no read).
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + clinic.scheduleID}, "status": {"free"},
		"start": {"ge" + now.Add(time.Hour).Format(time.RFC3339), "lt" + now.Add(3*time.Hour).Format(time.RFC3339)},
	}
	var found struct {
		Entry []struct{ Resource struct{ ID, Start string } }
	}
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &found)
	if len(found.Entry) == 0 {
		t.Fatal("no free slot")
	}
	var slot string
	var start time.Time
	for _, e := range found.Entry {
		at, err := time.Parse(time.RFC3339, e.Resource.Start)
		if err != nil {
			t.Fatal(err)
		}
		if slot == "" || at.Before(start) {
			slot, start = e.Resource.ID, at
		}
	}
	slots := []string{slot}
	// Due between 5 and 65 seconds from now: the offset in whole minutes just under the time to the start.
	offset := int(start.Sub(clock.System{}.Now().Add(5*time.Second)) / time.Minute)
	var settings map[string]any
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/appointment-settings", nil, &settings)
	settings["reminder_offsets_minutes"] = []int{offset}
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/appointment-settings", settings, nil)

	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slots[0]}, nil)
	text := mailTextWithin(t, p.addr, reminderSubject, 2*time.Minute)
	if !strings.Contains(text, clinic.name) || !strings.Contains(text, "(Europe/Sarajevo)") || !strings.Contains(text, "/a/") ||
		!strings.Contains(text, "/appointments") {
		t.Errorf("the reminder does not name the clinic, the time and zone, the one-time link and the app:\n%s", text)
	}
	if strings.Contains(text, clinic.practitionerName) {
		t.Error("the reminder names the practitioner")
	}
}
