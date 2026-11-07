import { useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { Bundle, FhirClient, Questionnaire } from './api/fhir'
import { attempt } from './api/result'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { libraryOf, type Status, type Version } from './questionnaire'
import { navigate } from './route'
import { loadTimeZone } from './staff'
import { Alert, Link, TipTag } from './ui'

type Filter = 'all' | Status
const filters: Filter[] = ['all', 'draft', 'active', 'retired']

// Questionnaires is the clinic's library (W1 to W8): each questionnaire's draft and active version, or its
// last retired one, filtered by status. "New questionnaire" creates an untitled draft and opens the builder.
export function Questionnaires({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state } = useLoad(async () => {
    const [zone, bundle] = await Promise.all([loadTimeZone(fhir, organizationId), fhir.get<Bundle<Questionnaire>>('Questionnaire')])
    return { zone, versions: libraryOf((bundle.entry ?? []).map((e) => e.resource)) }
  }, [fhir, organizationId])
  const [filter, setFilter] = useState<Filter>('all')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function create() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/questionnaires', { body: { title: t('questionnaire.untitled'), item: [] } }))
    setBusy(false)
    if (res.ok) navigate(`/questionnaires/${encodeURIComponent(res.data.id)}`)
    else setError(t('common.somethingWrong'))
  }

  return (
    <section className="panel">
      <div className="heading">
        <h1>{t('questionnaire.title')}</h1>
        <button type="button" className="primary" disabled={busy} onClick={create}>
          {t('questionnaire.create')}
        </button>
      </div>
      <Alert message={error} />
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('common.somethingWrong')} />}
      {state.name === 'loaded' && state.value.versions.length === 0 && <p className="muted">{t('questionnaire.emptyLibrary')}</p>}
      {state.name === 'loaded' && state.value.versions.length > 0 && (
        <>
          <div className="segmented" role="group">
            {filters.map((f) => (
              <button key={f} type="button" aria-pressed={filter === f} onClick={() => setFilter(f)}>
                {t(`questionnaire.filters.${f}`)}
              </button>
            ))}
          </div>
          <Library versions={state.value.versions.filter((v) => filter === 'all' || v.status === filter)} zone={state.value.zone} />
        </>
      )}
    </section>
  )
}

function Library({ versions, zone }: { versions: Version[]; zone: string }) {
  const t = useT()
  const format = useFormat()
  if (versions.length === 0) return <p className="muted">{t('questionnaire.emptyFilter')}</p>
  return (
    <ul className="rows" aria-label={t('questionnaire.title')}>
      {versions.map((v) => (
        <li key={v.id}>
          <Link to={`/questionnaires/${encodeURIComponent(v.id)}`}>{v.title}</Link>
          <TipTag className={v.status} label={t(`questionnaire.statuses.${v.status}`)} tip={t('questionnaire.versionTip', { n: v.version, date: format.date(new Date(v.lastUpdated), zone, 'long') })} />
        </li>
      ))}
    </ul>
  )
}
