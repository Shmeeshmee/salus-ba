import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import type { components } from './api/schema'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { VisitLabResults } from './LabResults'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'
import { org, patientId } from './test/fixtures'

afterEach(cleanup)

type Report = components['schemas']['DiagnosticReport']

const visit = 'e1000000-0000-4000-8000-000000000001'
const first = 'd5000000-0000-4000-8000-000000000001'
const zone = 'Europe/Sarajevo'

// Synthetic lab results: the names, values and ranges are placeholders, not clinical guidance.
function report(patch: Partial<Report> = {}): Report {
  return {
    id: first,
    organizationId: org,
    patientId,
    appointmentId: visit,
    status: 'preliminary',
    title: 'Placeholder panel',
    effectiveDate: '2026-11-02',
    results: [
      { test: 'Placeholder test A', value: 5.4, unit: 'mmol/L', referenceRange: '3.9-5.6', interpretation: 'N' },
      { test: 'Placeholder test B', valueText: 'Placeholder text value', interpretation: 'A' },
    ],
    hasPdf: false,
    createdAt: '2026-11-02T08:00:00Z',
    versionId: 1,
    ...patch,
  }
}

type Setup = { reports?: Report[]; routes?: Record<string, Route>; practitioner?: boolean; language?: Language }

function renderResults({ reports = [], routes = {}, practitioner = true, language = 'en' }: Setup = {}) {
  const fake = fakeFetch({ [`GET /clinic/appointments/${visit}/diagnostic-reports`]: () => Response.json(reports), ...routes })
  renderInLanguage(<VisitLabResults api={newApiClient(fake.fetch)} appointmentId={visit} zone={zone} practitioner={practitioner} />, language)
  return fake
}

const section = () => within(screen.getByRole('region', { name: 'Lab results' }))

