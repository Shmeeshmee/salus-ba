import type { useT } from './i18n/useT'

// The builder's item list as pure functions, so moves, inserts and checks are tested without the DOM.
// Single and multiple choice are separate kinds here (W11); the API knows them as choice with repeats.
export type Kind = 'group' | 'display' | 'string' | 'text' | 'boolean' | 'single' | 'multiple' | 'integer' | 'decimal' | 'date' | 'attachment'

export const kinds: Kind[] = ['group', 'display', 'string', 'text', 'boolean', 'single', 'multiple', 'integer', 'decimal', 'date', 'attachment']

// FileKind is a kind of file a file upload item accepts (attachments SPEC §2 "Item model"), in the order the
// builder lists them and the API is sent them.
export type FileKind = 'photo' | 'document' | 'video'

export const fileKinds: FileKind[] = ['photo', 'document', 'video']
export const maxFilesLimit = 10
// A new file upload takes one photo or document: the least a practitioner has to narrow down.
const defaultAccept: FileKind[] = ['photo', 'document']

export const maxItems = 200
export const maxTitle = 200
export const maxDescription = 1000
export const maxText = 500
export const maxHelp = 500
export const maxOptions = 50
export const maxOption = 200

// Condition is one FHIR R4 enableWhen: the earlier question, the operator and the value it is compared with
// (exists takes true for answered, false for not answered). An item's conditions combine by its enableBehavior,
// all when it has none.
export type Operator = 'exists' | '=' | '!=' | '>' | '<' | '>=' | '<='
export type Condition = { question: string; operator: Operator; answer: boolean | string | number }
export type Behavior = 'all' | 'any'

// Item is one item of the questionnaire; items is a section's content, empty for every other kind. accept and
// maxFiles are a file upload's only.
export type Item = {
  linkId: string
  kind: Kind
  text: string
  required: boolean
  help: string
  options: string[]
  enableWhen?: Condition[]
  enableBehavior?: Behavior
  items: Item[]
  accept?: FileKind[]
  maxFiles?: number
}

// Row is one line of the list as it is shown and dragged: an item, or the end of a section (where an item is
// dropped to go last in it).
export type Row = { id: string; item: Item; depth: 0 | 1 } | { id: string; end: Item }

export type Result = { ok: true; items: Item[] } | { ok: false; error: 'nested-section' }

export const isChoice = (k: Kind) => k === 'single' || k === 'multiple'
export const isQuestion = (k: Kind) => k !== 'group' && k !== 'display'

// operatorsFor is the operators a condition on a question of the kind may use (SAL-43): yes or no and choices is
// and is not; numbers every comparison; a date before, after and on; any question answered or not (exists).
export function operatorsFor(kind: Kind): Operator[] {
  if (kind === 'boolean' || isChoice(kind)) return ['=', '!=', 'exists']
  if (kind === 'integer' || kind === 'decimal') return ['=', '!=', '>', '>=', '<', '<=', 'exists']
  if (kind === 'date') return ['<', '>', '=', 'exists']
  return isQuestion(kind) ? ['exists'] : []
}

// on is whether a condition names the linkId.
const on = (it: Item, ids: Set<string>) => (it.enableWhen ?? []).some((c) => ids.has(c.question))

// textLabel is the message id of the label of the item's text: a section's title, instructions, or a question.
export function textLabel(kind: Kind): string {
  if (kind === 'group') return 'questionnaire.sectionTitle'
  if (kind === 'display') return 'questionnaire.instructions'
  return 'questionnaire.question'
}

export function endId(section: Item): string {
  return `end-${section.linkId}`
}

export function newItem(kind: Kind, linkId: string): Item {
  const it: Item = { linkId, kind, text: '', required: false, help: '', options: isChoice(kind) ? ['', ''] : [], items: [] }
  return kind === 'attachment' ? { ...it, accept: defaultAccept, maxFiles: 1 } : it
}

// rowsOf flattens the list in document order. While a section is moved, only the top level is listed: a
// section moves as a block and never lands inside another.
export function rowsOf(items: Item[], topOnly = false): Row[] {
  const rows: Row[] = []
  for (const it of items) {
    rows.push({ id: it.linkId, item: it, depth: 0 })
    if (it.kind !== 'group' || topOnly) continue
    for (const child of it.items) rows.push({ id: child.linkId, item: child, depth: 1 })
    rows.push({ id: endId(it), end: it })
  }
  return rows
}

// build turns rows back into the list: what follows a section's header up to its end row is in it.
function build(rows: Row[]): Result {
  const top: Item[] = []
  let open: Item | undefined
  for (const r of rows) {
    if ('end' in r) {
      open = undefined
      continue
    }
    if (r.item.kind !== 'group') {
      ;(open ? open.items : top).push(r.item)
      continue
    }
    if (open) return { ok: false, error: 'nested-section' }
    const hasEnd = rows.some((x) => 'end' in x && x.end.linkId === r.item.linkId)
    const section = { ...r.item, items: hasEnd ? [] : r.item.items }
    top.push(section)
    if (hasEnd) open = section
  }
  return { ok: true, items: top }
}

