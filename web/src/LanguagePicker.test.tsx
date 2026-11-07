import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import { loadSavedLanguage, saveLanguage } from './i18n/saved'
import { signInLanguages } from './language'
import { DeviceLanguagePicker, LanguagePicker } from './LanguagePicker'

afterEach(() => {
  cleanup()
  localStorage.clear()
})

describe('LanguagePicker', () => {
  it("offers the browser's language and each staff language by its own name", () => {
    renderInLanguage(<LanguagePicker value={null} onChange={() => {}} />, 'en')

    const picker = screen.getByRole('combobox', { name: 'Language' })
    expect(within(picker).getAllByRole('option').map((o) => o.textContent)).toEqual([
      "Use the browser's language",
      'Bosanski',
      'Hrvatski',
      'Srpski (latinica)',
      'Српски (ћирилица)',
      'English',
    ])
    expect((picker as HTMLSelectElement).value).toBe('')
  })

  it('is labelled in the UI language and shows the choice', () => {
    renderInLanguage(<LanguagePicker value="sr-Latn" onChange={() => {}} />, 'hr')

    const picker = screen.getByRole('combobox', { name: 'Jezik' }) as HTMLSelectElement
    expect(picker.value).toBe('sr-Latn')
    expect(within(picker).getAllByRole('option')[0].textContent).toBe('Koristi jezik preglednika')
  })

  it('gives the language picked, or null for the browser', () => {
    const onChange = vi.fn()
    renderInLanguage(<LanguagePicker value="hr" onChange={onChange} />, 'en')

    fireEvent.change(screen.getByRole('combobox', { name: 'Language' }), { target: { value: 'en' } })
    fireEvent.change(screen.getByRole('combobox', { name: 'Language' }), { target: { value: '' } })

    expect(onChange.mock.calls).toEqual([['en'], [null]])
  })
})

describe('DeviceLanguagePicker', () => {
  it('switches the UI and keeps the choice on the device', () => {
    renderInLanguage(<DeviceLanguagePicker />, 'en')

    fireEvent.change(screen.getByRole('combobox', { name: 'Language' }), { target: { value: 'bs' } })

    expect((screen.getByRole('combobox', { name: 'Jezik' }) as HTMLSelectElement).value).toBe('bs')
    expect(loadSavedLanguage()).toBe('bs')
  })

  it("clears the choice to follow the browser's language", () => {
    saveLanguage('hr')
    renderInLanguage(<DeviceLanguagePicker />, 'hr')

    fireEvent.change(screen.getByRole('combobox', { name: 'Jezik' }), { target: { value: '' } })

    expect(loadSavedLanguage()).toBeNull()
    expect((screen.getByRole('combobox', { name: 'Jezik' }) as HTMLSelectElement).value).toBe('')
  })
})

describe('signInLanguages', () => {
  it("sends the browser's language, and the choice kept on the device when there is one", () => {
    expect(signInLanguages(['sr-RS', 'en'])).toEqual({ detectedLanguage: 'sr-Cyrl' })
    saveLanguage('hr')
    expect(signInLanguages(['sr-RS', 'en'])).toEqual({ language: 'hr', detectedLanguage: 'sr-Cyrl' })
  })
})
