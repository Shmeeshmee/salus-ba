import { DiagnosisUnconfirmed, SignRefused, TranscriptionDeclined, type ConsultActions, type ConsultMessage, type DocumentTerms, type DraftEvent, type End, type LiveSource, type SuggestionEvent, type SignValue, type Template, type TranscriptEvent } from '../consult'

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
      version_id: 1,
      fields: [
        { key: 'reason', value: 'Placeholder reason for the visit', source_segment_ids: ['a1000000-0000-4000-8000-000000000001'], missing: false },
        { key: 'plan', value: '', missing: true },
      ],
      ...p,
    },
  }
}

// templates are two synthetic clinic templates, their labels placeholders. The visit summary needs its reason.
export const templates: Template[] = [
  {
    id: 'd1000000-0000-4000-8000-000000000001',
    title: 'Placeholder visit summary',
    language: 'en',
    fields: [
      { key: 'reason', label: 'Placeholder reason', required: true },
      { key: 'plan', label: 'Placeholder plan', required: false },
    ],
  },
  { id: 'd1000000-0000-4000-8000-000000000002', title: 'Placeholder referral', language: 'en', fields: [{ key: 'to', label: 'Placeholder referred to', required: false }] },
]

// fakeLive is a live source the test drives: emit delivers a message to every open subscription, end stops them.
export function fakeLive() {
  const listeners = new Set<(m: ConsultMessage) => void>()
  const enders = new Set<(e: End) => void>()
  const opened: string[] = []
  let closed = 0
  const live: LiveSource = {
    subscribe(appointmentId, onMessage, onEnd) {
      opened.push(appointmentId)
      listeners.add(onMessage)
      if (onEnd) enders.add(onEnd)
      return () => {
        closed++
        listeners.delete(onMessage)
        if (onEnd) enders.delete(onEnd)
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
    end: (e: End) => {
      for (const l of enders) l(e)
    },
  }
}

type ActionOptions = {
  templates?: Template[] | 'fail'
  failStatus?: boolean
  failSign?: boolean | { keys: string[] } | { unconfirmed: string[] }
  failGenerate?: boolean | 'declined'
  // terms are the documents' explained terms by draft id (SAL-13); none for a draft not named
  terms?: Record<string, DocumentTerms>
}

// fakeActions records every call. A generation waits until the test answers it with the stored draft (answer),
// as the drafting backend answers once the draft is stored; sign answers with the signed draft.
export function fakeActions(options: ActionOptions = {}) {
  const statuses: [string, SuggestionEvent['status']][] = []
  const generated: [string, string, string][] = []
  const signed: [string, number | undefined, SignValue[]][] = []
  const languages: string[] = []
  const waiting: ((d: DraftEvent) => void)[] = []
  const termsAsked: string[] = []
  const actions: ConsultActions = {
    async setSuggestionStatus(id, status) {
      statuses.push([id, status])
      if (options.failStatus) throw new Error('synthetic failure')
    },
    async templates(language) {
      languages.push(language)
      if (options.templates === 'fail') throw new Error('synthetic failure')
      return options.templates ?? templates
    },
    generate(appointmentId, templateId, language) {
      generated.push([appointmentId, templateId, language])
      if (options.failGenerate === 'declined') return Promise.reject(new TranscriptionDeclined())
      if (options.failGenerate) return Promise.reject(new Error('synthetic failure'))
      return new Promise<DraftEvent>((resolve) => waiting.push(resolve))
    },
    async sign(d, fields) {
      signed.push([d.draft_id, d.version_id, fields])
      if (typeof options.failSign === 'object' && 'unconfirmed' in options.failSign) throw new DiagnosisUnconfirmed(options.failSign.unconfirmed)
      if (typeof options.failSign === 'object') throw new SignRefused(options.failSign.keys)
      if (options.failSign) throw new Error('synthetic failure')
      return { ...d, status: 'final', version_id: (d.version_id ?? 0) + 1, fields: fields.map(({ key, value }) => ({ key, value, missing: value === '' })) }
    },
    async terms(id) {
      termsAsked.push(id)
      return options.terms?.[id] ?? { sections: [] }
    },
  }
  // answer answers the oldest generation still waiting with the draft d.
  const answer = (d: ConsultMessage = draft()) => waiting.shift()?.(d.payload as DraftEvent)
  return { actions, statuses, generated, signed, languages, answer, termsAsked }
}
