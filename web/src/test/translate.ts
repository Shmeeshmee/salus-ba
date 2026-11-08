import { createIntl } from 'react-intl'
import { messagesFor } from '../i18n/catalog'
import { intlLocale } from '../i18n/format'
import type { Language } from '../i18n/languages'
import type { useT } from '../i18n/useT'

// translator is what useT returns, outside React: the real catalogs in language, for tests of the functions
// that take a translator (builder.ts, responses.ts, attachment.ts, consultation.ts).
export function translator(language: Language = 'en'): ReturnType<typeof useT> {
  const intl = createIntl({ locale: intlLocale(language), defaultLocale: 'en', messages: messagesFor(language), onError: () => {} })
  return (id, values) => intl.formatMessage({ id }, values)
}
