package suggestion

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The live suggestions' pace (docs/plans/consult-intelligence.md, SAL-7; agent decisions, for the owner to
// review). A run sends the window to the model once RunEveryLines new final lines have been said, or RunEvery
// has passed with at least one, whichever comes first, and never sooner than MinGap after the last run. It shows
// at most MaxPerRun new suggestions, the most confident first.
const (
	RunEveryLines = 5
	RunEvery      = 20 * time.Second
	MinGap        = 5 * time.Second
	WindowLines   = 30
	MaxPerRun     = 3
)

// The worker's own settings.
const (
	// assistAgent names the worker as the agent of its reads' and writes' audit events.
	assistAgent = "suggestions_worker"
	// suggestTimeout bounds one call of the model.
	suggestTimeout = 15 * time.Second
	// leaseTTL is how long a member holds an encounter without renewing it: longer than a run, so no second
	// member takes an encounter over while a run is under way.
	leaseTTL = 30 * time.Second
	// entriesPage is the most entries one Step reads.
	entriesPage = 500
	// shownToModel bounds the earlier suggestions the model is told of, the latest kept.
	shownToModel = 50
)

// Live is the encounters' Redis Streams as the suggestions worker reads them (repository_redis.go).
type Live interface {
	// Encounters returns the ids of the encounters that have a stream.
	Encounters(ctx context.Context) ([]string, error)
	// Claim takes or renews the lease of owner on the encounter for ttl, and reports whether owner holds it.
	Claim(ctx context.Context, encounterID, owner string, ttl time.Duration) (bool, error)
	// Entries returns at most count entries of the encounter's stream after the entry id after ("" for the
	// first), oldest first.
	Entries(ctx context.Context, encounterID, after string, count int) ([]transcript.Entry, error)
}

// Clinics gives a clinic's default language, the practitioner's (organization.Service).
type Clinics interface {
	DefaultLanguage(ctx context.Context, organizationID string) (string, error)
}

// AssistPorts are the Assistant's dependencies. Tick is how often it reads each stream and looks for new ones
// (a second when zero). Log takes opaque ids, counts and reasons, never what was said or suggested.
type AssistPorts struct {
	Suggester llm.Suggester
	Live      Live
	Clinics   Clinics
	Clock     clock.Clock
	Log       *slog.Logger
	Tick      time.Duration
}

// Assistant is the live suggestions worker: it follows each consultation's stream, sends a rolling window of
// its final lines to the model, checks what comes back, and shows the practitioner what is new (stored, then
// published on the stream). It suggests only while the patient's consent is active, and never repeats what
// was shown, dismissed or pinned already: the stored suggestions, which the practitioner's dismissals reach
// before they are published, decide.
type Assistant struct {
	svc *Service
	AssistPorts
}

// NewAssistant returns the Assistant on svc and p.
func NewAssistant(svc *Service, p AssistPorts) *Assistant {
	if p.Tick <= 0 {
		p.Tick = time.Second
	}
	return &Assistant{svc: svc, AssistPorts: p}
}

// Session is what the Assistant knows of one encounter's stream: how far it has read, the window, and when it
// last ran. One member of the worker holds it at a time (Claim).
type Session struct {
	lastRun     time.Time
	encounterID string
	cursor      string
	lines       []llm.Utterance
	pending     int
	ran         bool
}

// NewSession starts following the encounter now, from its stream's first entry.
func (a *Assistant) NewSession(encounterID string) *Session {
	return &Session{encounterID: encounterID, lastRun: a.Clock.Now()}
}

