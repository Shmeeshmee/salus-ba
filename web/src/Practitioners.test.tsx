import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Practitioners } from './Practitioners'
import { newApiClient, type Account } from './api/client'
import { newFhirClient, type Bundle } from './api/fhir'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'

afterEach(cleanup)

const org = 'a0000000-0000-4000-8000-00000000000a'
const admin: Account = { id: 'a1000000-0000-4000-8000-000000000001', email: 'admin@example.test', role: 'admin', organizationId: org, mfaEnrolled: true }
const practitioner: Account = { ...admin, id: 'a1000000-0000-4000-8000-000000000002', email: 'cy@example.test', role: 'practitioner' }
const meta = { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' }

const bundle: Bundle = {
  resourceType: 'Bundle',
  type: 'searchset',
  total: 2,
  entry: [
    role('a4000000-0000-4000-8000-0000000000a1', 'a3000000-0000-4000-8000-0000000000a1', true),
    role('a4000000-0000-4000-8000-0000000000a2', 'a3000000-0000-4000-8000-0000000000a2', false),
    person('a3000000-0000-4000-8000-0000000000a1', 'Cy', 'Example'),
    person('a3000000-0000-4000-8000-0000000000a2', 'Dee', 'Sample'),
  ],
}

function role(id: string, practitionerId: string, active: boolean): NonNullable<Bundle['entry']>[number] {
  return {
    fullUrl: `https://api.example.test/fhir/R4/PractitionerRole/${id}`,
    resource: { resourceType: 'PractitionerRole', id, meta, active, practitioner: { reference: `Practitioner/${practitionerId}` }, organization: { reference: `Organization/${org}` } },
    search: { mode: 'match' },
  }
}

function person(id: string, given: string, family: string): NonNullable<Bundle['entry']>[number] {
  return {
    fullUrl: `https://api.example.test/fhir/R4/Practitioner/${id}`,
    resource: { resourceType: 'Practitioner', id, meta, active: true, name: [{ given: [given], family }] },
    search: { mode: 'include' },
  }
}

const openInvite = {
  id: 'a5000000-0000-4000-8000-000000000001',
  email: 'dana@example.test',
  role: 'practitioner',
  givenName: 'Dana',
  familyName: 'Invented',
  expiresAt: '2026-11-07T09:30:00Z',
}

function renderPractitioners(account: Account, routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  render(<Practitioners api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} />)
  return fake
}

const search = 'GET /fhir/R4/PractitionerRole'

describe('Practitioners', () => {
  it('lists the practitioners from the PractitionerRole search with the included Practitioners', async () => {
    const { requests } = renderPractitioners(practitioner, { [search]: () => fhirJSON(bundle) })

    expect(await screen.findByText('Cy Example')).toBeTruthy()
    expect(screen.getByText('Dee Sample')).toBeTruthy()
    expect(screen.getByText('Inactive')).toBeTruthy()
    const url = new URL(requests[0].url)
    expect(url.searchParams.get('organization')).toBe(org)
    expect(url.searchParams.get('_include')).toBe('PractitionerRole:practitioner')
  })

  it('shows no invites or invite form to a practitioner', async () => {
    const { requests } = renderPractitioners(practitioner, { [search]: () => fhirJSON(bundle) })

    await screen.findByText('Cy Example')

    expect(paths(requests)).toEqual([search])
    expect(screen.queryByRole('heading', { name: 'Open invites' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Send invite' })).toBeNull()
  })

  it('says when there are no practitioners yet', async () => {
    renderPractitioners(practitioner, { [search]: () => fhirJSON({ resourceType: 'Bundle', type: 'searchset', total: 0 }) })

    expect(await screen.findByText('No practitioners yet.')).toBeTruthy()
  })

  it('says when the list cannot be loaded', async () => {
    renderPractitioners(practitioner, { [search]: () => outcome(500, 'exception') })

    expect((await screen.findByRole('alert')).textContent).toBe('The practitioners could not be loaded. Reload the page to try again.')
  })

  it('shows the open invites to an admin', async () => {
    renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [openInvite] }),
    })

    const invites = await screen.findByRole('list', { name: 'Open invites' })
    expect(within(invites).getByText('dana@example.test')).toBeTruthy()
    expect(within(invites).getByText(/Dana Invented/)).toBeTruthy()
  })

  it('invites a practitioner and lists the invite', async () => {
    const { requests } = renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'POST /invites': () => Response.json(openInvite, { status: 201 }),
    })

    expect(await screen.findByText('No open invites.')).toBeTruthy()
    fireEvent.change(screen.getByLabelText('E-mail'), { target: { value: 'dana@example.test' } })
    fireEvent.change(screen.getByLabelText('Given name'), { target: { value: 'Dana' } })
    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Invented' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    expect((await screen.findByRole('status')).textContent).toBe('Invite sent.')
    expect(within(screen.getByRole('list', { name: 'Open invites' })).getByText('dana@example.test')).toBeTruthy()
    const post = requests.find((r) => r.method === 'POST')!
    expect(await post.json()).toEqual({ email: 'dana@example.test', role: 'practitioner', givenName: 'Dana', familyName: 'Invented' })
    expectCsrfOnWrites(requests)
    expect((screen.getByLabelText('E-mail') as HTMLInputElement).value).toBe('')
  })

  it('invites an admin with the e-mail only', async () => {
    const adminInvite = { id: 'a5000000-0000-4000-8000-000000000002', email: 'second@example.test', role: 'admin', expiresAt: '2026-11-07T09:30:00Z' }
    const { requests } = renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'POST /invites': () => Response.json(adminInvite, { status: 201 }),
    })

    fireEvent.click(await screen.findByLabelText('Admin'))
    expect(screen.queryByLabelText('Given name')).toBeNull()
    fireEvent.change(screen.getByLabelText('E-mail'), { target: { value: 'second@example.test' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    await screen.findByRole('status')
    const post = requests.find((r) => r.method === 'POST')!
    expect(await post.json()).toEqual({ email: 'second@example.test', role: 'admin' })
  })

  it('replaces an open invite for the same e-mail instead of listing it twice', async () => {
    renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [openInvite] }),
      'POST /invites': () => Response.json({ ...openInvite, id: 'a5000000-0000-4000-8000-000000000003' }, { status: 201 }),
    })

    await screen.findByRole('list', { name: 'Open invites' })
    fireEvent.change(screen.getByLabelText('E-mail'), { target: { value: 'dana@example.test' } })
    fireEvent.change(screen.getByLabelText('Given name'), { target: { value: 'Dana' } })
    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Invented' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    await screen.findByRole('status')
    expect(within(screen.getByRole('list', { name: 'Open invites' })).getAllByText('dana@example.test')).toHaveLength(1)
  })

  it.each([
    [400, 'invalid_email', 'Enter a valid e-mail address.'],
    [400, 'invalid_name', 'Enter the given and family name.'],
    [403, 'forbidden', 'Only a clinic admin can send invites.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'POST /invites': () => apiError(status, code),
    })

    fireEvent.change(await screen.findByLabelText('E-mail'), { target: { value: 'dana@example.test' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
  })
})
