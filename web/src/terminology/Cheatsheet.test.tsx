import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from '../api/client'
import type { components } from '../api/schema'
import { renderInLanguage } from '../i18n/renderInLanguage'
import type { Language } from '../i18n/languages'
import { Cheatsheet } from './Cheatsheet'

type Term = components['schemas']['Term']

// Synthetic terms: a liver with labels in every column, a draft ulcer with no English label.
const liver: Term = {
  id: '00000000-0000-4000-8000-000000000001',
  category: 'anatomy',
  reviewStatus: 'reviewed',
  codes: { wikidata: 'Q9368', ta98: 'A05.8.01.001' },
  labels: [
    { language: 'la', preferred: 'hepar', synonyms: ['iecur'] },
    { language: 'hr', preferred: 'jetra', synonyms: [] },
    { language: 'en', preferred: 'liver', synonyms: [] },
    { language: 'bs', preferred: 'jetra', synonyms: [] },
    { language: 'sr-Latn', preferred: 'jetra', synonyms: [] },
    { language: 'sr-Cyrl', preferred: 'јетра', synonyms: [] },
  ],
}
const ulcer: Term = {
  id: '00000000-0000-4000-8000-000000000002',
  category: 'diagnosis',
  reviewStatus: 'draft',
  codes: { wikidata: 'Q12356', icd10: 'K25' },
  labels: [
    { language: 'la', preferred: 'ulcus ventriculi', synonyms: [] },
    { language: 'hr', preferred: 'čir na želucu', synonyms: [] },
  ],
}

type Answer = (q: string) => Response | Promise<Response>

// setup renders the cheatsheet in language against a fake API that answers each search with answer(q) and keeps
// every request (the requests themselves, so their abort signals can be read).
function setup(language: Language, answer: Answer = () => Response.json({ terms: [liver, ulcer] })) {
  const requests: Request[] = []
  const api = newApiClient(async (request) => {
    requests.push(request)
    if (new URL(request.url).pathname !== '/terminology/search') throw new Error(`unexpected ${request.url}`)
    return answer(new URL(request.url).searchParams.get('q') ?? '')
  })
  renderInLanguage(<Cheatsheet api={api} />, language)
  return { requests }
}

const box = () => screen.getByRole('combobox')
const options = () => screen.queryAllByRole('option')
const type = (text: string) => fireEvent.change(box(), { target: { value: text } })

// search types text, lets the debounce run out and waits for the answer to be shown.
async function search(text: string) {
  type(text)
  act(() => vi.advanceTimersByTime(150))
  await vi.waitFor(() => expect(options().length + screen.queryAllByRole('alert').length + screen.queryAllByText(/No terms found/).length).toBeGreaterThan(0))
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  cleanup()
})

