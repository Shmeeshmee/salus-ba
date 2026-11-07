import { describe, expect, it } from 'vitest'
import {
  at,
  cancelled,
  canStep,
  changeKind,
  count,
  dependents,
  earlierQuestions,
  insert,
  move,
  newItem,
  placement,
  problems,
  remove,
  rowsOf,
  step,
  update,
  type Item,
} from './builder'
import { questionnaires as t } from './messages'

// A questionnaire with every shape: a question, a section with two items, a question after it.
//   a (yes or no)
//   s (section)
//     x (short answer)
//     y (single choice: P, Q)
//   b (long answer)
function sample(): Item[] {
  const a = { ...newItem('boolean', 'a'), text: 'Placeholder yes or no' }
  const x = { ...newItem('string', 'x'), text: 'Placeholder short answer' }
  const y = { ...newItem('single', 'y'), text: 'Placeholder choice', options: ['P', 'Q'] }
  const s = { ...newItem('group', 's'), text: 'Placeholder section', items: [x, y] }
  const b = { ...newItem('text', 'b'), text: 'Placeholder long answer' }
  return [a, s, b]
}

// shape writes the tree as ids, a section's items in brackets: "a s[x y] b".
function shape(items: Item[]): string {
  return items.map((it) => (it.kind === 'group' ? `${it.linkId}[${it.items.map((c) => c.linkId).join(' ')}]` : it.linkId)).join(' ')
}

function moved(items: Item[], active: string, over: string): string {
  const res = move(items, active, over)
  if (!res.ok) throw new Error(res.error)
  return shape(res.items)
}

describe('rowsOf', () => {
  it('lists the items in document order with an end row per section', () => {
    expect(rowsOf(sample()).map((r) => r.id)).toEqual(['a', 's', 'x', 'y', 'end-s', 'b'])
  })

  it('lists only the top level while a section is moved', () => {
    expect(rowsOf(sample(), true).map((r) => r.id)).toEqual(['a', 's', 'b'])
  })
})

describe('move', () => {
  it.each([
    ['a', 's', 's[a x y] b', 'down onto a section header: first in the section'],
    ['a', 'y', 's[x y a] b', 'down onto the last item of a section: last in it'],
    ['a', 'end-s', 's[x y] a b', 'down onto the end of a section: after it'],
    ['b', 'end-s', 'a s[x y b]', 'up onto the end of a section: last in it'],
    ['b', 's', 'a b s[x y]', 'up onto a section header: before it'],
    ['x', 'a', 'x a s[y] b', 'out of a section, up'],
    ['y', 'b', 'a s[x] b y', 'out of a section, down'],
    ['y', 'x', 'a s[y x] b', 'within a section'],
  ])('moves %s over %s to "%s" (%s)', (active, over, want, _why) => {
    expect(moved(sample(), active, over)).toBe(want)
  })

  it('moves a section with its items, as a block over the top level', () => {
    expect(moved(sample(), 's', 'a')).toBe('s[x y] a b')
    expect(moved(sample(), 's', 'b')).toBe('a b s[x y]')
  })

  it('keeps the list for an unknown target or the item itself', () => {
    expect(moved(sample(), 'a', 'nope')).toBe('a s[x y] b')
    expect(moved(sample(), 'a', 'a')).toBe('a s[x y] b')
  })

  it('does not change the list it was given', () => {
    const items = sample()
    move(items, 'a', 'y')
    expect(shape(items)).toBe('a s[x y] b')
  })
})

describe('insert', () => {
  const c = () => ({ ...newItem('date', 'c'), text: 'Placeholder date' })
  const t2 = () => newItem('group', 't')

  it.each([
    [undefined, 'a s[x y] b c', 'at the end without a target'],
    ['a', 'c a s[x y] b', 'before a top-level item'],
    ['s', 'a c s[x y] b', 'before a section'],
    ['y', 'a s[x c y] b', 'before an item of a section'],
    ['end-s', 'a s[x y c] b', 'at the end of a section'],
  ])('puts a new item over %s as "%s" (%s)', (over, want, _why) => {
    const res = insert(sample(), c(), over)
    expect(res.ok && shape(res.items)).toBe(want)
  })

  it('puts a new section on the top level', () => {
    const res = insert(sample(), t2(), 's')
    expect(res.ok && shape(res.items)).toBe('a t[] s[x y] b')
  })

  it.each(['x', 'end-s'])('rejects a section inside a section (over %s)', (over) => {
    expect(insert(sample(), t2(), over)).toEqual({ ok: false, error: 'nested-section' })
  })

  it('puts an item into an empty section through its end row', () => {
    const items = [newItem('group', 'e')]
    expect(rowsOf(items).map((r) => r.id)).toEqual(['e', 'end-e'])
    const res = insert(items, newItem('string', 'n'), 'end-e')
    expect(res.ok && shape(res.items)).toBe('e[n]')
  })
})

