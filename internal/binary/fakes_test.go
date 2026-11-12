package binary_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// Synthetic ids.
const (
	clinicA   = "a0000000-0000-4000-8000-00000000000a"
	patientA  = "aa000000-0000-4000-8000-0000000000aa"
	patientB  = "bb000000-0000-4000-8000-0000000000bb"
	accountA  = "acc00000-0000-4000-8000-0000000000aa"
	practAcc  = "ac100000-0000-4000-8000-0000000000a1"
	practA1   = "p1000000-0000-4000-8000-0000000000a1"
	apptA     = "e1000000-0000-4000-8000-0000000000a1"
	responseA = "d0000000-0000-4000-8000-0000000000d1"
	unknownID = "00000000-0000-4000-8000-000000000000"
	// title is a synthetic file name, asserted absent from every log line.
	title = "Synthetic scan of a placeholder.pdf"
)

var (
	now        = time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	me         = binary.Caller{AccountID: accountA, PatientID: patientA}
	pract      = binary.Caller{AccountID: practAcc, OrganizationID: clinicA, PractitionerID: practA1}
	errBackend = errors.New("the database is down")
	allKinds   = []binary.Kind{binary.KindPhoto, binary.KindDocument, binary.KindVideo}
)

type txKey struct{}

func inTx(ctx context.Context) error {
	if _, ok := postgres.Tx(ctx); ok {
		return nil
	}
	if ctx.Value(txKey{}) == nil {
		return errors.New("called outside a transaction")
	}
	return nil
}

// fakeTransactor runs fn in a "transaction"; when fn fails it drops what was written in it, as a rollback
// does. One transaction at a time.
type fakeTransactor struct{ f *fixture }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	f := t.f
	f.mu.Lock()
	defer f.mu.Unlock()
	events, files, attached, jobs := len(f.audit.events), maps.Clone(f.repo.files), slices.Clone(f.holder.attached), len(f.jobs.scans)
	f.repo.patient, f.repo.clinic, f.repo.job, f.repo.conversation, f.repo.chatJob = "", "", "", "", ""
	f.txs++
	err := fn(context.WithValue(ctx, txKey{}, true))
	if err != nil {
		f.audit.events, f.repo.files, f.holder.attached, f.jobs.scans = f.audit.events[:events], files, attached, f.jobs.scans[:jobs]
	}
	return err
}

type fakeAudit struct {
	events []auditevent.AuditEvent
	mu     sync.Mutex
}

func (a *fakeAudit) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil { // as a query on pgx fails on a cancelled context
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, e)
	return nil
}

// fakeRepository holds the rows and shows them as row-level security does, roughly: the patient's own, or
// the clinic's. The service must not rest on it alone.
type fakeRepository struct {
	err          error
	files        map[string]binary.Binary
	patient      string
	clinic       string
	job          string // the job scope's response; clinic is its clinic
	conversation string // app.conversation_id, as communication sets it for a chat file
	chatJob      string // the chat job scope's conversation; clinic is its clinic
	locked       []string
}

func (r *fakeRepository) visible(b binary.Binary) bool {
	if b.ConversationID != "" {
		switch {
		case r.chatJob != "":
			return b.ConversationID == r.chatJob && b.OrganizationID == r.clinic
		case r.patient != "":
			return b.PatientID == r.patient
		}
		return r.clinic != "" && r.conversation == b.ConversationID
	}
	switch {
	case r.patient != "":
		return b.PatientID == r.patient
	case r.job != "":
		return b.ResponseID == r.job && b.OrganizationID == r.clinic
	}
	return r.clinic != "" && b.OrganizationID == r.clinic
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	r.patient = id
	return inTx(ctx)
}

func (r *fakeRepository) SetClinic(ctx context.Context, id string) error {
	r.clinic = id
	return inTx(ctx)
}

func (r *fakeRepository) SetJob(ctx context.Context, responseID, org string) error {
	r.job, r.clinic = responseID, org
	return inTx(ctx)
}

