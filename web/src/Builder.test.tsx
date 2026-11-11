import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { newApiClient } from './api/client'
import { newFhirClient, type Questionnaire } from './api/fhir'
import appCSS from './app.css?raw'
import { Builder } from './Builder'
import type { SaveRequest } from './questionnaire'
import { navigate, usePath } from './route'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { searchset } from './test/fixtures'
import { questionnaireFHIR, questionnaireJSON, translatedFHIR, translatedJSON } from './test/questionnaires'

const path = '/questionnaires/q1'
const read = 'GET /fhir/R4/Questionnaire/q1'
const series = 'GET /fhir/R4/Questionnaire'

afterEach(cleanup)
beforeEach(() => history.replaceState(null, '', path))

// Rows get a place on the page from their position in the list, so @dnd-kit's keyboard moves find the row above and below.
beforeEach(() => {
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const row = this.closest('ul.items > li')
    const i = row ? Array.from(row.parentElement?.children ?? []).indexOf(row) : -1
    const top = i < 0 ? 0 : i * 60
    const height = i < 0 ? 0 : 50
    return { x: 0, y: top, top, left: 0, width: 400, height, right: 400, bottom: top + height, toJSON: () => ({}) } as DOMRect
  })
  return () => vi.restoreAllMocks()
})

function draft(extra: Partial<Questionnaire> = {}): Questionnaire {
  return { ...questionnaireFHIR(), ...extra }
}

function renderBuilder(routes: Record<string, Route>, q: Questionnaire = draft(), language: Language = 'en') {
  const fake = fakeFetch({ [read]: () => fhirJSON(q), [series]: () => fhirJSON(searchset([q])), ...routes })
  renderInLanguage(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />, language)
  return fake
}

function list(): HTMLElement {
  const ul = document.querySelector<HTMLElement>('ul.items')
  if (!ul) throw new Error('no item list')
  return ul
}

// toggles are the buttons that open and close the item cards, showing the item's text (or type).
function toggles(): HTMLElement[] {
  return within(list())
    .queryAllByRole('button')
    .filter((b) => b.classList.contains('item-toggle'))
}

const textOf = (b: HTMLElement) => b.querySelector('.item-text')?.textContent ?? ''

function cards(): string[] {
  return toggles().map(textOf)
}

function toggle(text: string): HTMLElement {
  const b = toggles().find((x) => textOf(x) === text)
  if (!b) throw new Error(`no card ${text}`)
  return b
}

function card(text: string) {
  return toggle(text).closest('li') as HTMLElement
}

function saveBody(requests: Request[]) {
  const put = requests.filter((r) => r.method === 'PUT').at(-1)
  return put?.json() as Promise<SaveRequest>
}

// description is the text an element's aria-describedby points at.
function description(el: HTMLElement): string {
  const ids = el.getAttribute('aria-describedby')?.split(' ') ?? []
  return ids.map((id) => document.getElementById(id)?.textContent ?? '').join(' ')
}

function deferred() {
  let resolve: () => void = () => undefined
  const promise = new Promise<void>((r) => (resolve = r))
  return { promise, resolve }
}

// withoutTranslation answers the question a publish with no translation asks first (B5): publish without.
function withoutTranslation() {
  fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Add a translation before publishing?' })).getByRole('button', { name: 'Publish without' }))
}

const saved = (extra = {}) => () => Response.json({ ...questionnaireJSON(), versionId: 4, ...extra })

