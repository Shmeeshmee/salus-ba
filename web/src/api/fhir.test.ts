import { describe, expect, it } from 'vitest'
import { fakeFetch, fhirJSON, outcome } from '../test/fakeFetch'
import { FhirError, newFhirClient, type Organization } from './fhir'

const org: Organization = {
  resourceType: 'Organization',
  id: 'a0000000-0000-4000-8000-00000000000a',
  name: 'Synthetic clinic',
  active: true,
  meta: { versionId: '3', lastUpdated: '2026-11-04T09:30:00Z' },
}

describe('newFhirClient', () => {
  it('reads with the FHIR media type and no CSRF header', async () => {
    const { fetch, requests } = fakeFetch({ [`GET /fhir/R4/Organization/${org.id}`]: () => fhirJSON(org) })

    const got = await newFhirClient(fetch).get<Organization>(`Organization/${org.id}`)

    expect(got).toEqual(org)
    expect(requests[0].headers.get('Accept')).toBe('application/fhir+json')
    expect(requests[0].headers.get('X-Salus-CSRF')).toBeNull()
  })

  it('updates with If-Match, the FHIR media type and the CSRF header', async () => {
    const { fetch, requests } = fakeFetch({
      [`PUT /fhir/R4/Organization/${org.id}`]: () => fhirJSON({ ...org, name: 'Renamed', meta: { ...org.meta, versionId: '4' } }),
    })

    const got = await newFhirClient(fetch).put<Organization>(`Organization/${org.id}`, { ...org, name: 'Renamed' }, '3')

    expect(got.meta.versionId).toBe('4')
    const sent = requests[0]
    expect(sent.headers.get('If-Match')).toBe('W/"3"')
    expect(sent.headers.get('Content-Type')).toBe('application/fhir+json')
    expect(sent.headers.get('X-Salus-CSRF')).toBe('1')
    expect((await sent.json()).name).toBe('Renamed')
  })

  it('turns an OperationOutcome into a FhirError with the status and the issue code', async () => {
    const { fetch } = fakeFetch({ [`GET /fhir/R4/Organization/${org.id}`]: () => outcome(404, 'not-found') })

    const err = await newFhirClient(fetch).get(`Organization/${org.id}`).catch((e: unknown) => e)

    expect(err).toBeInstanceOf(FhirError)
    expect((err as FhirError).status).toBe(404)
    expect((err as FhirError).code).toBe('not-found')
  })

  it('gives a FhirError without a code when the body is not an OperationOutcome', async () => {
    const { fetch } = fakeFetch({ 'GET /fhir/R4/Organization/x': () => new Response('bad gateway', { status: 502 }) })

    const err = await newFhirClient(fetch).get('Organization/x').catch((e: unknown) => e)

    expect((err as FhirError).status).toBe(502)
    expect((err as FhirError).code).toBeUndefined()
  })
})
