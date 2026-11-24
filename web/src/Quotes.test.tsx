import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Quotes } from './Quotes'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, fakeFetch, type Route } from './test/fakeFetch'
import { admin, clinicRoute } from './test/fixtures'

afterEach(cleanup)

const exam = 'c1000000-0000-4000-8000-000000000001'
const fresh = 'e2000000-0000-4000-8000-000000000002'
const old = 'e2000000-0000-4000-8000-000000000001'
const line = { serviceId: exam, name: 'Placeholder examination', quantity: 1, unitPriceMinor: 5000, discountMinor: 0, totalMinor: 5000 }
const patientId = 'aa000000-0000-4000-8000-0000000000aa'

// Synthetic quotes, newest first: one still valid, one past its 30 days.
const quotes = [
  { id: fresh, organizationId: admin.organizationId, patientId, issuedAt: '2026-11-20T09:00:00Z', validUntil: '2026-12-20T09:00:00Z', state: 'sent', currency: 'BAM', totalMinor: 5000, lines: [line] },
  { id: old, organizationId: admin.organizationId, patientId, issuedAt: '2026-09-01T09:00:00Z', validUntil: '2026-10-01T09:00:00Z', state: 'expired', currency: 'BAM', totalMinor: 10000, lines: [line, line] },
]

function renderQuotes(routes: Record<string, Route> = {}, language: Language = 'en') {
  const fake = fakeFetch({ ...clinicRoute, 'GET /clinic/quotes': () => Response.json(quotes), ...routes })
  renderInLanguage(<Quotes api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} />, language)
  return fake
}

describe('Quotes', () => {
  it("lists the clinic's quotes, newest first, with their state, total, validity and PDF", async () => {
    renderQuotes()

    const rows = within(await screen.findByRole('list', { name: 'Quotes' })).getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('Sent')
    expect(rows[0].textContent).toContain('valid until Sunday, 20 December 2026')
    expect(rows[1].textContent).toContain('Expired')
    const pdf = within(rows[0]).getByRole('link', { name: 'Open PDF' })
    expect(pdf.getAttribute('href')).toBe(`${location.origin}/clinic/quotes/${fresh}/pdf`)
    expect(screen.getByRole('link', { name: 'Receipts' }).getAttribute('href')).toBe('/receipts')
  })

  it('filters the quotes by state', async () => {
    renderQuotes()
    await screen.findByRole('list', { name: 'Quotes' })

    fireEvent.change(screen.getByLabelText('State'), { target: { value: 'expired' } })
    const rows = within(screen.getByRole('list', { name: 'Quotes' })).getAllByRole('listitem')
    expect(rows).toHaveLength(1)
    expect(within(rows[0]).getByRole('link', { name: 'Open PDF' }).getAttribute('href')).toContain(old)

    fireEvent.change(screen.getByLabelText('State'), { target: { value: 'all' } })
    expect(within(screen.getByRole('list', { name: 'Quotes' })).getAllByRole('listitem')).toHaveLength(2)
  })

  it('says when there are no quotes, none in the state, or the list cannot be loaded', async () => {
    renderQuotes({ 'GET /clinic/quotes': () => Response.json([]) })
    expect(await screen.findByText('No quotes yet.')).toBeTruthy()
    cleanup()

    renderQuotes({ 'GET /clinic/quotes': () => Response.json([quotes[0]]) })
    await screen.findByRole('list', { name: 'Quotes' })
    fireEvent.change(screen.getByLabelText('State'), { target: { value: 'expired' } })
    expect(screen.getByText('No quotes in this state.')).toBeTruthy()
    cleanup()

    renderQuotes({ 'GET /clinic/quotes': () => apiError(500, 'internal error') })
    expect((await screen.findByRole('alert')).textContent).toBe('The quotes could not be loaded. Reload the page to try again.')
  })

  it('reads in Bosnian', async () => {
    renderQuotes({}, 'bs')
    const rows = within(await screen.findByRole('list', { name: 'Ponude' })).getAllByRole('listitem')
    expect(rows[1].textContent).toContain('Istekla')
  })
})
