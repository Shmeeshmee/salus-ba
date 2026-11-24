import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { Alert, Field } from './ui'

type Row = components['schemas']['PatientImportRow']
type RowError = Row['errors'][number]
type T = ReturnType<typeof useT>

const errorLabel: Record<RowError, string> = {
  given_name_invalid: 'patientImport.givenNameInvalid',
  family_name_invalid: 'patientImport.familyNameInvalid',
  birth_date_invalid: 'patientImport.birthDateInvalid',
  phone_invalid: 'patientImport.phoneInvalid',
}

type Props = { api: ApiClient; onImported: (count: number) => void; onCancel: () => void }

// PatientImport imports the clinic's patients from a CSV or XLSX file (docs/clinic-cms/WAVE4.md, M3). The file is
// read first and the rows with errors and the likely duplicates are shown; nothing is saved until the admin imports,
// and then all rows are saved or none. Only the rows that need a look are listed, the rest are counted.
export function PatientImport({ api, onImported, onCancel }: Props) {
  const t = useT()
  const f = useFormat()
  const [file, setFile] = useState<File>()
  const [rows, setRows] = useState<Row[]>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const send = async (chosen: File, save: boolean) => {
    setBusy(true)
    setError(undefined)
    const form = new FormData()
    form.append('file', chosen)
    const res = await attempt(
      api.POST('/clinic/patients/import', {
        params: { query: save ? { save: true } : {} },
        body: { file: '' },
        bodySerializer: () => form,
      }),
    )
    setBusy(false)
    if (!res.ok) {
      setRows(undefined)
      setError(failure(t, res.code))
      return
    }
    if (save && res.data.saved) {
      onImported(res.data.rows.length)
      return
    }
    setRows(res.data.rows)
  }
  const choose = (chosen: File | undefined) => {
    setFile(chosen)
    setRows(undefined)
    if (chosen) void send(chosen, false)
  }
  const wrong = (rows ?? []).filter((r) => r.errors.length > 0)
  const duplicates = (rows ?? []).filter((r) => r.duplicate && r.patient)
  const flagged = (rows ?? []).filter((r) => r.errors.length > 0 || (r.duplicate && r.patient))

  return (
    <section className="panel stack" aria-label={t('patientImport.title')}>
      <h2>{t('patientImport.title')}</h2>
      <p className="hint">{t('patientImport.hint')}</p>
      <Field label={t('patientImport.file')} type="file" accept=".csv,.xlsx,text/csv" disabled={busy} onChange={(e) => choose(e.target.files?.[0])} />
      {busy && <p className="muted">{t('common.loading')}</p>}
      <Alert message={error} />
      {rows && wrong.length === 0 && <p>{t('patientImport.ready', { count: rows.length })}</p>}
      {rows && wrong.length > 0 && <Alert message={t('patientImport.wrong', { count: wrong.length, total: rows.length })} />}
      {rows && duplicates.length > 0 && <p className="hint">{t('patientImport.duplicates', { count: duplicates.length })}</p>}
      {flagged.length > 0 && (
        <ul className="rows">
          {flagged.map((r) => (
            <li key={r.line}>
              {r.errors.length > 0 || !r.patient
                ? t('patientImport.line', { line: r.line, errors: r.errors.map((e) => t(errorLabel[e])).join('; ') })
                : t('patientImport.duplicateLine', {
                    line: r.line,
                    name: `${r.patient.givenName} ${r.patient.familyName}`,
                    birthDate: f.calendarDate(r.patient.birthDate),
                  })}
            </li>
          ))}
        </ul>
      )}
      <div className="actions">
        {file && rows && rows.length > 0 && wrong.length === 0 && (
          <button type="button" className="primary" disabled={busy} onClick={() => void send(file, true)}>
            {t('patientImport.save', { count: rows.length })}
          </button>
        )}
        <button type="button" onClick={onCancel} disabled={busy}>
          {t('patientImport.cancel')}
        </button>
      </div>
    </section>
  )
}

function failure(t: T, code: string | undefined): string {
  if (code === 'invalid_import') return t('patientImport.unreadable')
  if (code === 'too_large') return t('patientImport.tooLarge')
  if (code === 'forbidden') return t('patientImport.forbidden')
  return t('patientImport.failed')
}