func (r *fakeRepository) LockResponse(ctx context.Context, id string) error {
	r.locked = append(r.locked, id)
	return inTx(ctx)
}

func (r *fakeRepository) Insert(ctx context.Context, b binary.Binary, at time.Time) (binary.Binary, error) {
	if err := inTx(ctx); err != nil {
		return binary.Binary{}, err
	}
	if r.err != nil {
		return binary.Binary{}, r.err
	}
	switch {
	case b.ConversationID != "" && r.conversation != b.ConversationID:
		return binary.Binary{}, errors.New("row-level security: a chat file goes into the conversation set")
	case b.ConversationID == "" && (r.patient == "" || b.PatientID != r.patient):
		return binary.Binary{}, errors.New("row-level security: the patient inserts only for themselves")
	}
	b.Status, b.CreatedAt, b.UpdatedAt = binary.StatusQuarantined, at, at
	r.files[b.ID] = b
	return b, nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (binary.Binary, error) {
	if err := inTx(ctx); err != nil {
		return binary.Binary{}, err
	}
	if r.err != nil {
		return binary.Binary{}, r.err
	}
	b, ok := r.files[strings.ToLower(id)] // as Postgres reads a uuid in any case
	if !ok || !r.visible(b) {
		return binary.Binary{}, binary.ErrNotFound
	}
	return b, nil
}

func (r *fakeRepository) OfItem(ctx context.Context, responseID, linkID string) ([]binary.Binary, error) {
	var out []binary.Binary
	for _, b := range r.files {
		if r.visible(b) && b.ResponseID == responseID && b.LinkID == linkID && b.Status != binary.StatusRejected {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b binary.Binary) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, inTx(ctx)
}

func (r *fakeRepository) ResponseSize(ctx context.Context, responseID string) (int64, error) {
	var n int64
	for _, b := range r.files {
		if r.visible(b) && b.ResponseID == responseID && b.Status != binary.StatusRejected {
			n += b.Size
		}
	}
	return n, inTx(ctx)
}

func (r *fakeRepository) Delete(ctx context.Context, id string) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	b, ok := r.files[id]
	if !ok || (r.patient == "" && r.job == "") || !r.visible(b) {
		return false, nil
	}
	delete(r.files, id)
	return true, nil
}

func (r *fakeRepository) MarkScanned(ctx context.Context, id string, status binary.Status, at time.Time) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	if r.err != nil {
		return false, r.err
	}
	b, ok := r.files[id]
	if !ok || (r.job == "" && r.chatJob == "") || !r.visible(b) || b.Status != binary.StatusQuarantined {
		return false, nil
	}
	b.Status, b.ScannedAt, b.UpdatedAt = status, at, at
	if status == binary.StatusRejected {
		b.WrappedKey = nil
	}
	r.files[id] = b
	return true, nil
}

