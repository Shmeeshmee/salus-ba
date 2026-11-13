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
    { specialty: '', booked: 10, fulfilled: 3, noshow: 0, cancelled: 4 },
  ],
}

function show(admin: boolean, f = figures) {
  renderInLanguage(
    <NoShowCharts figures={f} periods={['2026-09', '2026-10']} admin={admin} specialties={[gp]} staffName={(id) => (id === 'r1' ? 'Cy Example' : id)} />,
    'en',
  )
}

describe('NoShowCharts', () => {
  it('counts the rate over the closed appointments only, 0 without any', () => {
    expect(noShowRate(2, 18)).toBe(10)
    expect(noShowRate(1, 2)).toBe(33)
    expect(noShowRate(0, 0)).toBe(0)
  })

  it('draws the monthly rate and the refilled slots, a month the API leaves out as 0', () => {
    show(false)
    const rate = screen.getByRole('figure', { name: 'No-show rate per month' })
    expect(within(rate).getByRole('img', { name: 'October 2026, No-show rate (%): 10' })).toBeTruthy()
    expect(within(rate).getByRole('img', { name: 'September 2026, No-show rate (%): 0' })).toBeTruthy()
    const refilled = screen.getByRole('figure', { name: 'Slots refilled from the waitlist' })
    expect(within(refilled).getByRole('img', { name: 'October 2026, Refilled slots: 3' })).toBeTruthy()
  })

  it('gives an admin the rate per practitioner and per specialty, and a practitioner neither', () => {
    show(true)
    const byPractitioner = screen.getByRole('figure', { name: 'No-show rate by practitioner' })
    expect(within(byPractitioner).getByRole('img', { name: /^Cy Example: 10/ })).toBeTruthy()
    const bySpecialty = screen.getByRole('figure', { name: 'No-show rate by specialty' })
    expect(within(bySpecialty).getByRole('img', { name: /: 25/ })).toBeTruthy()
    expect(within(bySpecialty).getByRole('img', { name: /^No specialty: 0/ })).toBeTruthy()
    cleanup()

    show(false)
    expect(screen.queryByRole('figure', { name: 'No-show rate by practitioner' })).toBeNull()
    expect(screen.queryByRole('figure', { name: 'No-show rate by specialty' })).toBeNull()
  })
})
