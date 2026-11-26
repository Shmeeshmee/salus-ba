import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { CarePlans, type CarePlan } from './CarePlans'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'
import { org, patientId } from './test/fixtures'
import appCSS from './app.css?raw'

afterEach(cleanup)

const visit = 'e1000000-0000-4000-8000-000000000001'
const exam = 'c1000000-0000-4000-8000-000000000001'
const physio = 'c1000000-0000-4000-8000-000000000002'
const quoteId = 'b1000000-0000-4000-8000-000000000001'
const services = [
  { id: exam, name: 'Placeholder examination', priceMinor: 5000, currency: 'BAM', modes: ['in-person', 'video'], active: true, translations: [], versionId: 1 },
  { id: physio, name: 'Placeholder therapy session', priceMinor: 3000, currency: 'BAM', modes: ['in-person'], active: true, translations: [], versionId: 1 },
]
const quote = {
  id: quoteId, organizationId: org, patientId, appointmentId: visit, issuedAt: '2026-11-09T08:40:00Z', validUntil: '2026-12-09T08:40:00Z',
  state: 'sent', note: '', currency: 'BAM', totalMinor: 9500, lines: [],
}

export function plan(id: string, status: CarePlan['status'], extra: Partial<CarePlan> = {}): CarePlan {
  return {
    id, organizationId: org, patientId, status, title: 'Placeholder knee plan', createdAt: '2026-11-20T09:00:00Z', versionId: 2,
    steps: [
      { id: `${id.slice(0, -1)}a`, position: 1, serviceId: exam, dayOffset: 0, visitMode: 'in-person' },
      { id: `${id.slice(0, -1)}b`, position: 2, serviceId: physio, dayOffset: 7, visitMode: 'in-person' },
    ],
    ...extra,
  }
}

const draft = plan('d2000000-0000-4000-8000-000000000001', 'draft')
const booked = plan('d2000000-0000-4000-8000-000000000002', 'active', { startsOn: '2026-11-30', bookedAt: '2026-11-21T10:00:00Z' })
const list = `GET /clinic/patients/${patientId}/care-plans`

function renderPlans(routes: Record<string, Route>) {
  const fake = fakeFetch({
    'GET /clinic/services': () => Response.json(services),
    'GET /clinic/quotes': () => Response.json([quote]),
    ...routes,
  })
  renderInLanguage(
    <CarePlans api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} patientId={patientId} appointmentId={visit} zone="Europe/Sarajevo" />,
    'en',
  )
  return fake
}

