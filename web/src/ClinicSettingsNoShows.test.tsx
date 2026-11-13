import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { ClinicSettingsNoShows } from './ClinicSettingsNoShows'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const settings = {
  reminder_offsets_minutes: [1440, 60],
  sms_reminders: false,
  reschedule_notice_hours: 24,
  reschedule_max: 2,
  waitlist_enabled: true,
  waitlist_hold_minutes: 20,
  noshow_flag_threshold: 0,
  noshow_flag_window_days: 180,
}

function renderSection(routes: Record<string, Route>) {
  const fake = fakeFetch({ 'GET /clinic/appointment-settings': () => Response.json(settings), ...routes })
  renderInLanguage(<ClinicSettingsNoShows api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('ClinicSettingsNoShows', () => {
  it('saves the threshold and the window with the clinic’s other settings unchanged, with the CSRF header', async () => {
    const { requests } = renderSection({ 'PUT /clinic/appointment-settings': async (r) => Response.json(await r.json()) })

    fireEvent.change(await screen.findByLabelText('Flag from this many no-shows'), { target: { value: '3' } })
    fireEvent.change(screen.getByLabelText('Days counted'), { target: { value: '365' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save the flag' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    const put = requests.find((r) => r.method === 'PUT')!
    expect(await put.json()).toEqual({ ...settings, noshow_flag_threshold: 3, noshow_flag_window_days: 365 })
    expectCsrfOnWrites(requests)
  })

  it.each([
    ['a threshold over 10', '11', '180'],
    ['a window under 30 days', '2', '29'],
    ['an empty threshold', '', '180'],
  ])('refuses %s before sending anything', async (_, threshold, days) => {
    const { requests } = renderSection({})

    fireEvent.change(await screen.findByLabelText('Flag from this many no-shows'), { target: { value: threshold } })
    fireEvent.change(screen.getByLabelText('Days counted'), { target: { value: days } })
    fireEvent.click(screen.getByRole('button', { name: 'Save the flag' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Enter a number of no-shows from 0 to 10 and a number of days from 30 to 730.')
    expect(requests.filter((r) => r.method === 'PUT')).toHaveLength(0)
  })

  it('says so when the settings cannot be loaded or saved', async () => {
    renderSection({ 'GET /clinic/appointment-settings': () => apiError(500, 'internal error') })
    expect((await screen.findByRole('alert')).textContent).toBe('The no-show settings could not be loaded. Reload the page to try again.')
    cleanup()

    renderSection({ 'PUT /clinic/appointment-settings': () => apiError(400, 'invalid_request') })
    fireEvent.click(await screen.findByRole('button', { name: 'Save the flag' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
  })
})
