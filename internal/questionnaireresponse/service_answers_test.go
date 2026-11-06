package questionnaireresponse_test

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// withItems gives the active version the items.
func (f *fixture) withItems(items ...questionnaire.Item) {
	v := f.questionnaires.versions[qActive]
	v.Items = items
	f.questionnaires.versions[qActive] = v
}

// answered sets the saved answers of the response id.
func (f *fixture) answered(id string, a []questionnaireresponse.Answer) {
	for i := range f.repo.responses {
		if f.repo.responses[i].ID == id {
			f.repo.responses[i].Answers = a
		}
	}
}

func boolean(b bool) questionnaireresponse.Value  { return questionnaireresponse.Value{Boolean: &b} }
func str(s string) questionnaireresponse.Value    { return questionnaireresponse.Value{String: s} }
func integer(i int32) questionnaireresponse.Value { return questionnaireresponse.Value{Integer: &i} }
func decimal(d string) questionnaireresponse.Value {
	return questionnaireresponse.Value{Decimal: json.Number(d)}
}
func date(d string) questionnaireresponse.Value { return questionnaireresponse.Value{Date: d} }

func answer(linkID string, values ...questionnaireresponse.Value) questionnaireresponse.Answer {
	return questionnaireresponse.Answer{LinkID: linkID, Values: values}
}

func question(linkID string, typ questionnaire.ItemType) questionnaire.Item {
	return questionnaire.Item{LinkID: linkID, Text: "Placeholder question", Type: typ}
}

// typed is a version with an item of every type, a section and an instruction. Synthetic.
func typed() []questionnaire.Item {
	choice := question("c", questionnaire.ItemChoice)
	choice.Options = []questionnaire.Option{{Value: "Option A"}, {Value: "Option B"}, {Value: "Option C"}}
	multi := choice
	multi.LinkID, multi.Repeats = "m", true
	section := question("g", questionnaire.ItemGroup)
	section.Items = []questionnaire.Item{question("gs", questionnaire.ItemString), question("x", questionnaire.ItemDisplay)}
	return []questionnaire.Item{
		question("b", questionnaire.ItemBoolean), question("s", questionnaire.ItemString), question("t", questionnaire.ItemText),
		choice, multi, question("i", questionnaire.ItemInteger), question("d", questionnaire.ItemDecimal),
		question("dt", questionnaire.ItemDate), question("att", questionnaire.ItemAttachment), section,
	}
}

// The patient replaces the saved answers of their in-progress response before it is due, with the version
// they last read: stored, a new version, audited as an update beside their audited read of the
// questionnaire version.
func TestSaveAnswers_ByThePatientReplacesTheAnswersAndAudits(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	in := []questionnaireresponse.Answer{answer("q1", boolean(false)), answer("q2", str("Placeholder answer"))}

	got, err := f.svc.SaveAnswers(t.Context(), me, id, in, 3)

	if err != nil || !equalAnswers(got.Answers, in) || got.VersionID != 4 || !got.LastUpdated.Equal(now) || got.Status != questionnaireresponse.StatusInProgress {
		t.Fatalf("SaveAnswers() = %+v, %v; want the answers at version 4, in progress", got, err)
	}
	if stored := f.repo.responses[0]; !equalAnswers(stored.Answers, in) {
		t.Errorf("stored %+v, want the answers", stored.Answers)
	}
	if !slices.Equal(f.repo.locked, []string{id}) || f.repo.patient != patientA || f.repo.clinic != "" {
		t.Errorf("locked %v in patient %q, clinic %q; want the response, in the patient's scope", f.repo.locked, f.repo.patient, f.repo.clinic)
	}
	if want := []questionnaire.Caller{{AccountID: accountA}}; !slices.Equal(f.questionnaires.callers, want) || !slices.Equal(f.questionnaires.patients, []string{patientA}) {
		t.Errorf("read the version as %+v for %v, want the patient's account for patient A", f.questionnaires.callers, f.questionnaires.patients)
	}
	if len(f.audit.events) != 2 || f.audit.events[0].EntityReference != "Questionnaire/"+qActive {
		t.Errorf("audit %+v, want the version's read, then the update", f.audit.events)
	}
	if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess)}; !slices.Equal(got, want) {
		t.Errorf("audited %+v,\nwant %+v", got, want)
	}
}

