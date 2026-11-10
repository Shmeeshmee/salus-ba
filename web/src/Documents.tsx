import { useEffect, useId, useState, type Dispatch } from 'react'
import { SignRefused, TranscriptionDeclined, type ConsultActions, type ConsultMessage, type DocumentTerms, type DraftEvent, type DraftField, type Template } from './consult'
import { useLanguage, useT } from './i18n/useT'
import { TermTextLines } from './terminology/Explain'
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
// filled fields with their sources and the flags on what the conversation has not covered, edit, and sign. A
// generation answers with the stored draft, which also comes on the live source (and again on every change); the
// edits stay on this page until it is signed. A sign needs every required field filled.
export function Documents({ appointmentId, drafts, dispatch, actions, onSources }: Props) {
  const t = useT()
  const { language } = useLanguage()
  const id = useId()
  const [templates, setTemplates] = useState<Templates>({ state: 'loading' })
  const [chosen, setChosen] = useState('')
  // waiting is the template being generated, until the generation answers: an event of its draft meanwhile may
  // be the previous version (a replay), so only the answer ends it.
  const [waiting, setWaiting] = useState<string>()
  // refused are the required fields a sign found empty, by key.
  const [refused, setRefused] = useState<string[]>([])
  // edits are the practitioner's field values by draft, then by field key, so no edit reaches another draft.
  const [edits, setEdits] = useState<Record<string, Record<string, string>>>({})
  const [asking, setAsking] = useState<Asking>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  useEffect(() => {
    let open = true
    actions.templates(language).then(
      (list) => open && setTemplates({ state: 'ready', list }),
      () => open && setTemplates({ state: 'failed' }),
    )
    return () => {
      open = false
    }
  }, [actions, language])

  if (templates.state === 'loading') return <p className="muted">{t('consultation.live.loadingTemplates')}</p>
  if (templates.state === 'failed') return <p className="muted">{t('consultation.live.templatesFailed')}</p>
  if (templates.list.length === 0) return <p className="muted">{t('consultation.live.noTemplates')}</p>

  const template = templates.list.find((x) => x.id === chosen)
  const current = latest(drafts, chosen)
  const generating = waiting === chosen
  const draft = generating ? undefined : current

  async function generate() {
    if (!template) return
    setAsking(undefined)
    setError(undefined)
    setRefused([])
    if (current) setEdits((e) => without(e, current.draft_id))
    setWaiting(template.id)
    try {
      const stored = await actions.generate(appointmentId, template.id, template.language)
      dispatch({ type: 'draft', version: 1, payload: stored })
    } catch (e) {
      setError(t(e instanceof TranscriptionDeclined ? 'consultation.live.transcriptionConsentRequired' : 'consultation.live.generateFailed'))
    }
    setWaiting((w) => (w === template.id ? undefined : w))
  }

  async function sign(d: DraftEvent) {
    setBusy(true)
    setError(undefined)
    try {
      const signed = await actions.sign(d, d.fields.map((f) => ({ key: f.key, value: valueOf(d, f) })))
      dispatch({ type: 'draft', version: 1, payload: signed })
      setEdits((e) => without(e, d.draft_id))
    } catch (e) {
      if (e instanceof SignRefused) setRefused(e.keys)
      else setError(t('consultation.live.signFailed'))
    }
    setBusy(false)
    setAsking(undefined)
  }

  const labels = new Map(template?.fields.map((f) => [f.key, f.label]))
  const own = (draft && edits[draft.draft_id]) ?? {}
  const valueOf = (d: DraftEvent, f: DraftField) => edits[d.draft_id]?.[f.key] ?? f.value
  const blank = (f: DraftField) => (own[f.key] ?? f.value).trim() === ''
  const required = new Set([...(template?.fields.filter((f) => f.required).map((f) => f.key) ?? []), ...refused])
  // unfilled are the required fields of the draft still empty: they block a sign.
  const unfilled = new Set(draft?.fields.filter((f) => required.has(f.key) && blank(f)).map((f) => f.key))
  const shownUnfilled = new Set([...unfilled].filter((k) => refused.includes(k)))

  function ask(a: Asking | undefined) {
    if (a === 'sign' && unfilled.size > 0) {
      setRefused([...unfilled])
      return
    }
    if (a === 'regenerate' && Object.keys(own).length === 0) return void generate()
    setAsking(a)
  }

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
      <Alert message={error ?? (shownUnfilled.size > 0 ? t('consultation.live.requiredEmpty', { count: shownUnfilled.size }) : undefined)} />
      {generating && (
        <p className="muted" role="status">
          {t('consultation.live.generating')}
        </p>
      )}
      {draft?.status === 'final' && <Signed draft={draft} labels={labels} actions={actions} />}
      {draft && draft.status !== 'final' && (
        <>
          <p className="note">{t('consultation.live.draftNote')}</p>
          {draft.fields.map((f) => (
            <FieldEdit
              key={f.key}
              field={f}
              label={labels.get(f.key) ?? f.key}
              edited={own[f.key]}
              invalid={shownUnfilled.has(f.key)}
              onEdit={(value) => setEdits((e) => ({ ...e, [draft.draft_id]: { ...e[draft.draft_id], [f.key]: value } }))}
              onSources={onSources}
            />
          ))}
          <DraftActions
            asking={asking}
            busy={busy}
            empty={draft.fields.filter(blank).length}
            onAsk={ask}
            onSign={() => void sign(draft)}
            onRegenerate={() => void generate()}
          />
        </>
      )}
    </div>
  )
}