describe('Builder: a draft', () => {
  it('is in the UI language, the problems on the items and the screen reader texts included', async () => {
    renderBuilder({}, draft(), 'hr')

    expect(await screen.findByLabelText('Naslov')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Spremi nacrt' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Kratki odgovor' }))
    expect(screen.getByText('Dodajte tekst ovoj stavci.')).toBeTruthy()
    fireEvent.click(screen.getAllByRole('button', { name: 'Pomakni gore' }).at(-1) as HTMLElement)
    expect(document.querySelector('[role="status"][aria-live="polite"]')?.textContent).toMatch(/^Stavka Kratki odgovor je /)
  })

  it('shows the title, the template hint, the palette and the items, each closed', async () => {
    renderBuilder({})

    expect(((await screen.findByLabelText('Title')) as HTMLInputElement).value).toBe('Placeholder intake')
    expect(screen.getByText("This questionnaire is a template for all patients. Do not include any patient's details.")).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Add an item' })).toBeTruthy()
    expect(screen.getByText('Drag an item into the questionnaire, or select it to add it at the end.')).toBeTruthy()
    expect(screen.getByText('5 of 200 items')).toBeTruthy()
    expect(cards()).toEqual(['Placeholder instructions', 'Placeholder yes or no', 'Placeholder section', 'Placeholder single', 'Placeholder multiple'])
    expect(within(card('Placeholder yes or no')).getByText('Yes or no')).toBeTruthy()
    expect(within(card('Placeholder yes or no')).getByText('Required')).toBeTruthy()
    expect(screen.queryByLabelText('Question')).toBeNull()
  })

  it("names a card's toggle by what it shows: the text, the type and Required", async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')

    const b = screen.getByRole('button', { name: 'Placeholder yes or no Yes or no Required' })
    expect(b.getAttribute('aria-label')).toBeNull()
    expect(screen.getByRole('button', { name: 'Placeholder single Single choice' })).toBeTruthy()
  })

  it('labels the type of an opened card "Type" (W13)', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder single'))

    const select = within(card('Placeholder single')).getByLabelText('Type') as HTMLSelectElement
    expect(select.value).toBe('single')
    expect(within(card('Placeholder single')).getByText('Type').tagName).toBe('LABEL')
  })

  it('says an empty section needs an item (W28), and does not offer to publish it', async () => {
    const { requests } = renderBuilder({}, draft({ item: [{ linkId: 'sec', text: 'Placeholder section', type: 'group' }] }))
    await screen.findByLabelText('Title')

    expect(within(card('Placeholder section')).getByText('Add at least one item to this section.')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }))

    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(screen.getByRole('alert').textContent).toBe('Some items need attention.')
    expect(card('Placeholder section').classList.contains('attention')).toBe(true)
    expect(requests.some((r) => r.method !== 'GET')).toBe(false)
  })

  it('says an empty option needs filling in (W28)', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder single'))
    fireEvent.change(within(card('Placeholder single')).getByLabelText('Option 2'), { target: { value: '' } })

    expect(within(card('Placeholder single')).getByText('Fill in this option.')).toBeTruthy()
  })

  it('opens a card to edit it, with help text and the condition behind "More options"', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder multiple'))

    const c = card('Placeholder multiple')
    expect((within(c).getByLabelText('Question') as HTMLInputElement).value).toBe('Placeholder multiple')
    expect((within(c).getByLabelText('Option 1') as HTMLInputElement).value).toBe('C')
    expect(within(c).queryByLabelText('Help text')).toBeNull()

    fireEvent.click(within(c).getByText('More options'))
    expect(within(c).getByRole('button', { name: 'Add help text' })).toBeTruthy()
    expect(within(c).getByText('Show this only if')).toBeTruthy()
    expect((within(c).getByLabelText('Show this only if') as HTMLSelectElement).value).toBe('one')
    expect((within(c).getByLabelText('is') as HTMLSelectElement).value).toBe('B')

    fireEvent.click(within(c).getByRole('button', { name: 'Add help text' }))
    expect(within(c).getByLabelText('Help text')).toBeTruthy()
    expect(within(c).getByText('Patients see this when they tap the information icon.')).toBeTruthy()

    expect(toggle('Placeholder multiple').getAttribute('aria-expanded')).toBe('true')
    fireEvent.click(toggle('Placeholder multiple'))
    expect(toggle('Placeholder multiple').getAttribute('aria-expanded')).toBe('false')
    expect(within(c).queryByLabelText('Question')).toBeNull()
  })

  it('says when there is nothing a condition can depend on', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder instructions'))
    const c = card('Placeholder instructions')
    fireEvent.click(within(c).getByText('More options'))

    expect(within(c).getByText('A condition needs a yes or no question or a choice question above this item.')).toBeTruthy()
    expect(within(c).queryByRole('button', { name: 'Add condition' })).toBeNull()
    expect(within(c).queryByRole('button', { name: 'Add help text' })).toBeNull()
  })

  it('adds a type selected in the palette at the end, open to edit', async () => {
    renderBuilder({})
    fireEvent.click(await screen.findByRole('button', { name: 'Date' }))

    expect(cards().at(-1)).toBe('Date')
    expect(screen.getByText('6 of 200 items')).toBeTruthy()
    expect(document.activeElement).toBe(within(card('Date')).getByLabelText('Question'))
  })

  it('gives an item a new linkId when its type changes, and saves it', async () => {
    const { requests } = renderBuilder({ 'PUT /questionnaires/q1': saved() })
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder single'))
    fireEvent.change(within(card('Placeholder single')).getByLabelText('Type'), { target: { value: 'multiple' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
    const changed = (await saveBody(requests)).item[2].item?.[0]
    expect(changed).toMatchObject({ type: 'choice', repeats: true, text: 'Placeholder single' })
    expect(changed?.linkId).not.toBe('one')
    expect(changed?.linkId).toMatch(/^[A-Za-z0-9-]{1,64}$/)
  })
})

describe('Builder: a file upload item', () => {
  const upload = (extra = {}) =>
    draft({
      item: [
        {
          linkId: 'f',
          text: 'Placeholder upload',
          type: 'attachment',
          repeats: true,
          extension: [
            { url: 'http://hl7.org/fhir/StructureDefinition/mimeType', valueCode: 'application/pdf' },
            { url: 'http://hl7.org/fhir/StructureDefinition/questionnaire-maxOccurs', valueInteger: 3 },
          ],
          ...extra,
        },
      ],
    })

  it('is offered in the palette as "File upload" (W1), and opens with the accepted kinds and the number of files (W2 to W4)', async () => {
    renderBuilder({})
    fireEvent.click(await screen.findByRole('button', { name: 'File upload' }))

    const c = card('File upload')
    expect((within(c).getByLabelText('Type') as HTMLSelectElement).value).toBe('attachment')
    const kinds = within(c).getByRole('group', { name: 'Accepted files' })
    expect(within(kinds).getAllByRole('checkbox').map((b) => [b.closest('div')?.textContent, (b as HTMLInputElement).checked])).toEqual([
      ['Photos', true],
      ['Documents', true],
      ['Videos', false],
    ])
    expect(description(kinds)).toBe('Photos: JPEG or PNG, up to 20 MB. Documents: PDF, up to 20 MB. Videos: MP4, up to 200 MB.')
    const n = within(c).getByLabelText('Number of files') as HTMLInputElement
    expect(n.value).toBe('1')
    expect([n.min, n.max, n.step]).toEqual(['1', '10', '1'])
    expect(description(n)).toBe('From 1 to 10.')
    expect(within(c).getByLabelText('Required')).toBeTruthy()
    expect(within(card('File upload')).getAllByText('1 file')).toHaveLength(1)
  })

  it('reads the item and sums it up on the closed card after the type (W5)', async () => {
    renderBuilder({}, upload())
    await screen.findByLabelText('Title')
    expect(screen.getByRole('button', { name: 'Placeholder upload File upload Up to 3 files' })).toBeTruthy()

    fireEvent.click(toggle('Placeholder upload'))
    const c = card('Placeholder upload')
    expect(within(c).getAllByRole('checkbox', { checked: true }).map((b) => b.closest('div')?.textContent)).toEqual(['Documents'])
    expect((within(c).getByLabelText('Number of files') as HTMLInputElement).value).toBe('3')
  })

  it.each([
    ['no kind of file', (c: HTMLElement) => fireEvent.click(within(c).getByLabelText('Documents')), 'Choose at least one kind of file.'],
    ['11 files', (c: HTMLElement) => fireEvent.change(within(c).getByLabelText('Number of files'), { target: { value: '11' } }), 'The number of files must be from 1 to 10.'],
    ['0 files', (c: HTMLElement) => fireEvent.change(within(c).getByLabelText('Number of files'), { target: { value: '0' } }), 'The number of files must be from 1 to 10.'],
    ['an empty number', (c: HTMLElement) => fireEvent.change(within(c).getByLabelText('Number of files'), { target: { value: '' } }), 'The number of files must be from 1 to 10.'],
  ])('says so on the item with %s (W6), and does not offer to publish', async (_, change, message) => {
    const { requests } = renderBuilder({}, upload())
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder upload'))
    change(card('Placeholder upload'))

    expect(within(card('Placeholder upload')).getByText(message)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }))
    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(requests.some((r) => r.method !== 'GET')).toBe(false)
  })

  it('saves accept and maxFiles as chosen', async () => {
    const { requests } = renderBuilder({ 'PUT /questionnaires/q1': saved() }, upload())
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder upload'))
    const c = card('Placeholder upload')
    fireEvent.click(within(c).getByLabelText('Videos'))
    fireEvent.click(within(c).getByLabelText('Photos'))
    fireEvent.change(within(c).getByLabelText('Number of files'), { target: { value: '10' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
    expect((await saveBody(requests)).item).toEqual([
      { linkId: 'f', text: 'Placeholder upload', type: 'attachment', accept: ['photo', 'document', 'video'], maxFiles: 10 },
    ])
  })

  it('shows the summary on a published version too', async () => {
    renderBuilder({}, { ...upload(), status: 'active' })
    expect(await screen.findByText('Up to 3 files')).toBeTruthy()
    expect(screen.getByText('File upload')).toBeTruthy()
  })
})

describe('Builder: moving items', () => {
  // aria-disabled, not disabled: a button that becomes disabled under the focus drops it to the page.
  it('marks Move up on the first item and Move down on the last as unavailable, and ignores them', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')

    const unavailable = (text: string, name: string) => {
      const b = within(card(text)).getByRole('button', { name }) as HTMLButtonElement
      return !b.disabled && b.getAttribute('aria-disabled') === 'true'
    }
    expect(unavailable('Placeholder instructions', 'Move up')).toBe(true)
    expect(unavailable('Placeholder instructions', 'Move down')).toBe(false)
    expect(unavailable('Placeholder section', 'Move down')).toBe(true)
    expect(unavailable('Placeholder multiple', 'Move down')).toBe(false)

    const before = cards()
    fireEvent.click(within(card('Placeholder instructions')).getByRole('button', { name: 'Move up' }))
    expect(cards()).toEqual(before)
    expect(document.querySelector('[role="status"][aria-live]')?.textContent).toBe('')
  })

  it('keeps the focus on Move down when the item reaches the end', async () => {
    renderBuilder({}, draft({ item: [{ linkId: 'a', text: 'Placeholder A', type: 'string' }, { linkId: 'b', text: 'Placeholder B', type: 'string' }] }))
    await screen.findByLabelText('Title')
    const down = within(card('Placeholder A')).getByRole('button', { name: 'Move down' })
    down.focus()
    fireEvent.click(down)

    expect(cards()).toEqual(['Placeholder B', 'Placeholder A'])
    expect(document.activeElement).toBe(within(card('Placeholder A')).getByRole('button', { name: 'Move down' }))
    expect(document.activeElement?.getAttribute('aria-disabled')).toBe('true')
  })

  it('moves with Move up and Move down, and announces where the item is (W42)', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')

    fireEvent.click(within(card('Placeholder instructions')).getByRole('button', { name: 'Move down' }))
    expect(cards()).toEqual(['Placeholder yes or no', 'Placeholder instructions', 'Placeholder section', 'Placeholder single', 'Placeholder multiple'])
    expect(screen.getByText('Placeholder instructions moved to position 2 of 3.')).toBeTruthy()

    fireEvent.click(within(card('Placeholder instructions')).getByRole('button', { name: 'Move down' }))
    expect(screen.getByText('Placeholder instructions is in Placeholder section, at position 1 of 3.')).toBeTruthy()
  })

  it('reads the drag instructions of W41 on the handle', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')

    const handle = within(card('Placeholder yes or no')).getByRole('button', { name: 'Drag to reorder' })
    // @dnd-kit portals its hidden instructions into the page once it has mounted.
    await vi.waitFor(() =>
      expect(document.getElementById(handle.getAttribute('aria-describedby') ?? '')?.textContent).toBe(
        'To move this item, press space. Use the arrow keys to move it, space to drop it, or escape to cancel.',
      ),
    )
    expect(handle.getAttribute('aria-roledescription')).toBeNull()
  })

  it('moves by keyboard drag: space, arrow down, space, and announces it (W42)', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    await dndMounted()
    const handle = within(card('Placeholder instructions')).getByRole('button', { name: 'Drag to reorder' })

    handle.focus()
    await act(async () => fireEvent.keyDown(handle, { code: 'Space' }))
    expect(liveRegion()).toBe('Moving Placeholder instructions.')
    await sensorReady()
    await act(async () => fireEvent.keyDown(document, { code: 'ArrowDown' }))
    await vi.waitFor(() => expect(liveRegion()).toBe('Placeholder instructions is at position 2 of 3.'))
    await act(async () => fireEvent.keyDown(document, { code: 'Space' }))

    await vi.waitFor(() => expect(cards()[1]).toBe('Placeholder instructions'))
    expect(liveRegion()).toBe('Placeholder instructions moved to position 2 of 3.')
  })

  it('cancels a keyboard drag with escape, and says the item is back', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    await dndMounted()
    const handle = within(card('Placeholder instructions')).getByRole('button', { name: 'Drag to reorder' })

    handle.focus()
    await act(async () => fireEvent.keyDown(handle, { code: 'Space' }))
    await sensorReady()
    await act(async () => fireEvent.keyDown(document, { code: 'ArrowDown' }))
    await vi.waitFor(() => expect(liveRegion()).toBe('Placeholder instructions is at position 2 of 3.'))
    await act(async () => fireEvent.keyDown(document, { code: 'Escape' }))

    await vi.waitFor(() => expect(liveRegion()).toBe('Move cancelled. Placeholder instructions is back at position 1 of 3.'))
    expect(cards()[0]).toBe('Placeholder instructions')
  })
})

