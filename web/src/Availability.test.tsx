import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Availability } from './Availability'
import { newApiClient, type Account } from './api/client'
import { newFhirClient } from './api/fhir'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { admin, clinicRoute, cy, dee, practitioner, staffRoute } from './test/fixtures'
import { questionnaireFHIR } from './test/questionnaires'
import { responsesBundle } from './test/responses'

afterEach(cleanup)

const path = (roleId: string) => `/practitioner-roles/${roleId}/availability`
const saved = {
  id: 'b1000000-0000-4000-8000-000000000001',
  practitionerRoleId: cy.roleId,
  slotMinutes: 30,
  availability: [{ day: 1, start: '09:00', end: '12:00' }],
  versionId: 2,
}

const defaultPath = (roleId: string) => `/practitioner-roles/${roleId}/questionnaire`
const none = (roleId: string) => () => Response.json({ practitionerRoleId: roleId, versionId: 0 })
const intake = { ...questionnaireFHIR(), status: 'active' as const }
const followUp = { ...intake, id: 'q2', url: 'https://salus.invalid/fhir/Questionnaire/s2', title: 'Placeholder follow-up' }
const actives = 'GET /fhir/R4/Questionnaire'
const activesRoute = { [actives]: () => fhirJSON(responsesBundle([], [intake, followUp])) }

function renderEditor(account: Account, routes: Record<string, Route>) {
  const fake = fakeFetch({
    ...clinicRoute,
    ...staffRoute,
    ...activesRoute,
    [`GET ${defaultPath(cy.roleId)}`]: none(cy.roleId),
    [`GET ${defaultPath(dee.roleId)}`]: none(dee.roleId),
    ...routes,
  })
  render(<Availability api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} />)
  return fake
}

function rows() {
  return within(screen.getByRole('list', { name: 'Working hours' })).queryAllByRole('listitem')
}

describe('Availability', () => {
  it('renders the hours and the default questionnaire without a duplicate-key warning', async () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      renderEditor(practitioner, { [`GET ${path(cy.roleId)}`]: () => Response.json(saved) })
      await screen.findByLabelText('Day 1')
      await screen.findByLabelText('Questionnaire for every booking')
      const warnings = errors.mock.calls.map((c) => c.map(String).join(' ')).filter((m) => m.includes('same key'))
      expect(warnings).toEqual([])
    } finally {
      errors.mockRestore()
    }
  })

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

  describe('the questionnaire for every booking', () => {
    const hours = { [`GET ${path(cy.roleId)}`]: () => Response.json(saved) }
    const picker = async () => (await screen.findByLabelText('Questionnaire for every booking')) as HTMLSelectElement
    const options = (select: HTMLSelectElement) => Array.from(select.options).map((o) => o.textContent)

    it("reads the role's default among the clinic's active questionnaires", async () => {
      const { requests } = renderEditor(practitioner, { ...hours, [`GET ${defaultPath(cy.roleId)}`]: () => Response.json({ practitionerRoleId: cy.roleId, url: followUp.url, versionId: 3 }) })

      const select = await picker()
      expect(options(select)).toEqual(['None', 'Placeholder follow-up', 'Placeholder intake'])
      expect(select.value).toBe('q2')
      expect(screen.getByText('Patients who book from now on are asked to fill it in before their appointment.')).toBeTruthy()
      const search = requests.find((r) => new URL(r.url).pathname === '/fhir/R4/Questionnaire')!
      expect(new URL(search.url).searchParams.get('status')).toBe('active')
    })

    it('shows None while the role has no default', async () => {
      renderEditor(practitioner, hours)

      expect((await picker()).value).toBe('')
      expect(screen.queryByRole('alert')).toBeNull()
    })

    it('sets the default with the CSRF header', async () => {
      const { requests } = renderEditor(practitioner, {
        ...hours,
        [`PUT ${defaultPath(cy.roleId)}`]: () => Response.json({ practitionerRoleId: cy.roleId, url: intake.url, versionId: 1 }),
      })

      fireEvent.change(await picker(), { target: { value: 'q1' } })

      await vi.waitFor(() => expect(paths(requests)).toContain(`PUT ${defaultPath(cy.roleId)}`))
      expect(await requests.find((r) => r.method === 'PUT')!.json()).toEqual({ questionnaireId: 'q1' })
      expectCsrfOnWrites(requests)
      await vi.waitFor(async () => expect((await picker()).disabled).toBe(false))
      expect((await picker()).value).toBe('q1')
    })

    it('clears the default with None', async () => {
      const { requests } = renderEditor(practitioner, {
        ...hours,
        [`GET ${defaultPath(cy.roleId)}`]: () => Response.json({ practitionerRoleId: cy.roleId, url: intake.url, versionId: 1 }),
        [`DELETE ${defaultPath(cy.roleId)}`]: () => new Response(null, { status: 204 }),
      })

      const select = await picker()
      await vi.waitFor(() => expect(select.value).toBe('q1'))
      fireEvent.change(select, { target: { value: '' } })

      await vi.waitFor(() => expect(paths(requests)).toContain(`DELETE ${defaultPath(cy.roleId)}`))
      expectCsrfOnWrites(requests)
      await vi.waitFor(async () => expect((await picker()).disabled).toBe(false))
      expect((await picker()).value).toBe('')
    })

    it.each([
      [409, 'not_active', 'This questionnaire is not active. Choose an active one.'],
      [500, 'internal error', 'Something went wrong. Try again.'],
    ])('keeps the default and says why on %i %s', async (status, code, message) => {
      renderEditor(practitioner, { ...hours, [`PUT ${defaultPath(cy.roleId)}`]: () => apiError(status, code) })

      fireEvent.change(await picker(), { target: { value: 'q1' } })

      await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
      expect((await picker()).value).toBe('')
    })

    it('says so when the default has no active version any more', async () => {
      renderEditor(practitioner, {
        ...hours,
        [`GET ${defaultPath(cy.roleId)}`]: () => Response.json({ practitionerRoleId: cy.roleId, url: 'https://salus.invalid/fhir/Questionnaire/retired', versionId: 2 }),
      })

      expect((await screen.findByRole('alert')).textContent).toBe('This questionnaire is not active. Choose an active one.')
      expect((await picker()).value).toBe('')
    })

    it("lets an admin read and set another practitioner's role default", async () => {
      const { requests } = renderEditor(admin, {
        ...hours,
        [`GET ${path(dee.roleId)}`]: () => Response.json({ practitionerRoleId: dee.roleId, availability: [], versionId: 0 }),
        [`PUT ${defaultPath(dee.roleId)}`]: () => Response.json({ practitionerRoleId: dee.roleId, url: followUp.url, versionId: 1 }),
      })

      await picker()
      fireEvent.change(await screen.findByLabelText('Practitioner'), { target: { value: dee.practitionerId } })
      await vi.waitFor(() => expect(paths(requests)).toContain(`GET ${defaultPath(dee.roleId)}`))
      fireEvent.change(await picker(), { target: { value: 'q2' } })

      await vi.waitFor(() => expect(paths(requests)).toContain(`PUT ${defaultPath(dee.roleId)}`))
    })

    it('says when the default cannot be loaded', async () => {
      renderEditor(practitioner, { ...hours, [actives]: () => outcome(500, 'exception') })

      expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
      expect(screen.queryByLabelText('Questionnaire for every booking')).toBeNull()
    })
  })
})
