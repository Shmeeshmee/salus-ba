import { describe, expect, it } from 'vitest'
import { fromFHIR, type Translation } from './questionnaire'
import { firstIncomplete, progress, unused, withItem } from './translation'
import { translatedFHIR } from './test/questionnaires'

const v = fromFHIR(translatedFHIR())
const [english, german] = v.translations

describe('progress', () => {
  // The title, and per item its text, its help text and its options: intro 1, smoker 2, sec 2, one 3, many 3.
  it('counts the texts translated of the texts the original has, and names the items that lack any', () => {
    expect(progress(english, v.description, v.items)).toEqual({ n: 9, total: 12, missing: ['many'] })
    expect(progress(german, v.description, v.items)).toEqual({ n: 1, total: 12, missing: ['intro', 'smoker', 'sec', 'one', 'many'] })
  })

  it('counts the description when the original has one, and a blank text as not translated', () => {
    const t: Translation = { ...english, title: ' ', items: { ...english.items, many: { text: 'Placeholder multiple in English', options: ['C in English', ''] } } }
    expect(progress(t, 'Placeholder description', v.items)).toEqual({ n: 10, total: 13, missing: ['many'] })
  })

  it('is complete when every text is translated', () => {
    const t = withItem(english, 'many', { text: 'Placeholder multiple in English', options: ['C in English', 'D in English'] })
    expect(progress(t, '', v.items)).toEqual({ n: 12, total: 12, missing: [] })
  })
})

describe('firstIncomplete', () => {
  it('names the first incomplete translation in the language list order, as the API does', () => {
    expect(firstIncomplete([german, english], '', v.items)).toEqual({ language: 'en', linkIds: ['many'] })
  })

  it('is none when every translation is complete', () => {
    const t = withItem(english, 'many', { text: 'Placeholder multiple in English', options: ['C in English', 'D in English'] })
    expect(firstIncomplete([t], '', v.items)).toBeUndefined()
  })
})

describe('unused', () => {
  it('offers the languages that are neither the original nor translated, in the list order', () => {
    expect(unused('bs', [german, english])).toEqual(['hr', 'sr-Latn', 'sr-Cyrl', 'ar', 'tr', 'it', 'fr'])
  })
})

describe('withItem', () => {
  it("changes one item's texts and keeps the rest", () => {
    const t = withItem(english, 'smoker', { help: 'Placeholder help changed' })
    expect(t.items.smoker).toEqual({ text: 'Placeholder yes or no in English', help: 'Placeholder help changed' })
    expect(english.items.smoker.help).toBe('Placeholder help in English')
  })
})
