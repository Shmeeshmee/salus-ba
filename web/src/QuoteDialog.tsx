import { useId, useState, type FormEvent } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { Dialog } from './chat/Dialog'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { money } from './receipts'
import { Alert, Field, Submit } from './ui'

type Service = components['schemas']['Service']
export type Quote = components['schemas']['Quote']
type Line = { key: number; serviceId: string; quantity: string; discount: string }

type Props = {
  api: ApiClient
  appointmentId: string // the visit the quote is written at
  patientId: string // the visit's patient
  serviceId?: string // the service the booking named: the first line starts with it
  onSent: (quote: Quote) => void
  onClose: () => void
}

// QuoteDialog sends the visit's patient a treatment quote (POST /clinic/quotes, clinic-cms X1): lines picked from the
// clinic's active price list with a quantity and a discount, and a note. The total shown is the price list's; the
// server prices the lines again. The dialog says before the button that a quote is no receipt and charges nothing.
// The line fields are the receipt dialog's, with its words.
export function QuoteDialog({ api, appointmentId, patientId, serviceId, onSent, onClose }: Props) {
  const t = useT()
  const { language } = useLanguage()
  const id = useId()
  const [lines, setLines] = useState<Line[]>([{ key: 0, serviceId: serviceId ?? '', quantity: '1', discount: '' }])
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/services'))
    if (!res.ok) throw new Error('services')
    return res.data.filter((s) => s.active)
  }, [api])

  const services = state.name === 'loaded' ? state.value : []
  const picks = lines.map((l) => pick(l, services))
  const valid = picks.every((p) => p !== undefined)
  const currency = services[0]?.currency ?? 'BAM'
  const total = picks.reduce((sum, p) => sum + (p ? p.total : 0), 0)
  const change = (key: number, part: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...part } : l)))

  async function send(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(undefined)
    const body = {
      patientId,
      appointmentId,
      note: note.trim() === '' ? undefined : note.trim(),
      lines: picks.map((p) => ({ serviceId: p!.serviceId, quantity: p!.quantity, discountMinor: p!.discountMinor })),
    }
    const res = await attempt(api.POST('/clinic/quotes', { body }))
    setBusy(false)
    if (res.ok) onSent(res.data)
    else setError(message(t, res.code))
  }

  return (
    <Dialog label={t('receipts.quote.dialogTitle')} className="receipt-dialog" onClose={onClose}>
      <form onSubmit={send}>
        <h2>{t('receipts.quote.dialogTitle')}</h2>
        {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {state.name === 'failed' && <Alert message={t('receipts.dialog.servicesFailed')} />}
        {state.name === 'loaded' && services.length === 0 && <Alert message={t('receipts.quote.noServices')} />}
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
              <label htmlFor={`${id}-n`}>{t('receipts.quote.note')}</label>
              <textarea id={`${id}-n`} maxLength={2000} rows={3} value={note} onChange={(e) => setNote(e.target.value)} />
            </div>
            <p className="strong" aria-live="polite">
              {t('receipts.dialog.total', { total: money(total, currency, language) })}
            </p>
            <p className="muted small">{t('receipts.quote.effect')}</p>
          </>
        )}
        <Alert message={error} />
        <div className="actions">
          {services.length > 0 && <Submit busy={busy} disabled={!valid} label={t('receipts.quote.send')} busyLabel={t('receipts.quote.sending')} />}
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
  if (code === 'invalid_quote' || code === 'invalid_request') return t('receipts.quote.invalid')
  if (code === 'forbidden' || code === 'not_found') return t('receipts.quote.refused')
  return t('common.somethingWrong')
}

// VisitQuotes lists the quotes written at the visit, newest first, with their state (sent, expired) and validity in
// the clinic's zone (GET /clinic/quotes?appointmentId=: the admin's every quote of the visit, a practitioner's own).
export function VisitQuotes({ api, appointmentId, zone }: { api: ApiClient; appointmentId: string; zone: string }) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/quotes', { params: { query: { appointmentId } } }))
    if (!res.ok) throw new Error('quotes')
    return res.data
  }, [api, appointmentId])
  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('receipts.quote.loadFailed')} />
  if (state.value.length === 0) return <p className="muted">{t('receipts.quote.empty')}</p>
  return (
    <ul className="quotes">
      {state.value.map((q) => (
        <li key={q.id}>
          <span className={`tag ${q.state}`}>{t(`receipts.quote.state.${q.state}`)}</span>{' '}
          {t('receipts.quote.item', {
            issued: f.date(new Date(q.issuedAt), zone, 'long'),
            total: money(q.totalMinor, q.currency, language),
            until: f.date(new Date(q.validUntil), zone, 'long'),
          })}
        </li>
      ))}
    </ul>
  )
}
