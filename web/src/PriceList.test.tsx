import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { PriceList } from './PriceList'
import { newApiClient } from './api/client'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const path = '/clinic/services'
const id = 'd0000000-0000-4000-8000-000000000001'
// Placeholder services, not a real clinic's.
const exam = {
  id,
  name: 'Placeholder exam',
  code: 'EX-1',
  priceMinor: 5000,
  currency: 'BAM',
  durationMinutes: 30,
  vatRateBp: 1700,
  modes: ['in-person'],
  active: true,
  translations: [{ language: 'de', name: 'Platzhalter' }],
  versionId: 3,
}
const call = { id: 'd0000000-0000-4000-8000-000000000002', name: 'Placeholder call', priceMinor: 2550, currency: 'BAM', modes: ['video'], active: false, translations: [], versionId: 1 }

function renderList(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch({ 'GET /clinic/locations': () => Response.json([]), ...routes })
  renderInLanguage(<PriceList api={newApiClient(fake.fetch)} />, language)
  return fake
}

const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } })

describe('PriceList', () => {
  it('shows each service with its price, duration, VAT and modes', async () => {
    renderList({ [`GET ${path}`]: () => Response.json([call, exam]) })
    const rows = await screen.findAllByRole('listitem')
    expect(rows[0].textContent).toMatch(/Placeholder call.*BAM\s25\.50 · VAT exempt · By video.*Not offered/)
    expect(rows[1].textContent).toMatch(/Placeholder exam.*BAM\s50\.00 · 30 minutes · VAT 17% · At the clinic/)
    expect(within(rows[1]).queryByText('Not offered')).toBeNull()
  })

  it('formats the price in the language of the screen', async () => {
    renderList({ [`GET ${path}`]: () => Response.json([exam]) }, 'bs')
    expect((await screen.findByRole('listitem')).textContent).toMatch(/50,00\sBAM/)
  })

  it('says so when the clinic has no services, and when the list cannot be loaded', async () => {
    renderList({ [`GET ${path}`]: () => Response.json([]) })
    expect(await screen.findByText('The price list has no services yet.')).toBeTruthy()
    cleanup()
    renderList({ [`GET ${path}`]: () => apiError(500, 'internal') })
    expect(await screen.findByText('The price list could not be loaded. Reload the page to try again.')).toBeTruthy()
  })

  it('imports services from a file and reloads the list', async () => {
    let reads = 0
    const { requests } = renderList({
      [`GET ${path}`]: () => Response.json(reads++ === 0 ? [] : [exam]),
      [`POST ${path}/import`]: () => Response.json({ saved: true, rows: [{ line: 2, errors: [], service: { ...exam, id: undefined, versionId: undefined } }] }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Import from a file' }))
    fireEvent.change(screen.getByLabelText('File (CSV or XLSX)'), { target: { files: [new File(['name;price\n'], 'services.csv')] } })
    fireEvent.click(await screen.findByRole('button', { name: 'Import 1 service' }))
    expect(await screen.findByText('1 service imported.')).toBeTruthy()
    expect(await screen.findByText('Placeholder exam')).toBeTruthy()
    expect(requests.filter((r) => r.method === 'GET')).toHaveLength(2)
  })

  it('adds a service', async () => {
    const { requests } = renderList({
      [`GET ${path}`]: () => Response.json([exam]),
      [`POST ${path}`]: async (r) => Response.json({ ...(await r.json()), id: call.id, versionId: 1 }, { status: 201 }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Add a service' }))
    type('Name', ' A placeholder ')
    type('Price in BAM', '25,5')
    type('Deutsch', 'Platzhalter')
    fireEvent.click(screen.getByLabelText('At the clinic'))
    fireEvent.click(screen.getByRole('button', { name: 'Save service' }))

    expect(await screen.findByText('Saved.')).toBeTruthy()
    const body = await requests.find((r) => r.method === 'POST')!.json()
    expect(body).toEqual({ name: 'A placeholder', priceMinor: 2550, currency: 'BAM', modes: ['video'], active: true, translations: [{ language: 'de', name: 'Platzhalter' }] })
    expectCsrfOnWrites(requests)
    expect(screen.getAllByRole('listitem').map((li) => li.querySelector('.strong')?.textContent)).toEqual(['A placeholder', 'Placeholder exam'])
  })

  it('changes a service at the version it was read at', async () => {
    const { requests } = renderList({
      [`GET ${path}`]: () => Response.json([exam]),
      [`PUT ${path}/${id}`]: async (r) => Response.json({ ...(await r.json()), id, versionId: 4 }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
    expect((screen.getByLabelText('Price in BAM') as HTMLInputElement).value).toBe('50.00')
    expect((screen.getByLabelText('VAT rate in % (optional)') as HTMLInputElement).value).toBe('17')
    expect((screen.getByLabelText('Deutsch') as HTMLInputElement).value).toBe('Platzhalter')
    type('Price in BAM', '60')
    type('VAT rate in % (optional)', '')
    type('Duration in minutes (optional)', '')
    fireEvent.click(screen.getByLabelText('Offered to patients'))
    fireEvent.click(screen.getByRole('button', { name: 'Save service' }))

    expect(await screen.findByText('Not offered')).toBeTruthy()
    const put = requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"3"')
    expect(await put.json()).toEqual({ name: 'Placeholder exam', code: 'EX-1', priceMinor: 6000, currency: 'BAM', modes: ['in-person'], active: false, translations: [{ language: 'de', name: 'Platzhalter' }] })
  })

  it('sends nothing for a service it can tell is invalid', async () => {
    const { requests } = renderList({ [`GET ${path}`]: () => Response.json([]) })
    fireEvent.click(await screen.findByRole('button', { name: 'Add a service' }))
    const save = () => fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
    save()
    expect(await screen.findByText('Enter a name.')).toBeTruthy()
    type('Name', 'A placeholder')
    type('Price in BAM', '12.345')
    save()
    expect(await screen.findByText('Enter a price such as 50 or 49.90.')).toBeTruthy()
    type('Price in BAM', '0')
    type('Duration in minutes (optional)', '3')
    save()
    expect(await screen.findByText('Enter a duration from 5 to 480 minutes, or leave it empty.')).toBeTruthy()
    type('Duration in minutes (optional)', '')
    type('VAT rate in % (optional)', '101')
    save()
    expect(await screen.findByText('Enter a VAT rate from 0 to 100, or leave it empty.')).toBeTruthy()
    type('VAT rate in % (optional)', '')
    fireEvent.click(screen.getByLabelText('At the clinic'))
    fireEvent.click(screen.getByLabelText('By video'))
    save()
    expect(await screen.findByText('Choose at least one way the service is offered.')).toBeTruthy()
    expect(requests.filter((r) => r.method !== 'GET')).toEqual([])
  })

  it.each([
    [412, 'version_conflict', 'Someone else changed this service. Reload the page to see it, then make your change again.'],
    [403, 'forbidden', 'Only a clinic admin can change the price list.'],
    [400, 'invalid_service', 'The service was not accepted. Check its fields and try again.'],
    [500, 'internal', 'The service could not be saved. Try again.'],
  ])('explains a refused save (%i)', async (status, code, message) => {
    renderList({ [`GET ${path}`]: () => Response.json([exam]), [`PUT ${path}/${id}`]: () => apiError(status, code) })
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
    expect(await screen.findByText(message)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('button', { name: 'Save service' })).toBeNull()
  })

  describe('the device a service needs', () => {
    // Placeholder locations and devices, not a real clinic's.
    const siteA = { id: 'a5000000-0000-4000-8000-0000000000a1', name: 'Centre', active: true, versionId: 1 }
    const siteB = { id: 'a5000000-0000-4000-8000-0000000000a2', name: 'Branch', active: true, versionId: 1 }
    const closed = { id: 'a5000000-0000-4000-8000-0000000000a3', name: 'Closed site', active: false, versionId: 2 }
    const scanner = { id: 'a8000000-0000-4000-8000-0000000000a1', locationId: siteA.id, name: 'Placeholder scanner', active: true, versionId: 1 }
    const retired = { id: 'a8000000-0000-4000-8000-0000000000a2', locationId: siteA.id, name: 'Retired device', active: false, versionId: 2 }
    const monitor = { id: 'a8000000-0000-4000-8000-0000000000b1', locationId: siteB.id, name: 'Placeholder monitor', active: true, versionId: 1 }
    const sites = {
      'GET /clinic/locations': () => Response.json([siteA, siteB, closed]),
      [`GET /clinic/locations/${siteA.id}/devices`]: () => Response.json([retired, scanner]),
      [`GET /clinic/locations/${siteB.id}/devices`]: () => Response.json([monitor]),
    }
    const picker = () => screen.findByLabelText<HTMLSelectElement>('Device (optional)')

    it('offers the devices in use of each location in use, grouped by location, and none by default', async () => {
      const { requests } = renderList({
        ...sites,
        [`GET ${path}`]: () => Response.json([]),
        [`POST ${path}`]: async (r) => Response.json({ ...(await r.json()), id: call.id, versionId: 1 }, { status: 201 }),
      })
      fireEvent.click(await screen.findByRole('button', { name: 'Add a service' }))
      const select = await picker()
      expect(select.value).toBe('')
      expect([...select.querySelectorAll('optgroup')].map((g) => [g.label, [...g.querySelectorAll('option')].map((o) => o.textContent)])).toEqual([
        ['Centre', ['Placeholder scanner']],
        ['Branch', ['Placeholder monitor']],
      ])
      expect(within(select).getByRole('option', { name: 'None' })).toBeTruthy()
      expect(requests.some((r) => r.url.includes(closed.id))).toBe(false)

      type('Name', 'A placeholder')
      type('Price in BAM', '40')
      fireEvent.change(select, { target: { value: monitor.id } })
      fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
      expect(await screen.findByText('Saved.')).toBeTruthy()
      expect((await requests.find((r) => r.method === 'POST')!.json()).deviceId).toBe(monitor.id)
    })

    it('sends the device back on a change, and none once it is set to none', async () => {
      const withDevice = { ...exam, deviceId: scanner.id }
      const { requests } = renderList({
        ...sites,
        [`GET ${path}`]: () => Response.json([withDevice]),
        [`PUT ${path}/${id}`]: async (r) => Response.json({ ...(await r.json()), id, versionId: 4 }),
      })
      fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
      expect((await picker()).value).toBe(scanner.id)
      type('Price in BAM', '60')
      fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
      expect(await screen.findByText('Saved.')).toBeTruthy()
      expect((await requests.filter((r) => r.method === 'PUT')[0].json()).deviceId).toBe(scanner.id)

      fireEvent.click(screen.getByRole('button', { name: 'Edit Placeholder exam' }))
      fireEvent.change(await picker(), { target: { value: '' } })
      fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
      expect(await screen.findByText('Saved.')).toBeTruthy()
      expect(await requests.filter((r) => r.method === 'PUT')[1].json()).not.toHaveProperty('deviceId')
    })

    it('shows a device out of use that the service still names, marked so', async () => {
      renderList({ ...sites, [`GET ${path}`]: () => Response.json([{ ...exam, deviceId: retired.id }]) })
      fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
      const select = await picker()
      expect(select.value).toBe(retired.id)
      expect(within(select).getByRole('option', { name: 'Retired device (not in use)' })).toBeTruthy()
    })

    it('shows no picker without devices, and keeps the device when the locations cannot be read', async () => {
      renderList({ [`GET ${path}`]: () => Response.json([exam]) })
      fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
      expect(await screen.findByLabelText('Name')).toBeTruthy()
      expect(screen.queryByLabelText('Device (optional)')).toBeNull()
      cleanup()

      const { requests } = renderList({
        'GET /clinic/locations': () => apiError(500, 'internal'),
        [`GET ${path}`]: () => Response.json([{ ...exam, deviceId: scanner.id }]),
        [`PUT ${path}/${id}`]: async (r) => Response.json({ ...(await r.json()), id, versionId: 4 }),
      })
      fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder exam' }))
      fireEvent.click(screen.getByRole('button', { name: 'Save service' }))
      expect(await screen.findByText('Saved.')).toBeTruthy()
      expect(screen.queryByLabelText('Device (optional)')).toBeNull()
      expect((await requests.find((r) => r.method === 'PUT')!.json()).deviceId).toBe(scanner.id)
    })
  })
})
