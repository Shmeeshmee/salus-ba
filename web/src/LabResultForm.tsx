import { useId, useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { useT } from './i18n/useT'
import { flags, resultsOf, uploadPdf, writeMessage, type DiagnosticReport, type Flag, type Row } from './labResultApi'
import { Alert, Field, Submit } from './ui'

type Props = {
  api: ApiClient
  appointmentId: string
  onCancel: () => void
  onAttached: (report: DiagnosticReport, pdfFailed: boolean) => void
}

const emptyRow = (key: number): Row => ({ key, test: '', value: '', unit: '', range: '', flag: '' })

// LabResultForm attaches a lab result to a visit: its title, the sample's day, the values as the lab printed them, and
// the lab's PDF, sent after the result at its new version. A value that reads as a number (a comma or a point for the
// decimals) is sent as one, with its unit; any other value is sent as text and may not carry a unit.
export function LabResultForm({ api, appointmentId, onCancel, onAttached }: Props) {
  const t = useT()
  const id = useId()
  const [title, setTitle] = useState('')
  const [date, setDate] = useState('')
  const [rows, setRows] = useState<Row[]>([emptyRow(0)])
  const [file, setFile] = useState<File>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const change = (key: number, patch: Partial<Row>) => setRows(rows.map((r) => (r.key === key ? { ...r, ...patch } : r)))

  async function submit(e: FormEvent) {
    e.preventDefault()
    const results = resultsOf(rows)
    if (typeof results === 'string') return setError(t(results))
    if (!title.trim()) return setError(t('labResults.titleMissing'))
    setBusy(true)
    setError(undefined)
    const body = { title: title.trim(), ...(date ? { effectiveDate: date } : {}), results }
    const res = await attempt(api.POST('/clinic/appointments/{id}/diagnostic-reports', { params: { path: { id: appointmentId } }, body }))
    if (!res.ok) {
      setBusy(false)
      return setError(writeMessage(t, res.code))
    }
    const withPdf = file ? await uploadPdf(api, res.data, file) : undefined
    setBusy(false)
    onAttached(withPdf?.ok ? withPdf.data : res.data, withPdf !== undefined && !withPdf.ok)
  }

  return (
    <form className="lab-result-form" aria-label={t('labResults.newTitle')} onSubmit={submit} noValidate>
      <h3>{t('labResults.newTitle')}</h3>
      <Alert message={error} />
      <div className="grid">
        <Field label={t('labResults.titleLabel')} value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} />
        <Field label={t('labResults.sampleLabel')} type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </div>
      {rows.map((r, i) => (
        <fieldset key={r.key} className="lab-value">
          <legend>{t('labResults.valueN', { n: i + 1 })}</legend>
          <Field label={t('labResults.test')} value={r.test} maxLength={200} onChange={(e) => change(r.key, { test: e.target.value })} />
          <Field label={t('labResults.value')} value={r.value} maxLength={200} onChange={(e) => change(r.key, { value: e.target.value })} />
          <Field label={t('labResults.unit')} value={r.unit} maxLength={40} onChange={(e) => change(r.key, { unit: e.target.value })} />
          <Field label={t('labResults.referenceRange')} value={r.range} maxLength={100} onChange={(e) => change(r.key, { range: e.target.value })} />
          <div className="field">
            <label htmlFor={`${id}-f${r.key}`}>{t('labResults.flag')}</label>
            <select id={`${id}-f${r.key}`} value={r.flag} onChange={(e) => change(r.key, { flag: e.target.value as Flag | '' })}>
              <option value="">{t('labResults.noFlag')}</option>
              {flags.map((f) => (
                <option key={f} value={f}>
                  {t('labResults.interpretation', { flag: f })}
                </option>
              ))}
            </select>
          </div>
          {rows.length > 1 && (
            <button type="button" className="quiet" onClick={() => setRows(rows.filter((x) => x.key !== r.key))}>
              {t('labResults.removeValue', { n: i + 1 })}
            </button>
          )}
        </fieldset>
      ))}
      <div className="actions">
        <button type="button" disabled={rows.length >= 100} onClick={() => setRows([...rows, emptyRow(Math.max(...rows.map((r) => r.key)) + 1)])}>
          {t('labResults.addValue')}
        </button>
      </div>
      <Field label={t('labResults.pdfLabel')} type="file" accept="application/pdf" onChange={(e) => setFile(e.target.files?.[0])} />
      <div className="actions">
        <Submit busy={busy} label={t('labResults.attach')} busyLabel={t('labResults.attaching')} />
        <button type="button" disabled={busy} onClick={onCancel}>
          {t('labResults.cancel')}
        </button>
      </div>
    </form>
  )
}
