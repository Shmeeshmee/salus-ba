import type { components } from './api/schema'
import type { ApiClient } from './api/client'
import type { Bundle, FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { decodeMessage, SignRefused, TranscriptionDeclined, type ConsultActions, type DraftEvent, type End, type LiveSource, type SuggestionEvent, type Template } from './consult'

export type { End } from './consult'

// live.ts wires the call page's seams (consult.ts) to the consultation's event stream: the WebSocket
// GET /encounters/{id}/events (SAL-6, api/openapi.yaml). Its frames alternate a cursor with the stream entry's id
// and the entry's message; the page keeps a cursor once the message after it has arrived and resumes after it,
// so a reconnect neither loses nor repeats a message. The practitioner's commands (SAL-7) go out on the same
// socket; the documents (SAL-8) are the JSON API's templates, drafts and sign. Nothing here logs, and no text of a
// message leaves the page.

// retry is how long to wait before the next attempt to find the encounter or to connect, by failures in a row.
const retry = [1000, 2000, 5000, 10_000]

// ends are the close codes and reasons after which the stream does not come back: access has ended.
const ends: Record<string, End> = { '4403 consent_required': 'consent_required', '4403 forbidden': 'forbidden', '4404 not_found': 'not_found' }

type Pending = { id: string; status: SuggestionEvent['status']; resolve: () => void; reject: (e: Error) => void }

type Options = {
  api: ApiClient
  fhir: Pick<FhirClient, 'get'>
  // open makes the socket; a test passes a fake.
  open?: (url: string) => WebSocket
}

// consultSocket is the live source and the actions of one call page, which share its one socket.
export function consultSocket({ api, fhir, open = (url) => new WebSocket(url) }: Options): { live: LiveSource; actions: ConsultActions } {
  let current: WebSocket | undefined
  let pending: Pending[] = []
  // found is the encounter of each appointment once known; it does not change.
  const found = new Map<string, string>()

  // encounterOf is the appointment's encounter, or undefined while there is none.
  async function encounterOf(appointmentId: string): Promise<string | undefined> {
    const known = found.get(appointmentId)
    if (known) return known
    const bundle = await fhir.get<Bundle<{ id: string }>>(`Encounter?appointment=${encodeURIComponent(appointmentId)}`)
    const id = bundle.entry?.[0]?.resource.id
    if (id) found.set(appointmentId, id)
    return id
  }

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
        const encounter = await encounterOf(appointmentId).catch(() => undefined) // a failure is retried, like none yet
        if (stopped) return
        if (!encounter) return later(() => void find())
        connect(encounter)
        void stored(encounter)
      }

      // stored delivers the drafts the encounter has, which the stream may no longer hold (it is capped). A
      // failure leaves the stream's.
      async function stored(encounter: string) {
        const res = await attempt(api.GET('/encounters/{id}/drafts', { params: { path: { id: encounter } } }))
        if (!res.ok || stopped) return
        for (const d of res.data.drafts) onMessage({ type: 'draft', version: 1, payload: d })
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
            // One that names no suggestion (an unknown or unreadable command) is taken as refusing them all.
            settle((p) => frame.id === undefined || p.id === frame.id, new Error('command refused'))
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
    async templates(language) {
      const res = await attempt(api.GET('/document-templates'))
      if (!res.ok) throw new Error('templates not listed')
      return res.data.templates.map((t) => inLanguage(t, language))
    },
    async generate(appointmentId, templateId, language) {
      const encounter = await encounterOf(appointmentId)
      if (!encounter) throw new Error('no encounter')
      const res = await attempt(
        api.POST('/encounters/{id}/drafts', { params: { path: { id: encounter } }, body: { templateId, language: language as Language } }),
      )
      if (!res.ok && res.code === 'transcription_consent_required') throw new TranscriptionDeclined()
      if (!res.ok) throw new Error('not drafted')
      return res.data
    },
    async sign(draft, fields) {
      // A draft from before SAL-8 has no version to name: it cannot be signed as read.
      if (draft.version_id === undefined) throw new Error('no version')
      const { data, error, response } = await api.POST('/drafts/{id}/sign', {
        params: { path: { id: draft.draft_id }, header: { 'If-Match': `W/"${draft.version_id}"` } },
        body: { fields },
      })
      if (response.ok && data) return data as DraftEvent
      if (response.status === 422 && error && 'keys' in error && error.error === 'required_missing') throw new SignRefused(error.keys)
      throw new Error('not signed')
    },
    async terms(draftId) {
      const res = await attempt(api.GET('/compositions/{id}/terms', { params: { path: { id: draftId } } }))
      if (!res.ok) throw new Error('terms not read')
      return res.data
    },
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

type DocumentTemplate = components['schemas']['DocumentTemplate']
type Language = components['schemas']['Language']

// inLanguage is the template as the page shows it: in language where it is written or translated in it, in its
// own language otherwise.
function inLanguage(t: DocumentTemplate, language: string): Template {
  const translation = t.language === language ? undefined : t.translations.find((x) => x.language === language)
  return {
    id: t.id,
    title: translation?.title ?? t.title,
    language: translation?.language ?? t.language,
    fields: t.fields.map((f) => ({ key: f.key, label: translation?.fields[f.key]?.label ?? f.label, required: f.required })),
  }
}
