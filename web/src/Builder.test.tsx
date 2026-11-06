import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient, type Questionnaire } from './api/fhir'
import { Builder } from './Builder'
import type { SaveRequest } from './questionnaire'
import { navigate, usePath } from './route'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { searchset } from './test/fixtures'
import { questionnaireFHIR, questionnaireJSON } from './test/questionnaires'

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

function renderBuilder(routes: Record<string, Route>, q: Questionnaire = draft()) {
  const fake = fakeFetch({ [read]: () => fhirJSON(q), [series]: () => fhirJSON(searchset([q])), ...routes })
  render(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />)
  return fake
}

function list(): HTMLElement {
  const ul = document.querySelector<HTMLElement>('ul.items')
  if (!ul) throw new Error('no item list')
  return ul
}

// toggles are the buttons that open and close the item cards, named by the item's text (or type).
function toggles(): HTMLElement[] {
  return within(list())
    .queryAllByRole('button')
    .filter((b) => b.classList.contains('item-toggle'))
}

function cards(): string[] {
  return toggles().map((b) => b.getAttribute('aria-label') ?? '')
}

function toggle(text: string): HTMLElement {
  const b = toggles().find((x) => x.getAttribute('aria-label') === text)
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

const saved = (extra = {}) => () => Response.json({ ...questionnaireJSON(), versionId: 4, ...extra })

describe('Builder: a draft', () => {
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
    fireEvent.change(within(card('Placeholder single')).getByRole('combobox', { name: 'Placeholder single' }), { target: { value: 'multiple' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
    const changed = (await saveBody(requests)).item[2].item?.[0]
    expect(changed).toMatchObject({ type: 'choice', repeats: true, text: 'Placeholder single' })
    expect(changed?.linkId).not.toBe('one')
    expect(changed?.linkId).toMatch(/^[A-Za-z0-9-]{1,64}$/)
  })
})

describe('Builder: moving items', () => {
  it('disables Move up on the first item and Move down on the last', async () => {
    renderBuilder({})
    await screen.findByLabelText('Title')

    expect((within(card('Placeholder instructions')).getByRole('button', { name: 'Move up' }) as HTMLButtonElement).disabled).toBe(true)
    expect((within(card('Placeholder instructions')).getByRole('button', { name: 'Move down' }) as HTMLButtonElement).disabled).toBe(false)
    expect((within(card('Placeholder section')).getByRole('button', { name: 'Move down' }) as HTMLButtonElement).disabled).toBe(true)
    expect((within(card('Placeholder multiple')).getByRole('button', { name: 'Move down' }) as HTMLButtonElement).disabled).toBe(false)
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
    const confirm = within(card('Placeholder single')).getByRole('group', { name: 'Delete this question?' })
    expect(within(confirm).getByText('1 items are shown only after an answer to it. Their conditions will be removed.')).toBeTruthy()
    fireEvent.click(within(confirm).getByRole('button', { name: 'Cancel' }))
    expect(cards()).toContain('Placeholder single')

    fireEvent.click(within(card('Placeholder single')).getByRole('button', { name: 'Delete' }))
    const again = within(card('Placeholder single')).getByRole('group', { name: 'Delete this question?' })
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
  it('asks to confirm a publish, and keeps the draft on Cancel', async () => {
    const { requests } = renderBuilder({})
    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))

    expect(screen.getByText('Publish this questionnaire?')).toBeTruthy()
    expect(screen.getByText('Once published, it cannot be changed. To change it later, you create a new version.')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('group', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Cancel' }))
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
    render(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />)

    fireEvent.click(await screen.findByRole('button', { name: 'Publish' }))
    expect(screen.getByText('Version 1 will be retired. Patients who have already been asked keep that version.')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('group', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

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
    fireEvent.click(within(screen.getByRole('group', { name: 'Publish this questionnaire?' })).getByRole('button', { name: 'Publish' }))

    expect(await screen.findByText('This questionnaire is published and cannot be changed.')).toBeTruthy()
    expect(paths(requests).filter((p) => !p.startsWith('GET'))).toEqual(['PUT /questionnaires/q1', 'POST /questionnaires/q1/publish'])
    expect(requests.find((r) => r.method === 'POST')?.headers.get('If-Match')).toBe('W/"4"')
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

  it('offers the existing draft on 409 draft_exists', async () => {
    const active = draft({ status: 'active' })
    const fake = fakeFetch({
      [read]: () => fhirJSON(active),
      [series]: () => fhirJSON(searchset([active, draft({ id: 'q9', version: '2' })])),
      'POST /questionnaires/q1/revise': () => apiError(409, 'draft_exists'),
    })
    render(<Builder api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} id="q1" />)

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
    fireEvent.click(within(screen.getByRole('group', { name: 'Retire this questionnaire?' })).getByRole('button', { name: 'Retire' }))

    expect(await screen.findByText('This questionnaire is retired. It is no longer sent to patients.')).toBeTruthy()
    expectCsrfOnWrites(requests)
  })

  it('discards a draft after a confirmation and goes back to the library', async () => {
    const { requests } = renderBuilder({ 'DELETE /questionnaires/q1': () => new Response(null, { status: 204 }) })
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })
    fireEvent.click(screen.getByRole('button', { name: 'Discard draft' }))
    expect(screen.getByText('Discard this draft?')).toBeTruthy()
    expect(screen.getByText('This cannot be undone.')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('group', { name: 'Discard this draft?' })).getByRole('button', { name: 'Discard' }))

    await vi.waitFor(() => expect(location.pathname).toBe('/questionnaires'))
    expectCsrfOnWrites(requests)
  })

  it('says a questionnaire it cannot read is not available', async () => {
    renderBuilder({ [read]: () => outcome(404, 'not-found') })
    expect((await screen.findByRole('alert')).textContent).toBe('This questionnaire is not available.')
  })
})

describe('Builder: leaving with unsaved changes', () => {
  it('asks before leaving, stays on Stay, and leaves on Leave', async () => {
    renderBuilder({})
    fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'Placeholder unsaved' } })

    act(() => navigate('/'))
    expect(location.pathname).toBe(path)
    expect(screen.getByText('You have unsaved changes. Leave without saving?')).toBeTruthy()
    fireEvent.click(within(screen.getByRole('group', { name: 'You have unsaved changes. Leave without saving?' })).getByRole('button', { name: 'Stay' }))
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
    render(<Shell />)
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
