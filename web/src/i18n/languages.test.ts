import { describe, expect, it } from 'vitest'
import { matchLanguage, staffLanguages } from './languages'

describe('matchLanguage', () => {
  it.each([
    [['bs-BA'], 'bs'],
    [['bs'], 'bs'],
    [['hr-HR'], 'hr'],
    [['sr-Latn-BA'], 'sr-Latn'],
    [['sr-ME'], 'sr-Latn'],
    [['sh'], 'sr-Latn'],
    [['cnr'], 'sr-Latn'],
    [['sr'], 'sr-Cyrl'],
    [['sr-Cyrl-RS'], 'sr-Cyrl'],
    [['sr-RS'], 'sr-Cyrl'],
    [['sr-BA'], 'sr-Cyrl'],
    [['en-US'], 'en'],
    [['EN-gb'], 'en'],
  ] as const)('matches %j to %s', (browser, want) => {
    expect(matchLanguage(browser)).toBe(want)
  })

  it('takes the first language the browser lists that matches', () => {
    expect(matchLanguage(['fr-FR', 'en-GB', 'hr'])).toBe('en')
    expect(matchLanguage(['hr', 'en'])).toBe('hr')
  })

  it('skips patient languages the web app does not offer', () => {
    expect(matchLanguage(['de-DE', 'hr'])).toBe('hr')
  })

  it('falls back to Bosnian when nothing matches', () => {
    expect(matchLanguage(['ja-JP', 'de'])).toBe('bs')
    expect(matchLanguage([])).toBe('bs')
  })

  it('does not take a language that only starts with the same letters', () => {
    expect(matchLanguage(['bsx', 'srx', 'eng'])).toBe('bs')
  })

  it('offers the five staff languages', () => {
    expect(staffLanguages).toEqual(['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'])
  })
})
