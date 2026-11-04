import { useState, type ReactNode } from 'react'
import type { Account } from './api/client'
import { participant, type Appointment, type Bundle, type FhirClient } from './api/fhir'
import { ChevronIcon } from './icons'
import { useLoad } from './load'
import { statusLabels } from './messages'
import { PractitionerPicker } from './PractitionerPicker'
import { loadStaff, loadTimeZone } from './staff'
import { addDays, dayIn, dayLabel, sameDay, startOfDay, timeOf, weekOf, type Day } from './time'
import { Alert, Link } from './ui'

const loadFailed = 'The calendar could not be loaded. Reload the page to try again.'

type Props = { fhir: FhirClient; account: Account; now?: () => Date }

// Calendar is a practitioner's week, Monday to Sunday in the clinic's time zone: their own, or, for an admin,
// the practitioner they pick.
export function Calendar({ fhir, account, now = () => new Date() }: Props) {
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
      <Panel>{setup.name === 'loading' ? <p className="muted">Loading</p> : <Alert message={loadFailed} />}</Panel>
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
          <button type="button" className="icon" aria-label="Previous week" title="Previous week" onClick={() => setShown(addDays(start, -7))}>
            <ChevronIcon direction="left" />
          </button>
          <button type="button" className="quiet" onClick={() => setShown(undefined)} disabled={sameDay(start, thisWeek)}>
            This week
          </button>
          <button type="button" className="icon" aria-label="Next week" title="Next week" onClick={() => setShown(addDays(start, 7))}>
            <ChevronIcon direction="right" />
          </button>
        </div>
        <p className="muted small">Times in {zone}</p>
      </div>
      {practitionerId ? (
        <Week fhir={fhir} practitionerId={practitionerId} start={start} zone={zone} now={now} />
      ) : (
        <p className="muted">No practitioners yet.</p>
      )}
    </Panel>
  )
}

function Panel({ children }: { children: ReactNode }) {
  return (
    <section className="panel calendar">
      <h1>Calendar</h1>
      {children}
    </section>
  )
}

type WeekProps = { fhir: FhirClient; practitionerId: string; start: Day; zone: string; now: () => Date }

function Week({ fhir, practitionerId, start, zone, now }: WeekProps) {
  const from = startOfDay(start, zone)
  const to = startOfDay(addDays(start, 7), zone)
  const { state } = useLoad(async () => {
    const query = new URLSearchParams({ practitioner: `Practitioner/${practitionerId}` })
    query.append('date', `ge${instant(from)}`)
    query.append('date', `lt${instant(to)}`)
    const bundle = await fhir.get<Bundle<Appointment>>(`Appointment?${query}`)
    return (bundle.entry ?? []).map((e) => e.resource).sort((a, b) => a.start.localeCompare(b.start))
  }, [fhir, practitionerId, from.getTime(), to.getTime()])

  if (state.name === 'loading') return <p className="muted">Loading</p>
  if (state.name === 'failed') return <Alert message={loadFailed} />

  const days = Array.from({ length: 7 }, (_, i) => addDays(start, i))
  const today = dayIn(now(), zone)
  return (
    <>
      {state.value.length === 0 && <p className="muted">No appointments this week.</p>}
      <div className="week">
        {days.map((day) => {
          const appts = state.value.filter((a) => sameDay(dayIn(new Date(a.start), zone), day))
          const label = dayLabel(day)
          const classes = ['day', appts.length === 0 && 'empty', sameDay(day, today) && 'today'].filter(Boolean).join(' ')
          return (
            <section key={label} className={classes} aria-label={label}>
              <h2>{label}</h2>
              {appts.map((a) => (
                <Card key={a.id} appt={a} zone={zone} now={now()} />
              ))}
            </section>
          )
        })}
      </div>
    </>
  )
}

function Card({ appt, zone, now }: { appt: Appointment; zone: string; now: Date }) {
  const start = new Date(appt.start)
  const end = new Date(appt.end)
  const current = appt.status === 'booked' && start <= now && now < end
  const name = participant(appt, 'Patient')?.display
  return (
    <Link to={`/appointments/${appt.id}`} className={`appt ${appt.status}${current ? ' current' : ''}`}>
      <span className="appt-time">
        {timeOf(start, zone)}–{timeOf(end, zone)}
      </span>
      {name && <span className="appt-name">{name}</span>}
      {appt.status !== 'booked' && <span className="tag">{statusLabels[appt.status]}</span>}
      {current && <span className="now">Now</span>}
    </Link>
  )
}

// instant is an RFC 3339 instant in UTC without fractions, the form FHIR searches take.
function instant(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z')
}
