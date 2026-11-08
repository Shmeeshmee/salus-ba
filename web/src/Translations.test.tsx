import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { Builder } from './Builder'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import type { FHIRQuestionnaire, SaveRequest } from './questionnaire'
import { fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { searchset } from './test/fixtures'
import { questionnaireFHIR, translatedFHIR, translatedJSON } from './test/questionnaires'

// The builder's languages (COPY.md B1 to B4, B8): the original language, adding and removing translations, and
// the side-by-side editing of one translation. The publishing prompts (B5 to B7) are in Builder.test.tsx.

afterEach(cleanup)
beforeEach(() => history.replaceState(null, '', '/questionnaires/q1'))

function renderBuilder(routes: Record<string, Route> = {}, q: FHIRQuestionnaire = translatedFHIR(), language: Language = 'en') {
  const fake = fakeFetch({
    'GET /fhir/R4/Questionnaire/q1': () => fhirJSON(q),
    'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset([q])),
    'PUT /questionnaires/q1': () => Response.json({ ...translatedJSON(), versionId: 4 }),
    ...routes,
  })
  renderInLanguage(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />, language)
  return fake
}

async function saved(requests: Request[]): Promise<SaveRequest> {
  fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))
  await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
  return (await requests.filter((r) => r.method === 'PUT').at(-1)?.json()) as SaveRequest
}

const languages = () => screen.getByRole('region', { name: 'Languages' })

// row is the line of the translation editor that shows the original text.
function row(original: string): HTMLElement {
  const el = within(document.querySelector('.translation') as HTMLElement)
    .getAllByText(original)
    .find((x) => x.classList.contains('original-text'))
  if (!el) throw new Error(`no row ${original}`)
  return el.closest('.translation-row') as HTMLElement
}

const input = (original: string) => within(row(original)).getByRole('textbox') as HTMLInputElement

