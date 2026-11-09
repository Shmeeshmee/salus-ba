import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { Captions, ConsultSide } from './ConsultPanels'
import { captionOf, useConsult } from './consult'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { fakeActions, fakeLive, segment, suggestion } from './test/fakeConsult'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
const zone = 'Europe/Sarajevo'

let scrolled: Element[]
beforeEach(() => {
  scrolled = []
  // jsdom has no layout, so scrollIntoView is missing; the test records which element was brought into view.
  Element.prototype.scrollIntoView = function (this: Element) {
    scrolled.push(this)
  }
})

type Fake = ReturnType<typeof fakeLive>

// Panels renders the captions and the side panel on one state, as the call page does.
function Panels({ fake, actions }: { fake: Fake; actions: ReturnType<typeof fakeActions>['actions'] }) {
  const [state, dispatch] = useConsult(fake.live, id, true)
  return (
    <>
      <Captions caption={captionOf(state)} />
      <ConsultSide appointmentId={id} state={state} dispatch={dispatch} actions={actions} zone={zone} />
    </>
  )
}

function renderPanels(options: Parameters<typeof fakeActions>[0] = {}, language: Language = 'en') {
  const fake = fakeLive()
  const acts = fakeActions(options)
  renderInLanguage(<Panels fake={fake} actions={acts.actions} />, language)
  return { ...fake, ...acts, emit: (...m: Parameters<Fake['emit']>) => act(() => fake.emit(...m)) }
}

const tab = (name: string | RegExp) => screen.getByRole('tab', { name })

