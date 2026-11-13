import { useEffect, useId, useState, type FormEvent, type KeyboardEvent, type ReactNode } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import {
  allergyCategories,
  clinicalStatuses,
  doseUnits,
  medicationStatuses,
  periodUnits,
  severities,
  type AllergyInput,
  type ClinicalStatus,
  type DoseUnit,
  type MedicationInput,
  type MedicationStatus,
  type PeriodUnit,
  type Written,
  unitKey,
} from './entries'
import { useLanguage, useT } from './i18n/useT'
import { Alert } from './ui'

// The forms that record an allergy or a medication during the visit (SAL-41). The substance or the medication is
// searched in the terminology (the drug category): a term picked sends its id and its label in the UI language; a
// text typed by hand is sent as text only.

// Saved is how a save ended: as the API answered, or not tried because the consultation has no encounter yet.
export type Saved = Written | 'noVisit'

type Term = components['schemas']['Term']
type TermLanguage = components['schemas']['TermLanguage']
type Picked = { text: string; termId?: string }

const savedMessage: Record<Exclude<Saved, 'ok'>, string> = {
  noVisit: 'allergies.noVisit',
  invalid: 'allergies.invalid',
  forbidden: 'allergies.saveForbidden',
  changed: 'allergies.saveFailed',
  failed: 'allergies.saveFailed',
}

// useSave runs a save once at a time and keeps the message of how it ended.
function useSave<B>(onSave: (body: B) => Promise<Saved>) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  async function save(body: B) {
    setBusy(true)
    setError(undefined)
    const result = await onSave(body)
    setBusy(false)
    if (result !== 'ok') setError(t(savedMessage[result]))
  }
  return { busy, error, setError, save }
}

type FormProps<B> = { api: ApiClient; onSave: (body: B) => Promise<Saved>; onCancel: () => void }

export function AllergyForm({ api, onSave, onCancel }: FormProps<AllergyInput>) {
  const t = useT()
  const id = useId()
  const [picked, setPicked] = useState<Picked>({ text: '' })
  const [category, setCategory] = useState('')
  const [reaction, setReaction] = useState('')
  const [severity, setSeverity] = useState('')
  const [clinicalStatus, setClinicalStatus] = useState<ClinicalStatus>('active')
  const { busy, error, save } = useSave(onSave)

  function submit(e: FormEvent) {
    e.preventDefault()
    const text = picked.text.trim()
    if (!text) return
    const body: AllergyInput = { ...(picked.termId ? { termId: picked.termId } : {}), text, clinicalStatus }
    const r = reaction.trim()
    if (category) body.category = category as AllergyInput['category']
    if (r) body.reaction = r
    if (r && severity) body.severity = severity as AllergyInput['severity']
    void save(body)
  }

  return (
    <EntryForm title={t('allergies.addAllergy')} error={error} busy={busy} onSubmit={submit} onCancel={onCancel}>
      <TermField api={api} label={t('allergies.substance')} hint={t('allergies.substanceHint')} value={picked} onChange={setPicked} />
      <Select id={`${id}-category`} label={t('allergies.categoryLabel')} value={category} onChange={setCategory}>
        <option value="">{t('allergies.notSpecified')}</option>
        {allergyCategories.map((c) => (
          <option key={c} value={c}>
            {t(`allergies.category.${c}`)}
          </option>
        ))}
      </Select>
      <div className="field">
        <label htmlFor={`${id}-reaction`}>{t('allergies.reaction')}</label>
        <input id={`${id}-reaction`} type="text" maxLength={200} value={reaction} onChange={(e) => setReaction(e.target.value)} />
      </div>
      <Select id={`${id}-severity`} label={t('allergies.severityLabel')} value={reaction.trim() ? severity : ''} onChange={setSeverity} disabled={!reaction.trim()} hint={t('allergies.severityHint')}>
        <option value="">{t('allergies.notSpecified')}</option>
        {severities.map((s) => (
          <option key={s} value={s}>
            {t(`allergies.severity.${s}`)}
          </option>
        ))}
      </Select>
      <Select id={`${id}-status`} label={t('allergies.statusLabel')} value={clinicalStatus} onChange={(v) => setClinicalStatus(v as ClinicalStatus)}>
        {clinicalStatuses.map((s) => (
          <option key={s} value={s}>
            {t(`allergies.clinicalStatus.${s}`)}
          </option>
        ))}
      </Select>
    </EntryForm>
  )
}

