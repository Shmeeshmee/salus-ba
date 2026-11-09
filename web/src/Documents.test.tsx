import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { ConsultSide } from './ConsultPanels'
import { useConsult } from './consult'
import { renderInLanguage } from './i18n/renderInLanguage'
import { draft, fakeActions, fakeLive, segment, templates } from './test/fakeConsult'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
const summary = templates[0]

let scrolled: Element[]
const scrollIntoView = Element.prototype.scrollIntoView
afterEach(() => {
  Element.prototype.scrollIntoView = scrollIntoView
})
beforeEach(() => {
  scrolled = []
  Element.prototype.scrollIntoView = function (this: Element) {
    scrolled.push(this)
  }
})

type Fake = ReturnType<typeof fakeLive>

function Side({ fake, actions }: { fake: Fake; actions: ReturnType<typeof fakeActions>['actions'] }) {
  const [state, dispatch] = useConsult(fake.live, id, true)
  return <ConsultSide appointmentId={id} state={state} dispatch={dispatch} actions={actions} zone="Europe/Sarajevo" />
}

// openDocuments renders the side panel on the Documents tab.
function openDocuments(options: Parameters<typeof fakeActions>[0] = {}) {
  const fake = fakeLive()
  const acts = fakeActions(options)
  renderInLanguage(<Side fake={fake} actions={acts.actions} />, 'en')
  fireEvent.click(screen.getByRole('tab', { name: 'Documents' }))
  return { ...fake, ...acts, emit: (...m: Parameters<Fake['emit']>) => act(() => fake.emit(...m)) }
}

// generated picks the visit summary, generates it and delivers its draft on the live source.
async function generated(options: Parameters<typeof fakeActions>[0] = {}) {
  const docs = openDocuments(options)
  fireEvent.change(await screen.findByRole('combobox', { name: 'Template' }), { target: { value: summary.id } })
  fireEvent.click(screen.getByRole('button', { name: 'Generate' }))
  await screen.findByText('Generating the draft from the transcript…')
  docs.emit(draft())
  await screen.findByRole('textbox', { name: 'Placeholder reason' })
  return docs
}

const field = (name: string) => screen.getByRole('textbox', { name }) as HTMLTextAreaElement

