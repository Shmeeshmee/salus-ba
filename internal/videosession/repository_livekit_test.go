package videosession_test

import (
	"context"
	"crypto/rand"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/Shmeeshmee/salus/internal/videosession"
)

// The compose LiveKit (deployments/livekit.yaml), reached from the tools container. Like the databases, the
// tests fail without it; they never skip.
type livekitServer struct {
	url, key, secret string
}

func newLiveKitServer(t testing.TB) livekitServer {
	t.Helper()
	s := livekitServer{url: os.Getenv("TEST_LIVEKIT_URL"), key: os.Getenv("TEST_LIVEKIT_API_KEY"), secret: os.Getenv("TEST_LIVEKIT_API_SECRET")}
	if s.url == "" || s.key == "" || s.secret == "" {
		t.Fatal("TEST_LIVEKIT_URL, TEST_LIVEKIT_API_KEY and TEST_LIVEKIT_API_SECRET are not set; run the tests with `make test`, which points them at the compose LiveKit")
	}
	return s
}

func (s livekitServer) provider() videosession.Provider {
	return videosession.NewLiveKitProvider(s.url, s.key, s.secret)
}

// room creates a room with a random name for the test and deletes it afterwards.
func (s livekitServer) room(t *testing.T) string {
	t.Helper()
	name := "r_test_" + strings.ToLower(rand.Text())
	p := s.provider()
	if err := p.CreateRoom(t.Context(), videosession.Room{Name: name, MaxParticipants: 2, EmptyTimeout: time.Hour}); err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	t.Cleanup(func() { _ = p.DeleteRoom(context.Background(), name) })
	return name
}

