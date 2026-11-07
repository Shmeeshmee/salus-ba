import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { IntlProvider } from 'react-intl'
import { catalogs as allCatalogs, messagesFor, type Catalogs } from './catalog'
import { intlLocale } from './format'
import { matchLanguage, type Language } from './languages'
import { saveLanguage } from './saved'
import { languageContext } from './useT'

type Props = {
  // initial is the language to start in: the saved choice, else the browser's (startLanguage).
  initial: Language
  // browser is the browser's languages, in its order (navigator.languages).
  browser: readonly string[]
  catalogs?: Catalogs
  children: ReactNode
}

// A message missing in the language falls back to English (messagesFor), so react-intl's reports would only
// repeat what scripts/i18n-check.sh fails on; nothing is logged.
function ignore() {}

export function I18nProvider({ initial, browser, catalogs = allCatalogs, children }: Props) {
  const [language, setLanguage] = useState(initial)
  const messages = useMemo(() => messagesFor(language, catalogs), [language, catalogs])
  const context = useMemo(
    () => ({
      language,
      choose: (chosen: Language | null) => {
        saveLanguage(chosen)
        setLanguage(chosen ?? matchLanguage(browser))
      },
    }),
    [language, browser],
  )

  useEffect(() => {
    document.documentElement.lang = language
  }, [language])

  return (
    <languageContext.Provider value={context}>
      <IntlProvider locale={intlLocale(language)} defaultLocale="en" messages={messages} onError={ignore}>
        {children}
      </IntlProvider>
    </languageContext.Provider>
  )
}
