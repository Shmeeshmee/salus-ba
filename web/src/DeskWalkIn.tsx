import { useId, useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import type { Bundle, Extension, FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { Dialog } from './chat/Dialog'
import { DeskConsent } from './DeskConsent'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { PatientPicker, type ClinicPatient } from './PatientPicker'
import type { Member } from './staff'
import { addDays, dayIn, startOfDay } from './time'
import { Alert, Submit } from './ui'
import { visitModeURL } from './visitMode'

type Appointment = components['schemas']['Appointment']

// The FHIR elements of a Schedule and a free Slot the walk-in reads (the directory's search, CapabilityStatement).
type Schedule = { resourceType: 'Schedule'; id: string }
type Slot = { resourceType: 'Slot'; id: string; start: string; end: string; extension?: Extension[] }

type Props = {
  api: ApiClient
  fhir: FhirClient
  staff: Member[]
  zone: string
  now: () => Date
  onBooked: (appointment: Appointment) => void
  onClose: () => void
}

// DeskWalkIn books a patient who walked in (clinic-cms F3): pick or register the patient, record the consent they give
// in person when the clinic has none active from them (an imported patient, or one who withdrew it: wave 4), then one
// of today's free times at the clinic with a practitioner, and the service, as staff (POST /clinic/appointments, F2).
export function DeskWalkIn({ api, fhir, staff, zone, now, onBooked, onClose }: Props) {
  const t = useT()
  const [patient, setPatient] = useState<ClinicPatient>()
  return (
    <Dialog label={t('desk.walkIn.title')} className="receipt-dialog desk-dialog" onClose={onClose}>
      <h2>{t('desk.walkIn.title')}</h2>
      {!patient && <PatientPicker api={api} onPick={setPatient} />}
      {patient && !patient.consentActive && (
        <section>
          <p>{t('desk.consent.needed')}</p>
          <DeskConsent api={api} patientId={patient.id} onRecorded={() => setPatient({ ...patient, consentActive: true })} />
          <button type="button" className="quiet" onClick={() => setPatient(undefined)}>
            {t('desk.walkIn.changePatient')}
          </button>
        </section>
      )}
      {patient?.consentActive && (
        <Booking api={api} fhir={fhir} staff={staff} zone={zone} now={now} patient={patient} onChangePatient={() => setPatient(undefined)} onBooked={onBooked} />
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          {t('desk.walkIn.cancel')}
        </button>
      </div>
    </Dialog>
  )
}

type BookingProps = Omit<Props, 'onClose'> & { patient: ClinicPatient; onChangePatient: () => void }

function Booking({ api, fhir, staff, zone, now, patient, onChangePatient, onBooked }: BookingProps) {
  const t = useT()
  const f = useFormat()
  const id = useId()
  const [roleId, setRoleId] = useState(staff[0]?.roleId ?? '')
  const [slotId, setSlotId] = useState('')
  const [serviceId, setServiceId] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const slots = useLoad(() => freeToday(fhir, roleId, now(), zone), [fhir, roleId, zone])
  const services = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/services'))
    if (!res.ok) throw new Error('services')
    return res.data.filter((s) => s.active && s.modes.includes('in-person'))
  }, [api])

  async function book(e: FormEvent) {
    e.preventDefault()
    if (busy || slotId === '') return
    setBusy(true)
    setError(undefined)
    const body = { patientId: patient.id, slotId, serviceId: serviceId || undefined }
    const res = await attempt(api.POST('/clinic/appointments', { body }))
    setBusy(false)
    if (res.ok) {
      onBooked(res.data)
      return
    }
    setError(bookMessage(t, res.code))
    if (res.code === 'slot_unavailable') {
      setSlotId('')
      slots.reload()
    }
  }

  return (
    <form onSubmit={book}>
      <div className="desk-patient">
        <span className="stack">
          <span className="strong">
            {patient.givenName} {patient.familyName}
          </span>
          <span className="muted small">{f.calendarDate(patient.birthDate)}</span>
        </span>
        <button type="button" className="quiet" onClick={onChangePatient}>
          {t('desk.walkIn.changePatient')}
        </button>
      </div>
      <div className="field">
        <label htmlFor={`${id}-p`}>{t('common.practitioner')}</label>
        <select
          id={`${id}-p`}
          value={roleId}
          onChange={(e) => {
            setRoleId(e.target.value)
            setSlotId('')
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
      {slots.state.name === 'loaded' && slots.state.value.length === 0 && <p className="muted">{t('desk.walkIn.noSlots')}</p>}
      {slots.state.name === 'loaded' && slots.state.value.length > 0 && (
        <div className="field">
          <label htmlFor={`${id}-s`}>{t('desk.walkIn.time')}</label>
          <select id={`${id}-s`} value={slotId} onChange={(e) => setSlotId(e.target.value)}>
            <option value="">{t('desk.walkIn.chooseTime')}</option>
            {slots.state.value.map((s) => (
              <option key={s.id} value={s.id}>
                {f.time(new Date(s.start), zone)}–{f.time(new Date(s.end), zone)}
              </option>
            ))}
          </select>
        </div>
      )}
      <div className="field">
        <label htmlFor={`${id}-v`}>{t('desk.walkIn.service')}</label>
        <select id={`${id}-v`} value={serviceId} onChange={(e) => setServiceId(e.target.value)}>
          <option value="">{t('desk.walkIn.noService')}</option>
          {(services.state.name === 'loaded' ? services.state.value : []).map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
      </div>
      <Alert message={error} />
      <Submit busy={busy} disabled={slotId === ''} label={t('desk.walkIn.book')} busyLabel={t('desk.walkIn.booking')} />
    </form>
  )
}

// freeToday is the practitioner role's free times at the clinic from now to the end of the clinic's day: the role's
// schedule, then its free slots, without the video ones (a walk-in is at the desk).
async function freeToday(fhir: FhirClient, roleId: string, from: Date, zone: string): Promise<Slot[]> {
  if (roleId === '') return []
  const schedules = await fhir.get<Bundle<Schedule>>(`Schedule?${new URLSearchParams({ actor: `PractitionerRole/${roleId}` })}`)
  const schedule = schedules.entry?.[0]?.resource
  if (!schedule) return []
  const to = startOfDay(addDays(dayIn(from, zone), 1), zone)
  const query = new URLSearchParams([
    ['schedule', `Schedule/${schedule.id}`],
    ['status', 'free'],
    ['start', `ge${from.toISOString()}`],
    ['start', `lt${to.toISOString()}`],
  ])
  const slots = await fhir.get<Bundle<Slot>>(`Slot?${query}`)
  return (slots.entry ?? []).map((e) => e.resource).filter((s) => s.extension?.find((x) => x.url === visitModeURL)?.valueCode === 'in-person')
}

function bookMessage(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'slot_unavailable') return t('desk.walkIn.slotTaken')
  if (code === 'consent_required') return t('desk.walkIn.consentRequired')
  if (code === 'invalid_service') return t('desk.walkIn.invalidService')
  return t('desk.walkIn.failed')
}
