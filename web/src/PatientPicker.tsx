import { useId, useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { Alert, Field, Submit } from './ui'

export type ClinicPatient = components['schemas']['ClinicPatient']

type Found = { name: 'idle' } | { name: 'searching' } | { name: 'failed' } | { name: 'found'; patients: ClinicPatient[] }

// PatientPicker finds a patient of the clinic for the desk (clinic-cms F3): by name and, when given, birth date, or
// registers one who has no Salus account, with the care consent the patient gave at the desk (W1). It shows what the
// desk needs to tell two patients apart (W2) and nothing clinical.
export function PatientPicker({ api, onPick }: { api: ApiClient; onPick: (patient: ClinicPatient) => void }) {
  const t = useT()
  const f = useFormat()
  const [name, setName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [found, setFound] = useState<Found>({ name: 'idle' })
  const [registering, setRegistering] = useState(false)
  const query = name.trim()

  async function search(e: FormEvent) {
    e.preventDefault()
    if (query.length < 2) return
    setFound({ name: 'searching' })
    const res = await attempt(api.GET('/clinic/patients', { params: { query: { name: query, birthDate: birthDate || undefined } } }))
    setFound(res.ok ? { name: 'found', patients: res.data } : { name: 'failed' })
  }

  return (
    <div className="patient-picker">
      <form className="toolbar" role="search" onSubmit={search}>
        <Field label={t('desk.picker.name')} value={name} maxLength={200} autoComplete="off" onChange={(e) => setName(e.target.value)} />
        <Field label={t('desk.picker.birthDateOptional')} type="date" value={birthDate} onChange={(e) => setBirthDate(e.target.value)} />
        <Submit busy={found.name === 'searching'} disabled={query.length < 2} label={t('desk.picker.search')} busyLabel={t('desk.picker.searching')} />
      </form>
      {found.name === 'failed' && <Alert message={t('desk.picker.searchFailed')} />}
      {found.name === 'found' && found.patients.length === 0 && <p className="muted">{t('desk.picker.none')}</p>}
      {found.name === 'found' && found.patients.length > 0 && (
        <ul className="rows" aria-label={t('desk.picker.found')}>
          {found.patients.map((p) => (
            <li key={p.id}>
              <button type="button" className="quiet patient-option" onClick={() => onPick(p)}>
                <span>
                  {p.givenName} {p.familyName}
                </span>
                <span className="muted small">{f.calendarDate(p.birthDate)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {found.name === 'found' && !registering && (
        <button type="button" onClick={() => setRegistering(true)}>
          {t('desk.picker.register')}
        </button>
      )}
      {registering && <Register api={api} onRegistered={onPick} />}
    </div>
  )
}

// Register records a patient the clinic registers itself (POST /clinic/patients): name, birth date and, when they
// give one, a phone; the profile is the clinic's only, and the consent is recorded as given in person by the caller.
function Register({ api, onRegistered }: { api: ApiClient; onRegistered: (patient: ClinicPatient) => void }) {
  const t = useT()
  const id = useId()
  const [givenName, setGivenName] = useState('')
  const [familyName, setFamilyName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [phone, setPhone] = useState('')
  const [consent, setConsent] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const valid = givenName.trim() !== '' && familyName.trim() !== '' && birthDate !== '' && consent

  async function register(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(undefined)
    const body = { givenName: givenName.trim(), familyName: familyName.trim(), birthDate, phone: phone.replace(/\s/g, '') || undefined, consentGivenInPerson: true }
    const res = await attempt(api.POST('/clinic/patients', { body }))
    setBusy(false)
    if (res.ok) onRegistered(res.data)
    else setError(registerMessage(t, res.code))
  }

  return (
    <form onSubmit={register}>
      <fieldset className="patient-register">
        <legend>{t('desk.register.title')}</legend>
        <Field label={t('desk.register.givenName')} value={givenName} maxLength={200} required onChange={(e) => setGivenName(e.target.value)} />
        <Field label={t('desk.register.familyName')} value={familyName} maxLength={200} required onChange={(e) => setFamilyName(e.target.value)} />
        <Field label={t('desk.register.birthDate')} type="date" value={birthDate} required onChange={(e) => setBirthDate(e.target.value)} />
        <Field label={t('desk.register.phone')} type="tel" value={phone} autoComplete="off" onChange={(e) => setPhone(e.target.value)} />
        <div className="check">
          <input id={`${id}-consent`} type="checkbox" checked={consent} onChange={(e) => setConsent(e.target.checked)} />
          <label htmlFor={`${id}-consent`}>{t('desk.register.consent')}</label>
        </div>
        <Alert message={error} />
        <Submit busy={busy} disabled={!valid} label={t('desk.register.submit')} busyLabel={t('desk.register.submitting')} />
      </fieldset>
    </form>
  )
}

function registerMessage(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'invalid_phone') return t('desk.register.invalidPhone')
  if (code === 'invalid_birth_date') return t('desk.register.invalidBirthDate')
  if (code === 'invalid_name') return t('desk.register.invalidName')
  return t('desk.register.failed')
}
