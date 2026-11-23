package questionnaireresponse_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// vector is a file of test/testdata/enablewhen, the cases web and mobile evaluate too.
type vector struct {
	Questionnaire struct {
		Items []questionnaire.Item `json:"item"`
	} `json:"questionnaire"`
	Cases []struct {
		Answers map[string][]questionnaireresponse.Value `json:"answers"`
		Name    string                                   `json:"name"`
		Enabled []string                                 `json:"enabled"`
	} `json:"cases"`
}

// order gives the linkIds of the tree in document order.
func order(items []questionnaire.Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.LinkID)
		out = append(out, order(it.Items)...)
	}
	return out
}

// Every vector case through Submit: with the answers of its enabled items only, the response is submitted with
// them; with every answer of the case, the answers to hidden items are refused, named in document order.
func TestSubmit_EvaluatesEveryEnableWhenVector(t *testing.T) {
	paths, err := filepath.Glob("../../test/testdata/enablewhen/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no enableWhen vectors: %v", err)
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var v vector
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, c := range v.Cases {
			name := filepath.Base(p) + ": " + c.Name
			var shown, all []questionnaireresponse.Answer
			var kept, hidden []string
			for _, id := range order(v.Questionnaire.Items) {
				values, ok := c.Answers[id]
				if !ok {
					continue
				}
				all = append(all, questionnaireresponse.Answer{LinkID: id, Values: values})
				if slices.Contains(c.Enabled, id) {
					shown, kept = append(shown, all[len(all)-1]), append(kept, id)
				} else {
					hidden = append(hidden, id)
				}
			}
			submitVector(t, name, v.Questionnaire.Items, shown, kept, nil)
			if len(hidden) > 0 {
				submitVector(t, name+" (hidden answers sent)", v.Questionnaire.Items, all, nil, hidden)
			}
		}
	}
}

func submitVector(t *testing.T, name string, items []questionnaire.Item, answers []questionnaireresponse.Answer, kept, hidden []string) {
	t.Helper()
	f := newFixture(now)
	f.withItems(items...)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.answered(id, answers)
	got, err := f.svc.Submit(t.Context(), me, id, 3)
	if hidden != nil {
		var refused *questionnaireresponse.NotEnabledError
		if !errors.As(err, &refused) || !errors.Is(err, questionnaireresponse.ErrNotEnabled) || !slices.Equal(refused.LinkIDs, hidden) {
			t.Errorf("%s: Submit() = %v, want the hidden %v refused", name, err, hidden)
		}
		return
	}
	var ids []string
	for _, a := range got.Answers {
		ids = append(ids, a.LinkID)
	}
	if err != nil || !slices.Equal(ids, kept) {
		t.Errorf("%s: Submit() = %v, %v, want %v kept", name, ids, err, kept)
	}
}
