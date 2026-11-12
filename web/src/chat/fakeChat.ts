import { ChatError, type ChatApi, type ChatEnd, type ChatHours, type ChatMessage, type ChatMessageFrame, type Conversation } from './chatApi'

// fakeChat is the ChatApi in memory, for tests and for building the screens on synthetic data (CLAUDE.md "Test
// data is synthetic"). It keeps the API's rules that the screens see: pages newest first by cursor, idempotent
// sends, edits by the sender with the current version, the read position never going back, each frame once.

type Followers = Map<string, Set<(frame: ChatMessageFrame) => void>>

export type FakeChat = ChatApi & {
  // deliver writes a message as someone else (the other participant), as if it came over the socket.
  deliver(id: string, body: string, role?: 'patient' | 'practitioner'): ChatMessage
  // end ends every socket of the conversation with the reason.
  end(id: string, reason: ChatEnd): void
  conversations: Conversation[]
}

const uuid = (n: number, prefix = '0') => `${prefix.padEnd(8, '0')}-0000-4000-8000-${n.toString(16).padStart(12, '0')}`

// cursorOf is the message's cursor (fake-0 is the start's): the fake's own, opaque to the screens as the API's is.
const cursorOf = (seq: number) => `fake-${seq}`
const seqOf = (cursor: string) => Number(cursor.slice('fake-'.length))

export type FakeChatOptions = {
  // me is the caller's role and participant id: their messages are the ones they may edit.
  me?: { role: 'patient' | 'practitioner'; participantId: string }
  // history is how many synthetic messages the first conversation starts with (a long one for the virtualized list).
  history?: number
  start?: Date
  // hours is the clinic's chat hours (SAL-29): by default it states none and is open.
  hours?: ChatHours
}

export function fakeChat({
  me = { role: 'practitioner', participantId: uuid(2, 'd') },
  history = 0,
  start = new Date('2026-11-12T08:00:00Z'),
  hours = { time_zone: 'Europe/Sarajevo', hours: [], response_time_hours: 24, open_now: true },
}: FakeChatOptions = {}): FakeChat {
  const other = me.role === 'patient' ? { role: 'practitioner' as const, participantId: uuid(2, 'd') } : { role: 'patient' as const, participantId: uuid(1, 'd') }
  const messages = new Map<string, ChatMessage[]>()
  const revisions = new Map<string, { edited_at: string; body: string; version_id: number }[]>()
  const followers: Followers = new Map()
  const enders = new Map<string, Set<(end: ChatEnd) => void>>()
  const clientIds = new Map<string, ChatMessage>()
  let clock = start.getTime()
  let ids = 0

  const conversation: Conversation = {
    id: uuid(1, 'c'), organization_id: uuid(1, 'a'), patient_id: uuid(1, 'b'), practitioner_role_id: uuid(1, 'e'),
    created_at: start.toISOString(), last_seq: 0, cursor: cursorOf(0), unread: 0, staff_only: false,
    participants: [
      { id: uuid(1, 'd'), role: 'patient', status: 'active', patient_id: uuid(1, 'b'), added_at: start.toISOString(), history_from_seq: 0 },
      {
        id: uuid(2, 'd'), role: 'practitioner', status: 'active', practitioner_role_id: uuid(1, 'e'), organization_id: uuid(1, 'a'),
        added_at: start.toISOString(), history_from_seq: 0,
      },
    ],
  }
  const conversations = [conversation]
  messages.set(conversation.id, [])

  function find(id: string): Conversation {
    const c = conversations.find((c) => c.id === id)
    if (!c) throw new ChatError('not_found')
    return c
  }

  // readSeq is the caller's read position per conversation: the API never shows it, only the unread count.
  const readSeq = new Map<string, number>()

  function write(c: Conversation, from: { role: 'patient' | 'practitioner'; participantId: string }, body: string, clientId: string): ChatMessage {
    const list = messages.get(c.id) ?? []
    clock += 1000
    const seq = list.length + 1
    const m: ChatMessage = {
      id: uuid(++ids, 'f'), conversation_id: c.id, seq, cursor: cursorOf(seq), sender_participant_id: from.participantId,
      sender_role: from.role, kind: 'text', body, client_message_id: clientId, sent_at: new Date(clock).toISOString(), version_id: 1,
    }
    list.push(m)
    messages.set(c.id, list)
    c.last_seq = seq
    c.last_sent_at = m.sent_at
    c.cursor = m.cursor
    if (from.participantId === me.participantId) readSeq.set(c.id, seq)
    c.unread = c.last_seq - (readSeq.get(c.id) ?? 0)
    for (const f of followers.get(c.id) ?? []) f({ type: 'message', version: 1, payload: m })
    return m
  }

  function body(text: string): string {
    const t = text.trim()
    if (t.length < 1 || t.length > 4000) throw new ChatError('invalid_request')
    return t
  }

  for (let i = 0; i < history; i++) {
    write(conversation, i % 2 ? other : me, `Synthetic placeholder message ${i + 1}`, uuid(i + 1, 'cc'))
  }

  return {
    conversations,
    async list() {
      const sorted = [...conversations].sort((a, b) => (b.last_sent_at ?? b.created_at).localeCompare(a.last_sent_at ?? a.created_at))
      return { conversations: sorted.map((c) => ({ ...c, participants: undefined })) }
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
    async send(id, clientMessageId, text) {
      const c = find(id)
      const repeat = clientIds.get(`${id}/${clientMessageId}`)
      if (repeat) return repeat
      const m = write(c, me, body(text), clientMessageId)
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
      Object.assign(m, { body: body(text), version_id: m.version_id + 1, edited_at: new Date(clock).toISOString() })
      for (const f of followers.get(id) ?? []) f({ type: 'message_edited', version: 1, payload: { ...m } })
      return { ...m }
    },
    async revisions(id, messageId) {
      find(id)
      return [...(revisions.get(messageId) ?? [])]
    },
    async chatHours(organizationId) {
      if (!conversations.some((c) => c.organization_id === organizationId)) throw new ChatError('not_found')
      return { ...hours, hours: hours.hours.map((h) => ({ ...h })) }
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
    deliver(id, text, role = other.role) {
      const from = role === me.role ? me : other
      return write(find(id), from, body(text), uuid(++ids, 'dd'))
    },
    end(id, reason) {
      followers.delete(id)
      for (const e of enders.get(id) ?? []) e(reason)
      enders.delete(id)
    },
  }
}
