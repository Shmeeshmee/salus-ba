import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Settings = components['schemas']['AppointmentSettings']

// ClinicSettingsNoShows is the admin's repeat no-show flag (SAL-36, Q7): from how many no-shows, over how many days,
// staff see the flag beside a patient's appointments (0 for off). It reads the clinic's appointment settings and
// saves them whole (PUT replaces the row), changing only these two.
export function ClinicSettingsNoShows({ api }: { api: ApiClient }) {
  const t = useT()
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/appointment-settings'))
    if (!res.ok) throw new Error('appointment settings')
    return res.data
  }, [api])
  return (
    <section className="settings-part" aria-labelledby="noshow-settings">
      <h2 id="noshow-settings">{t('calendar.noshow.settingsTitle')}</h2>
      <p className="hint">{t('calendar.noshow.settingsIntro')}</p>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('calendar.noshow.loadFailed')} />}
      {state.name === 'loaded' && <Form api={api} settings={state.value} onSaved={set} />}
    </section>
  )
}

function Form({ api, settings, onSaved }: { api: ApiClient; settings: Settings; onSaved: (s: Settings) => void }) {
  const t = useT()
  const [threshold, setThreshold] = useState(String(settings.noshow_flag_threshold))
  const [days, setDays] = useState(String(settings.noshow_flag_window_days))
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const n = Number(threshold)
    const d = Number(days)
    if (!Number.isInteger(n) || n < 0 || n > 10 || !Number.isInteger(d) || d < 30 || d > 730 || threshold === '' || days === '') {
      return t('calendar.noshow.invalid')
    }
    const res = await attempt(api.PUT('/clinic/appointment-settings', { body: { ...settings, noshow_flag_threshold: n, noshow_flag_window_days: d } }))
    if (!res.ok) return t('common.somethingWrong')
    onSaved(res.data)
    setSaved(true)
    return undefined
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label={t('calendar.noshow.threshold')}
        hint={t('calendar.noshow.thresholdHint')}
        type="number"
        inputMode="numeric"
        min={0}
        max={10}
        value={threshold}
        onChange={(e) => {
          setThreshold(e.target.value)
          setSaved(false)
        }}
      />
      <Field
        label={t('calendar.noshow.window')}
        hint={t('calendar.noshow.windowHint')}
        type="number"
        inputMode="numeric"
        min={30}
        max={730}
        value={days}
        onChange={(e) => {
          setDays(e.target.value)
          setSaved(false)
        }}
      />
      <Alert message={error} />
      {saved && <Notice message={t('calendar.noshow.saved')} />}
      <div className="actions">
        <Submit busy={busy} label={t('calendar.noshow.save')} busyLabel={t('common.saving')} />
      </div>
    </form>
  )
}