describe('VisitLabResults (clinic-cms LB2)', () => {
  it("lists the visit's results with their values, ranges, flags, PDF and whether the patient sees them", async () => {
    renderResults({
      reports: [
        report(),
        report({ id: 'd5000000-0000-4000-8000-000000000002', title: 'Placeholder released panel', status: 'final', releasedAt: '2026-11-03T10:00:00Z', hasPdf: true, results: [] }),
      ],
    })
    const s = within(await screen.findByRole('region', { name: 'Lab results' }))
    const [draft, released] = s.getAllByRole('article')
    const d = within(draft)
    expect(d.getByRole('heading', { name: 'Placeholder panel' })).toBeTruthy()
    expect(d.getByText('Not released: the patient does not see it yet')).toBeTruthy()
    expect(d.getByText(/Sample taken on 2\. 11\. 2026|Sample taken on .*2026/)).toBeTruthy()
    const rows = d.getAllByRole('row').map((r) => r.textContent)
    expect(rows).toEqual(['TestValueReference rangeFlag', 'Placeholder test A5.4 mmol/L3.9-5.6Normal', 'Placeholder test BPlaceholder text valueAbnormal'])
    expect(d.queryByRole('link', { name: 'PDF' })).toBeNull()
    const r = within(released)
    expect(r.getByText(/Released to the patient on/)).toBeTruthy()
    expect(r.getByRole('link', { name: 'PDF' }).getAttribute('href')).toMatch(/\/clinic\/diagnostic-reports\/d5000000-0000-4000-8000-000000000002\/pdf$/)
    expect(r.queryByRole('button', { name: 'Release to the patient' })).toBeNull()
  })

  it('says when the visit has no results, and when they cannot be read', async () => {
    renderResults()
    expect(await section().findByText('No lab results on this visit.')).toBeTruthy()
    cleanup()
    renderResults({ routes: { [`GET /clinic/appointments/${visit}/diagnostic-reports`]: () => apiError(500, 'internal') } })
    expect(await screen.findByText('The lab results could not be loaded.')).toBeTruthy()
  })

  it('attaches typed-in values and the PDF at the new version', async () => {
    let body: unknown
    let pdf: Request | undefined
    const created = report({ results: [{ test: 'Placeholder test A', value: 5.4, unit: 'mmol/L', referenceRange: '3.9-5.6', interpretation: 'H' }, { test: 'Placeholder test B', valueText: 'Placeholder text value' }] })
    const fake = renderResults({
      routes: {
        [`POST /clinic/appointments/${visit}/diagnostic-reports`]: async (r) => {
          body = await r.json()
          return Response.json(created, { status: 201 })
        },
        [`PUT /clinic/diagnostic-reports/${first}/pdf`]: (r) => {
          pdf = r
          return Response.json({ ...created, hasPdf: true, versionId: 2 })
        },
      },
    })
    fireEvent.click(await section().findByRole('button', { name: 'Add a lab result' }))
    const form = within(screen.getByRole('form', { name: 'New lab result' }))
    fireEvent.change(form.getByLabelText('Title'), { target: { value: 'Placeholder panel' } })
    fireEvent.change(form.getByLabelText('Sample taken on'), { target: { value: '2026-11-02' } })
    fireEvent.change(form.getByLabelText('Test'), { target: { value: 'Placeholder test A' } })
    fireEvent.change(form.getByLabelText('Value'), { target: { value: '5,4' } })
    fireEvent.change(form.getByLabelText('Unit'), { target: { value: 'mmol/L' } })
    fireEvent.change(form.getByLabelText('Reference range'), { target: { value: '3.9-5.6' } })
    fireEvent.change(form.getByLabelText('Flag'), { target: { value: 'H' } })
    fireEvent.click(form.getByRole('button', { name: 'Add a value' }))
    fireEvent.change(form.getAllByLabelText('Test')[1], { target: { value: 'Placeholder test B' } })
    fireEvent.change(form.getAllByLabelText('Value')[1], { target: { value: 'Placeholder text value' } })
    const file = new File([new Uint8Array([0x25, 0x50, 0x44, 0x46])], 'placeholder.pdf', { type: 'application/pdf' })
    fireEvent.change(form.getByLabelText('The lab’s PDF (optional)'), { target: { files: [file] } })
    fireEvent.click(form.getByRole('button', { name: 'Attach' }))

    expect(await section().findByText('Lab result attached. The patient sees it once a practitioner releases it.')).toBeTruthy()
    expect(body).toEqual({
      title: 'Placeholder panel',
      effectiveDate: '2026-11-02',
      results: [
        { test: 'Placeholder test A', value: 5.4, unit: 'mmol/L', referenceRange: '3.9-5.6', interpretation: 'H' },
        { test: 'Placeholder test B', valueText: 'Placeholder text value' },
      ],
    })
    expect(pdf?.headers.get('If-Match')).toBe('W/"1"')
    expect(pdf?.headers.get('Content-Type')).toBe('application/pdf')
    expect(new Uint8Array(await pdf!.arrayBuffer())).toEqual(new Uint8Array([0x25, 0x50, 0x44, 0x46]))
    expect(section().getAllByRole('article')).toHaveLength(1)
    expect(section().getByRole('link', { name: 'PDF' })).toBeTruthy()
    expectCsrfOnWrites(fake.requests)
  })

  it('refuses a unit without a number before asking the server, and a form without a title', async () => {
    const fake = renderResults()
    fireEvent.click(await section().findByRole('button', { name: 'Add a lab result' }))
    const form = within(screen.getByRole('form', { name: 'New lab result' }))
    fireEvent.change(form.getByLabelText('Title'), { target: { value: 'Placeholder panel' } })
    fireEvent.change(form.getByLabelText('Test'), { target: { value: 'Placeholder test A' } })
    fireEvent.change(form.getByLabelText('Value'), { target: { value: 'Placeholder text value' } })
    fireEvent.change(form.getByLabelText('Unit'), { target: { value: 'mmol/L' } })
    fireEvent.click(form.getByRole('button', { name: 'Attach' }))
    expect(await form.findByText('A unit goes with a number. Remove the unit or type the value as a number.')).toBeTruthy()
    expect(paths(fake.requests).filter((p) => p.startsWith('POST'))).toEqual([])
  })

  it("says why the server refused the result: no consent or the visit is not the caller's", async () => {
    renderResults({ routes: { [`POST /clinic/appointments/${visit}/diagnostic-reports`]: () => apiError(404, 'not_found') } })
    fireEvent.click(await section().findByRole('button', { name: 'Add a lab result' }))
    const form = within(screen.getByRole('form', { name: 'New lab result' }))
    fireEvent.change(form.getByLabelText('Title'), { target: { value: 'Placeholder panel' } })
    fireEvent.click(form.getByRole('button', { name: 'Attach' }))
    expect(await form.findByText('The result cannot be attached: the visit is not one you may see, or its patient has no active consent to care at this clinic.')).toBeTruthy()
  })

  it('releases a result after asking, at the version read, for a practitioner only', async () => {
    let release: Request | undefined
    const fake = renderResults({
      reports: [report({ versionId: 4 })],
      routes: {
        [`POST /clinic/diagnostic-reports/${first}/release`]: (r) => {
          release = r
          return Response.json(report({ status: 'final', releasedAt: '2026-11-03T10:00:00Z', versionId: 5 }))
        },
      },
    })
    fireEvent.click(await section().findByRole('button', { name: 'Release to the patient' }))
    const confirm = within(screen.getByRole('alertdialog', { name: 'Release this result to the patient?' }))
    expect(confirm.getByText('The patient sees it in the app, and it cannot be changed after.')).toBeTruthy()
    fireEvent.click(confirm.getByRole('button', { name: 'Release' }))
    await waitFor(() => expect(section().getByText(/Released to the patient on/)).toBeTruthy())
    expect(release?.headers.get('If-Match')).toBe('W/"4"')
    expect(section().queryByRole('button', { name: 'Release to the patient' })).toBeNull()
    expectCsrfOnWrites(fake.requests)
  })

  it('reloads the list when the result was released or changed meanwhile', async () => {
    let reads = 0
    renderResults({
      routes: {
        [`GET /clinic/appointments/${visit}/diagnostic-reports`]: () => {
          reads++
          return Response.json([reads === 1 ? report() : report({ status: 'final', releasedAt: '2026-11-03T10:00:00Z', versionId: 2 })])
        },
        [`POST /clinic/diagnostic-reports/${first}/release`]: () => apiError(409, 'report_released'),
      },
    })
    fireEvent.click(await section().findByRole('button', { name: 'Release to the patient' }))
    fireEvent.click(screen.getByRole('button', { name: 'Release' }))
    expect(await section().findByText('This result was released already. The list is updated.')).toBeTruthy()
    await waitFor(() => expect(section().getByText(/Released to the patient on/)).toBeTruthy())
    expect(reads).toBe(2)
  })

  it('offers the admin no release', async () => {
    renderResults({ reports: [report()], practitioner: false })
    expect(await section().findByRole('heading', { name: 'Placeholder panel' })).toBeTruthy()
    expect(section().queryByRole('button', { name: 'Release to the patient' })).toBeNull()
    expect(section().getByRole('button', { name: 'Add a lab result' })).toBeTruthy()
  })

  it.each(['bs', 'hr', 'sr-Latn', 'sr-Cyrl'] as const)('is translated in %s', async (language) => {
    renderResults({ reports: [report()], language })
    const s = within(await screen.findByRole('region'))
    expect(s.queryByText('Lab results')).toBeNull()
    expect(s.queryByText('Not released: the patient does not see it yet')).toBeNull()
  })
})
