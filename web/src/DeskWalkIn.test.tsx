import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DeskWalkIn } from './DeskWalkIn'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Member } from './staff'
import { ana, cySchedule, exam, freeInPerson, ida, now, scheduleRoute, services, slotsRoute } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { cy, dee, org } from './test/fixtures'

afterEach(cleanup)

const staff: Member[] = [
  { roleId: cy.roleId, practitionerId: cy.practitionerId, name: 'Cy Example', active: true, version: '1' },
  { roleId: dee.roleId, practitionerId: dee.practitionerId, name: 'Dee Sample', active: true, version: '1' },
]
const booked = { id: 'e0000000-0000-4000-8000-0000000000e9', organizationId: org, practitionerRoleId: cy.roleId, slotId: freeInPerson, start: '2026-11-23T10:00:00Z', end: '2026-11-23T10:30:00Z', status: 'booked', versionId: 1, bookedVia: 'direct', rescheduleCount: 0, visitMode: 'in-person', serviceId: exam }
const emptyBundle = () => fhirJSON({ resourceType: 'Bundle', type: 'searchset', total: 0 })

function renderWalkIn(routes: Record<string, Route>) {
  const fake = fakeFetch({
    'GET /clinic/patients': () => Response.json([ana]),
    'GET /clinic/services': () => Response.json(services),
    'GET /fhir/R4/Schedule': scheduleRoute,
    'GET /fhir/R4/Slot': slotsRoute,
    ...routes,
  })
  const onBooked = vi.fn()
  const onClose = vi.fn()
  renderInLanguage(
    <DeskWalkIn api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} staff={staff} zone="Europe/Sarajevo" now={now} onBooked={onBooked} onClose={onClose} />,
    'en',
  )
  return { fake, onBooked, onClose }
}

async function pickAna() {
  const dialog = within(screen.getByRole('dialog', { name: 'Walk-in' }))
  fireEvent.change(dialog.getByLabelText('Name'), { target: { value: 'Ana' } })
  fireEvent.click(dialog.getByRole('button', { name: 'Search' }))
  fireEvent.click(await dialog.findByRole('button', { name: /Ana Example/ }))
  return dialog
}

