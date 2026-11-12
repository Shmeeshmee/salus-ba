import {
  ChatError,
  reasons,
  type ChatApi,
  type ChatAttachment,
  type ChatEnd,
  type ChatMessage,
  type ChatMessageFrame,
  type Conversation,
  type ConversationParticipant,
} from './chatApi'

// fakeChat is the ChatApi in memory, for tests and for building the screens on synthetic data (CLAUDE.md "Test
// data is synthetic"). It keeps the API's rules that the screens see: pages newest first by cursor, idempotent
// sends, edits by the sender with the current version, the read position never going back, each frame once, the
// summary only for a practitioner who reads 50 messages, system messages for added and removed doctors, threads
// never the patient's, attachments scanning until the test scans them.

type Who = { role: 'patient' | 'practitioner'; participantId: string }

export type FakeChat = ChatApi & {
  // deliver writes a message as someone else (the other participant), as if it came over the socket.
  deliver(id: string, body: string, role?: 'patient' | 'practitioner'): ChatMessage
  // end ends every socket of the conversation with the reason.
  end(id: string, reason: ChatEnd): void
  // scan gives an uploaded attachment the scan's outcome; the messages carrying it go out as message_edited.
  scan(attachmentId: string, status: 'clean' | 'rejected'): void
  conversations: Conversation[]
}

const uuid = (n: number, prefix = '0') => `${prefix.padEnd(8, '0')}-0000-4000-8000-${n.toString(16).padStart(12, '0')}`

// cursorOf is the message's cursor (fake-0 is the start's): the fake's own, opaque to the screens as the API's is.
const cursorOf = (seq: number) => `fake-${seq}`
const seqOf = (cursor: string) => Number(cursor.slice('fake-'.length))

// summaryMinimum is the API's "long conversation": the summary needs 50 messages the practitioner reads.
const summaryMinimum = 50

export type FakeChatOptions = {
  // me is the caller's role and participant id: their messages are the ones they may edit.
  me?: Who
  // history is how many synthetic messages the first conversation starts with (a long one for the virtualized list).
  history?: number
  start?: Date
}

