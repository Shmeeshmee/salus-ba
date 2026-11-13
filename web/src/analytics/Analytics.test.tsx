import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { admin, practitioner } from '../test/fixtures'
import { Analytics } from './Analytics'
import type { Account } from '../api/client'
import type { AnalyticsData } from './data'
import { fixtureData } from './fixtures'

afterEach(cleanup)

const now = new Date('2026-11-12T09:00:00Z')
const withPlatform: Account = { ...admin, features: ['analytics_platform'] }

function show(data: AnalyticsData, account: Account = admin) {
  return renderInLanguage(<Analytics data={data} account={account} now={now} />, 'en')
}

// noteOf is a chart's line above the plot: its unit and the partial tag.
const noteOf = (figure: HTMLElement) => figure.querySelector<HTMLElement>('.chart-note')!

const tabNames = () => screen.getAllByRole('tab').map((t) => t.textContent)

describe('Analytics tabs', () => {
  it('shows My clinic alone, and never asks for the platform months, without the feature', async () => {
    const data = fixtureData()
    const months = vi.spyOn(data, 'platformMonths')
    show(data)
    expect(await screen.findByRole('figure', { name: 'Appointments per month' })).toBeTruthy()
    expect(tabNames()).toEqual(['My clinic'])
    expect(months).not.toHaveBeenCalled()
  })

  it('hides All of Salus when the clinic does not contribute', async () => {
    const data = fixtureData({ contributes: false })
    const months = vi.spyOn(data, 'platformMonths')
    show(data, withPlatform)
    await screen.findByRole('figure', { name: 'Appointments per month' })
    await waitFor(() => expect(months).toHaveBeenCalled())
    expect(tabNames()).toEqual(['My clinic'])
  })

  it('switches to All of Salus with the arrow keys when the feature is on and the clinic contributes', async () => {
    show(fixtureData(), withPlatform)
    const mine = screen.getByRole('tab', { name: 'My clinic' })
    await screen.findByRole('tab', { name: 'All of Salus' })
    fireEvent.keyDown(mine, { key: 'ArrowRight' })
    const all = screen.getByRole('tab', { name: 'All of Salus' })
    expect(all.getAttribute('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(all)
    expect(await screen.findByRole('figure', { name: 'Patients per month' })).toBeTruthy()
    expect(screen.queryByRole('figure', { name: 'Appointments per month' })).toBeNull()
  })
})

describe('My clinic', () => {
  it('gives an admin the practitioner filter and breakdown', async () => {
    show(fixtureData({ admin: true }))
    expect(await screen.findByRole('figure', { name: 'Attended appointments by practitioner' })).toBeTruthy()
    expect(await screen.findByRole('img', { name: /^Cy Example: / })).toBeTruthy()
    expect(screen.getByLabelText('Practitioner')).toBeTruthy()
  })

  it('gives a practitioner neither, nor the specialty the API refuses them, and says the diagnoses are their own', async () => {
    show(fixtureData({ admin: false }), practitioner)
    await screen.findByRole('figure', { name: 'Appointments per month' })
    expect(screen.queryByLabelText('Practitioner')).toBeNull()
    expect(screen.queryByLabelText('Specialty')).toBeNull()
    expect(screen.queryByRole('figure', { name: 'Attended appointments by practitioner' })).toBeNull()
    expect(screen.getByText('Only the diagnoses of your own patients are shown.')).toBeTruthy()
  })

  it('drills down from a group to its subgroups and codes, and back', async () => {
    show(fixtureData())
    fireEvent.click(await screen.findByRole('button', { name: /^Group respiratory-infections: .*Open its breakdown$/ }))
    const subgroups = await screen.findByRole('figure', { name: 'Subgroups: Group respiratory-infections' })
    expect((screen.getByLabelText('Diagnosis group') as HTMLSelectElement).value).toBe('respiratory-infections')
    fireEvent.click(await within(subgroups).findByRole('button', { name: /^Group common-cold: / }))
    const codes = await screen.findByRole('figure', { name: 'Codes: Group common-cold' })
    expect(await within(codes).findByRole('img', { name: /^J00: \d+$/ })).toBeTruthy()
    const path = screen.getByRole('navigation', { name: 'Diagnosis breakdown' })
    fireEvent.click(within(path).getByRole('button', { name: 'Group respiratory-infections' }))
    expect(await screen.findByRole('figure', { name: 'Subgroups: Group respiratory-infections' })).toBeTruthy()
    fireEvent.click(within(screen.getByRole('navigation', { name: 'Diagnosis breakdown' })).getByRole('button', { name: 'All groups' }))
    expect(await screen.findByRole('figure', { name: 'Top diagnosis groups' })).toBeTruthy()
  })

  it('reloads every chart for a new range', async () => {
    const data = fixtureData()
    const appointments = vi.spyOn(data, 'clinicAppointments')
    show(data)
    await screen.findByRole('figure', { name: 'Appointments per month' })
    expect(appointments).toHaveBeenLastCalledWith({ from: '2025-12', to: '2026-11' })
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-10' } })
    await waitFor(() => expect(appointments).toHaveBeenLastCalledWith({ from: '2026-10', to: '2026-11' }))
    const chart = await screen.findByRole('figure', { name: 'Appointments per month' })
    await waitFor(() => expect(within(chart).getAllByRole('img')).toHaveLength(2 * 4))
  })

  it('says so when there are no diagnoses for the filters', async () => {
    show({ ...fixtureData(), clinicDiagnoses: async () => ({ rows: [] }) })
    const bars = await screen.findByRole('figure', { name: 'Top diagnosis groups' })
    await waitFor(() => expect(bars.textContent).toContain('No data for these filters.'))
  })

  it('says so when the figures cannot be loaded', async () => {
    show({ ...fixtureData(), clinicAppointments: () => Promise.reject(new Error('down')) })
    expect((await screen.findAllByRole('alert')).map((a) => a.textContent)).toContain('The figures could not be loaded. Try again.')
  })
})

describe('All of Salus', () => {
  async function platform(data: AnalyticsData = fixtureData()) {
    show(data, withPlatform)
    fireEvent.click(await screen.findByRole('tab', { name: 'All of Salus' }))
    await screen.findByRole('figure', { name: 'Patients per month' })
  }

  it('offers only the filters of the table shown, so it never asks for a combination the API refuses', async () => {
    const data = fixtureData()
    const refused: unknown[] = []
    const cells = data.platformCells
    data.platformCells = (q) => cells(q).catch((e: unknown) => (refused.push(q), Promise.reject(e)))
    await platform(data)
    const table = screen.getByLabelText('Table')
    expect(within(table).getAllByRole('option').map((o) => o.textContent)).toEqual(['Region and diagnosis', 'Region and specialty', 'Specialty and diagnosis'])
    const labels = () => ['Region', 'Diagnosis group', 'Specialty'].filter((l) => screen.queryByLabelText(l))
    expect(labels()).toEqual(['Region', 'Diagnosis group'])
    fireEvent.change(screen.getByLabelText('Region'), { target: { value: 'BA-09' } })
    fireEvent.change(screen.getByLabelText('Diagnosis group'), { target: { value: 'respiratory-infections' } })
    await screen.findByRole('figure', { name: 'By city: Kanton Sarajevo' })
    fireEvent.change(table, { target: { value: 'gs' } })
    expect(labels()).toEqual(['Region', 'Specialty'])
    expect(await screen.findByRole('figure', { name: 'By specialty' })).toBeTruthy()
    fireEvent.change(table, { target: { value: 'sd' } })
    expect(labels()).toEqual(['Diagnosis group', 'Specialty'])
    await screen.findByRole('figure', { name: 'Subgroups: Group respiratory-infections' })
    expect(screen.queryByRole('figure', { name: /^By city/ })).toBeNull()
    await waitFor(() => expect(screen.getAllByRole('figure').length).toBeGreaterThan(2))
    expect(refused).toEqual([])
  })

  it('reads a hidden cell as Hidden, and a sum over months as partial patient-months', async () => {
    await platform()
    fireEvent.change(screen.getByLabelText('Region'), { target: { value: 'BA-09' } })
    const cities = await screen.findByRole('figure', { name: 'By city: Kanton Sarajevo' })
    await waitFor(() => expect(noteOf(cities).textContent).toMatch(/^Patient-monthsPartialSome groups/))
    await waitFor(() => expect(within(cities).getAllByRole('button', { name: /: Hidden\. Open its breakdown$/ }).length).toBeGreaterThan(0))
  })

  it('counts patients, not patient-months, for a single month', async () => {
    await platform()
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-10' } })
    const regions = await screen.findByRole('figure', { name: 'By region' })
    await waitFor(() => expect(noteOf(regions).textContent).toMatch(/^Patients(Partial|$)/))
  })

  it('flags an unusual rise with its disclaimer and never calls it an outbreak', async () => {
    await platform()
    const rises = await screen.findByRole('region', { name: 'Unusual rises in the last 8 weeks' })
    await waitFor(() => expect(within(rises).getAllByText('Unusual rise').length).toBe(2))
    expect(within(rises).getByText('Group respiratory-infections, Kanton Sarajevo, week 44 of 2026')).toBeTruthy()
    expect(within(rises).getAllByRole('tooltip')[0].textContent).toBe('Compared with the previous weeks in this region. This is not a public-health alert.')
    expect(document.body.textContent?.toLowerCase()).not.toContain('outbreak')
  })

  it('shows the busiest hours of a published quarter', async () => {
    await platform()
    const heat = await screen.findByRole('figure', { name: 'Busiest weekdays and hours: all of Salus' })
    expect((screen.getByLabelText('Quarter') as HTMLSelectElement).value).toBe('2026-Q3')
    await waitFor(() => expect(within(heat).getAllByRole('img', { name: /: Hidden$/ }).length).toBeGreaterThan(0))
  })
})
