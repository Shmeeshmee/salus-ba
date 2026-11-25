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

// DataRequestDetail is one request targeting the clinic, for its admin (docs/clinic-cms/WAVE5.md D4): its patient's name
// and birth date, kind, state, due date, what it needs and its history; while open, its answer with a code and one
// extension with a reason; and, open or answered, the record's restriction at the clinic (S7) with the one action that
// applies, and the hand-over of the record of a patient the clinic registered (wave 5 walk, findings 3 to 5).
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
  return <Detail api={api} request={state.value} zone={zone.value} onChange={(r) => set({ ...state.value, ...r, patient: state.value.patient })} />
}

function Detail({ api, request: r, zone, onChange }: { api: ApiClient; request: DataRequest; zone: string; onChange: (r: DataRequest) => void }) {
  const t = useT()
  const f = useFormat()
  const day = (at: string) => f.date(new Date(at), zone, 'long')
  return (
    <section className="panel">
      <Link to="/data-requests">{t('privacy.back')}</Link>
      <h1>{t(`privacy.kind.${r.kind}`)}</h1>
      {r.patient && (
        <p className="stack">
          <strong>{`${r.patient.givenName} ${r.patient.familyName}`}</strong>
          <span className="muted">{t('privacy.born', { date: f.calendarDate(r.patient.birthDate) })}</span>
        </p>
      )}
      <p>
        <span className={`tag ${r.status}`}>{t(`privacy.status.${r.status}`)}</span>
      </p>
      <p>{t('privacy.due', { date: day(r.extendedUntil ?? r.dueAt) })}</p>
      <p className="muted">{t('privacy.received', { date: day(r.receivedAt) })}</p>
      {r.answerCode && <p>{t(`privacy.code.${r.answerCode}`)}</p>}
      {isOpen(r) && <p>{t(`privacy.needs.${r.kind}`)}</p>}
      {isOpen(r) && <Actions api={api} request={r} onChange={onChange} />}
      {r.patientId && <Restriction api={api} request={r} zone={zone} />}
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

type RestrictionState = components['schemas']['PatientRestriction']

// Restriction is the record's restriction at the clinic, open or answered (a restriction made for a request is lifted
// after it is answered): whether it is restricted, since when and by what, and only the action that applies: restrict,
// or lift the clinic's own; the erasure's is not the clinic's to lift (S6). Beside it, the hand-over of the record of a
// patient the clinic registered, which is how the admin reaches a restricted patient (S7: hidden from the desk's
// search); an app patient's record is handed over by their own export in the app.
function Restriction({ api, request: r, zone }: { api: ApiClient; request: DataRequest; zone: string }) {
  const t = useT()
  const f = useFormat()
  const path = { id: r.patientId ?? '' }
  const { state, set } = useLoad(async () => readRestriction(api, path), [api, r.patientId])
  const [error, setError] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const change = async (restricting: boolean) => {
    setError(undefined)
    setNotice(undefined)
    const res = restricting
      ? await attempt(api.PUT('/clinic/patients/{id}/restriction', { params: { path }, body: { dataRequestId: r.id } }))
      : await attempt(api.DELETE('/clinic/patients/{id}/restriction', { params: { path } }))
    if (res.ok) setNotice(t(restricting ? 'privacy.restricted' : 'privacy.lifted'))
    else setError(t(!restricting && res.code === 'forbidden' ? 'privacy.erasureKeeps' : 'privacy.actionFailed'))
    const again = await readRestriction(api, path).catch(() => undefined)
    if (again) set(again)
  }
  const since = (at?: string) => f.date(new Date(at ?? ''), zone, 'long')
  const now = state.name === 'loaded' ? state.value : undefined
  return (
    <>
      <h2>{t('privacy.recordTitle')}</h2>
      {state.name === 'failed' && <Alert message={t('privacy.restrictionLoadFailed')} />}
      {now && !now.restricted && <p>{t('privacy.notRestricted')}</p>}
      {now?.restricted && now.by === 'clinic' && <p>{t('privacy.restrictedByClinic', { date: since(now.restrictedAt) })}</p>}
      {now?.restricted && now.by === 'erasure' && <p>{t('privacy.restrictedByErasure', { date: since(now.restrictedAt) })}</p>}
      <div className="toolbar actions">
        {r.patient?.registeredByClinic && (
          <a href={handOverURL(r.patientId ?? '')} download={`patient-${r.patientId ?? ''}.json`}>
            {t('privacy.handOver')}
          </a>
        )}
        {now && !now.restricted && (
          <button type="button" onClick={() => void change(true)}>
            {t('privacy.restrict')}
          </button>
        )}
        {now?.restricted && now.by === 'clinic' && (
          <button type="button" className="secondary" onClick={() => void change(false)}>
            {t('privacy.lift')}
          </button>
        )}
      </div>
      {notice && <p role="status">{notice}</p>}
      <Alert message={error} />
    </>
  )
}

async function readRestriction(api: ApiClient, path: { id: string }): Promise<RestrictionState> {
  const res = await attempt(api.GET('/clinic/patients/{id}/restriction', { params: { path } }))
  if (!res.ok) throw new Error(res.code ?? 'restriction')
  return res.data
}

// handOverURL is where the patient's record at the clinic is handed over, on the same origin, so the session cookie
// goes with it.
function handOverURL(patientId: string): string {
  return `${globalThis.location.origin}/clinic/patients/${encodeURIComponent(patientId)}/record`
}
