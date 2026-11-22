import { useState } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { formatNumber } from './i18n/format'
import { useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { contentLanguageNames, contentLanguages } from './questionnaire'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Service = components['schemas']['Service']
type SaveRequest = components['schemas']['SaveServiceRequest']
type VisitMode = components['schemas']['VisitMode']
type T = ReturnType<typeof useT>

const visitModes: VisitMode[] = ['in-person', 'video']
const modeLabel = { 'in-person': 'prices.modeInPerson', video: 'prices.modeVideo' } as const

// PriceList is the admin's screen of the clinic's price list (docs/plans/clinic-cms.md, package B): its services with
// price, duration, VAT rate and the visit modes they are offered in, and their names for patients in other languages.
// Patients see the prices at booking and pay at the clinic; nothing is charged in the app. A service is never deleted:
// it is taken off the list, and a receipt already issued keeps the line it copied. One service is edited at a time,
// saved at the version it was read at (If-Match).
export function PriceList({ api }: { api: ApiClient }) {
  const t = useT()
  const { language } = useLanguage()
  const [editing, setEditing] = useState<string>() // a service's id, or 'new'
  const [saved, setSaved] = useState(false)
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/services'))
    if (!res.ok) throw new Error('services')
    return res.data
  }, [api])
  const services = state.name === 'loaded' ? state.value : []
  const open = (id: string | undefined) => {
    setEditing(id)
    setSaved(false)
  }
  const stored = (s: Service) => {
    const rest = services.filter((x) => x.id !== s.id)
    set([...rest, s].sort((a, b) => a.name.localeCompare(b.name, language)))
    setEditing(undefined)
    setSaved(true)
  }
  return (
    <section className="panel price-list">
      <h1>{t('prices.title')}</h1>
      <p className="hint">{t('prices.intro')}</p>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('prices.loadFailed')} />}
      {state.name === 'loaded' && services.length === 0 && editing !== 'new' && <p className="muted">{t('prices.empty')}</p>}
      {saved && <Notice message={t('prices.saved')} />}
      <ul className="rows">
        {services.map((s) =>
          editing === s.id ? (
            <li key={s.id}>
              <ServiceForm api={api} service={s} onStored={stored} onCancel={() => open(undefined)} />
            </li>
          ) : (
            <li key={s.id}>
              <div className="stack">
                <span className="strong">{s.name}</span>
                <span className="muted small">{details(t, language, s)}</span>
              </div>
              <div className="actions">
                {!s.active && <span className="tag">{t('prices.inactive')}</span>}
                <button type="button" aria-label={t('prices.editNamed', { name: s.name })} onClick={() => open(s.id)}>
                  {t('prices.edit')}
                </button>
              </div>
            </li>
          ),
        )}
      </ul>
      {state.name === 'loaded' && editing === 'new' && <ServiceForm api={api} onStored={stored} onCancel={() => open(undefined)} />}
      {state.name === 'loaded' && editing === undefined && (
        <div className="actions">
          <button type="button" onClick={() => open('new')}>
            {t('prices.add')}
          </button>
        </div>
      )}
    </section>
  )
}

// details is a service's line under its name: the price, the duration when it has its own, the VAT and the modes.
function details(t: T, language: ReturnType<typeof useLanguage>['language'], s: Service): string {
  const parts = [formatNumber(s.priceMinor / 100, language, { style: 'currency', currency: s.currency })]
  if (s.durationMinutes !== undefined) parts.push(t('prices.minutes', { count: s.durationMinutes }))
  parts.push(
    s.vatRateBp === undefined
      ? t('prices.exempt')
      : t('prices.vatShown', { rate: formatNumber(s.vatRateBp / 10000, language, { style: 'percent', maximumFractionDigits: 2 }) }),
  )
  for (const m of s.modes) parts.push(t(modeLabel[m]))
  return parts.join(' · ')
}

// amount reads a decimal with up to `whole` digits and two decimals (a comma or a point) as hundredths, or undefined.
function amount(text: string, whole: number): number | undefined {
  const m = new RegExp(`^(\\d{1,${whole}})(?:[.,](\\d{1,2}))?$`).exec(text.trim())
  return m ? Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0')) : undefined
}

type FormProps = { api: ApiClient; service?: Service; onStored: (s: Service) => void; onCancel: () => void }

