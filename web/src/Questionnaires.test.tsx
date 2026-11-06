import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient, type Questionnaire } from './api/fhir'
import { Questionnaires } from './Questionnaires'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, type Route } from './test/fakeFetch'
import { clinicRoute, practitioner, searchset } from './test/fixtures'
import { questionnaireFHIR, questionnaireJSON } from './test/questionnaires'

afterEach(cleanup)
beforeEach(() => history.replaceState(null, '', '/questionnaires'))

function version(id: string, series: string, n: number, status: Questionnaire['status'], title: string): Questionnaire {
  return {
    ...questionnaireFHIR(),
    id,
    url: `https://salus.invalid/fhir/Questionnaire/${series}`,
    version: String(n),
    status,
    title,
    meta: { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' },
  }
}

const library = [
  version('a1', 'a', 1, 'active', 'Placeholder intake'),
  version('a2', 'a', 2, 'draft', 'Placeholder intake'),
  version('b1', 'b', 1, 'retired', 'Placeholder follow-up'),
]

function renderLibrary(routes: Record<string, Route>) {
  const fake = fakeFetch({ ...clinicRoute, ...routes })
  render(<Questionnaires api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={practitioner} />)
  return fake
}

function rows() {
  return within(screen.getByRole('list', { name: 'Questionnaires' })).getAllByRole('listitem')
}

describe('Questionnaires', () => {
  it('lists title and status, with the version and last change in a tooltip', async () => {
    renderLibrary({ 'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset(library)) })

    expect(await screen.findByRole('heading', { name: 'Questionnaires' })).toBeTruthy()
    await screen.findByRole('list', { name: 'Questionnaires' })
    expect(rows().map((r) => r.textContent)).toEqual([
      'Placeholder follow-upRetiredVersion 1, last changed Wednesday 4 November 2026',
      'Placeholder intakeActiveVersion 1, last changed Wednesday 4 November 2026',
      'Placeholder intakeDraftVersion 2, last changed Wednesday 4 November 2026',
    ])
    const link = within(rows()[2]).getByRole('link', { name: 'Placeholder intake' })
    expect(link.getAttribute('href')).toBe('/questionnaires/a2')
    const status = within(rows()[2]).getByText('Draft')
    expect(document.getElementById(status.getAttribute('aria-describedby') ?? '')?.textContent).toBe(
      'Version 2, last changed Wednesday 4 November 2026',
    )
  })

  it.each([
    ['Drafts', ['Placeholder intake']],
    ['Active', ['Placeholder intake']],
    ['Retired', ['Placeholder follow-up']],
    ['All', ['Placeholder follow-up', 'Placeholder intake', 'Placeholder intake']],
  ])('filters by status: %s', async (filter, titles) => {
    renderLibrary({ 'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset(library)) })
    await screen.findByRole('list', { name: 'Questionnaires' })

    fireEvent.click(screen.getByRole('button', { name: filter }))

    expect(screen.getByRole('button', { name: filter }).getAttribute('aria-pressed')).toBe('true')
    expect(rows().map((r) => within(r).getByRole('link').textContent)).toEqual(titles)
  })

  it('says when the library is empty, and when a filter finds nothing', async () => {
    renderLibrary({ 'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset([library[2]])) })

    await screen.findByRole('list', { name: 'Questionnaires' })
    expect(screen.queryByText('No questionnaires with this status.')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Drafts' }))
    expect(screen.getByText('No questionnaires with this status.')).toBeTruthy()
  })

  it('says how to start in an empty library', async () => {
    renderLibrary({ 'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset([])) })

    expect(await screen.findByText('No questionnaires yet. Create one to ask patients questions before their appointments.')).toBeTruthy()
  })

  it('creates an untitled draft and opens it in the builder', async () => {
    const { requests } = renderLibrary({
      'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset([])),
      'POST /questionnaires': () => Response.json({ ...questionnaireJSON(), id: 'new1', title: 'Untitled questionnaire', item: [] }, { status: 201 }),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'New questionnaire' }))

    await vi.waitFor(() => expect(location.pathname).toBe('/questionnaires/new1'))
    const post = requests.find((r) => r.method === 'POST')
    expect(await post?.json()).toEqual({ title: 'Untitled questionnaire', item: [] })
    expectCsrfOnWrites(requests)
  })

  it('shows the fixed message when the draft cannot be created', async () => {
    renderLibrary({
      'GET /fhir/R4/Questionnaire': () => fhirJSON(searchset([])),
      'POST /questionnaires': () => apiError(500, 'internal error'),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'New questionnaire' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
    expect(location.pathname).toBe('/questionnaires')
  })

  it('says so when the library cannot be read', async () => {
    renderLibrary({ 'GET /fhir/R4/Questionnaire': () => outcome(500, 'exception') })

    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
  })
})