describe('Builder: languages', () => {
  it('shows the original language and how far each translation is, quietly (B1, B2, B4)', async () => {
    renderBuilder()

    const select = (await screen.findByLabelText('Original language')) as HTMLSelectElement
    expect(select.value).toBe('bs')
    const offered = Array.from(select.options).map((o) => o.textContent)
    expect(offered).toEqual(['Bosanski', 'Hrvatski', 'Srpski (latinica)', 'Српски (ћирилица)', 'العربية', 'Türkçe', 'Italiano', 'Français'])
    expect(within(languages()).getByRole('button', { name: 'English: 9 of 12 texts translated' })).toBeTruthy()
    expect(within(languages()).getByRole('button', { name: 'Deutsch: 1 of 12 texts translated' })).toBeTruthy()
    expect(within(languages()).getByRole('button', { name: 'Original (Bosanski)' }).getAttribute('aria-pressed')).toBe('true')
  })

  it('offers no language list to switch between while there is no translation', async () => {
    renderBuilder({}, questionnaireFHIR())

    expect(await screen.findByLabelText('Original language')).toBeTruthy()
    expect(within(languages()).queryByRole('button', { name: /^Original/ })).toBeNull()
    expect(within(languages()).getByRole('button', { name: 'Add translation' })).toBeTruthy()
  })

  it('saves the original language chosen', async () => {
    const { requests } = renderBuilder()
    fireEvent.change(await screen.findByLabelText('Original language'), { target: { value: 'hr' } })

    expect((await saved(requests)).language).toBe('hr')
  })

  it('adds a translation in a language not yet used (B3), and opens it side by side with the original (B8)', async () => {
    const { requests } = renderBuilder()
    fireEvent.click(await within(await screen.findByRole('region', { name: 'Languages' })).findByRole('button', { name: 'Add translation' }))

    const offered = within(languages()).getAllByRole('button').map((b) => b.textContent)
    expect(offered).toContain('Hrvatski')
    expect(offered).not.toContain('English')
    expect(offered).not.toContain('Bosanski')
    fireEvent.click(within(languages()).getByRole('button', { name: 'Hrvatski' }))

    expect(within(languages()).getByRole('button', { name: 'Hrvatski: 0 of 12 texts translated' }).getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByText('Original (Bosanski)', { selector: '.translation *' })).toBeTruthy()
    expect(screen.queryByRole('heading', { name: 'Add an item' })).toBeNull()
    fireEvent.change(input('Placeholder intake'), { target: { value: 'Placeholder intake in Croatian' } })
    expect(within(languages()).getByRole('button', { name: 'Hrvatski: 1 of 12 texts translated' })).toBeTruthy()

    const body = await saved(requests)
    expect(body.translations?.map((t) => [t.language, t.title])).toEqual([
      ['en', 'Placeholder intake in English'],
      ['de', 'Placeholder intake in German'],
      ['hr', 'Placeholder intake in Croatian'],
    ])
  })

  it('edits a translation item by item, the original next to it, until it is complete', async () => {
    const { requests } = renderBuilder()
    fireEvent.click(await within(await screen.findByRole('region', { name: 'Languages' })).findByRole('button', { name: 'English: 9 of 12 texts translated' }))

    expect(input('Placeholder intake').value).toBe('Placeholder intake in English')
    expect(input('Placeholder help').value).toBe('Placeholder help in English')
    expect(input('A').value).toBe('A in English')
    expect(input('Placeholder multiple').value).toBe('')
    expect(input('Placeholder multiple').getAttribute('lang')).toBe('en')
    fireEvent.change(input('Placeholder multiple'), { target: { value: 'Placeholder multiple in English' } })
    fireEvent.change(input('C'), { target: { value: 'C in English' } })
    fireEvent.change(input('D'), { target: { value: 'D in English' } })

    expect(within(languages()).getByRole('button', { name: 'English: complete' })).toBeTruthy()
    const body = await saved(requests)
    expect(body.translations?.[0].items.many).toEqual({ text: 'Placeholder multiple in English', options: ['C in English', 'D in English'] })
  })

  it('goes back to the items from the original', async () => {
    renderBuilder()
    fireEvent.click(await within(await screen.findByRole('region', { name: 'Languages' })).findByRole('button', { name: 'Deutsch: 1 of 12 texts translated' }))
    expect(screen.queryByRole('heading', { name: 'Add an item' })).toBeNull()

    fireEvent.click(within(languages()).getByRole('button', { name: 'Original (Bosanski)' }))
    expect(screen.getByRole('heading', { name: 'Add an item' })).toBeTruthy()
    expect(document.querySelector('.translation')).toBeNull()
  })

  it('removes a translation after a confirmation (B4), and saves without it', async () => {
    const { requests } = renderBuilder()
    const remove = within(await screen.findByRole('region', { name: 'Languages' })).getAllByRole('button', { name: 'Remove translation' })
    fireEvent.click(remove[0])

    const dialog = screen.getByRole('alertdialog', { name: 'Remove the English translation?' })
    expect(within(dialog).getByText('Its texts are deleted from this draft.')).toBeTruthy()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove' }))

    expect(within(languages()).queryByRole('button', { name: /^English/ })).toBeNull()
    expect((await saved(requests)).translations?.map((t) => t.language)).toEqual(['de'])
  })

  it('keeps what the save answers: a text whose original changed is no longer translated', async () => {
    const answer = translatedJSON()
    delete answer.translations?.[0].items.smoker
    renderBuilder({ 'PUT /questionnaires/q1': () => Response.json({ ...answer, versionId: 4 }) })
    fireEvent.click(await screen.findByRole('button', { name: 'Save draft' }))

    expect(await within(languages()).findByRole('button', { name: 'English: 7 of 12 texts translated' })).toBeTruthy()
  })

  it('is in the UI language', async () => {
    renderBuilder({}, translatedFHIR(), 'hr')

    expect(await screen.findByLabelText('Izvorni jezik')).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Jezici' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'English: prevedeno 9 od 12 tekstova' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Izvornik (Bosanski)' })).toBeTruthy()
  })
})

