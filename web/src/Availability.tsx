import { useId, useState, type ReactNode } from 'react'
import type { Account, ApiClient } from './api/client'
import type { Bundle, FhirClient, Questionnaire } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { CrossIcon } from './icons'
import { useLoad } from './load'
import { PractitionerPicker } from './PractitionerPicker'
import { fromFHIR, type Version } from './questionnaire'
import { loadStaff, loadTimeZone } from './staff'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Saved = components['schemas']['Availability']
type Rule = components['schemas']['AvailabilityRule']

type T = ReturnType<typeof useT>

const days = [1, 2, 3, 4, 5, 6, 7] // Monday to Sunday, as the API numbers them

// A time of day as the API takes it, 00:00 to 23:59. The fields are text, not the browser's time input, which
// shows a 12-hour clock in some browser locales (SPEC §3 "Formatting": times are 24-hour in every language).
const clock = /^([01]\d|2[0-3]):[0-5]\d$/

// Availability edits a practitioner's slot length and weekly working hours, in the clinic's time zone: their
// own, or, for an admin, those of the practitioner they pick. Saving sends If-Match with the version read.
// Below them, the role's default questionnaire (W30, W31), saved as soon as it is picked.
export function Availability({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const admin = account.role === 'admin'
  const { state: setup } = useLoad(async () => {
    const [zone, staff] = await Promise.all([loadTimeZone(fhir, organizationId), loadStaff(fhir, organizationId)])
    return { zone, staff: staff.filter((m) => m.active) }
  }, [fhir, organizationId])
  const [chosen, setChosen] = useState<string>()

  if (setup.name !== 'loaded') {
    return <Panel>{setup.name === 'loading' ? <p className="muted">{t('common.loading')}</p> : <Alert message={t('calendar.hoursLoadFailed')} />}</Panel>
  }
  const { zone, staff } = setup.value
  const practitionerId = admin ? (chosen ?? staff[0]?.practitionerId) : account.practitionerId
  const member = staff.find((m) => m.practitionerId === practitionerId)
  return (
    <Panel>
      <div className="toolbar">
        {admin && staff.length > 0 && <PractitionerPicker staff={staff} value={practitionerId ?? ''} onChange={setChosen} />}
        <p className="muted small">{t('calendar.timesIn', { zone })}</p>
      </div>
      {/* Both are keyed by the role, so that picking another practitioner starts them afresh; the prefixes keep
          the two siblings' keys apart. */}
      {member ? (
        <Editor key={`hours-${member.roleId}`} api={api} roleId={member.roleId} />
      ) : (
        <p className="muted">{t('common.noPractitioners')}</p>
      )}
      {member && <DefaultQuestionnaire key={`default-${member.roleId}`} api={api} fhir={fhir} roleId={member.roleId} />}
    </Panel>
  )
}

function Panel({ children }: { children: ReactNode }) {
  const t = useT()
  return (
    <section className="panel">
      <h1>{t('calendar.workingHours')}</h1>
      {children}
    </section>
  )
}

function Editor({ api, roleId }: { api: ApiClient; roleId: string }) {
  const t = useT()
  const { state, reload, set } = useLoad(async () => {
    const res = await attempt(api.GET('/practitioner-roles/{id}/availability', { params: { path: { id: roleId } } }))
    if (!res.ok) throw new Error('availability')
    return res.data
  }, [api, roleId])

  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('calendar.hoursLoadFailed')} />
  return <HoursForm api={api} saved={state.value} onSaved={set} onReload={reload} />
}

type Row = Rule & { key: number }
type FormProps = { api: ApiClient; saved: Saved; onSaved: (s: Saved) => void; onReload: () => void }

