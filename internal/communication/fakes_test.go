package communication_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	clinicB   = "b0000000-0000-4000-8000-00000000000b"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	patientB  = "bb000000-0000-4000-8000-0000000000bb"
	roleA1    = "a1000000-0000-4000-8000-0000000000a1"
	roleA2    = "a2000000-0000-4000-8000-0000000000a2"
	encA      = "e1000000-0000-4000-8000-0000000000e1"
	encOther  = "e2000000-0000-4000-8000-0000000000e2"
	unknownID = "00000000-0000-4000-8000-000000000000"
	clientID1 = "c1000000-0000-4000-8000-0000000000c1"
	clientID2 = "c2000000-0000-4000-8000-0000000000c2"
)

var (
	me     = appointment.Caller{AccountID: "acc0000a-0000-4000-8000-0000000000aa", PatientID: patientA}
	other  = appointment.Caller{AccountID: "acc0000b-0000-4000-8000-0000000000bb", PatientID: patientB}
	pract  = appointment.Caller{AccountID: "ac100000-0000-4000-8000-0000000000a1", OrganizationID: clinicA, PractitionerID: "p1000000-0000-4000-8000-0000000000a1"}
	pract2 = appointment.Caller{AccountID: "ac200000-0000-4000-8000-0000000000a2", OrganizationID: clinicA, PractitionerID: "p2000000-0000-4000-8000-0000000000a2"}
	admin  = appointment.Caller{AccountID: "ad000000-0000-4000-8000-0000000000ad", OrganizationID: clinicA, Admin: true}

	errBackend = errors.New("the database is down")
	start      = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
)

type txKey struct{}

