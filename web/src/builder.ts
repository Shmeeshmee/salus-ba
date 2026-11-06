import { questionnaires as t } from './messages'

// The builder's item list as pure functions, so moves, inserts and checks are tested without the DOM.
// Single and multiple choice are separate kinds here (W11); the API knows them as choice with repeats.
export type Kind = 'group' | 'display' | 'string' | 'text' | 'boolean' | 'single' | 'multiple' | 'integer' | 'decimal' | 'date'

export const kinds: Kind[] = ['group', 'display', 'string', 'text', 'boolean', 'single', 'multiple', 'integer', 'decimal', 'date']

export const maxItems = 200
export const maxTitle = 200
export const maxDescription = 1000
export const maxText = 500
export const maxHelp = 500
export const maxOptions = 50
export const maxOption = 200

// Condition shows an item only when the earlier yes or no or choice question is answered answer.
export type Condition = { question: string; answer: boolean | string }

// Item is one item of the questionnaire; items is a section's content, empty for every other kind.
export type Item = {
  linkId: string
  kind: Kind
  text: string
  required: boolean
  help: string
  options: string[]
  condition?: Condition
  items: Item[]
}

// Row is one line of the list as it is shown and dragged: an item, or the end of a section (where an item is
// dropped to go last in it).
export type Row = { id: string; item: Item; depth: 0 | 1 } | { id: string; end: Item }

export type Result = { ok: true; items: Item[] } | { ok: false; error: 'nested-section' }

export const isChoice = (k: Kind) => k === 'single' || k === 'multiple'
export const isQuestion = (k: Kind) => k !== 'group' && k !== 'display'
const conditional = (k: Kind) => k === 'boolean' || isChoice(k)

export function endId(section: Item): string {
  return `end-${section.linkId}`
}

export function newItem(kind: Kind, linkId: string): Item {
  return { linkId, kind, text: '', required: false, help: '', options: isChoice(kind) ? ['', ''] : [], items: [] }
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
function all(items: Item[]): Item[] {
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

// dependents counts the other items whose condition is on the item (or on an item of the section).
export function dependents(items: Item[], linkId: string): number {
  const gone = removed(items, linkId)
  return all(items).filter((it) => !gone.has(it.linkId) && it.condition && gone.has(it.condition.question)).length
}

// remove deletes the item (a section with its items) and the conditions that depended on it.
export function remove(items: Item[], linkId: string): Item[] {
  const gone = removed(items, linkId)
  const kept = items.filter((it) => !gone.has(it.linkId)).map((it) => ({ ...it, items: it.items.filter((c) => !gone.has(c.linkId)) }))
  return map(kept, (it) => (it.condition && gone.has(it.condition.question) ? { ...it, condition: undefined } : it))
}

// changeKind gives the item another type under a new linkId (SPEC §2 "linkIds": a linkId keeps its type
// across versions), keeps what the new type can have, and points the conditions on it at the new linkId.
export function changeKind(items: Item[], linkId: string, kind: Kind, newLinkId: string): Item[] {
  return map(items, (it) => {
    if (it.linkId === linkId) {
      const options = isChoice(kind) ? (isChoice(it.kind) ? it.options : ['', '']) : []
      return { ...it, linkId: newLinkId, kind, options, required: isQuestion(kind) && it.required, help: kind === 'display' ? '' : it.help }
    }
    if (it.condition?.question === linkId) return { ...it, condition: { ...it.condition, question: newLinkId } }
    return it
  })
}

// earlierQuestions are the yes or no and choice questions above the item, which a condition may depend on.
export function earlierQuestions(items: Item[], linkId: string): Item[] {
  const list = all(items)
  const at = list.findIndex((it) => it.linkId === linkId)
  return list.slice(0, Math.max(at, 0)).filter((it) => conditional(it.kind))
}

const blank = (s: string) => s.trim() === ''

// problems are the messages to show on each item (W27, W28), by linkId, checked as the API checks them.
export function problems(items: Item[]): Map<string, string[]> {
  const found = new Map<string, string[]>()
  const seen = new Map<string, Item>()
  for (const it of all(items)) {
    const messages = [...textProblems(it), ...optionProblems(it), ...conditionProblems(it, seen)]
    if (messages.length > 0) found.set(it.linkId, messages)
    seen.set(it.linkId, it)
  }
  return found
}

function textProblems(it: Item): string[] {
  return [
    ...(blank(it.text) ? [t.noText] : []),
    ...(isChoice(it.kind) && it.options.some(blank) ? [t.emptyOption] : []),
    ...(it.kind === 'group' && it.items.length === 0 ? [t.emptySection] : []),
  ]
}

function optionProblems(it: Item): string[] {
  if (!isChoice(it.kind)) return []
  if (it.options.length < 2) return [t.fewOptions]
  if (it.options.length > maxOptions) return [t.manyOptions]
  const filled = it.options.filter((o) => !blank(o))
  return new Set(filled).size < filled.length ? [t.sameOptions] : []
}

function conditionProblems(it: Item, seen: Map<string, Item>): string[] {
  if (!it.condition) return []
  const on = seen.get(it.condition.question)
  if (!on || !conditional(on.kind)) return [t.conditionOrder]
  const answer = it.condition.answer
  const fits = on.kind === 'boolean' ? typeof answer === 'boolean' : typeof answer === 'string' && on.options.includes(answer)
  return fits ? [] : [t.conditionAnswer]
}

// at is where the item is, said as W42 says it: in a section with the section's title, else at (or, after a
// move, moved to) its position. label is the item's text, or its type while it has none.
export function at(items: Item[], id: string, label: (it: Item) => string, moved: boolean): string {
  const it = find(items, id)
  if (!it) return ''
  const p = placement(items, id)
  if (p.section) return t.inSection(label(it), label(p.section), p.n, p.total)
  return moved ? t.moved(label(it), p.n, p.total) : t.at(label(it), p.n, p.total)
}

// cancelled is W42's cancelled move: the item back where it was.
export function cancelled(items: Item[], id: string, label: (it: Item) => string): string {
  const it = find(items, id)
  if (!it) return ''
  const p = placement(items, id)
  return t.cancelled(label(it), p.n, p.total)
}
