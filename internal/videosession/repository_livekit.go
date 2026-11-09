package videosession

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	protologger "github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/utils/protojson"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/tphakala/go-opus/opus"
	"github.com/twitchtv/twirp"
)

// callTimeout bounds every call to LiveKit's server API.
const callTimeout = 10 * time.Second

// The only sources a participant may publish (SPEC.md §2 "Token grant": no screen share).
var publishSources = []livekit.TrackSource{livekit.TrackSource_CAMERA, livekit.TrackSource_MICROPHONE}

// The only file that knows LiveKit (depguard): the official server SDK as the protocol client
// (docs/consultation/SPEC.md §3). Nothing of ours but the random room and identity reaches LiveKit.
type livekitProvider struct {
	rooms       *lksdk.RoomServiceClient
	key, secret string
}

// NewLiveKitProvider returns the Provider on LiveKit's server API at url, with the API key pair.
func NewLiveKitProvider(url, key, secret string) Provider {
	return &livekitProvider{rooms: lksdk.NewRoomServiceClient(url, key, secret), key: key, secret: secret}
}

// CreateRoom creates the room with no metadata. The departure timeout is the empty timeout too, so a room
// whose participants dropped is kept while they reconnect.
func (p *livekitProvider) CreateRoom(ctx context.Context, r Room) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	timeout := uint32(r.EmptyTimeout / time.Second)
	_, err := p.rooms.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name: r.Name, MaxParticipants: r.MaxParticipants, EmptyTimeout: timeout, DepartureTimeout: timeout,
	})
	return failed("create room", err)
}

// Token signs a token that joins g.Room as g.Identity for g.ValidFor: publishing the camera and the
// microphone, subscribing, no data, no own metadata, and no name, metadata or attributes. An agent's token
// has the kind agent and subscribes only; it is not hidden, so the clients see it is there.
func (p *livekitProvider) Token(g Grant) (string, error) {
	grant := &auth.VideoGrant{RoomJoin: true, Room: g.Room}
	grant.SetCanSubscribe(true)
	grant.SetCanPublish(!g.Agent)
	if !g.Agent {
		grant.SetCanPublishSources(publishSources)
	}
	grant.SetCanPublishData(false)
	grant.SetCanUpdateOwnMetadata(false)
	at := auth.NewAccessToken(p.key, p.secret).SetIdentity(g.Identity).SetValidFor(g.ValidFor).SetVideoGrant(grant)
	if g.Agent {
		at.SetKind(livekit.ParticipantInfo_AGENT)
	}
	token, err := at.ToJWT()
	if err != nil {
		return "", errors.New("videosession: livekit token: signing failed")
	}
	return token, nil
}

func (p *livekitProvider) RemoveParticipant(ctx context.Context, room, identity string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	_, err := p.rooms.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: room, Identity: identity})
	return failed("remove participant", gone(err))
}

func (p *livekitProvider) DeleteRoom(ctx context.Context, room string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	_, err := p.rooms.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: room})
	return failed("delete room", gone(err))
}

// Participants returns the people in the room: an agent is not listed.
func (p *livekitProvider) Participants(ctx context.Context, room string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := p.rooms.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room})
	if err = gone(err); err != nil || res == nil {
		return nil, failed("participants", err)
	}
	out := make([]string, 0, len(res.Participants))
	for _, pi := range res.Participants {
		if pi.GetKind() != livekit.ParticipantInfo_AGENT {
			out = append(out, pi.Identity)
		}
	}
	return out, nil
}

