package questionnaire_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

func yes() *bool { b := true; return &b }

func opts(n int) []questionnaire.Option {
	out := make([]questionnaire.Option, n)
	for i := range out {
		out[i] = questionnaire.Option{Value: fmt.Sprintf("Option %d", i+1)}
	}
	return out
}

func question(id string, t questionnaire.ItemType) questionnaire.Item {
	it := questionnaire.Item{LinkID: id, Text: "Placeholder", Type: t}
	if t == questionnaire.ItemChoice {
		it.Options = opts(2)
	}
	return it
}

func with(it questionnaire.Item, change func(*questionnaire.Item)) questionnaire.Item {
	change(&it)
	return it
}

func section(id string, children ...questionnaire.Item) questionnaire.Item {
	return questionnaire.Item{LinkID: id, Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: children}
}

func flat(n int) []questionnaire.Item {
	out := make([]questionnaire.Item, n)
	for i := range out {
		out[i] = question(fmt.Sprintf("i%d", i), questionnaire.ItemString)
	}
	return out
}

var (
	r              = strings.Repeat
	boolQ, choiceQ = question("b", questionnaire.ItemBoolean), question("c", questionnaire.ItemChoice)
)

// onB gives it a condition on the boolean b.
func onB(it questionnaire.Item) questionnaire.Item {
	return with(it, func(i *questionnaire.Item) { i.EnableWhen = &questionnaire.Condition{Question: "b", Boolean: yes()} })
}

