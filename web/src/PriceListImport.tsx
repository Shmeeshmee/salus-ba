import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useLanguage, useT } from './i18n/useT'
import { details } from './serviceDetails'
import { Alert, Field } from './ui'

type Row = components['schemas']['ServiceImportRow']
type RowError = Row['errors'][number]
type T = ReturnType<typeof useT>

const errorLabel: Record<RowError, string> = {
  name_missing: 'prices.importNameMissing',
  name_invalid: 'prices.importNameInvalid',
  code_invalid: 'prices.importCodeInvalid',
  price_invalid: 'prices.importPriceInvalid',
  duration_invalid: 'prices.importDurationInvalid',
  vat_invalid: 'prices.importVatInvalid',
  mode_invalid: 'prices.importModeInvalid',
  duplicate: 'prices.importDuplicate',
}

type Props = { api: ApiClient; onImported: (count: number) => void; onCancel: () => void }

// PriceListImport imports services into the clinic's price list from a CSV or XLSX file (docs/clinic-cms, O2). The
// file is read first and every row shown with its errors; nothing is saved until the admin imports, and then all rows
// are saved or none.
export function PriceListImport({ api, onImported, onCancel }: Props) {
  const t = useT()
  const { language } = useLanguage()
  const [file, setFile] = useState<File>()
  const [rows, setRows] = useState<Row[]>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const send = async (f: File, save: boolean) => {
    setBusy(true)
    setError(undefined)
    const form = new FormData()
    form.append('file', f)
    const res = await attempt(
      api.POST('/clinic/services/import', {
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
  const choose = (f: File | undefined) => {
    setFile(f)
    setRows(undefined)
    if (f) void send(f, false)
  }
  const wrong = (rows ?? []).filter((r) => r.errors.length > 0)

  return (
    <section className="stack" aria-label={t('prices.importTitle')}>
      <h2>{t('prices.importTitle')}</h2>
      <p className="hint">{t('prices.importHint')}</p>
      <Field label={t('prices.importFile')} type="file" accept=".csv,.xlsx,text/csv" disabled={busy} onChange={(e) => choose(e.target.files?.[0])} />
      {busy && <p className="muted">{t('common.loading')}</p>}
      <Alert message={error} />
      {rows && wrong.length === 0 && <p>{t('prices.importReady', { count: rows.length })}</p>}
      {rows && wrong.length > 0 && <Alert message={t('prices.importWrong', { count: wrong.length, total: rows.length })} />}
      {rows && (
        <ul className="rows">
          {rows.map((r) => (
            <li key={r.line}>
              {r.service ? (
                <div className="stack">
                  <span className="strong">{r.service.name}</span>
                  <span className="muted small">{details(t, language, r.service)}</span>
                </div>
              ) : (
                <span>{t('prices.importLine', { line: r.line, errors: r.errors.map((e) => t(errorLabel[e])).join('; ') })}</span>
              )}
            </li>
          ))}
        </ul>
      )}
      <div className="actions">
        {file && rows && rows.length > 0 && wrong.length === 0 && (
          <button type="button" className="primary" disabled={busy} onClick={() => void send(file, true)}>
            {t('prices.importSave', { count: rows.length })}
          </button>
        )}
        <button type="button" onClick={onCancel} disabled={busy}>
          {t('prices.cancel')}
        </button>
      </div>
    </section>
  )
}

function failure(t: T, code: string | undefined): string {
  if (code === 'invalid_import') return t('prices.importUnreadable')
  if (code === 'too_large') return t('prices.importTooLarge')
  if (code === 'forbidden') return t('prices.forbidden')
  return t('prices.importFailed')
}
