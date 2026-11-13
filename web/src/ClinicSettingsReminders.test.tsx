import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicSettingsReminders } from './ClinicSettingsReminders'
import { newApiClient } from './api/client'
import type { components } from './api/schema'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

type Settings = components['schemas']['AppointmentSettings']

// settings are a clinic's, with the other fields away from their defaults so that a save that drops one shows.
const settings: Settings = {
  reminder_offsets_minutes: [60, 1440],
  sms_reminders: false,
  reschedule_notice_hours: 48,
  reschedule_max: 1,
  waitlist_enabled: true,
  waitlist_hold_minutes: 30,
  noshow_flag_threshold: 3,
  noshow_flag_window_days: 365,
}

const now = () => new Date('2026-11-13T10:00:00Z')

function renderReminders(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch({
    'GET /clinic/appointment-settings': () => Response.json(settings),
    'GET /clinic/sms-usage': () => Response.json({ months: [] }),
    ...routes,
  })
  renderInLanguage(<ClinicSettingsReminders api={newApiClient(fake.fetch)} now={now} />, language)
  return fake
}

// echo answers a PUT with the settings it was sent, as the API does.
const echo: Route = async (r) => Response.json(await r.json())

async function offsets() {
  const legend = await screen.findByText('When patients are reminded')
  const list = legend.closest('fieldset')!.querySelector('ul')
  return list ? within(list).getAllByRole('listitem').map((li) => li.firstChild?.textContent) : []
}

async function putBody(requests: Request[]): Promise<Settings> {
  await vi.waitFor(() => expect(requests.some((r) => r.method === 'PUT')).toBe(true))
  return (await requests.find((r) => r.method === 'PUT')!.json()) as Settings
}

describe('ClinicSettingsReminders', () => {
  it('shows the reminders, the earliest first', async () => {
    renderReminders({})

    expect(await offsets()).toEqual(['24 hours before', '1 hour before'])
    expect((screen.getByLabelText('Reminders by SMS') as HTMLInputElement).checked).toBe(false)
    expect(screen.getByText(/\[PRICE\] per SMS/)).toBeTruthy()
  })

  it('names minutes, hours and days', async () => {
    renderReminders({ 'GET /clinic/appointment-settings': () => Response.json({ ...settings, reminder_offsets_minutes: [15, 90, 10080] }) })

    expect(await offsets()).toEqual(['7 days before', '90 minutes before', '15 minutes before'])
  })

  it('adds and removes a reminder and saves them with the other settings as read', async () => {
    const { requests } = renderReminders({ 'PUT /clinic/appointment-settings': echo })

    fireEvent.click(await screen.findByRole('button', { name: 'Remove the reminder 1 hour before' }))
    fireEvent.change(screen.getByLabelText('Add a reminder'), { target: { value: '2880' } })
    expect(await offsets()).toEqual(['2 days before', '24 hours before'])
    fireEvent.click(screen.getByLabelText('Reminders by SMS'))
    fireEvent.click(screen.getByRole('button', { name: 'Save reminders' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    expect(await putBody(requests)).toEqual({ ...settings, reminder_offsets_minutes: [1440, 2880], sms_reminders: true })
    expectCsrfOnWrites(requests)
  })

  it('offers only the reminders not chosen, and none once there are three', async () => {
    renderReminders({})

    const add = (await screen.findByLabelText('Add a reminder')) as HTMLSelectElement
    const options = [...add.options].map((o) => o.textContent)
    expect(options).not.toContain('24 hours before')
    expect(options).toContain('15 minutes before')
    expect(options).toContain('7 days before')

    fireEvent.change(add, { target: { value: '15' } })

    expect(await offsets()).toHaveLength(3)
    expect(screen.queryByLabelText('Add a reminder')).toBeNull()
  })

  it('says when no reminders are sent', async () => {
    renderReminders({ 'GET /clinic/appointment-settings': () => Response.json({ ...settings, reminder_offsets_minutes: [] }) })

    expect(await screen.findByText('No reminders are sent.')).toBeTruthy()
  })

  it.each([
    [409, 'sms_unavailable', 'SMS is not available yet. Turn off reminders by SMS to save the other settings.'],
    [400, 'invalid_request', 'Choose up to three different reminders, from 15 minutes to 7 days before the appointment.'],
    [403, 'forbidden', 'Only a clinic admin can change these settings.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderReminders({ 'PUT /clinic/appointment-settings': () => apiError(status, code) })

    fireEvent.click(await screen.findByRole('button', { name: 'Save reminders' }))

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })

  it('says when the settings cannot be loaded', async () => {
    renderReminders({ 'GET /clinic/appointment-settings': () => apiError(500, 'internal error') })

    expect((await screen.findByRole('alert')).textContent).toBe('The reminder settings could not be loaded. Reload the page to try again.')
  })
})

describe('the SMS usage', () => {
  it('asks for the last 12 months and lists messages and parts per month in the UI language', async () => {
    const { requests } = renderReminders(
      {
        'GET /clinic/sms-usage': () =>
          Response.json({
            months: [
              { month: '2026-10', messages: 1, segments: 2 },
              { month: '2026-11', messages: 12, segments: 21 },
            ],
          }),
      },
      'hr',
    )

    expect(await screen.findByText('listopad 2026.')).toBeTruthy()
    expect(screen.getByText('1 poruka · 2 dijela')).toBeTruthy()
    expect(screen.getByText('studeni 2026.')).toBeTruthy()
    expect(screen.getByText('12 poruka · 21 dio')).toBeTruthy()
    const query = new URL(requests.find((r) => new URL(r.url).pathname === '/clinic/sms-usage')!.url).searchParams
    expect([query.get('from'), query.get('to')]).toEqual(['2025-12', '2026-11'])
  })

  it('says when nothing was sent', async () => {
    renderReminders({})

    expect(await screen.findByText('No SMS sent in the last 12 months.')).toBeTruthy()
  })

  it('says when the usage cannot be loaded', async () => {
    renderReminders({ 'GET /clinic/sms-usage': () => apiError(500, 'internal error') })

    expect((await screen.findByRole('alert')).textContent).toBe('The SMS usage could not be loaded. Reload the page to try again.')
  })
})
