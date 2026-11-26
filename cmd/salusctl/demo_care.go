package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The demo's care beyond the visits: a lab result, quotes, a treatment plan, a questionnaire with branching, a chat,
// data requests. The clinical wording is placeholder content, labelled "(primjer)" ("example"): nothing is a real
// finding, and nothing presents itself as a diagnosis.

// pastVisit books a visit of the patient n with the practitioner c that took place, for what follows a visit (a lab
// result, a chat, a quote): the practitioner's first free in-person slot a week ahead, moved back four weeks as the
// month's are, and marked fulfilled. The same patients get them on every seed, which the script names.
func (d *demoSeed) pastVisit(ctx context.Context, n, c int) (demoVisit, error) {
	free, err := demoFreeSlots(ctx, d.db, d.practitioners[c].scheduleID)
	if err != nil {
		return demoVisit{}, err
	}
	s, ok := firstFree(free, "in-person", clock.System{}.Now().AddDate(0, 0, 7), nil)
	if !ok {
		return demoVisit{}, errors.New("demo-seed: no free slot for a past visit")
	}
	v := demoVisit{
		slot: s, clinician: c, patient: n, mode: s.mode, status: "fulfilled", moved: true, service: d.serviceFor(c, s.mode),
		start: s.start.In(d.loc).AddDate(0, 0, demoShift), end: s.end.In(d.loc).AddDate(0, 0, demoShift),
	}
	return v, d.visit(ctx, &v)
}

// labResult records a blood count on a visit of the second app patient with the first practitioner, as the
// practitioner, and releases it to the patient; the patient then asks about it in the chat (chat).
func (d *demoSeed) labResult(ctx context.Context) error {
	v, err := d.pastVisit(ctx, 1, 0)
	if err != nil {
		return err
	}
	d.followUp = v
	by := d.practitioners[v.clinician].user
	var report struct{ ID string }
	res, err := by.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/appointments/"+v.id+"/diagnostic-reports", map[string]any{
		"title": "Krvna slika (primjer)", "loinc": "58410-2", "effectiveDate": v.start.In(d.loc).Format(time.DateOnly),
		"results": []map[string]any{
			{"test": "Hemoglobin", "loinc": "718-7", "value": 138, "unit": "g/L", "referenceRange": "120-160", "interpretation": "N"},
			{"test": "Leukociti", "loinc": "6690-2", "value": 11.2, "unit": "10*9/L", "referenceRange": "4.0-10.0", "interpretation": "H"},
			{"test": "Trombociti", "loinc": "777-3", "value": 250, "unit": "10*9/L", "referenceRange": "150-400", "interpretation": "N"},
		},
	}, &report)
	if err != nil {
		return err
	}
	_, err = by.mustIf(ctx, http.StatusOK, http.MethodPost, "/clinic/diagnostic-reports/"+report.ID+"/release", res.etag, nil, nil)
	return err
}

// quotes sends two quotes after a visit, as its practitioner: one to an app patient, who sees it in the app, and one
// to a desk patient (a quote names its visit, and the desk sends none).
func (d *demoSeed) quotes(ctx context.Context) error {
	for _, q := range []struct {
		note               string
		codes              []string
		patient, clinician int
	}{
		{patient: 0, clinician: 1, note: "Procjena troškova kardiološke obrade (primjer).", codes: []string{"KP-01", "EKG-01", "LAB-01"}},
		{patient: demoAppPatients, clinician: 0, note: "Procjena troškova kontrola (primjer).", codes: []string{"KN-01", "LAB-01"}},
	} {
		v, err := d.pastVisit(ctx, q.patient, q.clinician)
		if err != nil {
			return err
		}
		var lines []map[string]any
		for _, code := range q.codes {
			lines = append(lines, map[string]any{"serviceId": d.services[code], "quantity": 1})
		}
		if _, err := d.practitioners[v.clinician].user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/quotes",
			map[string]any{"patientId": d.people[q.patient].id, "appointmentId": v.id, "note": q.note, "lines": lines}, nil); err != nil {
			return err
		}
	}
	return nil
}