// move puts the item active where the row over is, as a sortable list does: moved down it lands after over,
// moved up before it.
export function move(items: Item[], active: string, over: string): Result {
  const item = find(items, active)
  if (!item) return { ok: true, items }
  const rows = rowsOf(items, item.kind === 'group')
  const from = rows.findIndex((r) => r.id === active)
  const to = rows.findIndex((r) => r.id === over)
  if (from < 0 || to < 0 || from === to) return { ok: true, items }
  const next = [...rows]
  const [row] = next.splice(from, 1)
  next.splice(to, 0, row)
  return build(next)
}

// insert adds a new item before the row over (at the end of a section for its end row), or at the end.
export function insert(items: Item[], item: Item, over?: string): Result {
  const rows = rowsOf(items)
  const at = over === undefined ? -1 : rows.findIndex((r) => r.id === over)
  const next = [...rows]
  next.splice(at < 0 ? rows.length : at, 0, { id: item.linkId, item, depth: 0 })
  return build(next)
}

function neighbour(items: Item[], linkId: string, dir: -1 | 1): string | undefined {
  const item = find(items, linkId)
  if (!item) return undefined
  const rows = rowsOf(items, item.kind === 'group')
  return rows[rows.findIndex((r) => r.id === linkId) + dir]?.id
}

export function canStep(items: Item[], linkId: string, dir: -1 | 1): boolean {
  return neighbour(items, linkId, dir) !== undefined
}

// step is Move up (-1) and Move down (1): one row at a time, so an item steps into and out of sections. It is
// undefined at either end.
export function step(items: Item[], linkId: string, dir: -1 | 1): Result | undefined {
  const over = neighbour(items, linkId, dir)
  return over === undefined ? undefined : move(items, linkId, over)
}

// all is every item in document order.
export function all(items: Item[]): Item[] {
  return items.flatMap((it) => [it, ...it.items])
}

export function find(items: Item[], linkId: string): Item | undefined {
  return all(items).find((it) => it.linkId === linkId)
}

export function count(items: Item[]): number {
  return all(items).length
}

// placement is where the item is: its position in the top level, or in its section.
export function placement(items: Item[], linkId: string): { n: number; total: number; section?: Item } {
  const top = items.findIndex((it) => it.linkId === linkId)
  if (top >= 0) return { n: top + 1, total: items.length }
  for (const s of items) {
    const i = s.items.findIndex((it) => it.linkId === linkId)
    if (i >= 0) return { n: i + 1, total: s.items.length, section: s }
  }
  return { n: 0, total: 0 }
}

// map applies f to every item, sections' items included.
function map(items: Item[], f: (it: Item) => Item): Item[] {
  return items.map((it) => {
    const next = f(it)
    return next.items.length > 0 ? { ...next, items: next.items.map(f) } : next
  })
}

export function update(items: Item[], linkId: string, patch: Partial<Item>): Item[] {
  return map(items, (it) => (it.linkId === linkId ? { ...it, ...patch } : it))
}

function removed(items: Item[], linkId: string): Set<string> {
  const item = find(items, linkId)
  return new Set(item ? [item.linkId, ...item.items.map((c) => c.linkId)] : [])
}

// dependentsOf are the other items with a condition on the item (or on an item of the section).
export function dependentsOf(items: Item[], linkId: string): Item[] {
  const gone = removed(items, linkId)
  return all(items).filter((it) => !gone.has(it.linkId) && on(it, gone))
}

export function dependents(items: Item[], linkId: string): number {
  return dependentsOf(items, linkId).length
}

// without drops the conditions on the linkIds, and the behaviour with the last of them.
function without(it: Item, gone: Set<string>): Item {
  if (!on(it, gone)) return it
  const kept = (it.enableWhen ?? []).filter((c) => !gone.has(c.question))
  const { enableWhen: _w, enableBehavior: _b, ...rest } = it
  if (kept.length === 0) return rest
  return kept.length === 1 ? { ...rest, enableWhen: kept } : { ...it, enableWhen: kept }
}

// remove deletes the item (a section with its items) and the conditions that depended on it.
export function remove(items: Item[], linkId: string): Item[] {
  const gone = removed(items, linkId)
  const kept = items.filter((it) => !gone.has(it.linkId)).map((it) => ({ ...it, items: it.items.filter((c) => !gone.has(c.linkId)) }))
  return map(kept, (it) => without(it, gone))
}

