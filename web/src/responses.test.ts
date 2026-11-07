import { describe, expect, it } from 'vitest'
import type { Bundle, Questionnaire, QuestionnaireResponse } from './api/fhir'
import { answered, requestsOf, statusOf, valueText } from './responses'
import { newItem } from './builder'
import { fromFHIR } from './questionnaire'
import { questionnaireFHIR } from './test/questionnaires'
import { responseFHIR, responsesBundle } from './test/responses'
import { translator } from './test/translate'

const version = { ...questionnaireFHIR(), status: 'active' as const }
const items = fromFHIR(version).items
const zone = 'Europe/Sarajevo'

describe('answered', () => {
  it('lists the questions by section, with their answers, leaving instructions out', () => {
    const blocks = answered(items, [
      { linkId: 'smoker', answer: [{ valueBoolean: true }] },
      { linkId: 'sec', item: [{ linkId: 'one', answer: [{ valueString: 'B' }] }, { linkId: 'many', answer: [{ valueString: 'C' }, { valueString: 'D' }] }] },
    ])

    expect(blocks).toEqual([
      { questions: [{ linkId: 'smoker', text: 'Placeholder yes or no', kind: 'boolean', values: [{ valueBoolean: true }] }] },
      {
        section: 'Placeholder section',
        questions: [
          { linkId: 'one', text: 'Placeholder single', kind: 'single', values: [{ valueString: 'B' }] },
          { linkId: 'many', text: 'Placeholder multiple', kind: 'multiple', values: [{ valueString: 'C' }, { valueString: 'D' }] },
        ],
      },
    ])
  })

  it('keeps an enabled question without an answer, with no values', () => {
    const blocks = answered(items, [{ linkId: 'smoker', answer: [{ valueBoolean: true }] }, { linkId: 'sec', item: [{ linkId: 'one', answer: [{ valueString: 'A' }] }] }])

    expect(blocks[1].questions.map((q) => [q.linkId, q.values])).toEqual([['one', [{ valueString: 'A' }]]])
  })

  it('leaves out a hidden section and every item in it', () => {
    expect(answered(items, [{ linkId: 'smoker', answer: [{ valueBoolean: false }] }]).map((b) => b.section)).toEqual([undefined])
    expect(answered(items, []).flatMap((b) => b.questions.map((q) => q.linkId))).toEqual(['smoker'])
  })

  it('leaves out a question whose condition depends on a hidden question, along a chain of two', () => {
    const chain = [
      { ...newItem('boolean', 'a'), text: 'Placeholder a' },
      { ...newItem('single', 'b'), text: 'Placeholder b', options: ['A', 'B'], condition: { question: 'a', answer: true } },
      { ...newItem('string', 'c'), text: 'Placeholder c', condition: { question: 'b', answer: 'B' } },
    ]
    const shown = (a: boolean) =>
      answered(chain, [
        { linkId: 'a', answer: [{ valueBoolean: a }] },
        { linkId: 'b', answer: [{ valueString: 'B' }] },
      ]).flatMap((b) => b.questions.map((q) => q.linkId))

    expect(shown(true)).toEqual(['a', 'b', 'c'])
    expect(shown(false)).toEqual(['a'])
  })

  it('matches a condition on a choice with repeats when any answer matches', () => {
    const multi = [
      { ...newItem('multiple', 'm'), text: 'Placeholder m', options: ['C', 'D'] },
      { ...newItem('date', 'after'), text: 'Placeholder after', condition: { question: 'm', answer: 'D' } },
    ]
    const ids = (values: string[]) =>
      answered(multi, [{ linkId: 'm', answer: values.map((valueString) => ({ valueString })) }]).flatMap((b) => b.questions.map((q) => q.linkId))

    expect(ids(['C', 'D'])).toEqual(['m', 'after'])
    expect(ids(['C'])).toEqual(['m'])
  })
})

