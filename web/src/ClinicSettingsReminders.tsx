import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Settings = components['schemas']['AppointmentSettings']
type Month = components['schemas']['SmsUsage']['months'][number]
type T = ReturnType<typeof useT>

// The offsets the picker offers, in minutes before the start: 15 minutes to 7 days (the API's range).
const choices = [15, 30, 60, 120, 180, 360, 720, 1440, 2880, 4320, 10080]
const maxReminders = 3

// ClinicSettingsReminders is the admin's reminders section of the clinic settings: when reminders go out and the
// paid SMS option, saved with PUT /clinic/appointment-settings (the other fields kept as read), and the clinic's
// SMS usage over the last 12 months. now is when the 12 months end.
export function ClinicSettingsReminders({ api, now = () => new Date() }: { api: ApiClient; now?: () => Date }) {
  const t = useT()
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/appointment-settings'))
    if (!res.ok) throw new Error('appointment settings')
    return res.data
  }, [api])

  return (
    <section className="panel reminders">
      <h2>{t('reminders.title')}</h2>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('reminders.loadFailed')} />}
      {state.name === 'loaded' && <RemindersForm settings={state.value} api={api} onSaved={set} />}
      <SmsUsage api={api} now={now} />
    </section>
  )
}

function RemindersForm({ settings, api, onSaved }: { settings: Settings; api: ApiClient; onSaved: (s: Settings) => void }) {
  const t = useT()
  const [offsets, setOffsets] = useState(settings.reminder_offsets_minutes)
  const [sms, setSms] = useState(settings.sms_reminders)
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const res = await attempt(
      api.PUT('/clinic/appointment-settings', { body: { ...settings, reminder_offsets_minutes: offsets, sms_reminders: sms } }),
    )
    if (!res.ok) return saveMessage(t, res.code, res.response?.status)
    onSaved(res.data)
    setSaved(true)
    return undefined
  })
  const sorted = [...offsets].sort((a, b) => b - a)
  const open = choices.filter((m) => !offsets.includes(m))

  function change(next: number[]) {
    setOffsets(next)
    setSaved(false)
  }

  return (
    <form onSubmit={onSubmit} noValidate>
      <fieldset className="hours">
        <legend>{t('reminders.offsets')}</legend>
        {sorted.length === 0 ? (
          <p className="hint">{t('reminders.none')}</p>
        ) : (
          <ul className="rows">
            {sorted.map((m) => (
              <li key={m}>
                <span>{offsetLabel(t, m)}</span>
                <button
                  type="button"
                  className="quiet"
                  aria-label={t('reminders.removeOffset', { offset: offsetLabel(t, m) })}
                  onClick={() => change(offsets.filter((x) => x !== m))}
                >
                  {t('reminders.remove')}
                </button>
              </li>
            ))}
          </ul>
        )}
      </fieldset>
      {offsets.length < maxReminders && (
        <div className="field">
          <label htmlFor="add-reminder">{t('reminders.add')}</label>
          <select
            id="add-reminder"
            value=""
            onChange={(e) => e.target.value !== '' && change([...offsets, Number(e.target.value)])}
            aria-describedby="add-reminder-hint"
          >
            <option value="">{t('reminders.choose')}</option>
            {open.map((m) => (
              <option key={m} value={m}>
                {offsetLabel(t, m)}
              </option>
            ))}
          </select>
          <p className="hint" id="add-reminder-hint">
            {t('reminders.limit')}
          </p>
        </div>
      )}
      <div className="check">
        <input
          id="sms-reminders"
          type="checkbox"
          checked={sms}
          onChange={(e) => {
            setSms(e.target.checked)
            setSaved(false)
          }}
          aria-describedby="sms-reminders-hint"
        />
        <div className="stack">
          <label htmlFor="sms-reminders">{t('reminders.sms')}</label>
          <p className="hint" id="sms-reminders-hint">
            {t('reminders.smsHint')}
          </p>
        </div>
      </div>
      <Alert message={error} />
      {saved && <Notice message={t('admin.saved')} />}
      <div className="actions">
        <Submit busy={busy} label={t('reminders.save')} busyLabel={t('common.saving')} />
      </div>
    </form>
  )
}

// SmsUsage lists the messages and parts sent in each month of the last 12 (UTC), as GET /clinic/sms-usage gives them.
function SmsUsage({ api, now }: { api: ApiClient; now: () => Date }) {
  const t = useT()
  const format = useFormat()
  const [range] = useState(() => lastTwelveMonths(now()))
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/sms-usage', { params: { query: range } }))
    if (!res.ok) throw new Error('sms usage')
    return res.data.months
  }, [api, range])

  return (
    <div className="field">
      <h3 id="sms-usage">{t('reminders.usage')}</h3>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('reminders.usageLoadFailed')} />}
      {state.name === 'loaded' && state.value.length === 0 && <p className="hint">{t('reminders.noUsage')}</p>}
      {state.name === 'loaded' && state.value.length > 0 && (
        <ul className="rows" aria-labelledby="sms-usage">
          {state.value.map((m: Month) => (
            <li key={m.month}>
              <span>{format.month(m.month, 'long')}</span>
              <span className="muted">{t('reminders.usageMonth', { messages: m.messages, parts: m.segments })}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

// offsetLabel is "15 minutes before", "24 hours before" or "7 days before": whole days from 2 days, whole hours
// below that, minutes otherwise.
function offsetLabel(t: T, minutes: number): string {
  if (minutes >= 2880 && minutes % 1440 === 0) return t('reminders.daysBefore', { count: minutes / 1440 })
  if (minutes % 60 === 0) return t('reminders.hoursBefore', { count: minutes / 60 })
  return t('reminders.minutesBefore', { count: minutes })
}

function saveMessage(t: T, code: string | undefined, status: number | undefined): string {
  if (code === 'sms_unavailable') return t('reminders.smsUnavailable')
  if (status === 400) return t('reminders.invalid')
  if (status === 403) return t('admin.clinicForbidden')
  return t('common.somethingWrong')
}

// lastTwelveMonths is the month of now (UTC) and the 11 before it, as YYYY-MM.
function lastTwelveMonths(now: Date): { from: string; to: string } {
  const month = (d: Date) => `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
  return { from: month(new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 11, 1))), to: month(now) }
}
