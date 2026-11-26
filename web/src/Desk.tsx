import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { DeskCheckIn } from './DeskCheckIn'
import { DeskMove } from './DeskMove'
import { DeskPatient } from './DeskPatient'
import { DeskPatients } from './DeskPatients'
import { DeskWalkIn } from './DeskWalkIn'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { groupPhone } from './phone'
import { ReceiptDialog } from './ReceiptDialog'
import { VisitReceipts } from './VisitActions'
import { mergeReceipts, type VisitReceipt } from './visitReceipts'
import { loadStaff, loadTimeZone, type Member } from './staff'
import { startOfDay } from './time'
import { Alert, Notice } from './ui'

type DeskDay = components['schemas']['DeskDay']
type DeskVisit = components['schemas']['DeskVisit']
type Location = components['schemas']['Location']

type Props = { api: ApiClient; fhir: FhirClient; account: Account; now?: () => Date }

// Desk is the front desk (clinic-cms F3), for the clinic's admin and its receptionists: today's visits by practitioner
// and time with the patient, visit type and service; "Arrived" for an in-person visit; a walk-in; and the receipt from
// the row (C's dialog, started from the service the booking named, E). Nothing clinical is on it (W2).
export function Desk({ api, fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state } = useLoad(async () => {
    const [zone, staff, locations] = await Promise.all([loadTimeZone(fhir, organizationId), loadStaff(fhir, organizationId), attempt(api.GET('/clinic/locations'))])
    // Every location, those taken out of use too: a visit booked at one before still names it (RD6).
    return { zone, staff, locations: locations.ok ? locations.data : [] }
  }, [api, fhir, organizationId])
  if (state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('desk.loadFailed')} />
  return <Day api={api} fhir={fhir} zone={state.value.zone} staff={state.value.staff} locations={state.value.locations} now={now} />
}

type DayProps = { api: ApiClient; fhir: FhirClient; zone: string; staff: Member[]; locations: Location[]; now: () => Date }

function Day({ api, fhir, zone, staff, locations, now }: DayProps) {
  const t = useT()
  const f = useFormat()
  const [walkIn, setWalkIn] = useState(false)
  const [patients, setPatients] = useState(false)
  const [notice, setNotice] = useState<string>()
  const [locationId, setLocationId] = useState('') // '' for every location
  // The filter by location is offered only with two locations in use (LC2); without them, the desk is the day's.
  const inUse = locations.filter((l) => l.active)
  // While every location is shown, each in-person visit names its own when the clinic has two in use (U2), and always
  // when its location was taken out of use since, marked so (RD6); a picked location needs no name.
  const placeOf = (id = ''): string | undefined => {
    const l = locations.find((x) => x.id === id)
    if (!l || locationId) return undefined
    if (!l.active) return t('locations.placeNotInUse', { name: l.name })
    return inUse.length > 1 ? l.name : undefined
  }
  const { state, reload, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/desk/today', { params: { query: locationId ? { locationId } : {} } }))
    if (!res.ok) throw new Error('desk')
    return res.data
  }, [api, locationId])
  const names = new Map(staff.map((m) => [m.roleId, m.name]))
  const day = state.name === 'loaded' ? state.value : undefined

  const arrived = (visits: DeskDay, appointment: DeskVisit['appointment']) =>
    set({ ...visits, visits: visits.visits.map((v) => (v.appointment.id === appointment.id ? { ...v, appointment } : v)) })

  return (
    <section className="panel desk">
      <h1>{t('desk.title')}</h1>
      <div className="toolbar">
        {day && <h2>{f.date(startOfDay(calendarDay(day.date), zone), zone, 'long')}</h2>}
        {inUse.length > 1 && (
          <select className="desk-location" aria-label={t('locations.filter')} value={locationId} onChange={(e) => setLocationId(e.target.value)}>
            <option value="">{t('locations.allLocations')}</option>
            {inUse.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </select>
        )}
        <span className="actions">
          <button type="button" onClick={() => setPatients(true)}>
            {t('desk.patients.title')}
          </button>
          <button type="button" className="primary" onClick={() => setWalkIn(true)}>
            {t('desk.walkIn.open')}
          </button>
        </span>
      </div>
      <Notice message={notice} />
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('desk.dayFailed')} />}
      {day && day.visits.length === 0 && <p className="muted">{t('desk.empty')}</p>}
      {day &&
        byPractitioner(day.visits).map(([roleId, visits]) => {
          const name = names.get(roleId) ?? t('desk.former')
          return (
            <div key={roleId} className="desk-practitioner">
              <h3>{name}</h3>
              <ol className="rows" aria-label={name}>
                {visits.map((v) => (
                  <Row
                    key={v.appointment.id}
                    api={api}
                    visit={v}
                    zone={zone}
                    place={placeOf(v.appointment.locationId)}
                    move={{ fhir, staff: staff.filter((m) => m.active), now }}
                    onMoved={() => {
                      setNotice(t('desk.move.moved'))
                      reload()
                    }}
                    onArrived={(a) => arrived(day, a)}
                    onIssued={(number) => setNotice(t('receipts.issued', { number }))}
                  />
                ))}
              </ol>
            </div>
          )
        })}
      {patients && <DeskPatients api={api} onChanged={reload} onClose={() => setPatients(false)} />}
      {walkIn && (
        <DeskWalkIn
          api={api}
          fhir={fhir}
          staff={staff.filter((m) => m.active)}
          zone={zone}
          now={now}
          onClose={() => setWalkIn(false)}
          onBooked={() => {
            setWalkIn(false)
            setNotice(t('desk.walkIn.booked'))
            reload()
          }}
        />
      )}
    </section>
  )
}