// validationCases are questionnaires at and past each rule; an invalid one names the offending linkIds.
var validationCases = []struct {
	name  string
	links []string // the linkIds the error names
	in    questionnaire.Content
	valid bool
}{
	{name: "every type", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{
		section("s", question("d", questionnaire.ItemDisplay), question("st", questionnaire.ItemString), question("tx", questionnaire.ItemText)),
		boolQ, choiceQ, with(question("m", questionnaire.ItemChoice), func(i *questionnaire.Item) { i.Repeats = true; i.Required = true }),
		question("n", questionnaire.ItemInteger), question("x", questionnaire.ItemDecimal), question("dt", questionnaire.ItemDate),
	}}},
	{name: "no items", valid: true, in: questionnaire.Content{Title: "T"}},
	{name: "an empty section, which a draft keeps", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{section("s")}}},
	{name: "the reserved attachment type", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("a", questionnaire.ItemAttachment)}}, links: []string{"a"}},
	{name: "an unknown type", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("u", "open-choice")}}, links: []string{"u"}},

	{name: "a title of 200", valid: true, in: questionnaire.Content{Title: r("é", 200)}},
	{name: "a title of 201", in: questionnaire.Content{Title: r("é", 201)}},
	{name: "a blank title", in: questionnaire.Content{Title: "  "}},
	{name: "a description of 1000", valid: true, in: questionnaire.Content{Title: "T", Description: r("d", 1000)}},
	{name: "a description of 1001", in: questionnaire.Content{Title: "T", Description: r("d", 1001)}},
	{name: "200 items", valid: true, in: questionnaire.Content{Title: "T", Items: flat(200)}},
	{name: "201 items", in: questionnaire.Content{Title: "T", Items: flat(201)}},
	{name: "201 items with sections", in: questionnaire.Content{Title: "T", Items: append(flat(199), section("s", question("in", questionnaire.ItemString)))}},
	{name: "a text of 500", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Text = r("t", 500) })}}},
	{name: "a text of 501", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Text = r("t", 501) })}}, links: []string{"q"}},
	{name: "an item without text", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Text = " " })}}, links: []string{"q"}},
	{name: "a help text of 500", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Help = r("h", 500) })}}},
	{name: "a help text of 501", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Help = r("h", 501) })}}, links: []string{"q"}},
	{name: "help text on instructions", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("d", questionnaire.ItemDisplay), func(i *questionnaire.Item) { i.Help = "h" })}}, links: []string{"d"}},
	{name: "help text on a section", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(section("s"), func(i *questionnaire.Item) { i.Help = "h" })}}},

	{name: "2 options", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = opts(2) })}}},
	{name: "1 option", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = opts(1) })}}, links: []string{"c"}},
	{name: "50 options", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = opts(50) })}}},
	{name: "51 options", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = opts(51) })}}, links: []string{"c"}},
	{name: "an option of 200", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = []questionnaire.Option{{Value: r("o", 200)}, {Value: "b"}} })}}},
	{name: "an option of 201", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = []questionnaire.Option{{Value: r("o", 201)}, {Value: "b"}} })}}, links: []string{"c"}},
	{name: "an empty option", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = []questionnaire.Option{{Value: ""}, {Value: "b"}} })}}, links: []string{"c"}},
	{name: "options twice", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Options = []questionnaire.Option{{Value: "a"}, {Value: "a"}} })}}, links: []string{"c"}},
	{name: "options on a non-choice", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Options = opts(2) })}}, links: []string{"q"}},
	{name: "repeats on a non-choice", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Repeats = true })}}, links: []string{"q"}},
	{name: "required instructions", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("d", questionnaire.ItemDisplay), func(i *questionnaire.Item) { i.Required = true })}}, links: []string{"d"}},
	{name: "a required section", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(section("s"), func(i *questionnaire.Item) { i.Required = true })}}, links: []string{"s"}},

	{name: "a section in a section", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{section("s", section("inner"))}}, links: []string{"inner"}},
	{name: "a question with children", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.Items = flat(1) })}}, links: []string{"q"}},

	{name: "a linkId of 64", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question(r("a", 64), questionnaire.ItemString)}}},
	{name: "a linkId of 65", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question(r("a", 65), questionnaire.ItemString)}}},
	{name: "a UUID linkId", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("0b5e3c7a-9d0e-4f7e-8a3b-2c1d0e9f8a7b", questionnaire.ItemString)}}},
	{name: "an empty linkId", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("", questionnaire.ItemString)}}},
	{name: "a malformed linkId", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("a.b", questionnaire.ItemString)}}},
	{name: "a duplicate linkId", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("q", questionnaire.ItemString), section("s", question("q", questionnaire.ItemText))}}, links: []string{"q"}},

	{name: "a condition on an earlier boolean", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{boolQ, onB(question("q", questionnaire.ItemString))}}},
	{name: "a condition on an earlier choice", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{choiceQ,
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) {
			i.EnableWhen = &questionnaire.Condition{Question: "c", String: "Option 2"}
		})}}},
	{name: "a condition on a multiple choice", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(choiceQ, func(i *questionnaire.Item) { i.Repeats = true }),
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) {
			i.EnableWhen = &questionnaire.Condition{Question: "c", String: "Option 1"}
		})}}},
	{name: "a condition on a section", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{boolQ, onB(section("s", question("q", questionnaire.ItemString)))}}},
	{name: "a condition on a hidden section's item", valid: true, in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{
		boolQ, onB(section("s", question("b2", questionnaire.ItemBoolean))),
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.EnableWhen = &questionnaire.Condition{Question: "b2", Boolean: yes()} })}}},
	{name: "a condition on a later item", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{onB(question("q", questionnaire.ItemString)), boolQ}}, links: []string{"q"}},
	{name: "a condition on itself", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{with(boolQ, func(i *questionnaire.Item) { i.EnableWhen = &questionnaire.Condition{Question: "b", Boolean: yes()} })}}, links: []string{"b"}},
	{name: "a condition on a missing item", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{onB(question("q", questionnaire.ItemString))}}, links: []string{"q"}},
	{name: "a condition on a non-boolean non-choice item", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{question("b", questionnaire.ItemString), onB(question("q", questionnaire.ItemString))}}, links: []string{"q"}},
	{name: "a condition on a section's answer", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{section("b"), onB(question("q", questionnaire.ItemString))}}, links: []string{"q"}},
	{name: "a condition with a missing option", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{choiceQ,
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) {
			i.EnableWhen = &questionnaire.Condition{Question: "c", String: "Option 3"}
		})}}, links: []string{"q"}},
	{name: "a condition with a string on a boolean", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{boolQ,
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.EnableWhen = &questionnaire.Condition{Question: "b", String: "Yes"} })}}, links: []string{"q"}},
	{name: "a condition with a boolean on a choice", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{choiceQ,
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) {
			i.EnableWhen = &questionnaire.Condition{Question: "c", Boolean: yes(), String: "Option 1"}
		})}}, links: []string{"q"}},
	{name: "a condition with no answer", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{boolQ,
		with(question("q", questionnaire.ItemString), func(i *questionnaire.Item) { i.EnableWhen = &questionnaire.Condition{Question: "b"} })}}, links: []string{"q"}},
	{name: "two offending items, in document order", in: questionnaire.Content{Title: "T", Items: []questionnaire.Item{
		question("a", questionnaire.ItemAttachment), section("s", with(question("t", questionnaire.ItemString), func(i *questionnaire.Item) { i.Text = "" }))}}, links: []string{"a", "t"}},
}

