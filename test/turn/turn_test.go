//go:build turn

// Package turn_test proves the compose LiveKit's TURN (docs/consultation/SPEC.md §2 "Changed for SAL-3"): with
// UDP and LiveKit's ICE/TCP port blocked, as on a hotel Wi-Fi that lets only TCP 443 out, a call does not
// connect without TURN and does through LiveKit's TURN over TLS on 443. Run only by `make turn-check`
// (scripts/turn-check.sh blocks the ports first); the build tag keeps it out of `make test`.
package turn_test

import (
	"crypto/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	"github.com/Shmeeshmee/salus/internal/videosession"
)

// waitMedia is how long a subscriber waits for the first audio packet.
const waitMedia = 20 * time.Second

// opusSilence is one 20 ms Opus frame of silence: synthetic audio, never a recording.
var opusSilence = []byte{0xf8, 0xff, 0xfe}

type server struct{ url, key, secret string }

func newServer(t *testing.T) server {
	t.Helper()
	s := server{url: os.Getenv("TEST_LIVEKIT_URL"), key: os.Getenv("TEST_LIVEKIT_API_KEY"), secret: os.Getenv("TEST_LIVEKIT_API_SECRET")}
	if s.url == "" || s.key == "" || s.secret == "" {
		t.Fatal("TEST_LIVEKIT_URL, TEST_LIVEKIT_API_KEY and TEST_LIVEKIT_API_SECRET are not set; run `make turn-check`")
	}
	return s
}

// call creates a room for two, joins the patient, who publishes synthetic microphone audio, and the
// practitioner and the agent, who subscribe, all with opts; it reports which of the two subscribers received
// audio within waitMedia.
func call(t *testing.T, opts ...lksdk.ConnectOption) map[string]bool {
	t.Helper()
	s := newServer(t)
	p := videosession.NewLiveKitProvider(s.url, s.key, s.secret)
	name := "r_turn_" + strings.ToLower(rand.Text())
	if err := p.CreateRoom(t.Context(), videosession.Room{Name: name, MaxParticipants: 2, EmptyTimeout: time.Hour}); err != nil {
		t.Fatalf("CreateRoom() = %v", err)
	}
	t.Cleanup(func() { _ = p.DeleteRoom(t.Context(), name) })
	ws := strings.Replace(s.url, "http", "ws", 1)
	opts = append(opts, lksdk.WithConnectTimeout(15*time.Second))

	got := map[string]bool{}
	heard := make(chan string, 2)
	for _, who := range []string{"practitioner", "agent"} {
		token, err := p.Token(videosession.Grant{Room: name, Identity: map[string]string{"practitioner": "p_", "agent": "a_"}[who] + who, ValidFor: time.Minute, Agent: who == "agent"})
		if err != nil {
			t.Fatal(err)
		}
		cb := lksdk.NewRoomCallback()
		cb.ParticipantCallback.OnTrackSubscribed = func(track *webrtc.TrackRemote, _ *lksdk.RemoteTrackPublication, _ *lksdk.RemoteParticipant) {
			go func() {
				if _, _, err := track.ReadRTP(); err == nil {
					heard <- who
				}
			}()
		}
		room, err := lksdk.ConnectToRoomWithToken(ws, token, cb, opts...)
		if err != nil {
			t.Logf("the %s did not connect: %v", who, err)
			continue
		}
		t.Cleanup(room.Disconnect)
	}

	token, err := p.Token(videosession.Grant{Room: name, Identity: "p_patient", ValidFor: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	room, err := lksdk.ConnectToRoomWithToken(ws, token, lksdk.NewRoomCallback(), opts...)
	if err != nil {
		t.Logf("the patient did not connect: %v", err)
		return got
	}
	t.Cleanup(room.Disconnect)
	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Source: livekit.TrackSource_MICROPHONE}); err != nil {
		t.Logf("the patient could not publish: %v", err)
		return got
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				_ = track.WriteSample(media.Sample{Data: opusSilence, Duration: 20 * time.Millisecond}, nil)
			}
		}
	}()

	deadline := time.After(waitMedia)
	for len(got) < 2 {
		select {
		case who := <-heard:
			got[who] = true
		case <-deadline:
			return got
		}
	}
	return got
}

// Without TURN, the blocked ports leave no way through: no subscriber hears the patient. This shows the block
// holds, so the next test's success can only be TURN's.
func TestCall_DoesNotConnectWithoutTURN(t *testing.T) {
	if got := call(t, lksdk.WithDisableTURN()); got["practitioner"] || got["agent"] {
		t.Fatalf("heard %v without TURN; want nothing (are UDP and 7881 blocked? run `make turn-check`)", got)
	}
}

// With only relayed candidates the call connects through TURN over TLS on 443 (UDP is blocked, so TURN over
// UDP on 3478 is too): the practitioner and the subscribe-only agent both hear the patient.
func TestCall_ConnectsThroughTURNOverTLS(t *testing.T) {
	if got := call(t, lksdk.WithICETransportPolicy(webrtc.ICETransportPolicyRelay)); !got["practitioner"] || !got["agent"] {
		t.Fatalf("heard %v through TURN; want the practitioner and the agent", got)
	}
}
