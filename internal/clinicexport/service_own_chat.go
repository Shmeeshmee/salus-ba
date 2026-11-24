package clinicexport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// conversation is one of the patient's conversations with its messages, oldest first.
type conversation struct {
	files    map[string]string // the archive's file of each attachment the patient may open, by attachment id
	messages []communication.Message
	communication.Conversation
}

// chat reads, as the patient c, their conversations (of org alone when it is set) with every message and the files
// they may open in the app, into out: the same reads the app makes, each audited by the chat's service.
func (o *OwnRecords) chat(ctx context.Context, c appointment.Caller, org string, out *own) error {
	var convs []communication.Conversation
	for before := (*communication.ListCursor)(nil); ; {
		page, next, err := o.p.Chat.List(ctx, c, before)
		if err != nil {
			return err
		}
		convs = append(convs, page...)
		if next == nil {
			break
		}
		before = next
	}
	for _, k := range convs {
		if org != "" && k.OrganizationID != org {
			continue // a clinic target reads that clinic's conversations only
		}
		conv, err := o.conversation(ctx, c, k, out)
		if err != nil {
			return err
		}
		out.chats = append(out.chats, conv)
	}
	return nil
}

// conversation reads k's messages, page by page, and the files of them the patient may open.
func (o *OwnRecords) conversation(ctx context.Context, c appointment.Caller, k communication.Conversation, out *own) (conversation, error) {
	conv := conversation{Conversation: k, files: map[string]string{}}
	for before := (*communication.Cursor)(nil); ; {
		page, err := o.p.Chat.Messages(ctx, c, k.ID, before, communication.MaxPageSize)
		if err != nil {
			return conversation{}, err
		}
		conv.messages = append(conv.messages, page.Messages...)
		if page.Next == nil {
			break
		}
		before = page.Next
	}
	slices.Reverse(conv.messages) // the service gives the newest first
	for _, m := range conv.messages {
		for _, a := range m.Attachments {
			body, err := readAll(o.chatFile(ctx, c, k.ID, a.ID))
			switch {
			case errors.Is(err, communication.ErrNotFound), errors.Is(err, binary.ErrScanning):
				continue // the app cannot open it either
			case err != nil:
				return conversation{}, err
			}
			name := fileName(a.ID, a.ContentType)
			conv.files[a.ID] = name
			out.add(k.OrganizationID, OwnFile{Name: name, Body: body})
			out.chatFiles[k.OrganizationID]++
		}
	}
	return conv, nil
}

// chatFile opens the conversation id's attachment for the patient c and streams its bytes.
func (o *OwnRecords) chatFile(ctx context.Context, c appointment.Caller, id, attachmentID string) (io.ReadCloser, error) {
	f, err := o.p.Chat.Attachment(ctx, c, id, attachmentID, false)
	if err != nil {
		return nil, err
	}
	if f.Size == 0 {
		return io.NopCloser(strings.NewReader("")), nil
	}
	return o.p.Chat.Content(ctx, f, 0, f.Size-1)
}

// chatPart is a part's chat.json and how many conversations it holds.
type chatPart struct {
	body []byte
	n    int
}

// chatJSON is chat.json of the clinic org: each conversation with its messages, oldest first, as the patient reads
// them in the app (the sender's role, not their name; the text; the files by their place in the archive).
func (x own) chatJSON(org string) (chatPart, error) {
	type file struct {
		File        string `json:"file,omitempty"`
		Note        string `json:"note,omitempty"`
		ContentType string `json:"content_type"`
		Name        string `json:"name,omitempty"`
	}
	type message struct {
		SentAt     time.Time  `json:"sent_at"`
		EditedAt   *time.Time `json:"edited_at,omitempty"`
		From       string     `json:"from"`
		Kind       string     `json:"kind"`
		Text       string     `json:"text,omitempty"`
		SystemCode string     `json:"system_code,omitempty"`
		Files      []file     `json:"files,omitempty"`
	}
	type conv struct {
		Started      time.Time `json:"started"`
		Conversation string    `json:"conversation"`
		Messages     []message `json:"messages"`
	}
	var out []conv
	for _, k := range x.chats {
		if k.OrganizationID != org {
			continue
		}
		c := conv{Conversation: k.ID, Started: k.Created, Messages: []message{}}
		for _, m := range k.messages {
			msg := message{SentAt: m.SentAt, From: string(m.SenderRole), Kind: string(m.Kind), Text: m.Body, SystemCode: m.SystemCode}
			if !m.EditedAt.IsZero() {
				at := m.EditedAt
				msg.EditedAt = &at
			}
			for _, a := range m.Attachments {
				f := file{File: k.files[a.ID], ContentType: a.ContentType, Name: a.Filename}
				if f.File == "" {
					f.Note = "not in this archive: the app cannot open it"
				}
				msg.Files = append(msg.Files, f)
			}
			c.Messages = append(c.Messages, msg)
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return chatPart{}, nil
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return chatPart{}, err
	}
	return chatPart{body: append(body, '\n'), n: len(out)}, nil
}