// testClock is a clock.Fixed the test moves.
type testClock struct {
	t  time.Time
	mu sync.Mutex
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

// fixture is the service on fakes, with patient A's active consent and an appointment with role A1 in clinic A.
type fixture struct {
	svc     *communication.Service
	repo    *fakeRepo
	audit   *fakeAudit
	pub     *fakePublisher
	clock   *testClock
	consent map[string]access.Consent // by patient+clinic
	visits  map[string][]access.Visit
	chat    map[string]bool // the active chat consents (SAL-28), by conversation+role
	mu      sync.Mutex
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return newFixtureOn(t, nil, nil)
}

// newFixtureOn is the fixture on repo and tx (the fakes when nil): the repository tests run the service on
// Postgres.
func newFixtureOn(t *testing.T, repo communication.Repository, tx communication.Transactor) *fixture {
	t.Helper()
	f := &fixture{
		repo: newFakeRepo(), audit: &fakeAudit{}, pub: &fakePublisher{}, clock: &testClock{t: start},
		consent: map[string]access.Consent{patientA + clinicA: {Active: true}},
		visits:  map[string][]access.Visit{patientA + clinicA: {{PractitionerRole: roleA1, Start: start.Add(-48 * time.Hour)}}},
		chat:    map[string]bool{},
	}
	f.repo.consented = f.consented
	if repo == nil {
		repo, tx = f.repo, fakeTx{f}
	}
	f.svc = communication.NewService(communication.Ports{
		Repo: repo, Publisher: f.pub, Consents: fakeConsents{f}, Visits: fakeVisits{f}, Roles: fakeRoles{},
		Encounters: fakeEncounters{}, ChatConsents: fakeChatConsents{f}, Directory: fakeDirectory{}, Audit: f.audit, Tx: tx, Clock: f.clock,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return f
}

// revoke revokes patient A's consent with clinic A.
func (f *fixture) revoke() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consent[patientA+clinicA] = access.Consent{RevokedAt: f.clock.Now()}
}

// open opens patient A's conversation with role A1 as the patient and fails the test otherwise.
func (f *fixture) open(t *testing.T) communication.Conversation {
	t.Helper()
	c, _, err := f.svc.Open(t.Context(), me, communication.Open{OrganizationID: clinicA, PractitionerRoleID: roleA1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// send sends body as c with a fresh client id and fails the test otherwise.
func (f *fixture) send(t *testing.T, c appointment.Caller, conv, body string) communication.Message {
	t.Helper()
	f.mu.Lock()
	f.repo.clientSeq++
	id := fmt.Sprintf("c%07d-0000-4000-8000-000000000000", f.repo.clientSeq)
	f.mu.Unlock()
	m, _, err := f.svc.Send(t.Context(), c, conv, id, body)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type fakeTx struct{ f *fixture }

// txState is one fake transaction: its scope, and the conversation locks it holds until it ends.
type txState struct {
	scope  scope
	unlock []func()
}

// WithTx rolls back the fake repository and audit when fn fails, as a transaction does; the scope and the locks
// end with it. A rollback assumes no other transaction wrote meanwhile (the tests that fail one run alone).
func (t fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	t.f.repo.mu.Lock()
	snap := t.f.repo.snapshot()
	t.f.repo.mu.Unlock()
	events := t.f.audit.count()
	tx := &txState{}
	err := fn(context.WithValue(ctx, txKey{}, tx))
	if err != nil {
		t.f.repo.mu.Lock()
		t.f.repo.restore(snap)
		t.f.repo.mu.Unlock()
		t.f.audit.truncate(events)
	}
	for _, u := range tx.unlock {
		u()
	}
	return err
}

func txOf(ctx context.Context) *txState {
	if tx, ok := ctx.Value(txKey{}).(*txState); ok {
		return tx
	}
	return &txState{}
}

type fakeConsents struct{ f *fixture }

func (c fakeConsents) State(_ context.Context, patient, org string) (access.Consent, error) {
	c.f.mu.Lock()
	defer c.f.mu.Unlock()
	return c.f.consent[patient+org], nil
}

type fakeVisits struct{ f *fixture }

func (v fakeVisits) Visits(_ context.Context, patient, org string) ([]access.Visit, error) {
	v.f.mu.Lock()
	defer v.f.mu.Unlock()
	return v.f.visits[patient+org], nil
}

type fakeRoles struct{}

func (fakeRoles) OfPractitioner(_ context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error) {
	switch {
	case org == clinicA && practitionerID == pract.PractitionerID:
		return practitionerrole.PractitionerRole{ID: roleA1, OrganizationID: clinicA}, nil
	case org == clinicA && practitionerID == pract2.PractitionerID:
		return practitionerrole.PractitionerRole{ID: roleA2, OrganizationID: clinicA}, nil
	case org == clinicB && practitionerID == guest.PractitionerID:
		return practitionerrole.PractitionerRole{ID: roleB1, OrganizationID: clinicB}, nil
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

type fakeEncounters struct{}

func (fakeEncounters) Lookup(_ context.Context, id string) (encounter.Encounter, error) {
	switch id {
	case encA:
		return encounter.Encounter{ID: encA, PatientID: patientA, OrganizationID: clinicA}, nil
	case encOther:
		return encounter.Encounter{ID: encOther, PatientID: patientB, OrganizationID: clinicA}, nil
	}
	return encounter.Encounter{}, encounter.ErrNotFound
}

type fakeAudit struct {
	events []auditevent.AuditEvent
	mu     sync.Mutex
}

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, e)
	return nil
}

func (a *fakeAudit) truncate(n int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = a.events[:n]
}

func (a *fakeAudit) last(t *testing.T) auditevent.AuditEvent {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.events) == 0 {
		t.Fatal("nothing audited")
	}
	return a.events[len(a.events)-1]
}

func (a *fakeAudit) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.events)
}

type fakePublisher struct {
	err    error
	frames []wire.Envelope
	mu     sync.Mutex
}

func (p *fakePublisher) Publish(_ context.Context, channel string, frame []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	env, err := wire.Decode(frame)
	if err != nil {
		return err
	}
	if channel == "" {
		return errors.New("no channel")
	}
	p.frames = append(p.frames, env)
	return nil
}

func (p *fakePublisher) published() []wire.Envelope {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.frames)
}

// scope is the fake transaction's settings.
type scope struct{ patient, clinic, conversation, role string }

// fakeRepo keeps the rows in memory and applies the row-level security of migration 00041 that the rules lean on:
// a patient sees their own conversations but no doctor-only one; a clinic its own; messages only with the
// conversation set, or to their patient.
type fakeRepo struct {
	since    time.Time // the last MessageByClientID's window
	failNext error
	convs    map[string]communication.Conversation
	parts    map[string][]communication.Participant
	msgs     map[string][]communication.Message
	revs     map[string][]communication.Revision
	locks    map[string]*sync.Mutex
	// consented reports whether the chat consent for the conversation and role and the care consent of the patient
	// with the clinic are active (the fixture's), for the guest's row-level security.
	consented  func(conv, role, patient, clinic string) bool
	partitions []string // the AddPartitions calls, "<from>/<months>"
	nextID     int
	clientSeq  int
	mu         sync.Mutex
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		convs: map[string]communication.Conversation{}, parts: map[string][]communication.Participant{},
		msgs: map[string][]communication.Message{}, revs: map[string][]communication.Revision{}, locks: map[string]*sync.Mutex{},
	}
}

type repoSnapshot struct {
	convs map[string]communication.Conversation
	parts map[string][]communication.Participant
	msgs  map[string][]communication.Message
	revs  map[string][]communication.Revision
}

func cloneMap[V any](m map[string][]V) map[string][]V {
	out := make(map[string][]V, len(m))
	for k, v := range m {
		out[k] = slices.Clone(v)
	}
	return out
}

func (r *fakeRepo) snapshot() repoSnapshot {
	convs := make(map[string]communication.Conversation, len(r.convs))
	for k, v := range r.convs {
		convs[k] = v
	}
	return repoSnapshot{convs: convs, parts: cloneMap(r.parts), msgs: cloneMap(r.msgs), revs: cloneMap(r.revs)}
}

func (r *fakeRepo) restore(s repoSnapshot) {
	r.convs, r.parts, r.msgs, r.revs = s.convs, s.parts, s.msgs, s.revs
}

func (r *fakeRepo) id(prefix string) string {
	r.nextID++
	return fmt.Sprintf("%s%07d-0000-4000-8000-000000000000", prefix, r.nextID)
}

func (r *fakeRepo) fail() error {
	err := r.failNext
	r.failNext = nil
	return err
}

func (r *fakeRepo) SetPatient(ctx context.Context, id string) error {
	txOf(ctx).scope.patient = id
	return nil
}

func (r *fakeRepo) SetClinic(ctx context.Context, id string) error {
	txOf(ctx).scope.clinic = id
	return nil
}

func (r *fakeRepo) SetConversation(ctx context.Context, id string) error {
	txOf(ctx).scope.conversation = id
	return nil
}

// visible is the conversation row-level security for the scope (00041), and the guest's (00044): a conversation and
// its side threads to a doctor of another clinic who is an active participant of it with both consents active.
func (r *fakeRepo) visible(sc scope, c communication.Conversation) bool {
	if sc.patient != "" {
		return c.PatientID == sc.patient && !c.StaffOnly
	}
	if sc.clinic != "" && c.OrganizationID == sc.clinic {
		return true
	}
	return sc.clinic != "" && sc.role != "" && r.guestOf(sc, c)
}

func (r *fakeRepo) messagesVisible(ctx context.Context, convID string) bool {
	sc := txOf(ctx).scope
	c, ok := r.convs[convID]
	if !ok {
		return false
	}
	if sc.patient != "" && c.PatientID == sc.patient && !c.StaffOnly {
		return true
	}
	return sc.conversation == convID && r.visible(sc, c)
}

func (r *fakeRepo) CreateConversation(_ context.Context, c communication.Conversation, parts []communication.Participant) (communication.Conversation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return communication.Conversation{}, false, err
	}
	for _, have := range r.convs {
		if !have.StaffOnly && have.PatientID == c.PatientID && have.PractitionerRoleID == c.PractitionerRoleID && have.EncounterID == c.EncounterID {
			have.Participants = slices.Clone(r.parts[have.ID])
			return have, false, nil
		}
	}
	c.ID = r.id("c")
	r.convs[c.ID] = c
	for _, p := range parts {
		p.ID, p.ConversationID = r.id("d"), c.ID
		r.parts[c.ID] = append(r.parts[c.ID], p)
	}
	c.Participants = slices.Clone(r.parts[c.ID])
	return c, true, nil
}

