import { describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { saveAppointmentSettings, type AppointmentSettings } from './appointmentSettings'
import { apiError, fakeFetch } from './test/fakeFetch'

const shown: AppointmentSettings = {
  reminder_offsets_minutes: [1440, 60],
  sms_reminders: false,
  reschedule_notice_hours: 24,
  reschedule_max: 2,
  waitlist_enabled: false,
  waitlist_hold_minutes: 15,
  noshow_flag_threshold: 0,
  noshow_flag_window_days: 180,
}

const read = (s: AppointmentSettings, version: number) => () => Response.json(s, { headers: { ETag: `W/"${version}"` } })

describe('saveAppointmentSettings', () => {
  it('saves the changes on the settings as read now, If-Match the version read', async () => {
    const now = { ...shown, waitlist_enabled: true }
    const { fetch, requests } = fakeFetch({
      'GET /clinic/appointment-settings': read(now, 3),
      'PUT /clinic/appointment-settings': async (r) => Response.json(await r.json(), { headers: { ETag: 'W/"4"' } }),
    })

    const res = await saveAppointmentSettings(newApiClient(fetch), shown, { reschedule_max: 4 })

    expect(res).toEqual({ ok: true, data: { ...now, reschedule_max: 4 } })
    const put = requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"3"')
    expect(await put.json()).toEqual({ ...now, reschedule_max: 4 })
  })

  it('is a conflict, sending nothing, when someone else saved a field of the section since it was shown', async () => {
    const { fetch, requests } = fakeFetch({ 'GET /clinic/appointment-settings': read({ ...shown, reminder_offsets_minutes: [120] }, 2) })

    const res = await saveAppointmentSettings(newApiClient(fetch), shown, { reminder_offsets_minutes: [60], sms_reminders: false })

    expect(res).toEqual({ ok: false, conflict: true })
    expect(requests.filter((r) => r.method === 'PUT')).toHaveLength(0)
  })

  it('is a conflict when the API refuses the version (412), and the refusal otherwise', async () => {
    const conflict = fakeFetch({
      'GET /clinic/appointment-settings': read(shown, 1),
      'PUT /clinic/appointment-settings': () => apiError(412, 'version_conflict'),
    })
    expect(await saveAppointmentSettings(newApiClient(conflict.fetch), shown, { reschedule_max: 1 })).toEqual({ ok: false, conflict: true })

    const refused = fakeFetch({
      'GET /clinic/appointment-settings': read(shown, 1),
      'PUT /clinic/appointment-settings': () => apiError(409, 'sms_unavailable'),
    })
    expect(await saveAppointmentSettings(newApiClient(refused.fetch), shown, { sms_reminders: true })).toEqual({
      ok: false,
      conflict: false,
      code: 'sms_unavailable',
      status: 409,
    })
  })
})
