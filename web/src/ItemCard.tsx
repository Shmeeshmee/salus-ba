import { useId, useState, type CSSProperties } from 'react'
import {
  fileKinds,
  isChoice,
  isQuestion,
  kinds,
  maxFilesLimit,
  maxHelp,
  maxOption,
  maxOptions,
  maxText,
  operatorsFor,
  textLabel,
  type Behavior,
  type Condition,
  type FileKind,
  type Item,
  type Kind,
  type Operator,
} from './builder'
import { BuilderBranch } from './BuilderBranch'
import { ArrowIcon, CrossIcon, GripIcon } from './icons'
import { useList, useT } from './i18n/useT'
import { Confirm } from './ui'

// RowParts are what @dnd-kit attaches to the row: kept apart from the rest so no other prop is read as a ref.
type RowParts = {
  nodeRef: (el: HTMLElement | null) => void // the row, as @dnd-kit measures and moves it
  style: CSSProperties
  handle: Record<string, unknown> & { ref: (el: HTMLElement | null) => void }
  dragging: boolean
}

type EditProps = {
  item: Item
  depth: 0 | 1
  fresh: boolean // just added: its text takes the focus
  label: (it: Item) => string
  open: boolean
  problems: string[]
  missingIn?: string // B7, on an item a translation lacks a text of
  attention: boolean
  canUp: boolean
  canDown: boolean
  dependents: string[] // the items shown only through this one, by name (W19, SAL-43)
  branch: number // how deep in branches the item is
  joins: boolean // branches join at this item
  followUps: (answer: Condition['answer']) => Item[]
  onFollowUp: (answer: Condition['answer']) => void
  earlier: Item[] // the questions above the item a condition may depend on
  questionOf: (linkId: string) => Item | undefined
  onToggle: () => void
  onStep: (dir: -1 | 1) => void
  onDelete: () => void
  onChange: (patch: Partial<Item>) => void
  onKind: (kind: Kind) => void
}

export type CardProps = RowParts & EditProps

// ItemCard is one item of the builder: closed it shows the text, the type and whether it is required, with
// the handle and the item actions; open it edits the item, help text and the condition behind "More options".
export function ItemCard({ nodeRef, style, handle, dragging, ...p }: CardProps) {
  const t = useT()
  const { item } = p
  const [confirming, setConfirming] = useState(false)
  const list = useList()
  const name = p.label(item)
  const classes = ['item', item.kind === 'group' ? 'section' : '', p.depth === 1 ? 'nested' : '', p.branch > 0 ? 'in-branch' : '', p.attention ? 'attention' : '', dragging ? 'dragging' : '']
  const branchStyle = { ...style, '--branch': p.branch } as CSSProperties

  function remove() {
    if (p.dependents.length > 0) setConfirming(true)
    else p.onDelete()
  }

  return (
    <li ref={nodeRef} style={branchStyle} className={classes.filter(Boolean).join(' ')}>
      {p.joins && <p className="join-marker">{t('questionnaire.joinHere')}</p>}
      <div className="item-head">
        <button type="button" className="icon handle" aria-label={t('questionnaire.drag')} title={t('questionnaire.drag')} {...handle}>
          <GripIcon />
        </button>
        {/* Named by what it shows; the spaces keep the parts apart when it is read. */}
        <button type="button" className="item-toggle" aria-expanded={p.open} onClick={p.onToggle}>
          <span className="item-text">{name}</span> <span className="tag">{t(`questionnaire.kinds.${item.kind}`)}</span>
          <FilesSummary item={item} />
          {(item.enableWhen ?? []).length > 0 && (
            <>
              {' '}
              <span className="tag branch-tag">{t('questionnaire.conditional')}</span>
            </>
          )}
          {item.required && (
            <>
              {' '}
              <span className="tag">{t('questionnaire.required')}</span>
            </>
          )}
        </button>
        {/* aria-disabled, not disabled: a button disabled under the focus would drop it to the page. */}
        <button type="button" className="icon" aria-label={t('questionnaire.moveUp')} title={t('questionnaire.moveUp')} aria-disabled={!p.canUp} onClick={() => p.canUp && p.onStep(-1)}>
          <ArrowIcon direction="up" />
        </button>
        <button type="button" className="icon" aria-label={t('questionnaire.moveDown')} title={t('questionnaire.moveDown')} aria-disabled={!p.canDown} onClick={() => p.canDown && p.onStep(1)}>
          <ArrowIcon direction="down" />
        </button>
        <button type="button" className="icon" aria-label={t('questionnaire.delete')} title={t('questionnaire.delete')} onClick={remove}>
          <CrossIcon />
        </button>
      </div>
      {(p.problems.length > 0 || p.missingIn) && (
        <ul className="problems">
          {p.problems.map((m) => (
            <li key={m}>{t(m)}</li>
          ))}
          {p.missingIn && <li>{p.missingIn}</li>}
        </ul>
      )}
      {confirming && (
        <Confirm
          question={t('questionnaire.deleteQuestion')}
          lines={[t('questionnaire.dependents', { n: p.dependents.length }), t('questionnaire.dependentItems', { items: list(p.dependents) })]}
          confirm={t('questionnaire.delete')}
          cancel={t('questionnaire.cancel')}
          danger
          onConfirm={p.onDelete}
          onCancel={() => setConfirming(false)}
        />
      )}
      {p.open && <ItemEditor {...p} />}
    </li>
  )
}

