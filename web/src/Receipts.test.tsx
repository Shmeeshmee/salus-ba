import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { ClinicReports } from './ClinicReports'
import { Receipts } from './Receipts'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'
import { admin, clinicRoute } from './test/fixtures'

afterEach(cleanup)

const now = () => new Date('2026-11-21T10:00:00Z')
const exam = 'c1000000-0000-4000-8000-000000000001'
const first = 'd1000000-0000-4000-8000-000000000001'
const second = 'd1000000-0000-4000-8000-000000000002'

// Synthetic receipts, newest first: a company's invoice still being reported, and a fiscalized receipt to a patient.
const receipts = [
  {
    id: second, kind: 'invoice', number: 2, issuedAt: '2026-11-21T09:30:00Z', companyName: 'Synthetic company d.o.o.', paymentMethod: 'transfer',
    totalMinor: 5000, currency: 'BAM', fiscalStatus: 'pending',
    lines: [{ serviceId: exam, name: 'Placeholder examination', quantity: 1, unitPriceMinor: 5000, discountMinor: 0, totalMinor: 5000 }],
  },
  {
    id: first, kind: 'receipt', number: 1, issuedAt: '2026-11-20T08:00:00Z', patientId: 'aa000000-0000-4000-8000-0000000000aa', paymentMethod: 'cash',
    totalMinor: 10670, currency: 'BAM', fiscalStatus: 'fiscalized', fiscalId: 'TEST-1', fiscalCode: 'code-1',
    lines: [{ serviceId: exam, name: 'Placeholder examination', quantity: 2, unitPriceMinor: 5000, discountMinor: 500, totalMinor: 9500 }],
  },
]
const services = [
  { id: exam, name: 'Placeholder examination', priceMinor: 5000, currency: 'BAM', modes: ['in-person'], active: true, translations: [], versionId: 1 },
  { id: 'c1000000-0000-4000-8000-000000000002', name: 'Placeholder retired item', priceMinor: 100, currency: 'BAM', modes: ['video'], active: false, translations: [], versionId: 1 },
]
const report = {
  from: '2026-10-31T23:00:00Z', to: '2026-11-21T23:00:00Z', count: 2, totalMinor: 15670, currency: 'BAM',
  byPaymentMethod: [{ paymentMethod: 'cash', count: 1, totalMinor: 10670 }, { paymentMethod: 'transfer', count: 1, totalMinor: 5000 }],
  byService: [{ serviceId: exam, name: 'Placeholder examination', quantity: 3, totalMinor: 14500 }],
}

function renderReceipts(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch({
    ...clinicRoute,
    'GET /clinic/receipts': () => Response.json(receipts),
    'GET /clinic/services': () => Response.json(services),
    ...routes,
  })
  renderInLanguage(<Receipts api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} now={now} />, language)
  return fake
}

