import { cleanup, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from '../api/client'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { fakeFetch } from '../test/fakeFetch'
import { ExplanationsProvider, TermList, TermText } from './Explain'
import { pieces, type Explanation } from './terms'

afterEach(cleanup)

// Synthetic explanations: one approved in the UI language, one unreviewed English fallback.
const liver: Explanation = { termId: 't-liver', language: 'bs', text: 'Placeholder: organ u trbuhu.', reviewStatus: 'approved' }
const ulcer: Explanation = { termId: 't-ulcer', language: 'en', text: 'Placeholder: a sore in the stomach.', reviewStatus: 'unreviewed' }

function setup(ui: React.ReactNode, language: 'bs' | 'en' = 'en', explanations: Explanation[] = [liver, ulcer]) {
  const { fetch, requests } = fakeFetch({ 'GET /terminology/explanations': () => Response.json({ explanations }) })
  renderInLanguage(<ExplanationsProvider api={newApiClient(fetch)}>{ui}</ExplanationsProvider>, language)
  return { requests }
}

describe('pieces', () => {
  it('cuts a text at its spans, in UTF-16 offsets, leaving the text as it is', () => {
    const text = '😀 jetra i čir.'
    const got = pieces(text, [
      { term_id: 't-liver', start: 3, end: 8 },
      { term_id: 't-ulcer', start: 11, end: 14 },
    ])
    expect(got).toEqual([{ text: '😀 ' }, { text: 'jetra', termId: 't-liver' }, { text: ' i ' }, { text: 'čir', termId: 't-ulcer' }, { text: '.' }])
    expect(got.map((p) => p.text).join('')).toBe(text)
  })

  it('drops a span outside the text or overlapping the one before', () => {
    expect(pieces('jetra', [{ term_id: 'a', start: 0, end: 9 }])).toEqual([{ text: 'jetra' }])
    expect(
      pieces('jetra bol', [
        { term_id: 'a', start: 0, end: 5 },
        { term_id: 'b', start: 3, end: 9 },
      ]),
    ).toEqual([{ text: 'jetra', termId: 'a' }, { text: ' bol' }])
  })
})

describe('TermText', () => {
  it('explains a tagged term on hover and focus in the viewer language, labelled informational', async () => {
    const { requests } = setup(<TermText text="Bol u jetri." spans={[{ term_id: 't-liver', start: 6, end: 11 }]} />, 'bs')
    const term = await screen.findByText('jetri')
    expect(term.getAttribute('tabindex')).toBe('0')
    const tip = document.getElementById(term.getAttribute('aria-describedby')!)!
    expect(tip.getAttribute('role')).toBe('tooltip')
    expect(tip.textContent).toContain('Placeholder: organ u trbuhu.')
    expect(tip.textContent).toContain('Informativno, nije dijagnoza')
    expect(tip.textContent).not.toContain('Još nije provjereno')
    expect(new URL(requests[0].url).searchParams.get('language')).toBe('bs')
  })

  it('marks an unreviewed explanation, and an English fallback by its language', async () => {
    setup(<TermText text="A stomach ulcer." spans={[{ term_id: 't-ulcer', start: 2, end: 15 }]} />, 'bs')
    const term = await screen.findByText('stomach ulcer')
    const tip = document.getElementById(term.getAttribute('aria-describedby')!)!
    expect(tip.textContent).toContain('Još nije provjereno')
    expect(screen.getByText('Placeholder: a sore in the stomach.').getAttribute('lang')).toBe('en')
  })

  it('leaves a term without an explanation, and a text without spans, as plain text', async () => {
    const { requests } = setup(
      <>
        <p>
          <TermText text="Placeholder fatty liver." spans={[{ term_id: 't-other', start: 12, end: 23 }]} />
        </p>
        <p>
          <TermText text="Plain line." />
        </p>
      </>,
    )
    await waitFor(() => expect(requests).toHaveLength(1))
    expect(screen.getByText('Placeholder fatty liver.').tagName).toBe('P')
    expect(screen.getByText('Plain line.').tagName).toBe('P')
    expect(screen.queryAllByRole('tooltip')).toHaveLength(0)
  })
})

describe('TermList', () => {
  it('lists the explained terms of a text with the informational label, for a tooltip of its own', async () => {
    setup(<TermList text="Bol u jetri i čir." spans={[{ term_id: 't-liver', start: 6, end: 11 }, { term_id: 't-other', start: 14, end: 17 }]} />)
    expect(await screen.findByText('Placeholder: organ u trbuhu.')).toBeTruthy()
    expect(screen.getByText('jetri')).toBeTruthy()
    expect(screen.getByText('Informational, not a diagnosis')).toBeTruthy()
    expect(screen.queryByText('čir')).toBeNull()
  })
})