// carePlan writes a three-step treatment plan for an app patient as their practitioner and books it on the
// practitioner's free slots: in person this week, by video in two weeks, in person in three.
func (d *demoSeed) carePlan(ctx context.Context) error {
	const c = 0
	st := d.practitioners[c]
	steps := []struct {
		code, mode string
		days       int
	}{{"KN-01", "in-person", 2}, {"KN-01", "video", 14}, {"OP-01", "in-person", 21}}
	free, err := demoFreeSlots(ctx, d.db, st.scheduleID)
	if err != nil {
		return err
	}
	now := clock.System{}.Now()
	var draft []map[string]any
	var slots []string
	for _, s := range steps {
		draft = append(draft, map[string]any{"serviceId": d.services[s.code], "dayOffset": s.days, "visitMode": s.mode})
		slot, ok := firstFree(free, s.mode, now.AddDate(0, 0, s.days), slots)
		if !ok {
			return errors.New("demo-seed: no free slot for the treatment plan")
		}
		slots = append(slots, slot.id)
	}
	var plan struct{ ID string }
	res, err := st.user.must(ctx, http.StatusCreated, http.MethodPost, "/clinic/care-plans",
		map[string]any{"patientId": d.people[1].id, "title": "Plan kontrola (primjer)", "steps": draft}, &plan)
	if err != nil {
		return err
	}
	_, err = st.user.mustIf(ctx, http.StatusOK, http.MethodPost, "/clinic/care-plans/"+plan.ID+"/book", res.etag,
		map[string]any{"startsOn": now.In(d.loc).Format(time.DateOnly), "slotIds": slots}, nil)
	return err
}

// firstFree is the first free slot in mode from after on that is not taken.
func firstFree(free []demoSlot, mode string, after time.Time, taken []string) (demoSlot, bool) {
next:
	for _, s := range free {
		if s.mode != mode || s.start.Before(after) {
			continue
		}
		for _, t := range taken {
			if t == s.id {
				continue next
			}
		}
		return s, true
	}
	return demoSlot{}, false
}

// uuid is a random UUID (version 4), for a chat message's client id.
func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// chat opens the conversation of the patient with the lab result with its practitioner and exchanges a few messages,
// as the patient app and the web app send them.
func (d *demoSeed) chat(ctx context.Context) error {
	v := d.followUp
	patient, practitioner := d.people[v.patient].app, d.practitioners[v.clinician]
	var conv struct{ ID string }
	if _, err := patient.must(ctx, http.StatusCreated, http.MethodPost, "/conversations",
		map[string]string{"organization_id": d.org, "practitioner_role_id": practitioner.roleID}, &conv); err != nil {
		return err
	}
	for _, m := range []struct {
		from *demoUser
		body string
	}{
		{patient, "Dobar dan, nalaz krvi je stigao u aplikaciju. Da li trebam doći na kontrolu? (primjer poruke)"},
		{practitioner.user, "Dobar dan, nalaz je pregledan. Zakažite kontrolu za dvije sedmice. (primjer poruke)"},
		{patient, "Hvala, zakazat ću preko aplikacije. (primjer poruke)"},
	} {
		if _, err := m.from.must(ctx, http.StatusCreated, http.MethodPost, "/conversations/"+conv.ID+"/messages",
			map[string]string{"client_message_id": uuid(), "body": m.body}, nil); err != nil {
			return err
		}
	}
	return nil
}

// dataRequests files two requests of app patients to the clinic: an access request, which Salus answers itself with
// the patient's export, and a rectification left open, which the admin finds with its deadline.
func (d *demoSeed) dataRequests(ctx context.Context) error {
	for _, r := range []struct {
		kind    string
		patient int
	}{{"access", 4}, {"rectification", 3}} {
		if _, err := d.people[r.patient].app.must(ctx, http.StatusCreated, http.MethodPost, "/me/data-requests",
			map[string]string{"kind": r.kind, "target": "clinic", "organizationId": d.org}, nil); err != nil {
			return err
		}
	}
	return nil
}