describe('step (Move up and Move down)', () => {
  it('walks an item through every position, into and out of the section', () => {
    let items = sample()
    const seen: string[] = []
    while (canStep(items, 'a', 1)) {
      const res = step(items, 'a', 1)
      if (!res?.ok) throw new Error('step')
      items = res.items
      seen.push(shape(items))
    }
    expect(seen).toEqual(['s[a x y] b', 's[x a y] b', 's[x y a] b', 's[x y] a b', 's[x y] b a'])
  })

  it('moves a section past whole items and sections', () => {
    const res = step(sample(), 's', -1)
    expect(res?.ok && shape(res.items)).toBe('s[x y] a b')
  })

  it.each([
    ['a', -1, false],
    ['a', 1, true],
    ['b', 1, false],
    ['b', -1, true],
    ['s', -1, true],
    ['x', -1, true],
  ] as const)('tells whether %s can move %i: %s', (id, dir, want) => {
    expect(canStep(sample(), id, dir)).toBe(want)
    if (!want) expect(step(sample(), id, dir)).toBeUndefined()
  })
})

describe('placement', () => {
  it('is the position in the top level, or in the section', () => {
    expect(placement(sample(), 'b')).toEqual({ n: 3, total: 3 })
    expect(placement(sample(), 'y')).toMatchObject({ n: 2, total: 2, section: { linkId: 's' } })
  })
})

describe('remove', () => {
  function withConditions(): Item[] {
    const items = sample()
    const res = update(items, 'b', { condition: { question: 'a', answer: true } })
    return update(res, 'x', { condition: { question: 'a', answer: false } })
  }

  it('counts the items whose condition depends on the item', () => {
    expect(dependents(withConditions(), 'a')).toBe(2)
    expect(dependents(withConditions(), 'b')).toBe(0)
  })

  it('removes a condition whose question is deleted', () => {
    const items = remove(withConditions(), 'a')
    expect(shape(items)).toBe('s[x y] b')
    expect(items[1].condition).toBeUndefined()
    expect(items[0].items[0].condition).toBeUndefined()
  })

  it('removes a section with its items, and the conditions on them', () => {
    const items = update(sample(), 'b', { condition: { question: 'y', answer: 'P' } })
    expect(dependents(items, 's')).toBe(1)
    const left = remove(items, 's')
    expect(shape(left)).toBe('a b')
    expect(left[1].condition).toBeUndefined()
  })
})

describe('changeKind', () => {
  it('gives the item a new linkId, and points its dependents at it', () => {
    const items = update(sample(), 'b', { condition: { question: 'y', answer: 'P' } })
    const changed = changeKind(items, 'y', 'multiple', 'y2')
    expect(changed[1].items[1]).toMatchObject({ linkId: 'y2', kind: 'multiple', options: ['P', 'Q'] })
    expect(changed[2].condition).toEqual({ question: 'y2', answer: 'P' })
  })

  it('clears what the new type cannot have', () => {
    const items = update(sample(), 'y', { required: true, help: 'Placeholder help' })
    expect(changeKind(items, 'y', 'display', 'y2')[1].items[1]).toMatchObject({ kind: 'display', options: [], required: false, help: '' })
  })

  it('starts a new choice with two empty options', () => {
    expect(changeKind(sample(), 'b', 'single', 'b2')[2].options).toEqual(['', ''])
  })

  it('gives a new file upload the default kinds and one file, and drops them from another type', () => {
    const upload = changeKind(sample(), 'b', 'attachment', 'b2')[2]
    expect(upload).toMatchObject({ kind: 'attachment', accept: ['photo', 'document'], maxFiles: 1, options: [] })
    const back = changeKind([{ ...upload, accept: ['video'], maxFiles: 4 }], 'b2', 'string', 'b3')[0]
    expect(back.accept).toBeUndefined()
    expect(back.maxFiles).toBeUndefined()
  })
})