func equalAnswers(a, b []questionnaireresponse.Answer) bool {
	return slices.EqualFunc(a, b, func(x, y questionnaireresponse.Answer) bool {
		return x.LinkID == y.LinkID && slices.EqualFunc(x.Values, y.Values, func(v, w questionnaireresponse.Value) bool {
			return (v.Boolean == nil) == (w.Boolean == nil) && (v.Boolean == nil || *v.Boolean == *w.Boolean) &&
				(v.Integer == nil) == (w.Integer == nil) && (v.Integer == nil || *v.Integer == *w.Integer) &&
				v.Decimal == w.Decimal && v.String == w.String && v.Date == w.Date
		})
	})
}

// answerCase is one save of TestSaveAnswers_ChecksEachAnswerAgainstItsItemsTypeAndLimits: valid (ok), or
// refused naming the linkIds.
type answerCase struct {
	name  string
	in    []questionnaireresponse.Answer
	named []string
	ok    bool
}

func answerCases() []answerCase {
	yes := true
	return []answerCase{
		{name: "a boolean", in: []questionnaireresponse.Answer{answer("b", boolean(true))}, ok: true},
		{name: "a boolean false", in: []questionnaireresponse.Answer{answer("b", boolean(false))}, ok: true},
		{name: "a boolean as a string", in: []questionnaireresponse.Answer{answer("b", str("true"))}, named: []string{"b"}},
		{name: "two booleans", in: []questionnaireresponse.Answer{answer("b", boolean(true), boolean(false))}, named: []string{"b"}},
		{name: "no value", in: []questionnaireresponse.Answer{answer("b")}, named: []string{"b"}},
		{name: "a value with two kinds", in: []questionnaireresponse.Answer{answer("b", questionnaireresponse.Value{Boolean: &yes, String: "x"})}, named: []string{"b"}},
		{name: "an empty value", in: []questionnaireresponse.Answer{answer("b", questionnaireresponse.Value{})}, named: []string{"b"}},
		{name: "a string of 255", in: []questionnaireresponse.Answer{answer("s", str(strings.Repeat("a", 255)))}, ok: true},
		{name: "a string of 255 characters, more bytes", in: []questionnaireresponse.Answer{answer("s", str(strings.Repeat("é", 255)))}, ok: true},
		{name: "a string of 256", in: []questionnaireresponse.Answer{answer("s", str(strings.Repeat("a", 256)))}, named: []string{"s"}},
		{name: "a blank string", in: []questionnaireresponse.Answer{answer("s", str("  "))}, named: []string{"s"}},
		{name: "a string as an integer", in: []questionnaireresponse.Answer{answer("s", integer(1))}, named: []string{"s"}},
		{name: "a text of 4,000", in: []questionnaireresponse.Answer{answer("t", str(strings.Repeat("a", 4000)))}, ok: true},
		{name: "a text of 4,001", in: []questionnaireresponse.Answer{answer("t", str(strings.Repeat("a", 4001)))}, named: []string{"t"}},
		{name: "two texts", in: []questionnaireresponse.Answer{answer("t", str("a"), str("b"))}, named: []string{"t"}},
		{name: "a choice", in: []questionnaireresponse.Answer{answer("c", str("Option B"))}, ok: true},
		{name: "a choice not an option", in: []questionnaireresponse.Answer{answer("c", str("Option D"))}, named: []string{"c"}},
		{name: "two choices of a single choice", in: []questionnaireresponse.Answer{answer("c", str("Option A"), str("Option B"))}, named: []string{"c"}},
		{name: "several choices", in: []questionnaireresponse.Answer{answer("m", str("Option A"), str("Option C"))}, ok: true},
		{name: "every choice", in: []questionnaireresponse.Answer{answer("m", str("Option A"), str("Option B"), str("Option C"))}, ok: true},
		{name: "a choice twice", in: []questionnaireresponse.Answer{answer("m", str("Option A"), str("Option A"))}, named: []string{"m"}},
		{name: "one of several choices not an option", in: []questionnaireresponse.Answer{answer("m", str("Option A"), str("Option D"))}, named: []string{"m"}},
		{name: "the largest integer", in: []questionnaireresponse.Answer{answer("i", integer(math.MaxInt32))}, ok: true},
		{name: "the smallest integer", in: []questionnaireresponse.Answer{answer("i", integer(math.MinInt32))}, ok: true},
		{name: "an integer as a decimal", in: []questionnaireresponse.Answer{answer("i", decimal("1.5"))}, named: []string{"i"}},
		{name: "a decimal of 15 digits", in: []questionnaireresponse.Answer{answer("d", decimal("123456789012345"))}, ok: true},
		{name: "a decimal of 16 digits", in: []questionnaireresponse.Answer{answer("d", decimal("1234567890123456"))}, named: []string{"d"}},
		{name: "a fraction of 15 digits", in: []questionnaireresponse.Answer{answer("d", decimal("-123456789012.345"))}, ok: true},
		{name: "a fraction of 16 digits", in: []questionnaireresponse.Answer{answer("d", decimal("1234567890123.456"))}, named: []string{"d"}},
		{name: "leading zeros are not significant", in: []questionnaireresponse.Answer{answer("d", decimal("-0.0000000000000000123456789012345"))}, ok: true},
		{name: "16 digits after leading zeros", in: []questionnaireresponse.Answer{answer("d", decimal("0.00001234567890123456"))}, named: []string{"d"}},
		{name: "a small decimal", in: []questionnaireresponse.Answer{answer("d", decimal("0.000000000000001"))}, ok: true},
		{name: "trailing zeros are kept", in: []questionnaireresponse.Answer{answer("d", decimal("1.50"))}, ok: true},
		{name: "zero", in: []questionnaireresponse.Answer{answer("d", decimal("0"))}, ok: true},
		{name: "an exponent", in: []questionnaireresponse.Answer{answer("d", decimal("1e3"))}, named: []string{"d"}},
		{name: "a leading zero", in: []questionnaireresponse.Answer{answer("d", decimal("01"))}, named: []string{"d"}},
		{name: "no integer part", in: []questionnaireresponse.Answer{answer("d", decimal(".5"))}, named: []string{"d"}},
		{name: "not a number", in: []questionnaireresponse.Answer{answer("d", decimal("abc"))}, named: []string{"d"}},
		{name: "a date", in: []questionnaireresponse.Answer{answer("dt", date("2030-01-31"))}, ok: true},
		{name: "a day that does not exist", in: []questionnaireresponse.Answer{answer("dt", date("2030-02-30"))}, named: []string{"dt"}},
		{name: "a date without padding", in: []questionnaireresponse.Answer{answer("dt", date("2030-1-5"))}, named: []string{"dt"}},
		{name: "the year 0", in: []questionnaireresponse.Answer{answer("dt", date("0000-01-01"))}, named: []string{"dt"}},
		{name: "a date with a time", in: []questionnaireresponse.Answer{answer("dt", date("2030-01-31T09:00:00Z"))}, named: []string{"dt"}},
		{name: "a partial date", in: []questionnaireresponse.Answer{answer("dt", date("2030-01"))}, named: []string{"dt"}},
		{name: "a reserved attachment item", in: []questionnaireresponse.Answer{answer("att", str("x"))}, named: []string{"att"}},
		{name: "a question in a section", in: []questionnaireresponse.Answer{answer("gs", str("Placeholder"))}, ok: true},
		{name: "a section", in: []questionnaireresponse.Answer{answer("g", str("Placeholder"))}, named: []string{"g"}},
		{name: "an instruction", in: []questionnaireresponse.Answer{answer("x", str("Placeholder"))}, named: []string{"x"}},
		{name: "an item twice", in: []questionnaireresponse.Answer{answer("s", str("a")), answer("s", str("b"))}, named: []string{"s"}},
		{name: "an unknown item", in: []questionnaireresponse.Answer{answer("q9", str("a"))}},
		{name: "every type at once", ok: true, in: []questionnaireresponse.Answer{
			answer("b", boolean(true)), answer("s", str("a")), answer("t", str("b")), answer("c", str("Option A")),
			answer("m", str("Option B")), answer("i", integer(-3)), answer("d", decimal("2.5")), answer("dt", date("2030-01-01")), answer("gs", str("c")),
		}},
		{name: "the offending ones in document order", named: []string{"s", "i", "gs"}, in: []questionnaireresponse.Answer{
			answer("gs", str("")), answer("b", boolean(true)), answer("i", str("1")), answer("s", integer(1)),
		}},
		{name: "no answers", ok: true},
	}
}

