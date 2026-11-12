import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { AllergyWarning, EntriesPanel } from './AllergiesPanel'
import { allergiesOf, usePatientEntries, type Allergy, type PatientEntries } from './entries'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, paths, type Route } from './test/fakeFetch'
import { allergyBundle, clinicAllergy, clinicMedication, medicationBundle, ownAllergy, ownMedication, patientID } from './test/entries'

afterEach(cleanup)

const appointment = 'e1000000-0000-4000-8000-000000000001'
const encounter = 'f1000000-0000-4000-8000-0000000000f1'
const appointmentResource = {
  resourceType: 'Appointment',
  id: appointment,
  meta: { versionId: '1', lastUpdated: '2026-11-03T09:00:00Z' },
  status: 'booked',
  start: '2026-11-03T09:00:00Z',
  end: '2026-11-03T09:30:00Z',
  participant: [{ status: 'accepted', actor: { reference: `Patient/${patientID}`, display: 'Ena Example' } }],
}
const encounterFound = { resourceType: 'Bundle', type: 'searchset', entry: [{ fullUrl: `Encounter/${encounter}`, resource: { resourceType: 'Encounter', id: encounter } }] }

// A synthetic term of the drug category, its labels placeholders.
const term = {
  id: 'c3000000-0000-4000-8000-000000000001',
  category: 'drug',
  reviewStatus: 'approved',
  codes: { wikidata: 'Q1' },
  labels: [
    { language: 'en', preferred: 'Placeholder drug term', synonyms: [] },
    { language: 'la', preferred: 'Placeholder latinum', synonyms: [] },
  ],
}

function Panel({ fake, refresh }: { fake: ReturnType<typeof fakeFetch>; refresh?: number }) {
  const [{ fhir, api }] = useState(() => ({ fhir: newFhirClient(fake.fetch), api: newApiClient(fake.fetch) }))
  const entries = usePatientEntries(fhir, appointment, refresh)
  return (
    <>
      <AllergyWarning entries={entries} />
      <EntriesPanel api={api} fhir={fhir} appointmentId={appointment} entries={entries} />
    </>
  )
}

type Setup = { routes?: Record<string, Route>; allergies?: () => Response; medications?: () => Response; language?: Language; refresh?: number }

function renderPanel({ routes = {}, allergies, medications, language = 'en', refresh }: Setup = {}) {
  const fake = fakeFetch({
    [`GET /fhir/R4/Appointment/${appointment}`]: () => fhirJSON(appointmentResource),
    'GET /fhir/R4/AllergyIntolerance': allergies ?? (() => fhirJSON(allergyBundle(ownAllergy(), clinicAllergy()))),
    'GET /fhir/R4/MedicationStatement': medications ?? (() => fhirJSON(medicationBundle(ownMedication(), clinicMedication()))),
    'GET /fhir/R4/Encounter': () => fhirJSON(encounterFound),
    'GET /terminology/search': () => Response.json({ terms: [term] }),
    ...routes,
  })
  renderInLanguage(<Panel fake={fake} refresh={refresh} />, language)
  return fake
}

const reads = (requests: Request[], what: string) => paths(requests).filter((p) => p === `GET /fhir/R4/${what}`).length
const allergyList = () => screen.getByRole('list', { name: 'Allergies' })
const medicationList = () => screen.getByRole('list', { name: 'Medications' })

