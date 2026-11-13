import { useId, useRef, useState, type KeyboardEvent } from 'react'
import { useLanguage, useT } from '../i18n/useT'
import { useLoad, type Load } from '../load'
import { Alert } from '../ui'
import { ClinicView } from './ClinicView'
import type { Account } from '../api/client'
import { AnalyticsError, platformEnabled, type AnalyticsData, type PlatformMonths } from './data'
import { PlatformView, type PlatformReference } from './PlatformView'
import './analytics.css'

type Tab = 'clinic' | 'platform'

// Analytics is the staff's analytics page (SAL-22, plan §9 ANA-4): My clinic for every staff account, and All of
// Salus only when GET /auth/me has the analytics_platform feature (plan §2a) and the clinic contributes (O6: the API
// refuses the months with analytics_not_contributing, which hides the tab).
export function Analytics({ data, account, now }: { data: AnalyticsData; account: Account; now?: Date }) {
  const t = useT()
  const { language } = useLanguage()
  const id = useId()
  const admin = account.role === 'admin'
  const enabled = platformEnabled(account)
  const [tab, setTab] = useState<Tab>('clinic')
  const refs = useRef<Record<Tab, HTMLButtonElement | null>>({ clinic: null, platform: null })

  const reference = useLoad(async () => {
    const [groups, places, specialties, staff] = await Promise.all([data.groups(language), data.places(), data.specialties(), admin ? data.staff() : []])
    return { groups, places, specialties, staff }
  }, [data, language, admin])
  const platform = useLoad(() => (enabled ? data.platformMonths().catch(notContributing) : Promise.resolve(null)), [data, enabled])

  const p = platform.state
  const tabs: Tab[] = p.name === 'failed' || (p.name === 'loaded' && p.value !== null) ? ['clinic', 'platform'] : ['clinic']
  const shown = tabs.includes(tab) ? tab : 'clinic'

  function key(event: KeyboardEvent) {
    if (event.key !== 'ArrowRight' && event.key !== 'ArrowLeft') return
    event.preventDefault()
    const next = tabs[(tabs.indexOf(shown) + (event.key === 'ArrowRight' ? 1 : tabs.length - 1)) % tabs.length]
    setTab(next)
    refs.current[next]?.focus()
  }

  return (
    <section className="panel analytics">
      <h1>{t('analytics.title')}</h1>
      <div className="tabs analytics-tabs" role="tablist" aria-label={t('analytics.tabs')} onKeyDown={key}>
        {tabs.map((name) => (
          <button
            key={name}
            ref={(el) => {
              refs.current[name] = el
            }}
            type="button"
            role="tab"
            id={`${id}-${name}-tab`}
            aria-selected={shown === name}
            aria-controls={`${id}-${name}`}
            tabIndex={shown === name ? 0 : -1}
            onClick={() => setTab(name)}
          >
            {t(name === 'clinic' ? 'analytics.myClinic' : 'analytics.allOfSalus')}
          </button>
        ))}
      </div>
      <div role="tabpanel" id={`${id}-${shown}`} aria-labelledby={`${id}-${shown}-tab`}>
        {reference.state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {reference.state.name === 'failed' && <Alert message={t('analytics.loadFailed')} />}
        {reference.state.name === 'loaded' && shown === 'clinic' && <ClinicView data={data} reference={reference.state.value} admin={admin} now={now} />}
        {reference.state.name === 'loaded' && shown === 'platform' && <Platform data={data} state={platform.state} reference={reference.state.value} />}
      </div>
    </section>
  )
}

// notContributing is null for a clinic that does not contribute (no tab), the error otherwise.
function notContributing(error: unknown): null {
  if (error instanceof AnalyticsError && error.code === 'analytics_not_contributing') return null
  throw error
}

function Platform({ data, state, reference }: { data: AnalyticsData; state: Load<PlatformMonths | null>; reference: PlatformReference }) {
  const t = useT()
  if (state.name === 'failed') return <Alert message={t('analytics.loadFailed')} />
  if (state.name !== 'loaded' || state.value === null) return null
  if (state.value.months.length === 0) return <p className="muted">{t('analytics.noMonths')}</p>
  return <PlatformView data={data} reference={reference} months={state.value.months} quarters={state.value.quarters} />
}
