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

const cardiology = { system: 'http://snomed.info/sct', code: '394579002', display: 'Cardiology', text: 'Cardiology' }
const dermatology = { system: 'http://snomed.info/sct', code: '394582007', display: 'Dermatology', text: 'Dermatology' }
const specialties = () => Response.json({ specialties: [cardiology, dermatology] })

const bundle: Bundle = {
  resourceType: 'Bundle',
  type: 'searchset',
  total: 2,
  entry: [
    role('a4000000-0000-4000-8000-0000000000a1', 'a3000000-0000-4000-8000-0000000000a1', true, cardiology),
    role('a4000000-0000-4000-8000-0000000000a2', 'a3000000-0000-4000-8000-0000000000a2', false),
    person('a3000000-0000-4000-8000-0000000000a1', 'Cy', 'Example'),
    person('a3000000-0000-4000-8000-0000000000a2', 'Dee', 'Sample'),
  ],
}

function role(id: string, practitionerId: string, active: boolean, specialty?: typeof cardiology): NonNullable<Bundle['entry']>[number] {
  return {
    fullUrl: `https://api.example.test/fhir/R4/PractitionerRole/${id}`,
    resource: {
      resourceType: 'PractitionerRole',
      id,
      meta,
      active,
      practitioner: { reference: `Practitioner/${practitionerId}` },
      organization: { reference: `Organization/${org}` },
      specialty: specialty && [{ coding: [{ system: specialty.system, code: specialty.code, display: specialty.display }], text: specialty.text }],
    },
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
    expect(screen.getByText('Cardiology')).toBeTruthy()
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
      'GET /specialties': specialties,
    })

    const invites = await screen.findByRole('list', { name: 'Open invites' })
    expect(within(invites).getByText('dana@example.test')).toBeTruthy()
    expect(within(invites).getByText(/Dana Invented/)).toBeTruthy()
  })

  it('invites a practitioner and lists the invite', async () => {
    const { requests } = renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'GET /specialties': specialties,
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
      'GET /specialties': specialties,
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
      'GET /specialties': specialties,
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
      'GET /specialties': specialties,
      'POST /invites': () => apiError(status, code),
    })

    fireEvent.change(await screen.findByLabelText('E-mail'), { target: { value: 'dana@example.test' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
  })

  it('gives an admin a specialty editor that saves with If-Match of the role version and the CSRF header', async () => {
    const { requests } = renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'GET /specialties': specialties,
      'PUT /practitioner-roles/a4000000-0000-4000-8000-0000000000a1/specialty': async (r) => {
        const { code } = (await r.json()) as { code: string }
        const version = r.headers.get('If-Match') === 'W/"1"' ? 2 : 3
        return Response.json({ id: 'a4000000-0000-4000-8000-0000000000a1', versionId: version, specialty: code === dermatology.code ? dermatology : cardiology })
      },
    })

    const select = (await screen.findByLabelText('Specialty of Cy Example')) as HTMLSelectElement
    await vi.waitFor(() => expect(select.value).toBe(cardiology.code))
    expect([...select.options].map((o) => o.textContent)).toEqual(['Cardiology', 'Dermatology'])
    fireEvent.change(select, { target: { value: dermatology.code } })
    expect((await screen.findByRole('status')).textContent).toBe('Specialty saved.')
    fireEvent.change(select, { target: { value: cardiology.code } })

    const puts = () => requests.filter((r) => r.method === 'PUT')
    await vi.waitFor(() => expect(puts()).toHaveLength(2))
    expect(puts()[0].headers.get('If-Match')).toBe('W/"1"')
    expect(await puts()[0].json()).toEqual({ code: dermatology.code })
    expect(puts()[1].headers.get('If-Match')).toBe('W/"2"')
    expectCsrfOnWrites(requests)
  })

  it('offers a role without a specialty a choice, not a value', async () => {
    renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'GET /specialties': specialties,
    })

    const select = (await screen.findByLabelText('Specialty of Dee Sample')) as HTMLSelectElement
    await vi.waitFor(() => expect(select.options).toHaveLength(3))
    expect(select.value).toBe('')
    expect(select.options[0].textContent).toBe('Choose a specialty')
    expect(select.options[0].disabled).toBe(true)
  })

  it.each([
    [412, 'version_conflict', 'Someone else changed this practitioner. Reload the page to see the change.'],
    [400, 'invalid_specialty', 'Choose a specialty from the list.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps a specialty save %i %s to a message', async (status, code, message) => {
    renderPractitioners(admin, {
      [search]: () => fhirJSON(bundle),
      'GET /invites': () => Response.json({ invites: [] }),
      'GET /specialties': specialties,
      'PUT /practitioner-roles/a4000000-0000-4000-8000-0000000000a1/specialty': () => apiError(status, code),
    })

    const select = (await screen.findByLabelText('Specialty of Cy Example')) as HTMLSelectElement
    await vi.waitFor(() => expect(select.options.length).toBeGreaterThan(1))
    fireEvent.change(select, { target: { value: dermatology.code } })

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
    expect(select.value).toBe(cardiology.code)
  })
})
