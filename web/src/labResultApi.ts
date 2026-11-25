import type { ApiClient, ErrorCode } from './api/client'
import { attempt, type Outcome } from './api/result'
import type { components } from './api/schema'
import type { useT } from './i18n/useT'

// The lab results' calls and rules of the visit page (clinic-cms LB2), apart from its components.

export type DiagnosticReport = components['schemas']['DiagnosticReport']
export type Result = components['schemas']['DiagnosticReportResult']
export type Flag = components['schemas']['ObservationInterpretation']
export type Row = { key: number; test: string; value: string; unit: string; range: string; flag: Flag | '' }
type T = ReturnType<typeof useT>

export const flags: Flag[] = ['N', 'L', 'H', 'LL', 'HH', 'A']

// uploadPdf puts the lab's PDF on a result at the version read. The browser sets Content-Length from the file itself.
export function uploadPdf(api: ApiClient, report: DiagnosticReport, file: File): Promise<Outcome<DiagnosticReport>> {
  return attempt(
    api.PUT('/clinic/diagnostic-reports/{id}/pdf', {
      params: { path: { id: report.id }, header: { 'If-Match': `W/"${report.versionId}"`, 'Content-Length': file.size } },
      headers: { 'Content-Type': 'application/pdf' },
      body: file as unknown as string,
      bodySerializer: (body: unknown) => body as File,
    }),
  )
}

// labResultPdfURL is where a result's PDF is read, on the same origin as every other call, so the session cookie goes
// with it.
export function labResultPdfURL(id: string): string {
  return `${globalThis.location.origin}/clinic/diagnostic-reports/${encodeURIComponent(id)}/pdf`
}

export function writeMessage(t: T, code: ErrorCode | undefined, pdf = false): string {
  if (code === 'invalid_diagnostic_report') return t(pdf ? 'labResults.notPdf' : 'labResults.invalid')
  if (code === 'not_found') return t('labResults.notFound')
  if (code === 'forbidden') return t('labResults.forbidden')
  if (code === 'report_released') return t('labResults.alreadyReleased')
  if (code === 'version_conflict') return t('labResults.changed')
  if (code === 'too_large') return t('labResults.tooLarge')
  return t('common.somethingWrong')
}

// resultsOf turns the rows into the request's values, leaving out a row left empty, or names the message key of the
// first row that cannot be sent.
export function resultsOf(rows: Row[]): Result[] | string {
  const out: Result[] = []
  for (const r of rows) {
    const [test, value, unit, range] = [r.test.trim(), r.value.trim(), r.unit.trim(), r.range.trim()]
    if (!test && !value && !unit && !range && !r.flag) continue
    if (!test || !value) return 'labResults.rowIncomplete'
    const number = numberOf(value)
    if (number === undefined && unit) return 'labResults.unitWithoutNumber'
    out.push({
      test,
      ...(number === undefined ? { valueText: value } : { value: number, ...(unit ? { unit } : {}) }),
      ...(range ? { referenceRange: range } : {}),
      ...(r.flag ? { interpretation: r.flag } : {}),
    })
  }
  return out
}

function numberOf(text: string): number | undefined {
  const normal = text.replace(',', '.')
  return /^-?\d+(\.\d+)?$/.test(normal) ? Number(normal) : undefined
}