// without is edits less one draft's.
function without(edits: Record<string, Record<string, string>>, draftId: string) {
  const rest = { ...edits }
  delete rest[draftId]
  return rest
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

type FieldProps = {
  field: DraftField
  label: string
  edited?: string
  invalid: boolean
  onEdit: (value: string) => void
  onSources: (ids: string[]) => void
}

// FieldEdit is one field of the draft. Once the practitioner types in it, its sources and its flag no longer
// describe it, so both go.
// invalid marks a required field a sign found empty.
function FieldEdit({ field, label, edited, invalid, onEdit, onSources }: FieldProps) {
  const t = useT()
  const id = useId()
  const value = edited ?? field.value
  const sources = edited === undefined ? (field.source_segment_ids ?? []) : []
  const missing = edited === undefined && field.missing
  return (
    <div className={missing ? 'draft-field missing' : 'draft-field'}>
      <label htmlFor={id}>{label}</label>
      <textarea
        id={id}
        rows={2}
        value={value}
        aria-invalid={invalid || undefined}
        aria-describedby={missing ? `${id}-flag` : undefined}
        onChange={(e) => onEdit(e.target.value)}
      />
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

// Signed is a signed draft, read only, with its explained terms (SAL-13) marked once they are read: an overlay, the
// fields are shown as they were signed. Without the terms (not read yet, or not readable) they are plain text.
function Signed({ draft, labels, actions }: { draft: DraftEvent; labels: Map<string, string>; actions: ConsultActions }) {
  const t = useT()
  const [terms, setTerms] = useState<DocumentTerms>()
  useEffect(() => {
    let open = true
    actions.terms(draft.draft_id).then(
      (got) => open && setTerms(got),
      () => undefined,
    )
    return () => {
      open = false
    }
  }, [actions, draft.draft_id, draft.version_id])
  const lines = new Map(terms?.sections.map((s) => [s.key, s.lines]))
  return (
    <>
      <p className="note signed">{t('consultation.live.signed')}</p>
      <dl className="draft-signed">
        {draft.fields.map((f) => (
          <div key={f.key}>
            <dt>{labels.get(f.key) ?? f.key}</dt>
            <dd>
              <TermTextLines text={f.value} lines={lines.get(f.key)} />
            </dd>
          </div>
        ))}
      </dl>
    </>
  )
}
