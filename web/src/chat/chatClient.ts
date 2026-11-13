import type { ApiClient } from '../api/client'
import { attempt } from '../api/result'
import { ChatError, type ChatApi, type ChatEnd, type ChatMessage, type ChatMessageFrame } from './chatApi'

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
  const at = (id: string) => ({ params: { path: { id } } })

  return {
    list: (before) => call(api.GET('/conversations', { params: { query: before === undefined ? {} : { before } } })),
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
    chatHours: (organizationId) => call(api.GET('/organizations/{id}/chat-hours', at(organizationId))),
    summarize: (id, lang) => call(api.POST('/conversations/{id}/summary', { ...at(id), body: { lang } })),
    addParticipant: (id, req) => call(api.POST('/conversations/{id}/participants', { ...at(id), body: req })),
    async removeParticipant(id, participantId) {
      await call(api.DELETE('/conversations/{id}/participants/{participant_id}', { params: { path: { id, participant_id: participantId } } }))
    },
    async threads(id) {
      return (await call(api.GET('/conversations/{id}/threads', at(id)))).conversations
    },
    // A side thread is opened without a body: it has nothing of its own but its messages.
    openThread: (id) => call(api.POST('/conversations/{id}/threads', at(id))),
    // The browser sets Content-Length from the File itself (a page may not set it), so only the name is a header.
    upload(id, file) {
      const header = { 'Content-Disposition': filenameHeader(file.name) } as { 'Content-Length': number; 'Content-Disposition': string }
      return call(
        api.POST('/conversations/{id}/attachments', {
          params: { path: { id }, header },
          headers: { 'Content-Type': 'application/octet-stream' },
          body: file as unknown as string,
          bodySerializer: (b) => b as unknown as File,
        }),
      )
    },
    follow(id, after, onFrame, onEnd) {
      return follow(open, () => call(api.GET('/conversations/{id}', at(id))), id, after, onFrame, onEnd)
    },
  }
}

// refusals are the conversation's answers that end the socket for good, as its close codes do.
const refusals: Partial<Record<string, ChatEnd>> = { consent_required: 'consent_required', forbidden: 'forbidden', not_found: 'not_found', unauthenticated: 'unauthenticated' }

// follow keeps the conversation's socket open until stopped: on a dropped connection it comes back after the last
// message it delivered, so nothing is lost; a message whose seq it has already delivered is dropped. Frames it does
// not know (send_ack, receipt, typing, anything newer) are ignored. A connection that never opened may be a
// handshake the API refused (the page sees no status), so the conversation is read (check): a refusal of access or
// of the session ends the socket for good; anything else tries again. Each check is an audited read, so it is made
// on the first refused connection in a row and then on every third (about twice a minute at the longest wait).
function follow(
  open: (url: string) => WebSocket,
  check: () => Promise<unknown>,
  id: string,
  after: string | undefined,
  onFrame: (f: ChatMessageFrame) => void,
  onEnd: (e: ChatEnd) => void,
): () => void {
  let failures = 0
  let refused = 0
  let last = 0
  let stopped = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let socket: WebSocket | undefined

  function later() {
    if (!stopped) timer = setTimeout(connect, retry[Math.min(failures++, retry.length - 1)])
  }

  function connect() {
    const url = new URL(`/conversations/${encodeURIComponent(id)}/events`, location.href)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    if (after) url.searchParams.set('after', after)
    const ws = open(url.toString())
    let opened = false
    socket = ws
    ws.onopen = () => {
      opened = true
      refused = 0
    }
    ws.onmessage = (e) => {
      opened = true
      refused = 0
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
      if (opened) return later()
      refused++
      if (refused !== 1 && refused % 3 !== 0) return later()
      check().then(later, (err: unknown) => {
        const refused = err instanceof ChatError ? refusals[err.code] : undefined
        if (stopped) return
        if (refused) onEnd(refused)
        else later()
      })
    }
  }

  connect()
  return () => {
    stopped = true
    clearTimeout(timer)
    if (socket) {
      socket.onopen = socket.onmessage = socket.onclose = null
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