export function MedicationForm({ api, onSave, onCancel }: FormProps<MedicationInput>) {
  const t = useT()
  const id = useId()
  const [picked, setPicked] = useState<Picked>({ text: '' })
  const [dose, setDose] = useState('')
  const [unit, setUnit] = useState('')
  const [frequency, setFrequency] = useState('')
  const [period, setPeriod] = useState('')
  const [periodUnit, setPeriodUnit] = useState('')
  const [asNeeded, setAsNeeded] = useState(false)
  const [status, setStatus] = useState<MedicationStatus>('active')
  const { busy, error, setError, save } = useSave(onSave)

  function submit(e: FormEvent) {
    e.preventDefault()
    const text = picked.text.trim()
    if (!text) return
    const dosage: MedicationInput['dosage'] = {}
    if (Boolean(dose) !== Boolean(unit)) return setError(t('allergies.doseBoth'))
    const timing = [frequency, period, periodUnit].filter(Boolean).length
    if (timing !== 0 && timing !== 3) return setError(t('allergies.timingAll'))
    if (dose) Object.assign(dosage, { doseValue: Number(dose), doseUnit: unit as DoseUnit })
    if (timing === 3) Object.assign(dosage, { frequency: Number(frequency), period: Number(period), periodUnit: periodUnit as PeriodUnit })
    if (asNeeded) dosage.asNeeded = true
    void save({ ...(picked.termId ? { termId: picked.termId } : {}), text, status, dosage })
  }

  return (
    <EntryForm title={t('allergies.addMedication')} error={error} busy={busy} onSubmit={submit} onCancel={onCancel}>
      <TermField api={api} label={t('allergies.medication')} hint={t('allergies.medicationHint')} value={picked} onChange={setPicked} />
      <div className="entry-row">
        <NumberField id={`${id}-dose`} label={t('allergies.doseLabel')} value={dose} onChange={setDose} min={0} max={99999} step="any" />
        <Select id={`${id}-unit`} label={t('allergies.unitLabel')} value={unit} onChange={setUnit}>
          <option value="">{t('allergies.noUnit')}</option>
          {doseUnits.map((u) => (
            <option key={u} value={u}>
              {t(`allergies.unit.${unitKey(u)}`)}
            </option>
          ))}
        </Select>
      </div>
      <div className="entry-row">
        <NumberField id={`${id}-frequency`} label={t('allergies.frequency')} value={frequency} onChange={setFrequency} min={1} max={100} step="1" />
        <NumberField id={`${id}-period`} label={t('allergies.period')} value={period} onChange={setPeriod} min={0} max={1000} step="any" />
        <Select id={`${id}-periodUnit`} label={t('allergies.periodUnitLabel')} value={periodUnit} onChange={setPeriodUnit}>
          <option value="">{t('allergies.noUnit')}</option>
          {periodUnits.map((u) => (
            <option key={u} value={u}>
              {t(`allergies.periodUnit.${u}`)}
            </option>
          ))}
        </Select>
      </div>
      <label className="check">
        <input type="checkbox" checked={asNeeded} onChange={(e) => setAsNeeded(e.target.checked)} />
        {t('allergies.asNeeded')}
      </label>
      <Select id={`${id}-status`} label={t('allergies.statusLabel')} value={status} onChange={(v) => setStatus(v as MedicationStatus)}>
        {medicationStatuses.map((s) => (
          <option key={s} value={s}>
            {t(`allergies.status.${s}`)}
          </option>
        ))}
      </Select>
    </EntryForm>
  )
}

type EntryFormProps = { title: string; error?: string; busy: boolean; onSubmit: (e: FormEvent) => void; onCancel: () => void; children: ReactNode }

function EntryForm({ title, error, busy, onSubmit, onCancel, children }: EntryFormProps) {
  const t = useT()
  const id = useId()
  return (
    <form className="entry-form" aria-labelledby={id} onSubmit={onSubmit}>
      <h4 id={id}>{title}</h4>
      {children}
      <Alert message={error} />
      <div className="actions">
        <button type="submit" className="primary" disabled={busy}>
          {busy ? t('allergies.saving') : t('allergies.save')}
        </button>
        <button type="button" disabled={busy} onClick={onCancel}>
          {t('allergies.cancel')}
        </button>
      </div>
    </form>
  )
}

