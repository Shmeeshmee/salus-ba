//go:build e2e && loadtest

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The questionnaire endpoints' latency through the running stack (`make bench-questionnaire-http`,
// docs/questionnaire/SPEC.md §2 "Performance", PROGRESS.md package 6): qRequests saves of a 200-item
// questionnaire's answers, qConcurrency at a time, then qRequests FHIR searches of an appointment's responses with
// their version included, qConcurrency at a time, against the api container on the compose Postgres. Not part of
// `make e2e`.
const (
	qRequests    = 1000
	qConcurrency = 10
	qSections    = 10
	qPerSection  = 19 // 10 sections of 19 questions: 200 items
)

// TestQuestionnaireLatency books qConcurrency synthetic appointments, has the practitioner publish one 200-item
// questionnaire (every answer type) and request it on each, then measures: each worker saves the answers of its
// own response (If-Match chained), qRequests/qConcurrency times; every response is submitted; then the
// practitioner searches QuestionnaireResponse?appointment=…&_include=QuestionnaireResponse:questionnaire round
// the appointments, each answer a completed response with 190 answers nested in sections and the 200-item version.
func TestQuestionnaireLatency(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appointments := []string{p.bookFirstTime(clinic).ID}
	for _, slot := range freeSlots(t, p.api, clinic.scheduleID)[:qConcurrency-1] {
		var booked struct{ ID string }
		p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot}, &booked)
		appointments = append(appointments, booked.ID)
	}

	items, answers := loadQuestionnaire()
	var draft struct{ ID string }
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/questionnaires", map[string]any{"title": "Placeholder load-test questionnaire", "item": items}, &draft)
	clinic.practitioner.mustWith(http.Header{"If-Match": {`W/"1"`}}, http.StatusOK, http.MethodPost, "/questionnaires/"+draft.ID+"/publish", nil, nil)
	responses := make([]string, 0, len(appointments))
	for _, a := range appointments {
		var r struct{ ID string }
		clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/appointments/"+a+"/questionnaires", map[string]string{"questionnaireId": draft.ID}, &r)
		responses = append(responses, r.ID)
	}
	body, err := json.Marshal(map[string]any{"answers": answers})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("a save's body: %d bytes, %d answers", len(body), len(answers))

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: qConcurrency}}
	saves, wall := measureSaves(t, client, p.api, responses, body)
	report(t, "saves", saves, wall)

	for _, id := range responses {
		p.mustWith(http.Header{"If-Match": {fmt.Sprintf(`W/"%d"`, qRequests/qConcurrency+1)}}, http.StatusOK, http.MethodPost,
			"/questionnaire-responses/"+id+"/submit", nil, nil)
	}
	searches, wall := measureSearches(t, client, clinic.practitioner, appointments)
	report(t, "searches", searches, wall)
}