describe('opening and closing', () => {
  it('opens from the header button, named with the shortcut, and closes on Escape, giving the focus back', () => {
    setup('en')
    const button = screen.getByRole('button', { name: 'Search medical terms (Ctrl+K)' })
    expect(screen.queryByRole('dialog')).toBeNull()

    button.focus()
    fireEvent.click(button)

    expect(screen.getByRole('dialog', { name: 'Medical terms' })).toBeTruthy()
    expect(document.activeElement).toBe(box())
    expect(box().getAttribute('placeholder')).toBe('Search in Latin, Bosnian, Croatian, Serbian or English')

    fireEvent.keyDown(box(), { key: 'Escape' })

    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(button)
  })

  it('opens on Ctrl+K and on ⌘K from anywhere on the page', () => {
    setup('en')

    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    expect(screen.getByRole('dialog')).toBeTruthy()
    fireEvent.keyDown(box(), { key: 'Escape' })

    fireEvent.keyDown(document.body, { key: 'K', metaKey: true })
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('leaves a plain K alone', () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('names the shortcut ⌘K on macOS', () => {
    vi.spyOn(navigator, 'platform', 'get').mockReturnValue('MacIntel')
    setup('en')
    expect(screen.getByRole('button', { name: 'Search medical terms (⌘K)' })).toBeTruthy()
  })

  it('closes with its close button', () => {
    setup('hr')
    fireEvent.click(screen.getByRole('button', { name: 'Pretraži medicinske pojmove (Ctrl+K)' }))
    fireEvent.click(within(screen.getByRole('dialog', { name: 'Medicinski pojmovi' })).getByRole('button', { name: 'Zatvori' }))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('names the source in its footer', () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    expect(within(screen.getByRole('dialog')).getByText('Terms from Wikidata. Not every term has been reviewed yet.')).toBeTruthy()
  })
})

describe('searching', () => {
  it('waits 150 ms after the last change, then asks once for Latin, the UI language, English and the others', async () => {
    const { requests } = setup('hr')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })

    type('je')
    act(() => vi.advanceTimersByTime(149))
    type('jetra')
    act(() => vi.advanceTimersByTime(149))
    expect(requests).toHaveLength(0)

    act(() => vi.advanceTimersByTime(1))
    await vi.waitFor(() => expect(options()).toHaveLength(2))

    expect(requests).toHaveLength(1)
    const url = new URL(requests[0].url)
    expect(url.searchParams.get('q')).toBe('jetra')
    expect(url.searchParams.get('languages')).toBe('la,hr,en,bs,sr-Latn,sr-Cyrl')
  })

  it('cancels a search when the text changes, and never shows its late answer', async () => {
    const late: ((r: Response) => void)[] = []
    const { requests } = setup('en', (q) => (q === 'jetra' ? new Promise<Response>((resolve) => late.push(resolve)) : Response.json({ terms: [ulcer] })))
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })

    type('jetra')
    act(() => vi.advanceTimersByTime(150))
    await vi.waitFor(() => expect(late).toHaveLength(1))
    await search('cir')
    await act(async () => late[0](Response.json({ terms: [liver] })))

    expect(requests[0].signal.aborted).toBe(true)
    expect(requests[1].signal.aborted).toBe(false)
    expect(options()).toHaveLength(1)
    expect(within(options()[0]).getByText('ulcus ventriculi')).toBeTruthy()
  })

  it('asks nothing for blank text and clears the results', async () => {
    const { requests } = setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('hepar')

    type('   ')
    act(() => vi.advanceTimersByTime(500))

    expect(requests).toHaveLength(1)
    expect(options()).toHaveLength(0)
  })

  it('says when nothing is found', async () => {
    setup('en', () => Response.json({ terms: [] }))
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('xyz')
    expect(screen.getByText('No terms found. Try fewer letters or another language.')).toBeTruthy()
  })

  it('says when the search fails, without repeating the text', async () => {
    setup('sr-Latn', () => Response.json({ error: 'internal' }, { status: 500 }))
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('hepar')
    const alert = screen.getByRole('alert')
    expect(alert.textContent).toBe('Pretraga trenutno nije dostupna. Pokušajte ponovo.')
  })

  it('keeps the text to 100 characters', () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    expect(box().getAttribute('maxLength') ?? box().getAttribute('maxlength')).toBe('100')
  })
})

// cells is each language cell of an option as "language: preferred label".
function cells(option: HTMLElement): string[] {
  return Array.from(option.querySelectorAll('.term-label')).map(
    (c) => `${c.querySelector('.term-language')?.textContent}: ${c.querySelector('.term-preferred, .term-missing')?.textContent}`,
  )
}

