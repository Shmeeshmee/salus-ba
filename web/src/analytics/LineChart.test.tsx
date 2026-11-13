import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { LineChart } from './LineChart'

afterEach(cleanup)

const periods = ['2026-08', '2026-09', '2026-10']
const series = [
  { key: 'booked', label: 'Booked', color: 'var(--series-1)', values: [120, 1234, 98] },
  { key: 'cancelled', label: 'Cancelled', color: 'var(--series-3)', values: [16, null, 9] },
]

describe('LineChart', () => {
  it('names every point with its month and exact value', () => {
    renderInLanguage(<LineChart title="Appointments" periods={periods} series={series} />, 'en')
    const chart = screen.getByRole('figure', { name: 'Appointments' })
    expect(within(chart).getByRole('img', { name: 'September 2026, Booked: 1,234' })).toBeTruthy()
    expect(within(chart).getAllByRole('img')).toHaveLength(6)
  })

  it('shows the exact value in a tooltip on focus and moves between points with the arrow keys', () => {
    renderInLanguage(<LineChart title="Appointments" periods={periods} series={series} />, 'en')
    const first = screen.getByRole('img', { name: 'August 2026, Booked: 120' })
    expect(first.getAttribute('tabindex')).toBe('0')
    fireEvent.focus(first)
    expect(screen.getByRole('tooltip').textContent).toContain('120')
    fireEvent.keyDown(first, { key: 'ArrowRight' })
    const second = screen.getByRole('img', { name: 'September 2026, Booked: 1,234' })
    expect(document.activeElement).toBe(second)
    expect(second.getAttribute('tabindex')).toBe('0')
    expect(first.getAttribute('tabindex')).toBe('-1')
    fireEvent.keyDown(second, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(screen.getByRole('img', { name: /September 2026, Cancelled: Hidden/ }))
    expect(screen.getByRole('tooltip').textContent).toMatch(/hidden to protect privacy/)
    fireEvent.blur(document.activeElement!)
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('has a legend for two series and a table view', () => {
    renderInLanguage(<LineChart title="Appointments" periods={periods} series={series} />, 'en')
    const legend = screen.getByRole('list', { name: 'Legend' })
    expect(within(legend).getAllByRole('listitem').map((li) => li.textContent)).toEqual(['Booked', 'Cancelled'])
    const table = screen.getByRole('table')
    expect(within(table).getAllByRole('row')[2].textContent).toBe('September 20261,234Hidden')
  })

  it('has no legend for one series', () => {
    renderInLanguage(<LineChart title="Patients" periods={periods} series={[series[0]]} />, 'en')
    expect(screen.queryByRole('list', { name: 'Legend' })).toBeNull()
  })

  it('says so when there is nothing to show', () => {
    renderInLanguage(<LineChart title="Appointments" periods={[]} series={[]} />, 'en')
    expect(screen.getByRole('figure', { name: 'Appointments' }).textContent).toContain('No data for these filters.')
    expect(screen.queryByRole('img')).toBeNull()
  })
})

describe('LineChart hidden months', () => {
  it('describes a hidden month with the privacy explanation, for a screen reader', () => {
    renderInLanguage(<LineChart title="Months" periods={['2026-01', '2026-02']} series={[{ key: 'p', label: 'Patients', color: 'red', values: [5, null] }]} />, 'en')
    const hidden = screen.getByRole('img', { name: 'February 2026, Patients: Hidden' })
    expect(document.getElementById(hidden.getAttribute('aria-describedby') ?? '')?.textContent).toBe('Small or dominated groups are hidden to protect privacy. A hidden group can hold more patients than a group shown.')
    expect(screen.getByRole('img', { name: 'January 2026, Patients: 5' }).getAttribute('aria-describedby')).toBeNull()
  })
})
