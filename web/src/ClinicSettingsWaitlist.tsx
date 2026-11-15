import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { saveAppointmentSettings } from './appointmentSettings'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Settings = components['schemas']['AppointmentSettings']

// ClinicSettingsWaitlist is the admin's waitlist section of the clinic's appointment settings (SAL-35): whether
// patients may join the waitlist, and how long a freed slot is held for the patient it is offered to (5 to 120
// minutes). The settings are one row replaced whole (PUT /clinic/appointment-settings), so the other sections' values
// are read again just before the save and sent back as they are then.
export function ClinicSettingsWaitlist({ api }: { api: ApiClient }) {
  const t = useT()
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/appointment-settings'))
    if (!res.ok) throw new Error('settings')
    return res.data
  }, [api])
  return (
    <section className="panel">
      <h2>{t('waitlist.settingsTitle')}</h2>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('waitlist.settingsLoadFailed')} />}
      {state.name === 'loaded' && <WaitlistForm api={api} settings={state.value} onSaved={set} />}
    </section>
  )
}

function WaitlistForm({ api, settings, onSaved }: { api: ApiClient; settings: Settings; onSaved: (s: Settings) => void }) {
  const t = useT()
  const [enabled, setEnabled] = useState(settings.waitlist_enabled)
  const [hold, setHold] = useState(String(settings.waitlist_hold_minutes))
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const minutes = Number(hold)
    if (!Number.isInteger(minutes) || minutes < 5 || minutes > 120) return t('waitlist.holdRange')
    // Only this section's fields change; someone else's save of them since they were shown is a conflict.
    const res = await saveAppointmentSettings(api, settings, { waitlist_enabled: enabled, waitlist_hold_minutes: minutes })
    if (!res.ok && res.conflict) return t('appointments.settingsConflict')
    if (!res.ok) return res.code === 'invalid_request' ? t('waitlist.holdRange') : t('waitlist.settingsSaveFailed')
    onSaved(res.data)
    setSaved(true)
    return undefined
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <label className="check">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        <span>{t('waitlist.enabled')}</span>
      </label>
      <p className="hint">{t('waitlist.enabledHint')}</p>
      <Field
        label={t('waitlist.hold')}
        hint={t('waitlist.holdHint')}
        type="number"
        min={5}
        max={120}
        value={hold}
        onChange={(e) => setHold(e.target.value)}
      />
      <Alert message={error} />
      {saved && <Notice message={t('waitlist.settingsSaved')} />}
      <Submit busy={busy} label={t('waitlist.save')} busyLabel={t('waitlist.saving')} />
    </form>
  )
}
