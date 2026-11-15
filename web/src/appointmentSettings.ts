import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'

export type AppointmentSettings = components['schemas']['AppointmentSettings']

// SettingsSave is a section's save: the settings stored, someone else's save in the way (conflict), or the API's
// refusal.
export type SettingsSave =
  | { ok: true; data: AppointmentSettings }
  | { ok: false; conflict: true }
  | { ok: false; conflict: false; code?: ErrorCode; status?: number }

// saveAppointmentSettings saves one section's changes. The settings are one row replaced whole and versioned: it reads
// them again with their version, refuses (conflict) when a field of this section no longer holds what the section
// showed (someone else saved it since), and otherwise PUTs the row as read with the changes, If-Match the version read,
// so the other sections' values are kept and a save that comes in between is refused by the API (412, conflict).
// shown is what the section loaded or last saved; the caller keeps the user's input either way.
export async function saveAppointmentSettings(
  api: ApiClient,
  shown: AppointmentSettings,
  changes: Partial<AppointmentSettings>,
): Promise<SettingsSave> {
  const fresh = await attempt(api.GET('/clinic/appointment-settings'))
  if (!fresh.ok) return { ok: false, conflict: false, code: fresh.code, status: fresh.response?.status }
  const fields = Object.keys(changes) as (keyof AppointmentSettings)[]
  if (fields.some((k) => JSON.stringify(fresh.data[k]) !== JSON.stringify(shown[k]))) return { ok: false, conflict: true }
  const version = fresh.response.headers.get('ETag') ?? ''
  const res = await attempt(
    api.PUT('/clinic/appointment-settings', { params: { header: { 'If-Match': version } }, body: { ...fresh.data, ...changes } }),
  )
  if (res.ok) return { ok: true, data: res.data }
  if (res.response?.status === 412) return { ok: false, conflict: true }
  return { ok: false, conflict: false, code: res.code, status: res.response?.status }
}
