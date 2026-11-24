import { useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { Dialog } from './chat/Dialog'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Field, Notice, Submit } from './ui'

type ClinicPatient = components['schemas']['ClinicPatient']

// DeskPatient is the patient's name on a desk row (clinic-cms R2). For a patient the clinic registered (no Salus
// account) it opens the patient's card, where the desk exercises the patient's rights for them (V2): correct the name,
// birth date and phone, record that the patient withdrew the consent they gave in person, and, for the admin, download
// the record to hand over. A patient with their own account manages their profile in the app: their name is text.
export function DeskPatient({ api, patient }: { api: ApiClient; patient: ClinicPatient }) {
  const [shown, setShown] = useState(patient)
  const [open, setOpen] = useState(false)
  const name = `${shown.givenName} ${shown.familyName}`
  if (!shown.registeredByClinic) return <span>{name}</span>
  return (
    <>
      <button type="button" className="desk-name" onClick={() => setOpen(true)}>
        {name}
      </button>
      {open && <Card api={api} patient={shown} onSaved={setShown} onClose={() => setOpen(false)} />}
    </>
  )
}

type CardProps = { api: ApiClient; patient: ClinicPatient; onSaved: (p: ClinicPatient) => void; onClose: () => void }

function Card({ api, patient, onSaved, onClose }: CardProps) {
  const t = useT()
  // Who is at the desk: the record is the admin's to hand over (GET /clinic/patients/{id}/record refuses anyone else).
  const me = useLoad(async () => {
    const res = await attempt(api.GET('/auth/me'))
    return res.ok && res.data.role === 'admin'
  }, [api])
  return (
    <Dialog label={t('desk.patient.title')} className="receipt-dialog desk-dialog" onClose={onClose}>
      <h2>{t('desk.patient.title')}</h2>
      <Profile api={api} patient={patient} onSaved={onSaved} />
      <Withdrawal api={api} patientId={patient.id} />
      {me.state.name === 'loaded' && me.state.value && (
        <section>
          <h3>{t('desk.patient.recordTitle')}</h3>
          <p className="muted small">{t('desk.patient.recordHint')}</p>
          <a href={recordURL(patient.id)} download={`patient-${patient.id}.json`}>
            {t('desk.patient.download')}
          </a>
        </section>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          {t('desk.patient.close')}
        </button>
      </div>
    </Dialog>
  )
}

function Profile({ api, patient, onSaved }: { api: ApiClient; patient: ClinicPatient; onSaved: (p: ClinicPatient) => void }) {
  const t = useT()
  const [givenName, setGivenName] = useState(patient.givenName)
  const [familyName, setFamilyName] = useState(patient.familyName)
  const [birthDate, setBirthDate] = useState(patient.birthDate)
  const [phone, setPhone] = useState(patient.phone ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [saved, setSaved] = useState<string>()
  const valid = givenName.trim() !== '' && familyName.trim() !== '' && birthDate !== ''

  async function save(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(undefined)
    setSaved(undefined)
    const body = { givenName: givenName.trim(), familyName: familyName.trim(), birthDate, phone: phone.replace(/\s/g, '') || undefined }
    const res = await attempt(
      api.PUT('/clinic/patients/{id}', { params: { path: { id: patient.id }, header: { 'If-Match': `W/"${patient.versionId}"` } }, body }),
    )
    setBusy(false)
    if (res.ok) {
      onSaved(res.data)
      setSaved(t('desk.patient.saved'))
    } else setError(profileMessage(t, res.code))
  }

  return (
    <form onSubmit={save}>
      <fieldset className="patient-register">
        <legend>{t('desk.patient.profile')}</legend>
        <Field label={t('desk.register.givenName')} value={givenName} maxLength={200} required onChange={(e) => setGivenName(e.target.value)} />
        <Field label={t('desk.register.familyName')} value={familyName} maxLength={200} required onChange={(e) => setFamilyName(e.target.value)} />
        <Field label={t('desk.register.birthDate')} type="date" value={birthDate} required onChange={(e) => setBirthDate(e.target.value)} />
        <Field label={t('desk.register.phone')} type="tel" value={phone} autoComplete="off" onChange={(e) => setPhone(e.target.value)} />
        <Alert message={error} />
        <Notice message={saved} />
        <Submit busy={busy} disabled={!valid} label={t('desk.patient.save')} busyLabel={t('desk.patient.saving')} />
      </fieldset>
    </form>
  )
}

function Withdrawal({ api, patientId }: { api: ApiClient; patientId: string }) {
  const t = useT()
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [done, setDone] = useState(false)

  async function withdraw() {
    if (busy) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/clinic/patients/{id}/consent-withdrawal', { params: { path: { id: patientId } } }))
    setBusy(false)
    if (res.ok) {
      setDone(true)
      setConfirming(false)
    } else setError(res.code === 'consent_inactive' ? t('desk.patient.noConsent') : t('desk.patient.withdrawFailed'))
  }

  return (
    <section>
      <h3>{t('desk.patient.consentTitle')}</h3>
      <Notice message={done ? t('desk.patient.withdrawn') : undefined} />
      <Alert message={error} />
      {!done && !confirming && (
        <button type="button" onClick={() => setConfirming(true)}>
          {t('desk.patient.withdraw')}
        </button>
      )}
      {confirming && (
        <>
          <p>{t('desk.patient.withdrawEffect')}</p>
          <div className="actions">
            <button type="button" className="primary" disabled={busy} onClick={withdraw}>
              {t('desk.patient.confirmWithdraw')}
            </button>
            <button type="button" onClick={() => setConfirming(false)}>
              {t('desk.patient.keep')}
            </button>
          </div>
        </>
      )}
    </section>
  )
}

function profileMessage(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'version_conflict') return t('desk.patient.conflict')
  if (code === 'invalid_phone') return t('desk.register.invalidPhone')
  if (code === 'invalid_birth_date') return t('desk.register.invalidBirthDate')
  if (code === 'invalid_name') return t('desk.register.invalidName')
  return t('desk.patient.saveFailed')
}

// recordURL is where the admin downloads the record (GET /clinic/patients/{id}/record), on the same origin as every
// other call, so the session cookie goes with it.
function recordURL(id: string): string {
  return `${globalThis.location.origin}/clinic/patients/${encodeURIComponent(id)}/record`
}
