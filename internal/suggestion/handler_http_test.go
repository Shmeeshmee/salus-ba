package suggestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// stubStatuses records the status changes asked for and answers with err.
type stubStatuses struct {
	err error
	got []string
}

func (s *stubStatuses) SetStatus(_ context.Context, c appointment.Caller, encounterID, id string, status suggestion.Status) (suggestion.Event, error) {
	s.got = append(s.got, c.AccountID+" "+encounterID+" "+id+" "+string(status))
	return suggestion.Event{}, s.err
}

func command(t *testing.T, typ string, version int, payload string) wire.Envelope {
	t.Helper()
	return wire.Envelope{Type: typ, Version: version, Payload: json.RawMessage(payload)}
}

// A suggestion_status message sets the status for the caller on the stream's encounter and is answered with
// nothing: the suggestion with its status comes on the stream.
func TestCommand_SetsTheStatusAndAnswersNothing(t *testing.T) {
	svc := &stubStatuses{}
	h := suggestion.NewCommandHandler(svc, slog.New(slog.DiscardHandler))
	got := h.Command(t.Context(), pract, encounterID, command(t, "suggestion_status", 1, `{"id":"`+suggestionID+`","status":"dismissed"}`))
	if got != nil || len(svc.got) != 1 || svc.got[0] != pract.AccountID+" "+encounterID+" "+suggestionID+" dismissed" {
		t.Errorf("answer %s, calls %q", got, svc.got)
	}
}

// A refusal or failure is answered with a command_error naming the suggestion and the code; a 500's text goes to
// the log only.
func TestCommand_AnswersAFailureWithItsCode(t *testing.T) {
	for _, tt := range []struct {
		err  error
		code string
	}{
		{suggestion.ErrInvalid, "invalid"},
		{suggestion.ErrNotFound, "not_found"},
		{suggestion.ErrConsentRequired, "consent_required"},
		{suggestion.ErrForbidden, "forbidden"},
		{errors.New("the database is down at 10.0.0.1"), "internal"},
	} {
		var log bytes.Buffer
		h := suggestion.NewCommandHandler(&stubStatuses{err: tt.err}, slog.New(slog.NewJSONHandler(&log, nil)))
		got := h.Command(t.Context(), pract, encounterID, command(t, "suggestion_status", 1, `{"id":"`+suggestionID+`","status":"pinned"}`))
		want := `{"type":"command_error","version":1,"payload":{"command":"suggestion_status","id":"` + suggestionID + `","error":"` + tt.code + `"}}`
		if string(got) != want {
			t.Errorf("%s: answer %s, want %s", tt.code, got, want)
		}
		if tt.code == "internal" && !strings.Contains(log.String(), "10.0.0.1") {
			t.Errorf("internal: the log does not hold the error:\n%s", log.String())
		}
	}
}

// Another type or version, or a payload that is not the status change, is refused before anything is called.
func TestCommand_RefusesWhatIsNotAStatusChange(t *testing.T) {
	for name, tt := range map[string]struct {
		env  wire.Envelope
		want string
	}{
		"another type":    {command(t, "draft_sign", 1, `{}`), `{"command":"draft_sign","error":"unknown_command"}`},
		"another version": {command(t, "suggestion_status", 2, `{}`), `{"command":"suggestion_status","error":"unknown_command"}`},
		"not an object":   {command(t, "suggestion_status", 1, `"dismissed"`), `{"command":"suggestion_status","error":"invalid"}`},
	} {
		svc := &stubStatuses{}
		got := suggestion.NewCommandHandler(svc, slog.New(slog.DiscardHandler)).Command(t.Context(), pract, encounterID, tt.env)
		want := `{"type":"command_error","version":1,"payload":` + tt.want + `}`
		if string(got) != want || len(svc.got) != 0 {
			t.Errorf("%s: answer %s, calls %q; want %s and none", name, got, svc.got, want)
		}
	}
}
