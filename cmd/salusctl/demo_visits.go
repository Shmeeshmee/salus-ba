package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// demoVisit is one visit of the month: who, when, how, and the state the seed leaves it in.
type demoVisit struct {
	start, end         time.Time
	slot               demoSlot // the slot ahead it is booked on
	id, etag, service  string
	mode, status       string
	patient, clinician int // indexes of d.people and d.practitioners
	moved              bool
}

// The share of a practitioner's slots of the four weeks ahead the seed books (per cent): moved back four weeks, as the
// month behind; in the two weeks ahead; and later.
const (
	demoPastShare = 25
	demoNearShare = 30
	demoFarShare  = 4
)

// demoShift is how far back a visit of the month behind is moved: four weeks, so it lands on the same weekday and
// hour, inside the practitioner's hours, at the same place.
const demoShift = -28

// pick is a fixed spread of i and salt over 0..99 (a SplitMix64 step): the same seed gives the same month every time.
func pick(i, salt int) int {
	x := uint64(i)*0x9E3779B97F4A7C15 + uint64(salt)*0xBF58476D1CE4E5B9 //nolint:gosec // small non-negative ints
	x ^= x >> 31
	x *= 0x94D049BB133111EB
	x ^= x >> 29
	return int(x % 100) //nolint:gosec // below 100
}

// visits books the month: for each practitioner a share of their slots ahead, some moved back as the month behind,
// each left in its state (fulfilled, no-show, cancelled, arrived at the desk, booked).
func (d *demoSeed) visits(ctx context.Context) error {
	now := clock.System{}.Now()
	var month, spare []demoVisit // spare: today's in-person slots the share left out, for the arrival
	for c := range d.practitioners {
		free, err := demoFreeSlots(ctx, d.db, d.practitioners[c].scheduleID)
		if err != nil {
			return err
		}
		for i, s := range free {
			if v, ok := d.plan(c, i, s, now); ok {
				month = append(month, v)
			} else if s.mode == "in-person" && sameDay(s.start.In(d.loc), now.In(d.loc)) {
				spare = append(spare, demoVisit{slot: s, clinician: c, status: "booked", mode: s.mode, start: s.start, end: s.end,
					service: d.serviceFor(c, s.mode)})
			}
		}
	}
	month = arrival(month, spare, now, d.loc)
	for i := range month {
		month[i].patient = (i * 7) % len(d.people)
		if err := d.visit(ctx, &month[i]); err != nil {
			return err
		}
	}
	d.month = month
	return nil
}

// arrival makes one of today's in-person visits the patient in the waiting room: the next one still to come, or, late
// in the day, the last one that began; when the share booked none today, one of today's spare slots ahead.
func arrival(month, spare []demoVisit, now time.Time, loc *time.Location) []demoVisit {
	best := -1
	for i, v := range month {
		if v.mode != "in-person" || v.status == "cancelled" || !sameDay(v.start.In(loc), now.In(loc)) {
			continue
		}
		if best < 0 || closer(v.start, month[best].start, now) {
			best = i
		}
	}
	if best < 0 && len(spare) > 0 {
		month, best = append(month, spare[0]), len(month)
	}
	if best >= 0 {
		month[best].status = "arrived"
	}
	return month
}

// closer is whether a is a better arrival than b: one still to come before one that began, and nearer now.
func closer(a, b, now time.Time) bool {
	if a.After(now) != b.After(now) {
		return a.After(now)
	}
	if a.After(now) {
		return a.Before(b)
	}
	return a.After(b)
}

// plan decides what the seed makes of the practitioner c's i-th free slot ahead, s.
func (d *demoSeed) plan(c, i int, s demoSlot, now time.Time) (demoVisit, bool) {
	p, outcome := pick(i, c), pick(i, c+7)
	v := demoVisit{slot: s, clinician: c, mode: s.mode, start: s.start, end: s.end, service: d.serviceFor(c, s.mode)}
	switch {
	case p < demoPastShare:
		v.moved = true
		v.start, v.end = s.start.In(d.loc).AddDate(0, 0, demoShift), s.end.In(d.loc).AddDate(0, 0, demoShift)
		switch {
		case outcome < 72:
			v.status = "fulfilled"
		case outcome < 84:
			v.status = "noshow"
		default:
			v.status = "cancelled"
		}
	case p < demoPastShare+demoNearShare && s.start.Before(now.AddDate(0, 0, 14)), p < demoPastShare+demoFarShare:
		v.status = "booked"
		if outcome >= 90 {
			v.status = "cancelled"
		}
	default:
		return demoVisit{}, false
	}
	return v, true
}

