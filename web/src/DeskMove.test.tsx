import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DeskMove } from './DeskMove'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Member } from './staff'
import { anaVisit, boVisit, cySchedule, freeInPerson, freeVideo, now, scheduleRoute, slot, slotsRoute } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { cy, dee } from './test/fixtures'

afterEach(cleanup)

const staff: Member[] = [
  { roleId: cy.roleId, practitionerId: cy.practitionerId, name: 'Cy Example', active: true, version: '1' },
  { roleId: dee.roleId, practitionerId: dee.practitionerId, name: 'Dee Sample', active: true, version: '1' },
]

type Visit = typeof anaVisit
const move = `POST /clinic/appointments/${anaVisit.appointment.id}/reschedule`

function renderMove(routes: Record<string, Route>, visit: Visit = { ...anaVisit, appointment: { ...anaVisit.appointment, versionId: 4 } }) {
  const fake = fakeFetch({ 'GET /fhir/R4/Schedule': scheduleRoute, 'GET /fhir/R4/Slot': slotsRoute, ...routes })
  const onMoved = vi.fn()
  const onClose = vi.fn()
  renderInLanguage(
    <DeskMove api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} staff={staff} zone="Europe/Sarajevo" now={now} visit={visit} onMoved={onMoved} onClose={onClose} />,
    'en',
  )
  return { fake, onMoved, onClose, dialog: within(screen.getByRole('dialog', { name: 'Move visit' })) }
}

describe('DeskMove', () => {
  it("moves the visit to one of today's or later free times of its visit type, with the version read", async () => {
    let body: unknown
    let match: string | null = null
    const moved = { ...anaVisit.appointment, slotId: freeInPerson, start: '2026-11-23T10:00:00Z', end: '2026-11-23T10:30:00Z', versionId: 5 }
    const { fake, onMoved, dialog } = renderMove({
      [move]: async (r) => {
        body = await r.json()
        match = r.headers.get('If-Match')
        return Response.json(moved)
      },
    })
    expect(dialog.getByText('Ana Example')).toBeTruthy()
    expect((dialog.getByLabelText('Practitioner') as HTMLSelectElement).value).toBe(cy.roleId)
    const time = (await dialog.findByLabelText('New time')) as HTMLSelectElement
    // Only the in-person slot: the visit is at the clinic.
    await waitFor(() => expect(Array.from(time.options).map((o) => o.textContent)).toEqual(['Choose a time', '11:00–11:30']))
    const day = dialog.getByLabelText('Day') as HTMLSelectElement
    expect(Array.from(day.options).map((o) => o.textContent)).toEqual(['Monday, 23 November 2026'])
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Move visit' }))

    await waitFor(() => expect(onMoved).toHaveBeenCalledWith(moved))
    expect(body).toEqual({ slotId: freeInPerson })
    expect(match).toBe('W/"4"')
    const slots = new URL(fake.requests.find((r) => r.url.includes('/Slot'))!.url).searchParams
    expect(slots.get('schedule')).toBe(`Schedule/${cySchedule}`)
    expect(slots.get('status')).toBe('free')
    // From now to the end of the 14th day after today in Sarajevo (UTC+1).
    expect(slots.getAll('start')).toEqual(['ge2026-11-23T08:20:00.000Z', 'lt2026-12-07T23:00:00.000Z'])
    expectCsrfOnWrites(fake.requests)
  })

  it("offers another practitioner's times, and a video visit only video times", async () => {
    const { fake, dialog } = renderMove({}, boVisit)
    const time = (await dialog.findByLabelText('New time')) as HTMLSelectElement
    await waitFor(() => expect(Array.from(time.options).map((o) => o.value)).toEqual(['', freeVideo]))
    fireEvent.change(dialog.getByLabelText('Practitioner'), { target: { value: dee.roleId } })
    await waitFor(() => expect(fake.requests.some((r) => r.url.includes(encodeURIComponent(`PractitionerRole/${dee.roleId}`)))).toBe(true))
  })

  it('says why a move is refused, and reads the times again when the time was taken', async () => {
    for (const [code, message] of [
      ['slot_unavailable', 'This time was just taken. Pick another.'],
      ['appointment_closed', 'This visit is no longer booked. Reload the page to see it as it is now.'],
      ['version_conflict', 'The visit was changed in the meantime. Close this and reload the page.'],
      ['not_reschedulable', 'A video visit cannot be moved once its call can be joined.'],
      ['consent_required', 'The patient has not given the clinic their consent to care.'],
      ['internal', 'The visit could not be moved. Try again.'],
    ] as const) {
      const { fake, onMoved, dialog } = renderMove({ [move]: () => apiError(409, code) })
      const time = (await dialog.findByLabelText('New time')) as HTMLSelectElement
      await waitFor(() => expect(time.options).toHaveLength(2))
      fireEvent.change(time, { target: { value: freeInPerson } })
      fireEvent.click(dialog.getByRole('button', { name: 'Move visit' }))
      expect(await dialog.findByText(message)).toBeTruthy()
      expect(onMoved).not.toHaveBeenCalled()
      const reads = fake.requests.filter((r) => r.url.includes('/Slot')).length
      expect(reads).toBe(code === 'slot_unavailable' ? 2 : 1)
      cleanup()
    }
  })

  it('offers the free times of one day at a time, the first day with any first (wave 4 walk, finding 7)', async () => {
    const later = 'f0000000-0000-4000-8000-0000000000f9'
    const { dialog } = renderMove({
      'GET /fhir/R4/Slot': () =>
        fhirJSON({
          resourceType: 'Bundle', type: 'searchset', total: 2,
          entry: [slot(freeInPerson, '2026-11-23T10:00:00Z', '2026-11-23T10:30:00Z', 'in-person'), slot(later, '2026-11-25T13:00:00Z', '2026-11-25T13:30:00Z', 'in-person')],
        }),
    })
    const day = (await dialog.findByLabelText('Day')) as HTMLSelectElement
    expect(Array.from(day.options).map((o) => o.textContent)).toEqual(['Monday, 23 November 2026', 'Wednesday, 25 November 2026'])
    const time = dialog.getByLabelText('New time') as HTMLSelectElement
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.change(day, { target: { value: day.options[1].value } })
    expect(Array.from(time.options).map((o) => o.textContent)).toEqual(['Choose a time', '14:00–14:30'])
    expect(time.value).toBe('')
    expect((dialog.getByRole('button', { name: 'Move visit' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('closes without moving', async () => {
    const { fake, onClose, dialog } = renderMove({})
    fireEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalled()
    expect(fake.requests.some((r) => r.method === 'POST')).toBe(false)
  })
})
