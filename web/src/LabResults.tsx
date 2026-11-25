import { useState } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt, type Outcome } from './api/result'
import { labResultPdfURL, uploadPdf, writeMessage, type DiagnosticReport, type Result } from './labResultApi'
import { useFormat, useT } from './i18n/useT'
import { LabResultForm } from './LabResultForm'
import { useLoad } from './load'
import { Alert, Confirm, Field, Notice } from './ui'

type Props = {
  api: ApiClient
  appointmentId: string
  zone: string
  practitioner: boolean // only a practitioner releases a result to the patient (U5)
  canAdd?: boolean // a result is attached to a visit that takes place, not to a cancelled one or a no-show
}

// VisitLabResults are a visit's lab results (docs/clinic-cms WAVE6.md U5, package LB2), for the clinic's admin and the
// visit's practitioner: each with its values, its PDF and whether the patient sees it yet. Either attaches a result
// (values typed in, the lab's PDF, or both); a practitioner reviews and releases it, after which it never changes.
// Reading the list is audited by the server, so it is read once, when the visit page opens.
export function VisitLabResults({ api, appointmentId, zone, practitioner, canAdd = true }: Props) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const { state, set, reload } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/appointments/{id}/diagnostic-reports', { params: { path: { id: appointmentId } } }))
    if (!res.ok) throw new Error('lab results')
    return res.data
  }, [api, appointmentId])
  const reports = state.name === 'loaded' ? state.value : []

  function replaced(report: DiagnosticReport) {
    set(reports.map((r) => (r.id === report.id ? report : r)))
  }

  // refused shows why a write was refused; when the result changed or was released meanwhile, the list is read again.
  function refused(code: ErrorCode | undefined, pdf = false) {
    setNotice(undefined)
    setError(writeMessage(t, code, pdf))
    if (code === 'version_conflict' || code === 'report_released') reload()
  }

  return (
    <section className="lab-results" aria-label={t('labResults.title')}>
      <h2>{t('labResults.title')}</h2>
      <Alert message={error} />
      <Notice message={notice} />
      {state.name === 'loading' && <p className="muted">{t('labResults.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('labResults.loadFailed')} />}
      {state.name === 'loaded' && reports.length === 0 && !adding && <p className="muted">{t('labResults.none')}</p>}
      {reports.map((r) => (
        <Report key={r.id} api={api} report={r} zone={zone} practitioner={practitioner} onChanged={replaced} onRefused={refused} />
      ))}
      {adding ? (
        <LabResultForm
          api={api}
          appointmentId={appointmentId}
          onCancel={() => setAdding(false)}
          onAttached={(report, pdfFailed) => {
            setAdding(false)
            set([...reports, report])
            setError(pdfFailed ? t('labResults.pdfFailedAfterAttach') : undefined)
            setNotice(t('labResults.attached'))
          }}
        />
      ) : (
        state.name === 'loaded' &&
        canAdd && (
          <div className="actions">
            <button type="button" onClick={() => setAdding(true)}>
              {t('labResults.add')}
            </button>
          </div>
        )
      )}
    </section>
  )
}

type ReportProps = {
  api: ApiClient
  report: DiagnosticReport
  zone: string
  practitioner: boolean
  onChanged: (r: DiagnosticReport) => void
  onRefused: (code: ErrorCode | undefined, pdf?: boolean) => void
}

function Report({ api, report, zone, practitioner, onChanged, onRefused }: ReportProps) {
  const t = useT()
  const f = useFormat()
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const released = report.status === 'final'

  async function run(call: () => Promise<Outcome<DiagnosticReport>>, pdf = false) {
    setBusy(true)
    const res = await call()
    setBusy(false)
    setConfirming(false)
    if (res.ok) onChanged(res.data)
    else onRefused(res.code, pdf)
  }

  const params = { path: { id: report.id }, header: { 'If-Match': `W/"${report.versionId}"` } }
  const release = () => run(() => attempt(api.POST('/clinic/diagnostic-reports/{id}/release', { params })))
  const attachPdf = (file: File) => run(() => uploadPdf(api, report, file), true)

  return (
    <article className="lab-result">
      <h3>{report.title}</h3>
      <p className="when">
        {report.effectiveDate && <span>{t('labResults.sampleDate', { date: f.calendarDate(report.effectiveDate) })}</span>}
        {released && report.releasedAt ? (
          <span>{t('labResults.releasedOn', { date: f.date(new Date(report.releasedAt), zone, 'long') })}</span>
        ) : (
          <span className="tag">{t('labResults.notReleased')}</span>
        )}
      </p>
      {report.results.length > 0 && <Values results={report.results} />}
      {report.hasPdf && (
        <p>
          <a href={labResultPdfURL(report.id)} target="_blank" rel="noreferrer">
            {t('labResults.pdf')}
          </a>
        </p>
      )}
      {!released && (
        <div className="actions">
          <Field
            label={t(report.hasPdf ? 'labResults.replacePdf' : 'labResults.attachPdf')}
            type="file"
            accept="application/pdf"
            disabled={busy}
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) void attachPdf(file)
            }}
          />
          {practitioner && !confirming && (
            <button type="button" className="primary" disabled={busy} onClick={() => setConfirming(true)}>
              {t('labResults.release')}
            </button>
          )}
        </div>
      )}
      {confirming && (
        <Confirm
          question={t('labResults.releaseQuestion')}
          lines={[t('labResults.releaseEffect')]}
          confirm={t('labResults.releaseConfirm')}
          cancel={t('labResults.notYet')}
          busy={busy}
          onConfirm={release}
          onCancel={() => setConfirming(false)}
        />
      )}
    </article>
  )
}

function Values({ results }: { results: Result[] }) {
  const t = useT()
  const f = useFormat()
  const value = (r: Result) => (r.value === undefined ? (r.valueText ?? '') : [f.number(r.value), r.unit].filter(Boolean).join(' '))
  return (
    <table className="plain-table membership-table lab-values">
      <thead>
        <tr>
          <th scope="col">{t('labResults.test')}</th>
          <th scope="col">{t('labResults.value')}</th>
          <th scope="col">{t('labResults.referenceRange')}</th>
          <th scope="col">{t('labResults.flag')}</th>
        </tr>
      </thead>
      <tbody>
        {results.map((r, i) => (
          <tr key={i}>
            <td data-label={t('labResults.test')}>{r.test}</td>
            <td data-label={t('labResults.value')}>{value(r)}</td>
            <td data-label={t('labResults.referenceRange')}>{r.referenceRange}</td>
            <td data-label={t('labResults.flag')}>{r.interpretation && t('labResults.interpretation', { flag: r.interpretation })}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
