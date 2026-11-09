package transcript

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/stt"
)

// agent names the transcription agent as the agent of the reads it asks for (the same name videosession audits
// its token requests with).
const agent = "transcription_agent"

// sampleRate is the rate of the audio the agent hears (videosession.AudioTrack): 16 kHz.
const sampleRate = 16000

// defaultRetry is how long a failed recognition waits before it is opened again.
const defaultRetry = time.Second

// Calls puts the transcription agent in a session's room (videosession.Service).
type Calls interface {
	AgentListen(ctx context.Context, sessionID, org string) (videosession.AgentCall, error)
}

// Appointments gives the appointment's patient's account, audited with the agent (appointment.Service).
type Appointments interface {
	Notice(ctx context.Context, agent, org, id string) (appointment.Notice, error)
}

// Accounts gives an account's language, audited with the agent (auth.Service).
type Accounts interface {
	AccountLanguage(ctx context.Context, agent, id string) (string, error)
}

// Clinics gives a clinic's default language (organization.Service).
type Clinics interface {
	DefaultLanguage(ctx context.Context, organizationID string) (string, error)
}

// Ports are the Service's dependencies. Retry is how long a failed recognition waits before it is opened again
// (a second when zero). Log takes the session's id, a speaker's role and errors, never what was said.
type Ports struct {
	Calls        Calls
	Appointments Appointments
	Accounts     Accounts
	Clinics      Clinics
	Publisher    Publisher
	Recognizer   stt.Recognizer
	Clock        clock.Clock
	Log          *slog.Logger
	Retry        time.Duration
}

