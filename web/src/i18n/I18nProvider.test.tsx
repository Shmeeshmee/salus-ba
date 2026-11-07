import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from './I18nProvider'
import { loadSavedLanguage } from './saved'
import { useFormat, useLanguage, useT } from './useT'
import { renderInLanguage } from './renderInLanguage'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  localStorage.clear()
  document.documentElement.removeAttribute('lang')
})

const messages = {
  shell: {
    en: { greeting: 'Hello', only: 'Only in English', count: '{n, plural, one {# appointment} other {# appointments}}' },
    hr: { greeting: 'Bok', count: '{n, plural, one {# termin} few {# termina} other {# termina}}' },
  },
}

function Probe() {
  const t = useT()
  const { language, choose } = useLanguage()
  const f = useFormat()
  return (
    <div>
      <p>{t('shell.greeting')}</p>
      <p>{t('shell.only')}</p>
      <p>{t('shell.count', { n: 3 })}</p>
      <p>{f.time(new Date('2026-11-09T14:05:00Z'), 'Europe/Sarajevo')}</p>
      <p>language {language}</p>
      <button onClick={() => choose('en')}>English</button>
      <button onClick={() => choose(null)}>Browser</button>
    </div>
  )
}

describe('I18nProvider', () => {
  it('renders in the language it starts with and sets the page language', () => {
    render(
      <I18nProvider initial="hr" browser={['en']} catalogs={messages}>
        <Probe />
      </I18nProvider>,
    )
    expect(screen.getByText('Bok')).toBeTruthy()
    expect(screen.getByText('3 termina')).toBeTruthy()
    expect(screen.getByText('15:05')).toBeTruthy()
    expect(document.documentElement.lang).toBe('hr')
  })

  it('shows English for a key the language is missing, and logs nothing', () => {
    const error = vi.spyOn(console, 'error')
    const warn = vi.spyOn(console, 'warn')
    render(
      <I18nProvider initial="hr" browser={['en']} catalogs={messages}>
        <Probe />
      </I18nProvider>,
    )
    expect(screen.getByText('Only in English')).toBeTruthy()
    expect(error).not.toHaveBeenCalled()
    expect(warn).not.toHaveBeenCalled()
  })

  it('switches to a chosen language and keeps it on the device', () => {
    render(
      <I18nProvider initial="hr" browser={['hr']} catalogs={messages}>
        <Probe />
      </I18nProvider>,
    )
    act(() => screen.getByText('English').click())
    expect(screen.getByText('Hello')).toBeTruthy()
    expect(document.documentElement.lang).toBe('en')
    expect(loadSavedLanguage()).toBe('en')
  })

  it("goes back to the browser's language when the choice is cleared", () => {
    render(
      <I18nProvider initial="en" browser={['hr-HR']} catalogs={messages}>
        <Probe />
      </I18nProvider>,
    )
    act(() => screen.getByText('Browser').click())
    expect(screen.getByText('Bok')).toBeTruthy()
    expect(loadSavedLanguage()).toBeNull()
  })
})

describe('renderInLanguage', () => {
  it('renders a component in the language with the real catalogs', () => {
    renderInLanguage(<Probe />, 'sr-Cyrl')
    expect(screen.getByText('language sr-Cyrl')).toBeTruthy()
    expect(document.documentElement.lang).toBe('sr-Cyrl')
  })
})