// listRoom reads the room back from LiveKit's server API.
func (s livekitServer) listRoom(t *testing.T, name string) *livekit.Room {
	t.Helper()
	res, err := lksdk.NewRoomServiceClient(s.url, s.key, s.secret).ListRooms(t.Context(), &livekit.ListRoomsRequest{Names: []string{name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rooms) != 1 {
		return nil
	}
	return res.Rooms[0]
}

// connect joins the room with token as a Go SDK participant that publishes and subscribes to nothing, and
// reports its disconnection reason on the channel.
func (s livekitServer) connect(t *testing.T, token string) (*lksdk.Room, <-chan lksdk.DisconnectionReason) {
	t.Helper()
	gone := make(chan lksdk.DisconnectionReason, 1)
	cb := lksdk.NewRoomCallback()
	cb.OnDisconnectedWithReason = func(r lksdk.DisconnectionReason) { gone <- r }
	room, err := lksdk.ConnectToRoomWithToken(strings.Replace(s.url, "http", "ws", 1), token, cb, lksdk.WithAutoSubscribe(false))
	if err != nil {
		return nil, gone
	}
	t.Cleanup(room.Disconnect)
	return room, gone
}

// The token grants joining the one room as the identity for 5 minutes, publishing the camera and the
// microphone and subscribing, and nothing else: no data, no name, no metadata, no attributes, no room
// administration or recording.
func TestToken_GrantsOnlyTheOneRoomWithCameraAndMicrophone(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	token, err := lk.provider().Token(videosession.Grant{Room: "r_grant", Identity: "p_grant", ValidFor: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	v, err := auth.ParseAPIToken(token)
	if err != nil {
		t.Fatal(err)
	}
	claims, grants, err := v.Verify(lk.secret)
	if err != nil || v.APIKey() != lk.key {
		t.Fatalf("Verify() = %v with key %q; want a token signed with the dev pair", err, v.APIKey())
	}
	if ttl := claims.ExpiresAt.Sub(claims.NotBefore.Time); ttl != 5*time.Minute {
		t.Errorf("valid for %v, want 5 minutes", ttl)
	}
	if grants.Identity != "p_grant" || grants.Name != "" || grants.Metadata != "" || len(grants.Attributes) != 0 || grants.Kind != "" ||
		grants.RoomConfig != nil || grants.RoomPreset != "" || grants.SIP != nil || grants.Agent != nil {
		t.Errorf("claims %+v; want the identity only: no name, metadata, attributes, kind or room configuration", grants)
	}
	g := grants.Video
	if g == nil || !g.RoomJoin || g.Room != "r_grant" || g.RoomCreate || g.RoomList || g.RoomRecord || g.RoomAdmin || g.Recorder ||
		g.Hidden || g.Agent || g.IngressAdmin || g.DestinationRoom != "" {
		t.Fatalf("video grant %+v; want roomJoin for r_grant only", g)
	}
	if !g.GetCanSubscribe() || g.GetCanPublishData() || g.GetCanUpdateOwnMetadata() || g.GetCanSubscribeMetrics() || g.GetCanManageAgentSession() {
		t.Errorf("video grant %+v; want subscribing, and no data, metadata, metrics or agent sessions", g)
	}
	if got := g.GetCanPublishSources(); !slices.Equal(got, []livekit.TrackSource{livekit.TrackSource_CAMERA, livekit.TrackSource_MICROPHONE}) {
		t.Errorf("publish sources %v, want the camera and the microphone only", got)
	}
}

// The room holds two participants at most, has no metadata, and LiveKit keeps it while it is empty, and
// after the last participant left, for the empty timeout. Creating it again changes nothing.
func TestCreateRoom_CreatesARoomForTwoWithoutMetadata(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	if err := lk.provider().CreateRoom(t.Context(), videosession.Room{Name: name, MaxParticipants: 2, EmptyTimeout: time.Hour}); err != nil {
		t.Fatalf("CreateRoom() again: %v", err)
	}
	r := lk.listRoom(t, name)
	if r == nil || r.MaxParticipants != 2 || r.EmptyTimeout != 3600 || r.DepartureTimeout != 3600 || r.Metadata != "" {
		t.Errorf("room %+v; want two participants at most, an hour's empty and departure timeouts, no metadata", r)
	}
}

// A token alone creates no room (auto_create is off), and a token for one room joins no other.
func TestToken_JoinsNoRoomThatWasNotCreated(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	token, err := lk.provider().Token(videosession.Grant{Room: name + "_absent", Identity: "p_absent", ValidFor: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if room, _ := lk.connect(t, token); room != nil {
		t.Error("joined a room that was never created")
	}
	if r := lk.listRoom(t, name+"_absent"); r != nil {
		t.Errorf("a token created the room %+v", r)
	}
}

// A participant joins with the provider's token; a second connection with the same identity replaces the
// first, so the room never holds more than the two identities. RemoveParticipant disconnects it.
func TestToken_JoinsAndASecondConnectionReplacesTheFirst(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	p := lk.provider()
	token, err := p.Token(videosession.Grant{Room: name, Identity: "p_same", ValidFor: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	first, firstGone := lk.connect(t, token)
	if first == nil {
		t.Fatal("the participant did not join with the provider's token")
	}
	if second, _ := lk.connect(t, token); second == nil {
		t.Fatal("the second connection did not join")
	}
	select {
	case reason := <-firstGone:
		if reason != lksdk.DuplicateIdentity {
			t.Errorf("the first connection left for %q, want %q", reason, lksdk.DuplicateIdentity)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the first connection was not replaced")
	}
	if got, err := p.Participants(t.Context(), name); err != nil || !slices.Equal(got, []string{"p_same"}) {
		t.Errorf("Participants() = %v, %v; want the one identity", got, err)
	}

	if err := p.RemoveParticipant(t.Context(), name, "p_same"); err != nil {
		t.Fatalf("RemoveParticipant() error = %v", err)
	}
	waitFor(t, "the participant to be removed", func() bool {
		got, err := p.Participants(t.Context(), name)
		return err == nil && len(got) == 0
	})
}

// Removing someone who is not there, deleting a room twice, and asking who is in a room that does not exist
// change nothing and are no error.
func TestProvider_IsIdempotentOnWhatIsGone(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	p := lk.provider()
	if err := p.RemoveParticipant(t.Context(), name, "p_nobody"); err != nil {
		t.Errorf("RemoveParticipant() of nobody = %v", err)
	}
	for range 2 {
		if err := p.DeleteRoom(t.Context(), name); err != nil {
			t.Errorf("DeleteRoom() = %v", err)
		}
	}
	if r := lk.listRoom(t, name); r != nil {
		t.Errorf("room %+v is still there", r)
	}
	if got, err := p.Participants(t.Context(), name); err != nil || len(got) != 0 {
		t.Errorf("Participants() of a deleted room = %v, %v; want none", got, err)
	}
	if err := p.RemoveParticipant(t.Context(), name, "p_nobody"); err != nil {
		t.Errorf("RemoveParticipant() in a deleted room = %v", err)
	}
}

// A refused call fails without the room or the identity in its text.
func TestProvider_ErrorsHoldNoNames(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	p := videosession.NewLiveKitProvider(lk.url, lk.key, "a-wrong-synthetic-secret-of-enough-length")
	const room, identity = "r_secretroomname", "p_secretidentity"
	errs := []error{
		p.CreateRoom(t.Context(), videosession.Room{Name: room, MaxParticipants: 2, EmptyTimeout: time.Hour}),
		p.RemoveParticipant(t.Context(), room, identity),
		p.DeleteRoom(t.Context(), room),
	}
	_, err := p.Participants(t.Context(), room)
	errs = append(errs, err)
	for i, err := range errs {
		if err == nil {
			t.Errorf("call %d with a wrong secret: nil, want an error", i)
			continue
		}
		if strings.Contains(err.Error(), room) || strings.Contains(err.Error(), identity) {
			t.Errorf("call %d: error %q holds the room or the identity", i, err)
		}
	}
}

// waitFor waits for cond, polling, and fails after 10 seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	timeout := time.After(10 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for !cond() {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for %s", what)
		case <-tick.C:
		}
	}
}
