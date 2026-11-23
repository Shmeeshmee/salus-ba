import { useId, useState, type ReactNode } from 'react'
import { isChoice, isQuestion, type Item } from './builder'
import { discarded, enabled, enabledAnswers, type Answer, type Answers } from './enableWhen'
import { useT } from './i18n/useT'
import { Confirm } from './ui'

type Props = { title: string; description: string; items: Item[]; onClose: () => void }
type Pending = { linkId: string; next: Answer | undefined; lost: number }

// BuilderPreview is the draft as the patient fills it in (SAL-43): branches open and close as answers change,
// as enableWhen.ts decides (the app's evaluator follows the same FHIR rules), and a change that would hide
// answers already given asks first. Nothing here is saved.
export function BuilderPreview({ title, description, items, onClose }: Props) {
  const t = useT()
  const [answers, setAnswers] = useState<Answers>({})
  const [pending, setPending] = useState<Pending>()
  const on = enabled(items, answers)

  function answer(linkId: string, next: Answer | undefined) {
    const lost = discarded(items, answers, linkId, next).length
    if (lost > 0) setPending({ linkId, next, lost })
    else setAnswers({ ...answers, [linkId]: next })
  }

  function change() {
    if (!pending) return
    setAnswers(enabledAnswers(items, { ...answers, [pending.linkId]: pending.next }))
    setPending(undefined)
  }

  const show = (it: Item) =>
    on.has(it.linkId) && (
      <li key={it.linkId}>
        <PreviewItem item={it} answers={answers} onAnswer={answer}>
          {it.items.map(show)}
        </PreviewItem>
        {pending?.linkId === it.linkId && (
          <Confirm
            question={t('questionnaire.changeAnswerQuestion')}
            lines={[t('questionnaire.changeAnswerWarning', { n: pending.lost })]}
            confirm={t('questionnaire.changeAnswer')}
            cancel={t('questionnaire.keepAnswer')}
            danger
            onConfirm={change}
            onCancel={() => setPending(undefined)}
          />
        )}
      </li>
    )

  return (
    <div className="q-preview">
      <p className="notice">{t('questionnaire.previewHint')}</p>
      <div className="preview-sheet">
        <h2>{title.trim() || t('questionnaire.untitled')}</h2>
        {description.trim() !== '' && <p className="muted">{description}</p>}
        <ul className="preview-items">{items.map(show)}</ul>
      </div>
      <div className="actions">
        <button type="button" onClick={() => setAnswers({})}>
          {t('questionnaire.startAgain')}
        </button>
        <button type="button" className="quiet" onClick={onClose}>
          {t('questionnaire.closePreview')}
        </button>
      </div>
    </div>
  )
}

type ItemProps = { item: Item; answers: Answers; onAnswer: (linkId: string, a: Answer | undefined) => void; children: ReactNode }

function PreviewItem({ item, answers, onAnswer, children }: ItemProps) {
  const t = useT()
  if (item.kind === 'group') {
    return (
      <fieldset className="preview-section">
        <legend>{item.text}</legend>
        {item.help && <p className="hint">{item.help}</p>}
        <ul className="preview-items">{children}</ul>
      </fieldset>
    )
  }
  if (item.kind === 'display') return <p>{item.text}</p>
  const legend = (
    <>
      {item.text}
      {item.required && <span className="tag">{t('questionnaire.required')}</span>}
    </>
  )
  const value = answers[item.linkId]
  const set = (a: Answer | undefined) => onAnswer(item.linkId, a)
  if (item.kind === 'boolean' || isChoice(item.kind)) return <Choices item={item} legend={legend} value={value} onSet={set} />
  return <Field item={item} legend={legend} value={value} onSet={set} />
}

type FieldProps = { item: Item; legend: ReactNode; value: Answer | undefined; onSet: (a: Answer | undefined) => void }

function Choices({ item, legend, value, onSet }: FieldProps) {
  const t = useT()
  const name = useId()
  const options: { value: boolean | string; text: string }[] =
    item.kind === 'boolean'
      ? [
          { value: true, text: t('questionnaire.yes') },
          { value: false, text: t('questionnaire.no') },
        ]
      : item.options.map((o) => ({ value: o, text: o }))
  const many = item.kind === 'multiple'
  const list = Array.isArray(value) ? value : []
  return (
    <fieldset className="choice">
      <legend>{legend}</legend>
      {item.help && <p className="hint">{item.help}</p>}
      {options.map((o, i) => (
        <div className="check" key={i}>
          <input
            id={`${name}-${i}`}
            name={name}
            type={many ? 'checkbox' : 'radio'}
            checked={many ? list.includes(String(o.value)) : value === o.value}
            onChange={(e) => onSet(many ? (e.target.checked ? [...list, String(o.value)] : list.filter((x) => x !== o.value)) : o.value)}
          />
          <label htmlFor={`${name}-${i}`}>{o.text}</label>
        </div>
      ))}
    </fieldset>
  )
}

function Field({ item, legend, value, onSet }: FieldProps) {
  const t = useT()
  const id = useId()
  if (!isQuestion(item.kind) || item.kind === 'attachment') {
    return (
      <div className="field">
        <p className="preview-label">{legend}</p>
        <p className="hint">{t('questionnaire.previewFiles')}</p>
      </div>
    )
  }
  const number = item.kind === 'integer' || item.kind === 'decimal'
  const text = value === undefined ? '' : String(value)
  const read = (v: string): Answer | undefined => (v === '' ? undefined : number ? Number(v) : v)
  return (
    <div className="field">
      <label htmlFor={id}>{legend}</label>
      {item.help && <p className="hint">{item.help}</p>}
      {item.kind === 'text' ? (
        <textarea id={id} rows={3} value={text} onChange={(e) => onSet(read(e.target.value))} />
      ) : (
        <input
          id={id}
          className={number || item.kind === 'date' ? 'short' : undefined}
          type={number ? 'number' : item.kind === 'date' ? 'date' : 'text'}
          step={item.kind === 'decimal' ? 'any' : undefined}
          value={text}
          onChange={(e) => onSet(read(e.target.value))}
        />
      )}
    </div>
  )
}
