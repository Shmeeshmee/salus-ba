package worker_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
)

// watched is the compose LiveKit's listener, reporting each time the agent is out of a room.
type watched struct {
	videosession.Listener
	out   chan struct{}
	mu    sync.Mutex
	joins int
}

func (w *watched) Listen(ctx context.Context, url, token string) (videosession.Listening, error) {
	l, err := w.Listener.Listen(ctx, url, token)
	if err != nil {
		return l, err
	}
	w.mu.Lock()
	w.joins++
	w.mu.Unlock()
	tracks := make(chan videosession.RemoteAudio)
	go func() {
		for a := range l.Tracks {
			tracks <- a
		}
		close(tracks)
		w.out <- struct{}{}
	}()
	return videosession.Listening{Tracks: tracks, Leave: l.Leave}, nil
}

func (w *watched) joined() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.joins
}

// runAgentJob starts the agent's client, enqueues transcription_join for the session as the webhook does, and
// returns how its first attempt ended and the log; the client and the transcription stop at the test's end.
func runAgentJob(t *testing.T, db logins, w *watched, sessionID string) (river.EventKind, string) {
	t.Helper()
	logs := &syncBuffer{}
	video := worker.Video{Provider: liveKit(t), Listener: w, URL: os.Getenv("TEST_LIVEKIT_URL")}
	client, live, err := worker.NewAgent(db.app, redistest.NewClient(t), video, &stttest.Recognizer{}, slog.New(slog.NewJSONHandler(logs, nil)))
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	t.Cleanup(live.Stop)
	events, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobFailed, river.EventKindJobCancelled)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(context.WithoutCancel(t.Context())) }()
	inserted, err := client.Insert(t.Context(), videosession.AgentJoinArgs{SessionID: sessionID, OrganizationID: clinicID},
		&river.InsertOpts{Queue: videosession.QueueTranscription})
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case ev := <-events:
			if ev.Job.ID == inserted.Job.ID {
				return ev.Kind, logs.String()
			}
		case <-time.After(20 * time.Second):
			t.Fatal("transcription_join did not finish within 20 s")
		}
	}
}

// openConsultation opens, as the owner, the session of the appointment that started an hour ago with its window
// still open, the practitioner in the call and the Encounter in progress.
func openConsultation(t *testing.T, db logins, b booking) consultationRows {
	t.Helper()
	c := seedConsultation(t, db, b, true, true)
	if _, err := db.owner.Exec(t.Context(), `UPDATE video_session SET closes_at = now() + interval '30 minutes' WHERE id = $1`, c.session); err != nil {
		t.Fatal(err)
	}
	return c
}

// The agent joins the consultation's room on the job (SAL-4) and stays; when the patient revokes their consent,
// the revocation's job removes it, and it leaves at once. The next join is refused, and its job cancelled. The
// logs name the session, never the room, an identity or a name.
func TestTranscriptionJoin_JoinsTheRoomAndLeavesWhenTheConsentIsRevoked(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	c := openConsultation(t, db, b)
	w := &watched{Listener: videosession.NewLiveKitListener(), out: make(chan struct{}, 2)}

	if ended, logs := runAgentJob(t, db, w, c.session); ended != river.EventKindJobCompleted || w.joined() != 1 {
		t.Fatalf("transcription_join ended %s with %d joins, want completed with 1:\n%s", ended, w.joined(), logs)
	} else {
		checkConsultationLog(t, logs, b, c)
	}
	select {
	case <-w.out:
		t.Fatal("the agent left the room before the revocation")
	default:
	}

	if _, err := db.owner.Exec(t.Context(), "UPDATE consent SET status = 'inactive', revoked_at = now()"); err != nil {
		t.Fatal(err)
	}
	if ended, logs := runWorkerJobWith(t, db, liveKit(t), consent.RevokedArgs{PatientID: b.patient, OrganizationID: clinicID}); ended != river.EventKindJobCompleted {
		t.Fatalf("consent_revoked ended %s:\n%s", ended, logs)
	}
	select {
	case <-w.out:
	case <-time.After(15 * time.Second):
		t.Fatal("the agent was still in the room 15 s after the revocation's removal")
	}

	ended, logs := runAgentJob(t, db, w, c.session)
	if ended != river.EventKindJobCancelled || w.joined() != 1 || !strings.Contains(logs, "consent required") {
		t.Errorf("transcription_join after the revocation ended %s with %d joins, want cancelled with 1:\n%s", ended, w.joined(), logs)
	}
	checkConsultationLog(t, logs, b, c)
}

// NewAgent needs everything the agent works with.
func TestNewAgent_NeedsItsDependencies(t *testing.T) {
	t.Parallel()
	if _, _, err := worker.NewAgent(newLogins(t).app, nil, worker.Video{}, nil, slog.New(slog.DiscardHandler)); err == nil {
		t.Error("NewAgent() without Redis, video or a recognizer = nil error")
	}
}