type SelectProps = { id: string; label: string; value: string; onChange: (v: string) => void; disabled?: boolean; hint?: string; children: ReactNode }

function Select({ id, label, value, onChange, disabled, hint, children }: SelectProps) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <select id={id} value={value} disabled={disabled} aria-describedby={hint && disabled ? `${id}-hint` : undefined} onChange={(e) => onChange(e.target.value)}>
        {children}
      </select>
      {hint && disabled && (
        <p className="hint" id={`${id}-hint`}>
          {hint}
        </p>
      )}
    </div>
  )
}

type NumberProps = { id: string; label: string; value: string; onChange: (v: string) => void; min: number; max: number; step: string }

function NumberField({ id, label, value, onChange, min, max, step }: NumberProps) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input id={id} type="number" inputMode="decimal" min={min} max={max} step={step} value={value} onChange={(e) => onChange(e.target.value)} />
    </div>
  )
}

// The search waits this long after the last change before it asks, as the cheatsheet's.
const debounceMs = 150

type TermFieldProps = { api: ApiClient; label: string; hint: string; value: Picked; onChange: (p: Picked) => void }

// TermField is a text field that offers the terminology's drug terms matching what is typed. Typing makes the value
// text only; picking a term sets its label in the UI language (else its first) and its id.
function TermField({ api, label, hint, value, onChange }: TermFieldProps) {
  const t = useT()
  const id = useId()
  const { language } = useLanguage()
  const [terms, setTerms] = useState<Term[]>([])
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)

  useEffect(() => {
    const q = query.trim()
    if (q === '') return
    const abort = new AbortController()
    const timer = setTimeout(async () => {
      const languages = [...new Set<TermLanguage>([language, 'en', 'la'])]
      const res = await attempt(
        api.GET('/terminology/search', {
          params: { query: { q: q.slice(0, 100), languages, category: 'drug', limit: 8 } },
          querySerializer: { array: { style: 'form', explode: false } },
          signal: abort.signal,
        }),
      )
      if (abort.signal.aborted) return
      setTerms(res.ok ? res.data.terms : [])
      setActive(0)
    }, debounceMs)
    return () => {
      clearTimeout(timer)
      abort.abort()
    }
  }, [api, query, language])

  const labelOf = (term: Term) => (term.labels.find((l) => l.language === language) ?? term.labels[0])?.preferred ?? ''
  // Nothing is offered once the text is cleared, whatever an earlier search found.
  const shown = query.trim() === '' ? [] : terms.filter((x) => labelOf(x) !== '')

  function pick(term: Term) {
    onChange({ text: labelOf(term), termId: term.id })
    setQuery('')
    setTerms([])
  }

  function onKeyDown(e: KeyboardEvent) {
    if (shown.length === 0) return
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => Math.max(0, Math.min(shown.length - 1, a + (e.key === 'ArrowDown' ? 1 : -1))))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      pick(shown[active])
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setTerms([])
    }
  }

  const optionId = (i: number) => `${id}-term-${i}`
  return (
    <div className="field term-field">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        type="text"
        role="combobox"
        required
        maxLength={200}
        autoComplete="off"
        aria-expanded={shown.length > 0}
        aria-controls={`${id}-terms`}
        aria-activedescendant={shown.length > 0 ? optionId(active) : undefined}
        aria-describedby={`${id}-hint`}
        value={value.text}
        onChange={(e) => {
          onChange({ text: e.target.value })
          setQuery(e.target.value)
        }}
        onKeyDown={onKeyDown}
      />
      <p className="hint" id={`${id}-hint`}>
        {value.termId ? t('allergies.fromTerms') : hint}
      </p>
      {shown.length > 0 && (
        <ul className="term-options" role="listbox" id={`${id}-terms`} aria-label={t('allergies.terms')}>
          {shown.map((term, i) => (
            <li key={term.id} id={optionId(i)} role="option" aria-selected={i === active} onMouseDown={(e) => e.preventDefault()} onClick={() => pick(term)}>
              {labelOf(term)}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
