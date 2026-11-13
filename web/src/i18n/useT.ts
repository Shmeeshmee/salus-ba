import { createContext, useContext } from 'react'
import { useIntl } from 'react-intl'
import { formatCalendarDate, formatDate, formatDateTime, formatMonth, formatNumber, formatTime, formatWeekday } from './format'
import type { Language } from './languages'

export type LanguageContext = {
  language: Language
  // choose switches the UI to language and keeps it on the device; null follows the browser again.
  choose: (language: Language | null) => void
}

export const languageContext = createContext<LanguageContext | null>(null)

export function useLanguage(): LanguageContext {
  const value = useContext(languageContext)
  if (!value) throw new Error('useLanguage outside I18nProvider')
  return value
}

// useT returns t(id, values): the catalog message id ("<area>.<key>") formatted with ICU values.
export function useT() {
  const intl = useIntl()
  return (id: string, values?: Record<string, string | number | Date>) => intl.formatMessage({ id }, values)
}

// useFormat returns the date, time and number helpers in the UI language.
export function useFormat() {
  const { language } = useLanguage()
  return {
    time: (instant: Date, zone: string) => formatTime(instant, zone, language),
    date: (instant: Date, zone: string, style: 'long' | 'short') => formatDate(instant, zone, language, style),
    calendarDate: (date: string) => formatCalendarDate(date, language),
    dateTime: (instant: Date, zone: string) => formatDateTime(instant, zone, language),
    number: (n: number) => formatNumber(n, language),
    month: (month: string, style: 'short' | 'long') => formatMonth(month, language, style),
    weekday: (weekday: number) => formatWeekday(weekday, language),
  }
}
