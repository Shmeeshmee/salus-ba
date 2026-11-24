import { useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { Dialog } from './chat/Dialog'
import { DeskPatient } from './DeskPatient'
import { useFormat, useT } from './i18n/useT'
import type { ClinicPatient } from './PatientPicker'
import { groupPhone } from './phone'
import { Alert, Field, Submit } from './ui'

type Found = { name: 'idle' } | { name: 'searching' } | { name: 'failed' } | { name: 'found'; patients: ClinicPatient[] }

type Props = { api: ApiClient; onChanged: () => void; onClose: () => void }

// DeskPatients is the desk's search of the clinic's patients (wave 4 walk, finding 5; WAVE5 FX), for the admin and the
// receptionist: by name and, when given, birth date (GET /clinic/patients, each patient found audited), whatever their
// consent, so a patient who withdrew can still have their profile corrected, a new consent recorded or their record
// handed over. A patient the clinic registered opens their card (DeskPatient); onChanged tells the desk to read its
// day again after a card changed something (a withdrawal cancels the visits ahead).
export function DeskPatients({ api, onChanged, onClose }: Props) {
  const t = useT()
  const f = useFormat()
  const [name, setName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [found, setFound] = useState<Found>({ name: 'idle' })
  const query = name.trim()

  async function search(e: FormEvent) {
    e.preventDefault()
    if (query.length < 2) return
    setFound({ name: 'searching' })
    const res = await attempt(api.GET('/clinic/patients', { params: { query: { name: query, birthDate: birthDate || undefined } } }))
    setFound(res.ok ? { name: 'found', patients: res.data } : { name: 'failed' })
  }

  return (
    <Dialog label={t('desk.patients.title')} className="receipt-dialog desk-dialog" onClose={onClose}>
      <h2>{t('desk.patients.title')}</h2>
      <p className="muted small">{t('desk.patients.hint')}</p>
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
              <span className="stack">
                <DeskPatient api={api} patient={p} onChanged={onChanged} />
                <span className="muted small">{[f.calendarDate(p.birthDate), p.phone && groupPhone(p.phone)].filter(Boolean).join(' · ')}</span>
              </span>
            </li>
          ))}
        </ul>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          {t('desk.patient.close')}
        </button>
      </div>
    </Dialog>
  )
}
