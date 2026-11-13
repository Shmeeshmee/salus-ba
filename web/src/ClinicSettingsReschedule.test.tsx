import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicSettingsReschedule } from './ClinicSettingsReschedule'
import { newApiClient } from './api/client'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const path = '/clinic/appointment-settings'
const stored = {
  reminder_offsets_minutes: [1440, 60],
  sms_reminders: false,
  reschedule_notice_hours: 24,
  reschedule_max: 2,
  waitlist_enabled: true,
  waitlist_hold_minutes: 20,
  noshow_flag_threshold: 3,
  noshow_flag_window_days: 180,
}

function renderReschedule(routes: Record<string, Route>, language: Language = 'en') {
  const fake = fakeFetch(routes)
  renderInLanguage(<ClinicSettingsReschedule api={newApiClient(fake.fetch)} />, language)
  return fake
}

async function fill(notice: string, max: string) {
  fireEvent.change(await screen.findByLabelText('Notice in hours'), { target: { value: notice } })
  fireEvent.change(screen.getByLabelText('Reschedules per appointment'), { target: { value: max } })
  fireEvent.click(screen.getByRole('button', { name: 'Save rescheduling' }))
}

describe('ClinicSettingsReschedule', () => {
  it("shows the clinic's notice and limit as read", async () => {
    renderReschedule({ [`GET ${path}`]: () => Response.json(stored) })

    expect(((await screen.findByLabelText('Notice in hours')) as HTMLInputElement).value).toBe('24')
    expect((screen.getByLabelText('Reschedules per appointment') as HTMLInputElement).value).toBe('2')
  })

  it('saves the whole row, changing only the notice and the limit', async () => {
    const put = vi.fn(async (r: Request) => Response.json(await r.json()))
    const { requests } = renderReschedule({ [`GET ${path}`]: () => Response.json(stored), [`PUT ${path}`]: put })

    await fill('48', '1')

    expect(await screen.findByText('Saved.')).toBeTruthy()
    const body = await requests.find((r) => r.method === 'PUT')!.json()
    expect(body).toEqual({ ...stored, reschedule_notice_hours: 48, reschedule_max: 1 })
    expectCsrfOnWrites(requests)
  })

  it('reads the settings again just before saving, so another section’s or another admin’s change is kept', async () => {
    let reads = 0
    const newer = { ...stored, reminder_offsets_minutes: [120], waitlist_enabled: false, noshow_flag_threshold: 5 }
    const { requests } = renderReschedule({
      [`GET ${path}`]: () => Response.json(reads++ === 0 ? stored : newer),
      [`PUT ${path}`]: async (r) => Response.json(await r.json()),
    })

    await fill('48', '1')

    expect(await screen.findByText('Saved.')).toBeTruthy()
    expect(await requests.find((r) => r.method === 'PUT')!.json()).toEqual({ ...newer, reschedule_notice_hours: 48, reschedule_max: 1 })
    expect(reads).toBe(2)
  })

  it('says when someone else saved the settings first, keeping what was entered', async () => {
    renderReschedule({ [`GET ${path}`]: () => Response.json(stored), [`PUT ${path}`]: () => apiError(412, 'version_conflict') })

    await fill('48', '1')

    expect((await screen.findByRole('alert')).textContent).toBe('Settings were changed by someone else — reload the page. What you entered stays in the form.')
    expect((screen.getByLabelText('Notice in hours') as HTMLInputElement).value).toBe('48')
  })

  it('says that a limit of 0 turns self-service rescheduling off', async () => {
    renderReschedule({ [`GET ${path}`]: () => Response.json({ ...stored, reschedule_max: 0 }) })

    expect(await screen.findByText('Self-service rescheduling is off.')).toBeTruthy()
  })

  it.each([
    ['169', '2'],
    ['24', '6'],
    ['', '2'],
    ['1.5', '2'],
  ])('refuses a notice of %s hours and a limit of %s before sending', async (notice, max) => {
    const { requests } = renderReschedule({ [`GET ${path}`]: () => Response.json(stored) })

    await fill(notice, max)

    expect((await screen.findByRole('alert')).textContent).toBe('Enter a notice of 0 to 168 hours and a limit of 0 to 5 reschedules.')
    expect(requests.filter((r) => r.method === 'PUT')).toHaveLength(0)
  })

  it.each([
    [400, 'invalid_request', 'Enter a notice of 0 to 168 hours and a limit of 0 to 5 reschedules.'],
    [403, 'forbidden', "Only the clinic's admin can change these rules."],
    [500, 'internal', 'Something went wrong. Try again.'],
  ])('says what a %i %s answer means', async (status, code, message) => {
    renderReschedule({ [`GET ${path}`]: () => Response.json(stored), [`PUT ${path}`]: () => apiError(status, code) })

    await fill('12', '3')

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })

  it('says when the rules cannot be loaded', async () => {
    renderReschedule({ [`GET ${path}`]: () => apiError(500, 'internal') })

    expect((await screen.findByRole('alert')).textContent).toBe('The rescheduling rules could not be loaded. Reload the page to try again.')
  })

  it('writes the form in the language', async () => {
    renderReschedule({ [`GET ${path}`]: () => Response.json(stored) }, 'sr-Cyrl')

    expect(await screen.findByRole('heading', { name: 'Промјена термина' })).toBeTruthy()
  })
})
