import { useId, useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import type { Bundle, Extension, FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { dayStart, planMessage, stepText, type CarePlan } from './carePlan'
import { Dialog } from './chat/Dialog'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadStaff } from './staff'
import { dayIn, sameDay } from './time'
import { Alert, Submit } from './ui'
import { visitModeURL } from './visitMode'

type Service = components['schemas']['Service']
type Schedule = { resourceType: 'Schedule'; id: string }
type Slot = { resourceType: 'Slot'; id: string; start: string; end: string; extension?: Extension[] }

type Props = {
  api: ApiClient
  fhir: FhirClient
  plan: CarePlan // a draft
  services: Service[]
  zone: string
  practitionerId?: string // the signed-in practitioner, whose role is offered first
  today: string // the clinic's date today, YYYY-MM-DD: the first day offered
  onBooked: (plan: CarePlan) => void
  onClose: () => void
}

// CarePlanBook books every step of a draft plan (POST /clinic/care-plans/{id}/book, at the version read): one
// practitioner, the plan's first day, and for each step one free time of its visit type on its day (the first day
// plus the step's days), no time taken twice. The server books every visit or none.
export function CarePlanBook({ api, fhir, plan, services, zone, practitionerId, today, onBooked, onClose }: Props) {
  const t = useT()
  const f = useFormat()
  const id = useId()
  const [roleId, setRoleId] = useState('')
  const [startsOn, setStartsOn] = useState(today)
  const [picked, setPicked] = useState<string[]>(plan.steps.map(() => ''))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const staff = useLoad(async () => (await loadStaff(fhir, plan.organizationId)).filter((m) => m.active), [fhir, plan.organizationId])
  const members = staff.state.name === 'loaded' ? staff.state.value : []
  const role = roleId || (members.find((m) => m.practitionerId === practitionerId) ?? members[0])?.roleId || ''
  const last = Math.max(...plan.steps.map((s) => s.dayOffset))
  const dated = /^\d{4}-\d{2}-\d{2}$/.test(startsOn)
  const slots = useLoad(
    async () => (dated ? freeSlots(fhir, role, dayStart(startsOn, 0, zone), dayStart(startsOn, last + 1, zone)) : []),
    [fhir, role, startsOn, zone, last, dated],
  )
  const free = slots.state.name === 'loaded' ? slots.state.value : []
  const ready = dated && picked.every((p) => p !== '')

  async function book(e: FormEvent) {
    e.preventDefault()
    if (busy || !ready) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(
      api.POST('/clinic/care-plans/{id}/book', {
        params: { path: { id: plan.id }, header: { 'If-Match': `W/"${plan.versionId}"` } },
        body: { startsOn, slotIds: picked },
      }),
    )
    setBusy(false)
    if (res.ok) onBooked(res.data)
    else {
      setError(planMessage(t, res.code))
      if (res.code === 'slot_unavailable') slots.reload()
    }
  }

  const reset = () => setPicked(plan.steps.map(() => ''))
  return (
    <Dialog label={t('carePlans.bookTitle')} className="receipt-dialog care-plan-dialog" onClose={onClose}>
      <form onSubmit={book}>
        <h2>{t('carePlans.bookTitle')}</h2>
        <p className="strong">{plan.title}</p>
        <p className="muted small">{t('carePlans.bookHint')}</p>
        <div className="field">
          <label htmlFor={`${id}-r`}>{t('carePlans.practitioner')}</label>
          <select id={`${id}-r`} value={role} onChange={(e) => { setRoleId(e.target.value); reset() }}>
            {members.map((m) => (
              <option key={m.roleId} value={m.roleId}>
                {m.name}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor={`${id}-d`}>{t('carePlans.firstDay')}</label>
          <input id={`${id}-d`} type="date" min={today} value={startsOn} onChange={(e) => { setStartsOn(e.target.value); reset() }} />
        </div>
        {slots.state.name === 'loading' && dated && <p className="muted">{t('common.loading')}</p>}
        {(slots.state.name === 'failed' || staff.state.name === 'failed') && <Alert message={t('carePlans.slotsFailed')} />}
        {slots.state.name === 'loaded' &&
          dated &&
          plan.steps.map((s, i) => {
            const day = dayIn(dayStart(startsOn, s.dayOffset, zone), zone)
            const times = free.filter((x) => sameDay(dayIn(new Date(x.start), zone), day) && modeOf(x) === s.visitMode && !picked.some((p, j) => j !== i && p === x.id))
            const label = stepText(t, f, s, i, services, startsOn, zone)
            if (times.length === 0) {
              return (
                <p key={s.id ?? i}>
                  <span>{label}</span> <span className="muted">{t('carePlans.noTimes')}</span>
                </p>
              )
            }
            return (
              <div className="field" key={s.id ?? i}>
                <label htmlFor={`${id}-t${i}`}>{label}</label>
                <select id={`${id}-t${i}`} value={picked[i]} onChange={(e) => setPicked((all) => all.map((p, j) => (j === i ? e.target.value : p)))}>
                  <option value="">{t('carePlans.chooseTime')}</option>
                  {times.map((x) => (
                    <option key={x.id} value={x.id}>
                      {f.time(new Date(x.start), zone)}–{f.time(new Date(x.end), zone)}
                    </option>
                  ))}
                </select>
              </div>
            )
          })}
        <Alert message={error} />
        <div className="actions">
          <Submit busy={busy} disabled={!ready} label={t('carePlans.bookEvery')} busyLabel={t('carePlans.booking')} />
          <button type="button" disabled={busy} onClick={onClose}>
            {t('carePlans.close')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

function modeOf(s: Slot): string {
  return s.extension?.find((x) => x.url === visitModeURL)?.valueCode ?? 'video'
}

// freeSlots is the practitioner role's free slots from from to to: the role's schedule, then its free slots.
async function freeSlots(fhir: FhirClient, roleId: string, from: Date, to: Date): Promise<Slot[]> {
  if (roleId === '') return []
  const schedules = await fhir.get<Bundle<Schedule>>(`Schedule?${new URLSearchParams({ actor: `PractitionerRole/${roleId}` })}`)
  const schedule = schedules.entry?.[0]?.resource
  if (!schedule) return []
  const query = new URLSearchParams([
    ['schedule', `Schedule/${schedule.id}`],
    ['status', 'free'],
    ['start', `ge${from.toISOString()}`],
    ['start', `lt${to.toISOString()}`],
  ])
  const slots = await fhir.get<Bundle<Slot>>(`Slot?${query}`)
  return (slots.entry ?? []).map((e) => e.resource)
}