describe('ConsultPanels: captions', () => {
  it("shows the translation of what the patient says, with the original on hover and focus", () => {
    const { emit } = renderPanels()
    emit(segment({ original: 'Placeholder original', translation: 'Placeholder translation', lang: 'de' }))

    const caption = within(screen.getByRole('group', { name: 'Captions' })).getByText('Placeholder translation').closest('[tabindex]')!
    const tip = screen.getByRole('tooltip')
    expect(caption.getAttribute('aria-describedby')).toBe(tip.id)
    expect(tip.textContent).toBe('Original: Placeholder original')
    expect(within(tip).getByText('Placeholder original').getAttribute('lang')).toBe('de')
  })

  it('revises a partial caption in place until the final one', () => {
    const { emit } = renderPanels()
    emit(segment({ segment_id: 's1', translation: 'Placeholder par', is_final: false }))
    emit(segment({ segment_id: 's1', translation: 'Placeholder partial revised', is_final: false }))

    expect(screen.queryByText('Placeholder par')).toBeNull()
    expect(screen.getAllByText('Placeholder partial revised')).toHaveLength(2) // the caption and its transcript line
  })

  it('shows the original while there is no translation yet, without a tooltip', () => {
    const { emit } = renderPanels()
    emit(segment({ original: 'Placeholder untranslated', translation: '', is_final: false }))

    expect(within(screen.getByRole('group', { name: 'Captions' })).getByText('Placeholder untranslated')).toBeTruthy()
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it("does not caption the practitioner's own words", () => {
    const { emit } = renderPanels()
    emit(segment({ speaker: 'practitioner', original: 'Placeholder own words' }))

    expect(screen.queryByRole('group', { name: 'Captions' })).toBeNull()
  })

  it('clears a final caption after a few seconds of quiet, and keeps a partial one', () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    try {
      const { emit } = renderPanels()
      emit(segment({ segment_id: 's1', translation: 'Placeholder still talking', is_final: false }))
      act(() => vi.advanceTimersByTime(10_000))
      expect(screen.getByRole('group', { name: 'Captions' })).toBeTruthy()

      emit(segment({ segment_id: 's1', translation: 'Placeholder done', is_final: true }))
      act(() => vi.advanceTimersByTime(5_999))
      expect(screen.getByRole('group', { name: 'Captions' })).toBeTruthy()
      act(() => vi.advanceTimersByTime(1))
      expect(screen.queryByRole('group', { name: 'Captions' })).toBeNull()

      emit(segment({ segment_id: 's2', translation: 'Placeholder next', is_final: false }))
      expect(within(screen.getByRole('group', { name: 'Captions' })).getByText('Placeholder next')).toBeTruthy()
    } finally {
      vi.useRealTimers()
    }
  })
})

// sized gives the transcript's list a layout jsdom does not have: 1000 px of lines in a 300 px box.
function sized(list: HTMLElement, top: number) {
  Object.defineProperty(list, 'scrollHeight', { configurable: true, value: 1000 })
  Object.defineProperty(list, 'clientHeight', { configurable: true, value: 300 })
  Object.defineProperty(list, 'scrollTop', { configurable: true, writable: true, value: top })
}

describe('ConsultPanels: the tabs', () => {
  it('moves between tabs with the arrow keys, without scrolling the page', () => {
    renderPanels()

    expect(fireEvent.keyDown(tab('Transcript'), { key: 'ArrowRight' })).toBe(false) // the default was prevented
    expect(tab('Suggestions').getAttribute('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(tab('Suggestions'))
    fireEvent.keyDown(tab('Suggestions'), { key: 'ArrowLeft' })
    fireEvent.keyDown(tab('Transcript'), { key: 'ArrowLeft' })
    expect(tab('Documents').getAttribute('aria-selected')).toBe('true')
    expect(fireEvent.keyDown(tab('Documents'), { key: 'Enter' })).toBe(true)
  })
})

describe('ConsultPanels: the transcript', () => {
  it('says it is empty until someone speaks', () => {
    renderPanels()
    expect(tab('Transcript').getAttribute('aria-selected')).toBe('true')
    expect(screen.getByText('The transcript appears here as the conversation goes on.')).toBeTruthy()
  })

  it('shows each line with its speaker, time, original and translation', () => {
    const { emit } = renderPanels()
    emit(
      segment({ segment_id: 's1', speaker: 'patient', lang: 'de', original: 'Placeholder patient original', translation: 'Placeholder patient translation', ts_start: '2026-11-09T09:04:00Z' }),
      segment({ segment_id: 's2', speaker: 'practitioner', lang: 'bs', original: 'Placeholder practitioner original', translation: 'Placeholder practitioner translation', ts_start: '2026-11-09T09:05:00Z' }),
    )

    const lines = within(screen.getByRole('list', { name: 'Transcript' })).getAllByRole('listitem')
    expect(lines.map((l) => l.textContent)).toEqual([
      'Patient10:04Placeholder patient translationPlaceholder patient original',
      'Practitioner10:05Placeholder practitioner originalPlaceholder practitioner translation',
    ])
    expect(within(lines[0]).getByText('Placeholder patient original').getAttribute('lang')).toBe('de')
  })

  it('marks a line still being recognised', () => {
    const { emit } = renderPanels()
    emit(segment({ original: 'Placeholder partial', is_final: false }))

    const line = within(screen.getByRole('list', { name: 'Transcript' })).getByRole('listitem')
    expect(within(line).getByText('In progress')).toBeTruthy()
  })

  it('follows the latest line while the practitioner is at the bottom', () => {
    const { emit } = renderPanels()
    const list = screen.getByRole('list', { name: 'Transcript' })
    sized(list, 700)
    fireEvent.scroll(list)

    emit(segment({ segment_id: 's1' }))

    expect(list.scrollTop).toBe(1000)
    expect(screen.queryByRole('button', { name: 'Show the latest' })).toBeNull()
  })

  it('does not move the scroll when the practitioner has scrolled up, and offers the latest', () => {
    const { emit } = renderPanels()
    const list = screen.getByRole('list', { name: 'Transcript' })
    sized(list, 200)
    fireEvent.scroll(list)

    emit(segment({ segment_id: 's1' }))

    expect(list.scrollTop).toBe(200)
    fireEvent.click(screen.getByRole('button', { name: 'Show the latest' }))
    expect(list.scrollTop).toBe(1000)
    expect(screen.queryByRole('button', { name: 'Show the latest' })).toBeNull()
  })

  it('is in the UI language', () => {
    const { emit } = renderPanels({}, 'bs')
    emit(segment({ speaker: 'patient', is_final: false }))

    expect(tab('Transkript')).toBeTruthy()
    expect(within(screen.getByRole('list', { name: 'Transkript' })).getByText('Pacijent')).toBeTruthy()
  })
})

describe('ConsultPanels: suggestions', () => {
  const kinds = [
    ['diagnosis', 'Suggestion: a diagnosis to consider'],
    ['question', 'Suggestion: a question to ask'],
    ['red_flag', 'Suggestion: a red flag to check'],
    ['term', 'Suggestion: a term to explain'],
  ] as const

  it('labels every suggestion as a suggestion, never a diagnosis, and counts them on the tab', () => {
    const { emit } = renderPanels()
    emit(...kinds.map(([type], i) => suggestion({ id: `s${i}`, type, text: `Placeholder ${type}` })))

    expect(tab('Suggestions 4')).toBeTruthy()
    fireEvent.click(tab(/Suggestions/))
    expect(screen.getByText('Suggestions are prompts drawn from the transcript, not a diagnosis. Every decision is yours.')).toBeTruthy()
    for (const [type, label] of kinds) {
      const card = screen.getByRole('article', { name: label })
      expect(within(card).getByText(`Placeholder ${type}`)).toBeTruthy()
    }
  })

  it('says when there are none yet', () => {
    renderPanels()
    fireEvent.click(tab('Suggestions'))
    expect(screen.getByText('No suggestions yet.')).toBeTruthy()
  })

  it('pins a suggestion to the top and unpins it', async () => {
    const { emit, statuses } = renderPanels()
    emit(suggestion({ id: 'old', text: 'Placeholder older' }), suggestion({ id: 'new', text: 'Placeholder newer' }))
    fireEvent.click(tab(/Suggestions/))
    const texts = () => screen.getAllByRole('article').map((a) => a.querySelector('.suggestion-text')?.textContent)
    expect(texts()).toEqual(['Placeholder newer', 'Placeholder older'])

    const older = screen.getAllByRole('article')[1]
    fireEvent.click(within(older).getByRole('button', { name: 'Pin' }))

    await vi.waitFor(() => expect(texts()).toEqual(['Placeholder older', 'Placeholder newer']))
    expect(statuses).toEqual([['old', 'pinned']])
    const pinned = within(screen.getAllByRole('article')[0]).getByRole('button', { name: 'Unpin' })
    fireEvent.click(pinned)
    await vi.waitFor(() => expect(statuses).toEqual([['old', 'pinned'], ['old', 'shown']]))
  })

  it('dismisses a suggestion, which leaves the list and the count', async () => {
    const { emit, statuses } = renderPanels()
    emit(suggestion({ id: 's1' }), suggestion({ id: 's2', text: 'Placeholder other' }))
    fireEvent.click(tab(/Suggestions/))

    fireEvent.click(within(screen.getAllByRole('article')[1]).getByRole('button', { name: 'Dismiss' }))

    await vi.waitFor(() => expect(screen.getAllByRole('article')).toHaveLength(1))
    expect(statuses).toEqual([['s1', 'dismissed']])
    expect(tab('Suggestions 1')).toBeTruthy()
  })

  it('puts a suggestion back and says so when its status cannot be saved', async () => {
    const { emit } = renderPanels({ failStatus: true })
    emit(suggestion({ id: 's1' }))
    fireEvent.click(tab(/Suggestions/))

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))

    expect(await screen.findByRole('alert')).toHaveProperty('textContent', 'The suggestion could not be updated. Try again.')
    expect(screen.getAllByRole('article')).toHaveLength(1)
  })

  it('follows a status the server sends', () => {
    const { emit } = renderPanels()
    emit(suggestion({ id: 's1' }))
    emit(suggestion({ id: 's1', status: 'dismissed' }))
    fireEvent.click(tab('Suggestions'))
    expect(screen.queryByRole('article')).toBeNull()
  })

  it('shows its source lines in the transcript, highlighted and brought into view', () => {
    const { emit } = renderPanels()
    emit(
      segment({ segment_id: 's1', original: 'Placeholder first', ts_start: '2026-11-09T09:00:01Z' }),
      segment({ segment_id: 's2', original: 'Placeholder second', ts_start: '2026-11-09T09:00:02Z' }),
      segment({ segment_id: 's3', original: 'Placeholder third', ts_start: '2026-11-09T09:00:03Z' }),
      suggestion({ id: 'x', source_segment_ids: ['s2', 's3'] }),
    )
    fireEvent.click(tab(/Suggestions/))

    fireEvent.click(screen.getByRole('button', { name: 'Show the 2 source lines' }))

    expect(tab('Transcript').getAttribute('aria-selected')).toBe('true')
    const lines = within(screen.getByRole('list', { name: 'Transcript' })).getAllByRole('listitem')
    expect(lines.map((l) => l.getAttribute('aria-current'))).toEqual([null, 'true', 'true'])
    expect(scrolled).toEqual([lines[1]])
  })

  it('names a single source line in the singular', () => {
    const { emit } = renderPanels()
    emit(suggestion({ id: 'x', source_segment_ids: ['s1'] }))
    fireEvent.click(tab(/Suggestions/))
    expect(screen.getByRole('button', { name: 'Show the source line' })).toBeTruthy()
  })
})
