import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadStaff, loadTimeZone, type Member } from './staff'
import { Alert, Confirm } from './ui'

type Entry = components['schemas']['ClinicWaitlistEntry']
type Status = components['schemas']['WaitlistStatus']

// The list's pages: the API gives 500 entries at most at once.
const pageSize = 100
const maxPage = 500

// Waitlist is the clinic's waitlist for its staff (SAL-35): the entries in offer order (priority, then who joined
// first), each with what the patient waits for, the dates and times, the patient's name where the care consent gives
// it, and an open offer's hold; staff move an entry up or down (priority 0 to 100) and remove one. Offers are made
// by the system, one patient at a time, never from here.
export function Waitlist({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const [status, setStatus] = useState<Status>('active')
  const [limit, setLimit] = useState(pageSize)
  const { state: staff } = useLoad(async () => {
    const [members, zone] = await Promise.all([loadStaff(fhir, organizationId), loadTimeZone(fhir, organizationId)])
    return { members, zone }
  }, [fhir, organizationId])
  const { state, reload } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/waitlist', { params: { query: { status, limit } } }))
    if (!res.ok) throw new Error('waitlist')
    return res.data
  }, [api, status, limit])

  return (
    <section className="panel">
      <h1>{t('waitlist.title')}</h1>
      <div className="toolbar">
        <label>
          <span className="muted small">{t('waitlist.show')} </span>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as Status)
              setLimit(pageSize)
            }}
          >
            <option value="active">{t('waitlist.statusActive')}</option>
            <option value="expired">{t('waitlist.status.expired')}</option>
            <option value="booked">{t('waitlist.status.booked')}</option>
            <option value="withdrawn">{t('waitlist.status.withdrawn')}</option>
          </select>
        </label>
      </div>
      {(state.name === 'loading' || staff.name === 'loading') && <p className="muted">{t('common.loading')}</p>}
      {(state.name === 'failed' || staff.name === 'failed') && <Alert message={t('waitlist.loadFailed')} />}
      {state.name === 'loaded' && staff.name === 'loaded' && (
        <Entries api={api} entries={state.value.entries} members={staff.value.members} zone={staff.value.zone} onChanged={reload} />
      )}
      {state.name === 'loaded' && state.value.next_offset !== undefined && limit < maxPage && (
        <button type="button" onClick={() => setLimit((n) => Math.min(n + pageSize, maxPage))}>
          {t('waitlist.showMore')}
        </button>
      )}
    </section>
  )
}

function Entries({ api, entries, members, zone, onChanged }: { api: ApiClient; entries: Entry[]; members: Member[]; zone: string; onChanged: () => void }) {
  const t = useT()
  if (entries.length === 0) return <p className="muted">{t('waitlist.empty')}</p>
  return (
    <ol className="rows" aria-label={t('waitlist.title')}>
      {entries.map((e) => (
        <Row key={e.entry.id} api={api} entry={e} members={members} zone={zone} onChanged={onChanged} />
      ))}
    </ol>
  )
}

function Row({ api, entry, members, zone, onChanged }: { api: ApiClient; entry: Entry; members: Member[]; zone: string; onChanged: () => void }) {
  const t = useT()
  const f = useFormat()
  const [priority, setPriority] = useState(String(entry.priority))
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const e = entry.entry
  const member = members.find((m) => m.roleId === e.practitioner_role_id)
  const what = e.practitioner_role_id ? (member?.name ?? t('waitlist.aPractitioner')) : t(`common.specialty.${e.specialty}`)
  const active = e.status === 'active'

  async function act(run: () => Promise<{ ok: boolean }>, failed: string) {
    if (busy) return
    setBusy(true)
    setError(undefined)
    const res = await run()
    setBusy(false)
    if (!res.ok) {
      setError(failed)
      return
    }
    onChanged()
  }

  const savePriority = () => {
    const n = Number(priority)
    if (!Number.isInteger(n) || n < 0 || n > 100) {
      setError(t('waitlist.priorityRange'))
      return
    }
    void act(
      () => attempt(api.PUT('/clinic/waitlist/{id}/priority', { params: { path: { id: e.id } }, body: { priority: n } })),
      t('waitlist.priorityFailed'),
    )
  }
  const remove = () =>
    act(() => attempt(api.DELETE('/waitlist/{id}', { params: { path: { id: e.id } } })), t('waitlist.removeFailed'))

  return (
    <li>
      <span className="stack">
        <span>{entry.patient_name ?? t('waitlist.nameNotShown')}</span>
        <span className="muted small">
          {what} · {t('waitlist.between', { from: f.calendarDate(e.earliest), to: f.calendarDate(e.latest) })}
          {e.times && ` · ${t('waitlist.times', { start: e.times.start, end: e.times.end })}`}
        </span>
        {e.offer && <span className="tag">{t('waitlist.offered', { until: f.time(new Date(e.offer.expires_at), zone) })}</span>}
        {!active && <span className="tag">{t(`waitlist.status.${e.status}`)}</span>}
      </span>
      {active && (
        <span className="actions">
          <label>
            <span className="muted small">{t('waitlist.priority')} </span>
            <input
              type="number"
              min={0}
              max={100}
              size={3}
              value={priority}
              aria-label={t('waitlist.priority')}
              onChange={(ev) => setPriority(ev.target.value)}
            />
          </label>
          <button type="button" disabled={busy || priority === String(entry.priority)} onClick={savePriority}>
            {t('waitlist.savePriority')}
          </button>
          <button type="button" className="danger" disabled={busy} onClick={() => setConfirming(true)}>
            {t('waitlist.remove')}
          </button>
        </span>
      )}
      {error && <Alert message={error} />}
      {confirming && (
        <Confirm
          question={t('waitlist.removeQuestion')}
          lines={[t('waitlist.removeEffect')]}
          confirm={t('waitlist.remove')}
          cancel={t('waitlist.keep')}
          danger
          busy={busy}
          onConfirm={() => {
            setConfirming(false)
            void remove()
          }}
          onCancel={() => setConfirming(false)}
        />
      )}
    </li>
  )
}
