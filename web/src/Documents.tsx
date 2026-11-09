import { useEffect, useId, useState, type Dispatch } from 'react'
import type { ConsultActions, ConsultMessage, DraftEvent, DraftField, Template } from './consult'
import { useT } from './i18n/useT'
import { Alert, Confirm } from './ui'

type Props = {
  appointmentId: string
  drafts: DraftEvent[]
  dispatch: Dispatch<ConsultMessage>
  actions: ConsultActions
  onSources: (ids: string[]) => void
}

type Templates = { state: 'loading' } | { state: 'failed' } | { state: 'ready'; list: Template[] }

// Asking is the confirmation on screen: before a sign, or before a regenerate that would replace edits.
type Asking = 'sign' | 'regenerate'

// Documents drafts a clinic document during the call: pick a template, generate it from the transcript, read the
// filled fields with their sources and the flags on what the conversation has not covered, edit, and sign. The
// draft arrives on the live source (and again on every change); the edits stay on this page until it is signed.
export function Documents({ appointmentId, drafts, dispatch, actions, onSources }: Props) {
  const t = useT()
  const id = useId()
  const [templates, setTemplates] = useState<Templates>({ state: 'loading' })
  const [chosen, setChosen] = useState('')
  // waiting is the template being generated and its draft before, until a newer one arrives.
  const [waiting, setWaiting] = useState<{ template: string; before?: DraftEvent }>()
  const [edits, setEdits] = useState<Record<string, string>>({})
  const [asking, setAsking] = useState<Asking>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  useEffect(() => {
    let open = true
    actions.templates().then(
      (list) => open && setTemplates({ state: 'ready', list }),
      () => open && setTemplates({ state: 'failed' }),
    )
    return () => {
      open = false
    }
  }, [actions])

  if (templates.state === 'loading') return <p className="muted">{t('consultation.live.loadingTemplates')}</p>
  if (templates.state === 'failed') return <p className="muted">{t('consultation.live.templatesFailed')}</p>
  if (templates.list.length === 0) return <p className="muted">{t('consultation.live.noTemplates')}</p>

  const template = templates.list.find((x) => x.id === chosen)
  const current = latest(drafts, chosen)
  const generating = waiting?.template === chosen && waiting.before === current
  const draft = generating ? undefined : current

  async function generate() {
    setAsking(undefined)
    setError(undefined)
    setEdits({})
    setWaiting({ template: chosen, before: current })
    try {
      await actions.generate(appointmentId, chosen)
    } catch {
      setWaiting(undefined)
      setError(t('consultation.live.generateFailed'))
    }
  }

  async function sign(d: DraftEvent) {
    setBusy(true)
    setError(undefined)
    try {
      const signed = await actions.sign(
        d.draft_id,
        d.fields.map((f) => ({ key: f.key, value: edits[f.key] ?? f.value })),
      )
      dispatch({ type: 'draft', version: 1, payload: signed })
      setEdits({})
    } catch {
      setError(t('consultation.live.signFailed'))
    }
    setBusy(false)
    setAsking(undefined)
  }

  const labels = new Map(template?.fields.map((f) => [f.key, f.label]))

  return (
    <div className="documents">
      <div className="documents-pick">
        <label htmlFor={`${id}-template`}>{t('consultation.live.template')}</label>
        <select id={`${id}-template`} value={chosen} onChange={(e) => setChosen(e.target.value)}>
          <option value="">{t('consultation.live.chooseTemplate')}</option>
          {templates.list.map((x) => (
            <option key={x.id} value={x.id}>
              {x.title}
            </option>
          ))}
        </select>
        {!current && !generating && (
          <button type="button" className="primary" disabled={!chosen} onClick={() => void generate()}>
            {t('consultation.live.generate')}
          </button>
        )}
      </div>
      <Alert message={error} />
      {generating && (
        <p className="muted" role="status">
          {t('consultation.live.generating')}
        </p>
      )}
      {draft?.status === 'final' && <Signed draft={draft} labels={labels} />}
      {draft && draft.status !== 'final' && (
        <>
          <p className="note">{t('consultation.live.draftNote')}</p>
          {draft.fields.map((f) => (
            <FieldEdit
              key={f.key}
              field={f}
              label={labels.get(f.key) ?? f.key}
              edited={edits[f.key]}
              onEdit={(value) => setEdits((e) => ({ ...e, [f.key]: value }))}
              onSources={onSources}
            />
          ))}
          <DraftActions
            asking={asking}
            busy={busy}
            empty={draft.fields.filter((f) => (edits[f.key] ?? f.value).trim() === '').length}
            onAsk={(a) => (a === 'regenerate' && Object.keys(edits).length === 0 ? void generate() : setAsking(a))}
            onSign={() => void sign(draft)}
            onRegenerate={() => void generate()}
          />
        </>
      )}
    </div>
  )
}

