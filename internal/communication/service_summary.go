package communication

import (
	"context"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/llm"
)

// A conversation is long, and worth a summary, when the caller reads at least SummaryMinMessages of its messages:
// one page, what the chat shows when it opens, is read at a glance. A summary covers the latest
// SummaryMaxMessages, which bounds what one call reads and sends to the model (SAL-31, docs/plans/chat.PROGRESS.md).
const (
	SummaryMinMessages = PageSize
	SummaryMaxMessages = 500
	// SummaryTimeout bounds the model's call.
	SummaryTimeout = 30 * time.Second
)

// summaryLanguages are the staff languages (the web app's), in which a doctor asks for a summary; the first is the
// default.
var summaryLanguages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en"}

// Summary is an AI-generated summary of a conversation's messages up to UpToSeq (the SummaryResponse schema). It
// is not stored and never replaces a message; AIGenerated is always true, so every client labels it.
type Summary struct {
	GeneratedAt time.Time `json:"generated_at"`
	Summary     string    `json:"summary"`
	UpToSeq     int64     `json:"up_to_seq"`
	AIGenerated bool      `json:"ai_generated"`
}

// Summarize summarises the latest messages of a long conversation for a practitioner who may read it, in lang (a
// staff language; empty: bs). The read is audited as one read of the rows summarised, in its transaction; the model
// is called after the commit (Ports.Summarizer: the fake, config.LoadChatSummary). ErrForbidden for the patient
// (the summary is a tool for the doctor) and for a practitioner who does not take part, ErrInvalid for a short
// conversation or another language; access otherwise as enter's for a read.
func (s *Service) Summarize(ctx context.Context, c Caller, id, lang string) (Summary, error) {
	if lang == "" {
		lang = summaryLanguages[0]
	}
	if !slices.Contains(summaryLanguages, lang) {
		return Summary{}, invalid("lang")
	}
	var lines []llm.Utterance
	var upTo int64
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		switch {
		case err != nil:
			return err
		case e.who.patient():
			return ErrForbidden
		case e.conv.LastSeq-e.fromSeq() < SummaryMinMessages:
			return invalid("conversation")
		}
		msgs, err := s.Repo.Messages(ctx, id, nil, e.fromSeq(), SummaryMaxMessages)
		if err != nil {
			return err
		}
		upTo = msgs[0].Seq
		for _, m := range slices.Backward(msgs) {
			if m.Kind == KindText && m.Body != "" {
				lines = append(lines, llm.Utterance{ID: m.ID, Speaker: string(m.SenderRole), Text: m.Body})
			}
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeRead, "Patient/"+e.conv.PatientID, entityCommunication, int64(len(msgs)))
	})
	if err != nil {
		return Summary{}, wrap("summary", s.refusal(ctx, c, id, err))
	}
	ctx, cancel := context.WithTimeout(ctx, SummaryTimeout)
	defer cancel()
	text, err := s.Summarizer.Summarize(ctx, llm.SummarizeRequest{Lang: lang, Messages: lines})
	if err != nil {
		return Summary{}, wrap("summary", err)
	}
	return Summary{Summary: text, GeneratedAt: s.Clock.Now().UTC().Truncate(time.Second), UpToSeq: upTo, AIGenerated: true}, nil
}
