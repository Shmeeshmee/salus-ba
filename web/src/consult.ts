import { useEffect, useReducer } from 'react'
import type { components } from './api/schema'

// consult.ts is the call page's view of the consultation's live intelligence (docs/plans/consult-intelligence.md):
// the bilingual transcript, the suggestions and the drafted documents. They arrive as pkg/wire envelopes
// ({type, version, payload}) on a live source, and the practitioner's actions go through ConsultActions. Both are
// seams: the screens are built and tested against the in-memory fakes in src/test/fakeConsult.ts until the
// WebSocket gateway (SAL-6) and the drafting backend (SAL-8) are wired in. Nothing here logs, and no text of a
// message leaves the page.

export type ConsultMessage = components['schemas']['ConsultMessage']
export type TranscriptEvent = components['schemas']['TranscriptEvent']
export type SuggestionEvent = components['schemas']['SuggestionEvent']
export type DraftEvent = components['schemas']['DraftEvent']
export type DraftField = components['schemas']['DraftField']

// Template is a clinic's document template: its fields by key, with the label the practitioner reads.
export type Template = { id: string; title: string; fields: { key: string; label: string }[] }

// End is why a live source stopped for good: the patient withdrew their consent (consent_required), or the
// practitioner may no longer follow the consultation (forbidden, not_found).
export type End = 'consent_required' | 'forbidden' | 'not_found'

// LiveSource delivers the messages of an appointment's consultation: every message of it so far, then each
// new one as it comes, until onEnd says it has stopped. subscribe returns the function that closes the
// subscription.
export interface LiveSource {
  subscribe(appointmentId: string, onMessage: (message: ConsultMessage) => void, onEnd?: (end: End) => void): () => void
}

// ConsultActions are what the practitioner does: set a suggestion's status, list the clinic's templates,
// generate (or regenerate) a template's draft, whose result arrives on the live source, and sign a draft with
// the fields as the practitioner left them, which answers with the signed draft.
export interface ConsultActions {
  setSuggestionStatus(id: string, status: SuggestionEvent['status']): Promise<void>
  templates(): Promise<Template[]>
  generate(appointmentId: string, templateId: string): Promise<void>
  sign(draftId: string, fields: { key: string; value: string }[]): Promise<DraftEvent>
}


// decodeMessage reads one frame of the live source. A frame that is not an envelope, or one of a type or
// version this page does not know, is dropped, so a newer server does not break an open page.
export function decodeMessage(frame: string): ConsultMessage | undefined {
  let m: unknown
  try {
    m = JSON.parse(frame)
  } catch {
    return undefined
  }
  if (typeof m !== 'object' || m === null || Array.isArray(m)) return undefined
  const { type, version, payload } = m as Record<string, unknown>
  const shape = typeof type === 'string' ? shapes.get(type) : undefined
  if (!shape || version !== 1) return undefined
  if (typeof payload !== 'object' || payload === null || Array.isArray(payload)) return undefined
  return shape(payload as Record<string, unknown>) ? (m as ConsultMessage) : undefined
}

// shapes check the payload fields this page reads, so a malformed frame is dropped rather than breaking the page.
// A suggestion is shown only as what its origin says it is, AI assistance (SAL-7).
const str = (v: unknown) => typeof v === 'string'
const time = (v: unknown) => typeof v === 'string' && !Number.isNaN(Date.parse(v))
const strs = (v: unknown) => Array.isArray(v) && v.every(str)
const oneOf = (v: unknown, values: string[]) => typeof v === 'string' && values.includes(v)
const shapes = new Map<string, (p: Record<string, unknown>) => boolean>([
  [
    'transcript',
    (p) =>
      str(p.segment_id) && oneOf(p.speaker, ['practitioner', 'patient']) && str(p.lang) && str(p.original) && str(p.translation) && typeof p.is_final === 'boolean' && time(p.ts_start),
  ],
  [
    'suggestion',
    (p) => str(p.id) && oneOf(p.type, ['diagnosis', 'question', 'red_flag', 'term']) && str(p.text) && strs(p.source_segment_ids) && oneOf(p.status, ['shown', 'dismissed', 'pinned']) && p.origin === 'ai',
  ],
  [
    'draft',
    (p) =>
      str(p.draft_id) && str(p.template_id) && oneOf(p.status, ['preliminary', 'final', 'entered-in-error']) && Array.isArray(p.fields) && p.fields.every(field),
  ],
])