func (r *fakeRepository) OfResponse(ctx context.Context, responseID string) ([]binary.Binary, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	var out []binary.Binary
	for _, b := range r.files {
		if r.visible(b) && b.ResponseID == responseID {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b binary.Binary) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (r *fakeRepository) SetChatJob(ctx context.Context, conversationID, org string) error {
	r.chatJob, r.clinic = conversationID, org
	return inTx(ctx)
}

func (r *fakeRepository) AttachToMessage(ctx context.Context, m binary.MessageFiles, at time.Time) ([]binary.Binary, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	var out []binary.Binary
	for _, id := range m.IDs {
		b, ok := r.files[id]
		if !ok || !r.visible(b) || b.ConversationID != m.ConversationID || b.ParticipantID != m.ParticipantID || b.MessageID != "" ||
			b.Status == binary.StatusRejected {
			continue
		}
		b.MessageID, b.MessageSentAt, b.UpdatedAt = m.MessageID, m.SentAt, at
		r.files[id] = b
		out = append(out, b)
	}
	return out, nil
}

func (r *fakeRepository) OfMessages(ctx context.Context, conversationIDs, messageIDs []string) ([]binary.Binary, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	var out []binary.Binary
	for _, b := range r.files {
		if r.visible(b) && slices.Contains(conversationIDs, b.ConversationID) && slices.Contains(messageIDs, b.MessageID) {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b binary.Binary) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, r.err
}

func (r *fakeRepository) SetThumbnail(ctx context.Context, id string, t binary.Thumbnail, at time.Time) (bool, error) {
	if err := inTx(ctx); err != nil {
		return false, err
	}
	b, ok := r.files[id]
	if !ok || r.chatJob == "" || !r.visible(b) || b.Status != binary.StatusClean || b.Thumbnail.ID != "" {
		return false, nil
	}
	b.Thumbnail, b.UpdatedAt = t, at
	r.files[id] = b
	return true, nil
}

func (r *fakeRepository) DeleteUnattachedBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("the purger is tested on the real database")
}

func (r *fakeRepository) DeleteRejectedBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("the purger is tested on the real database")
}

func (r *fakeRepository) Existing(context.Context, []string) ([]string, error) {
	return nil, errors.New("the purger is tested on the real database")
}

// fakeHolder is the questionnaire response: each check answers what it is set to, and the attachment list is
// kept as the real holder keeps it.
type fakeHolder struct {
	uploadRefusal  error
	secondRefusal  error // UploadTarget's answer from its second call on (the upload's commit)
	editRefusal    error
	readRefusal    error
	attachErr      error
	readResponseID string
	attached       []binary.Attachment
	readCallers    []binary.Caller
	editCallers    []binary.Caller
	uploadCallers  []binary.Caller
	detached       []string
	cleared        []string
	target         binary.Target
	uploadTargets  int
}

func (h *fakeHolder) UploadTarget(ctx context.Context, c binary.Caller, responseID, linkID string) (binary.Target, error) {
	if err := inTx(ctx); err != nil {
		return binary.Target{}, err
	}
	h.uploadTargets++
	h.uploadCallers = append(h.uploadCallers, c)
	switch {
	case h.uploadRefusal != nil:
		return binary.Target{}, h.uploadRefusal
	case h.uploadTargets > 1 && h.secondRefusal != nil:
		return binary.Target{}, h.secondRefusal
	case responseID != responseA || linkID == "":
		return binary.Target{}, binary.ErrNotFound
	}
	return h.target, nil
}

func (h *fakeHolder) Editable(ctx context.Context, c binary.Caller, responseID string) error {
	h.editCallers = append(h.editCallers, c)
	if err := inTx(ctx); err != nil {
		return err
	}
	return h.editRefusal
}

func (h *fakeHolder) Readable(ctx context.Context, c binary.Caller, responseID string) (binary.Grant, error) {
	if err := inTx(ctx); err != nil {
		return binary.Grant{}, err
	}
	h.readCallers, h.readResponseID = append(h.readCallers, c), responseID
	if h.readRefusal != nil {
		return binary.Grant{}, h.readRefusal
	}
	if c.OrganizationID != "" {
		return binary.Grant{Purpose: auditevent.CodeTREAT}, nil
	}
	return binary.Grant{Purpose: auditevent.CodePATRQT}, nil
}

func (h *fakeHolder) Attach(ctx context.Context, responseID string, a binary.Attachment) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	if h.attachErr != nil {
		return h.attachErr
	}
	h.attached = append(h.attached, a)
	return nil
}

func (h *fakeHolder) Detach(ctx context.Context, responseID, binaryID string) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	h.detached = append(h.detached, binaryID)
	h.attached = slices.DeleteFunc(h.attached, func(a binary.Attachment) bool { return a.BinaryID == binaryID })
	return nil
}

func (h *fakeHolder) Cleared(ctx context.Context, responseID, binaryID string) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	h.cleared = append(h.cleared, binaryID)
	for i, a := range h.attached {
		if a.BinaryID == binaryID {
			h.attached[i].Pending = false
		}
	}
	return nil
}