func (r *fakeRepo) Conversation(ctx context.Context, id string, lock bool) (communication.Conversation, error) {
	r.mu.Lock()
	l := r.locks[id]
	if l == nil {
		l = &sync.Mutex{}
		r.locks[id] = l
	}
	r.mu.Unlock()
	if lock { // SELECT … FOR UPDATE: held until the transaction ends
		l.Lock()
		tx := txOf(ctx)
		tx.unlock = append(tx.unlock, l.Unlock)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return communication.Conversation{}, err
	}
	c, ok := r.convs[id]
	if !ok || !r.visible(txOf(ctx).scope, c) {
		return communication.Conversation{}, communication.ErrNotFound
	}
	return c, nil
}

func (r *fakeRepo) Participants(_ context.Context, id string) ([]communication.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.parts[id]), nil
}

func (r *fakeRepo) Conversations(ctx context.Context, f communication.ListFilter) ([]communication.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []communication.Conversation
	for _, c := range r.convs {
		if !r.visible(txOf(ctx).scope, c) {
			continue
		}
		for _, p := range r.parts[c.ID] {
			if (f.PatientID != "" && p.PatientID == f.PatientID) || (f.PractitionerRoleID != "" && p.PractitionerRoleID == f.PractitionerRoleID) {
				c.Unread = c.LastSeq - p.ReadSeq
				out = append(out, c)
			}
		}
	}
	slices.SortFunc(out, func(a, b communication.Conversation) int { return b.LastSentAt.Compare(a.LastSentAt) })
	return out[:min(len(out), f.Limit)], nil
}

