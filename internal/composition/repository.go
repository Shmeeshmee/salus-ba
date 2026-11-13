package composition

import (
	"context"
	"io"
	"time"

	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// Repository stores the templates (document_template, document_template_translation) and the drafts
// (document_draft). Row-level security shows a transaction the templates of the clinic set by SetClinic, and
// the drafts of the encounters that scope shows (with an active consent or, after a revocation, for an
// appointment that started before it); SetPatient shows the patient the signed (final) drafts of their own
// encounters, and no other.
type Repository interface {
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetPatient scopes the rest of the transaction in ctx to the patient id's own signed documents.
	SetPatient(ctx context.Context, id string) error
	// Templates returns the templates of the clinic org, by title, each with its translations. It selects by
	// org, so the owner (salusctl), past row-level security, sees no other clinic's.
	Templates(ctx context.Context, org string) ([]Template, error)
	// Template returns the template with its translations, locked until the transaction ends when lock is
	// set, or ErrTemplateNotFound (also outside the scope, or for an id that is no UUID).
	Template(ctx context.Context, id string, lock bool) (Template, error)
	// InsertTemplate stores a new template of the clinic org with its translations, at at, as version 1.
	InsertTemplate(ctx context.Context, org string, c TemplateContent, at time.Time) (Template, error)
	// UpdateTemplate replaces the content of the template id and its translations, at at, as its next version.
	UpdateTemplate(ctx context.Context, id string, c TemplateContent, at time.Time) (Template, error)
	// SaveDraft stores d as the encounter's open (preliminary) draft of its template, at at: a new one, or the
	// open one's fields, language, title and type replaced as its next version. It reports whether it created
	// one.
	SaveDraft(ctx context.Context, d Draft, at time.Time) (Draft, bool, error)
	// Draft returns the draft, locked until the transaction ends when lock is set, or ErrNotFound (also outside
	// the scope, or for an id that is no UUID).
	Draft(ctx context.Context, id string, lock bool) (Draft, error)
	// ByDocumentReference and ByProvenance return the signed draft whose DocumentReference or Provenance has
	// the id, or ErrNotFound.
	ByDocumentReference(ctx context.Context, id string) (Draft, error)
	ByProvenance(ctx context.Context, id string) (Draft, error)
	// Drafts returns the encounter's drafts, oldest first.
	Drafts(ctx context.Context, encounterID string) ([]Draft, error)
	// Signed returns the encounter's final (signed) drafts, in the order they were signed.
	Signed(ctx context.Context, encounterID string) ([]Draft, error)
	// SignedOf returns the final (signed) drafts of the encounters after the page's position (by signature, then
	// "Composition/<id>"), newest first, p.Limit at most.
	SignedOf(ctx context.Context, encounterIDs []string, p keyset.Page) ([]Draft, error)
	// Sign stores d's fields, status final and signature, at at, as the version after d.VersionID.
	Sign(ctx context.Context, d Draft, at time.Time) (Draft, error)
}

// Store holds the signed documents' encrypted PDFs in the object store (binary.Store, ADR 0010), under
// ObjectKey. It never sees a plaintext byte. Calls to it stay outside every transaction.
type Store interface {
	// Put stores exactly size bytes read from r under key.
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	// GetRange returns the bytes from offset from to offset to of the object at key, both inclusive.
	GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error)
	// Delete removes the object at key; a key that holds no object is not an error.
	Delete(ctx context.Context, key string) error
}

// Publisher puts a draft's event on its encounter's Redis Stream (transcript's Publisher: transcript.StreamKey,
// under transcript.StreamField). Errors carry no event.
type Publisher interface {
	Publish(ctx context.Context, encounterID string, envelope []byte) error
}

// Jobs enqueues the background work a signature causes, in the transaction in ctx (River's InsertTx): the
// patient's e-mail that a document is ready (the notifier's document_signed_email). Opaque ids only.
type Jobs interface {
	EnqueueSigned(ctx context.Context, appointmentID, org string) error
}
