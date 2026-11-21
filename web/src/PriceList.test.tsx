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
  const fake = fakeFetch(routes)
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
})
