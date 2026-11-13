import type { ReactNode } from 'react'
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { PatientHistory } from './History'
import { I18nProvider } from './i18n/I18nProvider'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { apiError, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { patientId } from './test/fixtures'
import { questionnaireFHIR } from './test/questionnaires'

afterEach(cleanup)

const appt = 'e1000000-0000-4000-8000-000000000001'
const own = 'c1000000-0000-4000-8000-0000000000a1'
const shared = 'c2000000-0000-4000-8000-0000000000b1'
const doc = 'd1000000-0000-4000-8000-0000000000a1'
const history = `GET /patients/${patientId}/history`
const summary = `GET /patients/${patientId}/history/summary`

const clinicA = { id: 'a0000000-0000-4000-8000-00000000000a', name: 'Synthetic clinic' }
const clinicB = { id: 'b0000000-0000-4000-8000-00000000000b', name: 'Other synthetic clinic' }

const page = {
  items: [
    { date: '2026-10-01T08:20:00Z', clinic: clinicA, type: 'document', reference: `Composition/${doc}`, encounterId: own, title: 'Placeholder report',
      diagnoses: [{ code: 'J06.9', label: 'Placeholder diagnosis', confirmed: true }, { code: 'K21.0', label: 'Placeholder rule out', confirmed: false }] },
    { date: '2026-10-01T08:00:00Z', clinic: clinicA, type: 'visit', reference: `Encounter/${own}`, encounterId: own },
    { date: '2026-09-01T07:00:00Z', clinic: clinicB, type: 'visit', reference: `Encounter/${shared}`, encounterId: shared, shared: true },
  ],
  next: 'cursor-1',
}

const fullSummary = {
  lastVisit: page.items[1],
  allergies: [{ reference: 'AllergyIntolerance/a1', text: 'Placeholder substance', reaction: 'Placeholder reaction', severity: 'severe', confirmed: true }],
  medications: [{ reference: 'MedicationStatement/m1', text: 'Placeholder medicine',
    dosage: { doseValue: 2, doseUnit: '{tablet}', frequency: 3, period: 1, periodUnit: 'd', asNeeded: true }, confirmed: false }],
  openItems: [{ type: 'questionnaire_pending', reference: 'QuestionnaireResponse/q1' }],
}

function renderHistory(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch(routes)
  renderInLanguage(
    <PatientHistory api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} patientId={patientId} appointmentId={appt} zone="Europe/Sarajevo" />,
    language,
  )
  return fake
}