function ItemEditor(p: EditProps) {
  const t = useT()
  const { item } = p
  const [more, setMore] = useState(false)
  const list = useList()
  const kindId = useId()
  return (
    <div className="item-body">
      {item.kind !== 'group' && (
        <div className="field">
          <label htmlFor={kindId}>{t('questionnaire.type')}</label>
          <select id={kindId} value={item.kind} onChange={(e) => p.onKind(e.target.value as Kind)} className="kind">
            {kinds
              .filter((k) => k !== 'group')
              .map((k) => (
                <option key={k} value={k}>
                  {t(`questionnaire.kinds.${k}`)}
                </option>
              ))}
          </select>
        </div>
      )}
      <TextField
        label={t(textLabel(item.kind))}
        value={item.text}
        max={maxText}
        long={item.kind === 'display'}
        autoFocus={p.fresh}
        onChange={(text) => p.onChange({ text })}
      />
      {isQuestion(item.kind) && (
        <div className="check">
          <input id={`${kindId}-required`} type="checkbox" checked={item.required} onChange={(e) => p.onChange({ required: e.target.checked })} />
          <label htmlFor={`${kindId}-required`}>{t('questionnaire.required')}</label>
        </div>
      )}
      {p.dependents.length > 0 && <p className="hint branch-dependents">{t('questionnaire.dependsOnThis', { items: list(p.dependents) })}</p>}
      {(item.kind === 'boolean' || isChoice(item.kind)) && <BuilderBranch item={item} label={p.label} followUps={p.followUps} onFollowUp={p.onFollowUp} />}
      {isChoice(item.kind) && <Options options={item.options} onChange={(options) => p.onChange({ options })} />}
      {item.kind === 'attachment' && <FileOptions item={item} onChange={p.onChange} />}
      <button type="button" className="quiet more-toggle" aria-expanded={more} onClick={() => setMore(!more)}>
        {t('questionnaire.moreOptions')}
      </button>
      {more && (
        <div className="more-options">
          {item.kind !== 'display' && <Help help={item.help} onChange={(help) => p.onChange({ help })} />}
          <ConditionEditor {...p} />
        </div>
      )}
    </div>
  )
}

type TextProps = { label: string; value: string; max: number; long?: boolean; hint?: string; autoFocus?: boolean; onChange: (v: string) => void }