// dndMounted waits for @dnd-kit's live region. findByLabelText returns once the loaded builder is on the page, which
// can be before React has run that render's effects: Testing Library's own wait (a setTimeout) and React's scheduler
// (a setImmediate) run in either order. @dnd-kit subscribes its announcer to the drag events in such an effect, and
// mounts the live region in the same one, so a drag started before then loses "Moving …" and announces its own
// place instead.
async function dndMounted() {
  await vi.waitFor(() => expect(document.querySelector('[id^="DndLiveRegion"]')).not.toBeNull())
}

// sensorReady lets @dnd-kit's keyboard sensor attach its key listener, which it does on the next task after a
// drag starts.
async function sensorReady() {
  await act(() => new Promise((resolve) => setTimeout(resolve)))
}

// liveRegion is the text @dnd-kit last announced.
function liveRegion(): string {
  return document.querySelector('[id^="DndLiveRegion"]')?.textContent ?? ''
}

describe('Builder: deleting', () => {
  it('deletes an item nothing depends on at once', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    fireEvent.click(within(card('Placeholder instructions')).getByRole('button', { name: 'Delete' }))
    expect(cards()).not.toContain('Placeholder instructions')
  })

  it('warns before deleting a question others depend on, and removes their conditions', async () => {
    const { requests } = renderBuilder({ 'PUT /questionnaires/q1': saved() })
    await screen.findByLabelText('Title')

    fireEvent.click(within(card('Placeholder single')).getByRole('button', { name: 'Delete' }))
    const confirm = within(card('Placeholder single')).getByRole('alertdialog', { name: 'Delete this question?' })
    expect(description(confirm)).toBe('1 item is shown only after an answer to it. Its condition will be removed.')
    expect(confirm.contains(document.activeElement)).toBe(true)
    fireEvent.click(within(confirm).getByRole('button', { name: 'Cancel' }))
    expect(cards()).toContain('Placeholder single')

    fireEvent.click(within(card('Placeholder single')).getByRole('button', { name: 'Delete' }))
    const again = within(card('Placeholder single')).getByRole('alertdialog', { name: 'Delete this question?' })
    fireEvent.click(within(again).getByRole('button', { name: 'Delete' }))
    expect(cards()).not.toContain('Placeholder single')

    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))
    await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
    expect((await saveBody(requests)).item[2].item?.[0]).toEqual({
      linkId: 'many',
      text: 'Placeholder multiple',
      type: 'choice',
      repeats: true,
      answerOption: [{ valueString: 'C' }, { valueString: 'D' }],
    })
  })
})

