import { describe, expect, it } from 'vitest'
import { areas, catalogs, messagesFor } from './catalog'
import { staffLanguages } from './languages'

describe('the catalogs', () => {
  it('have a file for every area and staff language', () => {
    for (const area of areas) for (const language of staffLanguages) expect(catalogs[area]?.[language], `${area}/${language}.json`).toBeDefined()
  })

  it('has the areas of the plan', () => {
    expect([...areas].sort()).toEqual(['admin', 'attachments', 'auth', 'calendar', 'common', 'consultation', 'questionnaire', 'shell'])
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