// Step reads what is new on the session's stream and, when a run is due, runs. Only a failed read of the
// stream or a failed store is returned; what the model gives that the contract does not allow is dropped and
// counted, and a failed model call is logged.
func (a *Assistant) Step(ctx context.Context, s *Session) error {
	entries, err := a.Live.Entries(ctx, s.encounterID, s.cursor, entriesPage)
	if err != nil {
		return fmt.Errorf("suggestion: read the stream: %w", err)
	}
	for _, e := range entries {
		s.cursor = e.ID
		s.take(e.Envelope)
	}
	now := a.Clock.Now()
	if !s.due(now) {
		return nil
	}
	s.lastRun, s.pending, s.ran = now, 0, true
	return a.run(ctx, s)
}

// take adds the entry's final transcript line of the session's encounter to the window; anything else (another
// encounter's line, misrouted to this stream, included) is skipped. A line seen already
// (a final event again) replaces its earlier version and is not new.
func (s *Session) take(envelope []byte) {
	env, err := wire.Decode(envelope)
	if err != nil || env.Type != transcript.MessageType || env.Version != transcript.MessageVersion {
		return
	}
	var e transcript.Event
	if json.Unmarshal(env.Payload, &e) != nil || !e.IsFinal || e.EncounterID != s.encounterID || e.Validate() != nil {
		return
	}
	u := llm.Utterance{ID: e.SegmentID, Speaker: string(e.Speaker), Text: e.Original}
	if i := slices.IndexFunc(s.lines, func(l llm.Utterance) bool { return l.ID == u.ID }); i >= 0 {
		s.lines[i] = u
		return
	}
	s.lines = append(s.lines, u)
	if len(s.lines) > WindowLines {
		s.lines = slices.Delete(s.lines, 0, len(s.lines)-WindowLines)
	}
	s.pending++
}

// due reports whether a run is due at now.
func (s *Session) due(now time.Time) bool {
	since := now.Sub(s.lastRun)
	switch {
	case s.pending == 0, s.ran && since < MinGap:
		return false
	}
	return s.pending >= RunEveryLines || since >= RunEvery
}

// run asks the model for suggestions on the window, and shows those the contract allows that are new.
func (a *Assistant) run(ctx context.Context, s *Session) error {
	enc, shown, err := a.svc.shown(ctx, s.encounterID)
	switch {
	case errors.Is(err, ErrConsentRequired), errors.Is(err, ErrNotFound):
		a.Log.InfoContext(ctx, "suggestions not run", "encounter_id", s.encounterID, "reason", err.Error())
		return nil
	case err != nil:
		return err
	}
	lang, err := a.Clinics.DefaultLanguage(ctx, enc.OrganizationID)
	if err != nil {
		return fmt.Errorf("suggestion: the clinic's language: %w", err)
	}
	window := slices.Clone(s.lines)
	texts := make([]string, 0, len(shown))
	for _, e := range shown[max(0, len(shown)-shownToModel):] {
		texts = append(texts, e.Text)
	}
	callCtx, cancel := context.WithTimeout(ctx, suggestTimeout)
	out, err := a.Suggester.Suggest(callCtx, llm.SuggestRequest{Lang: lang, Transcript: window, Shown: texts})
	cancel()
	if err != nil {
		// Not the error's text: an engine's error may echo what it was sent.
		a.Log.WarnContext(ctx, "suggester failed", "encounter_id", s.encounterID)
		return nil
	}
	fresh, dropped := pick(s.encounterID, out, window, shown)
	if dropped > 0 {
		a.Log.InfoContext(ctx, "suggestions dropped", "encounter_id", s.encounterID, "dropped", dropped)
	}
	for _, e := range fresh {
		err := a.svc.Record(ctx, assistAgent, e)
		switch {
		case errors.Is(err, ErrConsentRequired):
			return nil
		case err != nil:
			return err
		}
		if err := a.svc.publish(ctx, e); err != nil {
			a.Log.WarnContext(ctx, "suggestion not published", "encounter_id", s.encounterID, "suggestion_id", e.ID)
		}
	}
	return nil
}

