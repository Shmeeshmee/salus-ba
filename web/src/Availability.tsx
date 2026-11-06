import { useId, useState, type ReactNode } from 'react'
import type { Account, ApiClient } from './api/client'
import type { Bundle, FhirClient, Questionnaire } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { CrossIcon } from './icons'
import { useLoad } from './load'
import { questionnaires as t, somethingWrong } from './messages'
import { PractitionerPicker } from './PractitionerPicker'
import { fromFHIR, type Version } from './questionnaire'
import { loadStaff, loadTimeZone } from './staff'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Saved = components['schemas']['Availability']
type Rule = components['schemas']['AvailabilityRule']

const days = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']
const loadFailed = 'The working hours could not be loaded. Reload the page to try again.'

// Availability edits a practitioner's slot length and weekly working hours, in the clinic's time zone: their
// own, or, for an admin, those of the practitioner they pick. Saving sends If-Match with the version read.
// Below them, the role's default questionnaire (W30, W31), saved as soon as it is picked.
export function Availability({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
  const organizationId = account.organizationId ?? ''
  const admin = account.role === 'admin'
  const { state: setup } = useLoad(async () => {
    const [zone, staff] = await Promise.all([loadTimeZone(fhir, organizationId), loadStaff(fhir, organizationId)])
    return { zone, staff: staff.filter((m) => m.active) }
  }, [fhir, organizationId])
  const [chosen, setChosen] = useState<string>()

  if (setup.name !== 'loaded') {
    return <Panel>{setup.name === 'loading' ? <p className="muted">Loading</p> : <Alert message={loadFailed} />}</Panel>
  }
  const { zone, staff } = setup.value
  const practitionerId = admin ? (chosen ?? staff[0]?.practitionerId) : account.practitionerId
  const member = staff.find((m) => m.practitionerId === practitionerId)
  return (
    <Panel>
      <div className="toolbar">
        {admin && staff.length > 0 && <PractitionerPicker staff={staff} value={practitionerId ?? ''} onChange={setChosen} />}
        <p className="muted small">Times in {zone}</p>
      </div>
      {member ? <Editor key={member.roleId} api={api} roleId={member.roleId} /> : <p className="muted">No practitioners yet.</p>}
      {member && <DefaultQuestionnaire key={member.roleId} api={api} fhir={fhir} roleId={member.roleId} />}
    </Panel>
  )
}

function Panel({ children }: { children: ReactNode }) {
  return (
    <section className="panel">
      <h1>Working hours</h1>
      {children}
    </section>
  )
}

function Editor({ api, roleId }: { api: ApiClient; roleId: string }) {
  const { state, reload, set } = useLoad(async () => {
    const res = await attempt(api.GET('/practitioner-roles/{id}/availability', { params: { path: { id: roleId } } }))
    if (!res.ok) throw new Error('availability')
    return res.data
  }, [api, roleId])

  if (state.name === 'loading') return <p className="muted">Loading</p>
  if (state.name === 'failed') return <Alert message={loadFailed} />
  return <HoursForm api={api} saved={state.value} onSaved={set} onReload={reload} />
}

type Row = Rule & { key: number }
type FormProps = { api: ApiClient; saved: Saved; onSaved: (s: Saved) => void; onReload: () => void }

function HoursForm({ api, saved, onSaved, onReload }: FormProps) {
  const [slotMinutes, setSlotMinutes] = useState(String(saved.slotMinutes ?? 30))
  const [rows, setRows] = useState<Row[]>(saved.availability.map((r, key) => ({ ...r, key })))
  const [nextKey, setNextKey] = useState(saved.availability.length)
  const [done, setDone] = useState(false)
  const [conflict, setConflict] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setDone(false)
    setConflict(false)
    const minutes = Number(slotMinutes)
    if (!Number.isInteger(minutes)) return saveMessage('invalid_slot_minutes')
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
    return saveMessage(res.code)
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
        label="Slot length in minutes"
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
        <legend>Weekly hours</legend>
        {rows.length === 0 && <p className="muted">No working hours yet. Patients cannot book until there are some.</p>}
        <ul className="hours-rows" aria-label="Working hours">
          {rows.map((r, i) => (
            <li key={r.key}>
              <select aria-label={`Day ${i + 1}`} value={r.day} onChange={(e) => change(r.key, { day: Number(e.target.value) })}>
                {days.map((d, n) => (
                  <option key={d} value={n + 1}>
                    {d}
                  </option>
                ))}
              </select>
              <input type="time" step={300} aria-label={`From ${i + 1}`} value={r.start} onChange={(e) => change(r.key, { start: e.target.value })} />
              <span className="muted" aria-hidden="true">
                to
              </span>
              <input type="time" step={300} aria-label={`To ${i + 1}`} value={r.end} onChange={(e) => change(r.key, { end: e.target.value })} />
              <button
                type="button"
                className="icon"
                aria-label={`Remove hours ${i + 1}`}
                title="Remove"
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
            Add hours
          </button>
        </div>
      </fieldset>
      <Alert message={error} />
      {done && <Notice message="Saved. Free times follow within a few seconds." />}
      <div className="actions">
        <Submit busy={busy} label="Save" busyLabel="Saving" />
        {conflict && (
          <button type="button" onClick={onReload}>
            Reload
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

  if (state.name === 'loading') return <p className="muted">Loading</p>
  if (state.name === 'failed') return <Alert message={somethingWrong} />
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
    else setError(res.code === 'not_active' ? t.defaultNotActive : somethingWrong)
  }

  return (
    <div className="field default-questionnaire">
      <label htmlFor={id}>{t.defaultLabel}</label>
      <select id={id} value={current?.id ?? ''} disabled={busy} aria-describedby={`${id}-hint`} onChange={(e) => change(e.target.value)}>
        <option value="">{t.none}</option>
        {actives.map((v) => (
          <option key={v.id} value={v.id}>
            {v.title}
          </option>
        ))}
      </select>
      <p className="hint" id={`${id}-hint`}>
        {t.defaultHint}
      </p>
      <Alert message={error ?? (url !== undefined && !current ? t.defaultNotActive : undefined)} />
    </div>
  )
}

function saveMessage(code?: string): string {
  if (code === 'invalid_slot_minutes') return 'Enter a slot length of 5 to 240 minutes, in steps of 5.'
  if (code === 'invalid_availability') return 'Check the hours: each must end after it starts, and hours on the same day must not overlap.'
  if (code === 'version_conflict') return 'Someone else changed these hours. Reload to see their changes.'
  if (code === 'forbidden') return 'You cannot change these hours.'
  return somethingWrong
}
