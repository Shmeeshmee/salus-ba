import { describe, expect, it } from 'vitest'
import { translator } from '../test/translate'
import { areas, catalogs, messagesFor } from './catalog'
import { staffLanguages } from './languages'

describe('the catalogs', () => {
  it('have a file for every area and staff language', () => {
    for (const area of areas) for (const language of staffLanguages) expect(catalogs[area]?.[language], `${area}/${language}.json`).toBeDefined()
  })

  it('has the areas of the plan', () => {
    expect([...areas].sort()).toEqual(['admin', 'allergies', 'analytics', 'attachments', 'auth', 'calendar', 'chat', 'common', 'consultation', 'history', 'questionnaire', 'review', 'shell', 'terminology'])
  })
})

describe('messagesFor', () => {
  const fake = {
    shell: { en: { signOut: 'Sign out', settings: 'Settings' }, hr: { signOut: 'Odjava' } },
    common: { en: { save: 'Save' }, hr: { save: { defaultMessage: 'Spremi', description: 'legal' } } },
  }

  it('prefixes every key with its area', () => {
    expect(messagesFor('hr', fake)).toMatchObject({ 'shell.signOut': 'Odjava', 'common.save': 'Spremi' })
  })

  it('falls back to English for a key the language is missing', () => {
    expect(messagesFor('hr', fake)['shell.settings']).toBe('Settings')
  })

  it('reads a message tagged with a description', () => {
    expect(messagesFor('hr', fake)['common.save']).toBe('Spremi')
  })
})

// W19 (docs/i18n/SPEC.md, owner's answer of 2026-11-08): an ICU plural in every web language, the singular for 1.
describe('questionnaire.dependents', () => {
  it('is singular for one item in English', () => {
    const t = translator('en')
    expect(t('questionnaire.dependents', { n: 1 })).toBe('1 item is shown only after an answer to it. Its condition will be removed.')
    expect(t('questionnaire.dependents', { n: 2 })).toBe('2 items are shown only after an answer to it. Their conditions will be removed.')
  })

  it.each(staffLanguages)('has a singular for one item in %s', (language) => {
    const t = translator(language)
    expect(t('questionnaire.dependents', { n: 1 })).toMatch(/^1 /)
    expect(t('questionnaire.dependents', { n: 1 })).not.toBe(t('questionnaire.dependents', { n: 5 }).replace(/^5 /, '1 '))
  })
})

// T12 (docs/terminology/COPY.md, owner's answer of 2026-11-08): the palette's footer says not every term is reviewed.
describe('terminology.source', () => {
  it.each([
    ['bs', 'Pojmovi su preuzeti iz Wikidata. Nisu još svi pojmovi provjereni.'],
    ['hr', 'Pojmovi su preuzeti iz Wikidata. Nisu još svi pojmovi provjereni.'],
    ['sr-Latn', 'Pojmovi su preuzeti iz Wikidata. Nisu još svi pojmovi provjereni.'],
    ['sr-Cyrl', 'Појмови су преузети из Wikidata. Нису још сви појмови провјерени.'],
    ['en', 'Terms from Wikidata. Not every term has been reviewed yet.'],
  ] as const)('is the approved text in %s', (language, text) => {
    expect(messagesFor(language)['terminology.source']).toBe(text)
  })
})
