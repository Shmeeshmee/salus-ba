import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Settings = components['schemas']['AppointmentSettings']
type T = ReturnType<typeof useT>

// The ranges of the clinic's rescheduling rules (api/openapi.yaml AppointmentSettings).
const maxNoticeHours = 168
const maxReschedules = 5

// ClinicSettingsReschedule is the clinic admin's rescheduling rules: how many hours before the start a patient may
// still move their appointment, and how often. GET /clinic/appointment-settings, then a PUT of the whole row with
// every other field as read again just before the save.
export function ClinicSettingsReschedule({ api }: { api: ApiClient }) {
  const t = useT()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/appointment-settings'))
    if (!res.ok) throw new Error('appointment settings')
    return res.data
  }, [api])
  return (
    <section className="reschedule-settings" aria-labelledby="reschedule-settings">
      <h2 id="reschedule-settings">{t('appointments.reschedule')}</h2>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('appointments.rescheduleLoadFailed')} />}
      {state.name === 'loaded' && <RescheduleForm api={api} stored={state.value} />}
    </section>
  )
}

function RescheduleForm({ api, stored }: { api: ApiClient; stored: Settings }) {
  const t = useT()
  const [settings, setSettings] = useState(stored)
  const [notice, setNotice] = useState(String(stored.reschedule_notice_hours))
  const [max, setMax] = useState(String(stored.reschedule_max))
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const hours = whole(notice, maxNoticeHours)
    const times = whole(max, maxReschedules)
    if (hours === undefined || times === undefined) return t('appointments.invalidReschedule')
    // PUT replaces the whole row and the settings carry no version: read them again just before, so what another
    // section or another admin saved since the page opened is kept and only this section's fields change.
    const fresh = await attempt(api.GET('/clinic/appointment-settings'))
    if (!fresh.ok) return saveMessage(t, fresh.code)
    const body = { ...fresh.data, reschedule_notice_hours: hours, reschedule_max: times }
    const res = await attempt(api.PUT('/clinic/appointment-settings', { body }))
    if (!res.ok) return saveMessage(t, res.code)
    setSettings(res.data)
    setSaved(true)
    return undefined
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <p className="hint">{t('appointments.rescheduleHint')}</p>
      <Field
        label={t('appointments.noticeHours')}
        hint={t('appointments.noticeHoursHint')}
        type="number"
        inputMode="numeric"
        min={0}
        max={maxNoticeHours}
        value={notice}
        onChange={(e) => {
          setNotice(e.target.value)
          setSaved(false)
        }}
      />
      <Field
        label={t('appointments.rescheduleMax')}
        hint={t('appointments.rescheduleMaxHint')}
        type="number"
        inputMode="numeric"
        min={0}
        max={maxReschedules}
        value={max}
        onChange={(e) => {
          setMax(e.target.value)
          setSaved(false)
        }}
      />
      {settings.reschedule_max === 0 && <p className="hint">{t('appointments.rescheduleOff')}</p>}
      <Alert message={error} />
      {saved && <Notice message={t('appointments.rescheduleSaved')} />}
      <div className="actions">
        <Submit busy={busy} label={t('appointments.saveReschedule')} busyLabel={t('common.saving')} />
      </div>
    </form>
  )
}

// whole is the text as a whole number from 0 to hi, or undefined.
function whole(text: string, hi: number): number | undefined {
  if (!/^\d{1,3}$/.test(text.trim())) return undefined
  const n = Number(text.trim())
  return n <= hi ? n : undefined
}

function saveMessage(t: T, code?: string): string {
  if (code === 'invalid_request') return t('appointments.invalidReschedule')
  if (code === 'forbidden') return t('appointments.rescheduleForbidden')
  return t('common.somethingWrong')
}
