import { Fragment, useId, useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import type { Appointment, Bundle, FhirClient, Questionnaire, QuestionnaireResponse } from './api/fhir'
import { attempt } from './api/result'
import { filesOf } from './attachment'
import { Files } from './Files'
import { useLoad } from './load'
import { files as filesText, questionnaires as t, somethingWrong } from './messages'
import { fromFHIR, type Version } from './questionnaire'
import { requestOfVersion, requestsOf, statusOf, valueText, type Block, type Question, type Request } from './responses'
import { Alert, Confirm, TipTag } from './ui'

type Props = { api: ApiClient; fhir: FhirClient; appt: Appointment; zone: string; own: boolean; admin: boolean; now: Date }

// AppointmentQuestionnaires is the appointment's questionnaire requests and their status (W32 to W40). The
// appointment's practitioner (own) adds one while the appointment is booked and ahead, withdraws one not yet
// submitted, and reads the answers of a submitted one; an admin sees the requests and their status only.
export function AppointmentQuestionnaires({ api, fhir, appt, zone, own, admin, now }: Props) {
  const { state, set } = useLoad(async () => {
    const query = new URLSearchParams({ appointment: `Appointment/${appt.id}`, _include: 'QuestionnaireResponse:questionnaire' })
    return requestsOf(await fhir.get<Bundle<QuestionnaireResponse | Questionnaire>>(`QuestionnaireResponse?${query}`))
  }, [fhir, appt.id])
  const start = new Date(appt.start)
  const canAdd = own && appt.status === 'booked' && now < start

  return (
    <section className="requests" aria-label={t.requests}>
      <h2>{t.requests}</h2>
      {state.name === 'loading' && <p className="muted">Loading</p>}
      {state.name === 'failed' && <Alert message={somethingWrong} />}
      {state.name === 'loaded' && (
        <>
          {state.value.length === 0 && <p className="muted">{t.noRequests}</p>}
          {state.value.length > 0 && (
            <ul>
              {state.value.map((r) => (
                <RequestRow
                  key={r.id}
                  api={api}
                  request={r}
                  status={statusOf(r, start, now, zone)}
                  own={own}
                  admin={admin}
                  onChanged={(next) => set(state.value.map((x) => (x.id === next.id ? next : x)))}
                />
              ))}
            </ul>
          )}
          {canAdd && <Add api={api} fhir={fhir} appointmentId={appt.id} requested={state.value} onAdded={(r) => set([...state.value, r])} />}
        </>
      )}
    </section>
  )
}

type RowProps = { api: ApiClient; request: Request; status: string; own: boolean; admin: boolean; onChanged: (r: Request) => void }

function RequestRow({ api, request, status, own, admin, onChanged }: RowProps) {
  const [open, setOpen] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [refocus, setRefocus] = useState(false) // Withdraw takes the focus back when it returns
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function withdraw() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/questionnaire-responses/{id}/withdraw', { params: { path: { id: request.id } } }))
    setBusy(false)
    setConfirming(false)
    // Already withdrawn (by another tab) is what was asked for.
    if (res.ok || res.code === 'withdrawn') onChanged({ ...request, status: 'stopped', answers: [] })
    else {
      setError(t.requestErrors[res.code ?? ''] ?? somethingWrong)
      setRefocus(true)
    }
  }

  return (
    <li>
      <div className="request">
        <span className="request-title">{request.title}</span>
        {admin ? <TipTag className={request.status} label={status} tip={t.answersPrivate} /> : <span className={`tag ${request.status}`}>{status}</span>}
        {own && request.status === 'completed' && (
          <button type="button" className="quiet" aria-expanded={open} onClick={() => setOpen((o) => !o)}>
            {open ? t.hideAnswers : t.viewAnswers}
          </button>
        )}
        {own && request.status === 'in-progress' && !confirming && (
          // autoFocus only once the confirm closed without a withdrawal: the button was gone meanwhile, and gets
          // the focus back.
          <button type="button" className="quiet danger-text" autoFocus={refocus} onClick={() => setConfirming(true)}>
            {t.withdraw}
          </button>
        )}
      </div>
      <Alert message={error} />
      {confirming && (
        <Confirm
          question={t.withdrawQuestion}
          lines={[t.withdrawWarning]}
          confirm={t.withdraw}
          cancel={t.cancel}
          danger
          busy={busy}
          onConfirm={withdraw}
          onCancel={() => {
            setConfirming(false)
            setRefocus(true)
          }}
        />
      )}
      {open && <Answers api={api} blocks={request.answers} />}
    </li>
  )
}

