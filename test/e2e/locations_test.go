//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

const slotLocationURL = "https://salus.invalid/fhir/StructureDefinition/slot-location"

// sited is the part of a FHIR Slot that says where an in-person slot is.
type sited struct {
	ID        string
	Start     string
	Extension []struct {
		URL            string
		ValueReference struct{ Reference string }
	}
}

func (s sited) location() string {
	for _, e := range s.Extension {
		if e.URL == slotLocationURL {
			return strings.TrimPrefix(e.ValueReference.Reference, "Location/")
		}
	}
	return ""
}

// sites is a clinic with a second location (the annex) that has a room and a device, and a second practitioner.
type sites struct {
	*bookableClinic
	annex, room, device, service string
	second                       *api // the second practitioner
	secondRole                   string
}

// TestLocations_HoursRoomsDevicesAndTheVisitsAddress (docs/clinic-cms/WAVE6.md U1 to U4): the admin adds an annex with
// a room and a device, and a service that needs the device; the practitioner works at the annex in the room; a second
// practitioner is refused the same room at the same time and works at the annex without one. A patient books in person
// at the annex and the confirmation gives the annex's address; two bookings that need the device at the same time: the
// second is refused.
func TestLocations_HoursRoomsDevicesAndTheVisitsAddress(t *testing.T) {
	s := newSites(t)
	first := sitedSlots(t, s.scheduleID, s.annex)
	secondSchedule := s.hours(t, s.second, s.secondRole, "", http.StatusOK)
	second := sitedSlots(t, secondSchedule, s.annex)

	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": s.id}, nil)
	var booked struct{ ID, LocationID, RoomID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": first[0].ID, "serviceId": s.service}, &booked)
	if booked.LocationID != s.annex || booked.RoomID != s.room {
		t.Errorf("POST /appointments: location %q room %q, want the annex and its room", booked.LocationID, booked.RoomID)
	}
	text := mailText(t, p.addr, bookedSubject)
	if !strings.Contains(text, "Address: Synthetic annex 2, 88000") || strings.Contains(text, "Synthetic street 1") {
		t.Errorf("the confirmation of a visit at the annex does not give the annex's address:\n%s", text)
	}

	// The same device at the same time: the second booking is refused, whoever's hours it is in.
	var same string
	for _, sl := range second {
		if sl.Start == first[0].Start {
			same = sl.ID
		}
	}
	if same == "" {
		t.Fatalf("the second practitioner has no slot at %s", first[0].Start)
	}
	other := newPatientWithProfile(t)
	other.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": s.id}, nil)
	if code, body := other.refusal(http.MethodPost, "/appointments", map[string]string{"slotId": same, "serviceId": s.service}); code != http.StatusConflict ||
		!strings.Contains(body, "resource_busy") {
		t.Errorf("a second booking needing the device at the same time = %d %s, want 409 resource_busy", code, body)
	}
	other.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": same}, nil) // without the device it is free
}

// newSites makes the clinic of the journey: the annex with Room 1 and an ultrasound, a service that needs it, the first
// practitioner's hours at the annex in Room 1, and a second practitioner refused Room 1 at the same time.
func newSites(t *testing.T) *sites {
	t.Helper()
	s := &sites{bookableClinic: newBookableClinic(t)}
	var annex struct{ ID string }
	s.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations", map[string]any{
		"name": "Synthetic annex", "addressLine": "Synthetic annex 2", "postalCode": "88000", "active": true,
	}, &annex)
	s.annex = annex.ID
	var room, device, service struct{ ID string }
	s.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations/"+s.annex+"/rooms", map[string]any{"name": "Room 1", "active": true}, &room)
	s.admin.must(http.StatusCreated, http.MethodPost, "/clinic/locations/"+s.annex+"/devices", map[string]any{"name": "Ultrasound", "active": true}, &device)
	s.room, s.device = room.ID, device.ID
	s.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Synthetic ultrasound", "priceMinor": 8000, "modes": []string{"in-person"}, "active": true, "translations": []any{}, "deviceId": s.device,
	}, &service)
	s.service = service.ID
	if got := s.hours(t, s.practitioner, publicRole(t, s.id), s.room, http.StatusOK); got != s.scheduleID {
		t.Fatalf("the first practitioner's schedule %q, want %q", got, s.scheduleID)
	}

	addr := email(t, "practitioner2")
	s.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{
		"email": addr, "role": "practitioner", "givenName": "Di", "familyName": "Syntheticova",
	}, nil)
	s.second, _ = enrolFromInvite(t, addr)
	s.secondRole = roleOf(t, s.id, s.second.me().PractitionerID)
	s.hours(t, s.second, s.secondRole, s.room, http.StatusConflict) // Room 1 is the first practitioner's at that time
	return s
}

