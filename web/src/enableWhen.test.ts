import { describe, expect, it } from 'vitest'
import { newItem, type Condition, type Item } from './builder'
import { discarded, enabled, enabledAnswers, test as holds, type Answers } from './enableWhen'

// The ticket's gynaecology example with placeholder wording: a yes or no, a branch of two questions on Yes
// (the second branching again on a number), then the main question everyone answers.
const when = (question: string, operator: Condition['operator'], answer: Condition['answer']): Condition => ({ question, operator, answer })
function example(): Item[] {
  return [
    { ...newItem('boolean', 'had'), text: 'Placeholder yes or no' },
    { ...newItem('integer', 'month'), text: 'Placeholder month', enableWhen: [when('had', '=', true)] },
    { ...newItem('integer', 'age'), text: 'Placeholder age', enableWhen: [when('had', '=', true)] },
    { ...newItem('string', 'young'), text: 'Placeholder extra', enableWhen: [when('age', '<', 18)] },
    { ...newItem('string', 'main'), text: 'Placeholder main' },
  ]
}

describe('test', () => {
  it.each([
    ['= on yes or no', when('q', '=', true), true, true],
    ['= on yes or no, other answer', when('q', '=', true), false, false],
    ['!= on a choice', when('q', '!=', 'A'), 'B', true],
    ['= on a multiple choice, one of its answers', when('q', '=', 'B'), ['A', 'B'], true],
    ['!= on a multiple choice holding it', when('q', '!=', 'B'), ['A', 'B'], false],
    ['> on a number', when('q', '>', 20), 21, true],
    ['>= on a number', when('q', '>=', 20), 20, true],
    ['< on a number', when('q', '<', 18), 18, false],
    ['<= on a decimal', when('q', '<=', 1.5), 1.5, true],
    ['< on a date (before)', when('q', '<', '2026-01-01'), '2025-12-31', true],
    ['> on a date (after)', when('q', '>', '2026-01-01'), '2025-12-31', false],
    ['= on a date (on)', when('q', '=', '2026-01-01'), '2026-01-01', true],
    ['exists true, answered', when('q', 'exists', true), 'x', true],
    ['exists false, answered', when('q', 'exists', false), 'x', false],
  ])('%s', (_, c, answer, want) => {
    expect(holds(c, answer)).toBe(want)
  })

  it('holds nothing but exists false on an unanswered question', () => {
    expect(holds(when('q', '!=', 'A'), undefined)).toBe(false)
    expect(holds(when('q', 'exists', false), undefined)).toBe(true)
    expect(holds(when('q', 'exists', false), '')).toBe(true)
    expect(holds(when('q', 'exists', false), [])).toBe(true)
  })
})

describe('enabled', () => {
  it('takes the Yes path: the branch, then the main question', () => {
    expect([...enabled(example(), { had: true })]).toEqual(['had', 'month', 'age', 'main'])
  })

  it('takes the No path straight to the main question', () => {
    expect([...enabled(example(), { had: false })]).toEqual(['had', 'main'])
  })

  it('opens a nested branch on a number, and only while its own branch is open', () => {
    expect(enabled(example(), { had: true, age: 17 }).has('young')).toBe(true)
    expect(enabled(example(), { had: true, age: 30 }).has('young')).toBe(false)
    expect(enabled(example(), { had: false, age: 17 }).has('young')).toBe(false)
  })

  it('combines two conditions with all and with any', () => {
    const both = (behavior: 'all' | 'any'): Item[] => [
      newItem('boolean', 'a'),
      newItem('boolean', 'b'),
      { ...newItem('string', 'c'), enableWhen: [when('a', '=', true), when('b', '=', true)], enableBehavior: behavior },
    ]
    expect(enabled(both('all'), { a: true, b: false }).has('c')).toBe(false)
    expect(enabled(both('all'), { a: true, b: true }).has('c')).toBe(true)
    expect(enabled(both('any'), { a: true, b: false }).has('c')).toBe(true)
    expect(enabled(both('any'), { a: false, b: false }).has('c')).toBe(false)
  })

  it('hides a section with its items', () => {
    const items: Item[] = [newItem('boolean', 'a'), { ...newItem('group', 's'), enableWhen: [when('a', '=', true)], items: [newItem('string', 'x')] }]
    expect([...enabled(items, { a: false })]).toEqual(['a'])
    expect([...enabled(items, { a: true })]).toEqual(['a', 's', 'x'])
  })
})

describe('answers on the path taken', () => {
  const answers: Answers = { had: true, month: 3, age: 17, young: 'x', main: 'y' }

  it('keeps only the enabled items answers', () => {
    expect(enabledAnswers(example(), { ...answers, had: false })).toEqual({ had: false, main: 'y' })
  })

  it('names the answers a change would discard, nested branch included', () => {
    expect(discarded(example(), answers, 'had', false)).toEqual(['month', 'age', 'young'])
    expect(discarded(example(), answers, 'age', 30)).toEqual(['young'])
    expect(discarded(example(), answers, 'main', 'z')).toEqual([])
  })
})