// ParseWebhook checks the authorization as LiveKit's webhook receiver does (its webhook.Receive, which
// takes an *http.Request): a JWT issued by our API key, signed with our secret, unexpired, whose sha256 claim
// is the body's; only then is the body read.
func (p *livekitProvider) ParseWebhook(body []byte, authorization string) (Event, error) {
	v, err := auth.ParseAPIToken(authorization)
	if err != nil || v.APIKey() != p.key {
		return Event{}, ErrInvalidSignature
	}
	_, claims, err := v.Verify(p.secret)
	if err != nil {
		return Event{}, ErrInvalidSignature
	}
	sum := sha256.Sum256(body)
	if subtle.ConstantTimeCompare([]byte(claims.Sha256), []byte(base64.StdEncoding.EncodeToString(sum[:]))) != 1 {
		return Event{}, ErrInvalidSignature
	}
	var e livekit.WebhookEvent
	if err := protojson.Unmarshal(body, &e); err != nil {
		return Event{}, errors.New("videosession: livekit webhook: the signed body is not an event")
	}
	return toEvent(&e), nil
}

// toEvent is the domain's view of e: the kinds the consultation acts on, the room, the participant, and the
// time (the participant's join for a join, in milliseconds when LiveKit gives them). An agent's events are
// EventOther: the agent is no participant of the consultation.
func toEvent(e *livekit.WebhookEvent) Event {
	out := Event{Kind: EventOther, Room: e.GetRoom().GetName(), Identity: e.GetParticipant().GetIdentity(), At: time.Unix(e.GetCreatedAt(), 0).UTC()}
	if e.GetParticipant().GetKind() == livekit.ParticipantInfo_AGENT {
		return out
	}
	switch e.GetEvent() {
	case "participant_joined":
		out.Kind = EventParticipantJoined
		if ms := e.GetParticipant().GetJoinedAtMs(); ms > 0 {
			out.At = time.UnixMilli(ms).UTC()
		} else if sec := e.GetParticipant().GetJoinedAt(); sec > 0 {
			out.At = time.Unix(sec, 0).UTC()
		}
	case "participant_left":
		out.Kind = EventParticipantLeft
	case "room_finished":
		out.Kind = EventRoomFinished
	}
	return out
}

// gone is nil for LiveKit's answer that the room or the participant does not exist, and err otherwise.
func gone(err error) error {
	var te twirp.Error
	if errors.As(err, &te) && te.Code() == twirp.NotFound {
		return nil
	}
	return err
}

// failed is the error of the call op: LiveKit's error code only, as its message may name the room or the
// identity; other errors (the transport's) as they are.
func failed(op string, err error) error {
	var te twirp.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &te):
		return fmt.Errorf("videosession: livekit %s: %s", op, te.Code())
	}
	return fmt.Errorf("videosession: livekit %s: %w", op, err)
}

// frameBuffer is how many decoded frames (20 ms each) a track keeps for a slow reader before it drops them.
const frameBuffer = 50

// pcmRate is the rate the agent's audio is decoded to: 16 kHz, what speech-to-text engines take.
const pcmRate = 16000

type livekitListener struct{}

// NewLiveKitListener returns the Listener on LiveKit's Go SDK: the agent joins as a participant that subscribes
// to every audio track and decodes it in pure Go (Opus through github.com/tphakala/go-opus, a port of libopus that
// decodes every Opus mode; G.711 µ-law by its table), with no cgo.
//
// The SDK and its WebRTC stack log the room, the identities and the addresses through its package logger by
// default, which a connection's own logger does not fully replace: NewLiveKitListener silences that logger for
// the process. Only cmd/agent builds a Listener.
func NewLiveKitListener() Listener {
	lksdk.SetLogger(protologger.GetDiscardLogger())
	return livekitListener{}
}

