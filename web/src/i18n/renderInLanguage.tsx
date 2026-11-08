import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { I18nProvider } from './I18nProvider'
import type { Language } from './languages'

// renderInLanguage renders ui in language with the real catalogs, as a test of a screen in that language.
export function renderInLanguage(ui: ReactNode, language: Language) {
  return render(
    <I18nProvider initial={language} browser={[language]}>
      {ui}
    </I18nProvider>,
  )
}
