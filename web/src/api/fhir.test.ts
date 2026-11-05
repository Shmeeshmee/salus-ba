import { describe, expect, it } from 'vitest'
import { fakeFetch, fhirJSON, outcome } from '../test/fakeFetch'
import { FhirError, listedURL, newFhirClient, participant, settingsOf, timeZoneURL, withSettings, type Appointment, type Organization } from './fhir'

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

describe('clinic settings in the Organization', () => {
  const withExtensions: Organization = {
    ...org,
    extension: [
      { url: listedURL, valueBoolean: true },
      { url: timeZoneURL, valueCode: 'Europe/Berlin' },
      { url: 'https://example.test/other', valueCode: 'kept' },
    ],
  }

  it('reads listed and the time zone from their extensions', () => {
    expect(settingsOf(withExtensions)).toEqual({ name: 'Synthetic clinic', listed: true, timeZone: 'Europe/Berlin' })
  })

  it('reads an Organization without them as not listed, in the default zone', () => {
    expect(settingsOf(org)).toEqual({ name: 'Synthetic clinic', listed: false, timeZone: 'Europe/Sarajevo' })
  })

  it('writes the settings into the resource and keeps other extensions', () => {
    const out = withSettings(withExtensions, { name: 'Renamed', listed: false, timeZone: 'Europe/Sarajevo' })

    expect(out.name).toBe('Renamed')
    expect(out.extension).toEqual([
      { url: 'https://example.test/other', valueCode: 'kept' },
      { url: listedURL, valueBoolean: false },
      { url: timeZoneURL, valueCode: 'Europe/Sarajevo' },
    ])
  })
})

describe('appointment participants', () => {
  const appt: Appointment = {
    resourceType: 'Appointment',
    id: 'e1000000-0000-4000-8000-0000000000a1',
    meta: { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' },
    status: 'booked',
    start: '2026-11-09T08:00:00Z',
    end: '2026-11-09T08:30:00Z',
    participant: [
      { status: 'accepted', actor: { reference: 'Patient/aa000000-0000-4000-8000-0000000000aa', display: 'Ada Example' } },
      { status: 'accepted', actor: { reference: 'Practitioner/a3000000-0000-4000-8000-0000000000a1' } },
    ],
  }

  it('finds a participant by resource type, with its id and display', () => {
    expect(participant(appt, 'Patient')).toEqual({ id: 'aa000000-0000-4000-8000-0000000000aa', display: 'Ada Example' })
    expect(participant(appt, 'Practitioner')).toEqual({ id: 'a3000000-0000-4000-8000-0000000000a1', display: undefined })
    expect(participant(appt, 'PractitionerRole')).toBeUndefined()
  })
})
