//go:build loadtest

package api_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/composition"
)

// The drafting's own time (SAL-8, docs/plans/consult-intelligence.PROGRESS.md): POST /encounters/{id}/drafts
// through api.New on the compose Postgres, Redis and object store, with a 30-minute consultation's stored
// transcript (360 segments), the llmtest drafter answering at once: access, template, transcript read, both
// transactions, the audit and the stream. Then the sign: the PDF rendered, encrypted and put to Garage, and the
// final transaction. A real model's time comes on top of the first. Not part of make test.
func TestDraftingLatency(t *testing.T) {
	s := newStreamApp(t)
	created := s.d.practitioner.do(http.MethodPost, "/document-templates", templateBody)
	var tpl struct{ ID string }
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &tpl) != nil {
		t.Fatalf("POST /document-templates = %d %s", created.Code, created.Body)
	}
	if _, err := s.owner.Exec(t.Context(), `INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, translation, ts_start, ts_end)
		SELECT gen_random_uuid(), $1, CASE WHEN i % 2 = 0 THEN 'patient' ELSE 'practitioner' END, 'bs', $2, $2,
			now() - make_interval(secs => 1800 - i * 5), now() - make_interval(secs => 1800 - i * 5 - 4)
		FROM generate_series(0, 359) AS i`, s.encounter, strings.Repeat("Placeholder words of a sentence ", 4)); err != nil {
		t.Fatal(err)
	}
	var drafts, signs []time.Duration
	for range 50 {
		began := wallNow()
		rec := s.d.practitioner.do(http.MethodPost, "/encounters/"+s.encounter+"/drafts", `{"templateId":"`+tpl.ID+`"}`)
		drafts = append(drafts, wallNow().Sub(began))
		var d composition.Event
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &d) != nil {
			t.Fatalf("draft = %d %s", rec.Code, rec.Body)
		}
		s.d.practitioner.ifMatch = `W/"` + itoa(int(d.VersionID)) + `"`
		began = wallNow()
		rec = s.d.practitioner.do(http.MethodPost, "/drafts/"+d.DraftID+"/sign",
			`{"fields":[{"key":"findings","value":"`+strings.Repeat("Placeholder findings ", 40)+`"},{"key":"tests","value":"one\ntwo"}]}`)
		signs = append(signs, wallNow().Sub(began))
		s.d.practitioner.ifMatch = ""
		if rec.Code != http.StatusOK {
			t.Fatalf("sign = %d %s", rec.Code, rec.Body)
		}
	}
	for name, took := range map[string][]time.Duration{"draft (fake model)": drafts, "sign (PDF, Garage)": signs} {
		slices.Sort(took)
		q := func(p float64) time.Duration { return took[int(p*float64(len(took)-1))] }
		t.Logf("%s, 50 runs, 360 segments: p50 %v, p95 %v, max %v", name, q(0.5), q(0.95), took[len(took)-1])
	}
}
