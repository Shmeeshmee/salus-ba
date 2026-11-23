package questionnaire_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// vector is a file of test/testdata/enablewhen: a questionnaire's items and, here unused, its cases.
type vector struct {
	Name          string `json:"name"`
	Questionnaire struct {
		Items []questionnaire.Item `json:"item"`
	} `json:"questionnaire"`
}

func vectors(t *testing.T) map[string]vector {
	t.Helper()
	paths, err := filepath.Glob("../../test/testdata/enablewhen/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no enableWhen vectors: %v", err)
	}
	out := map[string]vector{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var v vector
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out[filepath.Base(p)] = v
	}
	return out
}

func TestPublish_AcceptsEveryEnableWhenVector(t *testing.T) {
	t.Parallel()
	for name, v := range vectors(t) {
		f := newFixture()
		draft, err := f.svc.Create(t.Context(), adminA, questionnaire.Content{Title: "T", Items: v.Questionnaire.Items})
		if err != nil {
			t.Errorf("%s: Create() = %v, want it stored", name, err)
			continue
		}
		if _, err := f.svc.Publish(t.Context(), adminA, draft.ID, draft.VersionID); err != nil {
			t.Errorf("%s: Publish() = %v, want it published", name, err)
		}
	}
}

func TestConditions_DecodeTheSingleObjectOfBeforeBranching(t *testing.T) {
	t.Parallel()
	var it questionnaire.Item
	if err := json.Unmarshal([]byte(`{"linkId":"q","text":"T","type":"string","enableWhen":{"question":"c","answerString":"Option 1"}}`), &it); err != nil {
		t.Fatal(err)
	}
	want := questionnaire.Conditions{{Question: "c", Operator: questionnaire.OpEqual, String: "Option 1"}}
	if len(it.EnableWhen) != 1 || it.EnableWhen[0] != want[0] {
		t.Errorf("EnableWhen = %+v, want %+v", it.EnableWhen, want)
	}
	if err := json.Unmarshal([]byte(`{"enableWhen":[{"question":"c","operator":"!=","answerString":"Option 1"}]}`), &it); err != nil ||
		len(it.EnableWhen) != 1 || it.EnableWhen[0].Operator != questionnaire.OpNotEqual {
		t.Errorf("the FHIR array decoded as %+v, %v", it.EnableWhen, err)
	}
	if err := json.Unmarshal([]byte(`{"enableWhen":"c"}`), &it); err == nil {
		t.Error("a string decoded as conditions")
	}
	b, _ := json.Marshal(questionnaire.Item{LinkID: "q", EnableWhen: want})
	if string(b) != `{"enableWhen":[{"question":"c","operator":"=","answerString":"Option 1"}],"linkId":"q","text":"","type":""}` {
		t.Errorf("encoded as %s, want the FHIR array", b)
	}
}

func no() *bool { b := false; return &b }

func i32(n int32) *int32 { return &n }

// on gives it the conditions cs, combined by behavior.
func on(it questionnaire.Item, behavior questionnaire.Behavior, cs ...questionnaire.Condition) questionnaire.Item {
	return with(it, func(i *questionnaire.Item) { i.EnableWhen, i.EnableBehavior = cs, behavior })
}

func when(q string, op questionnaire.Operator, change func(*questionnaire.Condition)) questionnaire.Condition {
	c := questionnaire.Condition{Question: q, Operator: op}
	if change != nil {
		change(&c)
	}
	return c
}

var (
	answered    = func(c *questionnaire.Condition) { c.Boolean = yes() }
	unanswered  = func(c *questionnaire.Condition) { c.Boolean = no() }
	option1     = func(c *questionnaire.Condition) { c.String = "Option 1" }
	eighteen    = func(c *questionnaire.Condition) { c.Integer = i32(18) }
	halfDegree  = func(c *questionnaire.Condition) { c.Decimal = "37.5" }
	ides        = func(c *questionnaire.Condition) { c.Date = "2020-03-15" }
	intQ, decQ  = question("n", questionnaire.ItemInteger), question("x", questionnaire.ItemDecimal)
	dateQ, strQ = question("d", questionnaire.ItemDate), question("s", questionnaire.ItemString)
	textQ, q    = question("t", questionnaire.ItemText), question("q", questionnaire.ItemString)
)

