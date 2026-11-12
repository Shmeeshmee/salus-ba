package communication_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// sal30Fakes are the ports SAL-30 adds: binary's chat files, the terms' tagger, the callers' languages, and the
// Redis presence.
type sal30Fakes struct {
	files     *fakeFiles
	terms     *fakeTagger
	languages *fakeLanguages
	presence  *fakePresence
}

func newSAL30Fakes(f *fixture, tx communication.Transactor) sal30Fakes {
	return sal30Fakes{
		files: &fakeFiles{tx: tx, rows: map[string]binary.Binary{}}, terms: &fakeTagger{}, languages: &fakeLanguages{by: map[string]string{}},
		presence: &fakePresence{clock: f.clock, receipts: map[string]map[string]int64{}, typed: map[string]time.Time{}},
	}
}

// fakeFiles is binary's chat files in memory: the type from the bytes (a PDF or a PNG), the commit in the upload's
// transaction, the attachment rules; what the scope shows is the chat's own (its conversation was entered).
type fakeFiles struct {
	tx     communication.Transactor
	err    error // the next UploadChat's, before its commit
	rows   map[string]binary.Binary
	opened []string // the ids OpenChat opened, "<id>" or "<id> thumbnail"
	nextID int
	mu     sync.Mutex
}

func (f *fakeFiles) UploadChat(ctx context.Context, in binary.ChatUpload, commit func(ctx context.Context, b binary.Binary) error) (binary.Binary, error) {
	body, err := io.ReadAll(in.Body)
	if err != nil {
		return binary.Binary{}, err
	}
	f.mu.Lock()
	err, f.err = f.err, nil
	f.nextID++
	b := binary.Binary{
		ID: fmt.Sprintf("b%07d-0000-4000-8000-000000000000", f.nextID), OrganizationID: in.OrganizationID, PatientID: in.PatientID,
		ConversationID: in.ConversationID, ParticipantID: in.ParticipantID, Title: in.Title, Size: int64(len(body)), Status: binary.StatusQuarantined,
	}
	f.mu.Unlock()
	switch {
	case err != nil:
		return binary.Binary{}, err
	case bytes.HasPrefix(body, []byte("%PDF")):
		b.ContentType = "application/pdf"
	case bytes.HasPrefix(body, []byte("\x89PNG")):
		b.ContentType = "image/png"
	default:
		return binary.Binary{}, binary.ErrUnsupportedType
	}
	err = f.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := commit(ctx, b); err != nil {
			return err
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.rows[b.ID] = b
		return nil
	})
	return b, err
}

func (f *fakeFiles) AttachChat(_ context.Context, m binary.MessageFiles) ([]binary.Binary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []binary.Binary
	for _, id := range m.IDs {
		b, ok := f.rows[id]
		if !ok || b.ConversationID != m.ConversationID || b.ParticipantID != m.ParticipantID || b.MessageID != "" || b.Status == binary.StatusRejected {
			return nil, binary.ErrNotFound
		}
		b.MessageID, b.MessageSentAt = m.MessageID, m.SentAt
		f.rows[id] = b
		out = append(out, b)
	}
	return out, nil
}

func (f *fakeFiles) ChatFiles(_ context.Context, convs, ids []string) ([]binary.Binary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []binary.Binary
	for _, b := range f.rows {
		if slices.Contains(convs, b.ConversationID) && slices.Contains(ids, b.MessageID) {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b binary.Binary) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *fakeFiles) ChatFile(_ context.Context, conv, id string) (binary.Binary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.rows[id]
	if !ok || b.ConversationID != conv {
		return binary.Binary{}, binary.ErrNotFound
	}
	return b, nil
}

func (f *fakeFiles) OpenChat(_ context.Context, b binary.Binary, thumbnail bool) (binary.File, error) {
	switch {
	case b.Status == binary.StatusQuarantined:
		return binary.File{}, binary.ErrScanning
	case b.Status != binary.StatusClean, thumbnail && b.Thumbnail.ID == "":
		return binary.File{}, binary.ErrNotFound
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	name := b.ID
	if thumbnail {
		name += " thumbnail"
	}
	f.opened = append(f.opened, name)
	return binary.File{Binary: b}, nil
}

func (f *fakeFiles) Content(context.Context, binary.File, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("synthetic")), nil
}

// set sets the status of the file id (the scan's result), with a thumbnail when thumbnail.
func (f *fakeFiles) set(id string, status binary.Status, thumbnail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.rows[id]
	b.Status = status
	if thumbnail {
		b.Thumbnail = binary.Thumbnail{ID: "t" + id[1:], Size: 9}
	}
	f.rows[id] = b
}

// fakeTagger knows two terms: "migraine" (English) and "glavobolja" (Bosnian), and fails when err is set.
type fakeTagger struct {
	err       error
	languages []string // of every text it was given
	mu        sync.Mutex
}

const (
	termMigraine = "10000000-0000-4000-8000-0000000000e1"
	termHeadache = "10000000-0000-4000-8000-0000000000e2"
)

func (t *fakeTagger) Tag(_ context.Context, texts []terminology.Text) ([][]terminology.Span, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err != nil {
		return nil, t.err
	}
	out := make([][]terminology.Span, len(texts))
	for i, x := range texts {
		t.languages = append(t.languages, x.Language)
		for word, term := range map[string]map[string]string{"migraine": {"en": termMigraine}, "glavobolja": {"bs": termHeadache}} {
			if id, ok := term[x.Language]; ok {
				if at := strings.Index(strings.ToLower(x.Body), word); at >= 0 {
					out[i] = append(out[i], terminology.Span{TermID: id, Start: at, End: at + len(word)})
				}
			}
		}
	}
	return out, nil
}

type fakeLanguages struct {
	by map[string]string // by account
	mu sync.Mutex
}

func (l *fakeLanguages) Language(_ context.Context, account string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.by[account], nil
}

// fakePresence is the Redis presence in memory: a participant types once a TypingEvery by the test's clock; receipts
// never go back.
type fakePresence struct {
	err      error
	clock    *testClock
	receipts map[string]map[string]int64
	typed    map[string]time.Time
	mu       sync.Mutex
}

func (p *fakePresence) AllowTyping(_ context.Context, conv, part string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.clock.Now()
	if last, ok := p.typed[conv+part]; ok && now.Sub(last) < communication.TypingEvery {
		return false, p.err
	}
	p.typed[conv+part] = now
	return true, p.err
}

func (p *fakePresence) SetReceipt(_ context.Context, conv, part string, seq int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	if p.receipts[conv] == nil {
		p.receipts[conv] = map[string]int64{}
	}
	p.receipts[conv][part] = max(p.receipts[conv][part], seq)
	return nil
}

func (p *fakePresence) Receipts(_ context.Context, conv string) (map[string]int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	out := map[string]int64{}
	for k, v := range p.receipts[conv] {
		out[k] = v
	}
	return out, nil
}

var errTag = errors.New("the term list is down")