// Each answer is checked against its item's type and limits (SPEC.md §2 "Limits"), each limit at and one past
// its boundary. A refused save stores nothing and names the offending linkId; an answer to no item of the
// version is refused without being named.
func TestSaveAnswers_ChecksEachAnswerAgainstItsItemsTypeAndLimits(t *testing.T) {
	tests := answerCases()
	for _, tt := range tests {
		f := newFixture(now)
		f.withItems(typed()...)
		id := f.seed(apptA, questionnaireresponse.StatusInProgress)
		f.answered(id, nil)

		got, err := f.svc.SaveAnswers(t.Context(), me, id, tt.in, 3)

		var invalid *questionnaireresponse.AnswersError
		switch {
		case tt.ok && (err != nil || !equalAnswers(got.Answers, tt.in)):
			t.Errorf("%s: SaveAnswers() = %+v, %v; want them saved", tt.name, got.Answers, err)
		case tt.ok:
		case !errors.As(err, &invalid) || !errors.Is(err, questionnaireresponse.ErrAnswersInvalid) || !slices.Equal(invalid.LinkIDs, tt.named):
			t.Errorf("%s: SaveAnswers() = %v (%+v), want an AnswersError naming %v", tt.name, err, invalid, tt.named)
		case f.repo.responses[0].VersionID != 3 || f.repo.responses[0].Answers != nil:
			t.Errorf("%s: stored %+v, want it unchanged", tt.name, f.repo.responses[0])
		}
		outcome := auditevent.OutcomeSuccess
		if !tt.ok {
			outcome = auditevent.OutcomeMinorFailure
		}
		if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, outcome)}; !slices.Equal(got, want) {
			t.Errorf("%s: audited %+v,\nwant %+v", tt.name, got, want)
		}
	}
}

