import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { DeskCheckIn } from './DeskCheckIn'
import { DeskMove } from './DeskMove'
import { DeskPatient } from './DeskPatient'
import { DeskWalkIn } from './DeskWalkIn'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { ReceiptDialog } from './ReceiptDialog'
import { VisitReceipts } from './VisitActions'
import { mergeReceipts, type VisitReceipt } from './visitReceipts'
import { loadStaff, loadTimeZone, type Member } from './staff'
import { startOfDay } from './time'
import { Alert, Notice } from './ui'

type DeskDay = components['schemas']['DeskDay']
type DeskVisit = components['schemas']['DeskVisit']

type Props = { api: ApiClient; fhir: FhirClient; account: Account; now?: () => Date }

// Desk is the front desk (clinic-cms F3), for the clinic's admin and its receptionists: today's visits by practitioner
// and time with the patient, visit type and service; "Arrived" for an in-person visit; a walk-in; and the receipt from
// the row (C's dialog, started from the service the booking named, E). Nothing clinical is on it (W2).
export function Desk({ api, fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state } = useLoad(async () => {
    const [zone, staff] = await Promise.all([loadTimeZone(fhir, organizationId), loadStaff(fhir, organizationId)])
    return { zone, staff }
  }, [fhir, organizationId])
  if (state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('desk.loadFailed')} />
  return <Day api={api} fhir={fhir} zone={state.value.zone} staff={state.value.staff} now={now} />
}

type DayProps = { api: ApiClient; fhir: FhirClient; zone: string; staff: Member[]; now: () => Date }

function Day({ api, fhir, zone, staff, now }: DayProps) {
  const t = useT()
  const f = useFormat()
  const [walkIn, setWalkIn] = useState(false)
  const [notice, setNotice] = useState<string>()
  const { state, reload, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/desk/today'))
    if (!res.ok) throw new Error('desk')
    return res.data
  }, [api])
  const names = new Map(staff.map((m) => [m.roleId, m.name]))
  const day = state.name === 'loaded' ? state.value : undefined

  const arrived = (visits: DeskDay, appointment: DeskVisit['appointment']) =>
    set({ ...visits, visits: visits.visits.map((v) => (v.appointment.id === appointment.id ? { ...v, appointment } : v)) })

  return (
    <section className="panel desk">
      <h1>{t('desk.title')}</h1>
      <div className="toolbar">
        {day && <h2>{f.date(startOfDay(calendarDay(day.date), zone), zone, 'long')}</h2>}
        <button type="button" className="primary" onClick={() => setWalkIn(true)}>
          {t('desk.walkIn.open')}
        </button>
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
  move: { fhir: FhirClient; staff: Member[]; now: () => Date } // what moving the visit needs (M1)
  onMoved: () => void
  onArrived: (a: DeskVisit['appointment']) => void
  onIssued: (number: number) => void
}

function Row({ api, visit, zone, move, onMoved, onArrived, onIssued }: RowProps) {
  const t = useT()
  const f = useFormat()
  const [issuing, setIssuing] = useState(false)
  const [moving, setMoving] = useState(false)
  const [issued, setIssued] = useState<VisitReceipt[]>([])
  const { appointment: a, patient: p } = visit
  const receipts = mergeReceipts(visit.receipts, issued)
  const active = receipts.filter((r) => !r.reversed)
  const status = a.status === 'arrived' && a.arrivedAt ? t('desk.arrivedAt', { time: f.time(new Date(a.arrivedAt), zone) }) : t(`desk.status.${a.status}`)
  const details = [f.calendarDate(p.birthDate), p.phone, t(a.visitMode === 'in-person' ? 'desk.inPerson' : 'desk.video'), visit.serviceName].filter(Boolean)
  return (
    <li>
      <span className="stack">
        <span>
          <span className="strong">
            {f.time(new Date(a.start), zone)}–{f.time(new Date(a.end), zone)}
          </span>{' '}
          <DeskPatient api={api} patient={p} />{' '}
          <span className={`tag desk-${a.status}`}>{status}</span>
        </span>
        <span className="muted small">{details.join(' · ')}</span>
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
