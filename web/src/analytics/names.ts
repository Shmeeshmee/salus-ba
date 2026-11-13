import { useIntl } from 'react-intl'
import { useLanguage } from '../i18n/useT'
import type { Place, Specialty } from './data'

// usePlaceName names a place in the UI language's script: Cyrillic for sr-Cyrl, Latin otherwise.
export function usePlaceName(places: Place[]) {
  const { language } = useLanguage()
  const byId = new Map(places.map((p) => [p.id, p]))
  return (id: string) => {
    const p = byId.get(id)
    if (!p) return id
    return language === 'sr-Cyrl' ? p.nameCyrl : p.nameLatn
  }
}

// useSpecialtyName names a specialty from the catalog by its SNOMED CT code, the API's text for any other code.
export function useSpecialtyName(specialties: Specialty[]) {
  const intl = useIntl()
  return (code: string) => {
    const id = `common.specialty.${code}`
    if (id in intl.messages) return intl.formatMessage({ id })
    return specialties.find((s) => s.code === code)?.text ?? code
  }
}

