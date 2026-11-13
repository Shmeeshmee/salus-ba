import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Waitlist } from './Waitlist'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, paths, type Route } from './test/fakeFetch'
import { admin, clinicRoute, cy, org, staffBundle } from './test/fixtures'

afterEach(cleanup)

const first = 'e1000000-0000-4000-8000-000000000001'
const second = 'e1000000-0000-4000-8000-000000000002'

// Two entries in offer order: Ada waits for Cy with an open offer, a patient whose name is not shown for cardiology.
const entries = {
  entries: [
    {
      entry: {
        id: first, organization_id: org, practitioner_role_id: cy.roleId, earliest: '2026-11-10', latest: '2026-11-20',
        times: { start: '08:00', end: '12:00' }, status: 'active', created_at: '2026-11-04T09:00:00Z',
        offer: { id: '0f000000-0000-4000-8000-000000000001', slot_start: '2026-11-12T09:00:00Z', slot_end: '2026-11-12T09:30:00Z', expires_at: '2026-11-11T09:15:00Z' },
      },
      patient_id: 'aa000000-0000-4000-8000-0000000000aa', patient_name: 'Ada Example', priority: 10,
    },
    {
      entry: { id: second, organization_id: org, specialty: '394579002', earliest: '2026-11-10', latest: '2026-11-30', status: 'active', created_at: '2026-11-04T10:00:00Z' },
      patient_id: 'bb000000-0000-4000-8000-0000000000bb', priority: 0,
    },
  ],
}

function renderWaitlist(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch({
    ...clinicRoute,
    'GET /fhir/R4/PractitionerRole': () => fhirJSON(staffBundle),
    'GET /clinic/waitlist': () => Response.json(entries),
    ...routes,
  })
  renderInLanguage(<Waitlist api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} />, language)
  return fake
}

describe('Waitlist', () => {
  it('lists the entries in offer order with what each waits for and an offer held in the clinic zone', async () => {
    const fake = renderWaitlist({})
    const rows = within(await screen.findByRole('list', { name: 'Waitlist' })).getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('Ada Example')
    expect(rows[0].textContent).toContain('Cy Example')
    expect(rows[0].textContent).toContain('08:00 to 12:00')
    expect(rows[0].textContent).toContain('held until 10:15')
    expect(rows[1].textContent).toContain('Name not shown')
    expect(rows[1].textContent).toContain('Cardiology')
    const query = new URL(fake.requests.find((r) => r.url.includes('/clinic/waitlist'))!.url).searchParams
    expect([query.get('status'), query.get('limit')]).toEqual(['active', '100'])
    expect(screen.queryByRole('button', { name: 'Show more' })).toBeNull()
  })

  it('saves a priority and removes an entry after asking, then reads the list again', async () => {
    const fake = renderWaitlist({
      [`PUT /clinic/waitlist/${second}/priority`]: () => Response.json(entries.entries[1]),
      [`DELETE /waitlist/${first}`]: () => new Response(null, { status: 204 }),
    })
    const rows = within(await screen.findByRole('list', { name: 'Waitlist' })).getAllByRole('listitem')
    fireEvent.change(within(rows[1]).getByRole('spinbutton', { name: 'Priority' }), { target: { value: '50' } })
    fireEvent.click(within(rows[1]).getByRole('button', { name: 'Save priority' }))
    await waitFor(() => expect(paths(fake.requests).filter((p) => p === 'GET /clinic/waitlist')).toHaveLength(2))
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(await put.json()).toEqual({ priority: 50 })
    const again = within(await screen.findByRole('list', { name: 'Waitlist' })).getAllByRole('listitem')
    fireEvent.click(within(again[0]).getByRole('button', { name: 'Remove' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Remove this patient from the waitlist?' })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(paths(fake.requests).filter((p) => p === 'GET /clinic/waitlist')).toHaveLength(3))
    expect(paths(fake.requests)).toContain(`DELETE /waitlist/${first}`)
    expectCsrfOnWrites(fake.requests)
  })

  it('refuses a priority out of range without asking the API', async () => {
    const fake = renderWaitlist({})
    const rows = within(await screen.findByRole('list', { name: 'Waitlist' })).getAllByRole('listitem')
    fireEvent.change(within(rows[0]).getByRole('spinbutton', { name: 'Priority' }), { target: { value: '101' } })
    fireEvent.click(within(rows[0]).getByRole('button', { name: 'Save priority' }))
    expect(await within(rows[0]).findByText('Enter a priority from 0 to 100.')).toBeTruthy()
    expect(fake.requests.some((r) => r.method === 'PUT')).toBe(false)
  })

  it('reads a longer page when there is more, and another status', async () => {
    const fake = renderWaitlist({ 'GET /clinic/waitlist': () => Response.json({ ...entries, next_offset: 100 }) })
    fireEvent.click(await screen.findByRole('button', { name: 'Show more' }))
    await waitFor(() => expect(fake.requests.map((r) => new URL(r.url).searchParams.get('limit'))).toContain('200'))
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'booked' } })
    await waitFor(() => expect(fake.requests.some((r) => new URL(r.url).searchParams.get('status') === 'booked')).toBe(true))
  })

  it('says when the list cannot be loaded, in Bosnian too', async () => {
    renderWaitlist({ 'GET /clinic/waitlist': () => apiError(500, 'internal') }, 'bs')
    expect(await screen.findByText('Lista čekanja se nije mogla učitati. Ponovo učitajte stranicu.')).toBeTruthy()
  })
})
