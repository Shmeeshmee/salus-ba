import { describe, expect, it } from 'vitest'
import type { Questionnaire } from './api/fhir'
import { newItem } from './builder'
import { fromFHIR, fromJSON, libraryOf, toRequest } from './questionnaire'
import { questionnaireFHIR, questionnaireJSON } from './test/questionnaires'

describe('fromFHIR', () => {
  it('reads a version: help text from its help item, single and multiple choice apart, the condition', () => {
    const v = fromFHIR(questionnaireFHIR())
    expect(v).toMatchObject({ id: 'q1', url: 'https://salus.invalid/fhir/Questionnaire/s1', version: 1, status: 'draft', versionId: 3, title: 'Placeholder intake' })
    expect(v.items).toEqual([
      { ...newItem('display', 'intro'), text: 'Placeholder instructions' },
      { ...newItem('boolean', 'smoker'), text: 'Placeholder yes or no', required: true, help: 'Placeholder help' },
      {
        ...newItem('group', 'sec'),
        text: 'Placeholder section',
        help: 'Placeholder section help',
        condition: { question: 'smoker', answer: true },
        items: [
          { ...newItem('single', 'one'), text: 'Placeholder single', options: ['A', 'B'] },
          { ...newItem('multiple', 'many'), text: 'Placeholder multiple', options: ['C', 'D'], condition: { question: 'one', answer: 'B' } },
        ],
      },
    ])
  })
})

describe('fromJSON', () => {
  it('reads the JSON API answer as the FHIR read', () => {
    expect(fromJSON(questionnaireJSON())).toEqual(fromFHIR(questionnaireFHIR()))
  })
})

describe('toRequest', () => {
  it('writes the items as the JSON API takes them', () => {
    const v = fromFHIR(questionnaireFHIR())
    expect(toRequest(v.title, '', v.items)).toEqual({
      title: 'Placeholder intake',
      item: questionnaireJSON().item,
    })
  })

  it('sends the description when there is one, and required only on questions', () => {
    const items = [{ ...newItem('display', 'd'), text: 'Placeholder', required: true }]
    expect(toRequest('T', 'Placeholder description', items)).toEqual({
      title: 'T',
      description: 'Placeholder description',
      item: [{ linkId: 'd', text: 'Placeholder', type: 'display' }],
    })
  })
})

describe('libraryOf', () => {
  const version = (id: string, series: string, n: number, status: Questionnaire['status'], title: string): Questionnaire => ({
    ...questionnaireFHIR(),
    id,
    url: `https://salus.invalid/fhir/Questionnaire/${series}`,
    version: String(n),
    status,
    title,
  })

  it("lists each questionnaire's draft and active version, or its latest retired one, by title", () => {
    const list = libraryOf([
      version('b1', 'b', 1, 'retired', 'Beta'),
      version('b2', 'b', 2, 'active', 'Beta'),
      version('b3', 'b', 3, 'draft', 'Beta'),
      version('a1', 'a', 1, 'retired', 'Alpha'),
      version('a2', 'a', 2, 'retired', 'Alpha'),
      version('c1', 'c', 1, 'draft', 'Gamma'),
    ])
    expect(list.map((v) => v.id)).toEqual(['a2', 'b2', 'b3', 'c1'])
  })
})
