package transcript

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// publish puts each result of the recognition on the encounter's stream until it ends: nil at its end (io.EOF),
// else why it stopped. start is when its audio started: a result's offsets are from there. Partial results revise
// one segment until its final one; the next result starts a new segment. A result without words is skipped; one
// the stream does not take is logged, by its field, and skipped.
func (s *Service) publish(ctx context.Context, c call, sp speaker, start time.Time, stream stt.Stream) error {
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
		b, err := wire.Encode(MessageType, MessageVersion, e)
		if err == nil {
			err = s.Publisher.Publish(ctx, c.EncounterID, b)
		}
		if err != nil && ctx.Err() == nil {
			s.Log.WarnContext(ctx, "transcript publish failed", "session_id", c.sessionID, "error", err.Error())
		}
	}
}
