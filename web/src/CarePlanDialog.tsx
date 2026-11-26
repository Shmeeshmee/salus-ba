import { useId, useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { planMessage, type CarePlan } from './carePlan'
import { Dialog } from './chat/Dialog'
import { useFormat, useLanguage, useT } from './i18n/useT'
import { useLoad, type Load } from './load'
import { money } from './receipts'
import { Alert, Field, Submit } from './ui'
import { visitModes, type VisitMode } from './visitMode'

type Service = components['schemas']['Service']
type Step = { key: number; serviceId: string; dayOffset: string; visitMode: VisitMode }

type Props = {
  api: ApiClient
  patientId: string
  appointmentId: string // the visit's quotes are offered as the plan's quote
  plan?: CarePlan // the draft to change; a new one without it
  services: Load<Service[]>
  zone: string
  onSaved: (plan: CarePlan) => void
  onClose: () => void
}

// CarePlanDialog lays out a treatment plan or changes a draft (POST /clinic/care-plans, PUT at the version read):
// a title, ordered steps (each a service of the active price list, the days from the plan's first day and a visit
// type the service is offered in), moved up and down in place, and optionally one of the visit's quotes (X1).
export function CarePlanDialog({ api, patientId, appointmentId, plan, services, zone, onSaved, onClose }: Props) {
  const t = useT()
  const id = useId()
  const [title, setTitle] = useState(plan?.title ?? '')
  const [quoteId, setQuoteId] = useState(plan?.quoteId ?? '')
  const [steps, setSteps] = useState<Step[]>(
    plan ? plan.steps.map((s, i) => ({ key: i, serviceId: s.serviceId, dayOffset: String(s.dayOffset), visitMode: s.visitMode })) : [newStep(0)],
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const offered = services.name === 'loaded' ? services.value.filter((s) => s.active || steps.some((x) => x.serviceId === s.id)) : []
  const valid = title.trim() !== '' && steps.every((s) => stepValid(s, offered))
  const change = (key: number, part: Partial<Step>) => setSteps((all) => all.map((s) => (s.key === key ? fit({ ...s, ...part }, offered) : s)))
  const move = (from: number, to: number) =>
    setSteps((all) => {
      const next = [...all]
      const [s] = next.splice(from, 1)
      next.splice(to, 0, s)
      return next
    })

  async function save(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(undefined)
    const body = {
      title: title.trim(),
      quoteId: quoteId === '' ? undefined : quoteId,
      steps: steps.map((s) => ({ serviceId: s.serviceId, dayOffset: Number(s.dayOffset), visitMode: s.visitMode })),
    }
    const res = plan
      ? await attempt(api.PUT('/clinic/care-plans/{id}', { params: { path: { id: plan.id }, header: { 'If-Match': `W/"${plan.versionId}"` } }, body }))
      : await attempt(api.POST('/clinic/care-plans', { body: { patientId, ...body } }))
    setBusy(false)
    if (res.ok) onSaved(res.data)
    else setError(planMessage(t, res.code))
  }

  const label = t(plan ? 'carePlans.dialogChange' : 'carePlans.dialogNew')
  return (
    <Dialog label={label} className="receipt-dialog care-plan-dialog" onClose={onClose}>
      <form onSubmit={save}>
        <h2>{label}</h2>
        <p className="muted small">{t('carePlans.draftHint')}</p>
        <Field label={t('carePlans.planTitle')} maxLength={200} value={title} onChange={(e) => setTitle(e.target.value)} />
        {services.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {services.name === 'failed' && <Alert message={t('carePlans.servicesFailed')} />}
        {steps.map((s, i) => (
          <fieldset key={s.key} className="receipt-line">
            <legend>{t('carePlans.stepLegend', { n: i + 1 })}</legend>
            <div className="field">
              <label htmlFor={`${id}-s${s.key}`}>{t('carePlans.service')}</label>
              <select id={`${id}-s${s.key}`} value={s.serviceId} onChange={(e) => change(s.key, { serviceId: e.target.value })}>
                <option value="">{t('carePlans.chooseService')}</option>
                {offered.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
              </select>
            </div>
            <Field
              label={t('carePlans.dayOffset')}
              type="number"
              min={0}
              max={730}
              step={1}
              value={s.dayOffset}
              onChange={(e) => change(s.key, { dayOffset: e.target.value })}
            />
            <div className="field">
              <label htmlFor={`${id}-m${s.key}`}>{t('carePlans.visitMode')}</label>
              <select id={`${id}-m${s.key}`} value={s.visitMode} onChange={(e) => change(s.key, { visitMode: e.target.value as VisitMode })}>
                {modesOf(s, offered).map((m) => (
                  <option key={m} value={m}>
                    {t(m === 'video' ? 'carePlans.mode.videoLabel' : 'carePlans.mode.inPersonLabel')}
                  </option>
                ))}
              </select>
            </div>
            <div className="actions">
              {i > 0 && (
                <button type="button" className="quiet" onClick={() => move(i, i - 1)}>
                  {t('carePlans.moveUp', { n: i + 1 })}
                </button>
              )}
              {i < steps.length - 1 && (
                <button type="button" className="quiet" onClick={() => move(i, i + 1)}>
                  {t('carePlans.moveDown', { n: i + 1 })}
                </button>
              )}
              {steps.length > 1 && (
                <button type="button" className="quiet" onClick={() => setSteps((all) => all.filter((x) => x.key !== s.key))}>
                  {t('carePlans.removeStep', { n: i + 1 })}
                </button>
              )}
            </div>
          </fieldset>
        ))}
        {steps.length < 50 && (
          <div className="actions">
            <button type="button" onClick={() => setSteps((all) => [...all, newStep(Math.max(...all.map((s) => s.key)) + 1)])}>
              {t('carePlans.addStep')}
            </button>
          </div>
        )}
        <QuotePick api={api} appointmentId={appointmentId} zone={zone} value={quoteId} onChange={setQuoteId} />
        <Alert message={error} />
        <div className="actions">
          <Submit busy={busy} disabled={!valid} label={t('carePlans.save')} busyLabel={t('carePlans.saving')} />
          <button type="button" disabled={busy} onClick={onClose}>
            {t('carePlans.close')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

// QuotePick offers the visit's quotes as the plan's quote; it is left out while the visit has none (and the plan
// names none).
function QuotePick({ api, appointmentId, zone, value, onChange }: { api: ApiClient; appointmentId: string; zone: string; value: string; onChange: (id: string) => void }) {
  const t = useT()
  const f = useFormat()
  const { language } = useLanguage()
  const id = useId()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/quotes', { params: { query: { appointmentId } } }))
    return res.ok ? res.data : []
  }, [api, appointmentId])
  const quotes = state.name === 'loaded' ? state.value : []
  if (quotes.length === 0 && value === '') return null
  return (
    <div className="field">
      <label htmlFor={id}>{t('carePlans.quote')}</label>
      <select id={id} value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{t('carePlans.noQuote')}</option>
        {value !== '' && !quotes.some((q) => q.id === value) && <option value={value}>{t('carePlans.otherQuote')}</option>}
        {quotes.map((q) => (
          <option key={q.id} value={q.id}>
            {t('receipts.quote.item', {
              issued: f.date(new Date(q.issuedAt), zone, 'long'),
              total: money(q.totalMinor, q.currency, language),
              until: f.date(new Date(q.validUntil), zone, 'long'),
            })}
          </option>
        ))}
      </select>
    </div>
  )
}

function newStep(key: number): Step {
  return { key, serviceId: '', dayOffset: '0', visitMode: 'in-person' }
}

// modesOf is the visit types a step may take: its service's, or both while it has none.
function modesOf(s: Step, services: Service[]): VisitMode[] {
  const service = services.find((x) => x.id === s.serviceId)
  return service ? visitModes.filter((m) => service.modes.includes(m)) : visitModes
}

// fit keeps a step's visit type one its service is offered in.
function fit(s: Step, services: Service[]): Step {
  const modes = modesOf(s, services)
  return modes.includes(s.visitMode) || modes.length === 0 ? s : { ...s, visitMode: modes[0] }
}

function stepValid(s: Step, services: Service[]): boolean {
  const day = Number(s.dayOffset)
  return services.some((x) => x.id === s.serviceId) && s.dayOffset !== '' && Number.isInteger(day) && day >= 0 && day <= 730 && modesOf(s, services).includes(s.visitMode)
}
