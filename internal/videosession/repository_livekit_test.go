package videosession_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	protologger "github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/utils/protojson"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/tphakala/go-opus/opus"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
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
	room, err := lksdk.ConnectToRoomWithToken(strings.Replace(s.url, "http", "ws", 1), token, cb, lksdk.WithAutoSubscribe(false),
		lksdk.WithLogger(protologger.GetDiscardLogger())) // so TestListen_LogsNothing hears only the agent
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

// webhookBody is a LiveKit webhook event as LiveKit sends it (protojson).
func webhookBody(t *testing.T, e *livekit.WebhookEvent) []byte {
	t.Helper()
	body, err := protojson.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// sign is the Authorization header LiveKit sends with body: a JWT of the key pair, valid for validFor,
// carrying the body's SHA-256.
func sign(t *testing.T, key, secret string, body []byte, validFor time.Duration) string {
	t.Helper()
	sum := sha256.Sum256(body)
	token, err := auth.NewAccessToken(key, secret).SetValidFor(validFor).SetSha256(base64.StdEncoding.EncodeToString(sum[:])).ToJWT()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// expired is sign's header for body, but expired an hour ago (past LiveKit's leeway of a minute); the SDK
// signs no expired token, so it is written here as LiveKit's HS256 JWT.
func expired(t *testing.T, key, secret string, body []byte) string {
	t.Helper()
	sum := sha256.Sum256(body)
	now := clock.System{}.Now()
	enc := base64.RawURLEncoding
	claims, err := json.Marshal(map[string]any{
		"iss": key, "nbf": now.Add(-2 * time.Hour).Unix(), "exp": now.Add(-time.Hour).Unix(),
		"sha256": base64.StdEncoding.EncodeToString(sum[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil))
}

// A webhook signed with our key pair and carrying its body's hash is accepted and read as the domain's
// event: the kind, the room, the participant's identity and when they joined (else when the event happened).
func TestParseWebhook_ReadsASignedEvent(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	p := lk.provider()
	joined := time.Date(2030, 1, 7, 8, 55, 1, 250_000_000, time.UTC)
	created := time.Date(2030, 1, 7, 8, 55, 3, 0, time.UTC)
	for _, tc := range []struct {
		event *livekit.WebhookEvent
		want  videosession.Event
	}{
		{
			event: &livekit.WebhookEvent{Event: "participant_joined", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"},
				Participant: &livekit.ParticipantInfo{Identity: "p_a", JoinedAt: joined.Unix(), JoinedAtMs: joined.UnixMilli()}},
			want: videosession.Event{Kind: videosession.EventParticipantJoined, Room: "r_a", Identity: "p_a", At: joined},
		},
		{
			event: &livekit.WebhookEvent{Event: "participant_joined", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"},
				Participant: &livekit.ParticipantInfo{Identity: "p_a", JoinedAt: joined.Unix()}},
			want: videosession.Event{Kind: videosession.EventParticipantJoined, Room: "r_a", Identity: "p_a", At: joined.Truncate(time.Second)},
		},
		{
			event: &livekit.WebhookEvent{Event: "participant_left", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"},
				Participant: &livekit.ParticipantInfo{Identity: "p_b", JoinedAtMs: joined.UnixMilli()}},
			want: videosession.Event{Kind: videosession.EventParticipantLeft, Room: "r_a", Identity: "p_b", At: created},
		},
		{
			event: &livekit.WebhookEvent{Event: "room_finished", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"}},
			want:  videosession.Event{Kind: videosession.EventRoomFinished, Room: "r_a", At: created},
		},
		{
			event: &livekit.WebhookEvent{Event: "room_started", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"}},
			want:  videosession.Event{Kind: videosession.EventOther, Room: "r_a", At: created},
		},
		{
			event: &livekit.WebhookEvent{Event: "track_published", CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"},
				Participant: &livekit.ParticipantInfo{Identity: "p_a"}, Track: &livekit.TrackInfo{Sid: "TR_x"}},
			want: videosession.Event{Kind: videosession.EventOther, Room: "r_a", Identity: "p_a", At: created},
		},
		{
			event: &livekit.WebhookEvent{Event: "egress_started", CreatedAt: created.Unix()},
			want:  videosession.Event{Kind: videosession.EventOther, At: created},
		},
	} {
		body := webhookBody(t, tc.event)
		got, err := p.ParseWebhook(body, sign(t, lk.key, lk.secret, body, 5*time.Minute))
		if err != nil || got != tc.want {
			t.Errorf("ParseWebhook(%s) = %+v, %v; want %+v", tc.event.Event, got, err, tc.want)
		}
	}
}

// A wrong key, a wrong secret, a changed body, no header, an expired token, a token without the body's hash
// and a header that is not a token are refused with ErrInvalidSignature, before the body is read.
func TestParseWebhook_RefusesWhatLiveKitDidNotSign(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	p := lk.provider()
	body := webhookBody(t, &livekit.WebhookEvent{Event: "participant_joined", Room: &livekit.Room{Name: "r_a"},
		Participant: &livekit.ParticipantInfo{Identity: "p_a"}})
	changed := webhookBody(t, &livekit.WebhookEvent{Event: "participant_joined", Room: &livekit.Room{Name: "r_b"},
		Participant: &livekit.ParticipantInfo{Identity: "p_a"}})
	noHash, err := auth.NewAccessToken(lk.key, lk.secret).SetValidFor(time.Minute).ToJWT()
	if err != nil {
		t.Fatal(err)
	}
	for name, header := range map[string]string{
		"a wrong key":     sign(t, "another-key", lk.secret, body, time.Minute),
		"a wrong secret":  sign(t, lk.key, "another-synthetic-secret-of-enough-length", body, time.Minute),
		"a changed body":  sign(t, lk.key, lk.secret, changed, time.Minute),
		"no header":       "",
		"an expired one":  expired(t, lk.key, lk.secret, body),
		"no body hash":    noHash,
		"not a token":     "Bearer not-a-token",
		"a garbled token": sign(t, lk.key, lk.secret, body, time.Minute) + "x",
	} {
		got, err := p.ParseWebhook(body, header)
		if !errors.Is(err, videosession.ErrInvalidSignature) || got != (videosession.Event{}) {
			t.Errorf("ParseWebhook() with %s = %+v, %v; want ErrInvalidSignature", name, got, err)
		}
	}
}

// A signed body that is not an event fails without its content in the error.
func TestParseWebhook_FailsOnASignedBodyThatIsNotAnEvent(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	body := []byte(`{"event": "participant_joined", "room": "r_secretroomname"`)
	_, err := lk.provider().ParseWebhook(body, sign(t, lk.key, lk.secret, body, time.Minute))
	if err == nil || errors.Is(err, videosession.ErrInvalidSignature) || strings.Contains(err.Error(), "r_secretroomname") {
		t.Errorf("ParseWebhook() = %v; want an error of its own without the body", err)
	}
}

// onLiveKit is the fixture with appointment A's session in a fresh room of the compose LiveKit, the service
// on the real provider, and the patient and the practitioner connected with its tokens; it returns their
// disconnections.
func onLiveKit(t *testing.T) (f *fixture, svc *videosession.Service, patientGone, practitionerGone <-chan lksdk.DisconnectionReason) {
	t.Helper()
	lk := newLiveKitServer(t)
	f = opened(during, start, start)
	s := f.session()
	s.Room, s.PatientIdentity, s.PractitionerIdentity = lk.room(t), "p_pat_"+strings.ToLower(rand.Text()), "p_pra_"+strings.ToLower(rand.Text())
	f.repo.sessions[apptA] = s
	p := lk.provider()
	svc = videosession.NewService(videosession.Ports{
		Repo: f.repo, Provider: p, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: during},
		Appointments: f.appts, Encounters: f.encounters, Consents: f.consents,
	})
	gone := make([]<-chan lksdk.DisconnectionReason, 2)
	for i, identity := range []string{s.PatientIdentity, s.PractitionerIdentity} {
		token, err := p.Token(videosession.Grant{Room: s.Room, Identity: identity, ValidFor: 5 * time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		var room *lksdk.Room
		if room, gone[i] = lk.connect(t, token); room == nil {
			t.Fatalf("participant %d did not join", i)
		}
	}
	waitFor(t, "both in the room", func() bool {
		in, err := p.Participants(t.Context(), s.Room)
		return err == nil && len(in) == 2
	})
	return f, svc, gone[0], gone[1]
}

// Real LiveKit: after EndAccess on a cancelled appointment both are gone (the room is deleted), the patient
// first.
func TestEndAccess_OnLiveKitDisconnectsThePatientAndDeletesTheRoom(t *testing.T) {
	t.Parallel()
	f, svc, patientGone, practitionerGone := onLiveKit(t)
	f.cancel()
	if err := svc.EndAccess(t.Context(), apptA, clinicA); err != nil {
		t.Fatalf("EndAccess() = %v", err)
	}
	for who, gone := range map[string]<-chan lksdk.DisconnectionReason{"patient": patientGone, "practitioner": practitionerGone} {
		select {
		case <-gone:
		case <-time.After(10 * time.Second):
			t.Errorf("the %s is still connected", who)
		}
	}
	if in, err := svc.Provider.Participants(t.Context(), f.session().Room); err != nil || len(in) != 0 {
		t.Errorf("Participants() = %v, %v; want none", in, err)
	}
}

// Real LiveKit: after the revocation's EndPatientAccess the patient is gone and the practitioner is still in
// the room; the removal of the patient, who was there, is audited.
func TestEndPatientAccess_OnLiveKitDisconnectsOnlyThePatient(t *testing.T) {
	t.Parallel()
	f, svc, patientGone, practitionerGone := onLiveKit(t)
	f.consents.revoked = true
	if err := svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("EndPatientAccess() = %v", err)
	}
	select {
	case reason := <-patientGone:
		if reason != lksdk.ParticipantRemoved {
			t.Errorf("the patient left for %q, want %q", reason, lksdk.ParticipantRemoved)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the patient is still connected")
	}
	in, err := svc.Provider.Participants(t.Context(), f.session().Room)
	if want := []string{f.session().PractitionerIdentity}; err != nil || !slices.Equal(in, want) {
		t.Errorf("Participants() = %v, %v; want only the practitioner %v", in, err, want)
	}
	select {
	case reason := <-practitionerGone:
		t.Errorf("the practitioner left (%q), want them still in the room", reason)
	default:
	}
	checkEvents(t, "EndPatientAccess", f.audit, removalEvent(revocationAgent))
}

// The agent's token (SPEC.md §2 "Changed for SAL-3") joins the one room as a participant of kind agent that
// subscribes and publishes nothing: no track, no data, no metadata, no name. It is not hidden, so both clients
// can show that it is there.
func TestToken_TheAgentGrantSubscribesOnly(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	token, err := lk.provider().Token(videosession.Grant{Room: "r_grant", Identity: "a_grant", ValidFor: 5 * time.Minute, Agent: true})
	if err != nil {
		t.Fatal(err)
	}
	v, err := auth.ParseAPIToken(token)
	if err != nil {
		t.Fatal(err)
	}
	claims, grants, err := v.Verify(lk.secret)
	if err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if ttl := claims.ExpiresAt.Sub(claims.NotBefore.Time); ttl != 5*time.Minute {
		t.Errorf("valid for %v, want 5 minutes", ttl)
	}
	if grants.Identity != "a_grant" || grants.Kind != "agent" || grants.Name != "" || grants.Metadata != "" || len(grants.Attributes) != 0 ||
		grants.RoomConfig != nil || grants.SIP != nil || grants.Agent != nil {
		t.Errorf("claims %+v; want the identity and the kind agent only", grants)
	}
	g := grants.Video
	if g == nil || !g.RoomJoin || g.Room != "r_grant" || g.RoomCreate || g.RoomList || g.RoomRecord || g.RoomAdmin || g.Recorder ||
		g.Hidden || g.Agent || g.IngressAdmin || g.DestinationRoom != "" {
		t.Fatalf("video grant %+v; want roomJoin for r_grant only, not hidden", g)
	}
	if !g.GetCanSubscribe() || g.GetCanPublish() || len(g.GetCanPublishSources()) != 0 || g.GetCanPublishData() || g.GetCanUpdateOwnMetadata() {
		t.Errorf("video grant %+v; want subscribing only: no publishing, no data, no metadata", g)
	}
}

// Real LiveKit: in a room for two, the agent joins beside the patient and the practitioner, and a third person
// still cannot. Participants names the people only, so the agent never keeps a room "occupied" when the window
// closes.
func TestToken_TheAgentJoinsARoomForTwoAndIsNoParticipant(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	p := lk.provider()
	join := func(identity string, agent bool) bool {
		token, err := p.Token(videosession.Grant{Room: name, Identity: identity, ValidFor: time.Minute, Agent: agent})
		if err != nil {
			t.Fatal(err)
		}
		room, _ := lk.connect(t, token)
		return room != nil
	}
	if !join("p_patient", false) || !join("p_practitioner", false) {
		t.Fatal("the two people did not join")
	}
	if !join("a_agent", true) {
		t.Fatal("the agent did not join the room for two")
	}
	if join("p_third", false) {
		t.Error("a third person joined the room for two")
	}
	waitFor(t, "the two people listed", func() bool {
		in, err := p.Participants(t.Context(), name)
		slices.Sort(in)
		return err == nil && slices.Equal(in, []string{"p_patient", "p_practitioner"})
	})
}

// An agent's join or leave is no participant's: the webhook reads it as EventOther, so it records no join and
// ends nothing.
func TestParseWebhook_ReadsAnAgentsEventsAsOther(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	p := lk.provider()
	created := time.Date(2030, 1, 7, 8, 55, 3, 0, time.UTC)
	for _, kind := range []string{"participant_joined", "participant_left"} {
		body := webhookBody(t, &livekit.WebhookEvent{Event: kind, CreatedAt: created.Unix(), Room: &livekit.Room{Name: "r_a"},
			Participant: &livekit.ParticipantInfo{Identity: "a_a", Kind: livekit.ParticipantInfo_AGENT, JoinedAtMs: created.UnixMilli()}})
		got, err := p.ParseWebhook(body, sign(t, lk.key, lk.secret, body, 5*time.Minute))
		want := videosession.Event{Kind: videosession.EventOther, Room: "r_a", Identity: "a_a", At: created}
		if err != nil || got != want {
			t.Errorf("ParseWebhook(%s of an agent) = %+v, %v; want %+v", kind, got, err, want)
		}
	}
}

// publishTone publishes a synthetic 440 Hz tone (never a recording) as the room's microphone until the test ends:
// Opus at 48 kHz, as browsers and phones send it, or G.711 µ-law at 8 kHz, as the e2e participant does.
func publishTone(t *testing.T, room *lksdk.Room, codec string) {
	t.Helper()
	capability := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2}
	if codec == webrtc.MimeTypePCMU {
		capability = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 1}
	}
	track, err := lksdk.NewLocalSampleTrack(capability)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Source: livekit.TrackSource_MICROPHONE}); err != nil {
		t.Fatalf("publish the tone: %v", err)
	}
	frame := toneFrames(t, codec)
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for n := 0; ; n++ {
			if err := track.WriteSample(media.Sample{Data: frame(n), Duration: 20 * time.Millisecond}, nil); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// toneFrames gives the n-th 20 ms frame of a 440 Hz tone in the codec (Opus from go-opus's encoder, which codes
// CELT only; the decoder takes SILK and hybrid too, checked by its RFC 6716 vectors).
func toneFrames(t *testing.T, codec string) func(n int) []byte {
	t.Helper()
	if codec == webrtc.MimeTypePCMU {
		return func(n int) []byte {
			b := make([]byte, 160)
			for i := range b {
				b[i] = mulawOf(int16(8000 * math.Sin(2*math.Pi*440*float64(n*160+i)/8000)))
			}
			return b
		}
	}
	enc, err := opus.NewEncoder(opus.EncoderConfig{SampleRate: 48000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	return func(n int) []byte {
		pcm := make([]int16, 960)
		for i := range pcm {
			pcm[i] = int16(8000 * math.Sin(2*math.Pi*440*float64(n*960+i)/48000))
		}
		packet := make([]byte, 1276)
		k, err := enc.Encode(pcm, packet)
		if err != nil {
			return nil
		}
		return packet[:k]
	}
}

// mulawOf is the G.711 µ-law code of a sample.
func mulawOf(sample int16) byte {
	const bias, clip = 0x84, 32635
	v, sign := int(sample), byte(0)
	if v < 0 {
		v, sign = -v, 0x80
	}
	v = min(v, clip) + bias
	exp := 7
	for mask := 0x4000; v&mask == 0 && exp > 0; mask >>= 1 {
		exp--
	}
	return ^(sign | byte(exp<<4) | byte((v>>(exp+3))&0x0f))
}

// pitch is the frequency of a tone in 16 kHz PCM, from its zero crossings.
func pitch(pcm []byte) float64 {
	crossings, prev := 0, int16(0)
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		if (prev < 0) != (s < 0) {
			crossings++
		}
		prev = s
	}
	return float64(crossings) / 2 / (float64(len(pcm)/2) / 16000)
}

// The agent hears each person's microphone in the room, Opus or G.711, decoded to 16 kHz 16-bit mono PCM in
// 20 ms frames, by their identity; when it is removed from the room the tracks end (SAL-4).
func TestListen_HandsOverEachMicrophoneDecodedTo16kHzUntilTheAgentIsRemoved(t *testing.T) {
	t.Parallel()
	lk := newLiveKitServer(t)
	name := lk.room(t)
	p := lk.provider()
	token, err := p.Token(videosession.Grant{Room: name, Identity: "a_listen", ValidFor: time.Minute, Agent: true})
	if err != nil {
		t.Fatal(err)
	}
	l, err := videosession.NewLiveKitListener().Listen(t.Context(), strings.Replace(lk.url, "http", "ws", 1), token)
	if err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	t.Cleanup(l.Leave)
	for identity, codec := range map[string]string{"p_opus": webrtc.MimeTypeOpus, "p_pcmu": webrtc.MimeTypePCMU} {
		pt, err := p.Token(videosession.Grant{Room: name, Identity: identity, ValidFor: time.Minute})
		if err != nil {
			t.Fatal(err)
		}
		room, _ := lk.connect(t, pt)
		if room == nil {
			t.Fatalf("%s did not connect", identity)
		}
		publishTone(t, room, codec)
	}

	heard, tracks := hear(t, l, 2)
	for identity, pcm := range heard {
		if f := pitch(pcm[len(pcm)/2:]); f < 400 || f > 480 {
			t.Errorf("%s: a tone of %.0f Hz, want 440", identity, f)
		}
	}

	if err := p.RemoveParticipant(t.Context(), name, "a_listen"); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		for a := range l.Tracks {
			tracks = append(tracks, a)
		}
		for _, a := range tracks {
			for range a.Frames { //nolint:revive // drained until the track ends
			}
		}
		close(ended)
	}()
	select {
	case <-ended:
	case <-time.After(15 * time.Second):
		t.Fatal("the agent's tracks did not end within 15 s of its removal")
	}
}

// The agent's connection logs nothing: LiveKit's SDK and WebRTC stack print the room, the identities and the
// connection's addresses to the standard logger by default (SPEC.md §2: rooms and identities are never logged).
// Not parallel, so it owns the standard logger while it runs.
func TestListen_LogsNothing(t *testing.T) {
	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(prev) })

	lk := newLiveKitServer(t)
	name := lk.room(t)
	p := lk.provider()
	token, err := p.Token(videosession.Grant{Room: name, Identity: "a_quiet", ValidFor: time.Minute, Agent: true})
	if err != nil {
		t.Fatal(err)
	}
	l, err := videosession.NewLiveKitListener().Listen(t.Context(), lk.url, token)
	if err != nil {
		t.Fatalf("Listen() = %v", err)
	}
	pt, err := p.Token(videosession.Grant{Room: name, Identity: "p_quiet", ValidFor: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	room, _ := lk.connect(t, pt)
	if room == nil {
		t.Fatal("the publisher did not connect")
	}
	publishTone(t, room, webrtc.MimeTypeOpus)
	select {
	case a := <-l.Tracks:
		<-a.Frames
	case <-time.After(20 * time.Second):
		t.Fatal("no track within 20 s")
	}
	l.Leave()
	if logs.Len() != 0 {
		t.Errorf("the connection logged %d bytes, want none; the first line: %s", logs.Len(), strings.SplitN(logs.String(), "\n", 2)[0])
	}
}

// hear reads a second of audio from each of the first n tracks, checking each frame is 20 ms of 16 kHz 16-bit
// mono PCM, and returns the audio by identity and the tracks.
func hear(t *testing.T, l videosession.Listening, n int) (map[string][]byte, []videosession.RemoteAudio) {
	t.Helper()
	heard := map[string][]byte{}
	var tracks []videosession.RemoteAudio
	deadline := time.After(20 * time.Second)
	for len(heard) < n {
		select {
		case a := <-l.Tracks:
			var pcm []byte
			for range 50 {
				f, ok := <-a.Frames
				if !ok {
					t.Fatalf("%s's track ended early", a.Identity)
				}
				if len(f) != 640 {
					t.Fatalf("%s: a frame of %d bytes, want 640 (20 ms at 16 kHz, 16-bit mono)", a.Identity, len(f))
				}
				pcm = append(pcm, f...)
			}
			heard[a.Identity] = pcm
			tracks = append(tracks, a)
		case <-deadline:
			t.Fatalf("heard %d tracks within 20 s, want %d", len(heard), n)
		}
	}
	return heard, tracks
}

// BenchmarkListen_FrameLatency is the LiveKit hop of the partial caption's latency (SAL-4): from a synthetic tone
// burst's first Opus frame written by a publisher on the compose LiveKit to the agent's first decoded frame of it,
// after 200 ms of silence each time; p50, p95 and p99 reported (make bench-stt).
func BenchmarkListen_FrameLatency(b *testing.B) {
	lk := newLiveKitServer(b)
	name := "r_bench_" + strings.ToLower(rand.Text())
	p := lk.provider()
	if err := p.CreateRoom(b.Context(), videosession.Room{Name: name, MaxParticipants: 2, EmptyTimeout: time.Hour}); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = p.DeleteRoom(context.Background(), name) })
	token, err := p.Token(videosession.Grant{Room: name, Identity: "a_bench", ValidFor: time.Minute, Agent: true})
	if err != nil {
		b.Fatal(err)
	}
	l, err := videosession.NewLiveKitListener().Listen(b.Context(), lk.url, token)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(l.Leave)
	pt, err := p.Token(videosession.Grant{Room: name, Identity: "p_bench", ValidFor: time.Minute})
	if err != nil {
		b.Fatal(err)
	}
	room, err := lksdk.ConnectToRoomWithToken(strings.Replace(lk.url, "http", "ws", 1), pt, lksdk.NewRoomCallback(),
		lksdk.WithAutoSubscribe(false), lksdk.WithLogger(protologger.GetDiscardLogger()))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(room.Disconnect)
	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2})
	if err != nil {
		b.Fatal(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Source: livekit.TrackSource_MICROPHONE}); err != nil {
		b.Fatal(err)
	}
	loud := listenLoud(b, l)
	tone, silence := burstFrames(b)
	write := func(f []byte) { _ = track.WriteSample(media.Sample{Data: f, Duration: 20 * time.Millisecond}, nil) }
	wall := clock.System{}
	took := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for range b.N {
		for range 10 {
			write(silence())
			time.Sleep(20 * time.Millisecond)
		}
		drainLoud(loud)
		start := wall.Now()
		got := false
		for i := 0; i < 50 && !got; i++ {
			write(tone())
			select {
			case at := <-loud:
				took, got = append(took, at.Sub(start)), true
			case <-time.After(20 * time.Millisecond):
			}
		}
		if !got {
			b.Fatal("the agent heard no tone within a second")
		}
	}
	b.StopTimer()
	reportPercentiles(b, took)
}

// listenLoud reports the time of each decoded frame of the first track that holds a tone (not silence).
func listenLoud(b *testing.B, l videosession.Listening) chan time.Time {
	b.Helper()
	loud := make(chan time.Time, 100)
	var a videosession.RemoteAudio
	select {
	case a = <-l.Tracks:
	case <-time.After(20 * time.Second):
		b.Fatal("no track within 20 s")
	}
	go func() {
		wall := clock.System{}
		for f := range a.Frames {
			var sum float64
			for i := 0; i+1 < len(f); i += 2 {
				s := float64(int16(uint16(f[i]) | uint16(f[i+1])<<8))
				sum += s * s
			}
			if math.Sqrt(sum/float64(len(f)/2)) > 1000 {
				select {
				case loud <- wall.Now():
				default:
				}
			}
		}
	}()
	return loud
}

func drainLoud(loud chan time.Time) {
	for {
		select {
		case <-loud:
		default:
			return
		}
	}
}

// burstFrames encodes, with one encoder, the frames of a continuous 440 Hz tone and of digital silence.
func burstFrames(b *testing.B) (tone, silence func() []byte) {
	b.Helper()
	enc, err := opus.NewEncoder(opus.EncoderConfig{SampleRate: 48000, Channels: 1})
	if err != nil {
		b.Fatal(err)
	}
	n := 0
	encode := func(amp float64) []byte {
		pcm := make([]int16, 960)
		for i := range pcm {
			pcm[i] = int16(amp * math.Sin(2*math.Pi*440*float64(n*960+i)/48000))
		}
		n++
		packet := make([]byte, 1276)
		k, err := enc.Encode(pcm, packet)
		if err != nil {
			return nil
		}
		return packet[:k]
	}
	return func() []byte { return encode(8000) }, func() []byte { return encode(0) }
}

// reportPercentiles reports p50, p95 and p99 of took in milliseconds.
func reportPercentiles(b *testing.B, took []time.Duration) {
	b.Helper()
	slices.Sort(took)
	for _, p := range []struct {
		name string
		q    float64
	}{{"p50-ms", 0.50}, {"p95-ms", 0.95}, {"p99-ms", 0.99}} {
		b.ReportMetric(float64(took[int(p.q*float64(len(took)-1))].Microseconds())/1000, p.name)
	}
}