describe('Builder: saving', () => {
  it('saves with If-Match of the version read and the CSRF header, then of the version saved', async () => {
    let version = 3
    const { requests } = renderBuilder({
      'PUT /questionnaires/q1': () => {
        version++
        return Response.json({ ...questionnaireJSON(), title: 'Placeholder renamed', versionId: version })
      },
    })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    expect(await screen.findByText('Saved')).toBeTruthy()
    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Placeholder renamed again' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))
    await vi.waitFor(() => expect(requests.filter((r) => r.method === 'PUT')).toHaveLength(2))

    const puts = requests.filter((r) => r.method === 'PUT')
    expect(puts.map((r) => r.headers.get('If-Match'))).toEqual(['W/"3"', 'W/"4"'])
    expect((await puts[0].json()).title).toBe('Placeholder renamed')
    expectCsrfOnWrites(requests)
  })

  it('keeps an edit made while a save is in flight unsaved, and asks before leaving', async () => {
    const put = deferred()
    const { requests } = renderBuilder({ 'PUT /questionnaires/q1': () => put.promise.then(saved()) })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))
    await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Placeholder renamed again' } })
    await act(async () => put.resolve())

    expect(screen.queryByText('Saved')).toBeNull()
    act(() => navigate('/'))
    expect(location.pathname).toBe(path)
    expect(screen.getByRole('alertdialog', { name: 'You have unsaved changes. Leave without saving?' })).toBeTruthy()
  })

  it('says someone else changed it on a 412, and reloads', async () => {
    const { requests } = renderBuilder({ 'PUT /questionnaires/q1': () => apiError(412, 'version_conflict') })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder mine' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    expect((await screen.findByRole('alert')).textContent).toBe(
      'Someone else changed this questionnaire. Reload to see their changes. Your changes since the last save will be lost.',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Reload' }))
    await vi.waitFor(() => expect(paths(requests).filter((p) => p === read)).toHaveLength(2))
    expect(((await screen.findByLabelText('Title')) as HTMLInputElement).value).toBe('Placeholder intake')
  })

  it('says a published version can no longer be changed (409 not_draft)', async () => {
    renderBuilder({ 'PUT /questionnaires/q1': () => apiError(409, 'not_draft') })
    await screen.findByLabelText('Title')
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))
    expect((await screen.findByRole('alert')).textContent).toBe('This version is published and can no longer be changed.')
  })

  it('shows a 422 on the items it names', async () => {
    renderBuilder({ 'PUT /questionnaires/q1': () => Response.json({ error: 'invalid_questionnaire', linkIds: ['one'] }, { status: 422 }) })
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder single'))
    fireEvent.change(within(card('Placeholder single')).getByLabelText('Option 1'), { target: { value: 'B' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Some items need attention.')
    expect(card('Placeholder single').classList.contains('attention')).toBe(true)
    expect(card('Placeholder multiple').classList.contains('attention')).toBe(false)
    expect(within(card('Placeholder single')).getByText('Each option must be different.')).toBeTruthy()
  })
})

describe('Builder: limits', () => {
  it('stops adding at 200 items and says why', async () => {
    const many = Array.from({ length: 200 }, (_, i) => ({ linkId: `i${i}`, text: `Placeholder ${i}`, type: 'string' }))
    renderBuilder({}, draft({ item: many }))

    expect(await screen.findByText('200 of 200 items')).toBeTruthy()
    expect(screen.getByText('A questionnaire can have up to 200 items.')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Date' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('counts the characters of a title near its limit', async () => {
    renderBuilder({})
    const title = await screen.findByLabelText('Title')
    expect(screen.queryByText(/of 200 characters/)).toBeNull()
    fireEvent.change(title, { target: { value: 'x'.repeat(170) } })
    expect(screen.getByText('170 of 200 characters')).toBeTruthy()
    expect(title.getAttribute('maxLength')).toBe('200')
  })

  it('stops adding options at 50 and says why', async () => {
    const options = Array.from({ length: 50 }, (_, i) => ({ valueString: `O${i}` }))
    renderBuilder({}, draft({ item: [{ linkId: 'c', text: 'Placeholder choice', type: 'choice', answerOption: options }] }))
    await screen.findByLabelText('Title')
    fireEvent.click(toggle('Placeholder choice'))

    expect((screen.getByRole('button', { name: 'Add option' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText('A choice question can have up to 50 options.')).toBeTruthy()
  })
})

describe('Builder: publishing and the lifecycle', () => {
  it('asks in an alert dialog that takes the focus, and gives it back on Cancel', async () => {
    renderBuilder({})
    const publish = await screen.findByRole('button', { name: 'Publish' })
    publish.focus()
    fireEvent.click(publish)
    withoutTranslation()

    const dialog = screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })
    expect(description(dialog)).toBe('Once published, it cannot be changed. To change it later, you create a new version.')
    expect(dialog.contains(document.activeElement)).toBe(true)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(document.activeElement).toBe(publish)
  })

  it('asks to confirm a publish, and keeps the draft on Cancel', async () => {
    const { requests } = renderBuilder({})
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    withoutTranslation()

    expect(screen.getByText('Publish this questionnaire?')).toBeTruthy()
    expect(screen.getByText('Once published, it cannot be changed. To change it later, you create a new version.')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText('Publish this questionnaire?')).toBeNull()
    expect(requests.some((r) => r.method === 'POST')).toBe(false)
  })

  it('says which version a publish retires, then publishes with If-Match', async () => {
    const v2 = draft({ id: 'q1', version: '2' })
    const v1 = draft({ id: 'q0', version: '1', status: 'active' })
    const fake = fakeFetch({
      [read]: () => fhirJSON(v2),
      [series]: () => fhirJSON(searchset([v1, v2])),
      'POST /questionnaires/q1/publish': () => Response.json({ ...questionnaireJSON(), version: 2, status: 'active', versionId: 4 }),
    })
    renderInLanguage(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />, 'en')

    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    withoutTranslation()
    expect(screen.getByText('Version 1 will be retired. Patients who have already been asked keep that version.')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
    const post = fake.requests.find((r) => r.method === 'POST')
    expect(post?.headers.get('If-Match')).toBe('W/"3"')
    expectCsrfOnWrites(fake.requests)
  })

  it('saves unsaved changes before it publishes', async () => {
    const { requests } = renderBuilder({
      'PUT /questionnaires/q1': saved(),
      'POST /questionnaires/q1/publish': () => Response.json({ ...questionnaireJSON(), status: 'active', versionId: 5 }),
    })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }))
    withoutTranslation()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
    expect(paths(requests).filter((p) => !p.startsWith('GET'))).toEqual(['PUT /questionnaires/q1', 'POST /questionnaires/q1/publish'])
    expect(requests.find((r) => r.method === 'POST')?.headers.get('If-Match')).toBe('W/"4"')
  })

  it('shows a publish refused with 422 on the section it names', async () => {
    renderBuilder({ 'POST /questionnaires/q1/publish': () => Response.json({ error: 'invalid_questionnaire', linkIds: ['sec'] }, { status: 422 }) })
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    withoutTranslation()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Some items need attention.')
    expect(card('Placeholder section').classList.contains('attention')).toBe(true)
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('keeps editing out of reach while it publishes, so no edit is lost to the published version', async () => {
    const put = deferred()
    renderBuilder({
      'PUT /questionnaires/q1': () => put.promise.then(saved()),
      'POST /questionnaires/q1/publish': () => Response.json({ ...questionnaireJSON(), status: 'active', versionId: 5 }),
    })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }))
    withoutTranslation()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    await vi.waitFor(() => expect(document.querySelector('.builder')?.hasAttribute('inert')).toBe(true))
    expect(screen.getByLabelText('Title').closest('[inert]')).toBeTruthy()
    put.resolve()
    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
  })

  it('opens the new version after "Edit as new version"', async () => {
    renderBuilder(
      { 'POST /questionnaires/q1/revise': () => Response.json({ ...questionnaireJSON(), id: 'q2', version: 2 }, { status: 201 }) },
      draft({ status: 'active' }),
    )
    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Save draft' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Drag to reorder' })).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Edit as new version' }))
    await vi.waitFor(() => expect(location.pathname).toBe('/questionnaires/q2'))
  })

  it('leaves the same space under "Edit as new version" and "Retire" as under the other action rows (W23)', async () => {
    const style = document.createElement('style')
    style.textContent = appCSS
    document.head.append(style)
    try {
      renderBuilder({}, draft({ status: 'active' }))
      const row = (await screen.findByRole('button', { name: 'Edit as new version' })).closest('.actions') as HTMLElement
      const items = document.querySelector('ul.items') as HTMLElement
      expect(row.nextElementSibling).toBe(items)
      // Block margins collapse: the space between the row and the list is the larger of the two.
      const space = Math.max(parseFloat(getComputedStyle(row).marginBottom) || 0, parseFloat(getComputedStyle(items).marginTop) || 0)
      expect(space).toBe(16)
      expect(getComputedStyle(row).marginTop).toBe('16px')
    } finally {
      style.remove()
    }
  })

  it('offers the existing draft on 409 draft_exists', async () => {
    const active = draft({ status: 'active' })
    const fake = fakeFetch({
      [read]: () => fhirJSON(active),
      [series]: () => fhirJSON(searchset([active, draft({ id: 'q9', version: '2' })])),
      'POST /questionnaires/q1/revise': () => apiError(409, 'draft_exists'),
    })
    renderInLanguage(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />, 'en')

    fireEvent.click(await screen.findByRole('button', { name: 'Edit as new version' }))
    expect((await screen.findByRole('alert')).textContent).toBe('A draft of this questionnaire already exists.')
    fireEvent.click(screen.getByRole('button', { name: 'Open draft' }))
    expect(location.pathname).toBe('/questionnaires/q9')
  })

  it('retires an active version after a confirmation', async () => {
    const { requests } = renderBuilder(
      { 'POST /questionnaires/q1/retire': () => Response.json({ ...questionnaireJSON(), status: 'retired', versionId: 5 }) },
      draft({ status: 'active' }),
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Retire' }))
    expect(screen.getByText('Retire this questionnaire?')).toBeTruthy()
    expect(
      screen.getByText(
        'It is no longer sent with new bookings or added to appointments, including as a default questionnaire. Patients who have already been asked can still submit it.',
      ),
    ).toBeTruthy()
    expect(screen.getByRole('alertdialog', { name: 'Retire this questionnaire?' }).contains(document.activeElement)).toBe(true)
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Retire this questionnaire?' })).getByRole('button', { name: 'Retire' }))

    expect(await screen.findByText('This questionnaire is retired. It is no longer sent to patients.')).toBeTruthy()
    expectCsrfOnWrites(requests)
  })

  it('discards a draft after a confirmation and goes back to the library', async () => {
    const { requests } = renderBuilder({ 'DELETE /questionnaires/q1': () => new Response(null, { status: 204 }) })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })
    fireEvent.click(screen.getByRole('button', { name: 'Discard draft' }))
    expect(screen.getByText('Discard this draft?')).toBeTruthy()
    expect(screen.getByText('This cannot be undone.')).toBeTruthy()
    expect(description(screen.getByRole('alertdialog', { name: 'Discard this draft?' }))).toBe('This cannot be undone.')
    expect(screen.getByRole('alertdialog', { name: 'Discard this draft?' }).contains(document.activeElement)).toBe(true)
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Discard this draft?' })).getByRole('button', { name: 'Discard' }))

    await vi.waitFor(() => expect(location.pathname).toBe('/questionnaires'))
    expectCsrfOnWrites(requests)
  })

  it('says a questionnaire it cannot read is not available', async () => {
    renderBuilder({ [read]: () => outcome(404, 'not-found') })
    expect((await screen.findByRole('alert')).textContent).toBe('This questionnaire is not available.')
  })
})

