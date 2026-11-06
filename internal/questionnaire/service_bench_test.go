package questionnaire_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// large is a synthetic questionnaire of 200 items (SPEC.md §2 "Performance"): 10 sections of 19 items each,
// every type, choices of 5 options, help text and conditions on earlier answers.
func large() questionnaire.Content {
	text := strings.Repeat("Placeholder question text ", 4)
	yes := true
	var items []questionnaire.Item
	for s := range 10 {
		sec := questionnaire.Item{LinkID: fmt.Sprintf("s%d", s), Text: "Placeholder section", Type: questionnaire.ItemGroup}
		kinds := []questionnaire.ItemType{questionnaire.ItemBoolean, questionnaire.ItemChoice, questionnaire.ItemString, questionnaire.ItemText,
			questionnaire.ItemInteger, questionnaire.ItemDecimal, questionnaire.ItemDate, questionnaire.ItemDisplay}
		for i := range 19 {
			it := questionnaire.Item{LinkID: fmt.Sprintf("s%d-%d", s, i), Text: text, Type: kinds[i%len(kinds)]}
			switch it.Type {
			case questionnaire.ItemChoice:
				it.Options = opts(5)
				it.Repeats = i%2 == 0
			case questionnaire.ItemDisplay:
			default:
				it.Help, it.Required = "Placeholder help text", true
			}
			if i > 0 && it.Type != questionnaire.ItemDisplay {
				it.EnableWhen = &questionnaire.Condition{Question: fmt.Sprintf("s%d-0", s), Boolean: &yes}
			}
			sec.Items = append(sec.Items, it)
		}
		items = append(items, sec)
	}
	return questionnaire.Content{Title: "Placeholder intake", Description: "Placeholder", Items: items}
}

// BenchmarkPublish validates and stores a 200-item draft, then publishes it, as salus_app with the audit.
func BenchmarkPublish(b *testing.B) {
	owner, app := pgtest.NewAppDatabase(b)
	seed(b, owner)
	svc, _ := realService(app)
	ctx := context.Background()
	in := large()
	for b.Loop() {
		q, err := svc.Create(ctx, adminA, in)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := svc.Publish(ctx, adminA, q.ID, q.VersionID); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadVersion reads a 200-item version, as salus_app with the audit.
func BenchmarkReadVersion(b *testing.B) {
	owner, app := pgtest.NewAppDatabase(b)
	seed(b, owner)
	svc, _ := realService(app)
	ctx := context.Background()
	q, err := svc.Create(ctx, adminA, large())
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := svc.Read(ctx, adminA, q.ID); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLarge_IsAValid200ItemQuestionnaire(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := f.svc.Create(t.Context(), adminA, large()); err != nil {
		t.Errorf("Create() of the benchmark's questionnaire = %v", err)
	}
	over := large()
	over.Items = append(over.Items, question("extra", questionnaire.ItemString))
	if _, err := f.svc.Create(t.Context(), adminA, over); err == nil {
		t.Error("Create() of 201 items = nil, want an InvalidError")
	}
}
