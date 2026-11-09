package redis

import (
	"context"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// groupPage is the most entries one read or one claim hands to the handler.
const groupPage = 100

// Group is one member of a Redis Streams consumer group over every stream whose key matches Pattern, found as
// streams appear: each entry is handed to one member of the group, at least once, and acknowledged when its
// handler succeeds. An entry whose handler fails, or whose member died before it acknowledged it, stays pending
// and is claimed and handed over again once it has been pending MinIdle; a handler must therefore be
// idempotent. A stream found late is read from its first entry.
type Group struct {
	Client *goredis.Client
	// OnError is told of a failed Redis call, after which Run waits Block and goes on; nil ignores it.
	OnError  func(error)
	Pattern  string // SCAN MATCH pattern of the stream keys
	Name     string // the consumer group
	Consumer string // this member's name, unique among the group's members
	// Block is how long a read waits for new entries, and how often Run looks for new streams and claims
	// entries pending past MinIdle.
	Block   time.Duration
	MinIdle time.Duration
}

// Handler handles one entry of a stream; nil acknowledges it, an error leaves it pending to be handed over
// again.
type Handler func(ctx context.Context, stream, id string, values map[string]any) error

// Run consumes the streams until ctx ends. A Redis error never ends it: it is reported to OnError and retried
// after Block.
func (g *Group) Run(ctx context.Context, handle Handler) {
	known := map[string]bool{}
	var streams []string
	var swept time.Time
	for ctx.Err() == nil {
		if time.Since(swept) >= g.Block {
			var err error
			if streams, err = g.discover(ctx, known); err != nil {
				g.fail(ctx, err)
				continue
			}
			if err := g.claim(ctx, streams, handle); err != nil {
				g.fail(ctx, err)
				continue
			}
			swept = time.Now()
		}
		if len(streams) == 0 {
			wait(ctx, g.Block)
			continue
		}
		err := g.read(ctx, streams, handle)
		switch {
		case err == nil:
		case strings.HasPrefix(err.Error(), "NOGROUP"): // a stream expired, or came back without the group
			clear(known)
			swept = time.Time{}
		default:
			g.fail(ctx, err)
		}
	}
}

// discover returns the keys of the streams matching Pattern, and creates the group on each one not in known
// (which it updates) from the stream's first entry.
func (g *Group) discover(ctx context.Context, known map[string]bool) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		page, next, err := g.Client.ScanType(ctx, cursor, g.Pattern, 1000, "stream").Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, page...)
		if cursor = next; cursor == 0 {
			break
		}
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		seen[key] = true
		if known[key] {
			continue
		}
		err := g.Client.XGroupCreate(ctx, key, g.Name, "0").Err()
		if err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
			return nil, err
		}
	}
	clear(known)
	for key := range seen {
		known[key] = true
	}
	return keys, nil
}

// claim takes over the entries of the streams pending past MinIdle, on any member, and hands them over again.
func (g *Group) claim(ctx context.Context, streams []string, handle Handler) error {
	for _, key := range streams {
		msgs, _, err := g.Client.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
			Stream: key, Group: g.Name, Consumer: g.Consumer, MinIdle: g.MinIdle, Start: "0-0", Count: groupPage,
		}).Result()
		if err != nil {
			if strings.HasPrefix(err.Error(), "NOGROUP") { // gone since discover
				continue
			}
			return err
		}
		if err := g.handle(ctx, key, msgs, handle); err != nil {
			return err
		}
	}
	return nil
}

// read hands over the streams' entries no member has had yet, waiting up to Block for one.
func (g *Group) read(ctx context.Context, streams []string, handle Handler) error {
	args := make([]string, 0, 2*len(streams))
	args = append(args, streams...)
	for range streams {
		args = append(args, ">")
	}
	res, err := g.Client.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group: g.Name, Consumer: g.Consumer, Streams: args, Count: groupPage, Block: max(g.Block, time.Millisecond),
	}).Result()
	switch {
	case err == goredis.Nil: // the wait passed
		return nil
	case err != nil:
		return err
	}
	for _, s := range res {
		if err := g.handle(ctx, s.Stream, s.Messages, handle); err != nil {
			return err
		}
	}
	return nil
}

// handle hands each message over in order and acknowledges those handled.
func (g *Group) handle(ctx context.Context, key string, msgs []goredis.XMessage, handle Handler) error {
	var done []string
	for _, m := range msgs {
		if handle(ctx, key, m.ID, m.Values) == nil {
			done = append(done, m.ID)
		}
	}
	if len(done) == 0 {
		return nil
	}
	return g.Client.XAck(ctx, key, g.Name, done...).Err()
}

// fail reports err unless ctx has ended, and waits Block.
func (g *Group) fail(ctx context.Context, err error) {
	if ctx.Err() != nil {
		return
	}
	if g.OnError != nil {
		g.OnError(err)
	}
	wait(ctx, g.Block)
}

// wait waits d, or until ctx ends.
func wait(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