// TextField is a text input with its limit: it cannot be passed, and the count shows once it is near.
export function TextField({ label, value, max, long, hint, autoFocus, onChange }: TextProps) {
  const t = useT()
  const id = useId()
  const near = value.length >= max * 0.8
  const describedBy = [hint ? `${id}-hint` : '', near ? `${id}-count` : ''].filter(Boolean).join(' ') || undefined
  const common = { id, value, maxLength: max, autoFocus, 'aria-describedby': describedBy }
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      {long ? <textarea {...common} rows={3} onChange={(e) => onChange(e.target.value)} /> : <input {...common} onChange={(e) => onChange(e.target.value)} />}
      {hint && (
        <p className="hint" id={`${id}-hint`}>
          {hint}
        </p>
      )}
      {near && (
        <p className="hint" id={`${id}-count`}>
          {t('questionnaire.characters', { n: value.length, max })}
        </p>
      )}
    </div>
  )
}

function Options({ options, onChange }: { options: string[]; onChange: (o: string[]) => void }) {
  const t = useT()
  const full = options.length >= maxOptions
  return (
    <fieldset className="options">
      <legend>{t('questionnaire.options')}</legend>
      <ol>
        {options.map((o, i) => (
          // Options have no id of their own: their position is their identity while they are edited.
          <li key={i}>
            <TextField label={t('questionnaire.option', { n: i + 1 })} value={o} max={maxOption} onChange={(v) => onChange(options.map((x, j) => (j === i ? v : x)))} />
            <button type="button" className="icon" aria-label={t('questionnaire.removeOption')} title={t('questionnaire.removeOption')} onClick={() => onChange(options.filter((_, j) => j !== i))}>
              <CrossIcon />
            </button>
          </li>
        ))}
      </ol>
      <div className="actions">
        <button type="button" disabled={full} onClick={() => onChange([...options, ''])}>
          {t('questionnaire.addOption')}
        </button>
      </div>
      {full && <p className="hint">{t('questionnaire.manyOptions')}</p>}
    </fieldset>
  )
}

// FilesSummary is W5 on a closed file upload, after its type: how many files it takes.
export function FilesSummary({ item }: { item: Item }) {
  const t = useT()
  if (item.kind !== 'attachment' || item.maxFiles === undefined || Number.isNaN(item.maxFiles)) return null
  return (
    <>
      {' '}
      <span className="tag">{t('questionnaire.maxFilesSummary', { n: item.maxFiles })}</span>
    </>
  )
}

// FileOptions are a file upload's own settings (W2 to W4): the kinds of file it accepts and how many.
function FileOptions({ item, onChange }: { item: Item; onChange: (patch: Partial<Item>) => void }) {
  const t = useT()
  const id = useId()
  const accept = item.accept ?? []
  const toggle = (k: FileKind, on: boolean) => onChange({ accept: fileKinds.filter((x) => (x === k ? on : accept.includes(x))) })
  const n = item.maxFiles ?? Number.NaN
  return (
    <>
      <fieldset className="choice file-kinds" aria-describedby={`${id}-accept`}>
        <legend>{t('questionnaire.acceptLabel')}</legend>
        {fileKinds.map((k) => (
          <div className="check" key={k}>
            <input id={`${id}-${k}`} type="checkbox" checked={accept.includes(k)} onChange={(e) => toggle(k, e.target.checked)} />
            <label htmlFor={`${id}-${k}`}>{t(`questionnaire.accept.${k}`)}</label>
          </div>
        ))}
        <p className="hint" id={`${id}-accept`}>
          {t('questionnaire.acceptHint')}
        </p>
      </fieldset>
      <div className="field">
        <label htmlFor={`${id}-n`}>{t('questionnaire.maxFilesLabel')}</label>
        <input
          id={`${id}-n`}
          type="number"
          className="short"
          min={1}
          max={maxFilesLimit}
          step={1}
          value={Number.isNaN(n) ? '' : String(n)}
          aria-describedby={`${id}-n-hint`}
          onChange={(e) => onChange({ maxFiles: e.target.value === '' ? Number.NaN : Number(e.target.value) })}
        />
        <p className="hint" id={`${id}-n-hint`}>
          {t('questionnaire.maxFilesHint')}
        </p>
      </div>
    </>
  )
}

