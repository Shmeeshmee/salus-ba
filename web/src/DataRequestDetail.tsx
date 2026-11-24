import { useId, useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import type { components } from './api/schema'
import { attempt } from './api/result'
import { isOpen, type DataRequest } from './dataRequest'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadTimeZone } from './staff'
import { Alert, Link } from './ui'

type AnswerCode = components['schemas']['DataRequestAnswerCode']
type Reason = components['schemas']['DataRequestExtensionReason']

// The codes that fulfil a request; the others refuse it (the DataRequestAnswerCode schema).
const fulfilling: AnswerCode[] = ['done', 'done_partly_kept']
const codes: AnswerCode[] = [...fulfilling, 'identity_unverified', 'manifestly_unfounded', 'excessive', 'retention_law', 'no_data']
const reasons: Reason[] = ['complex', 'many_requests']
// The answer's and the extension's refusals the admin can act on; any other is privacy.actionFailed.
const failures: Record<string, string> = { version_conflict: 'privacy.conflict', request_overdue: 'privacy.overdue' }

type Props = { api: ApiClient; fhir: FhirClient; account: Account; id: string }

// DataRequestDetail is one request targeting the clinic, for its admin (docs/clinic-cms/WAVE5.md D4): its kind, state,
// due date, what it needs and its history, and, while open, its actions: answer with a code, extend once with a reason,
// open the hand-over of the patient's record, restrict the record at the clinic or lift the restriction (S7).
export function DataRequestDetail({ api, fhir, account, id }: Props) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const { state: zone } = useLoad(() => loadTimeZone(fhir, organizationId), [fhir, organizationId])
  const { state, set } = useLoad(async () => {
    const res = await attempt(api.GET('/clinic/data-requests/{id}', { params: { path: { id } } }))
    if (!res.ok) throw new Error(res.code ?? 'data request')
    return res.data
  }, [api, id])

  if (state.name === 'failed') {
    const missing = state.error instanceof Error && state.error.message === 'not_found'
    return <Alert message={t(missing ? 'privacy.notFound' : 'privacy.loadFailed')} />
  }
  if (zone.name === 'failed') return <Alert message={t('privacy.loadFailed')} />
  if (zone.name === 'loading' || state.name === 'loading') return <p className="panel muted">{t('common.loading')}</p>
  return <Detail api={api} request={state.value} zone={zone.value} onChange={(r) => set({ ...state.value, ...r })} />
}

function Detail({ api, request: r, zone, onChange }: { api: ApiClient; request: DataRequest; zone: string; onChange: (r: DataRequest) => void }) {
  const t = useT()
  const f = useFormat()
  const day = (at: string) => f.date(new Date(at), zone, 'long')
  return (
    <section className="panel">
      <Link to="/data-requests">{t('privacy.back')}</Link>
      <h1>{t(`privacy.kind.${r.kind}`)}</h1>
      <p>
        <span className={`tag ${r.status}`}>{t(`privacy.status.${r.status}`)}</span>
      </p>
      <p>{t('privacy.due', { date: day(r.extendedUntil ?? r.dueAt) })}</p>
      <p className="muted">{t('privacy.received', { date: day(r.receivedAt) })}</p>
      {r.answerCode && <p>{t(`privacy.code.${r.answerCode}`)}</p>}
      {isOpen(r) && <p>{t(`privacy.needs.${r.kind}`)}</p>}
      {isOpen(r) && <Actions api={api} request={r} onChange={onChange} />}
      {r.patientId && <Restriction api={api} request={r} />}
      {r.history && r.history.length > 0 && (
        <>
          <h2>{t('privacy.history')}</h2>
          <ol className="rows" aria-label={t('privacy.history')}>
            {r.history.map((h) => (
              <li key={`${h.status}-${h.changedAt}`}>
                {t(`privacy.status.${h.status}`)} · {day(h.changedAt)}
              </li>
            ))}
          </ol>
        </>
      )}
    </section>
  )
}