// logicCases are the branching rules (docs/clinic-cms, Q1): FHIR R4 enableWhen with every operator that fits the
// question's type, all or any, branches as sections nested in sections.
var logicCases = []struct {
	name  string
	links []string
	items []questionnaire.Item
	valid bool
}{
	{name: "yes/no !=", valid: true, items: []questionnaire.Item{boolQ, on(q, "", when("b", questionnaire.OpNotEqual, answered))}},
	{name: "choice !=", valid: true, items: []questionnaire.Item{choiceQ, on(q, "", when("c", questionnaire.OpNotEqual, option1))}},
	{name: "exists on every kind of question", valid: true, items: []questionnaire.Item{boolQ, choiceQ, intQ, decQ, dateQ, strQ, textQ,
		on(q, questionnaire.BehaviorAny, when("b", questionnaire.OpExists, answered), when("c", questionnaire.OpExists, unanswered),
			when("n", questionnaire.OpExists, answered), when("x", questionnaire.OpExists, answered), when("d", questionnaire.OpExists, answered),
			when("s", questionnaire.OpExists, answered), when("t", questionnaire.OpExists, unanswered))}},
	{name: "every comparison on an integer, a decimal and a date", valid: true, items: []questionnaire.Item{intQ, decQ, dateQ,
		on(q, questionnaire.BehaviorAll, when("n", questionnaire.OpEqual, eighteen), when("n", questionnaire.OpNotEqual, eighteen),
			when("n", questionnaire.OpGreater, eighteen), when("n", questionnaire.OpLess, eighteen), when("n", questionnaire.OpGreaterEqual, eighteen),
			when("n", questionnaire.OpLessEqual, eighteen), when("x", questionnaire.OpGreater, halfDegree), when("x", questionnaire.OpLessEqual, halfDegree),
			when("d", questionnaire.OpLess, ides), when("d", questionnaire.OpGreater, ides), when("d", questionnaire.OpEqual, ides))}},
	{name: "one condition with a behavior", valid: true, items: []questionnaire.Item{boolQ, on(q, questionnaire.BehaviorAll, when("b", questionnaire.OpEqual, answered))}},
	{name: "a branch in a branch in a section", valid: true, items: []questionnaire.Item{section("s1", boolQ,
		on(section("branch", choiceQ, on(section("inner", q), "", when("c", questionnaire.OpEqual, option1))), "", when("b", questionnaire.OpEqual, answered)))}},
	{name: "a section in a section, empty while a draft", valid: true, items: []questionnaire.Item{section("s", section("inner"))}},

	{name: "no operator", items: []questionnaire.Item{boolQ, on(q, "", when("b", "", answered))}, links: []string{"q"}},
	{name: "an unknown operator", items: []questionnaire.Item{boolQ, on(q, "", when("b", "~", answered))}, links: []string{"q"}},
	{name: "> on a yes/no", items: []questionnaire.Item{boolQ, on(q, "", when("b", questionnaire.OpGreater, answered))}, links: []string{"q"}},
	{name: "< on a choice", items: []questionnaire.Item{choiceQ, on(q, "", when("c", questionnaire.OpLess, option1))}, links: []string{"q"}},
	{name: "= on a string", items: []questionnaire.Item{strQ, on(q, "", when("s", questionnaire.OpEqual, func(c *questionnaire.Condition) { c.String = "x" }))}, links: []string{"q"}},
	{name: "!= on a text", items: []questionnaire.Item{textQ, on(q, "", when("t", questionnaire.OpNotEqual, func(c *questionnaire.Condition) { c.String = "x" }))}, links: []string{"q"}},
	{name: "a decimal answer on an integer", items: []questionnaire.Item{intQ, on(q, "", when("n", questionnaire.OpEqual, halfDegree))}, links: []string{"q"}},
	{name: "an integer answer on a decimal", items: []questionnaire.Item{decQ, on(q, "", when("x", questionnaire.OpEqual, eighteen))}, links: []string{"q"}},
	{name: "a decimal with an exponent", items: []questionnaire.Item{decQ, on(q, "", when("x", questionnaire.OpEqual, func(c *questionnaire.Condition) { c.Decimal = "1e3" }))}, links: []string{"q"}},
	{name: "a date that does not exist", items: []questionnaire.Item{dateQ, on(q, "", when("d", questionnaire.OpEqual, func(c *questionnaire.Condition) { c.Date = "2020-02-30" }))}, links: []string{"q"}},
	{name: "a date answer on an integer", items: []questionnaire.Item{intQ, on(q, "", when("n", questionnaire.OpEqual, ides))}, links: []string{"q"}},
	{name: "two answers", items: []questionnaire.Item{intQ, on(q, "", when("n", questionnaire.OpEqual, func(c *questionnaire.Condition) { eighteen(c); ides(c) }))}, links: []string{"q"}},
	{name: "exists without an answer", items: []questionnaire.Item{strQ, on(q, "", when("s", questionnaire.OpExists, nil))}, links: []string{"q"}},
	{name: "exists with an integer too", items: []questionnaire.Item{intQ, on(q, "", when("n", questionnaire.OpExists, func(c *questionnaire.Condition) { answered(c); eighteen(c) }))}, links: []string{"q"}},
	{name: "exists on an attachment", items: []questionnaire.Item{attachment("a", 1, questionnaire.FilePhoto), on(q, "", when("a", questionnaire.OpExists, answered))}, links: []string{"q"}},
	{name: "exists on instructions", items: []questionnaire.Item{question("i", questionnaire.ItemDisplay), on(q, "", when("i", questionnaire.OpExists, answered))}, links: []string{"q"}},
	{name: "two conditions without a behavior", items: []questionnaire.Item{boolQ, intQ, on(q, "", when("b", questionnaire.OpEqual, answered), when("n", questionnaire.OpLess, eighteen))}, links: []string{"q"}},
	{name: "an unknown behavior", items: []questionnaire.Item{boolQ, on(q, "some", when("b", questionnaire.OpEqual, answered))}, links: []string{"q"}},
	{name: "a behavior without a condition", items: []questionnaire.Item{on(q, questionnaire.BehaviorAny)}, links: []string{"q"}},
	{name: "the second condition broken", items: []questionnaire.Item{boolQ, choiceQ,
		on(q, questionnaire.BehaviorAny, when("b", questionnaire.OpEqual, answered), when("c", questionnaire.OpEqual, func(c *questionnaire.Condition) { c.String = "Option 9" }))}, links: []string{"q"}},
	{name: "a section's condition on its own question", items: []questionnaire.Item{on(section("s", boolQ), "", when("b", questionnaire.OpEqual, answered))}, links: []string{"s"}},
	{name: "a condition on a later question in a branch", items: []questionnaire.Item{on(q, "", when("b", questionnaire.OpEqual, answered)), section("s", section("inner", boolQ))}, links: []string{"q"}},
}

