import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from '../api/client'
import type { components } from '../api/schema'
import { renderInLanguage } from '../i18n/renderInLanguage'
import type { Language } from '../i18n/languages'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from '../test/fakeFetch'
import { Review } from './Review'

type Entry = components['schemas']['TermReviewEntry']
type Translation = components['schemas']['TermTranslation']
type Explanation = components['schemas']['TermExplanation']

// Synthetic entries of the Croatian queue: a liver with its labels and no explanation, an ulcer without a Croatian
// label, and a heart with an approved translation and an unreviewed explanation.
const liverID = '00000000-0000-4000-8000-000000000001'
const ulcerID = '00000000-0000-4000-8000-000000000002'
const heartID = '00000000-0000-4000-8000-000000000003'
const at = '2026-11-10T09:00:00Z'
const translation = (termId: string, p: Partial<Translation> = {}): Translation => ({
  termId, language: 'hr', review: { status: 'unreviewed' }, versionId: 1, lastUpdated: at, ...p,
})
const explanation = (termId: string, p: Partial<Explanation> = {}): Explanation => ({
  termId, language: 'hr', text: 'Placeholder explanation text', review: { status: 'unreviewed' }, versionId: 1, lastUpdated: at, ...p,
})
const liver: Entry = {
  status: 'unreviewed',
  term: {
    id: liverID, category: 'anatomy', reviewStatus: 'unreviewed', codes: { ta98: 'A05.8.01.001' },
    labels: [
      { language: 'hr', preferred: 'jetra', synonyms: ['hepar'] },
      { language: 'la', preferred: 'hepar', synonyms: [] },
      { language: 'en', preferred: 'liver', synonyms: [] },
    ],
  },
  translation: translation(liverID),
}
const ulcer: Entry = {
  status: 'unreviewed',
  term: { id: ulcerID, category: 'diagnosis', reviewStatus: 'unreviewed', codes: { icd10: 'K25' }, labels: [{ language: 'la', preferred: 'ulcus ventriculi', synonyms: [] }] },
}
const heart: Entry = {
  status: 'unreviewed',
  term: { id: heartID, category: 'anatomy', reviewStatus: 'approved', codes: {}, labels: [{ language: 'hr', preferred: 'srce', synonyms: [] }] },
  translation: translation(heartID, { review: { status: 'approved', reviewedAt: at } }),
  explanation: explanation(heartID),
}
const progress = {
  languages: [
    { language: 'la', terms: 3, approved: 0, needsChanges: 0 },
    { language: 'hr', terms: 3, approved: 1, needsChanges: 1 },
    { language: 'en', terms: 3, approved: 2, needsChanges: 0 },
  ],
}

// setup renders the panel in the UI language ui (English) and picks the Croatian queue, as a reviewer unless told.
function setup(routes: Record<string, Route> = {}, { reviewer = true, ui = 'en' as Language } = {}) {
  const fake = fakeFetch({
    'GET /terminology/review': () => Response.json({ entries: [liver, ulcer, heart] }),
    'GET /terminology/review/progress': () => Response.json(progress),
    ...routes,
  })
  renderInLanguage(<Review api={newApiClient(fake.fetch)} reviewer={reviewer} />, ui)
  fireEvent.change(document.getElementById('review-language')!, { target: { value: 'hr' } })
  return fake
}

const entries = () => screen.getAllByRole('article')
const part = (entry: HTMLElement, name: string) => within(entry).getByRole('region', { name })
const queries = (requests: Request[]) => requests.filter((r) => new URL(r.url).pathname === '/terminology/review').map((r) => new URL(r.url).searchParams)

afterEach(cleanup)

