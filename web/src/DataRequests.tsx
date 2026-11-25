import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { isOpen, type DataRequest } from './dataRequest'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; account: Account }

// DataRequests is the queue of data subject requests targeting the clinic, for its admin (docs/clinic-cms/WAVE5.md
// D4): as the API orders them (open ones by due date first), each with its patient's name and birth date (wave 5 walk,
// finding 3), kind, state, due date in the clinic's zone (the extended one once extended), what an open one needs, and
// its detail.
export function DataRequests({ api, fhir, account }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state: zone } = useLoad(() => loadTimeZone(fhir, organizationId), [fhir, organizationId])
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/data-requests'))
    if (!res.ok) throw new Error('data requests')
    return res.data
  }, [api])

  if (zone.name === 'failed' || state.name === 'failed') return <Alert message={t('privacy.loadFailed')} />
  if (zone.name === 'loading' || state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  return (
    <section className="panel">
      <h1>{t('privacy.nav')}</h1>
      {state.value.length === 0 && <p className="muted">{t('privacy.empty')}</p>}
      {state.value.length > 0 && (
        <ol className="rows" aria-label={t('privacy.nav')}>
          {state.value.map((r) => (
            <Row key={r.id} request={r} zone={zone.value} />
          ))}
        </ol>
      )}
    </section>
  )
}

function Row({ request: r, zone }: { request: DataRequest; zone: string }) {
  const t = useT()
  const f = useFormat()
  return (
    <li>
      <span className="stack">
        {r.patient && (
          <span>
            <strong>{`${r.patient.givenName} ${r.patient.familyName}`}</strong>{' '}
            <span className="muted small">{t('privacy.born', { date: f.calendarDate(r.patient.birthDate) })}</span>
          </span>
        )}
        <span>
          {t(`privacy.kind.${r.kind}`)} · {t('privacy.due', { date: f.date(new Date(r.extendedUntil ?? r.dueAt), zone, 'long') })}
        </span>
        {isOpen(r) && <span className="muted">{t(`privacy.needs.${r.kind}`)}</span>}
        <span className="actions">
          <span className={`tag ${r.status}`}>{t(`privacy.status.${r.status}`)}</span>
          {!r.filedByPatient && <span className="muted">{t('privacy.filedAtDesk')}</span>}
        </span>
      </span>
      <span className="actions">
        <Link to={`/data-requests/${encodeURIComponent(r.id)}`}>{t('privacy.open')}</Link>
      </span>
    </li>
  )
}
