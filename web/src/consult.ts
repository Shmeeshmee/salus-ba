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

// LiveSource delivers the messages of an appointment's consultation: every message of it so far, then each
// new one as it comes. subscribe returns the function that closes the subscription.
export interface LiveSource {
  subscribe(appointmentId: string, onMessage: (message: ConsultMessage) => void): () => void
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

const types = new Set(['transcript', 'suggestion', 'draft'])

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
  if (typeof type !== 'string' || !types.has(type) || version !== 1) return undefined
  if (typeof payload !== 'object' || payload === null || Array.isArray(payload)) return undefined
  return m as ConsultMessage
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
      segments.sort((a, b) => a.ts_start.localeCompare(b.ts_start))
      return { ...state, segments, caption: m.payload.speaker === 'patient' ? m.payload.segment_id : state.caption }
    }
    case 'suggestion':
      return { ...state, suggestions: upsert(state.suggestions, m.payload, (s) => s.id) }
    case 'draft':
      return { ...state, drafts: upsert(state.drafts, m.payload, (d) => d.draft_id) }
  }
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

// useConsult follows the appointment's live source while active, folding every message into the state.
export function useConsult(live: LiveSource | undefined, appointmentId: string, active: boolean) {
  const [state, dispatch] = useReducer(apply, empty)
  useEffect(() => (live && active ? live.subscribe(appointmentId, dispatch) : undefined), [live, appointmentId, active])
  return [state, dispatch] as const
}
