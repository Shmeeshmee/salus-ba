import { describe, expect, it } from 'vitest'
import { apply, captionOf, decodeMessage, empty, type ConsultMessage } from './consult'
import { draft, segment, suggestion } from './test/fakeConsult'

describe('decodeMessage', () => {
  it('reads a transcript, suggestion or draft envelope of version 1', () => {
    const m = segment({ segment_id: 'a' })
    expect(decodeMessage(JSON.stringify(m))).toEqual(m)
    const s = suggestion({ id: 'b' })
    expect(decodeMessage(JSON.stringify(s))).toEqual(s)
    const d = draft({ draft_id: 'c' })
    expect(decodeMessage(JSON.stringify(d))).toEqual(d)
  })

  it.each([
    ['not JSON', 'not json'],
    ['no payload', '{"type":"transcript","version":1}'],
    ['a null payload', '{"type":"transcript","version":1,"payload":null}'],
    ['an unknown type', '{"type":"dubbing","version":1,"payload":{}}'],
    ['a later version', '{"type":"transcript","version":2,"payload":{}}'],
    ['an array', '[]'],
    ['an array payload', '{"type":"transcript","version":1,"payload":[]}'],
    ['a transcript payload without its fields', '{"type":"transcript","version":1,"payload":{}}'],
    ['a transcript with an unknown speaker', JSON.stringify({ ...segment(), payload: { ...segment().payload, speaker: 'doctor' } })],
    ['a transcript without a start time', JSON.stringify({ ...segment(), payload: { ...segment().payload, ts_start: undefined } })],
    ['a suggestion without its sources', JSON.stringify({ ...suggestion(), payload: { ...suggestion().payload, source_segment_ids: undefined } })],
    ['a suggestion of an unknown status', JSON.stringify({ ...suggestion(), payload: { ...suggestion().payload, status: 'hidden' } })],
    ['a draft without fields', JSON.stringify({ ...draft(), payload: { ...draft().payload, fields: undefined } })],
    ['a draft with a field without a key', JSON.stringify({ ...draft(), payload: { ...draft().payload, fields: [{ value: '', missing: true }] } })],
  ])('drops %s', (_, frame) => {
    expect(decodeMessage(frame)).toBeUndefined()
  })
})

const fold = (...messages: ConsultMessage[]) => messages.reduce(apply, empty)

describe('apply: the transcript', () => {
  it('revises a partial segment in place under its segment_id until the final one', () => {
    const state = fold(
      segment({ segment_id: 'a', original: 'Boli me', is_final: false }),
      segment({ segment_id: 'b', speaker: 'practitioner', ts_start: '2026-11-09T09:00:05Z' }),
      segment({ segment_id: 'a', original: 'Boli me glava', is_final: true }),
    )
    expect(state.segments.map((s) => [s.segment_id, s.original, s.is_final])).toEqual([
      ['a', 'Boli me glava', true],
      ['b', 'Placeholder practitioner line', true],
    ])
  })

  it('orders segments by when they started, whatever order they arrive in', () => {
    const state = fold(segment({ segment_id: 'late', ts_start: '2026-11-09T09:00:09Z' }), segment({ segment_id: 'early', ts_start: '2026-11-09T09:00:01Z' }))
    expect(state.segments.map((s) => s.segment_id)).toEqual(['early', 'late'])
  })

  it('is the same after a replay of the same messages', () => {
    const messages = [segment({ segment_id: 'a' }), suggestion({ id: 's' }), draft({ draft_id: 'd' })]
    expect(fold(...messages, ...messages)).toEqual(fold(...messages))
  })
})

describe('captionOf', () => {
  it("is the patient's latest segment, partial or final, and never the practitioner's", () => {
    let state = fold(segment({ segment_id: 'p1', speaker: 'patient', is_final: false }))
    expect(captionOf(state)?.segment_id).toBe('p1')
    state = apply(state, segment({ segment_id: 'd1', speaker: 'practitioner' }))
    expect(captionOf(state)?.segment_id).toBe('p1')
    state = apply(state, segment({ segment_id: 'p1', speaker: 'patient', original: 'revised', is_final: true }))
    expect(captionOf(state)?.original).toBe('revised')
  })

  it('stays on the later segment when an earlier one is revised late', () => {
    const state = fold(
      segment({ segment_id: 'p1', ts_start: '2026-11-09T09:00:01Z', is_final: false }),
      segment({ segment_id: 'p2', ts_start: '2026-11-09T09:00:05Z', is_final: false }),
      segment({ segment_id: 'p1', ts_start: '2026-11-09T09:00:01Z', is_final: true }),
    )
    expect(captionOf(state)?.segment_id).toBe('p2')
  })

  it('is nothing before the patient has spoken', () => {
    expect(captionOf(fold(segment({ speaker: 'practitioner' })))).toBeUndefined()
  })
})

describe('apply: suggestions and drafts', () => {
  it('keeps the latest status of every suggestion', () => {
    const state = fold(suggestion({ id: 's', status: 'shown' }), suggestion({ id: 't' }), suggestion({ id: 's', status: 'pinned' }))
    expect(state.suggestions.map((s) => [s.id, s.status])).toEqual([
      ['s', 'pinned'],
      ['t', 'shown'],
    ])
  })

  it('keeps the latest version of every draft', () => {
    const state = fold(draft({ draft_id: 'd', status: 'preliminary' }), draft({ draft_id: 'd', status: 'final' }))
    expect(state.drafts.map((d) => [d.draft_id, d.status])).toEqual([['d', 'final']])
  })
})
