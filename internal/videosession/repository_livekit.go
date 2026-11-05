package videosession

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
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
// microphone, subscribing, no data, no own metadata, and no name, metadata or attributes.
func (p *livekitProvider) Token(g Grant) (string, error) {
	grant := &auth.VideoGrant{RoomJoin: true, Room: g.Room}
	grant.SetCanSubscribe(true)
	grant.SetCanPublish(true)
	grant.SetCanPublishSources(publishSources)
	grant.SetCanPublishData(false)
	grant.SetCanUpdateOwnMetadata(false)
	token, err := auth.NewAccessToken(p.key, p.secret).SetIdentity(g.Identity).SetValidFor(g.ValidFor).SetVideoGrant(grant).ToJWT()
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

func (p *livekitProvider) Participants(ctx context.Context, room string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := p.rooms.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room})
	if err = gone(err); err != nil || res == nil {
		return nil, failed("participants", err)
	}
	out := make([]string, 0, len(res.Participants))
	for _, pi := range res.Participants {
		out = append(out, pi.Identity)
	}
	return out, nil
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
