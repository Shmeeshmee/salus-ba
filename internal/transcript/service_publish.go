package transcript

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// publish puts each result of the recognition on the encounter's stream until it ends: nil at its end (io.EOF),
// else why it stopped. start is when its audio started: a result's offsets are from there. Partial results revise
// one segment until its final one; the next result starts a new segment. A result without words is skipped; one
// the stream does not take is logged, by its field, and skipped. A final result is translated in the background
// (translate) and published once, with its translation or flagged; publish returns once every one of them is,
// or was dropped with the call.
func (s *Live) publish(ctx context.Context, c call, sp speaker, start time.Time, stream stt.Stream) error {
	var pending sync.WaitGroup
	defer pending.Wait()
	segment := newID()
	for {
		res, err := stream.Recv()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		case strings.TrimSpace(res.Text) == "":
			continue
		}
		e := Event{
			EncounterID: c.EncounterID, SegmentID: segment, Speaker: sp.Speaker, Lang: sp.locale, Original: res.Text,
			IsFinal: res.Final, TsStart: start.Add(res.Start), TsEnd: start.Add(max(res.End, res.Start)),
		}
		if res.Lang != "" && validLang(res.Lang) {
			e.Lang = res.Lang
		}
		if res.Final {
			segment = newID()
		}
		if err := e.Validate(); err != nil {
			s.Log.WarnContext(ctx, "transcript event refused", "session_id", c.sessionID, "error", err.Error())
			continue
		}
		if !e.IsFinal || !c.translated(sp.Speaker) {
			if e.IsFinal {
				s.tag(ctx, c, &e)
			}
			s.emit(ctx, c, e)
			continue
		}
		pending.Go(func() {
			s.translate(ctx, c, &e)
			s.tag(ctx, c, &e)
			if ctx.Err() == nil { // the call is over: nothing said is published after a removal
				s.emit(ctx, c, e)
			}
		})
	}
}

// emit puts the event on the encounter's stream; a failure is logged, without the event.
func (s *Live) emit(ctx context.Context, c call, e Event) {
	b, err := wire.Encode(MessageType, MessageVersion, e)
	if err == nil {
		err = s.Publisher.Publish(ctx, c.EncounterID, b)
	}
	if err != nil && ctx.Err() == nil {
		s.Log.WarnContext(ctx, "transcript publish failed", "session_id", c.sessionID, "error", err.Error())
	}
}