describe('Documents: templates', () => {
  it("offers the clinic's templates, and Generate once one is chosen", async () => {
    openDocuments()

    expect(screen.getByText('Loading the templates…')).toBeTruthy()
    const select = (await screen.findByRole('combobox', { name: 'Template' })) as HTMLSelectElement
    expect([...select.options].map((o) => o.textContent)).toEqual(['Choose a template', 'Placeholder visit summary', 'Placeholder referral'])
    expect((screen.getByRole('button', { name: 'Generate' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.change(select, { target: { value: summary.id } })
    expect((screen.getByRole('button', { name: 'Generate' }) as HTMLButtonElement).disabled).toBe(false)
  })

  it('says when the clinic has none', async () => {
    openDocuments({ templates: [] })
    expect(await screen.findByText('This clinic has no document templates yet.')).toBeTruthy()
  })

  it('says when they cannot be loaded', async () => {
    openDocuments({ templates: 'fail' })
    expect(await screen.findByText('The templates could not be loaded. Try again later.')).toBeTruthy()
  })
})

describe('Documents: the draft', () => {
  it('generates the chosen template for this appointment and shows its fields when the draft arrives', async () => {
    const { generated: calls } = await generated()

    expect(calls).toEqual([[id, summary.id]])
    expect(screen.queryByText('Generating the draft from the transcript…')).toBeNull()
    expect(screen.getByText('Drafted from the transcript. Check every field before you sign.')).toBeTruthy()
    expect(field('Placeholder reason').value).toBe('Placeholder reason for the visit')
    expect(field('Placeholder plan').value).toBe('')
  })

  it('flags a field the conversation has not covered yet', async () => {
    await generated()

    const plan = field('Placeholder plan')
    const flag = document.getElementById(plan.getAttribute('aria-describedby') ?? '')
    expect(flag?.textContent).toBe('Not covered in the conversation yet')
    expect(field('Placeholder reason').getAttribute('aria-describedby')).toBeNull()
  })

  it("shows a field's source lines in the transcript", async () => {
    const { emit } = await generated()
    emit(segment({ segment_id: 'a1000000-0000-4000-8000-000000000001', original: 'Placeholder cited line' }))

    fireEvent.click(screen.getByRole('button', { name: 'Show the source line' }))

    expect(screen.getByRole('tab', { name: 'Transcript' }).getAttribute('aria-selected')).toBe('true')
    const line = within(screen.getByRole('list', { name: 'Transcript' })).getByRole('listitem')
    expect(line.getAttribute('aria-current')).toBe('true')
    expect(scrolled).toEqual([line])
  })

  it('drops the sources and the flag of a field the practitioner edits', async () => {
    await generated()

    fireEvent.change(field('Placeholder reason'), { target: { value: 'Placeholder typed reason' } })
    fireEvent.change(field('Placeholder plan'), { target: { value: 'Placeholder typed plan' } })

    expect(screen.queryByRole('button', { name: 'Show the source line' })).toBeNull()
    expect(screen.queryByText('Not covered in the conversation yet')).toBeNull()
  })

  it('says when the draft cannot be generated', async () => {
    openDocuments({ failGenerate: true })
    fireEvent.change(await screen.findByRole('combobox', { name: 'Template' }), { target: { value: summary.id } })
    fireEvent.click(screen.getByRole('button', { name: 'Generate' }))

    expect((await screen.findByRole('alert')).textContent).toBe('The draft could not be generated. Try again.')
    expect(screen.queryByText('Generating the draft from the transcript…')).toBeNull()
  })
})

describe('Documents: regenerate', () => {
  it('regenerates at once when nothing was edited, and shows the new draft when it arrives', async () => {
    const { emit, generated: calls } = await generated()

    fireEvent.click(screen.getByRole('button', { name: 'Regenerate' }))

    expect(await screen.findByText('Generating the draft from the transcript…')).toBeTruthy()
    expect(calls).toHaveLength(2)
    emit(draft({ fields: [{ key: 'reason', value: 'Placeholder new reason', missing: false, source_segment_ids: ['s'] }, { key: 'plan', value: 'Placeholder new plan', missing: false, source_segment_ids: ['s'] }] }))
    expect(await screen.findByDisplayValue('Placeholder new plan')).toBeTruthy()
    expect(screen.queryByText('Generating the draft from the transcript…')).toBeNull()
  })

  it('asks first when it would replace edits, and keeps them on cancel', async () => {
    const { emit, generated: calls } = await generated()
    fireEvent.change(field('Placeholder reason'), { target: { value: 'Placeholder typed reason' } })

    fireEvent.click(screen.getByRole('button', { name: 'Regenerate' }))
    const ask = screen.getByRole('alertdialog', { name: 'Regenerate the draft?' })
    expect(ask.textContent).toContain('Your changes to its fields are replaced by a new draft from the transcript.')
    fireEvent.click(within(ask).getByRole('button', { name: 'Keep editing' }))
    expect(calls).toHaveLength(1)
    expect(field('Placeholder reason').value).toBe('Placeholder typed reason')

    fireEvent.click(screen.getByRole('button', { name: 'Regenerate' }))
    fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Regenerate' }))
    await screen.findByText('Generating the draft from the transcript…')
    expect(calls).toHaveLength(2)
    emit(draft())
    await screen.findByText('Drafted from the transcript. Check every field before you sign.')
    expect(field('Placeholder reason').value).toBe('Placeholder reason for the visit')
  })
})

describe('Documents: sign', () => {
  it('asks to confirm, saying how many fields are empty, then signs the fields as edited', async () => {
    const { signed } = await generated()
    fireEvent.change(field('Placeholder reason'), { target: { value: 'Placeholder typed reason' } })

    fireEvent.click(screen.getByRole('button', { name: 'Sign' }))
    const ask = screen.getByRole('alertdialog', { name: 'Sign this document?' })
    expect(ask.textContent).toContain('Once signed, it is final and can no longer be edited.')
    expect(ask.textContent).toContain('1 field is empty and stays empty.')
    fireEvent.click(within(ask).getByRole('button', { name: 'Sign' }))

    expect(await screen.findByText('Signed. This document is final.')).toBeTruthy()
    expect(signed).toEqual([
      [
        'c1000000-0000-4000-8000-000000000001',
        [
          { key: 'reason', value: 'Placeholder typed reason' },
          { key: 'plan', value: '' },
        ],
      ],
    ])
    expect(screen.queryByRole('textbox')).toBeNull()
    expect(screen.getByText('Placeholder typed reason')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Regenerate' })).toBeNull()
  })

  it('does not mention empty fields when every field is filled', async () => {
    await generated()
    fireEvent.change(field('Placeholder plan'), { target: { value: 'Placeholder typed plan' } })

    fireEvent.click(screen.getByRole('button', { name: 'Sign' }))

    expect(screen.getByRole('alertdialog').textContent).not.toContain('empty')
  })

  it('keeps the draft editable and says so when it cannot be signed', async () => {
    await generated({ failSign: true })

    fireEvent.click(screen.getByRole('button', { name: 'Sign' }))
    fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Sign' }))

    expect((await screen.findByRole('alert')).textContent).toBe('The document could not be signed. Try again.')
    expect(field('Placeholder reason').readOnly).toBe(false)
  })

  it('shows a draft signed elsewhere as final', async () => {
    const { emit } = await generated()
    emit(draft({ status: 'final' }))
    expect(await screen.findByText('Signed. This document is final.')).toBeTruthy()
    expect(screen.queryByRole('textbox')).toBeNull()
  })
})
