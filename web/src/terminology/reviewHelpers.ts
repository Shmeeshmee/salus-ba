import type { KeyboardEvent } from 'react'
import type { components } from '../api/schema'
import { useT } from '../i18n/useT'

type TermLanguage = components['schemas']['TermLanguage']

// The helpers the review panel's two lists share (reviewParts.tsx has the components).

// useLanguageName names a term language in the UI language.
export function useLanguageName() {
  const t = useT()
  return (l: TermLanguage) => {
    if (l === 'de' || l === 'ar' || l === 'tr' || l === 'it' || l === 'fr') return t(`review.language.${l}`)
    return t(`terminology.language.${l.replace('-', '')}`)
  }
}

// escapeCancels makes Escape in a form cancel it, without reaching the entry's shortcuts.
export function escapeCancels(onCancel: () => void) {
  return (event: KeyboardEvent<HTMLFormElement>) => {
    if (event.key !== 'Escape') return
    event.preventDefault()
    event.stopPropagation()
    onCancel()
  }
}
