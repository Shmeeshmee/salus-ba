import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Availability } from './Availability'
import { newApiClient, type Account } from './api/client'
import { newFhirClient } from './api/fhir'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'
import { admin, clinicRoute, cy, dee, practitioner, staffRoute } from './test/fixtures'

afterEach(cleanup)

const path = (roleId: string) => `/practitioner-roles/${roleId}/availability`
const saved = {
  id: 'b1000000-0000-4000-8000-000000000001',
  practitionerRoleId: cy.roleId,
  slotMinutes: 30,
  availability: [{ day: 1, start: '09:00', end: '12:00' }],
  versionId: 2,
}

function renderEditor(account: Account, routes: Record<string, Route>) {
  const fake = fakeFetch({ ...clinicRoute, ...staffRoute, ...routes })
  render(<Availability api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} />)
  return fake
}

function rows() {
  return within(screen.getByRole('list', { name: 'Working hours' })).queryAllByRole('listitem')
}

describe('Availability', () => {
  it("shows the practitioner's own slot length and weekly hours in the clinic zone", async () => {
    renderEditor(practitioner, { [`GET ${path(cy.roleId)}`]: () => Response.json(saved) })

    expect(((await screen.findByLabelText('Slot length in minutes')) as HTMLInputElement).value).toBe('30')
    expect(rows()).toHaveLength(1)
    expect((screen.getByLabelText('Day 1') as HTMLSelectElement).value).toBe('1')
    expect((screen.getByLabelText('From 1') as HTMLInputElement).value).toBe('09:00')
    expect((screen.getByLabelText('To 1') as HTMLInputElement).value).toBe('12:00')
    expect(screen.getByText('Times in Europe/Sarajevo')).toBeTruthy()
  })

  it('adds, edits and saves hours with If-Match of the version read, then of the version saved', async () => {
    let version = 2
    const { requests } = renderEditor(practitioner, {
      [`GET ${path(cy.roleId)}`]: () => Response.json(saved),
      [`PUT ${path(cy.roleId)}`]: async (r) => {
        version++
        const body = (await r.json()) as typeof saved
        return Response.json({ ...saved, ...body, versionId: version })
      },
    })

    await screen.findByLabelText('Day 1')
    fireEvent.click(screen.getByRole('button', { name: 'Add hours' }))
    fireEvent.change(screen.getByLabelText('Day 2'), { target: { value: '3' } })
    fireEvent.change(screen.getByLabelText('From 2'), { target: { value: '14:00' } })
    fireEvent.change(screen.getByLabelText('To 2'), { target: { value: '16:30' } })
    fireEvent.change(screen.getByLabelText('Slot length in minutes'), { target: { value: '20' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved. Free times follow within a few seconds.')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    const puts = () => requests.filter((r) => r.method === 'PUT')
    await vi.waitFor(() => expect(puts()).toHaveLength(2))
    expect(puts()[0].headers.get('If-Match')).toBe('W/"2"')
    expect(await puts()[0].json()).toEqual({
      slotMinutes: 20,
      availability: [
        { day: 1, start: '09:00', end: '12:00' },
        { day: 3, start: '14:00', end: '16:30' },
      ],
    })
    expect(puts()[1].headers.get('If-Match')).toBe('W/"3"')
    expectCsrfOnWrites(requests)
  })

  it('removes hours', async () => {
    const { requests } = renderEditor(practitioner, {
      [`GET ${path(cy.roleId)}`]: () => Response.json(saved),
      [`PUT ${path(cy.roleId)}`]: () => Response.json({ ...saved, availability: [], versionId: 3 }),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Remove hours 1' }))
    expect(rows()).toHaveLength(0)
    expect(screen.getByText('No working hours yet. Patients cannot book until there are some.')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await screen.findByRole('status')
    expect(await requests.find((r) => r.method === 'PUT')!.json()).toEqual({ slotMinutes: 30, availability: [] })
  })

  it('starts an empty schedule at version 0 with a 30-minute slot', async () => {
    const { requests } = renderEditor(practitioner, {
      [`GET ${path(cy.roleId)}`]: () => Response.json({ practitionerRoleId: cy.roleId, availability: [], versionId: 0 }),
      [`PUT ${path(cy.roleId)}`]: () => Response.json({ ...saved, versionId: 1 }),
    })

    expect(((await screen.findByLabelText('Slot length in minutes')) as HTMLInputElement).value).toBe('30')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await screen.findByRole('status')
    expect(requests.find((r) => r.method === 'PUT')!.headers.get('If-Match')).toBe('W/"0"')
  })

  it('lets an admin pick whose hours to edit', async () => {
    const { requests } = renderEditor(admin, {
      [`GET ${path(cy.roleId)}`]: () => Response.json(saved),
      [`GET ${path(dee.roleId)}`]: () => Response.json({ practitionerRoleId: dee.roleId, availability: [], versionId: 0 }),
    })

    fireEvent.change(await screen.findByLabelText('Practitioner'), { target: { value: dee.practitionerId } })

    expect(await screen.findByText('No working hours yet. Patients cannot book until there are some.')).toBeTruthy()
    expect(requests.map((r) => new URL(r.url).pathname)).toContain(path(dee.roleId))
  })

  it.each([
    [400, 'invalid_slot_minutes', 'Enter a slot length of 5 to 240 minutes, in steps of 5.'],
    [400, 'invalid_availability', 'Check the hours: each must end after it starts, and hours on the same day must not overlap.'],
    [412, 'version_conflict', 'Someone else changed these hours. Reload to see their changes.'],
    [403, 'forbidden', 'You cannot change these hours.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderEditor(practitioner, {
      [`GET ${path(cy.roleId)}`]: () => Response.json(saved),
      [`PUT ${path(cy.roleId)}`]: () => apiError(status, code),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Save' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
  })

  it('reloads the hours after a conflict', async () => {
    let reads = 0
    renderEditor(practitioner, {
      [`GET ${path(cy.roleId)}`]: () => {
        reads++
        return Response.json(reads === 1 ? saved : { ...saved, slotMinutes: 45, versionId: 5 })
      },
      [`PUT ${path(cy.roleId)}`]: () => apiError(412, 'version_conflict'),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Save' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Reload' }))

    await vi.waitFor(() => expect((screen.getByLabelText('Slot length in minutes') as HTMLInputElement).value).toBe('45'))
  })

  it('says when the hours cannot be loaded', async () => {
    renderEditor(practitioner, { [`GET ${path(cy.roleId)}`]: () => apiError(500, 'internal error') })

    expect((await screen.findByRole('alert')).textContent).toBe('The working hours could not be loaded. Reload the page to try again.')
  })
})