// Service is the live transcription of the consultations (docs/plans/consult-intelligence.md, SAL-4): the agent
// in each consultation's room, each person's microphone through speech-to-text in their language, every partial
// and final result as a transcript.Event on the encounter's stream. Translation is SAL-5's; storing the final
// segments is SAL-6's.
type Service struct {
	calls map[string]context.CancelFunc // the sessions the agent is in, by id
	Ports
	wg sync.WaitGroup
	mu sync.Mutex
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service {
	if p.Retry == 0 {
		p.Retry = defaultRetry
	}
	return &Service{Ports: p, calls: map[string]context.CancelFunc{}}
}

// Transcribe puts the agent in the room of the session of the clinic org, unless it is in it already, and
// transcribes the call in the background until the agent is out of the room (removed when the patient's consent
// is withdrawn, the room deleted at the end) or Stop. It returns once the agent is in: AgentListen's refusals
// (videosession.ErrConsentRequired, ErrConsultationEnded, ErrNotFound, ErrNotStarted), or an error to try again.
func (s *Service) Transcribe(ctx context.Context, sessionID, org string) error {
	callCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.mu.Lock()
	if _, in := s.calls[sessionID]; in {
		s.mu.Unlock()
		cancel()
		return nil
	}
	s.calls[sessionID] = cancel
	s.mu.Unlock()

	c, err := s.join(ctx, sessionID, org)
	if err != nil {
		s.forget(sessionID)
		cancel()
		return err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.forget(sessionID)
		defer cancel()
		s.transcribe(callCtx, c)
	}()
	return nil
}

// Stop ends every call (the agent leaves its rooms) and returns once their recognitions ended.
func (s *Service) Stop() {
	s.mu.Lock()
	for _, cancel := range s.calls {
		cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Service) forget(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.calls, sessionID)
}

// call is a consultation the agent is in.
type call struct {
	locales map[videosession.Role]string
	videosession.AgentCall
	sessionID string
}

// join puts the agent in the room and reads the languages its speakers are recognised in: the patient's
// account's (else the clinic's), the practitioner's the clinic's. Without them it leaves.
func (s *Service) join(ctx context.Context, sessionID, org string) (call, error) {
	ac, err := s.Calls.AgentListen(ctx, sessionID, org)
	if err != nil {
		return call{}, err
	}
	locales, err := s.locales(ctx, org, ac.AppointmentID)
	if err != nil {
		ac.Leave()
		for range ac.Tracks { //nolint:revive // read until closed, as AgentCall asks
		}
		return call{}, fmt.Errorf("transcript: languages: %w", err)
	}
	return call{AgentCall: ac, sessionID: sessionID, locales: locales}, nil
}

func (s *Service) locales(ctx context.Context, org, appointmentID string) (map[videosession.Role]string, error) {
	clinic, err := s.Clinics.DefaultLanguage(ctx, org)
	if err != nil {
		return nil, err
	}
	n, err := s.Appointments.Notice(ctx, agent, org, appointmentID)
	if err != nil {
		return nil, err
	}
	patient, err := s.Accounts.AccountLanguage(ctx, agent, n.AccountID)
	if err != nil {
		return nil, err
	}
	if patient == "" {
		patient = clinic
	}
	return map[videosession.Role]string{
		videosession.RolePatient: speechLocale(patient), videosession.RolePractitioner: speechLocale(clinic),
	}, nil
}

// transcribe recognises each track of the call until the agent is out of the room or ctx ends; then every
// recognition ends at once, so nothing said after the agent's removal is processed.
func (s *Service) transcribe(ctx context.Context, c call) {
	ctx, cancel := context.WithCancel(ctx)
	var tracks sync.WaitGroup
	defer tracks.Wait()
	defer cancel()
	defer c.Leave()
	for {
		select {
		case tr, ok := <-c.Tracks:
			if !ok {
				return
			}
			tracks.Add(1)
			go func() {
				defer tracks.Done()
				s.track(ctx, c, tr)
			}()
		case <-ctx.Done():
			c.Leave()
			for range c.Tracks { //nolint:revive // read until closed, as AgentCall asks
			}
			return
		}
	}
}

// speakerOf is the transcript's speaker of a role in the room.
func speakerOf(r videosession.Role) Speaker {
	if r == videosession.RolePractitioner {
		return SpeakerPractitioner
	}
	return SpeakerPatient
}

// track recognises one microphone until its audio ends or ctx ends, opening the recognition again after a
// failure.
func (s *Service) track(ctx context.Context, c call, tr videosession.AudioTrack) {
	sp := speaker{Speaker: speakerOf(tr.Role), locale: c.locales[tr.Role]}
	for {
		done, err := s.recognise(ctx, c, sp, tr.Frames)
		if done || ctx.Err() != nil {
			return
		}
		s.Log.WarnContext(ctx, "speech-to-text failed", "session_id", c.sessionID, "speaker", string(sp.Speaker), "error", err.Error())
		select {
		case <-time.After(s.Retry):
		case <-ctx.Done():
			return
		}
	}
}

// speaker is who speaks on a track, and the locale they are recognised in.
type speaker struct {
	Speaker
	locale string
}

// errStreamEnded is a recognition that ended while the person still had audio.
var errStreamEnded = errors.New("transcript: the recognition ended early")

// recognise runs one recognition over frames: done when they end (after the remaining results), or the
// recognition's failure.
func (s *Service) recognise(ctx context.Context, c call, sp speaker, frames <-chan []byte) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := s.Recognizer.Stream(ctx, stt.Config{Lang: sp.locale, SampleRate: sampleRate})
	if err != nil {
		return false, err
	}
	results := make(chan error, 1)
	go func() { results <- s.publish(ctx, c, sp, s.Clock.Now(), stream) }()
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				_ = stream.CloseSend()
				return true, <-results
			}
			if err := stream.Send(f); err != nil {
				cancel() // the results end with the context
				if rerr := <-results; rerr != nil && !errors.Is(rerr, context.Canceled) {
					err = errors.Join(err, rerr)
				}
				return false, err
			}
		case err := <-results:
			if err == nil {
				err = errStreamEnded
			}
			return false, err
		case <-ctx.Done():
			<-results //nolint:errcheck // the call is over: the results end with the context, unpublished
			return true, nil
		}
	}
}

// newID is a segment's id: a random (version 4) UUID.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