describe('valueText', () => {
  it.each([
    ['string', { valueString: 'Placeholder answer' }, 'Placeholder answer'],
    ['text', { valueString: 'Line one\nLine two' }, 'Line one\nLine two'],
    ['boolean', { valueBoolean: true }, 'Yes'],
    ['boolean', { valueBoolean: false }, 'No'],
    ['single', { valueString: 'B' }, 'B'],
    ['integer', { valueInteger: -12 }, '-12'],
    ['decimal', { valueDecimal: 36.6 }, '36.6'],
    ['integer', { valueInteger: 0 }, '0'],
    ['date', { valueDate: '1990-01-31' }, '31 January 1990'],
  ] as const)('writes a %s answer', (_, value, text) => {
    expect(valueText(value, translator('en'), 'en')).toBe(text)
  })

  it.each([
    [{ valueBoolean: true }, 'Da'],
    [{ valueDecimal: 36.6 }, '36,6'],
    [{ valueDate: '1990-01-31' }, '31. januar 1990.'],
  ] as const)('writes %o in the UI language', (value, text) => {
    expect(valueText(value, translator('bs'), 'bs')).toBe(text)
  })
})

describe('requestsOf', () => {
  it('pairs each response with the version it answers, and its answers once submitted', () => {
    const other: Questionnaire = { ...version, id: 'q2', url: 'https://salus.invalid/fhir/Questionnaire/s2', title: 'Placeholder follow-up' }
    const bundle: Bundle<QuestionnaireResponse | Questionnaire> = responsesBundle(
      [
        responseFHIR('r1', 'completed', {
          authored: '2026-11-08T17:45:00Z',
          item: [{ linkId: 'smoker', answer: [{ valueBoolean: false }] }],
        }),
        responseFHIR('r2', 'in-progress', { questionnaire: `${other.url}|1` }),
      ],
      [version, other],
    )

    expect(requestsOf(bundle)).toEqual([
      {
        id: 'r1',
        url: version.url,
        title: 'Placeholder intake',
        status: 'completed',
        authored: '2026-11-08T17:45:00Z',
        answers: [{ questions: [{ linkId: 'smoker', text: 'Placeholder yes or no', kind: 'boolean', values: [{ valueBoolean: false }] }] }],
      },
      { id: 'r2', url: other.url, title: 'Placeholder follow-up', status: 'in-progress', authored: undefined, answers: [] },
    ])
  })

  it('lists the enabled questions of a submitted response without answers as unanswered', () => {
    const [r] = requestsOf(responsesBundle([responseFHIR('r1', 'completed', { authored: '2026-11-08T17:45:00Z' })], [version]))

    expect(r.answers).toEqual([{ questions: [{ linkId: 'smoker', text: 'Placeholder yes or no', kind: 'boolean', values: [] }] }])
  })

  it('reads an empty search', () => {
    expect(requestsOf({ resourceType: 'Bundle', type: 'searchset' })).toEqual([])
  })
})

describe('statusOf', () => {
  const start = new Date('2026-11-09T08:00:00Z')
  const before = new Date('2026-11-09T07:59:59Z')

  it.each([
    ['in-progress', before, 'Not submitted yet'],
    ['in-progress', start, 'Not submitted before the appointment'],
    ['stopped', before, 'Withdrawn'],
    ['stopped', start, 'Withdrawn'],
  ] as const)('%s at %s is %s', (status, now, text) => {
    expect(statusOf({ status }, start, now, zone, translator('en'), 'en')).toBe(text)
  })

  it('gives the submitted date and time in the clinic zone', () => {
    expect(statusOf({ status: 'completed', authored: '2026-11-08T17:45:00Z' }, start, before, zone, translator('en'), 'en')).toBe(
      'Submitted Sunday, 8 November 2026, 18:45',
    )
  })

  it('writes the status in the UI language', () => {
    expect(statusOf({ status: 'completed', authored: '2026-11-08T17:45:00Z' }, start, before, zone, translator('hr'), 'hr')).toBe(
      'Predan: nedjelja, 8. studenoga 2026. u 18:45',
    )
    expect(statusOf({ status: 'stopped' }, start, before, zone, translator('sr-Cyrl'), 'sr-Cyrl')).toBe('Повучен')
  })
})
