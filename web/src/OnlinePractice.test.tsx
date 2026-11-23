import { cleanup, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { renderInLanguage } from './i18n/renderInLanguage'
import { OnlinePractice } from './OnlinePractice'
import { fakeFetch } from './test/fakeFetch'
import { admin, clinicRoute, cy, dee, staffRoute } from './test/fixtures'

afterEach(cleanup)

// Saturday 21 November 2026 in Sarajevo: the period is the 1st to today.
const now = () => new Date('2026-11-21T10:00:00Z')
const formerRole = 'a4000000-0000-4000-8000-0000000000ff'

const report = {
  from: '2026-10-31T23:00:00Z',
  to: '2026-11-21T23:00:00Z',
  currency: 'BAM',
  rows: [
    { practitionerRole: cy.roleId, visitMode: 'in-person', minutesOffered: 720, visits: 9, receiptedMinor: 45000 },
    { practitionerRole: cy.roleId, visitMode: 'video', minutesOffered: 150, visits: 2, receiptedMinor: 7500 },
    { practitionerRole: formerRole, visitMode: 'video', minutesOffered: 0, visits: 1, receiptedMinor: 0 },
  ],
}

function renderPractice(routes: Record<string, () => Response>, language: 'en' | 'bs' = 'en') {
  const fake = fakeFetch({ ...clinicRoute, ...staffRoute, ...routes })
  renderInLanguage(<OnlinePractice api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} now={now} />, language)
  return fake
}

describe('OnlinePractice', () => {
  it('shows each practitioner’s hours offered, visits held and receipted amount by visit mode, for this month', async () => {
    const fake = renderPractice({ 'GET /clinic/reports/practice': () => Response.json(report) })
    const table = within(await screen.findByRole('table', { name: 'By practitioner and visit' }))
    const rows = table.getAllByRole('row').slice(1).map((r) => r.textContent?.replace(/\s/g, ' '))
    expect(rows).toEqual(['Cy ExampleAt the clinic129BAM 450.00', 'Cy ExampleVideo2.52BAM 75.00', 'Former practitionerVideo01BAM 0.00'])
    const asked = new URL(fake.requests.find((r) => r.url.includes('/clinic/reports/practice'))!.url)
    expect([asked.searchParams.get('from'), asked.searchParams.get('to')]).toEqual(['2026-10-31T23:00:00.000Z', '2026-11-21T23:00:00.000Z'])
    expect(screen.queryByText(dee.roleId)).toBeNull()
  })

  it('says when the period has nothing to show', async () => {
    renderPractice({ 'GET /clinic/reports/practice': () => Response.json({ ...report, rows: [] }) })
    expect(await screen.findByText('No hours offered and no visits in this period.')).toBeTruthy()
  })

  it('says when the report cannot be loaded', async () => {
    renderPractice({ 'GET /clinic/reports/practice': () => Response.json({ error: 'forbidden' }, { status: 403 }) })
    expect(await screen.findByText('The report could not be loaded. Reload the page to try again.')).toBeTruthy()
  })

  it('speaks the UI language', async () => {
    renderPractice({ 'GET /clinic/reports/practice': () => Response.json(report) }, 'bs')
    expect(await screen.findByRole('table', { name: 'Po doktoru i vrsti posjete' })).toBeTruthy()
  })
})
