import { useId, useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import type { Bundle, Extension, FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { Dialog } from './chat/Dialog'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import type { Member } from './staff'
import { addDays, dayIn, startOfDay } from './time'
import { Alert, Submit } from './ui'
import { visitModeURL } from './visitMode'

type Appointment = components['schemas']['Appointment']
type DeskVisit = components['schemas']['DeskVisit']

// The FHIR elements of a Schedule and a free Slot the move reads (the clinic's own searches, CapabilityStatement).
type Schedule = { resourceType: 'Schedule'; id: string }
type Slot = { resourceType: 'Slot'; id: string; start: string; end: string; extension?: Extension[] }

// daysAhead is how far the move offers free times: today and the 14 days after it.
const daysAhead = 14

type Props = {
  api: ApiClient
  fhir: FhirClient
  staff: Member[]
  zone: string
  now: () => Date
  visit: DeskVisit
  onMoved: (appointment: Appointment) => void
  onClose: () => void
}

// DeskMove moves a booked visit at the desk (clinic-cms WAVE4 M1): any practitioner of the clinic, one of today's or
// a later free time of the visit's type, with the version read (POST /clinic/appointments/{id}/reschedule). The
// server keeps it off the patient's own reschedules and tells the patient.
export function DeskMove({ api, fhir, staff, zone, now, visit, onMoved, onClose }: Props) {
  const t = useT()
  const f = useFormat()
  const id = useId()
  const { appointment: a, patient: p } = visit
  const [roleId, setRoleId] = useState(staff.some((m) => m.roleId === a.practitionerRoleId) ? a.practitionerRoleId : (staff[0]?.roleId ?? ''))
  const [slotId, setSlotId] = useState('')
  const [dayKey, setDayKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const slots = useLoad(() => freeTimes(fhir, roleId, a, now(), zone), [fhir, roleId, zone])

  async function move(e: FormEvent) {
    e.preventDefault()
    if (busy || slotId === '') return
    setBusy(true)
    setError(undefined)
    const res = await attempt(
      api.POST('/clinic/appointments/{id}/reschedule', {
        params: { path: { id: a.id }, header: { 'If-Match': `W/"${a.versionId}"` } },
        body: { slotId },
      }),
    )
    setBusy(false)
    if (res.ok) {
      onMoved(res.data)
      return
    }
    setError(moveMessage(t, res.code))
    if (res.code === 'slot_unavailable') {
      setSlotId('')
      slots.reload()
    }
  }

  return (
    <Dialog label={t('desk.move.title')} className="receipt-dialog desk-dialog" onClose={onClose}>
      <h2>{t('desk.move.title')}</h2>
      <form onSubmit={move}>
        <div className="desk-patient">
          <span className="stack">
            <span className="strong">
              {p.givenName} {p.familyName}
            </span>
            <span className="muted small">
              {f.date(new Date(a.start), zone, 'long')}, {f.time(new Date(a.start), zone)}–{f.time(new Date(a.end), zone)}
            </span>
          </span>
        </div>
        <div className="field">
          <label htmlFor={`${id}-p`}>{t('common.practitioner')}</label>
          <select
            id={`${id}-p`}
            value={roleId}
            onChange={(e) => {
              setRoleId(e.target.value)
              setSlotId('')
              setDayKey('')
            }}
          >
            {staff.map((m) => (
              <option key={m.roleId} value={m.roleId}>
                {m.name}
              </option>
            ))}
          </select>
        </div>
        {slots.state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {slots.state.name === 'failed' && <Alert message={t('desk.walkIn.slotsFailed')} />}
        {slots.state.name === 'loaded' && slots.state.value.length === 0 && <p className="muted">{t('desk.move.noSlots')}</p>}
        {slots.state.name === 'loaded' && slots.state.value.length > 0 && (
          <DayTimes days={byDay(slots.state.value, zone)} dayKey={dayKey} slotId={slotId} zone={zone} onDay={(k) => { setDayKey(k); setSlotId('') }} onTime={setSlotId} />
        )}
        <Alert message={error} />
        <Submit busy={busy} disabled={slotId === ''} label={t('desk.move.submit')} busyLabel={t('desk.move.moving')} />
      </form>
      <div className="actions">
        <button type="button" onClick={onClose}>
          {t('desk.walkIn.cancel')}
        </button>
      </div>
    </Dialog>
  )
}

type DayTimesProps = { days: [string, Slot[]][]; dayKey: string; slotId: string; zone: string; onDay: (key: string) => void; onTime: (id: string) => void }

// DayTimes picks a day with free times, the first by default, then one of that day's times (wave 4 walk, finding 7):
// two short lists instead of one of every time in 15 days.
function DayTimes({ days, dayKey, slotId, zone, onDay, onTime }: DayTimesProps) {
  const t = useT()
  const f = useFormat()
  const id = useId()
  const [key, times] = days.find(([k]) => k === dayKey) ?? days[0]
  return (
    <>
      <div className="field">
        <label htmlFor={`${id}-d`}>{t('desk.move.day')}</label>
        <select id={`${id}-d`} value={key} onChange={(e) => onDay(e.target.value)}>
          {days.map(([k, day]) => (
            <option key={k} value={k}>
              {f.date(new Date(day[0].start), zone, 'long')}
            </option>
          ))}
        </select>
      </div>
      <div className="field">
        <label htmlFor={`${id}-s`}>{t('desk.move.time')}</label>
        <select id={`${id}-s`} value={slotId} onChange={(e) => onTime(e.target.value)}>
          <option value="">{t('desk.walkIn.chooseTime')}</option>
          {times.map((s) => (
            <option key={s.id} value={s.id}>
              {f.time(new Date(s.start), zone)}–{f.time(new Date(s.end), zone)}
            </option>
          ))}
        </select>
      </div>
    </>
  )
}

// freeTimes is the practitioner role's free times of the visit's type from now to the end of the clinic's day
// daysAhead days later, without the visit's own slot: the role's schedule, then its free slots.
async function freeTimes(fhir: FhirClient, roleId: string, a: Appointment, from: Date, zone: string): Promise<Slot[]> {
  if (roleId === '') return []
  const schedules = await fhir.get<Bundle<Schedule>>(`Schedule?${new URLSearchParams({ actor: `PractitionerRole/${roleId}` })}`)
  const schedule = schedules.entry?.[0]?.resource
  if (!schedule) return []
  const to = startOfDay(addDays(dayIn(from, zone), daysAhead + 1), zone)
  const query = new URLSearchParams([
    ['schedule', `Schedule/${schedule.id}`],
    ['status', 'free'],
    ['start', `ge${from.toISOString()}`],
    ['start', `lt${to.toISOString()}`],
  ])
  const mode = a.visitMode ?? 'video'
  const slots = await fhir.get<Bundle<Slot>>(`Slot?${query}`)
  return (slots.entry ?? [])
    .map((e) => e.resource)
    .filter((s) => s.id !== a.slotId && (s.extension?.find((x) => x.url === visitModeURL)?.valueCode ?? 'video') === mode)
}

// byDay groups the free times by their day in the clinic's zone, in the order the search gives them.
function byDay(slots: Slot[], zone: string): [string, Slot[]][] {
  const groups = new Map<string, Slot[]>()
  for (const s of slots) {
    const d = dayIn(new Date(s.start), zone)
    const key = `${d.year}-${d.month}-${d.day}`
    groups.set(key, [...(groups.get(key) ?? []), s])
  }
  return [...groups]
}

function moveMessage(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'slot_unavailable') return t('desk.walkIn.slotTaken')
  if (code === 'appointment_closed') return t('desk.closed')
  if (code === 'version_conflict') return t('desk.move.changed')
  if (code === 'not_reschedulable') return t('desk.move.joinOpen')
  if (code === 'consent_required') return t('desk.walkIn.consentRequired')
  return t('desk.move.failed')
}