describe('DeskWalkIn', () => {
  it("books a walk-in in one of today's free times at the clinic, with the service", async () => {
    let body: unknown
    const { fake, onBooked } = renderWalkIn({
      'POST /clinic/appointments': async (r) => {
        body = await r.json()
        return Response.json(booked, { status: 201 })
      },
    })
    const dialog = await pickAna()
    expect(dialog.getByText('Ana Example')).toBeTruthy()

    const time = (await dialog.findByLabelText('Time')) as HTMLSelectElement
    await waitFor(() => expect(Array.from(time.options).map((o) => o.textContent)).toEqual(['Choose a time', '11:00–11:30']))
    fireEvent.change(time, { target: { value: freeInPerson } })
    const service = dialog.getByLabelText('Service') as HTMLSelectElement
    expect(Array.from(service.options).map((o) => o.textContent)).toEqual(['No service', 'Placeholder examination'])
    fireEvent.change(service, { target: { value: exam } })
    fireEvent.click(dialog.getByRole('button', { name: 'Book visit' }))

    await waitFor(() => expect(onBooked).toHaveBeenCalled())
    expect(body).toEqual({ patientId: ana.id, slotId: freeInPerson, serviceId: exam })
    const schedule = new URL(fake.requests.find((r) => r.url.includes('/Schedule'))!.url).searchParams
    expect(schedule.get('actor')).toBe(`PractitionerRole/${cy.roleId}`)
    const slots = new URL(fake.requests.find((r) => r.url.includes('/Slot'))!.url).searchParams
    expect(slots.get('schedule')).toBe(`Schedule/${cySchedule}`)
    expect(slots.get('status')).toBe('free')
    // From now to the end of today in Sarajevo (UTC+1).
    expect(slots.getAll('start')).toEqual(['ge2026-11-23T08:20:00.000Z', 'lt2026-11-23T23:00:00.000Z'])
    expectCsrfOnWrites(fake.requests)
  })

  it('asks for the consent an imported patient gives in person before their booking (wave 4)', async () => {
    let body: unknown
    const { fake, onBooked } = renderWalkIn({
      'GET /clinic/patients': () => Response.json([ida]),
      [`POST /clinic/patients/${ida.id}/consent`]: () => new Response(null, { status: 204 }),
      'POST /clinic/appointments': async (r) => {
        body = await r.json()
        return Response.json(booked, { status: 201 })
      },
    })
    const dialog = within(screen.getByRole('dialog', { name: 'Walk-in' }))
    fireEvent.change(dialog.getByLabelText('Name'), { target: { value: 'Ida' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Search' }))
    fireEvent.click(await dialog.findByRole('button', { name: /Ida Import/ }))
    expect(dialog.getByText(/There is no active consent to care from this patient/)).toBeTruthy()
    expect(dialog.queryByLabelText('Time')).toBeNull()
    const record = dialog.getByRole('button', { name: 'Record consent' }) as HTMLButtonElement
    expect(record.disabled).toBe(true)
    fireEvent.click(dialog.getByLabelText('The patient gave the clinic their consent to care, in person.'))
    fireEvent.click(record)

    const time = (await dialog.findByLabelText('Time')) as HTMLSelectElement
    await waitFor(() => expect(time.options.length).toBe(2))
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Book visit' }))
    await waitFor(() => expect(onBooked).toHaveBeenCalled())
    expect(body).toEqual({ patientId: ida.id, slotId: freeInPerson })
    const posts = fake.requests.filter((r) => r.method === 'POST').map((r) => new URL(r.url).pathname)
    expect(posts).toEqual([`/clinic/patients/${ida.id}/consent`, '/clinic/appointments'])
    expectCsrfOnWrites(fake.requests)
  })

  it('says when the consent could not be recorded', async () => {
    renderWalkIn({ 'GET /clinic/patients': () => Response.json([ida]), [`POST /clinic/patients/${ida.id}/consent`]: () => apiError(500, 'internal') })
    const dialog = within(screen.getByRole('dialog', { name: 'Walk-in' }))
    fireEvent.change(dialog.getByLabelText('Name'), { target: { value: 'Ida' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Search' }))
    fireEvent.click(await dialog.findByRole('button', { name: /Ida Import/ }))
    fireEvent.click(dialog.getByLabelText('The patient gave the clinic their consent to care, in person.'))
    fireEvent.click(dialog.getByRole('button', { name: 'Record consent' }))
    expect(await dialog.findByText('The consent could not be recorded. Try again.')).toBeTruthy()
    expect(dialog.queryByLabelText('Time')).toBeNull()
  })

  it('books without a service when none is picked', async () => {
    let body: unknown
    renderWalkIn({
      'POST /clinic/appointments': async (r) => {
        body = await r.json()
        return Response.json(booked, { status: 201 })
      },
    })
    const dialog = await pickAna()
    const time = (await dialog.findByLabelText('Time')) as HTMLSelectElement
    await waitFor(() => expect(time.options).toHaveLength(2))
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Book visit' }))
    await waitFor(() => expect(body).toEqual({ patientId: ana.id, slotId: freeInPerson }))
  })

  it('reads the free times of the practitioner picked', async () => {
    const { fake } = renderWalkIn({})
    const dialog = await pickAna()
    await dialog.findByLabelText('Time')
    fireEvent.change(dialog.getByLabelText('Practitioner'), { target: { value: dee.roleId } })
    await waitFor(() => expect(fake.requests.some((r) => r.url.includes(encodeURIComponent(`PractitionerRole/${dee.roleId}`)))).toBe(true))
  })

  it('says when the practitioner has no free time left today', async () => {
    renderWalkIn({ 'GET /fhir/R4/Slot': emptyBundle })
    const dialog = await pickAna()
    expect(await dialog.findByText('No free time left today for this practitioner.')).toBeTruthy()
    cleanup()
    renderWalkIn({ 'GET /fhir/R4/Schedule': emptyBundle })
    const again = await pickAna()
    expect(await again.findByText('No free time left today for this practitioner.')).toBeTruthy()
  })

  it('reads the times again when the one picked was just taken', async () => {
    const { fake, onBooked } = renderWalkIn({ 'POST /clinic/appointments': () => apiError(409, 'slot_unavailable') })
    const dialog = await pickAna()
    const time = (await dialog.findByLabelText('Time')) as HTMLSelectElement
    await waitFor(() => expect(time.options).toHaveLength(2))
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Book visit' }))

    expect(await dialog.findByText('This time was just taken. Pick another.')).toBeTruthy()
    await waitFor(() => expect(fake.requests.filter((r) => r.url.includes('/Slot'))).toHaveLength(2))
    expect(onBooked).not.toHaveBeenCalled()
  })

  it.each([
    ['consent_required', 403, 'The patient has not given the clinic their consent to care.'],
    ['invalid_service', 400, 'This service is not offered for this visit.'],
    ['forbidden', 403, 'The visit could not be booked. Try again.'],
  ])('says why a booking was refused (%s)', async (code, status, text) => {
    renderWalkIn({ 'POST /clinic/appointments': () => apiError(status, code) })
    const dialog = await pickAna()
    const time = (await dialog.findByLabelText('Time')) as HTMLSelectElement
    await waitFor(() => expect(time.options).toHaveLength(2))
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Book visit' }))
    expect(await dialog.findByText(text)).toBeTruthy()
  })

  it('lets the patient be changed and the dialog be closed', async () => {
    const { onClose } = renderWalkIn({})
    const dialog = await pickAna()
    fireEvent.click(dialog.getByRole('button', { name: 'Change patient' }))
    expect(dialog.getByLabelText('Name')).toBeTruthy()
    fireEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalled()
  })
})