func TestCreate_ValidatesTheQuestionnaire(t *testing.T) {
	t.Parallel()
	for _, tt := range validationCases {
		f := newFixture()
		got, err := f.svc.Create(t.Context(), adminA, tt.in)
		var invalid *questionnaire.InvalidError
		switch {
		case tt.valid && err != nil:
			t.Errorf("%s: Create() = %v, want it stored", tt.name, err)
		case tt.valid:
			if got.ID == "" {
				t.Errorf("%s: Create() stored nothing", tt.name)
			}
		case !errors.As(err, &invalid) || !errors.Is(err, questionnaire.ErrInvalid):
			t.Errorf("%s: Create() = %v, want an InvalidError", tt.name, err)
		case !slices.Equal(invalid.LinkIDs, tt.links):
			t.Errorf("%s: the error names %v, want %v", tt.name, invalid.LinkIDs, tt.links)
		default:
			if len(f.repo.rows) != 3 {
				t.Errorf("%s: an invalid questionnaire was stored", tt.name)
			}
			assertEvents(t, f.audit, staffEvent(adminA, auditevent.CodeCreate, auditevent.ActionCreate, "", auditevent.OutcomeMinorFailure))
		}
	}
}

func TestUpdateDraft_ValidatesAndRefusesALinkIDOfAnEarlierVersionWithAnotherType(t *testing.T) {
	t.Parallel()
	// Version 1 of series A has q1 boolean and q2 single choice.
	tests := []struct {
		name  string
		items []questionnaire.Item
		valid bool
	}{
		{name: "the same types", valid: true, items: items()},
		{name: "an item removed and a new one", valid: true, items: []questionnaire.Item{question("q1", questionnaire.ItemBoolean), question("q3", questionnaire.ItemDate)}},
		{name: "a boolean become a string", items: []questionnaire.Item{question("q1", questionnaire.ItemString)}},
		{name: "a single choice become multiple", items: []questionnaire.Item{with(question("q2", questionnaire.ItemChoice), func(i *questionnaire.Item) { i.Repeats = true })}},
		{name: "a question become a section", items: []questionnaire.Item{section("q1")}},
		{name: "an invalid item", items: []questionnaire.Item{question("q1", questionnaire.ItemAttachment)}},
	}
	for _, tt := range tests {
		f := newFixture()
		_, err := f.svc.UpdateDraft(t.Context(), adminA, draftA, questionnaire.Content{Title: "T", Items: tt.items}, 1)
		var invalid *questionnaire.InvalidError
		switch {
		case tt.valid && err != nil:
			t.Errorf("%s: UpdateDraft() = %v, want it stored", tt.name, err)
		case !tt.valid && (!errors.As(err, &invalid) || len(invalid.LinkIDs) != 1 || invalid.LinkIDs[0] != tt.items[0].LinkID):
			t.Errorf("%s: UpdateDraft() = %v, want an InvalidError naming %s", tt.name, err, tt.items[0].LinkID)
		case !tt.valid:
			if f.row(draftA).VersionID != 1 {
				t.Errorf("%s: the draft changed", tt.name)
			}
			assertEvents(t, f.audit, staffEvent(adminA, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+draftA, auditevent.OutcomeMinorFailure))
		}
	}
}

func TestService_ErrorsHoldNoQuestionnaireContent(t *testing.T) {
	t.Parallel()
	f := newFixture()
	bad := content()
	bad.Items = append(bad.Items, question("a", questionnaire.ItemAttachment))
	_, err := f.svc.Create(t.Context(), adminA, bad)
	assertNoContent(t, err)
	_, err = f.svc.UpdateDraft(t.Context(), adminA, activeA, content(), 2)
	assertNoContent(t, err)
	f.repo.err = errors.New("db down")
	_, err = f.svc.Read(t.Context(), adminA, activeA)
	assertNoContent(t, err)
	if err == nil {
		t.Error("Read() with the repository failing = nil")
	}
}