// The integer range is held by the type: a JSON integer outside 32 bits does not decode into a Value.
func TestValue_AnIntegerOutside32BitsDoesNotDecode(t *testing.T) {
	var v questionnaireresponse.Value
	if err := json.Unmarshal([]byte(`{"valueInteger": 2147483647}`), &v); err != nil || *v.Integer != math.MaxInt32 {
		t.Errorf("the largest integer = %v, %v; want it", v.Integer, err)
	}
	for _, in := range []string{`{"valueInteger": 2147483648}`, `{"valueInteger": -2147483649}`, `{"valueInteger": 1.5}`} {
		if err := json.Unmarshal([]byte(in), new(questionnaireresponse.Value)); err == nil {
			t.Errorf("%s decoded, want an error", in)
		}
	}
}

// Saving and submitting are the patient's alone, on an in-progress response, before it is due, with an
// active consent and the version they last read. Every refusal changes nothing and is audited as an update
// that failed: with the entity when the caller may see the response, without it otherwise.
func TestAnswers_RefuseAnyoneButThePatientAndAClosedResponse(t *testing.T) {
	tests := []struct {
		setup   func(f *fixture)
		want    error
		name    string
		status  questionnaireresponse.Status
		at      time.Time
		caller  appointment.Caller
		version int32
		unknown bool
		entity  bool
	}{
		{name: "submitted", caller: me, status: questionnaireresponse.StatusCompleted, want: questionnaireresponse.ErrSubmitted, entity: true},
		{name: "withdrawn", caller: me, status: questionnaireresponse.StatusStopped, want: questionnaireresponse.ErrWithdrawn, entity: true},
		{name: "at the start", caller: me, at: start, want: questionnaireresponse.ErrPastDue, entity: true},
		{name: "after the start", caller: me, at: start.Add(time.Minute), want: questionnaireresponse.ErrPastDue, entity: true},
		{name: "without consent", caller: me, want: questionnaireresponse.ErrConsentRequired, entity: true,
			setup: func(f *fixture) { f.revoke(start) }},
		{name: "a stale version", caller: me, version: 2, want: questionnaireresponse.ErrConflict, entity: true},
		{name: "the appointment's practitioner", caller: pract, want: questionnaireresponse.ErrForbidden, entity: true},
		{name: "the clinic's admin", caller: admin, want: questionnaireresponse.ErrForbidden, entity: true},
		{name: "another practitioner", caller: pract2, want: questionnaireresponse.ErrNotFound},
		{name: "another patient", caller: other, want: questionnaireresponse.ErrNotFound},
		{name: "a patient without a profile", caller: noProfile, want: questionnaireresponse.ErrNotFound},
		{name: "another clinic's admin", caller: adminB, want: questionnaireresponse.ErrNotFound},
		{name: "an unknown response", caller: me, unknown: true, want: questionnaireresponse.ErrNotFound},
	}
	calls := map[string]func(f *fixture, c appointment.Caller, id string, version int32) error{
		"SaveAnswers": func(f *fixture, c appointment.Caller, id string, version int32) error {
			_, err := f.svc.SaveAnswers(t.Context(), c, id, []questionnaireresponse.Answer{answer("q1", boolean(false))}, version)
			return err
		},
		"Submit": func(f *fixture, c appointment.Caller, id string, version int32) error {
			_, err := f.svc.Submit(t.Context(), c, id, version)
			return err
		},
	}
	for call, do := range calls {
		for _, tt := range tests {
			at := now
			if !tt.at.IsZero() {
				at = tt.at
			}
			f := newFixture(at)
			status := tt.status
			if status == "" {
				status = questionnaireresponse.StatusInProgress
			}
			id := f.seed(apptA, status)
			if tt.setup != nil {
				tt.setup(f)
			}
			before := f.repo.responses[0]
			if tt.unknown {
				id = unknownID
			}
			version := tt.version
			if version == 0 {
				version = 3
			}

			err := do(f, tt.caller, id, version)

			if !errors.Is(err, tt.want) {
				t.Errorf("%s, %s: = %v, want %v", call, tt.name, err, tt.want)
			}
			if !equal(f.repo.responses[0], before) {
				t.Errorf("%s, %s: stored %+v, want it unchanged", call, tt.name, f.repo.responses[0])
			}
			entity := ""
			if tt.entity {
				entity = id
			}
			if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeUpdate, entity, auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
				t.Errorf("%s, %s: audited %+v,\nwant %+v (and no read of the version)", call, tt.name, f.audit.events, want)
			}
		}
	}
}