func (h *fakeHolder) Attached(ctx context.Context, responseID string) ([]string, error) {
	if err := inTx(ctx); err != nil {
		return nil, err
	}
	var out []string
	for _, a := range h.attached {
		out = append(out, a.BinaryID)
	}
	return out, nil
}

type fakeJobs struct{ scans []binary.ScanArgs }

func (j *fakeJobs) EnqueueScan(ctx context.Context, a binary.ScanArgs) error {
	if err := inTx(ctx); err != nil {
		return err
	}
	j.scans = append(j.scans, a)
	return nil
}

// failingStore fails its puts, and counts the deletes, around the real store.
type failingStore struct {
	binary.Store
	putErr   error
	afterPut func() // called once a put succeeded
	puts     []string
	deleted  []string
	mu       sync.Mutex
}

func (s *failingStore) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	s.mu.Lock()
	s.puts = append(s.puts, key)
	s.mu.Unlock()
	if s.putErr != nil {
		return s.putErr
	}
	if err := s.Store.Put(ctx, key, r, size); err != nil {
		return err
	}
	if s.afterPut != nil {
		s.afterPut()
	}
	return nil
}

func (s *failingStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	s.deleted = append(s.deleted, key)
	s.mu.Unlock()
	return s.Store.Delete(ctx, key)
}

// fixture is a Service on fakes and the real object store: response A of patient A in clinic A, whose item
// takes every kind, 3 files.
type fixture struct {
	svc    *binary.Service
	repo   *fakeRepository
	audit  *fakeAudit
	holder *fakeHolder
	jobs   *fakeJobs
	store  *failingStore
	kek    *envelope.KEK
	mu     sync.Mutex
	txs    int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	kek, err := envelope.NewKEK("test-1", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		repo: &fakeRepository{files: map[string]binary.Binary{}}, audit: &fakeAudit{}, jobs: &fakeJobs{}, kek: kek,
		holder: &fakeHolder{target: binary.Target{OrganizationID: clinicA, PatientID: patientA, AppointmentID: apptA, Accept: allKinds, MaxFiles: 3}},
		store:  &failingStore{Store: newStore(t, 1000)},
	}
	f.svc = binary.NewService(binary.Ports{
		Repo: f.repo, Store: f.store, Holder: f.holder, Audit: f.audit, Tx: fakeTransactor{f: f}, Jobs: f.jobs,
		Clock: clock.Fixed{T: now}, KEK: kek,
	})
	t.Cleanup(func() {
		for _, id := range f.store.puts {
			_ = f.store.Store.Delete(context.Background(), id)
		}
	})
	return f
}

// upload sends body to item "files" of response A as me.
func (f *fixture) upload(t *testing.T, body []byte) (binary.Binary, bool, error) {
	t.Helper()
	return f.svc.Upload(t.Context(), me, binary.Upload{
		Body: bytes.NewReader(body), ResponseID: responseA, LinkID: "files", Title: title, Size: int64(len(body)),
	})
}

// event is an audit event the service writes: interaction by c with outcome on entity ("" for none), in the
// file's clinic A.
func event(c binary.Caller, interaction, purpose, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return eventIn(c, clinicA, interaction, purpose, entity, outcome)
}

// eventIn is event in the clinic org: the caller's own when the call never learnt the file's.
func eventIn(c binary.Caller, org, interaction, purpose, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeRead: auditevent.ActionRead,
		auditevent.CodeDelete: auditevent.ActionDelete,
	}[interaction]
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:          action,
		Outcome:         outcome,
		AgentAccountID:  c.AccountID,
		OrganizationID:  org,
		EntityReference: entity,
	}
}

// events is the pair of events of one call on a file: Binary/<id> and QuestionnaireResponse/<response A>.
func events(c binary.Caller, interaction, purpose, id string, outcome auditevent.Outcome) []auditevent.AuditEvent {
	return []auditevent.AuditEvent{
		event(c, interaction, purpose, "Binary/"+id, outcome),
		event(c, interaction, purpose, "QuestionnaireResponse/"+responseA, outcome),
	}
}
