import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PriceListImport } from './PriceListImport'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const path = '/clinic/services/import'
// Placeholder services, not a real clinic's.
const exam = { name: 'Placeholder exam', priceMinor: 2550, currency: 'BAM', modes: ['in-person'], active: true, translations: [] }
const call = { name: 'Placeholder call', priceMinor: 1000, currency: 'BAM', durationMinutes: 20, vatRateBp: 1700, modes: ['video'], active: true, translations: [] }

function renderImport(route: Route) {
  const fake = fakeFetch({ [`POST ${path}`]: route })
  const onImported = vi.fn()
  const onCancel = vi.fn()
  renderInLanguage(<PriceListImport api={newApiClient(fake.fetch)} onImported={onImported} onCancel={onCancel} />, 'en')
  return { ...fake, onImported, onCancel }
}

const choose = () =>
  fireEvent.change(screen.getByLabelText('File (CSV or XLSX)'), { target: { files: [new File(['name;price\n'], 'services.csv', { type: 'text/csv' })] } })

describe('PriceListImport', () => {
  it('previews the file, then imports every row', async () => {
    const { requests, onImported } = renderImport(async (r) => {
      const save = new URL(r.url).searchParams.get('save') === 'true'
      return Response.json({ saved: save, rows: [{ line: 2, errors: [], service: exam }, { line: 3, errors: [], service: call }] })
    })
    choose()
    expect(await screen.findByText('2 services read. Nothing is saved until you import.')).toBeTruthy()
    expect(screen.getAllByRole('listitem')[1].textContent).toMatch(/BAM\s10\.00 · 20 minutes · VAT 17% · By video/)
    fireEvent.click(screen.getByRole('button', { name: 'Import 2 services' }))
    await vi.waitFor(() => expect(onImported).toHaveBeenCalledWith(2))
    expect(requests.map((r) => new URL(r.url).search)).toEqual(['', '?save=true'])
    // jsdom's File loses its name and bytes on the way into undici's FormData; the part and its type arrive.
    expect(await requests[0].text()).toMatch(/name="file"; filename="[^"]+"\r\nContent-Type: text\/csv/)
    expectCsrfOnWrites(requests)
  })

  it('lists every row with an error and imports nothing', async () => {
    const { requests } = renderImport(() =>
      Response.json({ saved: false, rows: [{ line: 2, errors: [], service: exam }, { line: 3, errors: ['name_missing', 'price_invalid'] }, { line: 4, errors: ['duplicate'] }] }),
    )
    choose()
    expect(await screen.findByText('2 of 3 rows have errors. Correct them in the file and choose it again.')).toBeTruthy()
    expect(screen.getByText('Line 3: the name is missing; the price is not a valid amount')).toBeTruthy()
    expect(screen.getByText('Line 4: a service of this name or code is already on the list or the file')).toBeTruthy()
    expect(screen.queryByRole('button', { name: /^Import/ })).toBeNull()
    expect(requests).toHaveLength(1)
  })

  it('says why a file cannot be read', async () => {
    for (const [response, message] of [
      [apiError(400, 'invalid_import'), 'The file could not be read. Use a CSV or XLSX file with a header row naming at least the name and price columns, and at most 1000 rows.'],
      [apiError(413, 'too_large'), 'The file is too large.'],
      [apiError(500, 'internal'), 'The file could not be imported. Try again.'],
    ] as const) {
      renderImport(() => response.clone())
      choose()
      expect(await screen.findByText(message)).toBeTruthy()
      cleanup()
    }
  })

  it('goes back without importing', () => {
    const { onCancel, requests } = renderImport(() => apiError(500, 'internal'))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalled()
    expect(requests).toHaveLength(0)
  })
})