// A revocation refuses the very next save.
func TestSaveAnswers_TheSaveAfterARevocationIsRefused(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	got, err := f.svc.SaveAnswers(t.Context(), me, id, answers, 3)
	if err != nil {
		t.Fatalf("SaveAnswers() before the revocation = %v", err)
	}
	f.revoke(start)

	if _, err := f.svc.SaveAnswers(t.Context(), me, id, answers, got.VersionID); !errors.Is(err, questionnaireresponse.ErrConsentRequired) {
		t.Errorf("SaveAnswers() after the revocation = %v, want ErrConsentRequired", err)
	}
	if _, err := f.svc.Submit(t.Context(), me, id, got.VersionID); !errors.Is(err, questionnaireresponse.ErrConsentRequired) {
		t.Errorf("Submit() after the revocation = %v, want ErrConsentRequired", err)
	}
}

// One second before the start the answers may still be saved.
func TestSaveAnswers_UntilTheStart(t *testing.T) {
	f := newFixture(start.Add(-time.Second))
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	if _, err := f.svc.SaveAnswers(t.Context(), me, id, answers, 3); err != nil {
		t.Errorf("SaveAnswers() a second before the start = %v, want it saved", err)
	}
}

// Submit completes the response with the saved answers, at the clock's time, audited as an update; it is final.
func TestSubmit_CompletesTheResponseAtTheClocksTimeOnce(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)

	got, err := f.svc.Submit(t.Context(), me, id, 3)

	if err != nil || got.Status != questionnaireresponse.StatusCompleted || !got.AuthoredAt.Equal(now) || got.VersionID != 4 || !equalAnswers(got.Answers, answers) {
		t.Fatalf("Submit() = %+v, %v; want it completed at now with its answers, version 4", got, err)
	}
	if stored := f.repo.responses[0]; stored.Status != questionnaireresponse.StatusCompleted || !stored.AuthoredAt.Equal(now) {
		t.Errorf("stored %+v, want it completed", stored)
	}
	if !slices.Equal(f.repo.locked, []string{id}) {
		t.Errorf("locked %v, want the response", f.repo.locked)
	}
	if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeSuccess)}; !slices.Equal(got, want) {
		t.Errorf("audited %+v,\nwant %+v", got, want)
	}
	if _, err := f.svc.Submit(t.Context(), me, id, got.VersionID); !errors.Is(err, questionnaireresponse.ErrSubmitted) {
		t.Errorf("Submit() again = %v, want ErrSubmitted", err)
	}
	if _, err := f.svc.SaveAnswers(t.Context(), me, id, answers, got.VersionID); !errors.Is(err, questionnaireresponse.ErrSubmitted) {
		t.Errorf("SaveAnswers() after the submit = %v, want ErrSubmitted", err)
	}
}