describe('Receipts', () => {
  it('lists the month in the clinic zone with number, buyer, total, payment, fiscal status and the PDF', async () => {
    const fake = renderReceipts({})
    const rows = within(await screen.findByRole('list', { name: 'Receipts' })).getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('No. 2')
    expect(rows[0].textContent).toContain('Synthetic company d.o.o.')
    expect(rows[0].textContent).toContain('Bank transfer')
    expect(rows[0].textContent).toContain('Reporting')
    expect(rows[1].textContent).toContain('Patient')
    expect(rows[1].textContent).toContain('106.70')
    expect(rows[1].textContent).toContain('Fiscalized')
    expect(rows[1].textContent).not.toContain('aa000000')
    expect(within(rows[1]).getByRole('link', { name: 'Open PDF' }).getAttribute('href')).toContain(`/clinic/receipts/${first}/pdf`)
    const query = new URL(fake.requests.find((r) => r.url.includes('/clinic/receipts'))!.url).searchParams
    // November in Europe/Sarajevo (UTC+1): from the 1st at local midnight to the end of today.
    expect([query.get('from'), query.get('to')]).toEqual(['2026-10-31T23:00:00.000Z', '2026-11-21T23:00:00.000Z'])
  })

  it('reverses a receipt after a confirmation that says it cannot be undone, with the refund method', async () => {
    let body: unknown
    const fake = renderReceipts({
      [`POST /clinic/receipts/${first}/reverse`]: async (r) => {
        body = await r.json()
        return Response.json({ ...receipts[1], id: 'd1000000-0000-4000-8000-000000000003', number: 3, reversesId: first }, { status: 201 })
      },
    })
    const rows = within(await screen.findByRole('list', { name: 'Receipts' })).getAllByRole('listitem')
    fireEvent.click(within(rows[1]).getByRole('button', { name: 'Reverse' }))
    const dialog = within(rows[1]).getByRole('alertdialog', { name: 'Reverse receipt no. 1?' })
    expect(dialog.textContent).toContain('This cannot be undone.')
    fireEvent.change(within(dialog).getByRole('combobox'), { target: { value: 'card' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Reverse' }))
    await waitFor(() => expect(body).toEqual({ paymentMethod: 'card' }))
    await waitFor(() => expect(fake.requests.filter((r) => r.method === 'GET' && r.url.includes('/clinic/receipts?')).length).toBe(2))
    expectCsrfOnWrites(fake.requests)
  })

  it('says so when the receipt was reversed already, and offers no reversal of a reversal', async () => {
    renderReceipts({
      'GET /clinic/receipts': () => Response.json([{ ...receipts[0], reversesId: first }, receipts[1]]),
      [`POST /clinic/receipts/${first}/reverse`]: () => apiError(409, 'already_reversed'),
    })
    const rows = within(await screen.findByRole('list', { name: 'Receipts' })).getAllByRole('listitem')
    expect(within(rows[0]).queryByRole('button', { name: 'Reverse' })).toBeNull()
    expect(rows[0].textContent).toContain('Reversal')
    fireEvent.click(within(rows[1]).getByRole('button', { name: 'Reverse' }))
    fireEvent.click(within(within(rows[1]).getByRole('alertdialog')).getByRole('button', { name: 'Reverse' }))
    expect(await screen.findByText('This receipt was already reversed.')).toBeTruthy()
  })

  it('issues a company invoice from the active price list and shows the new number', async () => {
    let body: unknown
    const fake = renderReceipts({
      'POST /clinic/receipts': async (r) => {
        body = await r.json()
        return Response.json({ ...receipts[0], number: 3 }, { status: 201 })
      },
    })
    fireEvent.click(await screen.findByRole('button', { name: 'New receipt' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New receipt' }))
    const service = await dialog.findByLabelText('Service')
    expect(within(service).queryByText(/retired/)).toBeNull()
    fireEvent.change(service, { target: { value: exam } })
    fireEvent.change(dialog.getByLabelText('Quantity'), { target: { value: '2' } })
    fireEvent.change(dialog.getByLabelText('Discount (BAM)'), { target: { value: '5.5' } })
    expect(dialog.getByText(/Total: .*94\.50/)).toBeTruthy()
    fireEvent.change(dialog.getByLabelText('Buyer'), { target: { value: 'company' } })
    fireEvent.change(dialog.getByLabelText('Company name'), { target: { value: ' Synthetic company d.o.o. ' } })
    fireEvent.change(dialog.getByLabelText('Company tax ID (JIB)'), { target: { value: '4200000000001' } })
    fireEvent.change(dialog.getByLabelText('Paid by'), { target: { value: 'transfer' } })
    expect(dialog.getByText(/An issued receipt cannot be changed/)).toBeTruthy()
    fireEvent.click(dialog.getByRole('button', { name: 'Issue receipt' }))
    await waitFor(() =>
      expect(body).toEqual({
        kind: 'invoice', paymentMethod: 'transfer', companyName: 'Synthetic company d.o.o.', companyTaxId: '4200000000001',
        lines: [{ serviceId: exam, quantity: 2, discountMinor: 550 }],
      }),
    )
    expect(await screen.findByText('Receipt no. 3 was issued.')).toBeTruthy()
    expect(screen.queryByRole('dialog')).toBeNull()
    expectCsrfOnWrites(fake.requests)
  })

  it('sends nothing while a line is incomplete, and explains missing billing details', async () => {
    const fake = renderReceipts({ 'POST /clinic/receipts': () => apiError(409, 'billing_details_missing') })
    fireEvent.click(await screen.findByRole('button', { name: 'New receipt' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New receipt' }))
    fireEvent.click(await dialog.findByRole('button', { name: 'Issue receipt' }))
    expect(fake.requests.some((r) => r.method === 'POST')).toBe(false)
    fireEvent.change(dialog.getByLabelText('Service'), { target: { value: exam } })
    fireEvent.click(dialog.getByRole('button', { name: 'Issue receipt' }))
    expect(await dialog.findByText(/legal name, tax ID and address are needed/)).toBeTruthy()
  })

  it('reads in Serbian Cyrillic', async () => {
    renderReceipts({}, 'sr-Cyrl')
    const rows = within(await screen.findByRole('list', { name: 'Рачуни' })).getAllByRole('listitem')
    expect(rows[1].textContent).toContain('Фискализован')
  })
})

describe('ClinicReports', () => {
  it('shows the period summed by payment method and service, with the CSV of the same period', async () => {
    const fake = fakeFetch({ ...clinicRoute, 'GET /clinic/reports/receipts': () => Response.json(report) })
    renderInLanguage(<ClinicReports api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} now={now} />, 'en')
    const byMethod = within(await screen.findByRole('table', { name: 'By payment method' }))
    expect(byMethod.getByRole('row', { name: /Cash/ }).textContent).toContain('106.70')
    expect(within(screen.getByRole('table', { name: 'By service' })).getByRole('row', { name: /Placeholder examination/ }).textContent).toContain('145.00')
    expect(screen.getByText(/156\.70/)).toBeTruthy()
    const csv = new URL(screen.getByRole('link', { name: 'Download CSV' }).getAttribute('href')!)
    expect([csv.pathname, csv.searchParams.get('format'), csv.searchParams.get('from'), csv.searchParams.get('to')]).toEqual([
      '/clinic/reports/receipts', 'csv', '2026-10-31T23:00:00.000Z', '2026-11-21T23:00:00.000Z',
    ])
  })

  it('says when the period has no receipts', async () => {
    const fake = fakeFetch({ ...clinicRoute, 'GET /clinic/reports/receipts': () => Response.json({ ...report, count: 0, totalMinor: 0, byPaymentMethod: [], byService: [] }) })
    renderInLanguage(<ClinicReports api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} now={now} />, 'en')
    expect(await screen.findByText('No receipts in this period.')).toBeTruthy()
  })
})
