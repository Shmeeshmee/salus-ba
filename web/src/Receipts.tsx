import { useEffect, useRef, useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { ReceiptDialog } from './ReceiptDialog'
import { ReceiptPeriod } from './ReceiptPeriod'
import { instants, money, paymentMethods, receiptPdfURL, thisMonth, type PaymentMethod, type Period, type Receipt } from './receipts'
import { loadTimeZone } from './staff'
import { Alert, Link, Notice } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; account: Account; now?: () => Date }

// Receipts is the clinic's receipts for its admin (clinic-cms C): the receipts issued in a period, newest first,
// each with its number, buyer, total, how it was paid and where its report to the tax authority stands; the admin
// issues a cash sale or a company's invoice, opens a receipt's PDF and reverses a receipt, once. A receipt for a
// visit is issued from the appointment.
export function Receipts({ api, fhir, account, now = () => new Date() }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state: zone } = useLoad(() => loadTimeZone(fhir, organizationId), [fhir, organizationId])
  if (zone.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  if (zone.name === 'failed') return <Alert message={t('receipts.loadFailed')} />
  return <List api={api} zone={zone.value} start={thisMonth(now(), zone.value)} />
}

function List({ api, zone, start }: { api: ApiClient; zone: string; start: Period }) {
  const t = useT()
  const [period, setPeriod] = useState(start)
  const [issuing, setIssuing] = useState(false)
  const [notice, setNotice] = useState<string>()
  const range = instants(period, zone)
  const { state, reload } = useLoad(async () => {
    if (!range) return []
    const res = await attempt(api.GET('/clinic/receipts', { params: { query: range } }))
    if (!res.ok) throw new Error('receipts')
    return res.data
  }, [api, range?.from, range?.to])

  return (
    <section className="panel">
      <h1>{t('receipts.title')}</h1>
      <div className="toolbar">
        <ReceiptPeriod period={period} onChange={setPeriod} />
        <button type="button" className="primary" onClick={() => setIssuing(true)}>
          {t('receipts.new')}
        </button>
        <Link to="/receipts/report">{t('receipts.openReport')}</Link>
      </div>
      <Notice message={notice} />
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('receipts.loadFailed')} />}
      {state.name === 'loaded' && state.value.length === 0 && <p className="muted">{t('receipts.empty')}</p>}
      {state.name === 'loaded' && state.value.length > 0 && (
        <ol className="rows" aria-label={t('receipts.title')}>
          {state.value.map((r) => (
            <Row
              key={r.id}
              api={api}
              receipt={r}
              zone={zone}
              onChanged={reload}
              onRetried={() => {
                setNotice(t('receipts.retried', { number: r.number }))
                reload()
              }}
            />
          ))}
        </ol>
      )}
      {issuing && (
        <ReceiptDialog
          api={api}
          onClose={() => setIssuing(false)}
          onIssued={(r) => {
            setIssuing(false)
            setNotice(t('receipts.issued', { number: r.number }))
            reload()
          }}
        />
      )}
    </section>
  )
}

type RowProps = { api: ApiClient; receipt: Receipt; zone: string; onChanged: () => void; onRetried: () => void }

function Row({ api, receipt: r, zone, onChanged, onRetried }: RowProps) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  const [asking, setAsking] = useState(false)
  const [method, setMethod] = useState<PaymentMethod>(r.paymentMethod)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  // Keep gives the focus back to Reverse, which the question replaced.
  const reverseButton = useRef<HTMLButtonElement>(null)
  const refocus = useRef(false)
  useEffect(() => {
    if (refocus.current && !asking) {
      reverseButton.current?.focus()
      refocus.current = false
    }
  }, [asking])
  const buyer = r.companyName ?? t(r.patientId ? 'receipts.buyer.patient' : 'receipts.buyer.cash')

  async function reverse() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/clinic/receipts/{id}/reverse', { params: { path: { id: r.id } }, body: { paymentMethod: method } }))
    setBusy(false)
    setAsking(false)
    if (res.ok) onChanged()
    else setError(t(res.code === 'already_reversed' ? 'receipts.alreadyReversed' : 'receipts.reverseFailed'))
  }

  // A receipt whose report failed is reported again by the same job (clinic-cms H).
  async function retry() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/clinic/receipts/{id}/retry', { params: { path: { id: r.id } } }))
    setBusy(false)
    if (res.ok) onRetried()
    else setError(t(res.code === 'fiscal_report_not_failed' ? 'receipts.notFailed' : 'receipts.retryFailed'))
  }

  return (
    <li>
      <span className="stack">
        <span>
          {t('receipts.number', { number: r.number })} · {money(r.totalMinor, r.currency, language)}
        </span>
        <span className="muted small">
          {f.dateTime(new Date(r.issuedAt), zone)} · {buyer} · {t(`receipts.method.${r.paymentMethod}`)}
        </span>
        <span className="actions">
          <span className={`tag fiscal-${r.fiscalStatus}`}>{t(`receipts.fiscal.${r.fiscalStatus}`)}</span>
          {r.reversesId && <span className="tag">{t('receipts.reversal')}</span>}
          {r.reversedById && <span className="tag">{t('receipts.reversed')}</span>}
        </span>
      </span>
      <span className="actions">
        <a href={receiptPdfURL(r.id)} target="_blank" rel="noreferrer">
          {t('receipts.pdf')}
        </a>
        {r.fiscalStatus === 'failed' && (
          <button type="button" disabled={busy} onClick={retry}>
            {t('receipts.retry')}
          </button>
        )}
        {/* A receipt whose report failed is not reversed: its reversal could not be reported either. */}
        {!r.reversesId && !r.reversedById && r.fiscalStatus !== 'failed' && !asking && (
          <button type="button" className="danger" ref={reverseButton} disabled={busy} onClick={() => setAsking(true)}>
            {t('receipts.reverse')}
          </button>
        )}
      </span>
      {error && <Alert message={error} />}
      {asking && (
        <div className="confirm" role="alertdialog" aria-label={t('receipts.reverseQuestion', { number: r.number })}>
          <p className="strong">{t('receipts.reverseQuestion', { number: r.number })}</p>
          <p>{t('receipts.reverseEffect')}</p>
          <label>
            <span className="muted small">{t('receipts.reverseMethod')} </span>
            {/* The question takes the focus from the Reverse button it replaces. */}
            <select autoFocus value={method} onChange={(e) => setMethod(e.target.value as PaymentMethod)}>
              {paymentMethods.map((m) => (
                <option key={m} value={m}>
                  {t(`receipts.method.${m}`)}
                </option>
              ))}
            </select>
          </label>
          <div className="actions">
            <button type="button" className="danger" disabled={busy} onClick={reverse}>
              {t('receipts.reverse')}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                refocus.current = true
                setAsking(false)
              }}
            >
              {t('receipts.keep')}
            </button>
          </div>
        </div>
      )}
    </li>
  )
}