// conditional is a version whose items show on earlier answers (SPEC.md §2 "Item model"): a boolean, a single
// choice, a choice with repeats, a section shown on the boolean, and a chain of two (x1 on s2, in the section).
// Every conditional question is required. Synthetic.
func conditional() []questionnaire.Item {
	yes, no := true, false
	on := func(it questionnaire.Item, q string, b *bool, s string) questionnaire.Item {
		it.EnableWhen, it.Required = &questionnaire.Condition{Question: q, Boolean: b, String: s}, true
		return it
	}
	choice := question("c", questionnaire.ItemChoice)
	choice.Options = []questionnaire.Option{{Value: "Option A"}, {Value: "Option B"}}
	multi := question("m", questionnaire.ItemChoice)
	multi.Options, multi.Repeats = []questionnaire.Option{{Value: "Option A"}, {Value: "Option B"}, {Value: "Option C"}}, true
	section := on(question("sec", questionnaire.ItemGroup), "b", &no, "")
	section.Required = false
	s1 := question("s1", questionnaire.ItemString)
	s1.Required = true
	section.Items = []questionnaire.Item{s1, question("s2", questionnaire.ItemBoolean)}
	return []questionnaire.Item{
		question("b", questionnaire.ItemBoolean), on(question("b1", questionnaire.ItemString), "b", &yes, ""),
		choice, on(question("c1", questionnaire.ItemString), "c", nil, "Option B"),
		multi, on(question("m1", questionnaire.ItemString), "m", nil, "Option B"),
		section, on(question("x1", questionnaire.ItemString), "s2", &yes, ""),
	}
}