// hours sets the role's in-person hours at the annex, every day from 00:00 to 23:30, in room ("" for none), as its
// practitioner, and returns the schedule's id; want is the answer expected (a refusal returns "").
func (s *sites) hours(t *testing.T, as *api, role, room string, want int) string {
	t.Helper()
	var current struct{ VersionID int } // 0 for a role without a schedule yet
	if code := as.do(http.MethodGet, "/practitioner-roles/"+role+"/availability", nil, &current); code != http.StatusOK {
		current.VersionID = 0
	}
	rules := make([]map[string]any, 0, 7)
	for day := 1; day <= 7; day++ {
		rule := map[string]any{"day": day, "start": "00:00", "end": "23:30", "mode": "in-person", "locationId": s.annex}
		if room != "" {
			rule["roomId"] = room
		}
		rules = append(rules, rule)
	}
	header := http.Header{"If-Match": {`W/"` + strconv.Itoa(current.VersionID) + `"`}}
	if want != http.StatusOK {
		code, body := as.send(header, http.MethodPut, "/practitioner-roles/"+role+"/availability", map[string]any{"slotMinutes": 30, "availability": rules})
		if code != want || !strings.Contains(string(body), "resource_busy") {
			t.Errorf("PUT the hours in a room another practitioner holds = %d %s, want %d resource_busy", code, body, want)
		}
		return ""
	}
	var saved struct{ ID string }
	as.mustWith(header, http.StatusOK, http.MethodPut, "/practitioner-roles/"+role+"/availability", map[string]any{"slotMinutes": 30, "availability": rules}, &saved)
	return saved.ID
}

// sitedSlots waits until the schedule's free slots between one hour and three days ahead are all at the location, as
// anyone reads them, and returns them.
func sitedSlots(t *testing.T, schedule, location string) []sited {
	t.Helper()
	anonymous := newAPI(t)
	var out []sited
	eventually(t, "the slots_generate job makes the schedule's free slots at the location", func() bool {
		now := clock.System{}.Now().UTC()
		q := url.Values{
			"schedule": {"Schedule/" + schedule}, "status": {"free"},
			"start": {"ge" + now.Add(time.Hour).Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
		}
		var found struct{ Entry []struct{ Resource sited } }
		anonymous.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &found)
		out = out[:0]
		for _, e := range found.Entry {
			if e.Resource.location() != location {
				return false
			}
			out = append(out, e.Resource)
		}
		return len(out) > 0
	})
	return out
}

// roleOf finds the clinic's practitioner role of the practitioner, as anyone does.
func roleOf(t *testing.T, clinic, practitioner string) string {
	t.Helper()
	var roles struct {
		Entry []struct {
			Resource struct {
				ResourceType, ID string
				Practitioner     struct{ Reference string }
			}
		}
	}
	q := url.Values{"organization": {clinic}}
	newAPI(t).must(http.StatusOK, http.MethodGet, "/fhir/R4/PractitionerRole?"+q.Encode(), nil, &roles)
	for _, e := range roles.Entry {
		if e.Resource.ResourceType == "PractitionerRole" && e.Resource.Practitioner.Reference == "Practitioner/"+practitioner {
			return e.Resource.ID
		}
	}
	t.Fatalf("the clinic has no public role of Practitioner/%s", practitioner)
	return ""
}
