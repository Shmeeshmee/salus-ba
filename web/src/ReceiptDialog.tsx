import { useId, useRef, useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { Dialog } from './chat/Dialog'
import { useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { money, paymentMethods, type PaymentMethod, type Receipt } from './receipts'
import { Alert, Field, Submit } from './ui'

type Service = components['schemas']['Service']
type Line = { key: number; serviceId: string; quantity: string; discount: string }
type Buyer = 'patient' | 'cash' | 'company'

type Props = {
  api: ApiClient
  // appointmentId is the visit the receipt is for; without one the receipt is a cash sale or a company's invoice.
  appointmentId?: string
  // serviceId is the service the visit's booking named: the first line starts with it (clinic-cms E).
  serviceId?: string
  onIssued: (receipt: Receipt) => void
  onClose: () => void
}

// ReceiptDialog issues a receipt (POST /clinic/receipts): lines picked from the clinic's active price list with a
// quantity and a discount, how it was paid, and who it is for: the visit's patient, a company, or nobody (a cash
// sale). The total shown is the price list's; the server prices the lines again at issue. An issued receipt never
// changes, which the dialog says before the button.
export function ReceiptDialog({ api, appointmentId, serviceId, onIssued, onClose }: Props) {
  const t = useT()
  const { language } = useLanguage()
  const id = useId()
  const [lines, setLines] = useState<Line[]>([{ key: 0, serviceId: serviceId ?? '', quantity: '1', discount: '' }])
  const [buyer, setBuyer] = useState<Buyer>(appointmentId ? 'patient' : 'cash')
  const [companyName, setCompanyName] = useState('')
  const [companyTaxId, setCompanyTaxId] = useState('')
  const [method, setMethod] = useState<PaymentMethod>('cash')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  // One key for every attempt of this receipt as it stands: a double click or a retry after a lost answer issues it
  // once; a receipt changed after a failed attempt is another receipt and gets a new key (the server answers a known
  // key with the receipt it first issued).
  const sentKey = useRef({ draft: '', key: '' })
  const keyFor = (draft: string) => {
    if (sentKey.current.draft !== draft) sentKey.current = { draft, key: crypto.randomUUID() }
    return sentKey.current.key
  }
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/services'))
    if (!res.ok) throw new Error('services')
    return res.data.filter((s) => s.active)
  }, [api])

  const services = state.name === 'loaded' ? state.value : []
  const picks = lines.map((l) => pick(l, services))
  const valid = picks.every((p) => p !== undefined) && (buyer !== 'company' || (companyName.trim() !== '' && /^(\d{12,13})?$/.test(companyTaxId)))
  const currency = services[0]?.currency ?? 'BAM'
  const total = picks.reduce((sum, p) => sum + (p ? p.total : 0), 0)

  const change = (key: number, part: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...part } : l)))

  async function issue(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(undefined)
    const body = {
      kind: buyer === 'company' ? ('invoice' as const) : ('receipt' as const),
      paymentMethod: method,
      appointmentId,
      companyName: buyer === 'company' ? companyName.trim() : undefined,
      companyTaxId: buyer === 'company' && companyTaxId !== '' ? companyTaxId : undefined,
      lines: picks.map((p) => ({ serviceId: p!.serviceId, quantity: p!.quantity, discountMinor: p!.discountMinor })),
    }
    const res = await attempt(api.POST('/clinic/receipts', { params: { header: { 'Idempotency-Key': keyFor(JSON.stringify(body)) } }, body }))
    setBusy(false)
    if (res.ok) onIssued(res.data)
    else setError(message(t, res.code))
  }

  return (
    <Dialog label={t('receipts.dialog.title')} className="receipt-dialog" onClose={onClose}>
      <form onSubmit={issue}>
        <h2>{t('receipts.dialog.title')}</h2>
        {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {state.name === 'failed' && <Alert message={t('receipts.dialog.servicesFailed')} />}
        {state.name === 'loaded' && services.length === 0 && <Alert message={t('receipts.dialog.noServices')} />}
        {services.length > 0 && (
          <>
            {lines.map((l, i) => (
              <fieldset key={l.key} className="receipt-line">
                <legend>{t('receipts.dialog.line', { n: i + 1 })}</legend>
                <div className="field">
                  <label htmlFor={`${id}-s${l.key}`}>{t('receipts.dialog.service')}</label>
                  <select id={`${id}-s${l.key}`} value={l.serviceId} onChange={(e) => change(l.key, { serviceId: e.target.value })}>
                    <option value="">{t('receipts.dialog.chooseService')}</option>
                    {services.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name} · {money(s.priceMinor, s.currency, language)}
                      </option>
                    ))}
                  </select>
                </div>
                <Field label={t('receipts.dialog.quantity')} type="number" min={1} step={1} value={l.quantity} onChange={(e) => change(l.key, { quantity: e.target.value })} />
                <Field
                  label={t('receipts.dialog.discount', { currency })}
                  type="number"
                  min={0}
                  step="0.01"
                  value={l.discount}
                  onChange={(e) => change(l.key, { discount: e.target.value })}
                />
                {lines.length > 1 && (
                  <button type="button" className="quiet" onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}>
                    {t('receipts.dialog.removeLine', { n: i + 1 })}
                  </button>
                )}
              </fieldset>
            ))}
            <button
              type="button"
              className="quiet"
              onClick={() => setLines((ls) => [...ls, { key: Math.max(...ls.map((x) => x.key)) + 1, serviceId: '', quantity: '1', discount: '' }])}
            >
              {t('receipts.dialog.addLine')}
            </button>
            <div className="field">
              <label htmlFor={`${id}-b`}>{t('receipts.dialog.buyer')}</label>
              <select id={`${id}-b`} value={buyer} onChange={(e) => setBuyer(e.target.value as Buyer)}>
                {appointmentId ? <option value="patient">{t('receipts.dialog.visitPatient')}</option> : <option value="cash">{t('receipts.buyer.cash')}</option>}
                <option value="company">{t('receipts.buyer.company')}</option>
              </select>
            </div>
            {buyer === 'company' && (
              <>
                <Field label={t('receipts.dialog.companyName')} value={companyName} required onChange={(e) => setCompanyName(e.target.value)} />
                <Field label={t('receipts.dialog.companyTaxId')} value={companyTaxId} inputMode="numeric" pattern="\d{12,13}" onChange={(e) => setCompanyTaxId(e.target.value.trim())} />
              </>
            )}
            <div className="field">
              <label htmlFor={`${id}-m`}>{t('receipts.dialog.payment')}</label>
              <select id={`${id}-m`} value={method} onChange={(e) => setMethod(e.target.value as PaymentMethod)}>
                {paymentMethods.map((m) => (
                  <option key={m} value={m}>
                    {t(`receipts.method.${m}`)}
                  </option>
                ))}
              </select>
            </div>
            <p className="strong" aria-live="polite">
              {t('receipts.dialog.total', { total: money(total, currency, language) })}
            </p>
            <p className="muted small">{t('receipts.dialog.effect')}</p>
          </>
        )}
        <Alert message={error} />
        <div className="actions">
          {services.length > 0 && <Submit busy={busy} disabled={!valid} label={t('receipts.dialog.issue')} busyLabel={t('receipts.dialog.issuing')} />}
          <button type="button" disabled={busy} onClick={onClose}>
            {t('receipts.dialog.cancel')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

// pick is the line as the API takes it, with its total from the price list; undefined while the line is not
// complete: no service, a quantity that is no whole number from 1, or a discount below zero or above the line.
function pick(l: Line, services: Service[]) {
  const service = services.find((s) => s.id === l.serviceId)
  const quantity = Number(l.quantity)
  const discountMinor = l.discount === '' ? 0 : Math.round(Number(l.discount) * 100)
  if (!service || !Number.isInteger(quantity) || quantity < 1 || !Number.isFinite(discountMinor) || discountMinor < 0) return undefined
  const total = quantity * service.priceMinor - discountMinor
  return total < 0 ? undefined : { serviceId: service.id, quantity, discountMinor, total }
}

function message(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'invalid_receipt' || code === 'invalid_request') return t('receipts.dialog.invalid')
  if (code === 'billing_details_missing') return t('receipts.dialog.billingDetailsMissing')
  if (code === 'forbidden' || code === 'not_found') return t('receipts.dialog.forbidden')
  return t('common.somethingWrong')
}