function field(f: unknown): boolean {
  if (typeof f !== 'object' || f === null) return false
  const { key, value, missing, source_segment_ids } = f as Record<string, unknown>
  return str(key) && str(value) && typeof missing === 'boolean' && (source_segment_ids === undefined || strs(source_segment_ids))
}

// ConsultState is what the page has received: segments in the order they started, suggestions and drafts
// in the order they first came, each at its latest version. caption is the patient's latest segment's id.
export type ConsultState = {
  segments: TranscriptEvent[]
  suggestions: SuggestionEvent[]
  drafts: DraftEvent[]
  caption?: string
}

export const empty: ConsultState = { segments: [], suggestions: [], drafts: [] }

// apply folds one message into the state. It is idempotent: a replay of messages already applied changes
// nothing, so a source may send everything again when it reconnects.
export function apply(state: ConsultState, m: ConsultMessage): ConsultState {
  switch (m.type) {
    case 'transcript': {
      const segments = upsert(state.segments, m.payload, (s) => s.segment_id)
      segments.sort((a, b) => started(a) - started(b))
      return { ...state, segments, caption: captionAfter(state, m.payload) }
    }
    case 'suggestion':
      return { ...state, suggestions: upsert(state.suggestions, m.payload, (s) => s.id) }
    case 'draft':
      return { ...state, drafts: upsert(state.drafts, m.payload, (d) => d.draft_id) }
  }
}

// captionAfter is the caption once segment s has come: s if the patient said it, no earlier than the caption's
// segment started; otherwise the caption stays, so a late revision of an earlier line does not bring it back.
function captionAfter(state: ConsultState, s: TranscriptEvent): string | undefined {
  if (s.speaker !== 'patient') return state.caption
  const current = captionOf(state)
  return !current || started(s) >= started(current) ? s.segment_id : state.caption
}

// started is the instant a segment started. Its text is not comparable: a date-time may carry any offset
// (api/openapi.yaml). decodeMessage has already refused a ts_start that is not a time.
function started(s: TranscriptEvent): number {
  return Date.parse(s.ts_start)
}

function upsert<T>(list: T[], item: T, key: (t: T) => string): T[] {
  const i = list.findIndex((x) => key(x) === key(item))
  return i < 0 ? [...list, item] : list.map((x, j) => (j === i ? item : x))
}

// captionOf is the segment the captions show: the patient's latest, partial or final. The practitioner's own
// words are not captioned on their side.
export function captionOf(state: ConsultState): TranscriptEvent | undefined {
  return state.segments.find((s) => s.segment_id === state.caption)
}

// Live is the page's view of its live source: what it has received, and why it stopped, if it has.
type Live = { state: ConsultState; end?: End }

// Event is a message to fold in, the source's end, or the subscription's close, which clears that end.
type Event = ConsultMessage | { type: 'end'; end: End } | { type: 'closed' }

// follow folds an event into the view. An end drops everything received: the page may no longer show it.
function follow(live: Live, e: Event): Live {
  if (e.type === 'end') return { state: empty, end: e.end }
  if (e.type === 'closed') return live.end ? { state: live.state } : live
  return { ...live, state: apply(live.state, e) }
}

// useConsult follows the appointment's live source while active, folding every message into the state,
// until the source ends. The end is forgotten once the subscription closes (the practitioner left), so it is
// gone before the page is back in the call.
export function useConsult(live: LiveSource | undefined, appointmentId: string, active: boolean) {
  const [view, dispatch] = useReducer(follow, { state: empty })
  useEffect(() => {
    if (!live || !active) return
    const close = live.subscribe(appointmentId, dispatch, (end) => dispatch({ type: 'end', end }))
    return () => {
      close()
      dispatch({ type: 'closed' })
    }
  }, [live, appointmentId, active])
  return [view.state, dispatch, view.end] as const
}
