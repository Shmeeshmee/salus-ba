package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

// ErrHubClosed is returned by Subscribe once the Hub is closed.
var ErrHubClosed = errors.New("redis: hub closed")

// Hub fans Redis pub/sub messages out to the subscribers in this process over one connection, however many
// subscribers there are: a channel is subscribed in Redis while it has a subscriber here. Pub/sub is lossy by
// design (a reconnect, or a subscriber that does not keep up, drops messages), so a Hub suits only live updates
// whose record is elsewhere: each subscriber gets a buffered channel, and a message for a full one is dropped
// rather than holding up the others. No Redis call is made under the lock the fan-out takes: a slow SUBSCRIBE (a
// reconnect) delays other subscriptions being set up, never the delivery of messages.
type Hub struct {
	rdb    *goredis.Client
	ps     *goredis.PubSub
	subs   map[string]map[*hubSub]struct{} // the subscribers here, by channel; under mu
	active map[string]bool                 // the channels subscribed in Redis; under redisMu
	done   chan struct{}
	buffer int
	mu     sync.Mutex // subs and closed: the fan-out's lock
	// redisMu serialises the SUBSCRIBE and UNSUBSCRIBE commands, so the last one sent for a channel is the one its
	// subscribers want (reconcile).
	redisMu sync.Mutex
	closed  bool
}

type hubSub struct {
	ch   chan []byte
	once sync.Once
}

// NewHub returns a Hub on rdb that gives each subscriber a buffer of buffer messages. It holds one connection of
// rdb until Close; ctx bounds the subscription connection's commands.
func NewHub(ctx context.Context, rdb *goredis.Client, buffer int) *Hub {
	h := &Hub{
		rdb: rdb, ps: rdb.Subscribe(ctx), subs: map[string]map[*hubSub]struct{}{}, active: map[string]bool{},
		done: make(chan struct{}), buffer: max(buffer, 1),
	}
	go h.run()
	return h
}

// run hands every message to the subscribers of its channel until the subscription closes.
func (h *Hub) run() {
	defer close(h.done)
	for m := range h.ps.Channel(goredis.WithChannelSize(1024)) {
		payload := []byte(m.Payload)
		h.mu.Lock()
		for s := range h.subs[m.Channel] {
			select {
			case s.ch <- payload:
			default: // the subscriber is behind: it fills the gap from the record
			}
		}
		h.mu.Unlock()
	}
}

// Subscribe returns the messages published on channel from now on, and the function that ends the subscription
// and closes them (safe to call more than once). The Redis subscription is sent before Subscribe returns; when it
// cannot be, the subscription is undone and the error returned.
func (h *Hub) Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	s := &hubSub{ch: make(chan []byte, h.buffer)}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, nil, ErrHubClosed
	}
	if h.subs[channel] == nil {
		h.subs[channel] = map[*hubSub]struct{}{}
	}
	h.subs[channel][s] = struct{}{}
	h.mu.Unlock()
	stop := func() { h.unsubscribe(channel, s) }
	if err := h.reconcile(ctx, channel); err != nil {
		stop()
		return nil, nil, fmt.Errorf("redis: subscribe: %w", err)
	}
	return s.ch, stop, nil
}

// unsubscribe removes s and, with the channel's last subscriber here, the Redis subscription.
func (h *Hub) unsubscribe(channel string, s *hubSub) {
	s.once.Do(func() {
		h.mu.Lock()
		delete(h.subs[channel], s)
		close(s.ch)
		if len(h.subs[channel]) == 0 {
			delete(h.subs, channel)
		}
		h.mu.Unlock()
		_ = h.reconcile(context.Background(), channel) // a failed UNSUBSCRIBE leaves messages nobody here takes
	})
}

// reconcile sends SUBSCRIBE or UNSUBSCRIBE when the channel's state in Redis is not what its subscribers here
// want now. Serialised by redisMu and decided under it, so whatever order two subscribers come and go in, the last
// command sent for a channel matches its subscribers.
func (h *Hub) reconcile(ctx context.Context, channel string) error {
	h.redisMu.Lock()
	defer h.redisMu.Unlock()
	h.mu.Lock()
	want, closed := len(h.subs[channel]) > 0, h.closed
	h.mu.Unlock()
	switch {
	case closed || want == h.active[channel]:
		return nil
	case want:
		if err := h.ps.Subscribe(ctx, channel); err != nil {
			_ = h.ps.Unsubscribe(context.WithoutCancel(ctx), channel) // go-redis keeps it for its reconnects otherwise
			return err
		}
		h.active[channel] = true
	default:
		delete(h.active, channel)
		return h.ps.Unsubscribe(ctx, channel)
	}
	return nil
}

// Publish publishes msg on channel, to every Hub's subscribers of it, in this process and the others.
func (h *Hub) Publish(ctx context.Context, channel string, msg []byte) error {
	if err := h.rdb.Publish(ctx, channel, msg).Err(); err != nil {
		return fmt.Errorf("redis: publish: %w", err)
	}
	return nil
}

// Close ends every subscription and releases the connection.
func (h *Hub) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	h.mu.Unlock()
	err := h.ps.Close()
	<-h.done
	return err
}