function Help({ help, onChange }: { help: string; onChange: (h: string) => void }) {
  const t = useT()
  const [shown, setShown] = useState(help !== '')
  if (!shown) {
    return (
      <div className="actions">
        <button type="button" onClick={() => setShown(true)}>
          {t('questionnaire.addHelp')}
        </button>
      </div>
    )
  }
  return <TextField label={t('questionnaire.help')} value={help} max={maxHelp} long hint={t('questionnaire.helpHint')} onChange={onChange} />
}

// valueFor is the value a new condition on the question starts with for the operator.
function valueFor(q: Item, operator: Operator): Condition['answer'] {
  if (operator === 'exists' || q.kind === 'boolean') return true
  if (isChoice(q.kind)) return q.options[0] ?? ''
  if (q.kind === 'integer' || q.kind === 'decimal') return 0
  return ''
}

function conditionOn(q: Item): Condition {
  const operator = operatorsFor(q.kind)[0]
  return { question: q.linkId, operator, answer: valueFor(q, operator) }
}

// The operator select's values: exists is two, answered and not answered.
const opValue = (c: Condition) => (c.operator === 'exists' ? `exists:${String(c.answer)}` : c.operator)
const opKeys: Record<string, string> = { '=': 'eq', '!=': 'ne', '>': 'gt', '>=': 'ge', '<': 'lt', '<=': 'le' }

function opLabel(kind: Kind, value: string, t: ReturnType<typeof useT>): string {
  if (value === 'exists:true') return t('questionnaire.op.answered')
  if (value === 'exists:false') return t('questionnaire.op.unanswered')
  if (kind === 'date') return t(`questionnaire.op.${{ '<': 'before', '>': 'after', '=': 'on' }[value as '<' | '>' | '=']}`)
  return t(`questionnaire.op.${opKeys[value]}`)
}

function ConditionEditor(p: EditProps) {
  const t = useT()
  const { item, earlier } = p
  const id = useId()
  const list = item.enableWhen ?? []
  const set = (next: Condition[], behavior = item.enableBehavior) => {
    if (next.length === 0) return p.onChange({ enableWhen: undefined, enableBehavior: undefined })
    p.onChange(next.length > 1 ? { enableWhen: next, enableBehavior: behavior ?? 'all' } : { enableWhen: next, enableBehavior: undefined })
  }
  if (list.length === 0) {
    if (earlier.length === 0) return <p className="hint">{t('questionnaire.nothingToDependOn')}</p>
    return (
      <div className="actions">
        <button type="button" onClick={() => set([conditionOn(earlier[0])])}>
          {t('questionnaire.addCondition')}
        </button>
      </div>
    )
  }
  return (
    <div className="conditions">
      {list.length > 1 && (
        <div className="field">
          <label htmlFor={`${id}-b`}>{t('questionnaire.behavior')}</label>
          <select id={`${id}-b`} value={item.enableBehavior ?? 'all'} onChange={(e) => set(list, e.target.value as Behavior)}>
            <option value="all">{t('questionnaire.behaviors.all')}</option>
            <option value="any">{t('questionnaire.behaviors.any')}</option>
          </select>
        </div>
      )}
      {list.map((c, i) => (
        <ConditionRow
          key={i}
          {...p}
          c={c}
          lead={i === 0 ? t('questionnaire.showOnlyIf') : t(item.enableBehavior === 'any' ? 'questionnaire.orIf' : 'questionnaire.andIf')}
          onSet={(next) => set(list.map((x, j) => (j === i ? next : x)))}
          onRemove={() => set(list.filter((_, j) => j !== i))}
        />
      ))}
      {earlier.length > 0 && (
        <div className="actions">
          <button type="button" onClick={() => set([...list, conditionOn(earlier[earlier.length - 1])])}>
            {t('questionnaire.addAnother')}
          </button>
        </div>
      )}
    </div>
  )
}