// measureSaves runs the saves: worker i saves response i's answers qRequests/qConcurrency times, each at the
// version the previous one answered.
func measureSaves(t *testing.T, client *http.Client, patient *api, responses []string, body []byte) ([]time.Duration, time.Duration) {
	t.Helper()
	took := make([]time.Duration, 0, qRequests)
	var mu sync.Mutex
	var wg sync.WaitGroup
	failed := make(chan error, qConcurrency)
	began := clock.System{}.Now()
	for _, id := range responses {
		wg.Go(func() {
			for n := 1; n <= qRequests/qConcurrency; n++ {
				d, status, err := timed(t, client, patient, http.MethodPut, "/questionnaire-responses/"+id+"/answers", fmt.Sprintf(`W/"%d"`, n), body)
				if err == nil && status != http.StatusOK {
					err = fmt.Errorf("save %d of %s: status %d", n, id, status)
				}
				if err != nil {
					failed <- err
					return
				}
				mu.Lock()
				took = append(took, d)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	wall := clock.System{}.Now().Sub(began)
	close(failed)
	for err := range failed {
		t.Fatal(err)
	}
	return took, wall
}

// measureSearches runs qRequests searches, qConcurrency at a time, round the appointments.
func measureSearches(t *testing.T, client *http.Client, practitioner *api, appointments []string) ([]time.Duration, time.Duration) {
	t.Helper()
	took := make([]time.Duration, qRequests)
	failed := make([]error, qRequests)
	next := make(chan int)
	var wg sync.WaitGroup
	for range qConcurrency {
		wg.Go(func() {
			for i := range next {
				path := "/fhir/R4/QuestionnaireResponse?appointment=" + appointments[i%len(appointments)] + "&_include=QuestionnaireResponse:questionnaire"
				var status int
				took[i], status, failed[i] = timed(t, client, practitioner, http.MethodGet, path, "", nil)
				if failed[i] == nil && status != http.StatusOK {
					failed[i] = fmt.Errorf("search %d: status %d", i, status)
				}
			}
		})
	}
	began := clock.System{}.Now()
	for i := range qRequests {
		next <- i
	}
	close(next)
	wg.Wait()
	wall := clock.System{}.Now().Sub(began)
	for _, err := range failed {
		if err != nil {
			t.Fatal(err)
		}
	}
	return took, wall
}

// timed sends one request as c and returns how long it took to the last byte of the answer, and its status.
func timed(t *testing.T, client *http.Client, c *api, method, path, ifMatch string, body []byte) (time.Duration, int, error) {
	req, err := http.NewRequestWithContext(t.Context(), method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if strings.HasPrefix(path, "/fhir/") {
		req.Header.Set("Accept", "application/fhir+json")
	}
	began := clock.System{}.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return clock.System{}.Now().Sub(began), resp.StatusCode, err
}

func report(t *testing.T, name string, d []time.Duration, wall time.Duration) {
	t.Helper()
	t.Logf("%d %s, %d concurrent, in %v (%.0f/s): p50 %v  p95 %v  p99 %v  max %v", len(d), name, qConcurrency, wall.Round(time.Millisecond),
		float64(len(d))/wall.Seconds(), pct(d, 50), pct(d, 95), pct(d, 99), pct(d, 100))
}

// loadQuestionnaire is a synthetic 200-item questionnaire, qSections sections of qPerSection questions cycling
// through every answer type, and an answer to each question. Placeholder wording.
func loadQuestionnaire() (items []map[string]any, answers []map[string]any) {
	options := []map[string]string{{"valueString": "Placeholder option A"}, {"valueString": "Placeholder option B"}, {"valueString": "Placeholder option C"}}
	kinds := []struct {
		item  map[string]any
		value []map[string]any
	}{
		{map[string]any{"type": "string"}, []map[string]any{{"valueString": "Placeholder short answer"}}},
		{map[string]any{"type": "text"}, []map[string]any{{"valueString": strings.Repeat("Placeholder longer answer. ", 8)}}},
		{map[string]any{"type": "boolean"}, []map[string]any{{"valueBoolean": true}}},
		{map[string]any{"type": "choice", "answerOption": options}, []map[string]any{{"valueString": "Placeholder option B"}}},
		{map[string]any{"type": "choice", "repeats": true, "answerOption": options}, []map[string]any{{"valueString": "Placeholder option A"}, {"valueString": "Placeholder option C"}}},
		{map[string]any{"type": "integer"}, []map[string]any{{"valueInteger": 42}}},
		{map[string]any{"type": "decimal"}, []map[string]any{{"valueDecimal": 36.6}}},
		{map[string]any{"type": "date"}, []map[string]any{{"valueDate": "2026-01-15"}}},
	}
	for s := range qSections {
		questions := make([]map[string]any, 0, qPerSection)
		for q := range qPerSection {
			k := kinds[(s*qPerSection+q)%len(kinds)]
			id := fmt.Sprintf("s%02d-q%02d", s, q)
			item := map[string]any{"linkId": id, "text": "Placeholder question", "required": true}
			for key, v := range k.item {
				item[key] = v
			}
			questions = append(questions, item)
			answers = append(answers, map[string]any{"linkId": id, "values": k.value})
		}
		items = append(items, map[string]any{"linkId": fmt.Sprintf("s%02d", s), "text": "Placeholder section", "type": "group", "item": questions})
	}
	return items, answers
}