describe('Builder: publishing and the translations', () => {
  const complete = () => {
    const q = translatedFHIR()
    const many = q.item?.[2].item?.[1]
    if (!many) throw new Error('no multiple choice item')
    const en = (content: string) => ({ extension: [{ url: 'http://hl7.org/fhir/StructureDefinition/translation', extension: [{ url: 'lang', valueCode: 'en' }, { url: 'content', valueString: content }] }] })
    many._text = en('Placeholder multiple in English')
    many.answerOption = [
      { valueString: 'C', _valueString: en('C in English') },
      { valueString: 'D', _valueString: en('D in English') },
    ]
    return { ...q, _title: { extension: q._title?.extension?.slice(0, 1) } }
  }

  it('asks first whether to add a translation when there is none (B5)', async () => {
    const { requests } = renderBuilder({})
    const publish = await screen.findByRole('button', { name: 'Publish' })
    publish.focus()
    fireEvent.click(publish)

    const dialog = screen.getByRole('alertdialog', { name: 'Add a translation before publishing?' })
    expect(description(dialog)).toBe("Patients who use Salus in another language will be asked to choose one of this questionnaire's languages.")
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add translation' }))

    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(within(screen.getByRole('region', { name: 'Languages' })).getByRole('button', { name: 'English' })).toBeTruthy()
    expect(requests.some((r) => r.method === 'POST')).toBe(false)
  })

  it('publishes without a translation when told to, after the usual confirmation', async () => {
    const { requests } = renderBuilder({ 'POST /questionnaires/q1/publish': () => Response.json({ ...questionnaireJSON(), status: 'active', versionId: 4 }) })
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    withoutTranslation()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
    expect(paths(requests).filter((p) => !p.startsWith('GET'))).toEqual(['POST /questionnaires/q1/publish'])
  })

  it('does not offer to publish an incomplete translation: says which (B6) and marks what it lacks (B7)', async () => {
    const { requests } = renderBuilder({}, translatedFHIR())
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))

    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(screen.getByRole('alert').textContent).toBe('The English translation is not complete. Finish it or remove it to publish.')
    expect(within(card('Placeholder multiple')).getByText('Missing in English')).toBeTruthy()
    expect(card('Placeholder multiple').classList.contains('attention')).toBe(true)
    expect(within(card('Placeholder single')).queryByText('Missing in English')).toBeNull()
    expect(requests.some((r) => r.method === 'POST')).toBe(false)
  })

  it('takes the marks away once the translation is removed', async () => {
    renderBuilder({}, translatedFHIR())
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    fireEvent.click(within(screen.getByRole('region', { name: 'Languages' })).getAllByRole('button', { name: 'Remove translation' })[0])
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Remove the English translation?' })).getByRole('button', { name: 'Remove' }))

    expect(screen.queryByText('Missing in English')).toBeNull()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('asks the usual confirmation when every translation is complete', async () => {
    renderBuilder({}, complete())
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))

    expect(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).toBeTruthy()
  })

  it('shows a publish refused with 422 translation_incomplete on the items it names', async () => {
    renderBuilder(
      {
        'PUT /questionnaires/q1': () => Response.json({ ...translatedJSON(), versionId: 4 }),
        'POST /questionnaires/q1/publish': () => Response.json({ error: 'translation_incomplete', language: 'en', linkIds: ['many'] }, { status: 422 }),
      },
      complete(),
    )
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }))
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect((await screen.findByRole('alert')).textContent).toBe('The English translation is not complete. Finish it or remove it to publish.')
    expect(card('Placeholder multiple').classList.contains('attention')).toBe(true)
    expect(within(card('Placeholder multiple')).getByText('Missing in English')).toBeTruthy()
  })

  it('says it in the UI language', async () => {
    renderBuilder({}, translatedFHIR(), 'bs')
    fireEvent.click(await screen.findByRole('button', { name: 'Objavi' }))

    expect(screen.getByRole('alert').textContent).toBe('Prijevod (English) nije završen. Završite ga ili ga uklonite da biste objavili upitnik.')
    expect(within(card('Placeholder multiple')).getByText('Nedostaje prijevod (English)')).toBeTruthy()
  })
})