describe('AllergyWarning', () => {
  const list = (...a: Parameters<typeof allergyBundle>): Allergy[] => allergiesOf(allergyBundle(...a))
  const loaded = (allergies: Allergy[]): PatientEntries => ({ entries: { allergies, medications: [] }, failed: false, reload() {} })

  it('lists each active allergy with its reaction and severity', () => {
    renderInLanguage(<AllergyWarning entries={loaded(list(ownAllergy(), clinicAllergy()))} />, 'en')
    const warning = screen.getByRole('region', { name: 'Active allergies' })
    const items = within(warning).getAllByRole('listitem').map((li) => li.textContent)
    expect(items).toEqual(['Placeholder substance APlaceholder reaction (moderate)', 'Placeholder substance B'])
  })

  it('is in the UI language', () => {
    renderInLanguage(<AllergyWarning entries={loaded(list(ownAllergy()))} />, 'sr-Cyrl')
    const warning = screen.getByRole('region', { name: 'Активне алергије' })
    expect(within(warning).getByText('Placeholder reaction (умјерена)')).toBeTruthy()
  })

  it('is not shown without an active allergy', () => {
    const resolved = ownAllergy({ clinicalStatus: { coding: [{ system: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical', code: 'resolved' }] } })
    renderInLanguage(<AllergyWarning entries={loaded(list(resolved))} />, 'en')
    expect(screen.queryByRole('region')).toBeNull()
  })

  // A list that is not read yet, or could not be read, never looks like a patient without allergies.
  it('says the allergies are being loaded until they are', () => {
    renderInLanguage(<AllergyWarning entries={{ failed: false, reload() {} }} />, 'en')
    expect(screen.getByRole('status').textContent).toBe('Loading allergies…')
  })

  it('says the allergies could not be loaded, and loads them again on Try again', () => {
    const reload = vi.fn()
    renderInLanguage(<AllergyWarning entries={{ failed: true, reload }} />, 'en')
    expect(screen.getByRole('alert').textContent).toContain('Allergies could not be loaded. Do not assume the patient has none.')
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(reload).toHaveBeenCalledOnce()
  })

  it('keeps the allergies last read beside the notice when a later read fails', () => {
    renderInLanguage(<AllergyWarning entries={{ ...loaded(list(ownAllergy())), failed: true }} />, 'en')
    expect(screen.getByRole('alert')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'Active allergies' })).getByText('Placeholder substance A')).toBeTruthy()
  })

  it('is in the UI language when the allergies could not be loaded', () => {
    renderInLanguage(<AllergyWarning entries={{ failed: true, reload() {} }} />, 'sr-Cyrl')
    expect(screen.getByRole('alert').textContent).toContain('Алергије није могуће учитати.')
    expect(screen.getByRole('button', { name: 'Покушајте поново' })).toBeTruthy()
  })
})

describe("EntriesPanel: the patient's entries", () => {
  it("reads the appointment's patient and lists their allergies and medications", async () => {
    const fake = renderPanel()

    await screen.findByRole('list', { name: 'Allergies' })
    const allergies = within(allergyList()).getAllByRole('listitem')
    expect(allergies[0].textContent).toContain('Medication')
    expect(allergies[0].textContent).toContain('Placeholder reaction (moderate)')
    expect(within(allergies[0]).getByText('Not confirmed')).toBeTruthy()
    expect(within(allergies[1]).getByText('Confirmed')).toBeTruthy()
    expect(within(allergies[1]).queryByRole('button')).toBeNull()

    const medications = within(medicationList()).getAllByRole('listitem')
    expect(medications[0].textContent).toContain('500 mg')
    expect(medications[0].textContent).toContain('2 times a day')
    expect(medications[1].textContent).toContain('As needed')
    expect(medications[1].textContent).toContain('Stopped')
    expect(within(medications[1]).getByText('Confirmed')).toBeTruthy()

    const searched = fake.requests.filter((r) => new URL(r.url).pathname.match(/AllergyIntolerance|MedicationStatement/))
    expect(searched.map((r) => new URL(r.url).searchParams.get('patient'))).toEqual([patientID, patientID])
  })

  it('says when there are none', async () => {
    renderPanel({ allergies: () => fhirJSON(allergyBundle()), medications: () => fhirJSON(medicationBundle()) })
    expect(await screen.findByText('No allergies recorded.')).toBeTruthy()
    expect(screen.getByText('No medications recorded.')).toBeTruthy()
  })

  it('says when they cannot be loaded', async () => {
    renderPanel({ allergies: () => fhirJSON({}, 500) })
    // The panel says so, and the warning above it too (AllergyWarning).
    const alerts = (await screen.findAllByRole('alert')).map((a) => a.textContent)
    expect(alerts).toEqual(['Allergies could not be loaded. Do not assume the patient has none.Try again', 'Allergies and medications could not be loaded.'])
  })
})

describe('usePatientEntries: during the visit', () => {
  it('reads the allergies and medications again while the page is open, so one added meanwhile is shown', async () => {
    let allergies = allergyBundle(ownAllergy())
    const fake = renderPanel({ allergies: () => fhirJSON(allergies), refresh: 20 })
    expect(await within(await screen.findByRole('list', { name: 'Allergies' })).findByText('Placeholder substance A')).toBeTruthy()
    allergies = allergyBundle(ownAllergy(), clinicAllergy())
    await waitFor(() => expect(within(allergyList()).getAllByRole('listitem')).toHaveLength(2))
    expect(reads(fake.requests, 'AllergyIntolerance')).toBeGreaterThan(1)
  })
})

describe('EntriesPanel: confirming an entry', () => {
  it('confirms with If-Match of the version read and reads the list again', async () => {
    const fake = renderPanel({
      routes: {
        'POST /allergies/a1000000-0000-4000-8000-0000000000a1/confirm': () => Response.json({ id: 'a1', versionId: '2' }),
        'POST /medications/b1000000-0000-4000-8000-0000000000b1/confirm': () => Response.json({ id: 'b1', versionId: '2' }),
      },
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm Placeholder substance A' }))
    await waitFor(() => expect(reads(fake.requests, 'AllergyIntolerance')).toBe(2))
    fireEvent.click(screen.getByRole('button', { name: 'Confirm Placeholder medication A' }))
    await waitFor(() => expect(reads(fake.requests, 'MedicationStatement')).toBe(3))

    const posts = fake.requests.filter((r) => r.method === 'POST')
    expect(posts.map((r) => r.headers.get('If-Match'))).toEqual(['W/"1"', 'W/"1"'])
    expectCsrfOnWrites(fake.requests)
  })

  it('reads the list again and says so when the entry changed since it was read', async () => {
    const fake = renderPanel({ routes: { 'POST /allergies/a1000000-0000-4000-8000-0000000000a1/confirm': () => apiError(412, 'version_conflict') } })
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm Placeholder substance A' }))
    expect((await screen.findByRole('alert')).textContent).toBe('This entry changed since it was read. The list is updated: check it and confirm again.')
    await waitFor(() => expect(reads(fake.requests, 'AllergyIntolerance')).toBe(2))
  })

  it('says when the practitioner may not confirm it', async () => {
    renderPanel({ routes: { 'POST /allergies/a1000000-0000-4000-8000-0000000000a1/confirm': () => apiError(403, 'forbidden') } })
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm Placeholder substance A' }))
    expect((await screen.findByRole('alert')).textContent).toBe('You cannot confirm this entry.')
  })
})

describe('EntriesPanel: recording an allergy', () => {
  const post = `POST /encounters/${encounter}/allergies`

  async function openForm(setup: Setup = {}) {
    const fake = renderPanel(setup)
    fireEvent.click(await screen.findByRole('button', { name: 'Add an allergy' }))
    return fake
  }

  it('records a term from the terminology with its label in the UI language, and reads the list again', async () => {
    const fake = await openForm({ routes: { [post]: () => Response.json({ id: 'a9', versionId: '1' }, { status: 201 }) } })
    const form = screen.getByRole('form', { name: 'Add an allergy' })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Substance' }), { target: { value: 'Placeh' } })
    fireEvent.click(await within(form).findByRole('option', { name: 'Placeholder drug term' }))
    expect((within(form).getByRole('combobox', { name: 'Substance' }) as HTMLInputElement).value).toBe('Placeholder drug term')
    fireEvent.change(within(form).getByRole('combobox', { name: 'Category' }), { target: { value: 'medication' } })
    fireEvent.change(within(form).getByRole('textbox', { name: 'Reaction' }), { target: { value: 'Placeholder rash' } })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Severity' }), { target: { value: 'severe' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(paths(fake.requests)).toContain(post))
    const body = await fake.requests.find((r) => r.method === 'POST')!.json()
    expect(body).toEqual({ termId: term.id, text: 'Placeholder drug term', category: 'medication', reaction: 'Placeholder rash', severity: 'severe', clinicalStatus: 'active' })
    const search = new URL(fake.requests.find((r) => r.url.includes('/terminology/search'))!.url).searchParams
    expect([search.get('q'), search.get('category')]).toEqual(['Placeh', 'drug'])
    await waitFor(() => expect(reads(fake.requests, 'AllergyIntolerance')).toBe(2))
    expect(screen.queryByRole('form', { name: 'Add an allergy' })).toBeNull()
    expectCsrfOnWrites(fake.requests)
  })

  it('records a substance typed by hand as text only, with no severity without a reaction', async () => {
    const fake = await openForm({ routes: { [post]: () => Response.json({ id: 'a9', versionId: '1' }, { status: 201 }) } })
    const form = screen.getByRole('form', { name: 'Add an allergy' })
    expect((within(form).getByRole('combobox', { name: 'Severity' }) as HTMLSelectElement).disabled).toBe(true)
    fireEvent.change(within(form).getByRole('combobox', { name: 'Substance' }), { target: { value: 'Placeholder typed substance' } })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Status' }), { target: { value: 'inactive' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(paths(fake.requests)).toContain(post))
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({ text: 'Placeholder typed substance', clinicalStatus: 'inactive' })
  })

  it('a term picked and then changed by hand is text only', async () => {
    const fake = await openForm({ routes: { [post]: () => Response.json({ id: 'a9', versionId: '1' }, { status: 201 }) } })
    const form = screen.getByRole('form', { name: 'Add an allergy' })
    const box = within(form).getByRole('combobox', { name: 'Substance' })
    fireEvent.change(box, { target: { value: 'Placeh' } })
    fireEvent.click(await within(form).findByRole('option', { name: 'Placeholder drug term' }))
    fireEvent.change(box, { target: { value: 'Placeholder drug term, other' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(paths(fake.requests)).toContain(post))
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({ text: 'Placeholder drug term, other', clinicalStatus: 'active' })
  })

  it.each([
    [() => apiError(400, 'invalid_request'), 'Check the entry: a value is missing or not allowed.'],
    [() => apiError(403, 'forbidden'), 'You cannot record entries in this consultation.'],
    [() => apiError(500, 'internal'), 'The entry could not be saved. Try again.'],
  ])('says when the entry is refused', async (answer, message) => {
    await openForm({ routes: { [post]: answer } })
    const form = screen.getByRole('form', { name: 'Add an allergy' })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Substance' }), { target: { value: 'Placeholder typed substance' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect((await within(form).findByRole('alert')).textContent).toBe(message)
  })

  it('asks to join first while the consultation has no encounter', async () => {
    const fake = await openForm({ routes: { 'GET /fhir/R4/Encounter': () => fhirJSON({ resourceType: 'Bundle', type: 'searchset' }) } })
    const form = screen.getByRole('form', { name: 'Add an allergy' })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Substance' }), { target: { value: 'Placeholder typed substance' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect((await within(form).findByRole('alert')).textContent).toBe('Join the consultation before you record an entry.')
    expect(fake.requests.filter((r) => r.method === 'POST')).toEqual([])
  })
})

describe('EntriesPanel: recording a medication', () => {
  const post = `POST /encounters/${encounter}/medications`

  async function openForm() {
    const fake = renderPanel({ routes: { [post]: () => Response.json({ id: 'b9', versionId: '1' }, { status: 201 }) } })
    fireEvent.click(await screen.findByRole('button', { name: 'Add a medication' }))
    const form = screen.getByRole('form', { name: 'Add a medication' })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Medication' }), { target: { value: 'Placeholder typed medication' } })
    return { fake, form }
  }

  it('records the dose, how often and the status, the units as their UCUM codes', async () => {
    const { fake, form } = await openForm()
    expect([...(within(form).getByRole('combobox', { name: 'Unit' }) as HTMLSelectElement).options].map((o) => o.textContent)).toEqual([
      'No unit',
      'mg',
      'g',
      'µg',
      'mL',
      'IU',
      'Tablet',
      'Capsule',
      'Drop',
      'Puff',
      'Patch',
    ])
    fireEvent.change(within(form).getByRole('spinbutton', { name: 'Dose' }), { target: { value: '1' } })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Unit' }), { target: { value: '{tablet}' } })
    fireEvent.change(within(form).getByRole('spinbutton', { name: 'Times' }), { target: { value: '3' } })
    fireEvent.change(within(form).getByRole('spinbutton', { name: 'Every' }), { target: { value: '1' } })
    fireEvent.change(within(form).getByRole('combobox', { name: 'Unit of time' }), { target: { value: 'wk' } })
    fireEvent.click(within(form).getByRole('checkbox', { name: 'As needed' }))
    fireEvent.change(within(form).getByRole('combobox', { name: 'Status' }), { target: { value: 'stopped' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(paths(fake.requests)).toContain(post))
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({
      text: 'Placeholder typed medication',
      status: 'stopped',
      dosage: { doseValue: 1, doseUnit: '{tablet}', frequency: 3, period: 1, periodUnit: 'wk', asNeeded: true },
    })
    await waitFor(() => expect(reads(fake.requests, 'MedicationStatement')).toBe(2))
  })

  it('records a medication with no dosage', async () => {
    const { fake, form } = await openForm()
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(paths(fake.requests)).toContain(post))
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({ text: 'Placeholder typed medication', status: 'active', dosage: {} })
  })

  it('asks for the dose and its unit together', async () => {
    const { fake, form } = await openForm()
    fireEvent.change(within(form).getByRole('spinbutton', { name: 'Dose' }), { target: { value: '5' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect(within(form).getByRole('alert').textContent).toBe('Enter both the dose and its unit, or neither.')
    expect(fake.requests.filter((r) => r.method === 'POST')).toEqual([])
  })

  it('asks for how many times, every how many and the unit of time together', async () => {
    const { fake, form } = await openForm()
    fireEvent.change(within(form).getByRole('spinbutton', { name: 'Times' }), { target: { value: '2' } })
    fireEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect(within(form).getByRole('alert').textContent).toBe('Fill in Times, Every and Unit of time, or leave all three empty.')
    expect(fake.requests.filter((r) => r.method === 'POST')).toEqual([])
  })
})