// Actions are an open request's: the answer and the one extension, at the version read.
function Actions({ api, request: r, onChange }: { api: ApiClient; request: DataRequest; onChange: (r: DataRequest) => void }) {
  const t = useT()
  const ids = { code: useId(), reason: useId() }
  const [code, setCode] = useState<AnswerCode>('done')
  const [reason, setReason] = useState<Reason>('complex')
  const [error, setError] = useState<string>()
  const ifMatch = { 'If-Match': `W/"${r.version}"` }

  const save = async (request: Promise<{ data?: DataRequest; error?: unknown; response: Response }>) => {
    setError(undefined)
    const res = await attempt(request)
    if (res.ok) onChange(res.data)
    else setError(t(failures[res.code ?? ''] ?? 'privacy.actionFailed'))
  }
  return (
    <>
      <div className="toolbar actions">
        <label htmlFor={ids.code}>{t('privacy.answer')}</label>
        <select id={ids.code} value={code} onChange={(e) => setCode(e.target.value as AnswerCode)}>
          {codes.map((c) => (
            <option key={c} value={c}>
              {t(`privacy.code.${c}`)}
            </option>
          ))}
        </select>
        <button type="button" className="primary" onClick={() => void save(api.POST('/clinic/data-requests/{id}/answer', {
          params: { path: { id: r.id }, header: ifMatch }, body: { status: fulfilling.includes(code) ? 'fulfilled' : 'refused', code },
        }))}>
          {t('privacy.sendAnswer')}
        </button>
      </div>
      {!r.extendedUntil && (
        <div className="toolbar actions">
          <label htmlFor={ids.reason}>{t('privacy.reason')}</label>
          <select id={ids.reason} value={reason} onChange={(e) => setReason(e.target.value as Reason)}>
            {reasons.map((x) => (
              <option key={x} value={x}>
                {t(`privacy.reasons.${x}`)}
              </option>
            ))}
          </select>
          <button type="button" onClick={() => void save(api.POST('/clinic/data-requests/{id}/extend', {
            params: { path: { id: r.id }, header: ifMatch }, body: { reason },
          }))}>
            {t('privacy.extend')}
          </button>
        </div>
      )}
      <Alert message={error} />
    </>
  )
}

// Restriction is the hand-over of the patient's record and its restriction at the clinic, open or answered: a
// restriction made for a request is lifted after it is answered. The erasure's restriction is not the clinic's to lift.
function Restriction({ api, request: r }: { api: ApiClient; request: DataRequest }) {
  const t = useT()
  const [error, setError] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const restrict = async (restricting: boolean) => {
    setError(undefined)
    setNotice(undefined)
    const path = { id: r.patientId ?? '' }
    const res = restricting
      ? await attempt(api.PUT('/clinic/patients/{id}/restriction', { params: { path }, body: { dataRequestId: r.id } }))
      : await attempt(api.DELETE('/clinic/patients/{id}/restriction', { params: { path } }))
    if (res.ok) setNotice(t(restricting ? 'privacy.restricted' : 'privacy.lifted'))
    else setError(t(!restricting && res.code === 'forbidden' ? 'privacy.erasureKeeps' : 'privacy.actionFailed'))
  }
  return (
    <>
      <div className="toolbar actions">
        <a href={handOverURL(r.patientId ?? '')} target="_blank" rel="noreferrer">
          {t('privacy.handOver')}
        </a>
        <button type="button" onClick={() => void restrict(true)}>
          {t('privacy.restrict')}
        </button>
        <button type="button" className="secondary" onClick={() => void restrict(false)}>
          {t('privacy.lift')}
        </button>
      </div>
      {notice && <p role="status">{notice}</p>}
      <Alert message={error} />
    </>
  )
}

// handOverURL is where the patient's record at the clinic is handed over, on the same origin, so the session cookie
// goes with it.
function handOverURL(patientId: string): string {
  return `${globalThis.location.origin}/clinic/patients/${encodeURIComponent(patientId)}/record`
}