describe('Builder: leaving with unsaved changes', () => {
  it('asks in an alert dialog that takes the focus', async () => {
    renderBuilder({})
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })
    act(() => navigate('/'))

    const dialog = screen.getByRole('alertdialog', { name: 'You have unsaved changes. Leave without saving?' })
    expect(dialog.contains(document.activeElement)).toBe(true)
    expect(document.activeElement?.textContent).toBe('Stay')
  })

  it('asks before leaving, stays on Stay, and leaves on Leave', async () => {
    renderBuilder({})
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })

    act(() => navigate('/'))
    expect(location.pathname).toBe(path)
    expect(screen.getByText('You have unsaved changes. Leave without saving?')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('alertdialog', { name: 'You have unsaved changes. Leave without saving?' })).getByRole('button', { name: 'Stay' }))
    expect(location.pathname).toBe(path)
    expect(screen.queryByText('You have unsaved changes. Leave without saving?')).toBeNull()

    act(() => navigate('/'))
    fireEvent.click(screen.getByRole('button', { name: 'Leave' }))
    expect(location.pathname).toBe('/')
  })

  it("asks on the browser's back, and stays on the page", async () => {
    const fake = fakeFetch({ [read]: () => fhirJSON(draft()), [series]: () => fhirJSON(searchset([draft()])) })
    function Shell() {
      return usePath() === path ? <Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" /> : <p>Elsewhere</p>
    }
    renderInLanguage(<Shell />, 'en')
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })

    act(() => {
      history.pushState(null, '', '/questionnaires') // what the browser's back does before popstate
      globalThis.dispatchEvent(new PopStateEvent('popstate'))
    })
    expect(location.pathname).toBe(path)
    expect(screen.getByText('You have unsaved changes. Leave without saving?')).toBeTruthy()
    expect(screen.queryByText('Elsewhere')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Leave' }))
    expect(location.pathname).toBe('/questionnaires')
    expect(screen.getByText('Elsewhere')).toBeTruthy()
  })

  it('leaves without asking when everything is saved', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')
    act(() => navigate('/'))
    expect(location.pathname).toBe('/')
  })
})