type RowProps = EditProps & { c: Condition; lead: string; onSet: (c: Condition) => void; onRemove: () => void }

// ConditionRow is one condition: the earlier question, the comparison its type allows, and the value.
function ConditionRow({ c, lead, onSet, onRemove, ...p }: RowProps) {
  const t = useT()
  const id = useId()
  const current = p.questionOf(c.question)
  const choices = current && !p.earlier.includes(current) ? [...p.earlier, current] : p.earlier
  const ops = current ? operatorsFor(current.kind).flatMap((o) => (o === 'exists' ? ['exists:true', 'exists:false'] : [o])) : []
  const value = opValue(c)
  return (
    <div className="condition">
      <div className="field">
        <label htmlFor={`${id}-q`}>{lead}</label>
        <select
          id={`${id}-q`}
          value={c.question}
          onChange={(e) => {
            const q = p.questionOf(e.target.value)
            if (q) onSet(conditionOn(q))
          }}
        >
          {choices.map((q) => (
            <option key={q.linkId} value={q.linkId}>
              {p.label(q)}
            </option>
          ))}
        </select>
      </div>
      {current && (
        <div className="field">
          <label htmlFor={`${id}-o`}>{t('questionnaire.comparison')}</label>
          <select
            id={`${id}-o`}
            value={value}
            onChange={(e) => {
              const [op, answered] = e.target.value.split(':') as [Operator, string?]
              onSet({ ...c, operator: op, answer: op === 'exists' ? answered === 'true' : c.operator === 'exists' ? valueFor(current, op) : c.answer })
            }}
          >
            {!ops.includes(value) && <option value={value}>{value}</option>}
            {ops.map((o) => (
              <option key={o} value={o}>
                {opLabel(current.kind, o, t)}
              </option>
            ))}
          </select>
        </div>
      )}
      {current && c.operator !== 'exists' && <ConditionValue id={`${id}-a`} q={current} c={c} onSet={onSet} />}
      <div className="actions">
        <button type="button" className="quiet" onClick={onRemove}>
          {t('questionnaire.removeCondition')}
        </button>
      </div>
    </div>
  )
}

function ConditionValue({ id, q, c, onSet }: { id: string; q: Item; c: Condition; onSet: (c: Condition) => void }) {
  const t = useT()
  const label = <label htmlFor={id}>{t('questionnaire.answer')}</label>
  if (q.kind === 'integer' || q.kind === 'decimal' || q.kind === 'date') {
    const number = q.kind !== 'date'
    const shown = typeof c.answer === 'number' ? (Number.isNaN(c.answer) ? '' : String(c.answer)) : String(c.answer)
    return (
      <div className="field">
        {label}
        <input
          id={id}
          className="short"
          type={number ? 'number' : 'date'}
          step={q.kind === 'decimal' ? 'any' : 1}
          value={shown}
          onChange={(e) => onSet({ ...c, answer: number ? (e.target.value === '' ? Number.NaN : Number(e.target.value)) : e.target.value })}
        />
      </div>
    )
  }
  const answers = q.kind === 'boolean' ? ['true', 'false'] : q.options
  const value = String(c.answer)
  const text = (a: string) => (q.kind === 'boolean' ? t(a === 'true' ? 'questionnaire.yes' : 'questionnaire.no') : a)
  return (
    <div className="field">
      {label}
      <select id={id} value={value} onChange={(e) => onSet({ ...c, answer: q.kind === 'boolean' ? e.target.value === 'true' : e.target.value })}>
        {!answers.includes(value) && <option value={value}>{value}</option>}
        {answers.map((a) => (
          <option key={a} value={a}>
            {text(a)}
          </option>
        ))}
      </select>
    </div>
  )
}