describe('newItem: a file upload', () => {
  it('accepts photos and documents, one file, and is a question no condition may depend on', () => {
    const upload = { ...newItem('attachment', 'f'), text: 'Placeholder upload' }
    expect(upload).toMatchObject({ kind: 'attachment', accept: ['photo', 'document'], maxFiles: 1 })
    expect(newItem('string', 's').accept).toBeUndefined()
    expect(earlierQuestions([upload, newItem('string', 'n')], 'n')).toEqual([])
  })
})

describe('count and earlierQuestions', () => {
  it('counts sections and their items', () => {
    expect(count(sample())).toBe(5)
  })

  it('offers the yes or no and choice questions above an item', () => {
    expect(earlierQuestions(sample(), 'b').map((i) => i.linkId)).toEqual(['a', 'y'])
    expect(earlierQuestions(sample(), 'x').map((i) => i.linkId)).toEqual(['a'])
    expect(earlierQuestions(sample(), 'a')).toEqual([])
  })
})

describe('problems', () => {
  it('finds none in a complete questionnaire', () => {
    expect(problems(sample()).size).toBe(0)
  })

  it.each([
    ['an empty text', (i: Item[]) => update(i, 'b', { text: ' ' }), 'b', t.noText],
    ['an empty option', (i: Item[]) => update(i, 'y', { options: ['P', ''] }), 'y', 'Fill in this option.'],
    ['an empty section', (i: Item[]) => update(i, 's', { items: [] }), 's', 'Add at least one item to this section.'],
    ['one option', (i: Item[]) => update(i, 'y', { options: ['P'] }), 'y', t.fewOptions],
    ['51 options', (i: Item[]) => update(i, 'y', { options: Array.from({ length: 51 }, (_, n) => `O${n}`) }), 'y', t.manyOptions],
    ['the same option twice', (i: Item[]) => update(i, 'y', { options: ['P', 'P'] }), 'y', t.sameOptions],
    ['a condition on a later question', (i: Item[]) => update(i, 'a', { condition: { question: 'y', answer: 'P' } }), 'a', t.conditionOrder],
    ['a condition on a missing question', (i: Item[]) => update(i, 'b', { condition: { question: 'gone', answer: true } }), 'b', t.conditionOrder],
    ['a condition on a short answer', (i: Item[]) => update(i, 'b', { condition: { question: 'x', answer: 'P' } }), 'b', t.conditionOrder],
    ['an answer that is not an option', (i: Item[]) => update(i, 'b', { condition: { question: 'y', answer: 'R' } }), 'b', t.conditionAnswer],
    ['a text answer to a yes or no', (i: Item[]) => update(i, 'b', { condition: { question: 'a', answer: 'P' } }), 'b', t.conditionAnswer],
  ])('names %s on its item', (_, change, id, message) => {
    expect(problems(change(sample())).get(id)).toEqual([message])
  })

  it.each([
    ['no kind of file', { accept: [] }, [t.noFileKind]],
    ['no file', { maxFiles: 0 }, [t.fileCount]],
    ['11 files', { maxFiles: 11 }, [t.fileCount]],
    ['a part of a file', { maxFiles: 1.5 }, [t.fileCount]],
    ['an empty number', { maxFiles: Number.NaN }, [t.fileCount]],
    ['10 files of every kind', { maxFiles: 10, accept: ['photo', 'document', 'video'] }, undefined],
  ] as [string, Partial<Item>, string[] | undefined][])('names a file upload with %s (W6)', (_, patch, messages) => {
    const items = [{ ...newItem('attachment', 'f'), text: 'Placeholder upload', ...patch }]
    expect(problems(items).get('f')).toEqual(messages)
  })

  it('finds a condition moved above its question', () => {
    const items = update(sample(), 'b', { condition: { question: 'y', answer: 'P' } })
    const res = move(items, 'b', 'a')
    expect(res.ok && problems(res.items).get('b')).toEqual([t.conditionOrder])
  })
})

describe('at and cancelled (W42)', () => {
  const label = (it: Item) => it.text || t.kinds[it.kind]
  it('says the position in the top level, after a move, in a section, and back after a cancel', () => {
    expect(at(sample(), 'b', label, false)).toBe('Placeholder long answer is at position 3 of 3.')
    expect(at(sample(), 'b', label, true)).toBe('Placeholder long answer moved to position 3 of 3.')
    expect(at(sample(), 'x', label, true)).toBe('Placeholder short answer is in Placeholder section, at position 1 of 2.')
    expect(at([newItem('string', 'n')], 'n', label, false)).toBe('Short answer is at position 1 of 1.')
    expect(cancelled(sample(), 'y', label)).toBe('Move cancelled. Placeholder choice is back at position 2 of 2.')
  })
})