function Answers({ api, blocks }: { api: ApiClient; blocks: Block[] }) {
  return (
    <div className="answers">
      {blocks.map((b) => (
        <Fragment key={b.questions[0].linkId}>
          {b.section !== undefined && <h3>{b.section}</h3>}
          <dl>
            {b.questions.map((q) => (
              <div key={q.linkId}>
                <dt>{q.text}</dt>
                <dd>{answerOf(api, q)}</dd>
              </div>
            ))}
          </dl>
        </Fragment>
      ))}
    </div>
  )
}

function answerOf(api: ApiClient, q: Question) {
  if (q.values.length === 0) return <span className="muted">{t.notAnswered}</span>
  if (q.kind === 'attachment') {
    const files = filesOf(q.values)
    return (
      <>
        {filesText.count(files.length)}
        <Files api={api} files={files} />
      </>
    )
  }
  if (q.kind !== 'multiple') return valueText(q.values[0])
  return (
    <ul>
      {q.values.map((v) => (
        <li key={valueText(v)}>{valueText(v)}</li>
      ))}
    </ul>
  )
}

type AddProps = { api: ApiClient; fhir: FhirClient; appointmentId: string; requested: Request[]; onAdded: (r: Request) => void }

function Add(props: AddProps) {
  const [open, setOpen] = useState(false)
  if (!open) {
    return (
      <div className="actions">
        <button type="button" onClick={() => setOpen(true)}>
          {t.add}
        </button>
      </div>
    )
  }
  return (
    <Picker
      {...props}
      onAdded={(r) => {
        setOpen(false)
        props.onAdded(r)
      }}
      onClose={() => setOpen(false)}
    />
  )
}

// Picker offers the clinic's active questionnaires that the appointment has no live request of (a withdrawn
// one may be requested again).
function Picker({ api, fhir, appointmentId, requested, onAdded, onClose }: AddProps & { onClose: () => void }) {
  const id = useId()
  const { state } = useLoad(async () => {
    const bundle = await fhir.get<Bundle<Questionnaire>>(`Questionnaire?${new URLSearchParams({ status: 'active' })}`)
    return (bundle.entry ?? []).map((e) => fromFHIR(e.resource)).sort((a, b) => a.title.localeCompare(b.title))
  }, [fhir])
  const [chosen, setChosen] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const live = new Set(requested.filter((r) => r.status !== 'stopped').map((r) => r.url))
  const options: Version[] = state.name === 'loaded' ? state.value.filter((v) => !live.has(v.url)) : []
  const picked = options.find((v) => v.id === chosen) ?? options[0]

  async function add(event: FormEvent) {
    event.preventDefault()
    if (!picked) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/appointments/{id}/questionnaires', { params: { path: { id: appointmentId } }, body: { questionnaireId: picked.id } }))
    setBusy(false)
    if (res.ok) onAdded(requestOfVersion(res.data.id, picked))
    else setError(t.requestErrors[res.code ?? ''] ?? somethingWrong)
  }

  return (
    <form className="confirm" onSubmit={add} noValidate>
      <div className="field">
        <label htmlFor={id}>{t.picker}</label>
        <select id={id} value={picked?.id ?? ''} aria-describedby={`${id}-hint`} onChange={(e) => setChosen(e.target.value)}>
          {options.map((v) => (
            <option key={v.id} value={v.id}>
              {v.title}
            </option>
          ))}
        </select>
        <p className="hint" id={`${id}-hint`}>
          {t.addHint}
        </p>
      </div>
      <Alert message={state.name === 'failed' ? somethingWrong : error} />
      <div className="actions">
        <button type="submit" className="primary" disabled={busy || !picked}>
          {t.addConfirm}
        </button>
        <button type="button" disabled={busy} onClick={onClose}>
          {t.cancel}
        </button>
      </div>
    </form>
  )
}
