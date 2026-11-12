// Package binary is a file a patient attached to a questionnaire answer: FHIR R4 Binary
// (docs/attachments/SPEC.md). It owns the object store, the "binary" table and the files' lifecycle: the
// upload pipeline (type from the bytes, envelope encryption, one streamed put), the status read, the ranged
// download and the deletion. Who may upload, read or delete a file is decided by the file's holder, the
// questionnaire response it answers (Holder); binary imports no domain but auditevent. Files are health data:
// every call is audited, refusals included, and no title, byte or content type reaches a log line or an error.
package binary

import (
	"io"
	"time"
)

// Status is where a file is in its malware scan.
type Status string

// The statuses.
const (
	StatusQuarantined Status = "quarantined" // stored, not yet scanned: nobody downloads it
	StatusClean       Status = "clean"       // scanned, may be opened
	StatusRejected    Status = "rejected"    // malware found: its key is gone and it is detached
)

// Kind is what an attachment item accepts (questionnaire.FileKind, by value).
type Kind string

// The kinds and their content types (SPEC.md §3 O1).
const (
	KindPhoto    Kind = "photo"    // image/jpeg, image/png
	KindDocument Kind = "document" // application/pdf
	KindVideo    Kind = "video"    // video/mp4
)

// The limits of SPEC.md §3 O2, in bytes (1 MB is 1,000,000 bytes, as the apps show them).
const (
	MaxPhotoSize    int64 = 20_000_000
	MaxDocumentSize int64 = 20_000_000
	MaxVideoSize    int64 = 200_000_000
	MaxResponseSize int64 = 500_000_000 // every file of one response together
	MaxTitle              = 200         // characters of a title
)

// Limit is the largest file of the kind; 0 for no kind.
func (k Kind) Limit() int64 {
	switch k {
	case KindPhoto:
		return MaxPhotoSize
	case KindDocument:
		return MaxDocumentSize
	case KindVideo:
		return MaxVideoSize
	}
	return 0
}

// kindOf is the kind of a detected content type; "" for none.
func kindOf(contentType string) Kind {
	switch contentType {
	case "image/jpeg", "image/png":
		return KindPhoto
	case "application/pdf":
		return KindDocument
	case "video/mp4":
		return KindVideo
	}
	return ""
}

// Binary is one stored file. OrganizationID, PatientID and AppointmentID are copied once from the holder's
// Target (a chat file's from its conversation). The bytes are in the object store under ID, sealed with a data key
// that WrappedKey holds, wrapped by the key-encryption key KEKID.
//
// A file has one of two parents: a questionnaire response's item (ResponseID, AppointmentID, LinkID), or a
// conversation of the chat (ConversationID, uploaded by the participant ParticipantID and attached to the message
// MessageID once it is sent; SAL-30).
type Binary struct {
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ScannedAt      time.Time // zero while quarantined
	MessageSentAt  time.Time // the message's time, with MessageID
	ID             string    // the Binary id and the object key, a lowercase UUID
	OrganizationID string
	PatientID      string
	AppointmentID  string
	ResponseID     string // the questionnaire response whose answer holds the file
	LinkID         string // the attachment item
	ConversationID string // a chat file's conversation
	ParticipantID  string // the conversation participant who uploaded a chat file
	MessageID      string // the message a chat file is attached to; empty until it is sent
	ContentType    string // detected from the bytes
	Title          string // the file's name as uploaded, at most MaxTitle characters; personal data
	KEKID          string
	Status         Status
	SHA1           []byte    // of the plaintext: Attachment.hash and the identical-retry check
	WrappedKey     []byte    // nil once rejected
	Thumbnail      Thumbnail // a clean chat image's; zero for none
	Size           int64     // plaintext bytes
	SegmentSize    int32
}

// Thumbnail is a chat image's preview, made by the scan job from the clean file: a JPEG stored and sealed like a file,
// under its own id (the object's key) and data key, wrapped by the file's key-encryption key.
type Thumbnail struct {
	ID   string
	Key  []byte // the wrapped data key
	Size int64  // plaintext bytes
}

// MessageFiles names the files a send attaches to its message: the sender's uploads to the conversation.
type MessageFiles struct {
	SentAt         time.Time
	ConversationID string
	ParticipantID  string
	MessageID      string
	IDs            []string
}

// Caller is who is signed in: a patient (PatientID, no clinic) or staff of a clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // staff only
	PractitionerID string // staff only
	PatientID      string // the patient's profile; empty for staff and before the profile exists
	Admin          bool
}

// Target is where an upload may go, as the holder allows it: the response's clinic, patient and appointment,
// and the item's accepted kinds and maximum number of files.
type Target struct {
	OrganizationID string
	PatientID      string
	AppointmentID  string
	Accept         []Kind
	MaxFiles       int
}

// Grant is a holder's permission to read a response's files, with the audit's purpose of use
// (auditevent.CodePATRQT for the patient, auditevent.CodeTREAT for staff).
type Grant struct {
	Purpose string
}

// Attachment is a file as its holder lists it in the answer (questionnaire_response.attachments): what the
// FHIR valueAttachment shows, and whether the file is still pending its scan.
type Attachment struct {
	LinkID      string `json:"linkId"`
	BinaryID    string `json:"binaryId"`
	ContentType string `json:"contentType"`
	Title       string `json:"title"`
	Hash        []byte `json:"hash"` // SHA-1, base64 in JSON as R4's Attachment.hash
	Size        int64  `json:"size"`
	Pending     bool   `json:"pending"`
}

// Upload is one file sent to an attachment item: exactly Size bytes of Body (Size -1 when the request gave no
// length).
type Upload struct {
	Body       io.Reader
	ResponseID string
	LinkID     string
	Title      string
	Size       int64
}

// File is a clean file its caller may read, with its data key, as Download returns it for Content.
type File struct {
	dek []byte
	Binary
}

// QueueScan is the River queue of the scan job (its workers come with package 3.1).
const QueueScan = "scan"

// ScanArgs is the binary_scan job, enqueued with each upload in its transaction. Opaque ids only: the file,
// and its response and clinic, which name the job scope that lets the job read the file's row.
// A chat file's job names its conversation instead of a response.
type ScanArgs struct {
	BinaryID       string `json:"binary_id"`
	ResponseID     string `json:"questionnaire_response_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	OrganizationID string `json:"organization_id"`
}

// ChatScanAttempts is how often a chat file's scan is tried before the worker rejects it as unscannable
// (Service.Unscannable): River's backoff of attempt⁴ seconds makes it about 80 minutes. A questionnaire file's keeps
// River's default and stays quarantined after the last.
const ChatScanAttempts = 8

// Kind names the job in river_job.
func (ScanArgs) Kind() string { return "binary_scan" }

// DiscardArgs is the binary_discard job, enqueued by the holder when it no longer lists some of a response's
// files (a withdrawal, the hidden items of a submit). Opaque ids only.
type DiscardArgs struct {
	ResponseID     string `json:"questionnaire_response_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (DiscardArgs) Kind() string { return "binary_discard" }
