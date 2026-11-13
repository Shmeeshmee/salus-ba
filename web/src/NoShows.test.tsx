import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppointmentDetail } from './AppointmentDetail'
import { newApiClient, type Account } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { admin, appointment, clinicRoute, cy, dee, org, patientId, practitioner } from './test/fixtures'

afterEach(cleanup)

// SAL-36 on the appointment's page: the correction of a no-show within 30 days (Q6) and the repeat no-show flag (Q7).
const id = 'e1000000-0000-4000-8000-000000000036'
const start = '2026-11-09T08:00:00Z'
const noShow = appointment(id, start, '2026-11-09T08:30:00Z', { status: 'noshow' }, 'Ada Example')
const day = 24 * 60 * 60 * 1000
const read = `GET /fhir/R4/Appointment/${id}`
const statusPost = `POST /appointments/${id}/status`
const flags = 'GET /clinic/noshow-flags'

function answer(status: string) {
  return Response.json({ id, organizationId: org, practitionerRoleId: cy.roleId, slotId: id, start, end: noShow.end, status, bookedVia: 'direct', versionId: 3, rescheduleCount: 0 })
}

function renderDetail(account: Account, routes: Record<string, Route>, now = () => new Date(Date.parse(start) + day), language: Language = 'en') {
  const fake = fakeFetch({
    ...clinicRoute,
    [read]: () => fhirJSON(noShow),
    [`GET /fhir/R4/Patient/${patientId}`]: () => outcome(404, 'not-found'),
    [`GET /appointments/${id}/consultation`]: () => apiError(404, 'not_found'),
    'GET /fhir/R4/QuestionnaireResponse': () => fhirJSON({ resourceType: 'Bundle', type: 'searchset' }),
    [flags]: () => Response.json({ flags: [] }),
    ...routes,
  })
  renderInLanguage(<AppointmentDetail api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} id={id} now={now} />, language)
  return fake
}

describe('NoShowCorrection', () => {
  it.each([
    ['the practitioner', practitioner],
    ['the admin', admin],
  ])('lets %s correct a no-show to fulfilled after a confirmation, with the CSRF header', async (_, account) => {
    const { requests } = renderDetail(account, { [statusPost]: () => answer('fulfilled') })

    fireEvent.click(await screen.findByRole('button', { name: 'Correct to fulfilled' }))
    expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
    expect(screen.getByRole('alertdialog', { name: 'Correct this no-show to fulfilled?' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Correct' }))

    expect(await screen.findByText('Fulfilled')).toBeTruthy()
    expect(await requests.find((r) => r.method === 'POST')!.json()).toEqual({ status: 'fulfilled' })
    expectCsrfOnWrites(requests)
    expect(screen.getByRole('button', { name: 'Correct to no-show' })).toBeTruthy()
  })

  it('keeps the status when the correction is not confirmed', async () => {
    const { requests } = renderDetail(practitioner, {})

    fireEvent.click(await screen.findByRole('button', { name: 'Correct to fulfilled' }))
    fireEvent.click(screen.getByRole('button', { name: 'Keep it' }))

    expect(screen.getByRole('button', { name: 'Correct to fulfilled' })).toBeTruthy()
    expect(paths(requests)).not.toContain(statusPost)
  })

  it('is not offered after 30 days, to another practitioner, or for a booked appointment', async () => {
    renderDetail(practitioner, {}, () => new Date(Date.parse(start) + 30 * day + 1000))
    expect(await screen.findByText('No-show')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Correct to fulfilled' })).toBeNull()
    cleanup()

    renderDetail({ ...practitioner, practitionerId: dee.practitionerId }, {})
    expect(await screen.findByText('No-show')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Correct to fulfilled' })).toBeNull()
    cleanup()

    renderDetail(practitioner, { [read]: () => fhirJSON({ ...noShow, status: 'booked' }) })
    expect(await screen.findByRole('button', { name: 'Mark fulfilled' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: /^Correct/ })).toBeNull()
  })

  it.each([
    ['correction_window_closed', 'This appointment can no longer be corrected: more than 30 days have passed since its start.'],
    ['appointment_closed', 'This appointment changed meanwhile. Reload the page to see its status.'],
  ])('says why on %s', async (code, message) => {
    renderDetail(practitioner, { [statusPost]: () => apiError(409, code) })

    fireEvent.click(await screen.findByRole('button', { name: 'Correct to fulfilled' }))
    fireEvent.click(screen.getByRole('button', { name: 'Correct' }))

    expect((await screen.findByRole('alert')).textContent).toBe(message)
    expect(screen.getByText('No-show')).toBeTruthy()
  })

  it('speaks Bosnian', async () => {
    renderDetail(practitioner, {}, undefined, 'bs')
    expect(await screen.findByRole('button', { name: 'Ispravi u obavljen' })).toBeTruthy()
  })
})

describe('NoShowFlag', () => {
  it('shows staff the patient’s recent no-shows with the detail in a tooltip, asked for this appointment only', async () => {
    const { requests } = renderDetail(admin, {
      [flags]: () => Response.json({ flags: [{ appointment_id: id, noshows: 3, window_days: 180 }] }),
    })

    expect(await screen.findByText('3 no-shows')).toBeTruthy()
    expect(screen.getByRole('tooltip').textContent).toBe(
      '3 missed appointments at this clinic in the last 180 days. Shown to staff only. It never stops the patient from booking.',
    )
    const url = new URL(requests.find((r) => new URL(r.url).pathname === '/clinic/noshow-flags')!.url)
    expect(url.searchParams.get('from')).toBe('2026-11-09T08:00:00.000Z')
    expect(url.searchParams.get('to')).toBe('2026-11-09T08:00:01.000Z')
  })

  it('shows nothing when the patient is not flagged or the flags cannot be read', async () => {
    renderDetail(admin, { [flags]: () => Response.json({ flags: [{ appointment_id: 'another', noshows: 3, window_days: 180 }] }) })
    expect(await screen.findByText('No-show')).toBeTruthy()
    await vi.waitFor(() => expect(screen.queryByRole('tooltip')).toBeNull())
    cleanup()

    renderDetail(admin, { [flags]: () => apiError(500, 'internal error') })
    expect(await screen.findByText('No-show')).toBeTruthy()
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('counts in the language’s plural forms', async () => {
    renderDetail(admin, { [flags]: () => Response.json({ flags: [{ appointment_id: id, noshows: 2, window_days: 90 }] }) }, undefined, 'sr-Cyrl')
    expect(await screen.findByText('2 недоласка')).toBeTruthy()
  })
})
