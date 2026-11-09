import type { ConsultActions, ConsultMessage, DraftEvent, LiveSource, SuggestionEvent, Template, TranscriptEvent } from '../consult'

// The in-memory stand-ins for the live source and the consultation's actions (consult.ts), and builders of the
// P0 envelopes. Every text is synthetic and labelled as a placeholder; no line is real medical wording.

const encounter = 'e2000000-0000-4000-8000-000000000001'

export function segment(p: Partial<TranscriptEvent> = {}): ConsultMessage {
  const speaker = p.speaker ?? 'patient'
  return {
    type: 'transcript',
    version: 1,
    payload: {
      encounter_id: encounter,
      segment_id: 'a1000000-0000-4000-8000-000000000001',
      speaker,
      lang: speaker === 'patient' ? 'de' : 'bs',
      original: `Placeholder ${speaker} line`,
      translation: '',
      is_final: true,
      ts_start: '2026-11-09T09:00:00Z',
      ts_end: '2026-11-09T09:00:03Z',
      ...p,
    },
  }
}

export function suggestion(p: Partial<SuggestionEvent> = {}): ConsultMessage {
  return {
    type: 'suggestion',
    version: 1,
    payload: {
      id: 'b1000000-0000-4000-8000-000000000001',
      encounter_id: encounter,
      type: 'question',
      text: 'Placeholder suggestion text',
      confidence: 0.5,
      source_segment_ids: ['a1000000-0000-4000-8000-000000000001'],
      status: 'shown',
      origin: 'ai',
      ...p,
    },
  }
}

export function draft(p: Partial<DraftEvent> = {}): ConsultMessage {
  return {
    type: 'draft',
    version: 1,
    payload: {
      draft_id: 'c1000000-0000-4000-8000-000000000001',
      encounter_id: encounter,
      template_id: templates[0].id,
      status: 'preliminary',
      fields: [
        { key: 'reason', value: 'Placeholder reason for the visit', source_segment_ids: ['a1000000-0000-4000-8000-000000000001'], missing: false },
        { key: 'plan', value: '', missing: true },
      ],
      ...p,
    },
  }
}

// templates are two synthetic clinic templates, their labels placeholders.
export const templates: Template[] = [
  {
    id: 'd1000000-0000-4000-8000-000000000001',
    title: 'Placeholder visit summary',
    fields: [
      { key: 'reason', label: 'Placeholder reason' },
      { key: 'plan', label: 'Placeholder plan' },
    ],
  },
  { id: 'd1000000-0000-4000-8000-000000000002', title: 'Placeholder referral', fields: [{ key: 'to', label: 'Placeholder referred to' }] },
]

// fakeLive is a live source the test drives: emit delivers a message to every open subscription.
export function fakeLive() {
  const listeners = new Set<(m: ConsultMessage) => void>()
  const opened: string[] = []
  let closed = 0
  const live: LiveSource = {
    subscribe(appointmentId, onMessage) {
      opened.push(appointmentId)
      listeners.add(onMessage)
      return () => {
        closed++
        listeners.delete(onMessage)
      }
    },
  }
  return {
    live,
    opened,
    closed: () => closed,
    emit: (...messages: ConsultMessage[]) => {
      for (const m of messages) for (const l of listeners) l(m)
    },
  }
}

type ActionOptions = { templates?: Template[] | 'fail'; failStatus?: boolean; failSign?: boolean; failGenerate?: boolean }

// fakeActions records every call. Generate answers at once and the draft arrives later on the live source, as
// it will from the drafting backend; sign answers with the signed draft.
export function fakeActions(options: ActionOptions = {}) {
  const statuses: [string, SuggestionEvent['status']][] = []
  const generated: [string, string][] = []
  const signed: [string, { key: string; value: string }[]][] = []
  const actions: ConsultActions = {
    async setSuggestionStatus(id, status) {
      statuses.push([id, status])
      if (options.failStatus) throw new Error('synthetic failure')
    },
    async templates() {
      if (options.templates === 'fail') throw new Error('synthetic failure')
      return options.templates ?? templates
    },
    async generate(appointmentId, templateId) {
      generated.push([appointmentId, templateId])
      if (options.failGenerate) throw new Error('synthetic failure')
    },
    async sign(draftId, fields) {
      signed.push([draftId, fields])
      if (options.failSign) throw new Error('synthetic failure')
      const d = draft({ draft_id: draftId, status: 'final' }).payload as DraftEvent
      return { ...d, fields: fields.map((f) => ({ ...f, missing: f.value === '' })) }
    },
  }
  return { actions, statuses, generated, signed }
}
