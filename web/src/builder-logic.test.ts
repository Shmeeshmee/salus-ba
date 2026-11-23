import { describe, expect, it } from 'vitest'
import { all, newItem, type Item } from './builder'
import { addFollowUp, branchDepths, followUps, joins } from './builder-logic'

const yes = (question: string) => [{ question, operator: '=' as const, answer: true }]
function example(): Item[] {
  return [
    { ...newItem('boolean', 'had'), text: 'Had' },
    { ...newItem('integer', 'month'), text: 'Month', enableWhen: yes('had') },
    { ...newItem('integer', 'age'), text: 'Age', enableWhen: yes('had') },
    { ...newItem('string', 'young'), text: 'Young', enableWhen: [{ question: 'age', operator: '<', answer: 18 }] },
    { ...newItem('string', 'main'), text: 'Main' },
  ]
}

describe('the branch tree', () => {
  it('nests a branch within a branch and joins back at the main question', () => {
    expect(Object.fromEntries(branchDepths(example()))).toEqual({ had: 0, month: 1, age: 1, young: 2, main: 0 })
    expect([...joins(example())]).toEqual(['main'])
  })

  it('puts a section and its items in the branch its condition opens', () => {
    const items: Item[] = [newItem('boolean', 'a'), { ...newItem('group', 's'), enableWhen: yes('a'), items: [newItem('string', 'x')] }, newItem('string', 'b')]
    expect(Object.fromEntries(branchDepths(items))).toEqual({ a: 0, s: 1, x: 1, b: 0 })
    expect([...joins(items)]).toEqual(['b'])
  })
})

describe('addFollowUp', () => {
  it('adds the question at the end of the branch, before the join', () => {
    const res = addFollowUp(example(), 'had', true, newItem('string', 'new'))
    expect(res.ok && all(res.items).map((it) => it.linkId)).toEqual(['had', 'month', 'age', 'young', 'new', 'main'])
    expect(res.ok && res.items[4].enableWhen).toEqual(yes('had'))
  })

  it('adds right after a question with no branch yet', () => {
    const res = addFollowUp(example(), 'main', 'x', newItem('string', 'new'))
    expect(res.ok && all(res.items).map((it) => it.linkId).slice(-2)).toEqual(['main', 'new'])
  })

  it('adds after a section that ends the branch', () => {
    const items: Item[] = [newItem('boolean', 'a'), { ...newItem('group', 's'), enableWhen: yes('a'), items: [newItem('string', 'x')] }, newItem('string', 'b')]
    const res = addFollowUp(items, 'a', true, newItem('string', 'new'))
    expect(res.ok && res.items.map((it) => it.linkId)).toEqual(['a', 's', 'new', 'b'])
  })

  it('stays in the section of a question inside one', () => {
    const items: Item[] = [{ ...newItem('group', 's'), items: [newItem('boolean', 'a'), newItem('string', 'x')] }]
    const res = addFollowUp(items, 'a', false, newItem('string', 'new'))
    expect(res.ok && res.items[0].items.map((it) => it.linkId)).toEqual(['a', 'new', 'x'])
  })
})

describe('followUps', () => {
  it('lists what each answer asks', () => {
    expect(followUps(example(), 'had', true).map((it) => it.linkId)).toEqual(['month', 'age'])
    expect(followUps(example(), 'had', false)).toEqual([])
  })
})