function HoursForm({ api, saved, onSaved, onReload }: FormProps) {
  const t = useT()
  const [slotMinutes, setSlotMinutes] = useState(String(saved.slotMinutes ?? 30))
  const [rows, setRows] = useState<Row[]>(saved.availability.map((r, key) => ({ ...r, key })))
  const [nextKey, setNextKey] = useState(saved.availability.length)
  const [done, setDone] = useState(false)
  const [conflict, setConflict] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setDone(false)
    setConflict(false)
    const minutes = Number(slotMinutes)
    if (!Number.isInteger(minutes)) return saveMessage(t, 'invalid_slot_minutes')
    if (rows.some((r) => !clock.test(r.start) || !clock.test(r.end))) return t('calendar.invalidTime')
    const res = await attempt(
      api.PUT('/practitioner-roles/{id}/availability', {
        params: { path: { id: saved.practitionerRoleId }, header: { 'If-Match': `W/"${saved.versionId}"` } },
        body: { slotMinutes: minutes, availability: rows.map(({ day, start, end }) => ({ day, start, end })) },
      }),
    )
    if (res.ok) {
      onSaved(res.data)
      setDone(true)
      return undefined
    }
    setConflict(res.code === 'version_conflict')
    return saveMessage(t, res.code)
  })

  function change(key: number, next: Partial<Rule>) {
    setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...next } : r)))
    setDone(false)
  }

  function add() {
    const last = rows.at(-1)
    setRows((rs) => [...rs, { key: nextKey, day: last ? Math.min(last.day + 1, 7) : 1, start: '09:00', end: '17:00' }])
    setNextKey((k) => k + 1)
    setDone(false)
  }

  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label={t('calendar.slotMinutes')}
        type="number"
        min={5}
        max={240}
        step={5}
        value={slotMinutes}
        onChange={(e) => {
          setSlotMinutes(e.target.value)
          setDone(false)
        }}
        className="short"
      />
      <fieldset className="hours">
        <legend>{t('calendar.weeklyHours')}</legend>
        {rows.length === 0 && <p className="muted">{t('calendar.noHours')}</p>}
        <ul className="hours-rows" aria-label={t('calendar.workingHours')}>
          {rows.map((r, i) => (
            <li key={r.key}>
              <select aria-label={t('calendar.dayN', { n: i + 1 })} value={r.day} onChange={(e) => change(r.key, { day: Number(e.target.value) })}>
                {days.map((d) => (
                  <option key={d} value={d}>
                    {t(`calendar.day.${d}`)}
                  </option>
                ))}
              </select>
              <ClockField label={t('calendar.fromN', { n: i + 1 })} value={r.start} onChange={(start) => change(r.key, { start })} />
              <span className="muted" aria-hidden="true">
                {t('calendar.to')}
              </span>
              <ClockField label={t('calendar.toN', { n: i + 1 })} value={r.end} onChange={(end) => change(r.key, { end })} />
              <button
                type="button"
                className="icon"
                aria-label={t('calendar.removeHoursN', { n: i + 1 })}
                title={t('calendar.remove')}
                onClick={() => {
                  setRows((rs) => rs.filter((x) => x.key !== r.key))
                  setDone(false)
                }}
              >
                <CrossIcon />
              </button>
            </li>
          ))}
        </ul>
        <div className="actions">
          <button type="button" onClick={add}>
            {t('calendar.addHours')}
          </button>
        </div>
      </fieldset>
      <Alert message={error} />
      {done && <Notice message={t('calendar.hoursSaved')} />}
      <div className="actions">
        <Submit busy={busy} label={t('common.save')} busyLabel={t('common.saving')} />
        {conflict && (
          <button type="button" onClick={onReload}>
            {t('common.reload')}
          </button>
        )}
      </div>
    </form>
  )
}

type Defaults = { url?: string; actives: Version[] }

// DefaultQuestionnaire picks the questionnaire requested with every booking with the role, among the clinic's
// active ones, or none. A default whose questionnaire has no active version any more shows as none, with W31.
function DefaultQuestionnaire({ api, fhir, roleId }: { api: ApiClient; fhir: FhirClient; roleId: string }) {
  const t = useT()
  const id = useId()
  const { state, set } = useLoad(async (): Promise<Defaults> => {
    const [res, bundle] = await Promise.all([
      attempt(api.GET('/practitioner-roles/{id}/questionnaire', { params: { path: { id: roleId } } })),
      fhir.get<Bundle<Questionnaire>>(`Questionnaire?${new URLSearchParams({ status: 'active' })}`),
    ])
    if (!res.ok) throw new Error('default questionnaire')
    const actives = (bundle.entry ?? []).map((e) => fromFHIR(e.resource)).sort((a, b) => a.title.localeCompare(b.title))
    return { url: res.data.url, actives }
  }, [api, fhir, roleId])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('common.somethingWrong')} />
  const { url, actives } = state.value
  const current = actives.find((v) => v.url === url)

  async function change(questionnaireId: string) {
    setBusy(true)
    setError(undefined)
    const params = { params: { path: { id: roleId } } }
    const res = await (questionnaireId === ''
      ? attempt(api.DELETE('/practitioner-roles/{id}/questionnaire', params))
      : attempt(api.PUT('/practitioner-roles/{id}/questionnaire', { ...params, body: { questionnaireId } })))
    setBusy(false)
    if (res.ok) set({ actives, url: actives.find((v) => v.id === questionnaireId)?.url })
    else setError(res.code === 'not_active' ? t('calendar.defaultNotActive') : t('common.somethingWrong'))
  }

  return (
    <div className="field default-questionnaire">
      <label htmlFor={id}>{t('calendar.defaultQuestionnaire')}</label>
      <select id={id} value={current?.id ?? ''} disabled={busy} aria-describedby={`${id}-hint`} onChange={(e) => change(e.target.value)}>
        <option value="">{t('calendar.none')}</option>
        {actives.map((v) => (
          <option key={v.id} value={v.id}>
            {v.title}
          </option>
        ))}
      </select>
      <p className="hint" id={`${id}-hint`}>
        {t('calendar.defaultHint')}
      </p>
      <Alert message={error ?? (url !== undefined && !current ? t('calendar.defaultNotActive') : undefined)} />
    </div>
  )
}

function saveMessage(t: T, code?: string): string {
  if (code === 'invalid_slot_minutes') return t('calendar.invalidSlotMinutes')
  if (code === 'invalid_availability') return t('calendar.invalidAvailability')
  if (code === 'version_conflict') return t('calendar.hoursConflict')
  if (code === 'forbidden') return t('calendar.hoursForbidden')
  return t('common.somethingWrong')
}

// ClockField is a time of day typed as HH:MM.
function ClockField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <input
      type="text"
      inputMode="numeric"
      autoComplete="off"
      maxLength={5}
      aria-label={label}
      value={value}
      onChange={(e) => onChange(e.target.value.trim())}
    />
  )
}
