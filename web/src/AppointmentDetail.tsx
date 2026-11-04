import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import { FhirError, nameOf, participant, type Appointment, type AppointmentStatus, type FhirClient, type Patient } from './api/fhir'
import { attempt, type Outcome } from './api/result'
import { ChevronIcon } from './icons'
import { useLoad } from './load'
import { somethingWrong, statusLabels } from './messages'
import { loadTimeZone } from './staff'
import { longDate, timeOf } from './time'
import { Alert, Link } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; account: Account; id: string; now?: () => Date }
type Detail = { zone: string; appt: Appointment; profile?: Patient }

// AppointmentDetail shows one appointment in the clinic's time zone. A practitioner sees the patient's profile
// when the Patient read answers, and otherwise the name the appointment carries, or nothing; an admin never
// has the profile (SPEC "FHIR reads and searches"), so it is not asked for. The practitioner records the
// outcome once the appointment has started; the practitioner and the admin can cancel it.
export function AppointmentDetail({ api, fhir, account, id, now = () => new Date() }: Props) {
  const organizationId = account.organizationId ?? ''
  const practitioner = account.role === 'practitioner'
  const { state, set } = useLoad(async (): Promise<Detail> => {
    const [zone, appt] = await Promise.all([loadTimeZone(fhir, organizationId), fhir.get<Appointment>(`Appointment/${encodeURIComponent(id)}`)])
    const patient = participant(appt, 'Patient')
    const profile = practitioner && patient ? await readPatient(fhir, patient.id) : undefined
    return { zone, appt, profile }
  }, [fhir, organizationId, id, practitioner])

  return (
    <section className="panel detail">
      <Link to="/" className="back">
        <ChevronIcon direction="left" />
        Calendar
      </Link>
      {state.name === 'loading' && <p className="muted">Loading</p>}
      {state.name === 'failed' && <Alert message={loadMessage(state.error)} />}
      {state.name === 'loaded' && (
        <Body
          {...state.value}
          api={api}
          practitioner={practitioner}
          now={now}
          onChanged={(status) => set({ ...state.value, appt: { ...state.value.appt, status } })}
        />
      )}
    </section>
  )
}

async function readPatient(fhir: FhirClient, id: string): Promise<Patient | undefined> {
  try {
    return await fhir.get<Patient>(`Patient/${encodeURIComponent(id)}`)
  } catch {
    return undefined // 404: not at full access; the name, if any, comes with the appointment
  }
}

function loadMessage(err: unknown): string {
  if (err instanceof FhirError && err.status === 404) return 'This appointment could not be found.'
  return 'The appointment could not be loaded. Reload the page to try again.'
}

type BodyProps = Detail & { api: ApiClient; practitioner: boolean; now: () => Date; onChanged: (s: AppointmentStatus) => void }

function Body({ zone, appt, profile, api, practitioner, now, onChanged }: BodyProps) {
  const start = new Date(appt.start)
  const end = new Date(appt.end)
  const name = profile ? nameOf(profile.name?.[0]) : participant(appt, 'Patient')?.display
  return (
    <>
      <h1>{name || 'Appointment'}</h1>
      <p className="when">
        <span>{`${longDate(start, zone)}, ${timeOf(start, zone)}–${timeOf(end, zone)}`}</span>
        <span className={`tag ${appt.status}`}>{statusLabels[appt.status]}</span>
      </p>
      {profile && <Profile patient={profile} />}
      {appt.status === 'booked' && <Actions api={api} id={appt.id} practitioner={practitioner} started={now() >= start} onChanged={onChanged} />}
    </>
  )
}

const birthDate = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', day: 'numeric', month: 'long', year: 'numeric' })

function Profile({ patient }: { patient: Patient }) {
  const phone = patient.telecom?.find((t) => t.system === 'phone')?.value
  return (
    <details className="more">
      <summary>Patient details</summary>
      <dl>
        {patient.birthDate && (
          <>
            <dt>Birth date</dt>
            <dd>{birthDate.format(new Date(`${patient.birthDate}T00:00:00Z`))}</dd>
          </>
        )}
        {phone && (
          <>
            <dt>Phone</dt>
            <dd>{phone}</dd>
          </>
        )}
      </dl>
    </details>
  )
}

type ActionsProps = { api: ApiClient; id: string; practitioner: boolean; started: boolean; onChanged: (s: AppointmentStatus) => void }

function Actions({ api, id, practitioner, started, onChanged }: ActionsProps) {
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string>()

  async function run(call: () => Promise<Outcome<{ status: AppointmentStatus }>>) {
    setBusy(true)
    setError(undefined)
    const res = await call()
    setBusy(false)
    setConfirming(false)
    if (res.ok) onChanged(res.data.status)
    else setError(actionMessage(res.code))
  }

  const params = { params: { path: { id } } }
  const setStatus = (status: 'fulfilled' | 'noshow') => run(() => attempt(api.POST('/appointments/{id}/status', { ...params, body: { status } })))
  const cancel = () => run(() => attempt(api.POST('/appointments/{id}/cancel', params)))

  return (
    <div className="appt-actions">
      <Alert message={error} />
      {practitioner && (
        <>
          <div className="actions">
            <button type="button" className="primary" disabled={busy || !started} onClick={() => setStatus('fulfilled')}>
              Mark fulfilled
            </button>
            <button type="button" disabled={busy || !started} onClick={() => setStatus('noshow')}>
              Mark no-show
            </button>
          </div>
          {!started && <p className="hint">You can record the outcome once the appointment has started.</p>}
        </>
      )}
      {confirming ? (
        <div className="confirm">
          <p>Cancel this appointment? The patient is told by e-mail, and the time becomes free again.</p>
          <div className="actions">
            <button type="button" className="danger" disabled={busy} onClick={cancel}>
              Yes, cancel
            </button>
            <button type="button" disabled={busy} onClick={() => setConfirming(false)}>
              Keep it
            </button>
          </div>
        </div>
      ) : (
        <div className="actions">
          <button type="button" className="quiet danger-text" disabled={busy} onClick={() => setConfirming(true)}>
            Cancel appointment
          </button>
        </div>
      )}
    </div>
  )
}

function actionMessage(code?: string): string {
  if (code === 'appointment_closed') return 'This appointment is already closed. Reload the page to see its status.'
  if (code === 'appointment_not_started') return 'The appointment has not started yet.'
  if (code === 'forbidden') return 'You cannot change this appointment.'
  if (code === 'not_found') return 'This appointment could not be found.'
  return somethingWrong
}
