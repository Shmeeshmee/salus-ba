import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Calendar } from './Calendar'
import type { Account } from './api/client'
import { newFhirClient } from './api/fhir'
import { formatDate } from './i18n/format'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { fakeFetch, fhirJSON, outcome, type Route } from './test/fakeFetch'
import { admin, appointment, clinicRoute, cy, dee, practitioner, searchset, staffRoute } from './test/fixtures'

afterEach(cleanup)

// Monday 9 November 2026, 09:10 in Europe/Sarajevo.
const now = () => new Date('2026-11-09T08:10:00Z')
const search = 'GET /fhir/R4/Appointment'

const week = [
  appointment('e1000000-0000-4000-8000-000000000001', '2026-11-09T08:00:00Z', '2026-11-09T08:30:00Z', {}, 'Ada Example'),
  appointment('e1000000-0000-4000-8000-000000000002', '2026-11-11T13:00:00Z', '2026-11-11T13:30:00Z'),
  appointment('e1000000-0000-4000-8000-000000000003', '2026-11-12T09:00:00Z', '2026-11-12T09:30:00Z', { status: 'cancelled' }, 'Bo Sample'),
]

function renderCalendar(account: Account, routes: Record<string, Route>, clock = now, language: Language = 'en') {
  const fake = fakeFetch({ ...clinicRoute, ...routes })
  renderInLanguage(<Calendar fhir={newFhirClient(fake.fetch)} account={account} now={clock} />, language)
  return fake
}

function searches(requests: Request[]) {
  return requests.filter((r) => new URL(r.url).pathname === '/fhir/R4/Appointment').map((r) => new URL(r.url).searchParams)
}

