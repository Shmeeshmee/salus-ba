import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { CarePlanBook } from './CarePlanBook'
import type { CarePlan } from './CarePlans'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import type { components } from './api/schema'
import { renderInLanguage } from './i18n/renderInLanguage'
import { cySchedule, scheduleRoute, slot } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { cy, org, patientId, staffRoute } from './test/fixtures'

afterEach(cleanup)

const exam = 'c1000000-0000-4000-8000-000000000001'
const physio = 'c1000000-0000-4000-8000-000000000002'
const services: components['schemas']['Service'][] = [
  { id: exam, name: 'Placeholder examination', priceMinor: 5000, currency: 'BAM', modes: ['in-person', 'video'], active: true, translations: [], versionId: 1 },
  { id: physio, name: 'Placeholder therapy session', priceMinor: 3000, currency: 'BAM', modes: ['in-person'], active: true, translations: [], versionId: 1 },
]
const draft: CarePlan = {
  id: 'd2000000-0000-4000-8000-000000000001', organizationId: org, patientId, status: 'draft', title: 'Placeholder knee plan',
  createdAt: '2026-11-20T09:00:00Z', versionId: 2,
  steps: [
    { id: 'd3000000-0000-4000-8000-000000000001', position: 1, serviceId: exam, dayOffset: 0, visitMode: 'in-person' },
    { id: 'd3000000-0000-4000-8000-000000000002', position: 2, serviceId: physio, dayOffset: 7, visitMode: 'in-person' },
  ],
}
// Free slots on the first day (30 November) and the eighth (7 December), in Sarajevo (UTC+1).
const day0InPerson = 'f0000000-0000-4000-8000-000000000001'
const day0Video = 'f0000000-0000-4000-8000-000000000002'
const day7InPerson = 'f0000000-0000-4000-8000-000000000003'
const day7Later = 'f0000000-0000-4000-8000-000000000004'
const slotsRoute = () =>
  fhirJSON({
    resourceType: 'Bundle', type: 'searchset', total: 4,
    entry: [
      slot(day0InPerson, '2026-11-30T08:00:00Z', '2026-11-30T08:30:00Z', 'in-person'),
      slot(day0Video, '2026-11-30T09:00:00Z', '2026-11-30T09:30:00Z', 'video'),
      slot(day7InPerson, '2026-12-07T08:00:00Z', '2026-12-07T08:30:00Z', 'in-person'),
      slot(day7Later, '2026-12-07T13:00:00Z', '2026-12-07T13:30:00Z', 'in-person'),
    ],
  })
const book = `POST /clinic/care-plans/${draft.id}/book`

function renderBook(routes: Record<string, Route>) {
  const fake = fakeFetch({ ...staffRoute, 'GET /fhir/R4/Schedule': scheduleRoute, 'GET /fhir/R4/Slot': slotsRoute, ...routes })
  const onBooked = vi.fn()
  const onClose = vi.fn()
  renderInLanguage(
    <CarePlanBook
      api={newApiClient(fake.fetch)}
      fhir={newFhirClient(fake.fetch)}
      plan={draft}
      services={services}
      zone="Europe/Sarajevo"
      practitionerId={cy.practitionerId}
      today="2026-11-25"
      onBooked={onBooked}
      onClose={onClose}
    />,
    'en',
  )
  return { fake, onBooked, onClose, dialog: within(screen.getByRole('dialog', { name: 'Book treatment plan' })) }
}

async function chooseTimes(dialog: ReturnType<typeof within>) {
  fireEvent.change(dialog.getByLabelText('First day'), { target: { value: '2026-11-30' } })
  const first = (await dialog.findByLabelText('1. Placeholder examination, at the clinic, Monday, 30 November 2026')) as HTMLSelectElement
  await waitFor(() => expect(Array.from(first.options).map((o) => o.textContent)).toEqual(['Choose a time', '09:00–09:30']))
  const second = dialog.getByLabelText('2. Placeholder therapy session, at the clinic, Monday, 7 December 2026') as HTMLSelectElement
  expect(Array.from(second.options).map((o) => o.value)).toEqual(['', day7InPerson, day7Later])
  fireEvent.change(first, { target: { value: day0InPerson } })
  fireEvent.change(second, { target: { value: day7Later } })
}

