package api_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// TestReceptionDesk_TwoDesksBookingOneSlotAtOnceGiveOneVisit is F2's race on the real stack (docs/clinic-cms/WAVE6.md
// BG1; F2 tested it with a fake): two desks book two patients into one free slot at the same moment through the API.
// One visit is booked, the other desk gets 409 slot_unavailable, and the slot holds exactly one appointment.
func TestReceptionDesk_TwoDesksBookingOneSlotAtOnceGiveOneVisit(t *testing.T) {
	t.Parallel()
	r, _ := newReception(t)
	seedFree(t, r, 3)
	var free string
	if err := r.owner.QueryRow(t.Context(), `SELECT id::text FROM slot WHERE status = 'free' ORDER BY lower(period) LIMIT 1`).Scan(&free); err != nil {
		t.Fatal(err)
	}
	patients := make([]string, 2)
	for i, family := range []string{"Prvakova", "Drugakova"} {
		rec := r.receptionist.do(http.MethodPost, "/clinic/patients",
			`{"givenName":"Synthetic","familyName":"`+family+`","birthDate":"1970-01-0`+string(rune('1'+i))+`","phone":"+3876100077`+string(rune('1'+i))+`","consentGivenInPerson":true}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /clinic/patients = %d %s", rec.Code, rec.Body)
		}
		var p struct{ ID string }
		decode(t, rec, &p)
		patients[i] = p.ID
	}

	desks := []client{*r.receptionist, *r.receptionist} // one session, two desks: no shared client state
	codes, bodies := make([]int, 2), make([]string, 2)
	var wg sync.WaitGroup
	for i := range desks {
		wg.Go(func() {
			rec := desks[i].do(http.MethodPost, "/clinic/appointments", `{"patientId":"`+patients[i]+`","slotId":"`+free+`"}`)
			codes[i], bodies[i] = rec.Code, rec.Body.String()
		})
	}
	wg.Wait()

	booked, refused := 0, 0
	for i, code := range codes {
		switch {
		case code == http.StatusCreated:
			booked++
		case code == http.StatusConflict && strings.Contains(bodies[i], "slot_unavailable"):
			refused++
		default:
			t.Errorf("desk %d: %d %s, want 201 or 409 slot_unavailable", i, code, bodies[i])
		}
	}
	if booked != 1 || refused != 1 {
		t.Errorf("codes %v; want one visit booked and one desk refused", codes)
	}
	var visits int
	if err := r.owner.QueryRow(t.Context(), `SELECT count(*) FROM appointment WHERE slot_id = $1`, free).Scan(&visits); err != nil || visits != 1 {
		t.Errorf("appointments in the slot = %d, %v; want exactly one", visits, err)
	}
}