describe('PatientHistory', () => {
  it('shows the summary and the history, a shared visit with its clinic', async () => {
    const fake = renderHistory({ [history]: () => Response.json(page), [summary]: () => Response.json(fullSummary) })

    expect(await screen.findByText('Placeholder substance')).toBeTruthy()
    expect(screen.getByText('Placeholder reaction, severe')).toBeTruthy()
    expect(screen.getAllByText('Confirmed by a clinician')).toHaveLength(1)
    expect(screen.getByText('Placeholder medicine')).toBeTruthy()
    expect(screen.getByText('2 tablets')).toBeTruthy()
    expect(screen.getByText('3 times a day')).toBeTruthy()
    expect(screen.getByText('As needed')).toBeTruthy()
    expect(screen.getByText('Questionnaire not completed')).toBeTruthy()
    expect(screen.getByText('Placeholder report')).toBeTruthy()
    expect(screen.getByText('Placeholder diagnosis: J06.9')).toBeTruthy()
    expect(screen.getByText('Placeholder rule out: K21.0')).toBeTruthy()
    expect(screen.getAllByText('Not confirmed')).toHaveLength(1)
    expect(screen.getByText('Shared by the patient from Other synthetic clinic')).toBeTruthy()
    const asked = fake.requests.map((r) => new URL(r.url))
    expect(asked.find((u) => u.pathname.endsWith('/summary'))?.searchParams.get('appointment')).toBe(appt)
  })

  it('says when there is nothing to show', async () => {
    renderHistory({ [history]: () => Response.json({ items: [] }), [summary]: () => Response.json({ allergies: [], medications: [], openItems: [] }) })

    expect(await screen.findByText('No history to show')).toBeTruthy()
    expect(screen.getByText('No allergies shown to you. Entries recorded in visits you cannot see are not included; ask the patient.')).toBeTruthy()
    expect(screen.getByText('No current medications shown to you. Entries recorded in visits you cannot see are not included; ask the patient.')).toBeTruthy()
    expect(screen.getByText('No earlier visits shown to you. Visits you cannot see are not included; ask the patient.')).toBeTruthy()
    expect(screen.getByText('Nothing open')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Show more' })).toBeNull()
  })

  it('loads the next page with the cursor', async () => {
    const fake = renderHistory({
      [history]: (r) =>
        new URL(r.url).searchParams.get('cursor') === 'cursor-1'
          ? Response.json({ items: [{ date: '2026-08-01T07:00:00Z', clinic: clinicA, type: 'visit', reference: 'Encounter/c3', encounterId: 'c3' }] })
          : Response.json(page),
      [summary]: () => Response.json(fullSummary),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Show more' }))
    await screen.findByText('1 Aug 2026, 09:00')
    expect(screen.queryByRole('button', { name: 'Show more' })).toBeNull()
    expect(fake.requests.filter((r) => new URL(r.url).pathname === `/patients/${patientId}/history`)).toHaveLength(2)
  })

  it('opens a signed document in place, as text', async () => {
    renderHistory({
      [history]: () => Response.json(page),
      [summary]: () => Response.json(fullSummary),
      [`GET /fhir/R4/Composition/${doc}`]: () =>
        fhirJSON({ resourceType: 'Composition', section: [{ title: 'Findings', text: { div: '<div xmlns="http://www.w3.org/1999/xhtml"><p>Placeholder findings</p></div>' } }] }),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Open' }))
    expect(await screen.findByText('Placeholder findings')).toBeTruthy()
    expect(screen.getByText('Findings')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByText('Placeholder findings')).toBeNull()
  })

  it('says when the history cannot be loaded', async () => {
    let fail = true
    renderHistory({ [history]: () => (fail ? apiError(500, 'internal') : Response.json(page)), [summary]: () => Response.json(fullSummary) })

    expect(await screen.findByText('The history could not be loaded.')).toBeTruthy()
    fail = false
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('Placeholder report')).toBeTruthy()
  })

  it('says a section could not be loaded, never that none is recorded', async () => {
    renderHistory({
      [history]: () => Response.json(page),
      [summary]: () => Response.json({ allergies: [], medications: [], openItems: [], allergiesUnavailable: true, medicationsUnavailable: true }),
    })

    expect(await screen.findByText('Allergies could not be loaded.')).toBeTruthy()
    expect(screen.getByText('Current medications could not be loaded.')).toBeTruthy()
    expect(screen.queryByText('No allergies shown to you. Entries recorded in visits you cannot see are not included; ask the patient.')).toBeNull()
    expect(screen.queryByText('No current medications shown to you. Entries recorded in visits you cannot see are not included; ask the patient.')).toBeNull()
  })

  it('leaves out a dose or a timing in a unit it does not know', async () => {
    const odd = { ...fullSummary, medications: [{ reference: 'MedicationStatement/m2', text: 'Placeholder medicine',
      dosage: { doseValue: 2, doseUnit: 'cup', frequency: 3, period: 1, periodUnit: 'year' }, confirmed: false }] }
    renderHistory({ [history]: () => Response.json(page), [summary]: () => Response.json(odd) })

    const name = await screen.findByText('Placeholder medicine')
    expect(name.closest('li')?.textContent).toBe('Placeholder medicine')
  })

  it('reads the summary again when asked, and after a failure on request', async () => {
    let fail = true
    const fake = renderHistory({ [history]: () => Response.json(page), [summary]: () => (fail ? apiError(500, 'internal') : Response.json(fullSummary)) })

    expect(await screen.findByText('The summary could not be loaded.')).toBeTruthy()
    fail = false
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('Placeholder substance')).toBeTruthy()
    const summaries = () => fake.requests.filter((r) => new URL(r.url).pathname.endsWith('/summary')).length
    const histories = () => fake.requests.filter((r) => new URL(r.url).pathname === `/patients/${patientId}/history`).length
    expect(summaries()).toBe(2)
    expect(histories()).toBe(1)
  })

  it('reads the summary again when its refresh changes, not the history', async () => {
    const fake = fakeFetch({ [history]: () => Response.json(page), [summary]: () => Response.json(fullSummary) })
    const api = newApiClient(fake.fetch)
    const fhir = newFhirClient(fake.fetch)
    const view = (refresh: string) => <PatientHistory api={api} fhir={fhir} patientId={patientId} appointmentId={appt} zone="Europe/Sarajevo" refresh={refresh} />
    const { rerender } = renderInLanguage(view('1'), 'en')
    expect(await screen.findByText('Placeholder substance')).toBeTruthy()
    rerender(inEnglish(view('2')))
    await screen.findByText('Placeholder substance')
    await waitFor(() => expect(fake.requests.filter((r) => new URL(r.url).pathname.endsWith('/summary'))).toHaveLength(2))
    expect(fake.requests.filter((r) => new URL(r.url).pathname === `/patients/${patientId}/history`)).toHaveLength(1)
  })

  it('keeps the history when the summary cannot be loaded', async () => {
    renderHistory({ [history]: () => Response.json(page), [summary]: () => apiError(500, 'internal') })

    expect(await screen.findByText('The summary could not be loaded.')).toBeTruthy()
    expect(screen.getByText('Placeholder report')).toBeTruthy()
    expect(screen.queryByText('No allergies shown to you. Entries recorded in visits you cannot see are not included; ask the patient.')).toBeNull()
  })

  it('marks a shared visit as shared when its clinic has no name', async () => {
    const nameless = { ...page.items[2], clinic: { id: clinicB.id, name: '' } }
    renderHistory({ [history]: () => Response.json({ items: [nameless] }), [summary]: () => Response.json(fullSummary) })

    expect(await screen.findByText('Shared by the patient')).toBeTruthy()
  })

  it('opens a questionnaire response with its questions', async () => {
    const response = { date: '2026-10-01T07:00:00Z', clinic: clinicA, type: 'questionnaire_response', reference: 'QuestionnaireResponse/r1',
      encounterId: own, title: 'Placeholder intake', questionnaire: 'q1' }
    renderHistory({
      [history]: () => Response.json({ items: [response] }),
      [summary]: () => Response.json(fullSummary),
      'GET /fhir/R4/QuestionnaireResponse/r1': () =>
        fhirJSON({ resourceType: 'QuestionnaireResponse', id: 'r1', status: 'completed', questionnaire: 'https://salus.invalid/fhir/Questionnaire/s1|1',
          item: [{ linkId: 'smoker', answer: [{ valueBoolean: false }] }] }),
      'GET /fhir/R4/Questionnaire/q1': () => fhirJSON(questionnaireFHIR()),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Open' }))
    expect(await screen.findByText('Placeholder yes or no')).toBeTruthy()
    expect(screen.getByText('No')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByText('Placeholder yes or no')).toBeNull()
  })

  it('speaks the staff languages', async () => {
    renderHistory({ [history]: () => Response.json(page), [summary]: () => Response.json(fullSummary) }, 'sr-Cyrl')

    expect(await screen.findByText('Историја пацијента')).toBeTruthy()
    expect(screen.getByText('Пацијент је подијелио из установе Other synthetic clinic')).toBeTruthy()
  })
})

// inEnglish is ui as renderInLanguage wraps it, for a rerender.
function inEnglish(ui: ReactNode) {
  return (
    <I18nProvider initial="en" browser={['en']}>
      {ui}
    </I18nProvider>
  )
}