describe('CarePlanBook (clinic-cms TP2)', () => {
  it("books every step at one free time of its day and visit type, the practitioner's own by default, at the version read", async () => {
    let body: unknown
    let match: string | null = null
    const active = { ...draft, status: 'active' as const, startsOn: '2026-11-30', versionId: 3 }
    const { fake, onBooked, dialog } = renderBook({
      [book]: async (r) => {
        body = await r.json()
        match = r.headers.get('If-Match')
        return Response.json(active)
      },
    })
    expect(dialog.getByText('Every step is booked, or none is.')).toBeTruthy()
    await waitFor(() => expect((dialog.getByLabelText('Practitioner') as HTMLSelectElement).value).toBe(cy.roleId))
    expect((dialog.getByRole('button', { name: 'Book every step' }) as HTMLButtonElement).disabled).toBe(true)
    await chooseTimes(dialog)
    fireEvent.click(dialog.getByRole('button', { name: 'Book every step' }))

    await waitFor(() => expect(onBooked).toHaveBeenCalledWith(active))
    expect(body).toEqual({ startsOn: '2026-11-30', slotIds: [day0InPerson, day7Later] })
    expect(match).toBe('W/"2"')
    // The last search, after the first day was chosen (the first was for today, the default).
    const slots = new URL(fake.requests.filter((r) => r.url.includes('/Slot')).at(-1)!.url).searchParams
    expect(slots.get('schedule')).toBe(`Schedule/${cySchedule}`)
    expect(slots.get('status')).toBe('free')
    // From the first day's midnight to the end of the last step's day, in the clinic's zone.
    expect(slots.getAll('start')).toEqual(['ge2026-11-29T23:00:00.000Z', 'lt2026-12-07T23:00:00.000Z'])
    expectCsrfOnWrites(fake.requests)
  })

  it('says why a booking is refused, and that nothing was booked', async () => {
    for (const [code, message] of [
      ['slot_unavailable', 'A time was taken in the meantime. Nothing was booked: choose the times again.'],
      ['resource_busy', 'A device one of the steps needs is busy at its time. Nothing was booked: choose another time.'],
      ['version_conflict', 'Someone else changed this plan. Reload the page to see it.'],
      ['care_plan_not_draft', 'This plan was booked or has ended. Reload the page to see it.'],
    ] as const) {
      const { onBooked, dialog } = renderBook({ [book]: () => apiError(409, code) })
      await chooseTimes(dialog)
      fireEvent.click(dialog.getByRole('button', { name: 'Book every step' }))
      expect(await dialog.findByText(message)).toBeTruthy()
      expect(onBooked).not.toHaveBeenCalled()
      cleanup()
    }
  })

  it("names the first day in the app's own format beside the browser's date field (RD6)", async () => {
    const { dialog } = renderBook({})
    const field = dialog.getByLabelText('First day')
    expect(dialog.getByText('Wednesday, 25 November 2026').id).toBe(field.getAttribute('aria-describedby'))
    fireEvent.change(field, { target: { value: '2026-11-30' } })
    expect(dialog.getByText('Monday, 30 November 2026').id).toBe(field.getAttribute('aria-describedby'))
    fireEvent.change(field, { target: { value: '' } })
    expect(field.getAttribute('aria-describedby')).toBeNull()
  })

  it('says so when a day has no free time of its type', async () => {
    const { dialog } = renderBook({ 'GET /fhir/R4/Slot': () => fhirJSON({ resourceType: 'Bundle', type: 'searchset', total: 0, entry: [] }) })
    fireEvent.change(dialog.getByLabelText('First day'), { target: { value: '2026-11-30' } })
    expect(await dialog.findAllByText('No free time that day.')).toHaveLength(2)
  })
})
