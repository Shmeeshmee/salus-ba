import { cleanup, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { noShowRate, type ClinicAppointments } from './data'
import { NoShowCharts } from './NoShowCharts'

afterEach(cleanup)

const gp = { system: 'http://snomed.info/sct' as const, code: '394814009', display: 'General practice', text: 'General practice' }
const figures: ClinicAppointments = {
  series: [{ period: '2026-10', booked: 30, fulfilled: 18, noshow: 2, cancelled: 4, refilled: 3 }],
  heatmap: [],
  practitioners: [{ practitionerRole: 'r1', fulfilled: 9, noshow: 1, booked: 12 }],
  specialties: [
    { specialty: gp.code, booked: 20, fulfilled: 15, noshow: 5, cancelled: 0 },
    { specialty: '', booked: 10, fulfilled: 0, noshow: 0, cancelled: 4 },
  ],
}

function show(admin: boolean, f = figures) {
  renderInLanguage(
    <NoShowCharts figures={f} periods={['2026-09', '2026-10']} admin={admin} specialties={[gp]} staffName={(id) => (id === 'r1' ? 'Cy Example' : id)} />,
    'en',
  )
}

describe('NoShowCharts', () => {
  it('counts the rate over the closed appointments only, none without any', () => {
    expect(noShowRate(2, 18)).toBe(10)
    expect(noShowRate(1, 2)).toBe(33)
    expect(noShowRate(0, 0)).toBeUndefined()
  })

  it('draws the monthly rate and the refilled slots, a month the API leaves out as 0', () => {
    show(false)
    const rate = screen.getByRole('figure', { name: 'No-show rate per month' })
    expect(within(rate).getByRole('img', { name: 'October 2026, No-show rate (%): 10' })).toBeTruthy()
    // September has no closed appointment: a gap, never a perfect 0 %.
    expect(within(rate).getByRole('img', { name: 'September 2026, No-show rate (%): No closed appointments' })).toBeTruthy()
    expect(within(rate).queryByRole('img', { name: 'September 2026, No-show rate (%): 0' })).toBeNull()
    expect(rate.querySelector('path.line')!.getAttribute('d')).toMatch(/^M[^L]*$/) // one point: no line to September
    expect(within(within(rate).getByRole('table')).getAllByRole('row')[1].textContent).toBe('September 2026No closed appointments')
    const refilled = screen.getByRole('figure', { name: 'Slots refilled from the waitlist' })
    expect(within(refilled).getByRole('img', { name: 'October 2026, Refilled slots: 3' })).toBeTruthy()
  })

  it('gives an admin the rate per practitioner and per specialty, and a practitioner neither', () => {
    show(true)
    const byPractitioner = screen.getByRole('figure', { name: 'No-show rate by practitioner' })
    expect(within(byPractitioner).getByRole('img', { name: /^Cy Example: 10/ })).toBeTruthy()
    const bySpecialty = screen.getByRole('figure', { name: 'No-show rate by specialty' })
    expect(within(bySpecialty).getByRole('img', { name: /: 25/ })).toBeTruthy()
    // The roles without a specialty closed no appointment: left out, not shown as 0 %.
    expect(within(bySpecialty).queryByRole('img', { name: /^No specialty/ })).toBeNull()
    expect(within(bySpecialty).getAllByRole('img')).toHaveLength(1)
    cleanup()

    show(false)
    expect(screen.queryByRole('figure', { name: 'No-show rate by practitioner' })).toBeNull()
    expect(screen.queryByRole('figure', { name: 'No-show rate by specialty' })).toBeNull()
  })
})