describe('Calendar', () => {
  it("searches the practitioner's own appointments for this week in the clinic's time zone", async () => {
    const { requests } = renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) })

    await screen.findByText('Ada Example')
    const [params] = searches(requests)
    expect(params.get('practitioner')).toBe(`Practitioner/${cy.practitionerId}`)
    expect(params.getAll('date')).toEqual(['ge2026-11-08T23:00:00Z', 'lt2026-11-15T23:00:00Z'])
    expect(screen.getByText('Times in Europe/Sarajevo')).toBeTruthy()
  })

  it('puts each appointment on its day with its local times, the name only where given', async () => {
    renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) })

    const monday = await screen.findByRole('region', { name: 'Mon 9 Nov' })
    const first = within(monday).getByRole('link')
    expect(first.getAttribute('href')).toBe('/appointments/e1000000-0000-4000-8000-000000000001')
    expect(first.textContent).toContain('09:00–09:30')
    expect(first.textContent).toContain('Ada Example')

    const wednesday = within(screen.getByRole('region', { name: 'Wed 11 Nov' })).getByRole('link')
    expect(wednesday.textContent).toBe('14:00–14:30')

    const thursday = within(screen.getByRole('region', { name: 'Thu 12 Nov' })).getByRole('link')
    expect(thursday.textContent).toContain('Cancelled')
  })

  it('marks the appointment under way as the current one, and no other', async () => {
    renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) })

    const current = (await screen.findByText('Ada Example')).closest('a')!
    expect(current.className).toContain('current')
    expect(within(current).getByText('Now')).toBeTruthy()
    expect(screen.getAllByText('Now')).toHaveLength(1)
  })

  it('moves a week forward and back to this week', async () => {
    const { requests } = renderCalendar(practitioner, { [search]: () => fhirJSON(searchset([])) })

    fireEvent.click(await screen.findByRole('button', { name: 'Next week' }))
    await vi.waitFor(() => expect(searches(requests)).toHaveLength(2))
    expect(searches(requests)[1].getAll('date')).toEqual(['ge2026-11-15T23:00:00Z', 'lt2026-11-22T23:00:00Z'])
    expect(await screen.findByRole('region', { name: 'Mon 16 Nov' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: 'This week' }))
    await vi.waitFor(() => expect(searches(requests)).toHaveLength(3))
    expect(searches(requests)[2].getAll('date')[0]).toBe('ge2026-11-08T23:00:00Z')
  })

  it('lets an admin pick whose calendar to see', async () => {
    const { requests } = renderCalendar(admin, { ...staffRoute, [search]: () => fhirJSON(searchset([])) })

    const picker = (await screen.findByLabelText('Practitioner')) as HTMLSelectElement
    await vi.waitFor(() => expect(searches(requests)).toHaveLength(1))
    expect(searches(requests)[0].get('practitioner')).toBe(`Practitioner/${cy.practitionerId}`)
    expect([...picker.options].map((o) => o.textContent)).toEqual(['Cy Example', 'Dee Sample'])

    fireEvent.change(picker, { target: { value: dee.practitionerId } })

    await vi.waitFor(() => expect(searches(requests)).toHaveLength(2))
    expect(searches(requests)[1].get('practitioner')).toBe(`Practitioner/${dee.practitionerId}`)
  })

  it('says when the week has no appointments', async () => {
    renderCalendar(practitioner, { [search]: () => fhirJSON(searchset([])) })

    expect(await screen.findByText('No appointments this week.')).toBeTruthy()
  })

  it('says when the calendar cannot be loaded', async () => {
    renderCalendar(practitioner, { [search]: () => outcome(500, 'exception') })

    expect((await screen.findByRole('alert')).textContent).toBe('The calendar could not be loaded. Reload the page to try again.')
  })

  describe('the join button', () => {
    const first = 'e1000000-0000-4000-8000-000000000001' // Monday 08:00 to 08:30 UTC
    const joinButton = () => screen.queryByRole('button', { name: 'Join consultation' })

    it.each([
      ['10 minutes before the start', '2026-11-09T07:50:00Z', true],
      ['just before that', '2026-11-09T07:49:59Z', false],
      ['until 30 minutes after the end', '2026-11-09T08:59:59Z', true],
      ['from 30 minutes after the end', '2026-11-09T09:00:00Z', false],
    ])('is shown to the practitioner %s', async (_, at, shown) => {
      renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) }, () => new Date(at))

      await screen.findByText('Ada Example')
      expect(joinButton() !== null).toBe(shown)
    })

    it('opens the consultation of its appointment', async () => {
      renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) })

      const monday = await screen.findByRole('region', { name: 'Mon 9 Nov' })
      fireEvent.click(within(monday).getByRole('button', { name: 'Join consultation' }))

      expect(location.pathname).toBe(`/appointments/${first}/call`)
      history.replaceState(null, '', '/')
    })

    it('is hidden on an appointment that is no longer booked', async () => {
      const fulfilled = { ...week[0], status: 'fulfilled' as const }
      renderCalendar(practitioner, { [search]: () => fhirJSON(searchset([fulfilled])) })

      await screen.findByText('Ada Example')
      expect(joinButton()).toBeNull()
    })

    it('is not offered to an admin, who cannot join', async () => {
      renderCalendar(admin, { ...staffRoute, [search]: () => fhirJSON(searchset(week)) })

      await screen.findByText('Ada Example')
      expect(joinButton()).toBeNull()
    })

    it('appears when the window opens while the calendar is shown', async () => {
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
      try {
        let at = new Date('2026-11-09T07:49:50Z')
        renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) }, () => at)
        await screen.findByText('Ada Example')
        expect(joinButton()).toBeNull()

        at = new Date('2026-11-09T07:50:05Z')
        act(() => vi.advanceTimersByTime(15_000))

        expect(joinButton()).not.toBeNull()
      } finally {
        vi.useRealTimers()
      }
    })
  })
})

describe('Calendar in a language', () => {
  it('names the week, its days and the statuses in the language', async () => {
    renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) }, now, 'hr')

    const monday = formatDate(new Date('2026-11-09T12:00:00Z'), 'Europe/Sarajevo', 'hr', 'short')
    expect(monday).toMatch(/pon/i)
    expect(within(await screen.findByRole('region', { name: monday })).getByText('Ada Example')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Kalendar' })).toBeTruthy()
    expect(screen.getByText('Vremenska zona: Europe/Sarajevo')).toBeTruthy()
    expect(screen.getByText('Otkazan')).toBeTruthy()
    expect(screen.getByText('Sada')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Sljedeći tjedan' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Pridruži se konzultaciji' })).toBeTruthy()
  })

  it('writes times in 24 hours', async () => {
    renderCalendar(practitioner, { [search]: () => fhirJSON(searchset(week)) }, now, 'sr-Cyrl')

    expect(await screen.findByText('14:00–14:30')).toBeTruthy()
  })
})
