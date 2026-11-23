import { useId, useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { formatNumber } from './i18n/format'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { ReceiptPeriod } from './ReceiptPeriod'
import { instants, money, thisMonth, type Period } from './receipts'
import { loadStaff, loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; account: Account; now?: () => Date }

// OnlinePractice is the online practice report for the clinic's admin (clinic-cms E): for a period of the clinic's
// calendar, each practitioner's hours offered, visits held and the amount receipted for them, at the clinic and by
// video, beside the receipts report. The hours follow the availability as it is now.
export function OnlinePractice({ api, fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state } = useLoad(async () => {
    const [zone, staff] = await Promise.all([loadTimeZone(fhir, organizationId), loadStaff(fhir, organizationId)])
    return { zone, names: new Map(staff.map((m) => [m.roleId, m.name])) }
  }, [fhir, organizationId])
  if (state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('receipts.practice.loadFailed')} />
  return <Report api={api} zone={state.value.zone} names={state.value.names} start={thisMonth(now(), state.value.zone)} />
}

function Report({ api, zone, names, start }: { api: ApiClient; zone: string; names: Map<string, string>; start: Period }) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  const [period, setPeriod] = useState(start)
  const range = instants(period, zone)
  const { state } = useLoad(async () => {
    if (!range) return undefined
    const res = await attempt(api.GET('/clinic/reports/practice', { params: { query: range } }))
    if (!res.ok) throw new Error('report')
    return res.data
  }, [api, range?.from, range?.to])
  const report = state.name === 'loaded' ? state.value : undefined

  return (
    <section className="panel">
      <h1>{t('receipts.practice.title')}</h1>
      <div className="toolbar">
        <ReceiptPeriod period={period} onChange={setPeriod} />
        <Link to="/receipts/report">{t('receipts.practice.back')}</Link>
      </div>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('receipts.practice.loadFailed')} />}
      {report && report.rows.length === 0 && <p className="muted">{t('receipts.practice.empty')}</p>}
      {report && report.rows.length > 0 && (
        <table className="report-table">
          <caption>{t('receipts.practice.caption')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('receipts.practice.practitioner')}</th>
              <th scope="col">{t('receipts.practice.mode')}</th>
              <th scope="col">
                <HoursOffered />
              </th>
              <th scope="col">{t('receipts.practice.visits')}</th>
              <th scope="col">{t('receipts.practice.amount')}</th>
            </tr>
          </thead>
          <tbody>
            {report.rows.map((r) => (
              <tr key={`${r.practitionerRole}|${r.visitMode}`}>
                <th scope="row">{names.get(r.practitionerRole) ?? t('receipts.practice.former')}</th>
                <td>{t(r.visitMode === 'in-person' ? 'receipts.practice.inPerson' : 'receipts.practice.video')}</td>
                <td>{formatNumber(r.minutesOffered / 60, language, { maximumFractionDigits: 1 })}</td>
                <td>{f.number(r.visits)}</td>
                <td>{money(r.receiptedMinor, report.currency, language)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}

// HoursOffered is the column's heading with its hint on hover and focus: the hours of a past period come from the
// working hours as they are today, since earlier ones are not kept (clinic-cms E, wave 2 integration).
function HoursOffered() {
  const t = useT()
  const id = useId()
  return (
    <span className="tip">
      <span className="has-hint" tabIndex={0} aria-describedby={id}>
        {t('receipts.practice.hours')}
      </span>
      <span role="tooltip" id={id}>
        {t('receipts.practice.hoursHint')}
      </span>
    </span>
  )
}
