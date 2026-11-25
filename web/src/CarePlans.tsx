import { useState } from 'react'
import type { ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { dayStart, isoDay, planMessage, stepText, type CarePlan } from './carePlan'
import { CarePlanBook } from './CarePlanBook'
import { CarePlanDialog } from './CarePlanDialog'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Confirm, Notice } from './ui'

type Service = components['schemas']['Service']
export type { CarePlan }

type Props = {
  api: ApiClient
  fhir: FhirClient
  patientId: string
  appointmentId: string // the visit the plans are opened from: its quotes may go with a new plan
  practitionerId?: string // the signed-in practitioner, whose times a booking offers first
  zone: string
}

type Open = { name: 'write' } | { name: 'change'; plan: CarePlan } | { name: 'book'; plan: CarePlan } | { name: 'cancel'; plan: CarePlan }

// CarePlans is a patient's treatment plans at the clinic (clinic-cms U6, package TP2), opened from a visit by its
// practitioner or the clinic's admin: newest first, each with its state and its steps in order (a booked one with
// each step's day). A draft is laid out, changed and booked here; a booked plan is cancelled here, its future visits
// with it. A step of a booked plan moves by moving its visit, and the plan then shows the new order.
export function CarePlans({ api, fhir, patientId, appointmentId, practitionerId, zone }: Props) {
  const t = useT()
  const [open, setOpen] = useState<Open>()
  const [notice, setNotice] = useState<string>()
  const [today] = useState(() => isoDay(new Date(), zone)) // the first day a booking offers
  const plans = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/patients/{id}/care-plans', { params: { path: { id: patientId } } }))
    if (!res.ok) throw new Error('care plans')
    return res.data
  }, [api, patientId])
  const services = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/services'))
    if (!res.ok) throw new Error('services')
    return res.data
  }, [api])
  const priceList = services.state.name === 'loaded' ? services.state.value : []

  function stored(plan: CarePlan, message: string) {
    setOpen(undefined)
    setNotice(message)
    if (plans.state.name !== 'loaded') return
    const rest = plans.state.value.filter((p) => p.id !== plan.id)
    plans.set(rest.length === plans.state.value.length ? [plan, ...rest] : plans.state.value.map((p) => (p.id === plan.id ? plan : p)))
  }

  return (
    <section className="care-plans" aria-label={t('carePlans.title')}>
      <h2>{t('carePlans.title')}</h2>
      <Notice message={notice} />
      {plans.state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {plans.state.name === 'failed' && <Alert message={t('carePlans.loadFailed')} />}
      {plans.state.name === 'loaded' && plans.state.value.length === 0 && <p className="muted">{t('carePlans.empty')}</p>}
      {plans.state.name === 'loaded' && plans.state.value.length > 0 && (
        <ol className="rows" aria-label={t('carePlans.title')}>
          {plans.state.value.map((p) => (
            <Plan key={p.id} plan={p} services={priceList} zone={zone} onOpen={setOpen} />
          ))}
        </ol>
      )}
      {plans.state.name === 'loaded' && (
        <div className="actions">
          <button type="button" onClick={() => setOpen({ name: 'write' })}>
            {t('carePlans.new')}
          </button>
        </div>
      )}
      {open?.name === 'cancel' && <CancelPlan api={api} plan={open.plan} onCancelled={(p) => stored(p, t('carePlans.cancelled'))} onClose={() => setOpen(undefined)} />}
      {(open?.name === 'write' || open?.name === 'change') && (
        <CarePlanDialog
          api={api}
          patientId={patientId}
          appointmentId={appointmentId}
          plan={open.name === 'change' ? open.plan : undefined}
          services={services.state}
          zone={zone}
          onSaved={(p) => stored(p, t('carePlans.saved'))}
          onClose={() => setOpen(undefined)}
        />
      )}
      {open?.name === 'book' && (
        <CarePlanBook
          api={api}
          fhir={fhir}
          plan={open.plan}
          services={priceList}
          zone={zone}
          practitionerId={practitionerId}
          today={today}
          onBooked={(p) => stored(p, t('carePlans.booked'))}
          onClose={() => setOpen(undefined)}
        />
      )}
    </section>
  )
}

function Plan({ plan: p, services, zone, onOpen }: { plan: CarePlan; services: Service[]; zone: string; onOpen: (o: Open) => void }) {
  const t = useT()
  const f = useFormat()
  const ended = p.status === 'completed' || p.status === 'revoked'
  return (
    <li>
      <span className="stack">
        <span>
          <span className="strong">{p.title}</span> <span className={`tag ${p.status}`}>{t(`carePlans.status.${p.status}`)}</span>
        </span>
        {p.startsOn && <span>{t('carePlans.startsOn', { date: f.date(dayStart(p.startsOn, 0, zone), zone, 'long') })}</span>}
        <ol className="steps">
          {p.steps.map((s, i) => (
            <li key={s.id ?? i}>{stepText(t, f, s, i, services, p.startsOn, zone)}</li>
          ))}
        </ol>
      </span>
      <span className="actions">
        {p.status === 'draft' && (
          <>
            <button type="button" onClick={() => onOpen({ name: 'change', plan: p })}>
              {t('carePlans.change')}
            </button>
            <button type="button" onClick={() => onOpen({ name: 'book', plan: p })}>
              {t('carePlans.book')}
            </button>
          </>
        )}
        {!ended && (
          <button type="button" onClick={() => onOpen({ name: 'cancel', plan: p })}>
            {t('carePlans.cancel')}
          </button>
        )}
      </span>
    </li>
  )
}

function CancelPlan({ api, plan, onCancelled, onClose }: { api: ApiClient; plan: CarePlan; onCancelled: (p: CarePlan) => void; onClose: () => void }) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  async function cancel() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(
      api.POST('/clinic/care-plans/{id}/cancel', { params: { path: { id: plan.id }, header: { 'If-Match': `W/"${plan.versionId}"` } } }),
    )
    setBusy(false)
    if (res.ok) onCancelled(res.data)
    else setError(planMessage(t, res.code))
  }
  return (
    <>
      <Confirm
        question={t('carePlans.cancelQuestion')}
        lines={[t('carePlans.cancelEffect')]}
        confirm={t('carePlans.cancel')}
        cancel={t('carePlans.keep')}
        danger
        busy={busy}
        onConfirm={() => void cancel()}
        onCancel={onClose}
      />
      <Alert message={error} />
    </>
  )
}
