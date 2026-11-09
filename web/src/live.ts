import type { Bundle, FhirClient } from './api/fhir'
import { decodeMessage, type ConsultActions, type End, type LiveSource, type SuggestionEvent } from './consult'

export type { End } from './consult'

// live.ts wires the call page's seams (consult.ts) to the consultation's event stream: the WebSocket
// GET /encounters/{id}/events (SAL-6, api/openapi.yaml). Its frames alternate a cursor with the stream entry's id
// and the entry's message; the page keeps a cursor once the message after it has arrived and resumes after it,
// so a reconnect neither loses nor repeats a message. The practitioner's commands (SAL-7) go out on the same
// socket. Nothing here logs, and no text of a message leaves the page.

// retry is how long to wait before the next attempt to find the encounter or to connect, by failures in a row.
const retry = [1000, 2000, 5000, 10_000]

// ends are the close codes and reasons after which the stream does not come back: access has ended.
const ends: Record<string, End> = { '4403 consent_required': 'consent_required', '4403 forbidden': 'forbidden', '4404 not_found': 'not_found' }

type Pending = { id: string; status: SuggestionEvent['status']; resolve: () => void; reject: (e: Error) => void }

type Options = {
  fhir: Pick<FhirClient, 'get'>
  // open makes the socket; a test passes a fake.
  open?: (url: string) => WebSocket
}

// consultSocket is the live source and the actions of one call page, which share its one socket.
export function consultSocket({ fhir, open = (url) => new WebSocket(url) }: Options): { live: LiveSource; actions: ConsultActions } {
  let current: WebSocket | undefined
  let pending: Pending[] = []

  function settle(match: (p: Pending) => boolean, error?: Error) {
    const [done, rest] = [pending.filter(match), pending.filter((p) => !match(p))]
    pending = rest
    for (const p of done) {
      if (error) p.reject(error)
      else p.resolve()
    }
  }

  const live: LiveSource = {
    subscribe(appointmentId, onMessage, onEnd) {
      let stopped = false
      let failures = 0
      let after: string | undefined
      let timer: ReturnType<typeof setTimeout> | undefined
      let socket: WebSocket | undefined

      const later = (attempt: () => void) => {
        timer = setTimeout(attempt, retry[Math.min(failures++, retry.length - 1)])
      }

      async function find() {
        let encounter: string | undefined
        try {
          const found = await fhir.get<Bundle<{ id: string }>>(`Encounter?appointment=${encodeURIComponent(appointmentId)}`)
          encounter = found.entry?.[0]?.resource.id
        } catch {
          // retried below, like an encounter not started yet
        }
        if (stopped) return
        if (encounter) connect(encounter)
        else later(() => void find())
      }

      function connect(encounter: string) {
        const url = new URL(`/encounters/${encodeURIComponent(encounter)}/events`, location.href)
        url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
        if (after) url.searchParams.set('after', after)
        const ws = open(url.toString())
        socket = ws
        let cursor: string | undefined
        ws.onopen = () => {
          current = ws
        }
        ws.onmessage = (e) => {
          if (typeof e.data !== 'string') return
          failures = 0
          const frame = parse(e.data)
          if (frame?.type === 'cursor') {
            cursor = frame.id
            return
          }
          if (frame?.type === 'command_error') {
            settle((p) => p.id === frame.id, new Error('command refused'))
            return
          }
          if (cursor) after = cursor
          cursor = undefined
          const m = decodeMessage(e.data)
          if (!m) return
          onMessage(m)
          if (m.type === 'suggestion') settle((p) => p.id === m.payload.id && p.status === m.payload.status)
        }
        ws.onclose = (e) => {
          if (current === ws) current = undefined
          settle(() => true, new Error('connection closed'))
          const end = ends[`${e.code} ${e.reason}`]
          if (end) onEnd?.(end)
          else later(() => connect(encounter))
        }
      }

      void find()
      return () => {
        stopped = true
        clearTimeout(timer)
        if (socket) {
          socket.onopen = socket.onmessage = socket.onclose = null
          socket.close()
          if (current === socket) current = undefined
        }
        settle(() => true, new Error('connection closed'))
      }
    },
  }

  const actions: ConsultActions = {
    setSuggestionStatus(id, status) {
      const ws = current
      if (!ws) return Promise.reject(new Error('not connected'))
      return new Promise<void>((resolve, reject) => {
        pending.push({ id, status, resolve, reject })
        ws.send(JSON.stringify({ type: 'suggestion_status', version: 1, payload: { id, status } }))
      })
    },
    // The drafting endpoints (SAL-8) are not wired yet: no templates, so nothing can be generated or signed.
    templates: async () => [],
    generate: () => Promise.reject(new Error('drafting is not available')),
    sign: () => Promise.reject(new Error('drafting is not available')),
  }

  return { live, actions }
}

type Frame = { type: 'cursor'; id: string } | { type: 'command_error'; id?: string } | { type: 'other' }

// parse reads the two frames the live messages do not cover: the cursor and a command's error.
function parse(data: string): Frame | undefined {
  let m: unknown
  try {
    m = JSON.parse(data)
  } catch {
    return undefined
  }
  if (typeof m !== 'object' || m === null) return undefined
  const { type, version, payload } = m as { type?: unknown; version?: unknown; payload?: { id?: unknown } | null }
  if (version !== 1 || typeof payload !== 'object' || payload === null) return { type: 'other' }
  if (type === 'cursor' && typeof payload.id === 'string') return { type: 'cursor', id: payload.id }
  if (type === 'command_error') return { type: 'command_error', id: typeof payload.id === 'string' ? payload.id : undefined }
  return { type: 'other' }
}