// Submit checks every enabled required question is answered, naming the missing ones in document order; a
// hidden question is not required, and the answers of hidden items are removed.
func TestSubmit_EvaluatesTheConditions(t *testing.T) {
	v := str("Placeholder")
	tests := []struct {
		name    string
		in      []questionnaireresponse.Answer
		missing []string
		kept    []string
	}{
		{name: "a boolean shows its question", in: []questionnaireresponse.Answer{answer("b", boolean(true))}, missing: []string{"b1"}},
		{name: "a boolean's question answered", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v)}, kept: []string{"b", "b1"}},
		{name: "a boolean hides its question, shows the section", in: []questionnaireresponse.Answer{answer("b", boolean(false)), answer("b1", v)}, missing: []string{"s1"}},
		{name: "a chain through the section", in: []questionnaireresponse.Answer{answer("b", boolean(false)), answer("s1", v), answer("s2", boolean(true))}, missing: []string{"x1"}},
		{name: "the chain answered", in: []questionnaireresponse.Answer{answer("b", boolean(false)), answer("s1", v), answer("s2", boolean(true)), answer("x1", v)},
			kept: []string{"b", "s1", "s2", "x1"}},
		{name: "the chain off", in: []questionnaireresponse.Answer{answer("b", boolean(false)), answer("s1", v), answer("s2", boolean(false)), answer("x1", v)},
			kept: []string{"b", "s1", "s2"}},
		{name: "a hidden section hides its items and the chain", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v), answer("s1", v), answer("s2", boolean(true)), answer("x1", v)},
			kept: []string{"b", "b1"}},
		{name: "a single choice shows its question", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v), answer("c", str("Option B"))}, missing: []string{"c1"}},
		{name: "a single choice hides its question", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v), answer("c", str("Option A")), answer("c1", v)},
			kept: []string{"b", "b1", "c"}},
		{name: "any of several choices shows its question", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v), answer("m", str("Option A"), str("Option B"))}, missing: []string{"m1"}},
		{name: "several choices without it hide it", in: []questionnaireresponse.Answer{answer("b", boolean(true)), answer("b1", v), answer("m", str("Option A"), str("Option C")), answer("m1", v)},
			kept: []string{"b", "b1", "m"}},
		{name: "an unanswered question hides its own", in: []questionnaireresponse.Answer{answer("c1", v), answer("m1", v), answer("b1", v)}},
		{name: "the missing ones in document order", in: []questionnaireresponse.Answer{answer("m", str("Option B")), answer("c", str("Option B")), answer("b", boolean(false))},
			missing: []string{"c1", "m1", "s1"}},
	}
	for _, tt := range tests {
		f := newFixture(now)
		f.withItems(conditional()...)
		id := f.seed(apptA, questionnaireresponse.StatusInProgress)
		f.answered(id, tt.in)

		got, err := f.svc.Submit(t.Context(), me, id, 3)

		var invalid *questionnaireresponse.AnswersError
		switch {
		case tt.missing != nil && (!errors.As(err, &invalid) || !slices.Equal(invalid.LinkIDs, tt.missing)):
			t.Errorf("%s: Submit() = %v (%+v), want the missing %v", tt.name, err, invalid, tt.missing)
		case tt.missing != nil && f.repo.responses[0].Status != questionnaireresponse.StatusInProgress:
			t.Errorf("%s: stored %+v, want it in progress", tt.name, f.repo.responses[0])
		case tt.missing == nil && err != nil:
			t.Errorf("%s: Submit() = %v, want it submitted", tt.name, err)
		case tt.missing == nil:
			var kept []string
			for _, a := range got.Answers {
				kept = append(kept, a.LinkID)
			}
			if !slices.Equal(kept, tt.kept) || !equalAnswers(f.repo.responses[0].Answers, got.Answers) {
				t.Errorf("%s: kept %v, want %v", tt.name, kept, tt.kept)
			}
		}
	}
}