function ServiceForm({ api, service, onStored, onCancel }: FormProps) {
  const t = useT()
  const [name, setName] = useState(service?.name ?? '')
  const [code, setCode] = useState(service?.code ?? '')
  const [price, setPrice] = useState(service ? (service.priceMinor / 100).toFixed(2) : '')
  const [duration, setDuration] = useState(service?.durationMinutes?.toString() ?? '')
  const [vat, setVat] = useState(service?.vatRateBp === undefined ? '' : String(service.vatRateBp / 100))
  const [modes, setModes] = useState<VisitMode[]>(service?.modes ?? visitModes)
  const [active, setActive] = useState(service?.active ?? true)
  const [names, setNames] = useState<Record<string, string>>(Object.fromEntries((service?.translations ?? []).map((x) => [x.language, x.name])))
  const currency = service?.currency ?? 'BAM'
  const { busy, error, onSubmit } = useSubmit(async () => {
    const priceMinor = amount(price, 7)
    const minutes = duration.trim() === '' ? undefined : Number(duration)
    const vatRateBp = vat.trim() === '' ? undefined : amount(vat, 3)
    if (name.trim() === '') return t('prices.invalidName')
    if (priceMinor === undefined) return t('prices.invalidPrice')
    if (minutes !== undefined && (!Number.isInteger(minutes) || minutes < 5 || minutes > 480)) return t('prices.invalidDuration')
    if (vat.trim() !== '' && (vatRateBp === undefined || vatRateBp > 10000)) return t('prices.invalidVat')
    if (modes.length === 0) return t('prices.invalidModes')
    const body: SaveRequest = {
      name: name.trim(),
      ...(code.trim() === '' ? {} : { code: code.trim() }),
      priceMinor,
      currency,
      ...(minutes === undefined ? {} : { durationMinutes: minutes }),
      ...(vatRateBp === undefined ? {} : { vatRateBp }),
      modes: visitModes.filter((m) => modes.includes(m)),
      active,
      translations: contentLanguages.filter((l) => (names[l] ?? '').trim() !== '').map((l) => ({ language: l, name: names[l].trim() })),
    }
    const res = service
      ? await attempt(api.PUT('/clinic/services/{id}', { params: { path: { id: service.id }, header: { 'If-Match': `W/"${service.versionId}"` } }, body }))
      : await attempt(api.POST('/clinic/services', { body }))
    if (!res.ok) return saveMessage(t, res.response?.status, res.code)
    onStored(res.data)
    return undefined
  })
  return (
    <form onSubmit={onSubmit} noValidate aria-label={service ? t('prices.editNamed', { name: service.name }) : t('prices.add')}>
      <Field label={t('prices.name')} hint={t('prices.nameHint')} maxLength={200} value={name} onChange={(e) => setName(e.target.value)} />
      <div className="pair">
        <Field label={t('prices.price', { currency })} inputMode="decimal" value={price} onChange={(e) => setPrice(e.target.value)} />
        <Field label={t('prices.code')} maxLength={40} value={code} onChange={(e) => setCode(e.target.value)} />
      </div>
      <div className="pair">
        <Field label={t('prices.duration')} hint={t('prices.durationHint')} inputMode="numeric" value={duration} onChange={(e) => setDuration(e.target.value)} />
        <Field label={t('prices.vat')} hint={t('prices.vatHint')} inputMode="decimal" value={vat} onChange={(e) => setVat(e.target.value)} />
      </div>
      <fieldset className="choice">
        <legend>{t('prices.modes')}</legend>
        {visitModes.map((m) => (
          <label className="check" key={m}>
            <input
              type="checkbox"
              checked={modes.includes(m)}
              onChange={(e) => setModes(e.target.checked ? [...modes, m] : modes.filter((x) => x !== m))}
            />
            <span>{t(modeLabel[m])}</span>
          </label>
        ))}
      </fieldset>
      <label className="check">
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
        <span>{t('prices.active')}</span>
      </label>
      <p className="hint">{t('prices.activeHint')}</p>
      <details>
        <summary>{t('prices.translations')}</summary>
        <p className="hint">{t('prices.translationsHint')}</p>
        {contentLanguages.map((l) => (
          <Field
            key={l}
            label={contentLanguageNames[l]}
            lang={l}
            maxLength={200}
            value={names[l] ?? ''}
            onChange={(e) => setNames({ ...names, [l]: e.target.value })}
          />
        ))}
      </details>
      <Alert message={error} />
      <div className="actions">
        <Submit busy={busy} label={t('prices.save')} busyLabel={t('common.saving')} />
        <button type="button" onClick={onCancel} disabled={busy}>
          {t('prices.cancel')}
        </button>
      </div>
    </form>
  )
}

function saveMessage(t: T, status: number | undefined, code: ErrorCode | undefined): string {
  if (status === 412) return t('prices.conflict')
  if (code === 'forbidden') return t('prices.forbidden')
  if (code === 'invalid_service' || code === 'invalid_request') return t('prices.invalidService')
  return t('prices.saveFailed')
}