type RowProps = {
  api: ApiClient
  visit: DeskVisit
  zone: string
  place?: string // the in-person visit's location, named while the desk shows every location
  move: { fhir: FhirClient; staff: Member[]; now: () => Date } // what moving the visit needs (M1)
  onMoved: () => void // the day is read again: after a move, or a change on the patient's card
  onArrived: (a: DeskVisit['appointment']) => void
  onIssued: (number: number) => void
}

function Row({ api, visit, zone, place, move, onMoved, onArrived, onIssued }: RowProps) {
  const t = useT()
  const f = useFormat()
  const [issuing, setIssuing] = useState(false)
  const [moving, setMoving] = useState(false)
  const [issued, setIssued] = useState<VisitReceipt[]>([])
  const { appointment: a, patient: p } = visit
  const receipts = mergeReceipts(visit.receipts, issued)
  const active = receipts.filter((r) => !r.reversed)
  const status = a.status === 'arrived' && a.arrivedAt ? t('desk.arrivedAt', { time: f.time(new Date(a.arrivedAt), zone) }) : t(`desk.status.${a.status}`)
  const details = [f.calendarDate(p.birthDate), p.phone && groupPhone(p.phone), t(a.visitMode === 'in-person' ? 'desk.inPerson' : 'desk.video'), place, visit.serviceName].filter(Boolean)
  return (
    <li>
      <span className="stack">
        <span>
          <span className="strong">
            {f.time(new Date(a.start), zone)}–{f.time(new Date(a.end), zone)}
          </span>{' '}
          <DeskPatient api={api} patient={p} onChanged={onMoved} />{' '}
          <span className={`tag desk-${a.status}`}>{status}</span>
        </span>
        <span className="muted small">{details.join(' · ')}</span>
        {a.status === 'booked' && visit.reminder !== undefined && <span className="muted small">{reminderText(t, visit)}</span>}
        <VisitReceipts receipts={receipts} />
      </span>
      <span className="actions">
        {a.status === 'booked' && a.visitMode === 'in-person' && <DeskCheckIn api={api} appointmentId={a.id} onArrived={onArrived} />}
        {a.status === 'booked' && (
          <button type="button" onClick={() => setMoving(true)}>
            {t('desk.move.open')}
          </button>
        )}
        {(a.status === 'booked' || a.status === 'arrived' || a.status === 'fulfilled') && (
          <button type="button" onClick={() => setIssuing(true)}>
            {t(active.length > 0 ? 'receipts.issueAnother' : 'receipts.dialog.issue')}
          </button>
        )}
      </span>
      {moving && (
        <DeskMove
          api={api}
          {...move}
          zone={zone}
          visit={visit}
          onClose={() => setMoving(false)}
          onMoved={() => {
            setMoving(false)
            onMoved()
          }}
        />
      )}
      {issuing && (
        <ReceiptDialog
          api={api}
          appointmentId={a.id}
          serviceId={a.serviceId}
          issued={active.map((r) => r.number)}
          onClose={() => setIssuing(false)}
          onIssued={(r) => {
            setIssuing(false)
            setIssued((all) => [...all, { id: r.id, number: r.number, reversed: false }])
            onIssued(r.number)
          }}
        />
      )}
    </li>
  )
}

// reminderText says whether a reminder will go for the visit, and why not (clinic-cms R3: a patient without an account
// is reminded by SMS only, while the clinic sends SMS and the profile has a phone).
function reminderText(t: ReturnType<typeof useT>, visit: DeskVisit): string {
  if (visit.reminder) return t('desk.reminder.on')
  return t(visit.reminderOff === 'no_phone' ? 'desk.reminder.noPhone' : 'desk.reminder.smsOff')
}

// byPractitioner groups the day's visits by practitioner role in the order the server gives them (by practitioner,
// then start).
function byPractitioner(visits: DeskVisit[]): [string, DeskVisit[]][] {
  const groups = new Map<string, DeskVisit[]>()
  for (const v of visits) groups.set(v.appointment.practitionerRoleId, [...(groups.get(v.appointment.practitionerRoleId) ?? []), v])
  return [...groups]
}

function calendarDay(date: string) {
  const [year, month, day] = date.split('-').map(Number)
  return { year, month, day }
}
