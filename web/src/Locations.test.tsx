import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Locations } from './Locations'
import { newApiClient } from './api/client'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const org = 'a0000000-0000-4000-8000-000000000001'
// Placeholder sites, not a real clinic's.
const main = {
  id: 'f1000000-0000-4000-8000-000000000001', organizationId: org, name: 'Placeholder main site', addressLine: 'Example street 1',
  postalCode: '71000', cityId: 'Q1270190', phone: '+38733000001', timeZone: 'Europe/Sarajevo', active: true, versionId: 4,
}
const annex = { id: 'f1000000-0000-4000-8000-000000000002', organizationId: org, name: 'Placeholder annex', timeZone: 'Europe/Sarajevo', active: false, versionId: 1 }
const room = { id: 'f2000000-0000-4000-8000-000000000001', locationId: main.id, name: 'Room 1', active: true, versionId: 2 }
const device = { id: 'f3000000-0000-4000-8000-000000000001', locationId: main.id, name: 'Placeholder ultrasound', active: false, versionId: 1 }

const places = {
  places: [
    { id: 'BA-BIH', kind: 'entity', nameLatn: 'Federacija Bosne i Hercegovine', nameCyrl: 'Федерација Босне и Херцеговине' },
    { id: 'BA-09', kind: 'region', parent: 'BA-BIH', nameLatn: 'Kanton Sarajevo', nameCyrl: 'Кантон Сарајево' },
    { id: 'BA-SRP', kind: 'region', nameLatn: 'Republika Srpska', nameCyrl: 'Република Српска' },
    { id: 'Q1270190', kind: 'city', parent: 'BA-09', nameLatn: 'Centar', nameCyrl: 'Центар' },
    { id: 'Q1', kind: 'city', parent: 'BA-SRP', nameLatn: 'Banja Luka', nameCyrl: 'Бања Лука' },
  ],
}

const list = '/clinic/locations'

function renderLocations(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch({ 'GET /places': () => Response.json(places), ...routes })
  renderInLanguage(<Locations api={newApiClient(fake.fetch)} />, language)
  return fake
}

const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } })

