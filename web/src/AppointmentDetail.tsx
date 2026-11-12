import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import { FhirError, nameOf, participant, type Appointment, type AppointmentStatus, type FhirClient, type Patient } from './api/fhir'
import { attempt, type Outcome } from './api/result'
import type { components } from './api/schema'
import { AppointmentQuestionnaires } from './AppointmentQuestionnaires'
import { PatientHistory } from './History'
import { joinWindow, useNow } from './consultation'
import { useFormat, useT } from './i18n/useT'
import { ChevronIcon } from './icons'
import { useLoad } from './load'
import { navigate } from './route'
import { loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type T = ReturnType<typeof useT>

type Props = { api: ApiClient; fhir: FhirClient; account: Account; id: string; now?: () => Date }
type ConsultationStatus = components['schemas']['ConsultationStatus']
type Detail = { zone: string; appt: Appointment; profile?: Patient; consultation?: ConsultationStatus }

// AppointmentDetail shows one appointment in the clinic's time zone. A practitioner sees the patient's profile
// when the Patient read answers, and otherwise the name the appointment carries, or nothing; an admin never
// has the profile (SPEC "FHIR reads and searches"), so it is not asked for. The practitioner records the
// outcome once the appointment has started; the practitioner and the admin can cancel it. The practitioner joins
// the video consultation inside its window (an admin cannot join), and both see whether each side joined.
// Both see the appointment's questionnaire requests; only the appointment's own practitioner changes them. Its own
// practitioner also sees the patient's history and the pre-visit summary (SAL-40).
// Everything above the questionnaires is read in one load, so the page appears in one piece: a section that came
// in on its own would push down what is already shown, under a click on its way.
export function AppointmentDetail({ api, fhir, account, id, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const practitioner = account.role === 'practitioner'
  const { state, set } = useLoad(async (): Promise<Detail> => {
    const [zone, appt, consultation] = await Promise.all([
      loadTimeZone(fhir, organizationId),
      fhir.get<Appointment>(`Appointment/${encodeURIComponent(id)}`),
      readConsultation(api, id),
    ])
    const patient = participant(appt, 'Patient')
    const profile = practitioner && patient ? await readPatient(fhir, patient.id) : undefined
    return { zone, appt, profile, consultation }
  }, [api, fhir, organizationId, id, practitioner])

  return (
    <section className="panel detail">
      <Link to="/" className="back">
        <ChevronIcon direction="left" />
        {t('calendar.title')}
      </Link>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={loadMessage(t, state.error)} />}
      {state.name === 'loaded' && (
        <Body
          {...state.value}
          api={api}
          fhir={fhir}
          practitioner={practitioner}
          own={practitioner && participant(state.value.appt, 'Practitioner')?.id === account.practitionerId}
          admin={account.role === 'admin'}
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

// readConsultation is the consultation's status, or nothing when it cannot be read (the section is left out).
async function readConsultation(api: ApiClient, id: string): Promise<ConsultationStatus | undefined> {
  const res = await attempt(api.GET('/appointments/{id}/consultation', { params: { path: { id } } }))
  return res.ok ? res.data : undefined
}

function loadMessage(t: T, err: unknown): string {
  if (err instanceof FhirError && err.status === 404) return t('calendar.notFound')
  return t('calendar.appointmentLoadFailed')
}

type BodyProps = Detail & {
  api: ApiClient
  fhir: FhirClient
  practitioner: boolean
  own: boolean
  admin: boolean
  now: () => Date
  onChanged: (s: AppointmentStatus) => void
}

function Body({ zone, appt, profile, consultation, api, fhir, practitioner, own, admin, now, onChanged }: BodyProps) {
  const t = useT()
  const f = useFormat()
  const at = useNow(now)
  const join = joinWindow(appt, at)
  const start = new Date(appt.start)
  const end = new Date(appt.end)
  const name = profile ? nameOf(profile.name?.[0]) : participant(appt, 'Patient')?.display
  const patientId = participant(appt, 'Patient')?.id
  return (
    <>
      <h1>{name || t('calendar.appointment')}</h1>
      <p className="when">
        <span>{t('calendar.when', { date: f.date(start, zone, 'long'), start: f.time(start, zone), end: f.time(end, zone) })}</span>
        <span className={`tag ${appt.status}`}>{t(`calendar.status.${appt.status}`)}</span>
      </p>
      {profile && <Profile patient={profile} />}
      {practitioner && join === 'open' && (
        <div className="actions join">
          <button type="button" className="primary" onClick={() => navigate(`/appointments/${appt.id}/call`)}>
            {t('calendar.join')}
          </button>
        </div>
      )}
      {practitioner && join === 'before' && <p className="hint join">{t('calendar.joinFrom')}</p>}
      {consultation && <Consultation status={consultation} zone={zone} />}
      <AppointmentQuestionnaires api={api} fhir={fhir} appt={appt} zone={zone} own={own} admin={admin} now={at} />
      {own && patientId && <PatientHistory api={api} fhir={fhir} patientId={patientId} appointmentId={appt.id} zone={zone} />}
      {appt.status === 'booked' && <Actions api={api} id={appt.id} practitioner={practitioner} started={at >= start} onChanged={onChanged} />}
    </>
  )
}

// Consultation is whether the video consultation started, and whether each side joined (SPEC §2 "Consultation
// status"). A side that has not joined is named only once the consultation has ended. Left out when the status
// cannot be read.
function Consultation({ status, zone }: { status: ConsultationStatus; zone: string }) {
  const t = useT()
  const f = useFormat()
  const ended = status.state === 'ended'
  const joined = (at: string | undefined, who: 'patient' | 'practitioner') => {
    if (at) return <li>{t(`calendar.${who}JoinedAt`, { time: f.time(new Date(at), zone) })}</li>
    return ended ? <li>{t(`calendar.${who}DidNotJoin`)}</li> : null
  }
  return (
    <section className="consultation" aria-label={t('calendar.consultation')}>
      <h2>{t('calendar.consultation')}</h2>
      <ul>
        <li>{t(`calendar.consultationState.${status.state}`)}</li>
        {joined(status.patientJoinedAt, 'patient')}
        {joined(status.practitionerJoinedAt, 'practitioner')}
      </ul>
    </section>
  )
}

function Profile({ patient }: { patient: Patient }) {
  const t = useT()
  const f = useFormat()
  const phone = patient.telecom?.find((c) => c.system === 'phone')?.value
  return (
    <details className="more">
      <summary>{t('calendar.patientDetails')}</summary>
      <dl>
        {patient.birthDate && (
          <>
            <dt>{t('calendar.birthDate')}</dt>
            <dd>{f.calendarDate(patient.birthDate)}</dd>
          </>
        )}
        {phone && (
          <>
            <dt>{t('calendar.phone')}</dt>
            <dd>{phone}</dd>
          </>
        )}
      </dl>
    </details>
  )
}

type ActionsProps = { api: ApiClient; id: string; practitioner: boolean; started: boolean; onChanged: (s: AppointmentStatus) => void }

function Actions({ api, id, practitioner, started, onChanged }: ActionsProps) {
  const t = useT()
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
    else setError(actionMessage(t, res.code))
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
              {t('calendar.markFulfilled')}
            </button>
            <button type="button" disabled={busy || !started} onClick={() => setStatus('noshow')}>
              {t('calendar.markNoShow')}
            </button>
          </div>
          {!started && <p className="hint">{t('calendar.outcomeHint')}</p>}
        </>
      )}
      {confirming ? (
        <div className="confirm">
          <p>{t('calendar.cancelQuestion')}</p>
          <div className="actions">
            <button type="button" className="danger" disabled={busy} onClick={cancel}>
              {t('calendar.yesCancel')}
            </button>
            <button type="button" disabled={busy} onClick={() => setConfirming(false)}>
              {t('calendar.keepIt')}
            </button>
          </div>
        </div>
      ) : (
        <div className="actions">
          <button type="button" className="quiet danger-text" disabled={busy} onClick={() => setConfirming(true)}>
            {t('calendar.cancelAppointment')}
          </button>
        </div>
      )}
    </div>
  )
}

function actionMessage(t: T, code?: string): string {
  if (code === 'appointment_closed') return t('calendar.closed')
  if (code === 'appointment_not_started') return t('calendar.notStarted')
  if (code === 'forbidden') return t('calendar.forbidden')
  if (code === 'not_found') return t('calendar.notFound')
  return t('common.somethingWrong')
}
