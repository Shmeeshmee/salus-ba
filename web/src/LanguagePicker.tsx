import { useId, useState } from 'react'
import { isStaffLanguage, staffLanguages, type Language } from './i18n/languages'
import { loadSavedLanguage } from './i18n/saved'
import { useLanguage, useT } from './i18n/useT'
import { languageNames } from './language'

type Props = { value: Language | null; disabled?: boolean; onChange: (language: Language | null) => void }

// LanguagePicker chooses a staff language, or none to follow the browser's (COPY.md L1 to L3).
export function LanguagePicker({ value, disabled, onChange }: Props) {
  const t = useT()
  const id = useId()
  return (
    <div className="field language">
      <label htmlFor={id}>{t('shell.language')}</label>
      <select id={id} value={value ?? ''} disabled={disabled} onChange={(e) => onChange(isStaffLanguage(e.target.value) ? e.target.value : null)}>
        <option value="">{t('shell.browserLanguage')}</option>
        {staffLanguages.map((l) => (
          <option key={l} value={l} lang={l}>
            {languageNames[l]}
          </option>
        ))}
      </select>
    </div>
  )
}

// DeviceLanguagePicker is the picker before sign-in: the choice switches the UI and is kept on the device until
// the next sign-in sends it (signInLanguages).
export function DeviceLanguagePicker() {
  const { choose } = useLanguage()
  const [value, setValue] = useState(loadSavedLanguage)
  return (
    <LanguagePicker
      value={value}
      onChange={(language) => {
        setValue(language)
        choose(language)
      }}
    />
  )
}