// serviceFor is the id of the service a visit with the practitioner c in mode is for.
func (d *demoSeed) serviceFor(c int, mode string) string {
	if mode == "video" {
		return d.services[d.practitioners[c].practitioner.video]
	}
	return d.services[d.practitioners[c].practitioner.inPerson]
}

// sameDay is whether a and b are the same local date.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// visit books v on the slot s, as its patient in the app or as the desk, and takes it to its state: cancelled
// before it is moved back (a cancellation is of a visit ahead), then moved, then marked by its practitioner.
func (d *demoSeed) visit(ctx context.Context, v *demoVisit) error {
	s := v.slot
	who := d.people[v.patient]
	var booked struct{ ID string }
	var res demoResponse
	var err error
	if who.app != nil && !v.moved {
		res, err = who.app.must(ctx, http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": s.id, "serviceId": v.service}, &booked)
	} else {
		res, err = d.desk.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/appointments",
			map[string]string{"patientId": who.id, "slotId": s.id, "serviceId": v.service}, &booked)
	}
	if err != nil {
		return err
	}
	v.id, v.etag = booked.ID, res.etag
	if v.status == "cancelled" {
		if err := d.cancel(ctx, v, who.app != nil && !v.moved); err != nil {
			return err
		}
	}
	if v.moved {
		if err := demoMoveBack(ctx, d.db, v.id, v.start, v.end); err != nil {
			return err
		}
	}
	switch v.status {
	case "arrived":
		_, err = d.desk.user.must(ctx, http.StatusOK, http.MethodPost, "/clinic/appointments/"+v.id+"/arrive", nil, nil)
	case "fulfilled", "noshow":
		_, err = d.practitioners[v.clinician].user.must(ctx, http.StatusOK, http.MethodPost, "/appointments/"+v.id+"/status",
			map[string]string{"status": v.status}, nil)
	}
	return err
}

// cancel cancels the visit ahead: the patient who booked it in the app cancels it there, the desk the others.
func (d *demoSeed) cancel(ctx context.Context, v *demoVisit, byPatient bool) error {
	if byPatient {
		_, err := d.people[v.patient].app.must(ctx, http.StatusOK, http.MethodPost, "/appointments/"+v.id+"/cancel", nil, nil)
		return err
	}
	_, err := d.desk.user.mustIf(ctx, http.StatusOK, http.MethodPost, "/clinic/appointments/"+v.id+"/cancel", v.etag, nil, nil)
	return err
}

// receipts issues the receipts of the month at the desk: for most visits that took place, the visit's service, and
// for some cardiology visits an ECG beside it; the last one is reversed by the admin, as a mistaken receipt is.
func (d *demoSeed) receipts(ctx context.Context) error {
	methods := []string{"cash", "card", "cash", "transfer"}
	var last string
	for i, v := range d.month {
		if v.status != "fulfilled" || pick(i, 3) >= 85 {
			continue
		}
		lines := []map[string]any{{"serviceId": v.service, "quantity": 1}}
		if d.practitioners[v.clinician].practitioner.inPerson == "KP-01" && v.mode == "in-person" && pick(i, 5) < 40 {
			lines = append(lines, map[string]any{"serviceId": d.services["EKG-01"], "quantity": 1})
		}
		var r struct{ ID string }
		if _, err := d.desk.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/receipts", map[string]any{
			"kind": "receipt", "appointmentId": v.id, "paymentMethod": methods[i%len(methods)], "lines": lines,
		}, &r); err != nil {
			return err
		}
		last = r.ID
	}
	if last == "" {
		return fmt.Errorf("demo-seed: no receipt was issued")
	}
	_, err := d.admin.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/receipts/"+last+"/reverse",
		map[string]string{"paymentMethod": "cash"}, nil)
	return err
}
