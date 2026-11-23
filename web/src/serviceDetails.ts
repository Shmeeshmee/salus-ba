import type { components } from './api/schema'
import { formatNumber } from './i18n/format'
import type { Language } from './i18n/languages'
import type { useT } from './i18n/useT'

type T = ReturnType<typeof useT>
type Shown = Pick<components['schemas']['Service'], 'priceMinor' | 'currency' | 'durationMinutes' | 'vatRateBp' | 'modes'>

export const modeLabel = { 'in-person': 'prices.modeInPerson', video: 'prices.modeVideo' } as const

// details is a service's line under its name: the price, the duration when it has its own, the VAT and the modes.
export function details(t: T, language: Language, s: Shown): string {
  const parts = [formatNumber(s.priceMinor / 100, language, { style: 'currency', currency: s.currency })]
  if (s.durationMinutes !== undefined) parts.push(t('prices.minutes', { count: s.durationMinutes }))
  parts.push(
    s.vatRateBp === undefined
      ? t('prices.exempt')
      : t('prices.vatShown', { rate: formatNumber(s.vatRateBp / 10000, language, { style: 'percent', maximumFractionDigits: 2 }) }),
  )
  for (const m of s.modes) parts.push(t(modeLabel[m]))
  return parts.join(' · ')
}

