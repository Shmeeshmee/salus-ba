import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Subscription, SubscriptionBanner } from './Subscription'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const membership = {
  planCode: 'basic',
  planName: 'Basic',
  pricePerSeatMinor: 10000,
  currency: 'BAM',
  status: 'active',
  currentPeriodStart: '2026-12-01T11:00:00Z',
  currentPeriodEnd: '2027-01-01T11:00:00Z',
}
const invoice = {
  id: '0d000000-0000-4000-8000-000000000001',
  number: 7,
  periodStart: '2026-12-01',
  periodEnd: '2027-01-01',
  seats: 3,
  amountMinor: 30000,
  currency: 'BAM',
  issuedAt: '2026-12-01T11:00:00Z',
  dueAt: '2026-12-15T11:00:00Z',
  status: 'open',
}

function renderScreen(routes: Record<string, Route> = {}) {
  const fake = fakeFetch({
    'GET /clinic/subscription': () => Response.json(membership),
    'GET /clinic/subscription/invoices': () => Response.json([invoice]),
    'GET /clinic/billing-details': () => Response.json({ vatRegistered: false }),
    ...routes,
  })
  renderInLanguage(<Subscription api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('Subscription', () => {
  it('shows the plan, its price per practitioner, the status and the invoices', async () => {
    renderScreen()
    expect(await screen.findByText('Basic')).toBeTruthy()
    expect(screen.getByText(/100\.00.*per active practitioner per month/)).toBeTruthy()
    expect(screen.getByText('Active')).toBeTruthy()
    const row = (await screen.findByText('7')).closest('tr')
    expect(row?.textContent).toContain('300.00')
    expect(row?.textContent).toContain('Open')
    expect(row?.textContent).toMatch(/1 Dec.* to 31 Dec/)
  })

  it('says so when the clinic has no membership yet', async () => {
    renderScreen({
      'GET /clinic/subscription': () => apiError(404, 'not_found'),
      'GET /clinic/subscription/invoices': () => apiError(404, 'not_found'),
    })
    expect(await screen.findByText('No membership has been set up for this clinic yet.')).toBeTruthy()
    expect(await screen.findByText('No invoices yet.')).toBeTruthy()
  })

  it('reports a membership that could not be loaded', async () => {
    renderScreen({ 'GET /clinic/subscription': () => apiError(500, 'internal') })
    expect(await screen.findByText('The membership could not be loaded. Reload the page to try again.')).toBeTruthy()
  })

  it('saves the billing details, leaving out what is not stated', async () => {
    let sent: unknown
    const fake = renderScreen({
      'PUT /clinic/billing-details': async (r) => {
        sent = await r.json()
        return Response.json(sent)
      },
    })
    fireEvent.change(await screen.findByLabelText('Legal name'), { target: { value: ' Synthetic clinic d.o.o. ' } })
    fireEvent.change(screen.getByLabelText('Tax ID'), { target: { value: '4200000000001' } })
    fireEvent.click(screen.getByLabelText('Registered for VAT'))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Billing details saved.')).toBeTruthy()
    expect(sent).toEqual({ vatRegistered: true, legalName: 'Synthetic clinic d.o.o.', taxId: '4200000000001' })
    expectCsrfOnWrites(fake.requests)
  })

  it('refuses a tax id that is not 12 or 13 digits before asking the api', async () => {
    const fake = renderScreen()
    fireEvent.change(await screen.findByLabelText('Tax ID'), { target: { value: '42' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Check the tax ID and the e-mail address.')).toBeTruthy()
    expect(fake.requests.some((r) => r.method === 'PUT')).toBe(false)
  })

  it('reports a save the api refused', async () => {
    renderScreen({ 'PUT /clinic/billing-details': () => apiError(500, 'internal') })
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }))
    expect(await screen.findByText('The billing details could not be saved. Try again.')).toBeTruthy()
  })
})

describe('SubscriptionBanner', () => {
  const days = (n: number) => new Date(Date.now() - n * 24 * 60 * 60 * 1000).toISOString()
  function renderBanner(route: Route) {
    const fake = fakeFetch({ 'GET /clinic/subscription': route })
    renderInLanguage(<SubscriptionBanner api={newApiClient(fake.fetch)} />, 'en')
    return fake
  }

  it('shows from 14 days past due, with a link to the invoices', async () => {
    renderBanner(() => Response.json({ ...membership, status: 'past_due', pastDueSince: days(15) }))
    expect((await screen.findByRole('alert')).textContent).toContain('New bookings stop 30 days after the due date.')
    expect(screen.getByRole('link', { name: 'View invoices' }).getAttribute('href')).toBe('/clinic/subscription')
  })

  it('says that new bookings are stopped and care is not when the membership is suspended', async () => {
    renderBanner(() => Response.json({ ...membership, status: 'suspended', pastDueSince: days(31) }))
    expect((await screen.findByRole('alert')).textContent).toContain('Existing appointments and records are not affected.')
  })

  it('stays out of the way before 14 days, without a membership and when the read fails', async () => {
    for (const route of [
      () => Response.json({ ...membership, status: 'past_due', pastDueSince: days(13) }),
      () => Response.json(membership),
      () => apiError(404, 'not_found'),
      () => apiError(500, 'internal'),
    ]) {
      const fake = renderBanner(route)
      await waitFor(() => expect(fake.requests.length).toBe(1))
      await Promise.resolve()
      expect(screen.queryByRole('alert')).toBeNull()
      cleanup()
    }
  })
})