export function fakeChat({ me = { role: 'practitioner', participantId: uuid(2, 'd') }, history = 0, start = new Date('2026-11-12T08:00:00Z') }: FakeChatOptions = {}): FakeChat {
  const other: Who = me.role === 'patient' ? { role: 'practitioner', participantId: uuid(2, 'd') } : { role: 'patient', participantId: uuid(1, 'd') }
  const messages = new Map<string, ChatMessage[]>()
  const revisions = new Map<string, { edited_at: string; body: string; version_id: number }[]>()
  const followers = new Map<string, Set<(frame: ChatMessageFrame) => void>>()
  const enders = new Map<string, Set<(end: ChatEnd) => void>>()
  const clientIds = new Map<string, ChatMessage>()
  const uploads = new Map<string, ChatAttachment>()
  let clock = start.getTime()
  let ids = 0
  const now = () => new Date(clock).toISOString()

  const conversation: Conversation = {
    id: uuid(1, 'c'), organization_id: uuid(1, 'a'), patient_id: uuid(1, 'b'), practitioner_role_id: uuid(1, 'e'),
    created_at: start.toISOString(), last_seq: 0, cursor: cursorOf(0), unread: 0, staff_only: false,
    participants: [
      { id: uuid(1, 'd'), role: 'patient', patient_id: uuid(1, 'b'), added_at: start.toISOString(), history_from_seq: 0 },
      { id: uuid(2, 'd'), role: 'practitioner', practitioner_role_id: uuid(1, 'e'), organization_id: uuid(1, 'a'), status: 'active', added_at: start.toISOString(), history_from_seq: 0 },
    ],
  }
  const conversations = [conversation]
  messages.set(conversation.id, [])

  function find(id: string): Conversation {
    const c = conversations.find((c) => c.id === id)
    if (!c) throw new ChatError('not_found')
    return c
  }

  function practitionerOnly() {
    if (me.role !== 'practitioner') throw new ChatError('forbidden')
  }

  // readSeq is the caller's read position per conversation: the API never shows it, only the unread count.
  const readSeq = new Map<string, number>()

  function emit(id: string, frame: ChatMessageFrame) {
    for (const f of followers.get(id) ?? []) f(frame)
  }

  function write(c: Conversation, from: Who | undefined, fields: Pick<ChatMessage, 'kind' | 'body' | 'system' | 'attachments'>, clientId: string): ChatMessage {
    const list = messages.get(c.id) ?? []
    clock += 1000
    const seq = list.length + 1
    const m: ChatMessage = {
      id: uuid(++ids, 'f'), conversation_id: c.id, seq, cursor: cursorOf(seq), sender_participant_id: from?.participantId,
      sender_role: from?.role ?? 'system', client_message_id: clientId, sent_at: now(), version_id: 1, ...fields,
    }
    if (!from) delete m.sender_participant_id
    list.push(m)
    messages.set(c.id, list)
    c.last_seq = seq
    c.last_sent_at = m.sent_at
    c.cursor = m.cursor
    if (from?.participantId === me.participantId) readSeq.set(c.id, seq)
    c.unread = c.last_seq - (readSeq.get(c.id) ?? 0)
    emit(c.id, { type: 'message', version: 1, payload: m })
    return m
  }

  function system(c: Conversation, code: string, args: Record<string, string>) {
    write(c, undefined, { kind: 'system', system: { code, args } }, uuid(++ids, 'ee'))
  }

  function body(text: string, attachments: ChatAttachment[] = []): string | undefined {
    const t = text.trim()
    if (t.length === 0 && attachments.length > 0) return undefined
    if (t.length < 1 || t.length > 4000) throw new ChatError('invalid_request')
    return t
  }

  function attachmentsOf(ids: string[] = []): ChatAttachment[] {
    return ids.map((id) => {
      const a = uploads.get(id)
      if (!a) throw new ChatError('invalid_request')
      return a
    })
  }

  for (let i = 0; i < history; i++) {
    write(conversation, i % 2 ? other : me, { kind: 'text', body: `Synthetic placeholder message ${i + 1}` }, uuid(i + 1, 'cc'))
  }

  return {
    conversations,
    async list() {
      const sorted = [...conversations].sort((a, b) => (b.last_sent_at ?? b.created_at).localeCompare(a.last_sent_at ?? a.created_at))
      return { conversations: sorted.filter((c) => me.role === 'practitioner' || !c.staff_only).map((c) => ({ ...c, participants: undefined })) }
    },
    async open() {
      return conversation
    },
    async get(id) {
      return find(id)
    },
    async messages(id, { before, limit = 50 } = {}) {
      if (limit < 1 || limit > 100) throw new ChatError('invalid_request')
      const list = messages.get(find(id).id) ?? []
      const end = before === undefined ? list.length : seqOf(before) - 1
      const page = list.slice(Math.max(0, end - limit), end).reverse()
      const oldest = page.at(-1)
      return oldest && oldest.seq > 1 ? { messages: page, next: oldest.cursor } : { messages: page }
    },
    async send(id, clientMessageId, text, attachmentIds) {
      const c = find(id)
      const repeat = clientIds.get(`${id}/${clientMessageId}`)
      if (repeat) return repeat
      const attachments = attachmentsOf(attachmentIds)
      const fields = attachments.length > 0 ? { attachments: attachments.map((a) => ({ ...a })) } : {}
      const m = write(c, me, { kind: 'text', body: body(text, attachments), ...fields }, clientMessageId)
      if (m.body === undefined) delete m.body
      clientIds.set(`${id}/${clientMessageId}`, m)
      return m
    },
    async edit(id, messageId, text, version) {
      const m = (messages.get(find(id).id) ?? []).find((m) => m.id === messageId)
      if (!m) throw new ChatError('not_found')
      if (m.sender_participant_id !== me.participantId || m.kind !== 'text') throw new ChatError('forbidden')
      if (m.version_id !== version) throw new ChatError('version_conflict')
      const earlier = revisions.get(m.id) ?? []
      earlier.push({ edited_at: m.edited_at ?? m.sent_at, body: m.body ?? '', version_id: m.version_id })
      revisions.set(m.id, earlier)
      clock += 1000
      Object.assign(m, { body: body(text), version_id: m.version_id + 1, edited_at: now() })
      emit(id, { type: 'message_edited', version: 1, payload: { ...m } })
      return { ...m }
    },
    async revisions(id, messageId) {
      find(id)
      return [...(revisions.get(messageId) ?? [])]
    },
    async markRead(id, seq) {
      const c = find(id)
      if (seq < 0 || seq > c.last_seq) throw new ChatError('invalid_request')
      readSeq.set(c.id, Math.max(readSeq.get(c.id) ?? 0, seq))
      c.unread = c.last_seq - (readSeq.get(c.id) ?? 0)
    },
    follow(id, after, onFrame, onEnd) {
      const list = messages.get(find(id).id) ?? []
      let last = after === undefined ? list.length : seqOf(after)
      const forward = (frame: ChatMessageFrame) => {
        if (frame.type === 'message' && frame.payload.seq <= last) return
        if (frame.type === 'message') last = frame.payload.seq
        onFrame(frame)
      }
      for (const m of list.slice(last)) forward({ type: 'message', version: 1, payload: m })
      const fs = followers.get(id) ?? new Set()
      fs.add(forward)
      followers.set(id, fs)
      const es = enders.get(id) ?? new Set()
      es.add(onEnd)
      enders.set(id, es)
      return () => {
        fs.delete(forward)
        es.delete(onEnd)
      }
    },
    async summarize(id, lang) {
      practitionerOnly()
      const c = find(id)
      const mine = c.participants?.find((p) => p.id === me.participantId)
      const read = c.last_seq - (mine?.history_from_seq ?? 0)
      if (read < summaryMinimum) throw new ChatError('invalid_request')
      return { summary: `[${lang}] Placeholder summary of ${Math.min(read, 500)} messages`, generated_at: now(), up_to_seq: c.last_seq, ai_generated: true }
    },
    async addParticipant(id, req) {
      practitionerOnly()
      const c = find(id)
      const known = c.participants?.some((p) => p.practitioner_role_id === req.practitioner_role_id)
      const from = req.history === 'from' ? req.from_seq : 0
      if (known || !reasons.includes(req.reason) || from === undefined || from < 0 || from > c.last_seq + 1) throw new ChatError('invalid_request')
      const status = req.organization_id === c.organization_id ? 'active' : 'pending_consent'
      const p: ConversationParticipant = {
        id: uuid(++ids, 'da'), role: 'practitioner', practitioner_role_id: req.practitioner_role_id, organization_id: req.organization_id,
        status, added_at: now(), history_from_seq: Math.max(0, from - 1),
      }
      c.participants = [...(c.participants ?? []), p]
      const args = { practitioner_role_id: req.practitioner_role_id, organization_id: req.organization_id, reason: req.reason }
      system(c, status === 'active' ? 'participant_added' : 'participant_invited', args)
      return p
    },
    async removeParticipant(id, participantId) {
      practitionerOnly()
      const c = find(id)
      const p = c.participants?.find((p) => p.id === participantId)
      if (!p) throw new ChatError('not_found')
      if (p.role === 'patient' || p.id === me.participantId) throw new ChatError('forbidden')
      c.participants = c.participants?.filter((x) => x.id !== participantId)
      system(c, 'participant_removed', { practitioner_role_id: p.practitioner_role_id ?? '' })
    },
    async threads(id) {
      practitionerOnly()
      find(id)
      return conversations.filter((c) => c.parent_id === id).map((c) => ({ ...c, participants: undefined }))
    },
    async openThread(id) {
      practitionerOnly()
      const parent = find(id)
      const thread: Conversation = {
        ...parent, id: uuid(++ids, 'ca'), parent_id: parent.id, staff_only: true, created_at: now(), last_seq: 0, cursor: cursorOf(0), unread: 0,
        last_sent_at: undefined, participants: parent.participants?.filter((p) => p.role === 'practitioner' && p.status !== 'pending_consent'),
      }
      conversations.push(thread)
      messages.set(thread.id, [])
      return thread
    },
    async upload(id, file) {
      find(id)
      const a: ChatAttachment = { id: uuid(++ids, 'ab'), content_type: file.type, size: file.size, filename: file.name, status: 'scanning' }
      uploads.set(a.id, a)
      return { ...a }
    },
    scan(attachmentId, status) {
      const a = uploads.get(attachmentId)
      if (!a) return
      Object.assign(a, { status, thumbnail: status === 'clean' && a.content_type.startsWith('image/') })
      for (const [cid, list] of messages) {
        for (const m of list) {
          const at = m.attachments?.findIndex((x) => x.id === attachmentId) ?? -1
          if (at < 0 || !m.attachments) continue
          m.attachments = m.attachments.map((x, i) => (i === at ? { ...a } : x))
          emit(cid, { type: 'message_edited', version: 1, payload: { ...m } })
        }
      }
    },
    deliver(id, text, role = other.role) {
      const from = role === me.role ? me : other
      return write(find(id), from, { kind: 'text', body: body(text) }, uuid(++ids, 'dd'))
    },
    end(id, reason) {
      followers.delete(id)
      for (const e of enders.get(id) ?? []) e(reason)
      enders.delete(id)
    },
  }
}