func (r *fakeRepo) Messages(ctx context.Context, id string, before *communication.Cursor, fromSeq int64, limit int) ([]communication.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.messagesVisible(ctx, id) {
		return nil, nil
	}
	var out []communication.Message
	all := r.msgs[id]
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		m := all[i]
		if m.Seq <= fromSeq {
			break
		}
		if before == nil || m.Seq < before.Seq {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *fakeRepo) After(ctx context.Context, id string, after communication.Cursor, fromSeq int64, limit int) ([]communication.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.messagesVisible(ctx, id) {
		return nil, nil
	}
	var out []communication.Message
	for _, m := range r.msgs[id] {
		if m.Seq > after.Seq && m.Seq > fromSeq && len(out) < limit {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *fakeRepo) Message(ctx context.Context, convID, id string) (communication.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.messagesVisible(ctx, convID) {
		for _, m := range r.msgs[convID] {
			if m.ID == id {
				return m, nil
			}
		}
	}
	return communication.Message{}, communication.ErrNotFound
}

func (r *fakeRepo) MessageByClientID(_ context.Context, convID, partID, clientID string, since time.Time) (communication.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.since = since
	for _, m := range r.msgs[convID] {
		if m.SenderParticipantID == partID && m.ClientMessageID == clientID && !m.SentAt.Before(since) {
			return m, nil
		}
	}
	return communication.Message{}, communication.ErrNotFound
}

func (r *fakeRepo) InsertMessage(ctx context.Context, m communication.Message) (communication.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return communication.Message{}, err
	}
	if !r.messagesVisible(ctx, m.ConversationID) {
		return communication.Message{}, errors.New("row-level security")
	}
	m.ID = r.id("f")
	r.msgs[m.ConversationID] = append(r.msgs[m.ConversationID], m)
	return m, nil
}

func (r *fakeRepo) Advance(_ context.Context, id string, seq int64, sent time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.convs[id]
	c.LastSeq, c.LastSentAt = seq, sent
	r.convs[id] = c
	return nil
}

func (r *fakeRepo) MarkRead(_ context.Context, partID string, seq int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, parts := range r.parts {
		for i, p := range parts {
			if p.ID == partID {
				r.parts[id][i].ReadSeq = max(p.ReadSeq, seq)
			}
		}
	}
	return nil
}

func (r *fakeRepo) UpdateBody(_ context.Context, m communication.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, have := range r.msgs[m.ConversationID] {
		if have.ID != m.ID {
			continue
		}
		if have.VersionID != m.VersionID-1 {
			return communication.ErrVersionConflict
		}
		edited := have.EditedAt
		if edited.IsZero() {
			edited = have.SentAt
		}
		r.revs[m.ID] = append(r.revs[m.ID], communication.Revision{EditedAt: edited, Body: have.Body, VersionID: have.VersionID})
		r.msgs[m.ConversationID][i] = m
		return nil
	}
	return communication.ErrNotFound
}

func (r *fakeRepo) Revisions(_ context.Context, _, id string) ([]communication.Revision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.revs[id]), nil
}

// addPractitioner adds role as a participant of the conversation, as SAL-28 will, from history_from_seq.
func (r *fakeRepo) addPractitioner(conv, role string, from int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parts[conv] = append(r.parts[conv], communication.Participant{
		ID: r.id("d"), ConversationID: conv, Role: communication.RolePractitioner, PractitionerRoleID: role, HistoryFromSeq: from,
		OrganizationID: clinicA, Status: communication.StatusActive,
	})
}

func (r *fakeRepo) AddPartitions(_ context.Context, from time.Time, months int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fail(); err != nil {
		return err
	}
	r.partitions = append(r.partitions, fmt.Sprintf("%s/%d", from.Format(time.DateOnly), months))
	return nil
}
