//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// TestDesk_AReceptionistMovesAVisit (docs/clinic-cms WAVE4, M1): the receptionist books a walk-in in a free in-person
// time and moves the visit to a later one with the version read: the same appointment at the new time, not counted
// as the patient's reschedule; the old time is free again and the new one is not; a second move from the old version
// is refused with version_conflict.
func TestDesk_AReceptionistMovesAVisit(t *testing.T) {
	clinic := newBookableClinic(t)
	atTheClinic(t, clinic)
	deskAddr := email(t, "reception")
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/invites", map[string]string{"email": deskAddr, "role": "reception"}, nil)
	desk, _ := enrolFromInvite(t, deskAddr)

	free := deskFreeTimes(t, desk, clinic)
	if len(free) < 2 {
		t.Fatalf("the receptionist's Slot search holds %d free in-person slots, want two", len(free))
	}
	var walkIn struct{ ID string }
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/patients", map[string]any{
		"givenName": "Mirza", "familyName": "Premjestic", "birthDate": "1971-07-07", "consentGivenInPerson": true,
	}, &walkIn)
	type visit struct {
		ID, SlotID, Status string
		VersionID          int
		RescheduleCount    int
	}
	var booked visit
	desk.must(http.StatusCreated, http.MethodPost, "/clinic/appointments", map[string]string{"patientId": walkIn.ID, "slotId": free[0]}, &booked)

	match := http.Header{"If-Match": {`W/"` + strconv.Itoa(booked.VersionID) + `"`}}
	var moved visit
	desk.mustWith(match, http.StatusOK, http.MethodPost, "/clinic/appointments/"+booked.ID+"/reschedule", map[string]string{"slotId": free[1]}, &moved)
	if moved.ID != booked.ID || moved.SlotID != free[1] || moved.RescheduleCount != 0 || moved.VersionID != booked.VersionID+1 || moved.Status != "booked" {
		t.Errorf("the move = %+v, want the same visit at the new slot, uncounted, at the next version", moved)
	}
	now := deskFreeTimes(t, desk, clinic)
	if !slices.Contains(now, free[0]) || slices.Contains(now, free[1]) {
		t.Errorf("free after the move %v, want %s free again and %s taken", now, free[0], free[1])
	}
	if code, body := desk.refusalWith(match, http.MethodPost, "/clinic/appointments/"+booked.ID+"/reschedule"); code != http.StatusBadRequest {
		t.Errorf("a move without a body = %d %s, want 400", code, body)
	}
	if code := desk.do(http.MethodPost, "/clinic/appointments/"+booked.ID+"/reschedule", map[string]string{"slotId": free[0]}, nil); code != http.StatusBadRequest {
		t.Errorf("a move without If-Match = %d, want 400 if_match_required", code)
	}
	code, raw := desk.send(match, http.MethodPost, "/clinic/appointments/"+booked.ID+"/reschedule", map[string]string{"slotId": free[0]})
	var stale struct{ Error string }
	_ = json.Unmarshal(raw, &stale)
	if code != http.StatusConflict || stale.Error != "version_conflict" {
		t.Errorf("a move from the old version = %d %s, want 409 version_conflict", code, raw)
	}
}

// deskFreeTimes are the free in-person slots of the clinic's schedule in the next 48 hours, by start, as the
// receptionist's own Schedule and Slot searches give them.
func deskFreeTimes(t *testing.T, desk *api, clinic *bookableClinic) []string {
	t.Helper()
	now := clock.System{}.Now().UTC()
	q := url.Values{
		"schedule": {"Schedule/" + clinic.scheduleID}, "status": {"free"},
		"start": {"ge" + now.Add(10*time.Minute).Format(time.RFC3339), "lt" + now.Add(48*time.Hour).Format(time.RFC3339)},
	}
	var slots struct {
		Entry []struct {
			Resource struct {
				ID        string
				Extension []struct{ URL, ValueCode string }
			}
		}
	}
	desk.must(http.StatusOK, http.MethodGet, "/fhir/R4/Slot?"+q.Encode(), nil, &slots)
	var out []string
	for _, e := range slots.Entry {
		for _, x := range e.Resource.Extension {
			if x.URL == visitModeURL && x.ValueCode == "in-person" {
				out = append(out, e.Resource.ID)
			}
		}
	}
	return out
}
