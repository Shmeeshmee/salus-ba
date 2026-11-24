import { useId, useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import type { components } from './api/schema'
import { attempt } from './api/result'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { money } from './receipts'
import { loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type QuoteState = components['schemas']['QuoteState']
type Filter = QuoteState | 'all'
const filters: Filter[] = ['all', 'sent', 'expired']

type Props = { api: ApiClient; fhir: FhirClient; account: Account }

// Quotes is the clinic's treatment quotes for its admin (clinic-cms M4): newest first, each with its state (sent,
// expired), total and validity in the clinic's zone, filtered by state, and its PDF. A quote is written at a visit.
export function Quotes({ api, fhir, account }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state: zone } = useLoad(() => loadTimeZone(fhir, organizationId), [fhir, organizationId])
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/quotes'))
    if (!res.ok) throw new Error('quotes')
    return res.data
  }, [api])
  const [filter, setFilter] = useState<Filter>('all')
  const id = useId()

  if (zone.name === 'failed' || state.name === 'failed') return <Alert message={t('receipts.quote.loadFailed')} />
  if (zone.name === 'loading' || state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  const shown = state.value.filter((q) => filter === 'all' || q.state === filter)
  return (
    <section className="panel">
      <h1>{t('receipts.quotes.nav')}</h1>
      <div className="toolbar">
        <label htmlFor={id}>{t('receipts.quotes.state')}</label>
        <select id={id} value={filter} onChange={(e) => setFilter(e.target.value as Filter)}>
          {filters.map((f) => (
            <option key={f} value={f}>
              {t(f === 'all' ? 'receipts.quotes.all' : `receipts.quote.state.${f}`)}
            </option>
          ))}
        </select>
        <Link to="/receipts">{t('receipts.nav')}</Link>
      </div>
      {state.value.length === 0 && <p className="muted">{t('receipts.quotes.empty')}</p>}
      {state.value.length > 0 && shown.length === 0 && <p className="muted">{t('receipts.quotes.emptyState')}</p>}
      {shown.length > 0 && (
        <ol className="rows" aria-label={t('receipts.quotes.nav')}>
          {shown.map((q) => (
            <Row key={q.id} quote={q} zone={zone.value} />
          ))}
        </ol>
      )}
    </section>
  )
}

function Row({ quote: q, zone }: { quote: components['schemas']['Quote']; zone: string }) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  return (
    <li>
      <span className="stack">
        <span>
          {t('receipts.quote.item', {
            issued: f.date(new Date(q.issuedAt), zone, 'long'),
            total: money(q.totalMinor, q.currency, language),
            until: f.date(new Date(q.validUntil), zone, 'long'),
          })}
        </span>
        <span className="actions">
          <span className={`tag ${q.state}`}>{t(`receipts.quote.state.${q.state}`)}</span>
        </span>
      </span>
      <span className="actions">
        <a href={quotePdfURL(q.id)} target="_blank" rel="noreferrer">
          {t('receipts.pdf')}
        </a>
      </span>
    </li>
  )
}

// quotePdfURL is where a quote's PDF is read, on the same origin as every other call, so the session cookie goes with
// it.
function quotePdfURL(id: string): string {
  return `${globalThis.location.origin}/clinic/quotes/${encodeURIComponent(id)}/pdf`
}
