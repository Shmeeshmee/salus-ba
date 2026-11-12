package questionnaireresponse_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// large is a synthetic 200-question version (SPEC.md §2 "Performance") and a full set of answers to it: every
// type in turn, each question required and shown on the first one's answer. No sections: the 200-item limit
// counts them, and 200 answers need 200 questions.
func large() ([]questionnaire.Item, []questionnaireresponse.Answer) {
	yes := true
	kinds := []questionnaire.ItemType{questionnaire.ItemBoolean, questionnaire.ItemChoice, questionnaire.ItemChoice, questionnaire.ItemString,
		questionnaire.ItemText, questionnaire.ItemInteger, questionnaire.ItemDecimal, questionnaire.ItemDate}
	opts := []questionnaire.Option{{Value: "Option A"}, {Value: "Option B"}, {Value: "Option C"}, {Value: "Option D"}, {Value: "Option E"}}
	var items []questionnaire.Item
	var answers []questionnaireresponse.Answer
	for i := range 200 {
		it := question(fmt.Sprintf("q%03d", i), kinds[i%len(kinds)])
		it.Required = true
		if i > 0 {
			it.EnableWhen = &questionnaire.Condition{Question: "q000", Boolean: &yes}
		}
		var v []questionnaireresponse.Value
		switch i % len(kinds) {
		case 0:
			v = []questionnaireresponse.Value{boolean(true)}
		case 1:
			it.Options, v = opts, []questionnaireresponse.Value{str("Option C")}
		case 2:
			it.Options, it.Repeats, v = opts, true, []questionnaireresponse.Value{str("Option A"), str("Option D"), str("Option E")}
		case 3:
			v = []questionnaireresponse.Value{str(strings.Repeat("Placeholder ", 8))}
		case 4:
			v = []questionnaireresponse.Value{str(strings.Repeat("Placeholder answer text. ", 40))}
		case 5:
			v = []questionnaireresponse.Value{integer(int32(i))}
		case 6:
			v = []questionnaireresponse.Value{decimal("1234.5678")}
		case 7:
			v = []questionnaireresponse.Value{date("2030-01-02")}
		}
		items, answers = append(items, it), append(answers, answer(it.LinkID, v...))
	}
	return items, answers
}

func TestLarge_IsA200AnswerSetThatSubmits(t *testing.T) {
	items, in := large()
	f := newFixture(now)
	f.withItems(items...)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	saved, err := f.svc.SaveAnswers(t.Context(), me, id, in, 3)
	if err != nil {
		t.Fatalf("SaveAnswers() = %v", err)
	}
	if got, err := f.svc.Submit(t.Context(), me, id, saved.VersionID); err != nil || len(got.Answers) != 200 {
		t.Errorf("Submit() = %d answers, %v; want all 200", len(got.Answers), err)
	}
}

// benchAudit counts the events recorded inside a database transaction (auditevent's own tests cover the
// storage, and its repository is not this domain's to build).
type benchAudit struct {
	mu sync.Mutex
	n  int
}

func (a *benchAudit) Record(ctx context.Context, _ auditevent.AuditEvent) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("audit outside a transaction")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n++
	return nil
}

func (a *benchAudit) RecordAll(ctx context.Context, es []auditevent.AuditEvent) error {
	for _, e := range es {
		if err := a.Record(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// stubAttendance is the patient's own appointment with an active consent; stubVersion the 200-question
// version. Both are read through other domains' services, measured there.
type stubAttendance struct{ a appointment.Appointment }

func (s stubAttendance) Attendance(context.Context, appointment.Caller, string) (appointment.Attendance, error) {
	return appointment.Attendance{Appointment: s.a, Level: access.Full, Patient: true, ConsentActive: true}, nil
}

func (s stubAttendance) Lookup(context.Context, string) (appointment.Appointment, error) {
	return appointment.Appointment{}, errors.New("not for the benchmarks")
}

func (s stubAttendance) LookupLocked(context.Context, string) (appointment.Appointment, error) {
	return s.a, nil
}

type stubVersion struct{ q questionnaire.Questionnaire }

func (s stubVersion) ActiveDefault(context.Context, string, string, string) (questionnaire.Questionnaire, error) {
	return questionnaire.Questionnaire{}, errors.New("not for the benchmarks")
}

func (s stubVersion) Read(context.Context, questionnaire.Caller, string) (questionnaire.Questionnaire, error) {
	return s.q, nil
}

func (s stubVersion) Referenced(context.Context, questionnaire.Caller, string, string) (questionnaire.Questionnaire, error) {
	return s.q, nil
}

// BenchmarkSaveAnswers saves 200 answers: lock and read the response, check each answer, store them, audit.
func BenchmarkSaveAnswers(b *testing.B) {
	svc, _, id, me := benchService(b)
	_, in := large()
	ctx := context.Background()
	version := int32(1)
	for b.Loop() {
		got, err := svc.SaveAnswers(ctx, me, id, in, version)
		if err != nil {
			b.Fatal(err)
		}
		version = got.VersionID
	}
}

// BenchmarkSubmit submits 200 saved answers: lock and read the response, check them in full, evaluate the
// conditions, store the completed response, audit. The response is put back in progress between runs, off
// the clock.
func BenchmarkSubmit(b *testing.B) {
	svc, owner, id, me := benchService(b)
	_, in := large()
	ctx := context.Background()
	saved, err := svc.SaveAnswers(ctx, me, id, in, 1)
	if err != nil {
		b.Fatal(err)
	}
	version := saved.VersionID
	for b.Loop() {
		got, err := svc.Submit(ctx, me, id, version)
		if err != nil || len(got.Answers) != 200 {
			b.Fatal(len(got.Answers), err)
		}
		b.StopTimer()
		if err := owner.QueryRow(ctx, `UPDATE questionnaire_response SET status = 'in-progress', authored_at = NULL WHERE id = $1
			RETURNING version_id`, id).Scan(&version); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}
