package communication

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// The search's limits (SAL-30).
const (
	SearchSize   = 50  // results, the newest
	MaxQuery     = 200 // characters of a query
	maxQueryTerm = 8   // words of a query that count
)

// Search returns the messages of the conversation whose body has every word of q (each a word's beginning, case
// and accents aside: chat_search, migration 00044), newest first, at most SearchSize, of those the caller reads
// (enter; after their history_from_seq), with their files and terms; audited as one search of the patient's chat.
// ErrInvalid for a query without a word of two letters or longer than MaxQuery; access as enter's for a read.
func (s *Service) Search(ctx context.Context, c Caller, id, q string) ([]SearchResult, error) {
	query, err := searchQuery(q)
	if err != nil {
		return nil, err
	}
	var out []Message
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.enter(ctx, c, id, false, false)
		if err != nil {
			return err
		}
		if out, err = s.Repo.SearchConversation(ctx, id, query, e.fromSeq(), SearchSize); err != nil {
			return err
		}
		if err := s.withAttachments(ctx, out); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeSearchType, "Patient/"+e.conv.PatientID, entityCommunication, int64(len(out)))
	})
	if err != nil {
		return nil, wrap("search", s.refusal(ctx, c, id, err))
	}
	return s.results(ctx, c, out), nil
}

// SearchAll is Search across the caller's conversations: a patient's own, never a doctor-only thread; a
// practitioner's, those their role takes part in, after its history_from_seq and while the patient's care consent
// is active (the practitioner role's row-level security scope, migration 00044); an admin's none (ErrForbidden).
// Audited as one search.
func (s *Service) SearchAll(ctx context.Context, c Caller, q string) ([]SearchResult, error) {
	query, err := searchQuery(q)
	if err != nil {
		return nil, err
	}
	var out []Message
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		w, err := s.identify(ctx, c)
		if err != nil {
			return err
		}
		if !w.patient() {
			if err := s.Repo.SetPractitionerRole(ctx, w.role); err != nil {
				return err
			}
		}
		if out, err = s.Repo.Search(ctx, query, SearchSize); err != nil {
			return err
		}
		if out, err = s.readableResults(ctx, w, out); err != nil {
			return err
		}
		if err := s.withAttachments(ctx, out); err != nil {
			return err
		}
		return s.record(ctx, c, auditevent.ActionRead, auditevent.CodeSearchType, "", entityCommunication, int64(len(out)))
	})
	if err != nil {
		return nil, wrap("search", s.refusal(ctx, c, "", err))
	}
	return s.results(ctx, c, out), nil
}

// readableResults keeps of msgs, found by the scope's row-level security, those the caller reads by the chat's own rules,
// as enter decides for one conversation: access.ReadsChat over the participant and the patient's consent, after
// their history_from_seq. The database and the service decide alike (CLAUDE.md, "Access control in the service").
func (s *Service) readableResults(ctx context.Context, w who, msgs []Message) ([]Message, error) {
	from := map[string]int64{} // per conversation, the number after which the caller reads; -1 for none
	out := msgs[:0:0]
	for _, m := range msgs {
		f, ok := from[m.ConversationID]
		if !ok {
			var err error
			if f, err = s.readsFrom(ctx, w, m.ConversationID); err != nil {
				return nil, err
			}
			from[m.ConversationID] = f
		}
		if f >= 0 && m.Seq > f {
			out = append(out, m)
		}
	}
	return out, nil
}

// readsFrom is the number after which the caller reads the conversation id, or -1 when they may not read it.
func (s *Service) readsFrom(ctx context.Context, w who, id string) (int64, error) {
	conv, err := s.Repo.Conversation(ctx, id, false)
	switch {
	case errors.Is(err, ErrNotFound):
		return -1, nil
	case err != nil:
		return 0, err
	}
	parts, err := s.Repo.Participants(ctx, id)
	if err != nil {
		return 0, err
	}
	part, in := participantOf(parts, w)
	state, err := s.Consents.State(ctx, conv.PatientID, conv.OrganizationID)
	if err != nil {
		return 0, err
	}
	if !access.ReadsChat(w.viewer, access.Facts{Patient: conv.PatientID, Clinic: conv.OrganizationID, Consent: state}, in) {
		return -1, nil
	}
	return part.HistoryFromSeq, nil
}

// results are the found messages with their terms, each with its conversation.
func (s *Service) results(ctx context.Context, c Caller, msgs []Message) []SearchResult {
	s.withTerms(ctx, c, msgs)
	out := make([]SearchResult, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, SearchResult{ConversationID: m.ConversationID, Message: m})
	}
	return out
}

// searchQuery is the text search query of q: its words (runs of letters, digits and combining marks; anything else
// separates them, so nothing of the query's syntax reaches to_tsquery), those of two characters or more, the first
// maxQueryTerm, each as a prefix, all required. ErrInvalid for none, or for a q over MaxQuery characters.
func searchQuery(q string) (string, error) {
	if utf8.RuneCountInString(q) > MaxQuery || !utf8.ValidString(q) {
		return "", invalid("q")
	}
	var terms []string
	for _, w := range strings.FieldsFunc(q, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) }) {
		if utf8.RuneCountInString(w) >= 2 && len(terms) < maxQueryTerm {
			terms = append(terms, "'"+w+"':*")
		}
	}
	if len(terms) == 0 {
		return "", invalid("q")
	}
	return strings.Join(terms, " & "), nil
}
