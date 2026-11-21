import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { ReceiptPeriod } from './ReceiptPeriod'
import { instants, money, reportCsvURL, thisMonth, type Period } from './receipts'
import { loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; account: Account; now?: () => Date }

// ClinicReports is the receipts summed for the clinic's accountant (clinic-cms C): for a day or a period of the
// clinic's calendar, how many receipts were issued and their total, by payment method and by service, with the same
// figures as a CSV file. Reversals count as receipts and take their amounts back.
export function ClinicReports({ api, fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state: zone } = useLoad(() => loadTimeZone(fhir, organizationId), [fhir, organizationId])
  if (zone.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  if (zone.name === 'failed') return <Alert message={t('receipts.report.loadFailed')} />
  return <Report api={api} zone={zone.value} start={thisMonth(now(), zone.value)} />
}

function Report({ api, zone, start }: { api: ApiClient; zone: string; start: Period }) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  const [period, setPeriod] = useState(start)
  const range = instants(period, zone)
  const { state } = useLoad(async () => {
    if (!range) return undefined
    const res = await attempt(api.GET('/clinic/reports/receipts', { params: { query: range } }))
    if (!res.ok || typeof res.data === 'string') throw new Error('report') // a string is the CSV, never asked for here
    return res.data
  }, [api, range?.from, range?.to])
  const report = state.name === 'loaded' ? state.value : undefined

  return (
    <section className="panel">
      <h1>{t('receipts.report.title')}</h1>
      <div className="toolbar">
        <ReceiptPeriod period={period} onChange={setPeriod} />
        {range && <a href={reportCsvURL(range.from, range.to)}>{t('receipts.report.csv')}</a>}
        <Link to="/receipts">{t('receipts.report.back')}</Link>
      </div>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('receipts.report.loadFailed')} />}
      {report && report.count === 0 && <p className="muted">{t('receipts.empty')}</p>}
      {report && report.count > 0 && (
        <>
          <dl className="facts">
            <dt>{t('receipts.report.count')}</dt>
            <dd>{f.number(report.count)}</dd>
            <dt>{t('receipts.report.total')}</dt>
            <dd>{money(report.totalMinor, report.currency, language)}</dd>
          </dl>
          <table>
            <caption>{t('receipts.report.byMethod')}</caption>
            <thead>
              <tr>
                <th scope="col">{t('receipts.report.method')}</th>
                <th scope="col">{t('receipts.report.receipts')}</th>
                <th scope="col">{t('receipts.report.amount')}</th>
              </tr>
            </thead>
            <tbody>
              {report.byPaymentMethod.map((m) => (
                <tr key={m.paymentMethod}>
                  <th scope="row">{t(`receipts.method.${m.paymentMethod}`)}</th>
                  <td>{f.number(m.count)}</td>
                  <td>{money(m.totalMinor, report.currency, language)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <table>
            <caption>{t('receipts.report.byService')}</caption>
            <thead>
              <tr>
                <th scope="col">{t('receipts.report.service')}</th>
                <th scope="col">{t('receipts.report.quantity')}</th>
                <th scope="col">{t('receipts.report.amount')}</th>
              </tr>
            </thead>
            <tbody>
              {report.byService.map((s) => (
                <tr key={`${s.serviceId ?? ''}|${s.name}`}>
                  <th scope="row">{s.name}</th>
                  <td>{f.number(s.quantity)}</td>
                  <td>{money(s.totalMinor, report.currency, language)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </section>
  )
}