// shown returns the encounter, in its clinic's scope, and the suggestions shown in it so far (whatever their
// status), audited as the worker's read; ErrConsentRequired, audited, once the patient's care or transcription
// consent with the clinic is no longer active (access.Transcribed, SAL-12), and ErrNotFound for an encounter the
// scope does not show.
func (s *Service) shown(ctx context.Context, encounterID string) (encounter.Encounter, []Event, error) {
	var enc encounter.Encounter
	var out []Event
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if enc, err = s.Encounters.Lookup(ctx, encounterID); err != nil {
			return err
		}
		consent, err := s.Consents.State(ctx, enc.PatientID, enc.OrganizationID)
		switch {
		case err != nil:
			return err
		case !access.Transcribed(consent):
			refused = ErrConsentRequired
			return s.Audit.Record(ctx, systemEvent(assistAgent, enc, auditevent.CodeRead, auditevent.OutcomeMinorFailure, 0))
		}
		if out, err = s.Repo.ByEncounter(ctx, encounterID); err != nil {
			return err
		}
		return s.Audit.Record(ctx, systemEvent(assistAgent, enc, auditevent.CodeRead, auditevent.OutcomeSuccess, int64(len(out))))
	})
	switch {
	case errors.Is(err, encounter.ErrNotFound):
		return encounter.Encounter{}, nil, ErrNotFound
	case err != nil:
		return encounter.Encounter{}, nil, fmt.Errorf("suggestion: shown: %w", err)
	}
	return enc, out, refused
}

// pick returns the model's suggestions to show, as events of the encounter: those the contract allows (dropped
// counts the others), not like one shown before nor like one picked before them, the most confident first, at
// most MaxPerRun.
func pick(encounterID string, out []llm.Suggestion, window []llm.Utterance, shown []Event) (fresh []Event, dropped int) {
	inWindow := make(map[string]bool, len(window))
	for _, u := range window {
		inWindow[u.ID] = true
	}
	var valid []Event
	for _, s := range out {
		e := Event{
			ID: newID(), EncounterID: encounterID, Type: Type(s.Kind), Text: strings.TrimSpace(s.Text), Status: StatusShown,
			Origin: OriginAI, SourceSegmentIDs: slices.Clone(s.SourceIDs), Confidence: s.Confidence,
		}
		if e.Validate() != nil || !cited(e.SourceSegmentIDs, inWindow) {
			dropped++
			continue
		}
		valid = append(valid, e)
	}
	slices.SortStableFunc(valid, func(a, b Event) int { return cmp.Compare(b.Confidence, a.Confidence) })
	seen := make([][]string, 0, len(shown)+MaxPerRun)
	for _, e := range shown {
		seen = append(seen, words(e.Text))
	}
	for _, e := range valid {
		w := words(e.Text)
		if len(fresh) == MaxPerRun || slices.ContainsFunc(seen, func(s []string) bool { return alike(s, w) }) {
			continue
		}
		seen = append(seen, w)
		fresh = append(fresh, e)
	}
	return fresh, dropped
}

// cited reports whether every source is a line of the window.
func cited(sources []string, inWindow map[string]bool) bool {
	for _, id := range sources {
		if !inWindow[id] {
			return false
		}
	}
	return true
}

// words is the text's words, lower case, without punctuation, sorted and each once.
func words(text string) []string {
	w := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	slices.Sort(w)
	return slices.Compact(w)
}

// alikeShare is how much two suggestions' words must share to say the same: three words of four ("ask about
// the cough" and "ask about cough"), while "ask about chest pain" and "ask about back pain" share three of five.
const alikeShare = 0.75

// alike reports whether two suggestions say nearly the same: their word sets (sorted) overlap by alikeShare of
// their union (Jaccard).
func alike(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	common := 0
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch strings.Compare(a[i], b[j]) {
		case 0:
			common++
			i++
			j++
		case -1:
			i++
		default:
			j++
		}
	}
	return float64(common) >= alikeShare*float64(len(a)+len(b)-common)
}

// newID is a random UUID (version 4).
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