// latest is the template's last draft not entered in error (a loop, as Array.prototype.findLast is missing in
// some browsers the app supports).
function latest(drafts: DraftEvent[], template: string): DraftEvent | undefined {
  for (let i = drafts.length - 1; i >= 0; i--) {
    if (drafts[i].template_id === template && drafts[i].status !== 'entered-in-error') return drafts[i]
  }
  return undefined
}

type ActionsProps = {
  asking?: Asking
  busy: boolean
  empty: number
  onAsk: (a: Asking | undefined) => void
  onSign: () => void
  onRegenerate: () => void
}

// DraftActions are Sign and Regenerate, each confirmed in place: a sign always, a regenerate only when it
// would replace the practitioner's edits (the caller decides).
function DraftActions({ asking, busy, empty, onAsk, onSign, onRegenerate }: ActionsProps) {
  const t = useT()
  if (asking === 'sign') {
    return (
      <Confirm
        question={t('consultation.live.signQuestion')}
        lines={[t('consultation.live.signFinal'), ...(empty > 0 ? [t('consultation.live.signEmpty', { count: empty })] : [])]}
        confirm={t('consultation.live.sign')}
        cancel={t('consultation.live.keepEditing')}
        busy={busy}
        onConfirm={onSign}
        onCancel={() => onAsk(undefined)}
      />
    )
  }
  if (asking === 'regenerate') {
    return (
      <Confirm
        question={t('consultation.live.regenerateQuestion')}
        lines={[t('consultation.live.regenerateReplaces')]}
        confirm={t('consultation.live.regenerate')}
        cancel={t('consultation.live.keepEditing')}
        onConfirm={onRegenerate}
        onCancel={() => onAsk(undefined)}
      />
    )
  }
  return (
    <div className="actions">
      <button type="button" className="primary" onClick={() => onAsk('sign')}>
        {t('consultation.live.sign')}
      </button>
      <button type="button" onClick={() => onAsk('regenerate')}>
        {t('consultation.live.regenerate')}
      </button>
    </div>
  )
}

type FieldProps = { field: DraftField; label: string; edited?: string; onEdit: (value: string) => void; onSources: (ids: string[]) => void }

// FieldEdit is one field of the draft. Once the practitioner types in it, its sources and its flag no longer
// describe it, so both go.
function FieldEdit({ field, label, edited, onEdit, onSources }: FieldProps) {
  const t = useT()
  const id = useId()
  const value = edited ?? field.value
  const sources = edited === undefined ? (field.source_segment_ids ?? []) : []
  const missing = edited === undefined && field.missing
  return (
    <div className={missing ? 'draft-field missing' : 'draft-field'}>
      <label htmlFor={id}>{label}</label>
      <textarea id={id} rows={2} value={value} aria-describedby={missing ? `${id}-flag` : undefined} onChange={(e) => onEdit(e.target.value)} />
      {missing && (
        <p className="flag" id={`${id}-flag`}>
          {t('consultation.live.missing')}
        </p>
      )}
      {sources.length > 0 && (
        <button type="button" className="quiet" onClick={() => onSources(sources)}>
          {t('consultation.live.sources', { count: sources.length })}
        </button>
      )}
    </div>
  )
}

function Signed({ draft, labels }: { draft: DraftEvent; labels: Map<string, string> }) {
  const t = useT()
  return (
    <>
      <p className="note signed">{t('consultation.live.signed')}</p>
      <dl className="draft-signed">
        {draft.fields.map((f) => (
          <div key={f.key}>
            <dt>{labels.get(f.key) ?? f.key}</dt>
            <dd>{f.value}</dd>
          </div>
        ))}
      </dl>
    </>
  )
}