// changeKind gives the item another type under a new linkId (SPEC §2 "linkIds": a linkId keeps its type
// across versions), keeps what the new type can have, and points the conditions on it at the new linkId.
export function changeKind(items: Item[], linkId: string, kind: Kind, newLinkId: string): Item[] {
  return map(items, (it) => {
    if (it.linkId === linkId) {
      const options = isChoice(kind) ? (isChoice(it.kind) ? it.options : ['', '']) : []
      const { accept: _a, maxFiles: _m, ...rest } = it
      const files = kind === 'attachment' ? { accept: defaultAccept, maxFiles: 1 } : {}
      return { ...rest, ...files, linkId: newLinkId, kind, options, required: isQuestion(kind) && it.required, help: kind === 'display' ? '' : it.help }
    }
    if (!on(it, new Set([linkId]))) return it
    return { ...it, enableWhen: (it.enableWhen ?? []).map((c) => (c.question === linkId ? { ...c, question: newLinkId } : c)) }
  })
}

// earlierQuestions are the questions above the item, which a condition may depend on.
export function earlierQuestions(items: Item[], linkId: string): Item[] {
  const list = all(items)
  const at = list.findIndex((it) => it.linkId === linkId)
  return list.slice(0, Math.max(at, 0)).filter((it) => isQuestion(it.kind))
}

const blank = (s: string) => s.trim() === ''

// problems are the messages to show on each item (W27, W28), as catalog message ids, by linkId, checked as the
// API checks them.
export function problems(items: Item[]): Map<string, string[]> {
  const found = new Map<string, string[]>()
  const seen = new Map<string, Item>()
  for (const it of all(items)) {
    const messages = [...textProblems(it), ...optionProblems(it), ...fileProblems(it), ...conditionProblems(it, seen)]
    if (messages.length > 0) found.set(it.linkId, messages)
    seen.set(it.linkId, it)
  }
  return found
}

function textProblems(it: Item): string[] {
  return [
    ...(blank(it.text) ? ['questionnaire.noText'] : []),
    ...(isChoice(it.kind) && it.options.some(blank) ? ['questionnaire.emptyOption'] : []),
    ...(it.kind === 'group' && it.items.length === 0 ? ['questionnaire.emptySection'] : []),
  ]
}

function optionProblems(it: Item): string[] {
  if (!isChoice(it.kind)) return []
  if (it.options.length < 2) return ['questionnaire.fewOptions']
  if (it.options.length > maxOptions) return ['questionnaire.manyOptions']
  const filled = it.options.filter((o) => !blank(o))
  return new Set(filled).size < filled.length ? ['questionnaire.sameOptions'] : []
}

function fileProblems(it: Item): string[] {
  if (it.kind !== 'attachment') return []
  const n = it.maxFiles ?? 0
  return [...((it.accept ?? []).length === 0 ? ['questionnaire.noFileKind'] : []), ...(Number.isInteger(n) && n >= 1 && n <= maxFilesLimit ? [] : ['questionnaire.fileCount'])]
}

// conditionProblems are W28 on each condition, as the API checks them (SAL-43): the question is above the item,
// the operator fits its type, and the value fits the operator and the question (one of a choice's options).
function conditionProblems(it: Item, seen: Map<string, Item>): string[] {
  const found = new Set<string>()
  for (const c of it.enableWhen ?? []) {
    const q = seen.get(c.question)
    if (!q || !isQuestion(q.kind)) found.add('questionnaire.conditionOrder')
    else if (!operatorsFor(q.kind).includes(c.operator)) found.add('questionnaire.conditionOperator')
    else if (!fits(q, c)) found.add('questionnaire.conditionAnswer')
  }
  return [...found]
}

function fits(q: Item, c: Condition): boolean {
  const a = c.answer
  if (c.operator === 'exists' || q.kind === 'boolean') return typeof a === 'boolean'
  if (isChoice(q.kind)) return typeof a === 'string' && q.options.includes(a)
  if (q.kind === 'integer') return Number.isInteger(a)
  if (q.kind === 'decimal') return typeof a === 'number' && Number.isFinite(a)
  return typeof a === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(a)
}

// at is where the item is, said as W42 says it: in a section with the section's title, else at (or, after a
// move, moved to) its position. label is the item's text, or its type while it has none.
export function at(items: Item[], id: string, label: (it: Item) => string, moved: boolean, t: ReturnType<typeof useT>): string {
  const it = find(items, id)
  if (!it) return ''
  const p = placement(items, id)
  const values = { item: label(it), n: p.n, total: p.total }
  if (p.section) return t('questionnaire.inSection', { ...values, section: label(p.section) })
  return t(moved ? 'questionnaire.moved' : 'questionnaire.at', values)
}

// cancelled is W42's cancelled move: the item back where it was.
export function cancelled(items: Item[], id: string, label: (it: Item) => string, t: ReturnType<typeof useT>): string {
  const it = find(items, id)
  if (!it) return ''
  const p = placement(items, id)
  return t('questionnaire.cancelled', { item: label(it), n: p.n, total: p.total })
}