// Submit checks the saved answers in full again: one that breaks its item's rules is named.
func TestSubmit_RefusesASavedAnswerThatBreaksItsItem(t *testing.T) {
	f := newFixture(now)
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	f.answered(id, []questionnaireresponse.Answer{answer("q1", boolean(true)), answer("q2", str(strings.Repeat("a", 256)))})

	_, err := f.svc.Submit(t.Context(), me, id, 3)

	var invalid *questionnaireresponse.AnswersError
	if !errors.As(err, &invalid) || !slices.Equal(invalid.LinkIDs, []string{"q2"}) {
		t.Errorf("Submit() = %v, want q2 named", err)
	}
	if got, want := own(f.audit.events), []auditevent.AuditEvent{event(me, auditevent.CodeUpdate, id, auditevent.OutcomeMinorFailure)}; !slices.Equal(got, want) {
		t.Errorf("audited %+v,\nwant %+v", got, want)
	}
}

// No error names a linkId or holds an answer: they reach logs.
func TestAnswers_ErrorsHoldNoAnswerAndNoLinkID(t *testing.T) {
	f := newFixture(now)
	f.withItems(question("linkid-not-in-errors", questionnaire.ItemString))
	id := f.seed(apptA, questionnaireresponse.StatusInProgress)
	value := "Placeholder answer not in errors " + strings.Repeat("a", 255)

	_, saveErr := f.svc.SaveAnswers(t.Context(), me, id, []questionnaireresponse.Answer{answer("linkid-not-in-errors", str(value))}, 3)
	f.repo.err = errBackend
	_, backendErr := f.svc.SaveAnswers(t.Context(), me, id, []questionnaireresponse.Answer{answer("linkid-not-in-errors", str("Placeholder answer not in errors"))}, 3)

	for _, err := range []error{saveErr, backendErr} {
		if err == nil || strings.Contains(err.Error(), "linkid-not-in-errors") || strings.Contains(err.Error(), "Placeholder answer") {
			t.Errorf("error %q, want one without the linkId or the answer", err)
		}
	}
}

func TestAnswers_FailWithTheBackendAndKeepNothing(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"the repository":     func(f *fixture) { f.repo.err = errBackend },
		"the appointments":   func(f *fixture) { f.appts.err = errBackend },
		"the questionnaires": func(f *fixture) { f.questionnaires.err = errBackend },
		"a missing version":  func(f *fixture) { delete(f.questionnaires.versions, qActive) },
	} {
		for call, do := range map[string]func(f *fixture, id string) error{
			"SaveAnswers": func(f *fixture, id string) error {
				_, err := f.svc.SaveAnswers(t.Context(), me, id, answers, 3)
				return err
			},
			"Submit": func(f *fixture, id string) error { _, err := f.svc.Submit(t.Context(), me, id, 3); return err },
		} {
			f := newFixture(now)
			id := f.seed(apptA, questionnaireresponse.StatusInProgress)
			before := f.repo.responses[0]
			setup(f)
			if err := do(f, id); err == nil || errors.Is(err, questionnaireresponse.ErrNotFound) {
				t.Errorf("%s, %s: = %v, want the failure", call, name, err)
			}
			if !equal(f.repo.responses[0], before) || len(f.audit.events) != 0 {
				t.Errorf("%s, %s: kept %+v and %d events, want nothing changed (rolled back)", call, name, f.repo.responses[0], len(f.audit.events))
			}
		}
	}
}
