import { describe, expect, it } from 'vitest'
import type { Item } from './builder'
import { enabled, type Answer, type Answers } from './enableWhen'
import { fromFHIR, type FHIRItem, type FHIRQuestionnaire } from './questionnaire'

// The shared enableWhen vectors of test/testdata/enablewhen (docs/clinic-cms, package Q1): the server and the
// patient app run the same files, so the preview shows what the patient is asked and the server accepts.
type Value = { valueBoolean?: boolean; valueInteger?: number; valueDecimal?: number; valueString?: string; valueDate?: string }
type Vector = {
  name: string
  questionnaire: { item: FHIRItem[] }
  cases: { name: string; answers: Record<string, Value[]>; enabled: string[] }[]
}

const files = import.meta.glob<Vector>('../../test/testdata/enablewhen/*.json', { eager: true, import: 'default' })
const vectors: Vector[] = Object.keys(files)
  .sort()
  .map((f) => files[f])

// itemsOf reads the vector's items through the builder's own FHIR read.
function itemsOf(v: Vector): Item[] {
  const q = { resourceType: 'Questionnaire', id: 'v', url: 'urn:v', version: '1', status: 'draft', title: v.name, item: v.questionnaire.item, meta: { versionId: '1', lastUpdated: '' } }
  return fromFHIR(q as unknown as FHIRQuestionnaire).items
}

const valueOf = (x: Value): boolean | string | number => x.valueBoolean ?? x.valueInteger ?? x.valueDecimal ?? x.valueDate ?? x.valueString ?? ''

// answersOf is the preview's answers: a multiple choice's are its list, every other question's its one value.
function answersOf(items: Item[], given: Record<string, Value[]>): Answers {
  const kinds = new Map<string, Item['kind']>()
  const visit = (it: Item) => {
    kinds.set(it.linkId, it.kind)
    it.items.forEach(visit)
  }
  items.forEach(visit)
  const out: Answers = {}
  for (const [id, values] of Object.entries(given)) {
    out[id] = (kinds.get(id) === 'multiple' ? values.map((x) => String(valueOf(x))) : valueOf(values[0])) as Answer
  }
  return out
}

it('finds the vectors', () => expect(vectors.length).toBeGreaterThan(0))

describe.each(vectors.map((v) => [v.name, v] as const))('%s', (_, v) => {
  const items = itemsOf(v)
  it.each(v.cases.map((c) => [c.name, c] as const))('%s', (_, c) => {
    expect([...enabled(items, answersOf(items, c.answers))]).toEqual(c.enabled)
  })
})
