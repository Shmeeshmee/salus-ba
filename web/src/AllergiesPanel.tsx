import { useId, useState } from 'react'
import type { ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { activeAllergies, confirmEntry, encounterOf, recordAllergy, recordMedication, type Allergy, type EntryKind, type Medication, type PatientEntries, type Written, unitKey } from './entries'
import { AllergyForm, MedicationForm, type Saved } from './EntryForms'
import { useT } from './i18n/useT'
import { Alert } from './ui'

// The patient's allergies and current medications on the call page (SAL-41): the warning of the active allergies
// on top of the page and in the drafting editor, and the panel that lists them, records them during the visit and
// confirms what the patient or another clinic entered. Nothing here reaches the drafting or the language model
// (ADR 0014 "No AI"): the warning is shown, never sent.

type Fhir = Pick<FhirClient, 'get'>

// AllergyWarning lists every active allergy with its reaction and severity; nothing when there is none.
export function AllergyWarning({ allergies }: { allergies: Allergy[] }) {
  const t = useT()
  const id = useId()
  const active = activeAllergies(allergies)
  if (active.length === 0) return null
  return (
    <section className="allergy-warning" aria-labelledby={id}>
      <h2 id={id}>{t('allergies.warningTitle')}</h2>
      <ul>
        {active.map((a) => (
          <li key={a.id}>
            <span className="strong">{a.text}</span>
            {a.reaction && <span>{reactionText(t, a)}</span>}
          </li>
        ))}
      </ul>
    </section>
  )
}

type T = ReturnType<typeof useT>

function reactionText(t: T, a: Allergy): string | undefined {
  if (!a.reaction) return undefined
  return a.severity ? t('allergies.reactionSeverity', { reaction: a.reaction, severity: a.severity }) : a.reaction
}

type PanelProps = { api: ApiClient; fhir: Fhir; appointmentId: string; entries: PatientEntries }

type Adding = 'allergy' | 'medication'

// EntriesPanel lists the patient's allergies and medications, records new ones in the visit's encounter and
// confirms an unconfirmed one at the version read. Every write reads the lists again.
export function EntriesPanel({ api, fhir, appointmentId, entries }: PanelProps) {
  const t = useT()
  const [adding, setAdding] = useState<Adding>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState<string>()

  async function confirm(kind: EntryKind, id: string, version: string) {
    setBusy(id)
    setError(undefined)
    const result = await confirmEntry(api, kind, id, version)
    setBusy(undefined)
    if (result === 'ok' || result === 'changed') entries.reload()
    if (result !== 'ok') setError(t(confirmMessage[result]))
  }

  // record writes in the visit's encounter, which exists once the consultation has started.
  async function record(write: (encounter: string) => Promise<Written>): Promise<Saved> {
    const encounter = await encounterOf(fhir, appointmentId).catch(() => undefined)
    if (!encounter) return 'noVisit'
    const result = await write(encounter)
    if (result === 'ok') {
      setAdding(undefined)
      entries.reload()
    }
    return result
  }

  const loaded = entries.entries
  return (
    <div className="entries">
      <Alert message={entries.failed ? t('allergies.loadFailed') : error} />
      {!loaded && !entries.failed && <p className="muted">{t('allergies.loading')}</p>}
      {loaded && (
        <>
          <AllergyList allergies={loaded.allergies} busy={busy} onConfirm={(a) => void confirm('allergy', a.id, a.version)} />
          {adding === 'allergy' ? (
            <AllergyForm api={api} onSave={(body) => record((e) => recordAllergy(api, e, body))} onCancel={() => setAdding(undefined)} />
          ) : (
            <button type="button" onClick={() => setAdding('allergy')}>
              {t('allergies.addAllergy')}
            </button>
          )}
          <MedicationList medications={loaded.medications} busy={busy} onConfirm={(m) => void confirm('medication', m.id, m.version)} />
          {adding === 'medication' ? (
            <MedicationForm api={api} onSave={(body) => record((e) => recordMedication(api, e, body))} onCancel={() => setAdding(undefined)} />
          ) : (
            <button type="button" onClick={() => setAdding('medication')}>
              {t('allergies.addMedication')}
            </button>
          )}
        </>
      )}
    </div>
  )
}

const confirmMessage: Record<Exclude<Written, 'ok'>, string> = {
  changed: 'allergies.confirmChanged',
  forbidden: 'allergies.confirmForbidden',
  invalid: 'allergies.confirmFailed',
  failed: 'allergies.confirmFailed',
}

function AllergyList({ allergies, busy, onConfirm }: { allergies: Allergy[]; busy?: string; onConfirm: (a: Allergy) => void }) {
  const t = useT()
  const id = useId()
  return (
    <section className="entry-group">
      <h3 id={id}>{t('allergies.allergies')}</h3>
      {allergies.length === 0 ? (
        <p className="muted">{t('allergies.noAllergies')}</p>
      ) : (
        <ul className="entry-list" aria-labelledby={id}>
          {allergies.map((a) => (
            <li key={a.id}>
              <p className="strong">{a.text}</p>
              <p className="entry-meta">
                {a.category && <span>{t(`allergies.category.${a.category}`)}</span>}
                {a.reaction && <span>{reactionText(t, a)}</span>}
                {a.clinicalStatus && a.clinicalStatus !== 'active' && <span>{t(`allergies.clinicalStatus.${a.clinicalStatus}`)}</span>}
              </p>
              <Confirmation confirmed={a.confirmed} text={a.text} busy={busy === a.id} onConfirm={() => onConfirm(a)} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function MedicationList({ medications, busy, onConfirm }: { medications: Medication[]; busy?: string; onConfirm: (m: Medication) => void }) {
  const t = useT()
  const id = useId()
  return (
    <section className="entry-group">
      <h3 id={id}>{t('allergies.medications')}</h3>
      {medications.length === 0 ? (
        <p className="muted">{t('allergies.noMedications')}</p>
      ) : (
        <ul className="entry-list" aria-labelledby={id}>
          {medications.map((m) => (
            <li key={m.id}>
              <p className="strong">{m.text}</p>
              <p className="entry-meta">
                {m.dose && <span>{t(`allergies.dose.${unitKey(m.dose.unit)}`, { value: m.dose.value })}</span>}
                {m.timing && <span>{t(`allergies.timing.${m.timing.periodUnit}`, { frequency: m.timing.frequency, period: m.timing.period })}</span>}
                {m.asNeeded && <span>{t('allergies.asNeeded')}</span>}
                {m.status === 'stopped' && <span>{t('allergies.status.stopped')}</span>}
              </p>
              <Confirmation confirmed={m.confirmed} text={m.text} busy={busy === m.id} onConfirm={() => onConfirm(m)} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function Confirmation({ confirmed, text, busy, onConfirm }: { confirmed: boolean; text: string; busy: boolean; onConfirm: () => void }) {
  const t = useT()
  if (confirmed) return <p className="tag confirmed">{t('allergies.confirmed')}</p>
  return (
    <p className="entry-confirm">
      <span className="tag">{t('allergies.unconfirmed')}</span>
      <button type="button" className="quiet" disabled={busy} aria-label={t('allergies.confirmLabel', { text })} onClick={onConfirm}>
        {t('allergies.confirm')}
      </button>
    </p>
  )
}
