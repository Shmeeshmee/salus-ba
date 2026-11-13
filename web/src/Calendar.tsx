import { useState, type ReactNode } from 'react'
import type { Account } from './api/client'
import { participant, type Appointment, type Bundle, type FhirClient } from './api/fhir'
import { joinWindow, useNow } from './consultation'
import { useFormat, useT } from './i18n/useT'
import { ChevronIcon } from './icons'
import { useLoad } from './load'
import { PractitionerPicker } from './PractitionerPicker'
import { navigate } from './route'
import { loadStaff, loadTimeZone } from './staff'
import { addDays, dayIn, sameDay, startOfDay, weekOf, type Day } from './time'
import { Alert, Link } from './ui'

type Props = { fhir: FhirClient; account: Account; now?: () => Date }

// Calendar is a practitioner's week, Monday to Sunday in the clinic's time zone: their own, or, for an admin,
// the practitioner they pick.
export function Calendar({ fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const admin = account.role === 'admin'
  const { state: setup } = useLoad(async () => {
    const [zone, staff] = await Promise.all([loadTimeZone(fhir, organizationId), admin ? loadStaff(fhir, organizationId) : []])
    return { zone, staff: staff.filter((m) => m.active) }
  }, [fhir, organizationId, admin])
  const [chosen, setChosen] = useState<string>()
  const [shown, setShown] = useState<Day>()

  if (setup.name !== 'loaded') {
    return (
      <Panel>{setup.name === 'loading' ? <p className="muted">{t('common.loading')}</p> : <Alert message={t('calendar.loadFailed')} />}</Panel>
    )
  }

  const { zone, staff } = setup.value
  const practitionerId = admin ? (chosen ?? staff[0]?.practitionerId) : account.practitionerId
  const thisWeek = weekOf(dayIn(now(), zone))
  const start = shown ?? thisWeek

  return (
    <Panel>
      <div className="toolbar">
        {admin && staff.length > 0 && <PractitionerPicker staff={staff} value={practitionerId ?? ''} onChange={setChosen} />}
        <div className="week-nav">
          <button
            type="button"
            className="icon"
            aria-label={t('calendar.previousWeek')}
            title={t('calendar.previousWeek')}
            onClick={() => setShown(addDays(start, -7))}
          >
            <ChevronIcon direction="left" />
          </button>
          <button type="button" className="quiet" onClick={() => setShown(undefined)} disabled={sameDay(start, thisWeek)}>
            {t('calendar.thisWeek')}
          </button>
          <button type="button" className="icon" aria-label={t('calendar.nextWeek')} title={t('calendar.nextWeek')} onClick={() => setShown(addDays(start, 7))}>
            <ChevronIcon direction="right" />
          </button>
        </div>
        <p className="muted small">{t('calendar.timesIn', { zone })}</p>
      </div>
      {practitionerId ? (
        <Week fhir={fhir} practitionerId={practitionerId} start={start} zone={zone} now={now} joins={account.role === 'practitioner'} />
      ) : (
        <p className="muted">{t('common.noPractitioners')}</p>
      )}
    </Panel>
  )
}

function Panel({ children }: { children: ReactNode }) {
  const t = useT()
  return (
    <section className="panel calendar">
      <h1>{t('calendar.title')}</h1>
      {children}
    </section>
  )
}

// joins: the signed-in practitioner sees the join button on their own appointments; an admin cannot join.
type WeekProps = { fhir: FhirClient; practitionerId: string; start: Day; zone: string; now: () => Date; joins: boolean }

function Week({ fhir, practitionerId, start, zone, now, joins }: WeekProps) {
  const t = useT()
  const f = useFormat()
  const from = startOfDay(start, zone)
  const to = startOfDay(addDays(start, 7), zone)
  const { state } = useLoad(async () => {
    const query = new URLSearchParams({ practitioner: `Practitioner/${practitionerId}` })
    query.append('date', `ge${instant(from)}`)
    query.append('date', `lt${instant(to)}`)
    const bundle = await fhir.get<Bundle<Appointment>>(`Appointment?${query}`)
    return (bundle.entry ?? []).map((e) => e.resource).sort((a, b) => a.start.localeCompare(b.start))
  }, [fhir, practitionerId, from.getTime(), to.getTime()])
  const at = useNow(now)

  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('calendar.loadFailed')} />

  const days = Array.from({ length: 7 }, (_, i) => addDays(start, i))
  const today = dayIn(at, zone)
  return (
    <>
      {state.value.length === 0 && <p className="muted">{t('calendar.noAppointments')}</p>}
      <div className="week">
        {days.map((day) => {
          const appts = state.value.filter((a) => sameDay(dayIn(new Date(a.start), zone), day))
          const label = f.date(startOfDay(day, zone), zone, 'short')
          const classes = ['day', appts.length === 0 && 'empty', sameDay(day, today) && 'today'].filter(Boolean).join(' ')
          return (
            <section key={`${day.year}-${day.month}-${day.day}`} className={classes} aria-label={label}>
              <h2>{label}</h2>
              {appts.map((a) => (
                <Card key={a.id} appt={a} zone={zone} now={at} joins={joins} />
              ))}
            </section>
          )
        })}
      </div>
    </>
  )
}

function Card({ appt, zone, now, joins }: { appt: Appointment; zone: string; now: Date; joins: boolean }) {
  const t = useT()
  const f = useFormat()
  const start = new Date(appt.start)
  const end = new Date(appt.end)
  const current = appt.status === 'booked' && start <= now && now < end
  const name = participant(appt, 'Patient')?.display
  const card = (
    <Link to={`/appointments/${appt.id}`} className={`appt ${appt.status}${current ? ' current' : ''}`}>
      <span className="appt-time">
        {f.time(start, zone)}–{f.time(end, zone)}
      </span>
      {name && <span className="appt-name">{name}</span>}
      {appt.status !== 'booked' && <span className="tag">{t(`calendar.status.${appt.status}`)}</span>}
      <ConfirmationTag appt={appt} />
      {current && <span className="now">{t('calendar.now')}</span>}
    </Link>
  )
  if (!joins || joinWindow(appt, now) !== 'open') return card
  return (
    <div className="appt-join">
      {card}
      <button type="button" className="primary" onClick={() => navigate(`/appointments/${appt.id}/call`)}>
        {t('calendar.join')}
      </button>
    </div>
  )
}

// confirmation is whether the patient of a booked appointment confirmed they will come: their participant is
// accepted once confirmed and tentative until then (ADR 0015); nothing for an appointment no longer booked.
function confirmation(appt: Appointment): 'confirmed' | 'unconfirmed' | undefined {
  if (appt.status !== 'booked') return undefined
  const status = appt.participant.find((p) => p.actor.reference.startsWith('Patient/'))?.status
  if (status === 'accepted') return 'confirmed'
  if (status === 'tentative') return 'unconfirmed'
  return undefined
}

// ConfirmationTag is the badge of confirmation(appt), or nothing.
export function ConfirmationTag({ appt }: { appt: Appointment }) {
  const t = useT()
  const c = confirmation(appt)
  if (!c) return null
  return <span className={`tag ${c}`}>{t(c === 'confirmed' ? 'appointments.confirmed' : 'appointments.notConfirmed')}</span>
}

// instant is an RFC 3339 instant in UTC without fractions, the form FHIR searches take.
function instant(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z')
}