func TestCreate_ValidatesTheBranchingLogic(t *testing.T) {
	t.Parallel()
	for _, tt := range logicCases {
		f := newFixture()
		_, err := f.svc.Create(t.Context(), adminA, questionnaire.Content{Title: "T", Items: tt.items})
		var invalid *questionnaire.InvalidError
		switch {
		case tt.valid && err != nil:
			t.Errorf("%s: Create() = %v, want it stored", tt.name, err)
		case tt.valid:
		case !errors.As(err, &invalid):
			t.Errorf("%s: Create() = %v, want an InvalidError", tt.name, err)
		case !slices.Equal(invalid.LinkIDs, tt.links):
			t.Errorf("%s: the error names %v, want %v", tt.name, invalid.LinkIDs, tt.links)
		}
	}
}

func TestPublish_RefusesAnEmptySectionInASection(t *testing.T) {
	t.Parallel()
	f := newFixture()
	draft, err := f.svc.Create(t.Context(), adminA, questionnaire.Content{Title: "T", Items: []questionnaire.Item{section("s", q, section("inner"))}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Publish(t.Context(), adminA, draft.ID, draft.VersionID)
	var invalid *questionnaire.InvalidError
	if !errors.As(err, &invalid) || !slices.Equal(invalid.LinkIDs, []string{"inner"}) {
		t.Errorf("Publish() = %v, want an InvalidError naming inner", err)
	}
}
