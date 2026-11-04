import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicSettings } from './ClinicSettings'
import { newFhirClient, type Organization } from './api/fhir'
import { expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, type Route } from './test/fakeFetch'

afterEach(cleanup)

const id = 'a0000000-0000-4000-8000-00000000000a'
const path = `/fhir/R4/Organization/${id}`
const org: Organization = {
  resourceType: 'Organization',
  id,
  name: 'Synthetic clinic',
  active: true,
  meta: { versionId: '3', lastUpdated: '2026-11-04T09:30:00Z' },
}

function renderSettings(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  render(<ClinicSettings fhir={newFhirClient(fake.fetch)} organizationId={id} />)
  return fake
}

async function rename(name: string) {
  fireEvent.change(await screen.findByLabelText('Clinic name'), { target: { value: name } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
}

describe('ClinicSettings', () => {
  it('shows that the clinic is loading', () => {
    renderSettings({ [`GET ${path}`]: () => new Promise<Response>(() => {}) })

    expect(screen.getByText('Loading')).toBeTruthy()
  })

  it('shows the clinic name from the Organization', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org) })

    expect(((await screen.findByLabelText('Clinic name')) as HTMLInputElement).value).toBe('Synthetic clinic')
  })

  it('saves with If-Match of the version read, then of the version saved', async () => {
    let version = 3
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => {
        version++
        const body = (await r.json()) as Organization
        return fhirJSON({ ...org, name: body.name, meta: { ...org.meta, versionId: String(version) } })
      },
    })

    const puts = () => requests.filter((r) => r.method === 'PUT')

    await rename('Renamed clinic')
    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    await rename('Renamed again')

    await vi.waitFor(() => expect(puts()).toHaveLength(2))
    expect(puts()[0].headers.get('If-Match')).toBe('W/"3"')
    expect(((await puts()[0].json()) as Organization).name).toBe('Renamed clinic')
    expect(puts()[1].headers.get('If-Match')).toBe('W/"4"')
    expectCsrfOnWrites(requests)
  })

  it.each([
    [412, 'conflict', 'Someone else changed the clinic settings. Reload to see their changes.'],
    [422, 'business-rule', 'Enter a name of 1 to 200 characters.'],
    [403, 'forbidden', 'Only a clinic admin can change these settings.'],
    [500, 'exception', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org), [`PUT ${path}`]: () => outcome(status, code) })

    await rename('Renamed clinic')

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })

  it('reloads the clinic after a conflict', async () => {
    let reads = 0
    renderSettings({
      [`GET ${path}`]: () => {
        reads++
        return fhirJSON(reads === 1 ? org : { ...org, name: 'Changed elsewhere', meta: { ...org.meta, versionId: '5' } })
      },
      [`PUT ${path}`]: () => outcome(412, 'conflict'),
    })

    await rename('Renamed clinic')
    fireEvent.click(await screen.findByRole('button', { name: 'Reload' }))

    await vi.waitFor(async () =>
      expect(((await screen.findByLabelText('Clinic name')) as HTMLInputElement).value).toBe('Changed elsewhere'),
    )
  })

  it('says when the clinic cannot be loaded', async () => {
    renderSettings({ [`GET ${path}`]: () => outcome(404, 'not-found') })

    expect((await screen.findByRole('alert')).textContent).toBe('The clinic could not be loaded. Reload the page to try again.')
  })
})
