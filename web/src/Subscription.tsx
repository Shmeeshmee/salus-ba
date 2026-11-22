import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadTimeZone } from './staff'
import { Alert, Field, Link, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Membership = components['schemas']['Subscription']
type Invoice = components['schemas']['SubscriptionInvoice']
type Details = components['schemas']['BillingDetails']

// The admin's banner shows from 14 days past the due date (docs/plans/clinic-cms.md Q4); new bookings stop at 30,
// which the api decides (status suspended).
const bannerAfterMs = 14 * 24 * 60 * 60 * 1000

type Props = { api: ApiClient; fhir: FhirClient; account: Account }

// loadWithZone reads the clinic's membership with the clinic's time zone, in which its periods and dates are shown.
async function loadWithZone(api: ApiClient, fhir: FhirClient, organizationId: string) {
  const [zone, m] = await Promise.all([loadTimeZone(fhir, organizationId), loadMembership(api)])
  return { zone, m }
}

// loadMembership reads the clinic's membership; null for a clinic that has none yet (404).
async function loadMembership(api: ApiClient): Promise<Membership | null> {
  const res = await attempt(api.GET('/clinic/subscription'))
  if (res.ok) return res.data
  if (res.response?.status === 404) return null
  throw new Error('subscription')
}

// Subscription is the admin's membership screen (package D of docs/plans/clinic-cms.md): the clinic's plan and status,
// Salus's monthly invoices to it, and its billing details. Nothing here is personal data.
export function Subscription({ api, fhir, account }: Props) {
  const t = useT()
  const f = useFormat()
  const organizationId = account.organizationId ?? ''
  const { state: loaded } = useLoad(() => loadWithZone(api, fhir, organizationId), [api, fhir, organizationId])
  const state = loaded.name === 'loaded' ? { name: 'loaded' as const, value: loaded.value.m } : loaded
  const zone = () => (loaded.name === 'loaded' ? loaded.value.zone : '')
  return (
    <>
      <section className="panel">
        <h1>{t('subscription.title')}</h1>
        {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {state.name === 'failed' && <Alert message={t('subscription.loadFailed')} />}
        {state.name === 'loaded' && state.value === null && <p className="muted">{t('subscription.none')}</p>}
        {state.name === 'loaded' && state.value !== null && (
          <dl className="facts">
            <dt>{t('subscription.plan')}</dt>
            <dd>
              {state.value.planName}
              <span className="hint"> {t('subscription.price', { price: f.money(state.value.pricePerSeatMinor, state.value.currency) })}</span>
            </dd>
            <dt>{t('subscription.status')}</dt>
            <dd>
              {t(`subscription.status.${state.value.status}`)}
              {state.value.status === 'trial' && state.value.trialEndsAt && (
                <span className="hint"> {t('subscription.trialEnds', { date: f.date(new Date(state.value.trialEndsAt), zone(), 'long') })}</span>
              )}
              {state.value.currentPeriodStart && state.value.currentPeriodEnd && (
                <span className="hint">
                  {' '}
                  {t('subscription.period', {
                    from: f.date(new Date(state.value.currentPeriodStart), zone(), 'long'),
                    to: f.date(new Date(state.value.currentPeriodEnd), zone(), 'long'),
                  })}
                </span>
              )}
            </dd>
          </dl>
        )}
      </section>
      {loaded.name === 'loaded' && <Invoices api={api} zone={loaded.value.zone} />}
      <BillingDetails api={api} />
    </>
  )
}

function Invoices({ api, zone }: { api: ApiClient; zone: string }) {
  const t = useT()
  const f = useFormat()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/subscription/invoices'))
    if (res.ok) return res.data
    if (res.response?.status === 404) return [] as Invoice[]
    throw new Error('invoices')
  }, [api])
  return (
    <section className="panel membership-invoices">
      <h2>{t('subscription.invoicesTitle')}</h2>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('subscription.invoicesLoadFailed')} />}
      {state.name === 'loaded' && state.value.length === 0 && <p className="muted">{t('subscription.invoicesEmpty')}</p>}
      {state.name === 'loaded' && state.value.length > 0 && (
        <table className="plain-table">
          <thead>
            <tr>
              <th scope="col">{t('subscription.col.number')}</th>
              <th scope="col">{t('subscription.col.period')}</th>
              <th scope="col">{t('subscription.col.seats')}</th>
              <th scope="col">{t('subscription.col.amount')}</th>
              <th scope="col">{t('subscription.col.due')}</th>
              <th scope="col">{t('subscription.col.status')}</th>
            </tr>
          </thead>
          <tbody>
            {state.value.map((inv) => (
              <tr key={inv.id}>
                <td>{inv.number}</td>
                <td>{t('subscription.periodRange', { from: f.calendarDate(inv.periodStart), to: f.calendarDate(lastDay(inv.periodEnd)) })}</td>
                <td>{f.number(inv.seats)}</td>
                <td>{f.money(inv.amountMinor, inv.currency)}</td>
                <td>{f.date(new Date(inv.dueAt), zone, 'short')}</td>
                <td>{t(`subscription.invoice.${inv.status}`)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}

// lastDay is the day before the exclusive end date ("2027-01-01" gives "2026-12-31").
function lastDay(end: string): string {
  const d = new Date(`${end}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() - 1)
  return d.toISOString().slice(0, 10)
}

function BillingDetails({ api }: { api: ApiClient }) {
  const t = useT()
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/billing-details'))
    if (!res.ok) throw new Error('billing')
    return res.data
  }, [api])
  return (
    <section className="panel">
      <h2>{t('subscription.billingTitle')}</h2>
      <p className="hint">{t('subscription.billingHint')}</p>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('subscription.billingLoadFailed')} />}
      {state.name === 'loaded' && <BillingForm api={api} details={state.value} onSaved={set} />}
    </section>
  )
}

function BillingForm({ api, details, onSaved }: { api: ApiClient; details: Details; onSaved: (d: Details) => void }) {
  const t = useT()
  const [legalName, setLegalName] = useState(details.legalName ?? '')
  const [taxId, setTaxId] = useState(details.taxId ?? '')
  const [vat, setVat] = useState(details.vatRegistered)
  const [email, setEmail] = useState(details.billingEmail ?? '')
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const tax = taxId.trim()
    if (tax !== '' && !/^[0-9]{12,13}$/.test(tax)) return t('subscription.billingInvalid')
    // A detail left empty is not stated: it is left out, as the schema's optional fields are.
    const body: Details = { vatRegistered: vat }
    if (legalName.trim() !== '') body.legalName = legalName.trim()
    if (tax !== '') body.taxId = tax
    if (email.trim() !== '') body.billingEmail = email.trim()
    const res = await attempt(api.PUT('/clinic/billing-details', { body }))
    if (!res.ok) return res.code === 'invalid_request' ? t('subscription.billingInvalid') : t('subscription.billingSaveFailed')
    onSaved(res.data)
    setSaved(true)
    return undefined
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field label={t('subscription.legalName')} value={legalName} maxLength={200} onChange={(e) => setLegalName(e.target.value)} />
      <Field label={t('subscription.taxId')} hint={t('subscription.taxIdHint')} inputMode="numeric" value={taxId} onChange={(e) => setTaxId(e.target.value)} />
      <label className="check">
        <input type="checkbox" checked={vat} onChange={(e) => setVat(e.target.checked)} />
        <span>{t('subscription.vatRegistered')}</span>
      </label>
      <Field label={t('subscription.billingEmail')} hint={t('subscription.billingEmailHint')} type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
      <Alert message={error} />
      {saved && <Notice message={t('subscription.billingSaved')} />}
      <Submit busy={busy} label={t('common.save')} busyLabel={t('common.saving')} />
    </form>
  )
}

// SubscriptionBanner tells the clinic's admin, on every screen, that an invoice is 14 days past due or that the
// membership is suspended. It shows nothing while loading, without a membership, or when the read fails: the banner is
// a reminder, never in the way of the work.
export function SubscriptionBanner({ api, fhir, account }: Props) {
  const t = useT()
  const f = useFormat()
  const organizationId = account.organizationId ?? ''
  // The clock is read with the answer, not while rendering.
  const { state } = useLoad(async () => {
    const { zone, m } = await loadWithZone(api, fhir, organizationId).catch(() => ({ zone: '', m: null }))
    const overdue = m?.pastDueSince !== undefined && Date.now() - new Date(m.pastDueSince).getTime() >= bannerAfterMs
    return m === null ? null : { m, overdue, zone }
  }, [api, fhir, organizationId])
  if (state.name !== 'loaded' || state.value === null) return null
  const { m, overdue, zone } = state.value
  if (m.status !== 'suspended' && !overdue) return null
  return (
    <div className="alert" role="alert">
      {m.status === 'suspended'
        ? t('subscription.bannerSuspended')
        : t('subscription.bannerPastDue', { date: f.date(new Date(m.pastDueSince ?? ''), zone, 'long') })}{' '}
      <Link to="/membership">{t('subscription.bannerLink')}</Link>
    </div>
  )
}
