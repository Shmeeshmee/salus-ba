import { all, type Condition, type Item } from './builder'

// The FHIR R4 enableWhen evaluator (Questionnaire.item.enableWhen, enableBehavior) for the builder's preview:
// pure, so every operator and type is tested without the DOM. A multiple choice's answer is its list.
export type Answer = boolean | string | number | string[]
export type Answers = Record<string, Answer | undefined>

const answered = (a: Answer | undefined) => a !== undefined && a !== '' && !(Array.isArray(a) && a.length === 0)

function compare(a: boolean | string | number, b: Condition['answer']): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  return String(a) < String(b) ? -1 : String(a) > String(b) ? 1 : 0
}

function one(op: Condition['operator'], a: boolean | string | number, b: Condition['answer']): boolean {
  if (op === '=') return a === b
  if (op === '!=') return a !== b
  const d = compare(a, b)
  if (op === '>') return d > 0
  if (op === '>=') return d >= 0
  if (op === '<') return d < 0
  return d <= 0
}

// test is one condition against the answer to its question. exists holds when the question's being answered
// is the condition's answer; any other operator never holds on an unanswered question. On a multiple choice,
// = holds when one answer is the value and != when none is.
export function test(c: Condition, a: Answer | undefined): boolean {
  if (c.operator === 'exists') return answered(a) === c.answer
  if (!answered(a) || a === undefined) return false
  if (Array.isArray(a)) return c.operator === '!=' ? !a.includes(String(c.answer)) : a.some((x) => one(c.operator, x, c.answer))
  return one(c.operator, a, c.answer)
}

function holds(it: Item, answers: Answers): boolean {
  const conditions = it.enableWhen ?? []
  if (conditions.length === 0) return true
  const results = conditions.map((c) => test(c, answers[c.question]))
  return it.enableBehavior === 'any' ? results.some(Boolean) : results.every(Boolean)
}

// enabled is the linkIds of the items shown, in document order. An item inside a hidden section is hidden,
// and a hidden question counts as unanswered (a condition names only an earlier question, so one pass does).
export function enabled(items: Item[], answers: Answers): Set<string> {
  const out = new Set<string>()
  const seen: Answers = {}
  const visit = (it: Item, parentOn: boolean) => {
    const on = parentOn && holds(it, seen)
    if (on) {
      out.add(it.linkId)
      seen[it.linkId] = answers[it.linkId]
    }
    for (const child of it.items) visit(child, on)
  }
  for (const it of items) visit(it, true)
  return out
}

// enabledAnswers is what is submitted: the answers to the items shown only.
export function enabledAnswers(items: Item[], answers: Answers): Answers {
  const on = enabled(items, answers)
  return Object.fromEntries(Object.entries(answers).filter(([id, a]) => on.has(id) && answered(a)))
}

// discarded is the linkIds of the answers given that changing linkId's answer to next would hide.
export function discarded(items: Item[], answers: Answers, linkId: string, next: Answer | undefined): string[] {
  const on = enabled(items, { ...answers, [linkId]: next })
  return all(items)
    .map((it) => it.linkId)
    .filter((id) => id !== linkId && answered(answers[id]) && !on.has(id))
}
