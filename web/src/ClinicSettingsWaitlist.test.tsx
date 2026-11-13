import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { ClinicSettingsWaitlist } from './ClinicSettingsWaitlist'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const settings = {
  reminder_offsets_minutes: [1440, 60], sms_reminders: false, reschedule_notice_hours: 24, reschedule_max: 2,
  waitlist_enabled: false, waitlist_hold_minutes: 15, noshow_flag_threshold: 0, noshow_flag_window_days: 180,
}

function renderSettings(routes: Record<string, Route>) {
  const fake = fakeFetch({ 'GET /clinic/appointment-settings': () => Response.json(settings), ...routes })
  renderInLanguage(<ClinicSettingsWaitlist api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('ClinicSettingsWaitlist', () => {
  it('turns the waitlist on with its hold and sends the other settings back as read', async () => {
    const fake = renderSettings({ 'PUT /clinic/appointment-settings': async (r) => Response.json(await r.json()) })
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Patients may join the waitlist' }))
    fireEvent.change(screen.getByLabelText(/Hold for an offer/), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Waitlist settings saved.')).toBeTruthy()
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(await put.json()).toEqual({ ...settings, waitlist_enabled: true, waitlist_hold_minutes: 30 })
    expectCsrfOnWrites(fake.requests)
  })

  it('reads the settings again just before saving, so another section’s or another admin’s change is kept', async () => {
    let reads = 0
    const newer = { ...settings, reminder_offsets_minutes: [120], reschedule_max: 4, noshow_flag_threshold: 5 }
    const fake = renderSettings({
      'GET /clinic/appointment-settings': () => Response.json(reads++ === 0 ? settings : newer),
      'PUT /clinic/appointment-settings': async (r) => Response.json(await r.json()),
    })
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Patients may join the waitlist' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Waitlist settings saved.')).toBeTruthy()
    expect(await fake.requests.find((r) => r.method === 'PUT')!.json()).toEqual({ ...newer, waitlist_enabled: true })
    expect(reads).toBe(2)
  })

  it('says when someone else saved the settings first, keeping what was entered', async () => {
    renderSettings({ 'PUT /clinic/appointment-settings': () => apiError(412, 'version_conflict') })
    fireEvent.change(await screen.findByLabelText(/Hold for an offer/), { target: { value: '45' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Settings were changed by someone else — reload the page. What you entered stays in the form.')).toBeTruthy()
    expect((screen.getByLabelText(/Hold for an offer/) as HTMLInputElement).value).toBe('45')
  })

  it('refuses a hold out of range, and says when saving failed', async () => {
    const fake = renderSettings({ 'PUT /clinic/appointment-settings': () => apiError(500, 'internal') })
    fireEvent.change(await screen.findByLabelText(/Hold for an offer/), { target: { value: '200' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Enter a hold from 5 to 120 minutes.')).toBeTruthy()
    expect(fake.requests.some((r) => r.method === 'PUT')).toBe(false)
    fireEvent.change(screen.getByLabelText(/Hold for an offer/), { target: { value: '20' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('The settings could not be saved. Try again.')).toBeTruthy()
  })
})
