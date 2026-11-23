import { all, insert, rowsOf, type Condition, type Item, type Result } from './builder'

// The builder's view of the branching (SAL-43) over FHIR enableWhen: a branch is the items a condition shows,
// and the branches join again at the next item without one. Pure, so the tree is tested without the DOM.

// branchDepths is how deep in branches each item is: an item with conditions is one deeper than the deepest
// question they name, and an item in a section at least as deep as the section.
export function branchDepths(items: Item[]): Map<string, number> {
  const depth = new Map<string, number>()
  const visit = (it: Item, base: number) => {
    const named = (it.enableWhen ?? []).map((c) => (depth.get(c.question) ?? 0) + 1)
    const d = Math.max(base, ...named)
    depth.set(it.linkId, d)
    for (const child of it.items) visit(child, d)
  }
  for (const it of items) visit(it, 0)
  return depth
}

// joins is the items where branches join: an item less deep in branches than the one before it.
export function joins(items: Item[]): Set<string> {
  const depth = branchDepths(items)
  const list = all(items)
  return new Set(list.filter((it, i) => i > 0 && (depth.get(it.linkId) ?? 0) < (depth.get(list[i - 1].linkId) ?? 0)).map((it) => it.linkId))
}

// followUps are the items "If the answer is answer" asks: one condition, = answer, on the question.
export function followUps(items: Item[], question: string, answer: Condition['answer']): Item[] {
  return all(items).filter((it) => {
    const c = it.enableWhen ?? []
    return c.length === 1 && c[0].question === question && c[0].operator === '=' && c[0].answer === answer
  })
}

// addFollowUp puts item, shown only when question is answered answer, at the end of the question's branches:
// after the items following it that are deeper in branches than it.
export function addFollowUp(items: Item[], question: string, answer: Condition['answer'], item: Item): Result {
  const depth = branchDepths(items)
  const list = all(items)
  const at = list.findIndex((it) => it.linkId === question)
  if (at < 0) return { ok: true, items }
  const base = depth.get(question) ?? 0
  let last = list[at]
  for (const it of list.slice(at + 1)) {
    if ((depth.get(it.linkId) ?? 0) <= base) break
    last = it
  }
  // A branch that ends in a section the question is not in ends at the section's end.
  const section = items.find((s) => s.items.includes(last) && !s.items.some((c) => c.linkId === question))
  const end = last.kind === 'group' && last !== list[at] ? last : section
  const rows = rowsOf(items)
  const lastRow = end ? rows.findIndex((r) => 'end' in r && r.end.linkId === end.linkId) : rows.findIndex((r) => r.id === last.linkId)
  const follow = { ...item, enableWhen: [{ question, operator: '=' as const, answer }] }
  return insert(items, follow, rows[lastRow + 1]?.id)
}
