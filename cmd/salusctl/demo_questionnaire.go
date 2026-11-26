package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// demoQuestionnaire is the intake form with branching: a section shown on a yes, and a question inside it shown only
// for an answer above a number. Its wording is placeholder content.
func demoQuestionnaire() map[string]any {
	when := func(question, operator, answer string, value any) []map[string]any {
		return []map[string]any{{"question": question, "operator": operator, answer: value}}
	}
	return map[string]any{
		"title": "Upitnik prije pregleda (primjer)",
		"item": []map[string]any{
			{"linkId": "reason", "text": "Razlog dolaska (primjer pitanja)", "type": "text", "required": true},
			{"linkId": "allergy", "text": "Imate li alergiju na lijekove? (primjer pitanja)", "type": "boolean", "required": true},
			{"linkId": "allergy-details", "text": "O alergiji (primjer odjeljka)", "type": "group",
				"enableWhen": when("allergy", "=", "answerBoolean", true), "item": []map[string]any{
					{"linkId": "allergy-what", "text": "Na koji lijek? (primjer pitanja)", "type": "string"},
				}},
			{"linkId": "smoker", "text": "Pušite li? (primjer pitanja)", "type": "boolean"},
			{"linkId": "smoking", "text": "O pušenju (primjer odjeljka)", "type": "group",
				"enableWhen": when("smoker", "=", "answerBoolean", true), "item": []map[string]any{
					{"linkId": "smoking-per-day", "text": "Koliko cigareta dnevno? (primjer pitanja)", "type": "integer"},
					{"linkId": "smoking-quit", "text": "Želite li savjet o prestanku pušenja? (primjer pitanja)", "type": "boolean",
						"enableWhen": when("smoking-per-day", ">", "answerInteger", 10)},
				}},
			{"linkId": "contact", "text": "Kako vas najlakše kontaktirati? (primjer pitanja)", "type": "choice",
				"answerOption": []map[string]string{{"valueString": "Telefon"}, {"valueString": "E-mail"}, {"valueString": "Aplikacija"}}},
		},
	}
}

// questionnaire writes and publishes the form as the first practitioner, asks it of two app patients before their
// next visit, and the first of them fills it in on the branch that opens: one answered, one still waiting.
func (d *demoSeed) questionnaire(ctx context.Context) error {
	st := d.practitioners[0]
	var q struct{ ID string }
	res, err := st.user.must(ctx, http.StatusCreated, http.MethodPost, "/questionnaires", demoQuestionnaire(), &q)
	if err != nil {
		return err
	}
	if _, err := st.user.mustIf(ctx, http.StatusOK, http.MethodPost, "/questionnaires/"+q.ID+"/publish", res.etag, nil, nil); err != nil {
		return err
	}
	free, err := demoFreeSlots(ctx, d.db, st.scheduleID)
	if err != nil {
		return err
	}
	var taken []string
	for _, n := range []int{2, 3} {
		slot, ok := firstFree(free, "in-person", clock.System{}.Now().AddDate(0, 0, 3), taken)
		if !ok {
			return errors.New("demo-seed: no free slot for the questionnaire's visit")
		}
		taken = append(taken, slot.id)
		var booked struct{ ID string }
		if _, err := d.people[n].app.must(ctx, http.StatusCreated, http.MethodPost, "/appointments",
			map[string]string{"slotId": slot.id, "serviceId": d.services[st.practitioner.inPerson]}, &booked); err != nil {
			return err
		}
		var requested struct{ ID string }
		if _, err := st.user.must(ctx, http.StatusCreated, http.MethodPost, "/appointments/"+booked.ID+"/questionnaires",
			map[string]string{"questionnaireId": q.ID}, &requested); err != nil {
			return err
		}
		if n == 2 {
			if err := d.answer(ctx, d.people[n].app, requested.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// answer fills the request in as the patient app does: start, the answers of the questions shown, submit.
func (d *demoSeed) answer(ctx context.Context, patient *demoUser, id string) error {
	path := "/questionnaire-responses/" + id
	res, err := patient.must(ctx, http.StatusOK, http.MethodPost, path+"/start", map[string]string{"language": demoLanguage}, nil)
	if err != nil {
		return err
	}
	answers := []map[string]any{
		{"linkId": "reason", "values": []map[string]any{{"valueString": "Kontrola pritiska (primjer odgovora)"}}},
		{"linkId": "allergy", "values": []map[string]any{{"valueBoolean": true}}},
		{"linkId": "allergy-what", "values": []map[string]any{{"valueString": "Penicilin (primjer odgovora)"}}},
		{"linkId": "smoker", "values": []map[string]any{{"valueBoolean": true}}},
		{"linkId": "smoking-per-day", "values": []map[string]any{{"valueInteger": 15}}},
		{"linkId": "smoking-quit", "values": []map[string]any{{"valueBoolean": true}}},
		{"linkId": "contact", "values": []map[string]any{{"valueString": "Aplikacija"}}},
	}
	if res, err = patient.mustIf(ctx, http.StatusOK, http.MethodPut, path+"/answers", res.etag, map[string]any{"answers": answers}, nil); err != nil {
		return err
	}
	_, err = patient.mustIf(ctx, http.StatusOK, http.MethodPost, path+"/submit", res.etag, nil, nil)
	return err
}