describe('Locations', () => {
  it('shows each location with its address, city and phone, and the ones not in use', async () => {
    renderLocations({ [`GET ${list}`]: () => Response.json([main, annex]) })
    const rows = await screen.findAllByRole('listitem')
    expect(rows[0].textContent).toMatch(/Placeholder main site.*Example street 1, 71000 Centar · \+387 33 000 001/)
    expect(within(rows[0]).queryByText('Not in use')).toBeNull()
    expect(rows[1].textContent).toMatch(/Placeholder annex.*Not in use/)
  })

  it('names the city in Cyrillic on a Cyrillic screen', async () => {
    renderLocations({ [`GET ${list}`]: () => Response.json([main]) }, 'sr-Cyrl')
    expect((await screen.findAllByRole('listitem'))[0].textContent).toMatch(/71000 Центар/)
  })

  it('says so when the locations cannot be loaded', async () => {
    renderLocations({ [`GET ${list}`]: () => apiError(500, 'internal') })
    expect(await screen.findByText('The locations could not be loaded. Reload the page to try again.')).toBeTruthy()
  })

  it('adds a location with a city and a phone typed in groups', async () => {
    const { requests } = renderLocations({
      [`GET ${list}`]: () => Response.json([main]),
      [`POST ${list}`]: async (r) => Response.json({ ...(await r.json()), id: annex.id, organizationId: org, timeZone: 'Europe/Sarajevo', versionId: 1 }, { status: 201 }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Add a location' }))
    type('Name', ' Placeholder second site ')
    type('Street address (optional)', 'Example road 2')
    type('Postal code (optional)', '78000')
    type('City (optional)', 'Q1')
    type('Phone (optional)', '+387 51 000 002')
    fireEvent.click(screen.getByRole('button', { name: 'Save location' }))
    expect(await screen.findByText('Location saved.')).toBeTruthy()
    expect(screen.getByText('Placeholder second site')).toBeTruthy()
    const post = requests.find((r) => r.method === 'POST')!
    expect(await post.json()).toEqual({ name: 'Placeholder second site', addressLine: 'Example road 2', postalCode: '78000', cityId: 'Q1', phone: '+38751000002', active: true })
    expectCsrfOnWrites(requests)
  })

  it('changes a location at the version it read and keeps its time zone', async () => {
    const { requests } = renderLocations({
      [`GET ${list}`]: () => Response.json([main]),
      [`PUT ${list}/${main.id}`]: async (r) => Response.json({ ...main, ...(await r.json()), versionId: 5 }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder main site' }))
    expect((screen.getByLabelText('Phone (optional)') as HTMLInputElement).value).toBe('+387 33 000 001')
    type('Street address (optional)', '')
    fireEvent.click(screen.getByLabelText('In use'))
    fireEvent.click(screen.getByRole('button', { name: 'Save location' }))
    expect(await screen.findByText('Location saved.')).toBeTruthy()
    const put = requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"4"')
    expect(await put.json()).toEqual({ name: main.name, postalCode: '71000', cityId: 'Q1270190', phone: '+38733000001', timeZone: 'Europe/Sarajevo', active: false })
  })

  it.each([
    [400, 'invalid_location', 'Check the name, the city and the phone number.'],
    [412, 'version_conflict', 'Someone else changed this location. Reload the page and try again.'],
    [403, 'forbidden', 'Only the clinic admin can change the locations.'],
    [500, 'internal', 'The location could not be saved. Try again.'],
  ])('says why a save was refused (%i %s)', async (status, code, message) => {
    renderLocations({ [`GET ${list}`]: () => Response.json([main]), [`PUT ${list}/${main.id}`]: () => apiError(status, code) })
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder main site' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save location' }))
    expect(await screen.findByText(message)).toBeTruthy()
  })

  it('refuses an empty name without asking the server', async () => {
    const { requests } = renderLocations({ [`GET ${list}`]: () => Response.json([]) })
    fireEvent.click(await screen.findByRole('button', { name: 'Add a location' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save location' }))
    expect(await screen.findByText('Enter a name.')).toBeTruthy()
    expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
  })
})

describe('Locations: rooms and devices', () => {
  const routes = (more: Record<string, Route> = {}) => ({
    [`GET ${list}`]: () => Response.json([main]),
    [`GET ${list}/${main.id}/rooms`]: () => Response.json([room]),
    [`GET ${list}/${main.id}/devices`]: () => Response.json([device]),
    ...more,
  })

  it('lists a location’s rooms and devices once opened', async () => {
    renderLocations(routes())
    fireEvent.click(await screen.findByRole('button', { name: 'Rooms and devices of Placeholder main site' }))
    const rooms = await screen.findByRole('list', { name: 'Rooms' })
    expect(within(rooms).getByText('Room 1')).toBeTruthy()
    const devices = screen.getByRole('list', { name: 'Devices' })
    expect(within(devices).getByRole('listitem').textContent).toMatch(/Placeholder ultrasound.*Not in use/)
  })

  it('adds a room', async () => {
    const { requests } = renderLocations(
      routes({
        [`POST ${list}/${main.id}/rooms`]: async (r) => Response.json({ ...(await r.json()), id: 'f2000000-0000-4000-8000-000000000002', locationId: main.id, versionId: 1 }, { status: 201 }),
      }),
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Rooms and devices of Placeholder main site' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Add a room' }))
    type('Room name', ' Room 2 ')
    fireEvent.click(screen.getByRole('button', { name: 'Save room' }))
    expect(await within(screen.getByRole('list', { name: 'Rooms' })).findByText('Room 2')).toBeTruthy()
    expect(await requests.find((r) => r.method === 'POST')!.json()).toEqual({ name: 'Room 2', active: true })
    expectCsrfOnWrites(requests)
  })

  it('puts a device back in use at the version it read', async () => {
    const { requests } = renderLocations(
      routes({ [`PUT /clinic/devices/${device.id}`]: async (r) => Response.json({ ...device, ...(await r.json()), versionId: 2 }) }),
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Rooms and devices of Placeholder main site' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Placeholder ultrasound' }))
    fireEvent.click(screen.getByLabelText('In use'))
    fireEvent.click(screen.getByRole('button', { name: 'Save device' }))
    const devices = screen.getByRole('list', { name: 'Devices' })
    expect(await within(devices).findByRole('button', { name: 'Edit Placeholder ultrasound' })).toBeTruthy()
    expect(within(devices).queryByText('Not in use')).toBeNull()
    const put = requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"1"')
    expect(await put.json()).toEqual({ name: 'Placeholder ultrasound', active: true })
  })

  it('says so when a room is refused', async () => {
    renderLocations(routes({ [`PUT /clinic/rooms/${room.id}`]: () => apiError(412, 'version_conflict') }))
    fireEvent.click(await screen.findByRole('button', { name: 'Rooms and devices of Placeholder main site' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Edit Room 1' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save room' }))
    expect(await screen.findByText('Someone else changed this. Reload the page and try again.')).toBeTruthy()
  })

  it('says so when the rooms and devices cannot be loaded', async () => {
    renderLocations(routes({ [`GET ${list}/${main.id}/rooms`]: () => apiError(500, 'internal') }))
    fireEvent.click(await screen.findByRole('button', { name: 'Rooms and devices of Placeholder main site' }))
    expect(await screen.findByText('The rooms and devices could not be loaded. Reload the page to try again.')).toBeTruthy()
  })
})
