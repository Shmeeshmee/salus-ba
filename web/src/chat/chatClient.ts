import type { ApiClient } from '../api/client'
import { attempt } from '../api/result'
import { ChatError, type ChatApi, type ChatEnd, type ChatMessage, type ChatMessageFrame, type Conversation, type ConversationParticipant, type ChatAttachment } from './chatApi'

// chatClient is the ChatApi on the JSON API (api/client.ts: same origin, the session cookie, X-Salus-CSRF on every
// write) and the conversation's WebSocket, GET /conversations/{id}/events (SAL-27). Nothing here logs, and no
// message text leaves the page but to the API.

// retry is how long to wait before the next attempt to connect, by failures in a row (as live.ts).
const retry = [1000, 2000, 5000, 10_000]

// ends are the close codes and reasons after which the socket does not come back: access or the session ended.
const ends: Record<string, ChatEnd> = {
  '4403 consent_required': 'consent_required',
  '4403 forbidden': 'forbidden',
  '4404 not_found': 'not_found',
  '4401 unauthenticated': 'unauthenticated',
}

// Wave is the API client without its route types, for the routes of SAL-28 and SAL-30 that are not in the
// generated schema yet (the plan's "Contract of wave 1"); the same client, so the same origin and CSRF header.
// These calls move to the typed routes once SAL-28 to SAL-30 are merged and `make gen-clients` has them.
type Init = { params?: { path?: Record<string, string> }; body?: unknown; headers?: Record<string, string>; bodySerializer?: (body: unknown) => BodyInit }
type Answer = Promise<{ data?: unknown; error?: unknown; response: Response }>
type Wave = { GET(path: string, init?: Init): Answer; POST(path: string, init?: Init): Answer; DELETE(path: string, init?: Init): Answer }

type Options = {
  api: ApiClient
  // open makes the socket; a test passes a fake.
  open?: (url: string) => WebSocket
}

// call is the answer's data, or a ChatError with the API's code; a plain Error when there is no code (the API
// could not be reached or answered something else).
async function call<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const res = await attempt(request)
  if (res.ok) return res.data
  if (res.code) throw new ChatError(res.code)
  throw new Error('chat call failed')
}

// filenameHeader is RFC 6266's Content-Disposition with the name as filename* (RFC 8187), as the questionnaire's
// uploads send it.
function filenameHeader(name: string): string {
  const encoded = encodeURIComponent(name).replace(/['()*]/g, (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`)
  return `attachment; filename*=UTF-8''${encoded}`
}

export function chatClient({ api, open = (url) => new WebSocket(url) }: Options): ChatApi {
  const wave = api as unknown as Wave
  const at = (id: string) => ({ params: { path: { id } } })

  return {
    async list(before) {
      const data = await call(api.GET('/conversations', { params: { query: before === undefined ? {} : { before } } }))
      return data as { conversations: Conversation[]; next?: string }
    },
    open: (req) => call(api.POST('/conversations', { body: req })),
    get: (id) => call(api.GET('/conversations/{id}', at(id))),
    messages: (id, { before, limit } = {}) => call(api.GET('/conversations/{id}/messages', { params: { path: { id }, query: { before, limit } } })),
    send(id, clientMessageId, body, attachmentIds) {
      const req: { client_message_id: string; body: string; attachment_ids?: string[] } = { client_message_id: clientMessageId, body }
      if (attachmentIds && attachmentIds.length > 0) req.attachment_ids = attachmentIds
      return call(api.POST('/conversations/{id}/messages', { ...at(id), body: req }))
    },
    edit: (id, messageId, body, version) =>
      call(
        api.PATCH('/conversations/{id}/messages/{message_id}', {
          params: { path: { id, message_id: messageId }, header: { 'If-Match': `W/"${version}"` } },
          body: { body },
        }),
      ),
    async revisions(id, messageId) {
      return (await call(api.GET('/conversations/{id}/messages/{message_id}/revisions', { params: { path: { id, message_id: messageId } } }))).revisions
    },
    async markRead(id, seq) {
      await call(api.PUT('/conversations/{id}/read', { ...at(id), body: { seq } }))
    },
    summarize: (id, lang) => call(api.POST('/conversations/{id}/summary', { ...at(id), body: { lang } })),
    async addParticipant(id, req) {
      return (await call(wave.POST('/conversations/{id}/participants', { ...at(id), body: req }))) as ConversationParticipant
    },
    async removeParticipant(id, participantId) {
      await call(wave.DELETE('/conversations/{id}/participants/{participant_id}', { params: { path: { id, participant_id: participantId } } }))
    },
    // The list is taken to be {conversations: [...]}, as GET /conversations answers.
    async threads(id) {
      return ((await call(wave.GET('/conversations/{id}/threads', at(id)))) as { conversations: Conversation[] }).conversations
    },
    // A thread is opened with an empty body: the parent names its patient and doctors.
    async openThread(id) {
      return (await call(wave.POST('/conversations/{id}/threads', { ...at(id), body: {} }))) as Conversation
    },
    async upload(id, file) {
      const headers = { 'Content-Type': 'application/octet-stream', 'Content-Disposition': filenameHeader(file.name) }
      const res = wave.POST('/conversations/{id}/attachments', { ...at(id), body: file, headers, bodySerializer: (b) => b as File })
      return (await call(res)) as ChatAttachment
    },
    follow(id, after, onFrame, onEnd) {
      return follow(open, id, after, onFrame, onEnd)
    },
  }
}

// follow keeps the conversation's socket open until stopped: on a dropped connection it comes back after the last
// message it delivered, so nothing is lost; a message whose seq it has already delivered is dropped. Frames it does
// not know (send_ack, receipt, typing, anything newer) are ignored.
function follow(open: (url: string) => WebSocket, id: string, after: string | undefined, onFrame: (f: ChatMessageFrame) => void, onEnd: (e: ChatEnd) => void): () => void {
  let failures = 0
  let last = 0
  let timer: ReturnType<typeof setTimeout> | undefined
  let socket: WebSocket | undefined

  function connect() {
    const url = new URL(`/conversations/${encodeURIComponent(id)}/events`, location.href)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    if (after) url.searchParams.set('after', after)
    const ws = open(url.toString())
    socket = ws
    ws.onmessage = (e) => {
      const f = typeof e.data === 'string' ? parse(e.data) : undefined
      if (!f) return
      failures = 0
      if (f.type === 'message') {
        if (f.payload.seq <= last) return
        last = f.payload.seq
        after = f.payload.cursor
      }
      onFrame(f)
    }
    ws.onclose = (e) => {
      const end = ends[`${e.code} ${e.reason}`]
      if (end) return onEnd(end)
      timer = setTimeout(connect, retry[Math.min(failures++, retry.length - 1)])
    }
  }

  connect()
  return () => {
    clearTimeout(timer)
    if (socket) {
      socket.onmessage = socket.onclose = null
      socket.close()
    }
  }
}

// parse reads a message or message_edited frame; anything else is undefined.
function parse(data: string): ChatMessageFrame | undefined {
  let f: unknown
  try {
    f = JSON.parse(data)
  } catch {
    return undefined
  }
  if (typeof f !== 'object' || f === null) return undefined
  const { type, version, payload } = f as { type?: unknown; version?: unknown; payload?: Partial<ChatMessage> | null }
  if ((type !== 'message' && type !== 'message_edited') || version !== 1) return undefined
  if (typeof payload !== 'object' || payload === null || typeof payload.seq !== 'number' || typeof payload.cursor !== 'string') return undefined
  return { type, version, payload: payload as ChatMessage }
}