describe('the language columns', () => {
  it('shows Latin, the UI language and English, then the others on More languages', async () => {
    setup('hr')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('jetra')

    expect(cells(options()[0])).toEqual(['Latinski: hepar', 'Hrvatski: jetra', 'Engleski: liver'])
    expect(cells(options()[1])).toEqual(['Latinski: ulcus ventriculi', 'Hrvatski: čir na želucu', 'Engleski: Engleski: još nema izraza'])

    fireEvent.click(screen.getByRole('button', { name: 'Više jezika' }))
    expect(cells(options()[0])).toEqual([
      'Latinski: hepar',
      'Hrvatski: jetra',
      'Engleski: liver',
      'Bosanski: jetra',
      'Srpski (latinica): jetra',
      'Srpski (ćirilica): јетра',
    ])

    fireEvent.click(screen.getByRole('button', { name: 'Manje jezika' }))
    expect(cells(options()[0])).toHaveLength(3)
  })

  it('shows the Cyrillic label in the Serbian Cyrillic UI', async () => {
    setup('sr-Cyrl')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('јетра')
    expect(cells(options()[0])).toEqual(['Латински: hepar', 'Српски (ћирилица): јетра', 'Енглески: liver'])
  })

  it('has two columns in the English UI', async () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('liver')
    expect(cells(options()[0])).toEqual(['Latin: hepar', 'English: liver'])
    expect(cells(options()[1])).toEqual(['Latin: ulcus ventriculi', 'English: No term in English yet'])
  })

  it('shows synonyms, the category, the codes behind a tooltip, and marks a draft term', async () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('hepar')
    const [first, second] = options()

    expect(within(first).getByText('iecur')).toBeTruthy()
    expect(within(first).getByText('Anatomy')).toBeTruthy()
    expect(within(first).getByRole('tooltip').textContent).toBe('TA98 A05.8.01.001')
    expect(within(first).queryByText('Not reviewed yet')).toBeNull()

    expect(within(second).getByText('Diagnosis')).toBeTruthy()
    expect(within(second).getByRole('tooltip').textContent).toBe('ICD-10 K25')
    expect(within(second).getByText('Not reviewed yet')).toBeTruthy()
  })
})

describe('copying', () => {
  let writeText: ReturnType<typeof vi.fn>
  beforeEach(() => {
    writeText = vi.fn(() => Promise.resolve())
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  })

  it('moves with the arrows and copies the UI-language label of the chosen term on Enter', async () => {
    setup('hr')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('jetra')
    const selected = () => options().findIndex((o) => o.getAttribute('aria-selected') === 'true')
    expect(selected()).toBe(0)
    expect(box().getAttribute('aria-activedescendant')).toBe(options()[0].id)

    fireEvent.keyDown(box(), { key: 'ArrowDown' })
    expect(selected()).toBe(1)
    expect(box().getAttribute('aria-activedescendant')).toBe(options()[1].id)
    fireEvent.keyDown(box(), { key: 'ArrowDown' })
    expect(selected()).toBe(1)
    fireEvent.keyDown(box(), { key: 'ArrowUp' })
    fireEvent.keyDown(box(), { key: 'ArrowUp' })
    expect(selected()).toBe(0)
    fireEvent.keyDown(box(), { key: 'ArrowDown' })

    await act(async () => fireEvent.keyDown(box(), { key: 'Enter' }))

    expect(writeText).toHaveBeenCalledWith('čir na želucu')
    expect(screen.getByRole('status').textContent).toBe('Kopirano')
  })

  it('copies any shown label with its button', async () => {
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('hepar')
    const latin = options()[0].querySelector('.term-label') as HTMLElement

    await act(async () => fireEvent.click(within(latin).getByRole('button', { name: 'Copy' })))

    expect(writeText).toHaveBeenCalledWith('hepar')
    expect(within(latin).getByRole('button', { name: 'Copied' })).toBeTruthy()
    expect(within(options()[0]).getAllByRole('button', { name: 'Copy' })).toHaveLength(1)
  })

  it('does not say Copied when the clipboard refuses', async () => {
    writeText.mockImplementation(() => Promise.reject(new Error('denied')))
    setup('en')
    fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true })
    await search('hepar')

    await act(async () => fireEvent.keyDown(box(), { key: 'Enter' }))

    expect(writeText).toHaveBeenCalledWith('liver')
    expect(screen.queryByText('Copied')).toBeNull()
  })
})