// Listen connects to the room at url (ws:// or http://) with the agent's token. It does not wait on ctx: the
// SDK's connection has its own timeout.
func (livekitListener) Listen(_ context.Context, url, token string) (Listening, error) {
	l := &listening{tracks: make(chan RemoteAudio, 4)}
	cb := lksdk.NewRoomCallback()
	cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, _ *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		dec, err := newDecoder(track.Codec().MimeType)
		if err != nil {
			return
		}
		frames := make(chan []byte, frameBuffer)
		if !l.hand(RemoteAudio{Identity: rp.Identity(), Frames: frames}) {
			dec.close()
			return
		}
		go decodeTrack(track, dec, frames)
	}
	cb.OnDisconnected = l.end
	// The SDK logs the room, the identities and the addresses to the standard logger by default: never.
	room, err := lksdk.ConnectToRoomWithToken(strings.Replace(url, "http", "ws", 1), token, cb,
		lksdk.WithAutoSubscribe(true), lksdk.WithLogger(protologger.GetDiscardLogger()))
	if err != nil {
		return Listening{}, failed("listen", err)
	}
	var once sync.Once
	return Listening{Tracks: l.tracks, Leave: func() { once.Do(func() { room.Disconnect(); l.end() }) }}, nil
}

// listening hands the room's tracks over until the agent is out of it.
type listening struct {
	tracks chan RemoteAudio
	mu     sync.Mutex
	ended  bool
}

// hand hands a track over unless the agent is out of the room.
func (l *listening) hand(a RemoteAudio) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return false
	}
	l.tracks <- a
	return true
}

// end closes the tracks once: the agent is out of the room.
func (l *listening) end() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.ended {
		l.ended = true
		close(l.tracks)
	}
}

// decodeTrack decodes the track's RTP into frames until it ends, dropping frames nobody reads in time.
func decodeTrack(track *webrtc.TrackRemote, dec decoder, frames chan<- []byte) {
	defer close(frames)
	defer dec.close()
	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			return
		}
		pcm := dec.decode(pkt.Payload)
		if len(pcm) == 0 {
			continue
		}
		select {
		case frames <- pcm:
		default:
		}
	}
}

// decoder turns one track's RTP payloads into 16 kHz 16-bit little-endian mono PCM.
type decoder interface {
	decode(payload []byte) []byte
	close()
}

// newDecoder is the decoder of the codec: Opus or G.711 µ-law; other codecs are not heard.
func newDecoder(mime string) (decoder, error) {
	switch {
	case strings.EqualFold(mime, webrtc.MimeTypeOpus):
		d, err := opus.NewDecoder(pcmRate, 1)
		if err != nil {
			return nil, errors.New("videosession: opus decoder")
		}
		return &opusDecoder{d: d, pcm: make([]int16, pcmRate*120/1000)}, nil
	case strings.EqualFold(mime, webrtc.MimeTypePCMU):
		return &pcmuDecoder{}, nil
	}
	return nil, errors.New("videosession: an audio codec the agent does not decode")
}

// opusDecoder decodes Opus at 16 kHz, mono (a stereo stream is mixed down).
type opusDecoder struct {
	d   *opus.Decoder
	pcm []int16 // up to 120 ms, Opus's longest frame
}

func (o *opusDecoder) decode(payload []byte) []byte {
	n, err := o.d.Decode(payload, o.pcm)
	if err != nil {
		return nil
	}
	return le16(o.pcm[:n])
}

func (*opusDecoder) close() {}

// pcmuDecoder decodes G.711 µ-law at 8 kHz and doubles its rate to 16 kHz, each new sample the mean of its
// neighbours.
type pcmuDecoder struct{ prev int16 }

func (p *pcmuDecoder) decode(payload []byte) []byte {
	out := make([]int16, 0, 2*len(payload))
	for _, c := range payload {
		s := ulaw(c)
		out = append(out, int16((int32(p.prev)+int32(s))/2), s)
		p.prev = s
	}
	return le16(out)
}

func (*pcmuDecoder) close() {}

// ulaw is the sample of a G.711 µ-law code.
func ulaw(c byte) int16 {
	c = ^c
	t := (int32(c&0x0f)<<3 + 0x84) << ((c & 0x70) >> 4)
	if c&0x80 != 0 {
		return int16(0x84 - t)
	}
	return int16(t - 0x84)
}

// le16 is samples as 16-bit little-endian bytes.
func le16(samples []int16) []byte {
	b := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[2*i:], uint16(s))
	}
	return b
}
