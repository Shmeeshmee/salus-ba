import { afterEach, describe, expect, it, vi } from 'vitest'
import { loadSavedLanguage, saveLanguage, startLanguage } from './saved'

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('the language saved on the device', () => {
  it('is empty until one is saved', () => {
    expect(loadSavedLanguage()).toBeNull()
  })

  it('keeps a saved language', () => {
    saveLanguage('sr-Cyrl')
    expect(loadSavedLanguage()).toBe('sr-Cyrl')
  })

  it('is cleared by saving no language', () => {
    saveLanguage('hr')
    saveLanguage(null)
    expect(loadSavedLanguage()).toBeNull()
  })

  it('ignores a value that is not a staff language', () => {
    saveLanguage('hr')
    localStorage.setItem(localStorage.key(0)!, 'de')
    expect(loadSavedLanguage()).toBeNull()
  })

  it('works without storage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(() => saveLanguage('hr')).not.toThrow()
    expect(() => saveLanguage(null)).not.toThrow()
    expect(loadSavedLanguage()).toBeNull()
  })
})

describe('startLanguage', () => {
  it("is the browser's language when none is saved", () => {
    expect(startLanguage(['sr-RS', 'en'])).toBe('sr-Cyrl')
  })

  it('is the saved language over the browser', () => {
    saveLanguage('hr')
    expect(startLanguage(['sr-RS', 'en'])).toBe('hr')
  })
})
