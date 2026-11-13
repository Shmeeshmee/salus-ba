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

// The subject of the waitlist offer's e-mail (internal/notification/templates/en/waitlist_offer.txt).
const offerSubject = "A time you were waiting for is free"

// TestWaitlist_AFreedSlotIsOfferedAndAcceptedThroughTheApp (SAL-35): the clinic's admin turns the waitlist on; a
// patient consents and joins it for the practitioner; another patient books a time more than a day ahead and cancels
// it; the worker offers the freed time to the waiting patient, who is e-mailed and finds the offer in the app, and
// accepting it books the appointment through the waitlist. The clinic's list then shows the entry booked.
func TestWaitlist_AFreedSlotIsOfferedAndAcceptedThroughTheApp(t *testing.T) {
	clinic := newBookableClinic(t)
	var settings map[string]any
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/appointment-settings", nil, &settings)
	settings["waitlist_enabled"] = true
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/appointment-settings", settings, nil)

	role := publicRole(t, clinic.id)
	waiting := newPatientWithProfile(t)
	waiting.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	today := clock.System{}.Now().In(loc)
	var entry struct{ ID, Status string }
	waiting.must(http.StatusCreated, http.MethodPost, "/waitlist", map[string]string{
		"organization_id": clinic.id, "practitioner_role_id": role,
		"earliest": today.Format(time.DateOnly), "latest": today.AddDate(0, 0, 4).Format(time.DateOnly),
	}, &entry)
	if entry.Status != "active" {
		t.Fatalf("POST /waitlist: status %q, want active", entry.Status)
	}

	holder := newPatientWithProfile(t)
	holder.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	slotID, start := freeSlotAfter(t, holder.api, clinic.scheduleID, 26*time.Hour)
	var booked struct{ ID string }
	holder.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slotID}, &booked)
	holder.must(http.StatusOK, http.MethodPost, "/appointments/"+booked.ID+"/cancel", nil, nil)

	var offer struct{ ID, SlotStart string }
	eventually(t, "the worker offers the freed time to the waiting patient", func() bool {
		var mine struct {
			Entries []struct {
				Offer *struct {
					ID        string `json:"id"`
					SlotStart string `json:"slot_start"`
				}
				ID string
			}
		}
		waiting.must(http.StatusOK, http.MethodGet, "/me/waitlist", nil, &mine)
		for _, e := range mine.Entries {
			if e.ID == entry.ID && e.Offer != nil {
				offer.ID, offer.SlotStart = e.Offer.ID, e.Offer.SlotStart
				return true
			}
		}
		return false
	})
	if got, err := time.Parse(time.RFC3339, offer.SlotStart); err != nil || !got.Equal(start) {
		t.Errorf("the offer's time %q, want the freed slot's %v", offer.SlotStart, start)
	}
	text := mailText(t, waiting.addr, offerSubject)
	if !strings.Contains(text, clinic.name) || !strings.Contains(text, "/waitlist") || strings.Contains(text, clinic.practitionerName) {
		t.Errorf("the offer's e-mail does not name the clinic and link the app, or names the practitioner")
	}
	if got, code := holder.refusal(http.MethodPost, "/appointments", map[string]string{"slotId": slotID}); got != http.StatusConflict || code != "slot_unavailable" {
		t.Errorf("a direct booking of the offered time = %d %q, want 409 slot_unavailable", got, code)
	}

	var appt struct{ ID, SlotID, BookedVia, Status string }
	waiting.must(http.StatusCreated, http.MethodPost, "/waitlist/offers/"+offer.ID+"/accept", nil, &appt)
	if appt.SlotID != slotID || appt.BookedVia != "waitlist" || appt.Status != "booked" {
		t.Errorf("the acceptance booked %+v, want the freed slot booked via the waitlist", appt)
	}
	var again struct{ ID string }
	waiting.must(http.StatusCreated, http.MethodPost, "/waitlist/offers/"+offer.ID+"/accept", nil, &again)
	if again.ID != appt.ID {
		t.Errorf("a second acceptance (another device) gave %s, want the booked appointment %s", again.ID, appt.ID)
	}
	var list struct {
		Entries []struct {
			Entry struct{ ID, Status string }
		}
	}
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/waitlist?status=booked", nil, &list)
	if len(list.Entries) != 1 || list.Entries[0].Entry.ID != entry.ID {
		t.Errorf("the clinic's booked waitlist %+v, want the one entry", list.Entries)
	}
}

// freeSlotAfter returns a free slot of the schedule that starts at least after from now (within the next three days), and
// its start.
func freeSlotAfter(t *testing.T, c *api, schedule string, after time.Duration) (string, time.Time) {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + schedule}, "status": {"free"},
		"start": {"ge" + now.Add(after).Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
	}
	var slots struct {
		Entry []struct {
			Resource struct{ ID, Start string }
		}
	}
	c.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	if len(slots.Entry) == 0 {
		t.Fatal("no free slot more than a day ahead")
	}
	start, err := time.Parse(time.RFC3339, slots.Entry[0].Resource.Start)
	if err != nil {
		t.Fatal(err)
	}
	return slots.Entry[0].Resource.ID, start
}