describe('the queue', () => {
  it('reads every entry of the UI language by default, needs changes first, and shows each term with its labels, context and explanation', async () => {
    const fake = setup()

    expect(await screen.findByRole('heading', { name: 'Term review' })).toBeTruthy()
    await screen.findByRole('heading', { name: 'jetra' })
    expect(queries(fake.requests)[0].get('language')).toBe('en') // the UI language first
    expect(queries(fake.requests).at(-1)?.get('language')).toBe('hr')
    expect(queries(fake.requests).at(-1)?.has('status')).toBe(false) // All: the server orders needs changes first
    const filter = screen.getByLabelText('Status') as HTMLSelectElement
    expect(filter.value).toBe('all')
    expect([...filter.options].map((o) => o.value)).toEqual(['all', 'needs_changes', 'unreviewed', 'approved'])
    const [first, second, third] = entries()
    expect(part(first, 'Labels').textContent).toContain('hepar')
    expect(part(first, 'Labels').textContent).toContain('Unreviewed')
    expect(first.textContent).toContain('liver')
    expect(part(first, 'Explanation for patients (informational)').textContent).toContain('No explanation yet')
    expect(within(second).getByRole('heading', { level: 2 }).textContent).toBe('No label in Croatian yet')
    expect(part(third, 'Labels').textContent).toContain('Approved')
    expect(part(third, 'Explanation for patients (informational)').textContent).toContain('Placeholder explanation text')
  })

  it('shows the progress of the chosen language and every language behind a disclosure', async () => {
    setup()

    expect(await screen.findByText('1 of 3 approved in Croatian')).toBeTruthy()
    expect(screen.getByText('1 needs changes')).toBeTruthy()
    expect(screen.getByText('All languages')).toBeTruthy()
    expect(screen.getByRole('cell', { name: 'Latin' })).toBeTruthy()
  })

  it('reloads from the start when a filter changes, with the chosen status, and hides explanations in Latin', async () => {
    const fake = setup()
    await screen.findByRole('heading', { name: 'jetra' })

    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'unreviewed' } })
    await screen.findByRole('heading', { name: 'jetra' })
    fireEvent.change(screen.getByLabelText('Language'), { target: { value: 'la' } })
    await screen.findAllByRole('article')

    const asked = queries(fake.requests)
    expect(asked.at(-2)?.get('status')).toBe('unreviewed')
    expect(asked.at(-1)?.get('status')).toBe('unreviewed')
    expect(asked.at(-1)?.get('language')).toBe('la')
    expect(asked.at(-1)?.has('after')).toBe(false)
    expect(screen.queryByRole('region', { name: 'Explanation for patients (informational)' })).toBeNull()
  })

  it('loads more with the page token, appending', async () => {
    const fake = setup({
      'GET /terminology/review': (r) =>
        Response.json(new URL(r.url).searchParams.has('after') ? { entries: [heart] } : { entries: [liver, ulcer], next: 'tok-2' }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Load more' }))

    await screen.findByRole('heading', { name: 'srce' })
    expect(entries()).toHaveLength(3)
    expect(queries(fake.requests).at(-1)?.get('after')).toBe('tok-2')
    expect(screen.queryByRole('button', { name: 'Load more' })).toBeNull()
  })

  it('says when there is nothing to review and when the queue cannot be read', async () => {
    setup({ 'GET /terminology/review': () => Response.json({ entries: [] }) })
    expect(await screen.findByText('Nothing to review here.')).toBeTruthy()
    cleanup()

    setup({ 'GET /terminology/review': () => apiError(500, 'internal error') })
    expect((await screen.findByRole('alert')).textContent).toBe('The review queue is not available. Try again.')
  })

  it('has no edit or approve controls for a staff member who is not a reviewer', async () => {
    setup({}, { reviewer: false })
    await screen.findByRole('heading', { name: 'jetra' })

    expect(screen.queryByRole('button', { name: 'Approve' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Edit' })).toBeNull()
    expect(screen.getByText('Only reviewers can edit and approve terms.')).toBeTruthy()
    fireEvent.keyDown(entries()[0], { key: 'e' })
    expect(screen.queryByRole('textbox')).toBeNull()
  })
})

describe('reviewing with the keyboard', () => {
  it('moves between entries with the arrows and j and k', async () => {
    setup()
    await screen.findByRole('heading', { name: 'jetra' })
    const [first, second, third] = entries()
    first.focus()

    fireEvent.keyDown(first, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(second)
    fireEvent.keyDown(second, { key: 'j' })
    expect(document.activeElement).toBe(third)
    fireEvent.keyDown(third, { key: 'k' })
    expect(document.activeElement).toBe(second)
    fireEvent.keyDown(second, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(first)
  })

  it('approves the labels with a, If-Match the version read, and moves to the next entry', async () => {
    let body: unknown
    let ifMatch: string | null = null
    const fake = setup({
      [`POST /terminology/terms/${liverID}/translations/hr/review`]: async (r) => {
        body = await r.json()
        ifMatch = r.headers.get('If-Match')
        return Response.json(translation(liverID, { review: { status: 'approved', reviewedAt: at }, versionId: 2 }))
      },
    })
    await screen.findByRole('heading', { name: 'jetra' })
    const [first, second] = entries()
    first.focus()

    fireEvent.keyDown(first, { key: 'a' })

    await vi.waitFor(() => expect(part(first, 'Labels').textContent).toContain('Approved'))
    expect(body).toEqual({ status: 'approved' })
    expect(ifMatch).toBe('W/"1"')
    expect(document.activeElement).toBe(second)
    expectCsrfOnWrites(fake.requests)
    expect(fake.requests.filter((r) => new URL(r.url).pathname === '/terminology/review/progress').length).toBe(2)
  })

  it('edits the labels with e: Enter saves the preferred label and the synonyms, the entry turns unreviewed', async () => {
    let body: unknown
    let ifMatch: string | null = null
    setup({
      [`PUT /terminology/terms/${heartID}/translations/hr`]: async (r) => {
        body = await r.json()
        ifMatch = r.headers.get('If-Match')
        return Response.json(translation(heartID, { review: { status: 'unreviewed', reviewedAt: at }, versionId: 2 }))
      },
    })
    await screen.findByRole('heading', { name: 'srce' })
    const third = entries()[2]
    third.focus()

    fireEvent.keyDown(third, { key: 'e' })
    const preferred = within(third).getByLabelText('Preferred label') as HTMLInputElement
    expect(document.activeElement).toBe(preferred)
    fireEvent.change(preferred, { target: { value: '  srce  ' } })
    fireEvent.change(within(third).getByLabelText('Synonyms, separated by |'), { target: { value: 'cor | kardija ' } })
    fireEvent.submit(preferred.form!)

    await vi.waitFor(() => expect(part(third, 'Labels').textContent).toContain('Unreviewed'))
    expect(body).toEqual({ preferred: 'srce', synonyms: ['cor', 'kardija'] })
    expect(ifMatch).toBe('W/"1"')
    expect(part(third, 'Labels').textContent).toContain('cor, kardija')
    expect(document.activeElement).toBe(third)
  })

  it('adds the first label of a term with W/"0"', async () => {
    let ifMatch: string | null = null
    setup({
      [`PUT /terminology/terms/${ulcerID}/translations/hr`]: (r) => {
        ifMatch = r.headers.get('If-Match')
        return Response.json(translation(ulcerID))
      },
    })
    await screen.findByRole('heading', { name: 'jetra' })
    const second = entries()[1]
    fireEvent.keyDown(second, { key: 'e' })
    fireEvent.change(within(second).getByLabelText('Preferred label'), { target: { value: 'čir na želucu' } })
    fireEvent.submit(within(second).getByLabelText('Preferred label').closest('form')!)

    await within(second).findByRole('heading', { name: 'čir na želucu' })
    expect(ifMatch).toBe('W/"0"')
  })

  it('cancels an edit with Escape and gives the focus back to the entry', async () => {
    const fake = setup()
    await screen.findByRole('heading', { name: 'jetra' })
    const first = entries()[0]
    fireEvent.keyDown(first, { key: 'e' })
    const preferred = within(first).getByLabelText('Preferred label')
    fireEvent.keyDown(preferred, { key: 'a' }) // typing: no shortcut

    fireEvent.keyDown(preferred, { key: 'Escape' })

    expect(within(first).queryByLabelText('Preferred label')).toBeNull()
    expect(document.activeElement).toBe(first)
    expect(fake.requests.every((r) => r.method === 'GET')).toBe(true)
  })

  it('requests changes on the labels with n and a note, and refuses an empty note', async () => {
    let body: unknown
    setup({
      [`POST /terminology/terms/${liverID}/translations/hr/review`]: async (r) => {
        body = await r.json()
        return Response.json(translation(liverID, { review: { status: 'needs_changes', reviewedAt: at, note: 'Placeholder note' }, versionId: 2 }))
      },
    })
    await screen.findByRole('heading', { name: 'jetra' })
    const first = entries()[0]
    fireEvent.keyDown(first, { key: 'n' })
    const note = within(first).getByLabelText('What needs to change')

    fireEvent.submit(note.closest('form')!)
    expect(screen.getByRole('alert').textContent).toBe('Write what needs to change.')
    fireEvent.change(note, { target: { value: 'Placeholder note' } })
    fireEvent.submit(note.closest('form')!)

    await vi.waitFor(() => expect(part(first, 'Labels').textContent).toContain('Needs changes'))
    expect(body).toEqual({ status: 'needs_changes', note: 'Placeholder note' })
    expect(part(first, 'Labels').textContent).toContain('Note: Placeholder note')
    expect(first.className).toContain('needs_changes')
  })

  it('writes an explanation with Shift+E and approves one with Shift+A', async () => {
    let written: unknown
    let ifMatch: string | null = null
    setup({
      [`PUT /terminology/terms/${liverID}/explanations/hr`]: async (r) => {
        written = await r.json()
        ifMatch = r.headers.get('If-Match')
        return Response.json(explanation(liverID, { text: 'Placeholder text' }))
      },
      [`POST /terminology/terms/${heartID}/explanations/hr/review`]: () =>
        Response.json(explanation(heartID, { review: { status: 'approved', reviewedAt: at }, versionId: 2 })),
    })
    await screen.findByRole('heading', { name: 'jetra' })
    const [first, , third] = entries()

    fireEvent.keyDown(first, { key: 'E', shiftKey: true })
    const text = within(first).getByLabelText('Explanation text')
    fireEvent.change(text, { target: { value: ' Placeholder text ' } })
    fireEvent.keyDown(text, { key: 'Enter' })
    await vi.waitFor(() => expect(within(first).queryByLabelText('Explanation text')).toBeNull())
    expect(written).toEqual({ text: 'Placeholder text' })
    expect(part(first, 'Explanation for patients (informational)').textContent).toContain('Placeholder text')
    expect(ifMatch).toBe('W/"0"')

    fireEvent.keyDown(third, { key: 'A', shiftKey: true })
    await vi.waitFor(() => expect(part(third, 'Explanation for patients (informational)').textContent).toContain('Approved'))
    expect(third.className).toContain('approved')
  })

  it('reloads the queue on a version conflict and says why', async () => {
    let reads = 0
    setup({
      'GET /terminology/review': () => {
        reads++
        return Response.json({ entries: [liver, ulcer, heart] })
      },
      [`POST /terminology/terms/${liverID}/translations/hr/review`]: () => apiError(412, 'version_conflict'),
    })
    await screen.findByRole('heading', { name: 'jetra' })
    fireEvent.keyDown(entries()[0], { key: 'a' })

    expect((await screen.findByRole('alert')).textContent).toBe('Someone changed this entry in the meantime. The queue has been reloaded.')
    await vi.waitFor(() => expect(reads).toBe(3)) // English, Croatian, and Croatian again
  })

  it('says a refused write is for reviewers', async () => {
    setup({ [`POST /terminology/terms/${liverID}/translations/hr/review`]: () => apiError(403, 'forbidden') })
    await screen.findByRole('heading', { name: 'jetra' })
    fireEvent.click(within(entries()[0]).getAllByRole('button', { name: 'Approve' })[0])

    expect((await screen.findByRole('alert')).textContent).toBe('Only reviewers can edit and approve terms.')
  })
})

describe('in another language', () => {
  it('names the languages and statuses in Bosnian', async () => {
    setup({}, { ui: 'bs' })
    await screen.findByRole('heading', { name: 'jetra' })
    expect(screen.getByRole('option', { name: 'Njemački' })).toBeTruthy()
  })
})
