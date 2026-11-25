import { cleanup, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { DataRequests } from './DataRequests'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, fakeFetch, type Route } from './test/fakeFetch'
import { requests } from './test/dataRequests'
import { admin, clinicRoute } from './test/fixtures'

afterEach(cleanup)

function renderList(routes: Record<string, Route> = {}, language: Language = 'en') {
  const fake = fakeFetch({ ...clinicRoute, 'GET /clinic/data-requests': () => Response.json(requests), ...routes })
  renderInLanguage(<DataRequests api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} />, language)
  return fake
}

describe('DataRequests', () => {
  it("lists the clinic's requests with their due date, state and what each needs, each opening its detail", async () => {
    renderList()

    const rows = within(await screen.findByRole('list', { name: 'Data requests' })).getAllByRole('listitem')
    expect(rows).toHaveLength(3)
    expect(rows[0].textContent).toContain('Ada Example')
    expect(rows[0].textContent).toContain('Born 1 January 1990')
    expect(rows[1].textContent).toContain('Rina Sample')
    expect(rows[2].textContent).not.toContain('Born')
    expect(rows[0].textContent).toContain('Erasure')
    expect(rows[0].textContent).toContain('Due Wednesday, 2 December 2026')
    expect(rows[0].textContent).toContain('Received')
    expect(rows[0].textContent).toContain('Erase what has no legal basis to stay')
    expect(rows[1].textContent).toContain('Due Friday, 1 January 2027')
    expect(rows[1].textContent).toContain('Filed at the desk')
    expect(rows[2].textContent).toContain('Fulfilled')
    expect(rows[2].textContent).not.toContain('Leave the patient out')
    const open = within(rows[0]).getByRole('link', { name: 'Open' })
    expect(open.getAttribute('href')).toBe(`/data-requests/${requests[0].id}`)
  })

  it('says when there are none, or the list cannot be loaded', async () => {
    renderList({ 'GET /clinic/data-requests': () => Response.json([]) })
    expect(await screen.findByText('No data requests.')).toBeTruthy()
    cleanup()

    renderList({ 'GET /clinic/data-requests': () => apiError(500, 'internal error') })
    expect((await screen.findByRole('alert')).textContent).toBe('The requests could not be loaded. Reload the page to try again.')
  })

  it('reads in Bosnian', async () => {
    renderList({}, 'bs')
    const rows = within(await screen.findByRole('list', { name: 'Zahtjevi za podatke' })).getAllByRole('listitem')
    expect(rows[0].textContent).toContain('Brisanje')
  })
})
