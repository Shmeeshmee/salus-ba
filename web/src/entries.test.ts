import { describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { activeAllergies, allergiesOf, confirmEntry, encounterOf, medicationsOf, readEntries, recordAllergy, recordMedication } from './entries'
import { apiError, fakeFetch, fhirJSON, outcome } from './test/fakeFetch'
import { allergyBundle, clinicAllergy, clinicMedication, medicationBundle, ownAllergy, ownMedication, patientID } from './test/entries'

describe('entries: reading the FHIR resources', () => {
  it('reads an allergy with its term, category, reaction, severity, status and confirmation', () => {
    expect(allergiesOf(allergyBundle(ownAllergy(), clinicAllergy()))).toEqual([
      {
        id: 'a1000000-0000-4000-8000-0000000000a1',
        version: '1',
        text: 'Placeholder substance A',
        category: 'medication',
        reaction: 'Placeholder reaction',
        severity: 'moderate',
        clinicalStatus: 'active',
        confirmed: false,
      },
      { id: 'a1000000-0000-4000-8000-0000000000a2', version: '3', text: 'Placeholder substance B', category: 'food', clinicalStatus: 'active', confirmed: true },
    ])
  })

  it('reads a medication with its dose, timing, as needed, status and confirmation', () => {
    expect(medicationsOf(medicationBundle(ownMedication(), clinicMedication()))).toEqual([
      {
        id: 'b1000000-0000-4000-8000-0000000000b1',
        version: '1',
        text: 'Placeholder medication A',
        status: 'active',
        dose: { value: 500, unit: 'mg' },
        timing: { frequency: 2, period: 1, periodUnit: 'd' },
        asNeeded: false,
        confirmed: false,
      },
      { id: 'b1000000-0000-4000-8000-0000000000b2', version: '2', text: 'Placeholder medication B', status: 'stopped', asNeeded: true, confirmed: true },
    ])
  })

  it('reads an empty bundle as none', () => {
    expect(allergiesOf({ resourceType: 'Bundle', type: 'searchset' })).toEqual([])
    expect(medicationsOf({ resourceType: 'Bundle', type: 'searchset' })).toEqual([])
  })

  it('keeps only the active allergies for the warning', () => {
    const resolved = ownAllergy({ id: 'a1000000-0000-4000-8000-0000000000a3', clinicalStatus: { coding: [{ system: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical', code: 'resolved' }] } })
    const none = ownAllergy({ id: 'a1000000-0000-4000-8000-0000000000a4', clinicalStatus: undefined })
    const active = activeAllergies(allergiesOf(allergyBundle(ownAllergy(), resolved, none)))
    expect(active.map((a) => a.id)).toEqual(['a1000000-0000-4000-8000-0000000000a1'])
  })
})

describe('entries: the requests', () => {
  it("searches the patient's allergies and medications", async () => {
    const fake = fakeFetch({
      'GET /fhir/R4/AllergyIntolerance': () => fhirJSON(allergyBundle(ownAllergy())),
      'GET /fhir/R4/MedicationStatement': () => fhirJSON(medicationBundle(ownMedication())),
    })
    const got = await readEntries(newFhirClient(fake.fetch), patientID)
    expect(got.allergies.map((a) => a.text)).toEqual(['Placeholder substance A'])
    expect(got.medications.map((m) => m.text)).toEqual(['Placeholder medication A'])
    expect(fake.requests.map((r) => new URL(r.url).search).sort()).toEqual([`?patient=${patientID}`, `?patient=${patientID}`])
  })

  // The search answers an empty searchset when the caller sees nothing, never a 404: a 404 is a failure, never none.
  it('fails on a 404', async () => {
    const fake = fakeFetch({
      'GET /fhir/R4/AllergyIntolerance': () => outcome(404, 'not-found'),
      'GET /fhir/R4/MedicationStatement': () => fhirJSON(medicationBundle()),
    })
    await expect(readEntries(newFhirClient(fake.fetch), patientID)).rejects.toThrow()
  })

  it('fails on another error', async () => {
    const fake = fakeFetch({
      'GET /fhir/R4/AllergyIntolerance': () => outcome(500, 'exception'),
      'GET /fhir/R4/MedicationStatement': () => fhirJSON(medicationBundle()),
    })
    await expect(readEntries(newFhirClient(fake.fetch), patientID)).rejects.toThrow()
  })

  it("finds the appointment's encounter, none while there is none", async () => {
    const encounter = 'f1000000-0000-4000-8000-0000000000f1'
    let found = false
    const fake = fakeFetch({
      'GET /fhir/R4/Encounter': () => fhirJSON({ resourceType: 'Bundle', type: 'searchset', entry: found ? [{ fullUrl: `Encounter/${encounter}`, resource: { id: encounter } }] : [] }),
    })
    const fhir = newFhirClient(fake.fetch)
    expect(await encounterOf(fhir, 'e1000000-0000-4000-8000-000000000001')).toBeUndefined()
    found = true
    expect(await encounterOf(fhir, 'e1000000-0000-4000-8000-000000000001')).toBe(encounter)
    expect(new URL(fake.requests[0].url).search).toBe('?appointment=e1000000-0000-4000-8000-000000000001')
  })

  it('confirms with If-Match of the version read and the CSRF header', async () => {
    const fake = fakeFetch({
      'POST /allergies/a1/confirm': () => Response.json({ id: 'a1', versionId: '2' }),
      'POST /medications/b1/confirm': () => Response.json({ id: 'b1', versionId: '4' }),
    })
    const api = newApiClient(fake.fetch)
    expect(await confirmEntry(api, 'allergy', 'a1', '1')).toBe('ok')
    expect(await confirmEntry(api, 'medication', 'b1', '3')).toBe('ok')
    expect(fake.requests.map((r) => [r.headers.get('If-Match'), r.headers.get('X-Salus-CSRF')])).toEqual([
      ['W/"1"', '1'],
      ['W/"3"', '1'],
    ])
  })

  it.each([
    [412, 'version_conflict', 'changed'],
    [403, 'forbidden', 'forbidden'],
    [404, 'not_found', 'failed'],
    [500, 'internal', 'failed'],
  ] as const)('answers a confirmation refused with %i as %s', async (status, code, want) => {
    const fake = fakeFetch({ 'POST /allergies/a1/confirm': () => apiError(status, code) })
    expect(await confirmEntry(newApiClient(fake.fetch), 'allergy', 'a1', '1')).toBe(want)
  })

  it("records an allergy and a medication in the visit's encounter", async () => {
    const fake = fakeFetch({
      'POST /encounters/f1/allergies': () => Response.json({ id: 'a1', versionId: '1' }, { status: 201 }),
      'POST /encounters/f1/medications': () => apiError(400, 'invalid_request'),
    })
    const api = newApiClient(fake.fetch)
    expect(await recordAllergy(api, 'f1', { text: 'Placeholder substance', clinicalStatus: 'active' })).toBe('ok')
    expect(await recordMedication(api, 'f1', { text: 'Placeholder medication', status: 'active', dosage: {} })).toBe('invalid')
    expect(await fake.requests[0].json()).toEqual({ text: 'Placeholder substance', clinicalStatus: 'active' })
    expect(fake.requests.every((r) => r.headers.get('X-Salus-CSRF') === '1')).toBe(true)
  })
})
