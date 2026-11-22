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

const visitModeURL = "https://salus.invalid/fhir/StructureDefinition/visit-mode"

// moded is the part of a FHIR Slot or Appointment that says how the patient attends.
type moded struct {
	ID        string
	Extension []struct{ URL, ValueCode string }
}

func (m moded) mode() string {
	for _, e := range m.Extension {
		if e.URL == visitModeURL {
			return e.ValueCode
		}
	}
	return ""
}

// TestVisitModes_AnInPersonVisitHasTheAddressInItsEmailAndNoCall (docs/plans/clinic-cms.md, package A): the admin sets
// the clinic's address, the practitioner turns the working hours into visits at the clinic, and the free slots become
// in-person ones; a patient books one, the confirmation says that it is at the clinic and where, the appointment keeps
// the mode, and neither side gets a call for it.
func TestVisitModes_AnInPersonVisitHasTheAddressInItsEmailAndNoCall(t *testing.T) {
	clinic := newBookableClinic(t)

	var org map[string]any
	clinic.admin.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization/"+clinic.id, nil, &org)
	version, _ := org["meta"].(map[string]any)["versionId"].(string)
	org["address"] = []map[string]any{{
		"state": "BA-09", "line": []string{"Synthetic street 1"}, "postalCode": "71000",
		"extension": []map[string]any{{"url": "https://salus.invalid/fhir/StructureDefinition/organization-city", "valueCode": "Q1270190"}},
	}}
	clinic.admin.mustWith(http.Header{"If-Match": {`W/"` + version + `"`}}, http.StatusOK, http.MethodPut, "/fhir/R4/Organization/"+clinic.id, org, nil)

	role := publicRole(t, clinic.id)
	var hours struct{ VersionID int }
	clinic.practitioner.must(http.StatusOK, http.MethodGet, "/practitioner-roles/"+role+"/availability", nil, &hours)
	rules := make([]map[string]any, 0, 7)
	for day := 1; day <= 7; day++ {
		rules = append(rules, map[string]any{"day": day, "start": "00:00", "end": "23:30", "mode": "in-person"})
	}
	clinic.practitioner.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(hours.VersionID) + `"`}}, http.StatusOK, http.MethodPut,
		"/practitioner-roles/"+role+"/availability", map[string]any{"slotMinutes": 30, "availability": rules}, nil)

	p := newPatientWithProfile(t)
	var slot moded
	eventually(t, "the slots_generate job replaces the free video slots with in-person ones", func() bool {
		now := clock.System{}.Now().UTC()
		q := url.Values{
			"schedule": {"Schedule/" + clinic.scheduleID}, "status": {"free"},
			"start": {"ge" + now.Add(time.Hour).Format(time.RFC3339), "lt" + now.Add(72*time.Hour).Format(time.RFC3339)},
		}
		var found struct{ Entry []struct{ Resource moded } }
		p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &found)
		for _, e := range found.Entry {
			if e.Resource.mode() != "in-person" {
				return false
			}
		}
		if len(found.Entry) == 0 {
			return false
		}
		slot = found.Entry[0].Resource
		return true
	})

	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var booked struct{ ID, VisitMode string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot.ID}, &booked)
	if booked.VisitMode != "in-person" {
		t.Errorf("POST /appointments: visitMode %q, want in-person", booked.VisitMode)
	}
	var appt moded
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Appointment/"+booked.ID, nil, &appt)
	if appt.mode() != "in-person" {
		t.Errorf("Appointment/%s: visit mode %q, want in-person", booked.ID, appt.mode())
	}

	text := mailText(t, p.addr, bookedSubject)
	for _, want := range []string{clinic.name, "at the clinic, not by video", "Synthetic street 1, 71000 Centar"} {
		if !strings.Contains(text, want) {
			t.Errorf("the confirmation of an in-person visit lacks %q", want)
		}
	}
	if strings.Contains(text, "/consultation") || strings.Contains(text, "/call") {
		t.Error("the confirmation of an in-person visit holds a link to a call")
	}

	join := "/appointments/" + booked.ID + "/consultation/join"
	for who, c := range map[string]*api{"the patient": p.api, "the practitioner": clinic.practitioner} {
		if got := c.do(http.MethodPost, join, nil, nil); got != http.StatusNotFound {
			t.Errorf("%s joins an in-person visit: %d, want 404 (it has no call)", who, got)
		}
	}
}