describe('CarePlans (clinic-cms TP2)', () => {
  it("lists the patient's plans with their state and steps, a booked one with its days", async () => {
    renderPlans({ [list]: () => Response.json([booked, draft]) })
    const plans = within(await screen.findByRole('list', { name: 'Treatment plans' }))
    const items = plans.getAllByRole('listitem').filter((li) => li.parentElement?.getAttribute('aria-label') === 'Treatment plans')
    expect(items).toHaveLength(2)
    expect(within(items[0]).getByText('Booked')).toBeTruthy()
    expect(within(items[0]).getByText('Starts on Monday, 30 November 2026')).toBeTruthy()
    expect(within(items[0]).getByText('1. Placeholder examination, at the clinic, Monday, 30 November 2026')).toBeTruthy()
    expect(within(items[0]).getByText('2. Placeholder therapy session, at the clinic, Monday, 7 December 2026')).toBeTruthy()
    expect(within(items[1]).getByText('Draft')).toBeTruthy()
    expect(within(items[1]).getByText('2. Placeholder therapy session, at the clinic, day 7')).toBeTruthy()
    // A draft is changed or booked; a booked plan is only cancelled.
    expect(within(items[1]).getByRole('button', { name: 'Change' })).toBeTruthy()
    expect(within(items[1]).getByRole('button', { name: 'Book' })).toBeTruthy()
    expect(within(items[0]).queryByRole('button', { name: 'Change' })).toBeNull()
    expect(within(items[0]).getByRole('button', { name: 'Cancel plan' })).toBeTruthy()
  })

  it("shows a booked step on its visit's day and time after a move, and its service by the plan's name once off the price list", async () => {
    const moved = plan('d2000000-0000-4000-8000-000000000003', 'active', {
      startsOn: '2026-11-30', bookedAt: '2026-11-21T10:00:00Z',
      steps: [
        { id: 'd3000000-0000-4000-8000-000000000001', position: 1, serviceId: physio, dayOffset: 7, visitMode: 'in-person', visitStart: '2026-12-07T09:00:00Z' },
        // The examination's visit moved after the session: its new day, in the clinic's zone, not its offset's.
        { id: 'd3000000-0000-4000-8000-000000000002', position: 2, serviceId: 'c1000000-0000-4000-8000-0000000000ff', serviceName: 'Placeholder retired service', dayOffset: 0, visitMode: 'in-person', visitStart: '2026-12-09T23:30:00Z' },
      ],
    })
    renderPlans({ [list]: () => Response.json([moved]) })
    const plans = within(await screen.findByRole('list', { name: 'Treatment plans' }))
    expect(plans.getByText('1. Placeholder therapy session, at the clinic, Monday, 7 December 2026, 10:00')).toBeTruthy()
    expect(plans.getByText('2. Placeholder retired service, at the clinic, Thursday, 10 December 2026, 00:30')).toBeTruthy()
  })

  it('says when the patient has no plan, and when the plans cannot be read', async () => {
    renderPlans({ [list]: () => Response.json([]) })
    expect(await screen.findByText('No treatment plans for this patient yet.')).toBeTruthy()
    cleanup()
    renderPlans({ [list]: () => apiError(404, 'not_found') })
    expect(await screen.findByText('The treatment plans could not be loaded. Reload the page to try again.')).toBeTruthy()
  })

  it('lays out a draft: ordered steps of the price list, each a day and a visit type, with the visit quote', async () => {
    let body: unknown
    const written = plan('d2000000-0000-4000-8000-000000000003', 'draft')
    let plans: CarePlan[] = []
    const fake = renderPlans({
      [list]: () => Response.json(plans),
      'POST /clinic/care-plans': async (r) => {
        body = await r.json()
        plans = [written]
        return Response.json(written, { status: 201 })
      },
    })
    fireEvent.click(await screen.findByRole('button', { name: 'New treatment plan' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New treatment plan' }))
    expect(dialog.getByText('The patient does not see a draft. Booking it books a visit for every step, or none.')).toBeTruthy()
    fireEvent.change(dialog.getByLabelText('Title'), { target: { value: ' Placeholder knee plan ' } })
    fireEvent.change(await dialog.findByLabelText('Service'), { target: { value: physio } })
    fireEvent.click(dialog.getByRole('button', { name: 'Add step' }))
    fireEvent.change(dialog.getAllByLabelText('Service')[1], { target: { value: exam } })
    fireEvent.change(dialog.getAllByLabelText('Days from the start')[1], { target: { value: '7' } })
    fireEvent.change(dialog.getAllByLabelText('Visit type')[1], { target: { value: 'video' } })
    // The second step goes first.
    fireEvent.click(dialog.getByRole('button', { name: 'Move step 2 up' }))
    fireEvent.change(await dialog.findByLabelText('Quote'), { target: { value: quoteId } })
    fireEvent.click(dialog.getByRole('button', { name: 'Save draft' }))

    expect(await screen.findByText('The draft was saved.')).toBeTruthy()
    expect(body).toEqual({
      patientId,
      title: 'Placeholder knee plan',
      quoteId,
      steps: [
        { serviceId: exam, dayOffset: 7, visitMode: 'video' },
        { serviceId: physio, dayOffset: 0, visitMode: 'in-person' },
      ],
    })
    expect(screen.queryByRole('dialog')).toBeNull()
    await screen.findByText('Draft')
    expectCsrfOnWrites(fake.requests)
  })

  it('draws "Add step" as a button, outlined as the form\'s other buttons (RD6)', async () => {
    const style = document.createElement('style')
    style.textContent = appCSS
    document.head.append(style)
    try {
      renderPlans({ [list]: () => Response.json([]) })
      fireEvent.click(await screen.findByRole('button', { name: 'New treatment plan' }))
      const dialog = within(await screen.findByRole('dialog', { name: 'New treatment plan' }))
      // jsdom drops a declaration that uses var() (the border's colour), so the outlined button is told by its padding.
      const look = (b: HTMLElement) => getComputedStyle(b).padding
      expect(look(dialog.getByRole('button', { name: 'Add step' }))).toBe('10px 16px')
      expect(look(dialog.getByRole('button', { name: 'Add step' }))).toBe(look(dialog.getByRole('button', { name: 'Cancel' })))
    } finally {
      style.remove()
    }
  })

  it('offers a step only the visit types its service has, and saves nothing while a step has no service', async () => {
    renderPlans({ [list]: () => Response.json([]) })
    fireEvent.click(await screen.findByRole('button', { name: 'New treatment plan' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New treatment plan' }))
    fireEvent.change(dialog.getByLabelText('Title'), { target: { value: 'Placeholder plan' } })
    expect((dialog.getByRole('button', { name: 'Save draft' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.change(await dialog.findByLabelText('Service'), { target: { value: physio } })
    const mode = dialog.getByLabelText('Visit type') as HTMLSelectElement
    expect(Array.from(mode.options).map((o) => o.value)).toEqual(['in-person'])
    expect((dialog.getByRole('button', { name: 'Save draft' }) as HTMLButtonElement).disabled).toBe(false)
  })

  it('changes a draft at the version read, and says why a change is refused', async () => {
    let match: string | null = null
    let body: unknown
    renderPlans({
      [list]: () => Response.json([draft]),
      [`PUT /clinic/care-plans/${draft.id}`]: async (r) => {
        match = r.headers.get('If-Match')
        body = await r.json()
        return apiError(409, 'care_plan_not_draft')
      },
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Change' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Change treatment plan' }))
    expect((dialog.getByLabelText('Title') as HTMLInputElement).value).toBe('Placeholder knee plan')
    await waitFor(() => expect((dialog.getAllByLabelText('Service')[1] as HTMLSelectElement).value).toBe(physio))
    fireEvent.click(dialog.getByRole('button', { name: 'Remove step 1' }))
    fireEvent.click(dialog.getByRole('button', { name: 'Save draft' }))
    expect(await dialog.findByText('This plan was booked or has ended. Reload the page to see it.')).toBeTruthy()
    expect(match).toBe('W/"2"')
    expect(body).toEqual({ title: 'Placeholder knee plan', steps: [{ serviceId: physio, dayOffset: 7, visitMode: 'in-person' }] })
  })

  it('cancels a booked plan at the version read, after asking, and shows it cancelled', async () => {
    let match: string | null = null
    const cancelled = { ...booked, status: 'revoked' as const, revokedAt: '2026-11-25T10:00:00Z', versionId: 3 }
    renderPlans({
      [list]: () => Response.json([booked]),
      [`POST /clinic/care-plans/${booked.id}/cancel`]: (r) => {
        match = r.headers.get('If-Match')
        return Response.json(cancelled)
      },
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Cancel plan' }))
    const ask = within(screen.getByRole('alertdialog', { name: 'Cancel this treatment plan?' }))
    expect(ask.getByText("Its visits that have not taken place are cancelled. Visits that took place stay in the patient's record.")).toBeTruthy()
    fireEvent.click(ask.getByRole('button', { name: 'Cancel plan' }))
    expect(await screen.findByText('The treatment plan was cancelled.')).toBeTruthy()
    expect(screen.getByText('Cancelled')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Cancel plan' })).toBeNull()
    expect(match).toBe('W/"2"')
  })

  it('keeps the plan when the question is answered no, and says when it has ended already', async () => {
    renderPlans({
      [list]: () => Response.json([booked]),
      [`POST /clinic/care-plans/${booked.id}/cancel`]: () => apiError(409, 'care_plan_ended'),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Cancel plan' }))
    fireEvent.click(screen.getByRole('button', { name: 'Keep the plan' }))
    expect(screen.queryByRole('alertdialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel plan' }))
    fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Cancel plan' }))
    expect(await screen.findByText('This plan has ended already. Reload the page to see it.')).toBeTruthy()
  })
})
