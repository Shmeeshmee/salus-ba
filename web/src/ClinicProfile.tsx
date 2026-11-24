import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { contentLanguageNames, contentLanguages, type ContentLanguage } from './questionnaire'
import { Alert, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Profile = components['schemas']['ClinicProfile']
type Loaded = { profile: Profile; version: string }

// ClinicProfile is the admin's profile of the clinic for patients abroad (docs/clinic-cms, package X2): the languages
// its staff speak, whether it welcomes patients from abroad, and a short description in the clinic's default language
// with its translations (behind a disclosure; an empty one is left out). Saved whole with the version read
// (PUT /clinic/profile, If-Match).
export function ClinicProfile({ api }: { api: ApiClient }) {
  const t = useT()
  const { state, set } = useLoad(async (): Promise<Loaded> => {
    const res = await attempt(api.GET('/clinic/profile'))
    if (!res.ok) throw new Error('profile')
    return { profile: res.data, version: res.response.headers.get('ETag') ?? '' }
  }, [api])
  return (
    <section className="panel" id="clinic-profile">
      <h2>{t('clinicProfile.title')}</h2>
      <p className="hint">{t('clinicProfile.intro')}</p>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('clinicProfile.loadFailed')} />}
      {state.name === 'loaded' && <ProfileForm api={api} loaded={state.value} onSaved={set} />}
    </section>
  )
}

function ProfileForm({ api, loaded, onSaved }: { api: ApiClient; loaded: Loaded; onSaved: (l: Loaded) => void }) {
  const t = useT()
  const { profile } = loaded
  const [languages, setLanguages] = useState<string[]>(profile.languages)
  const [welcomes, setWelcomes] = useState(profile.welcomesInternational)
  const [description, setDescription] = useState(profile.description)
  const [translations, setTranslations] = useState<Partial<Record<ContentLanguage, string>>>(
    Object.fromEntries(profile.translations.map((x) => [x.language, x.description])),
  )
  const [saved, setSaved] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    const filled = contentLanguages
      .map((language) => ({ language, description: (translations[language] ?? '').trim() }))
      .filter((x) => x.description !== '')
    if (description.trim() === '' && filled.length > 0) return t('clinicProfile.translationsNeedDescription')
    const body: Profile = {
      languages: contentLanguages.filter((l) => languages.includes(l)),
      description: description.trim(),
      translations: filled,
      welcomesInternational: welcomes,
    }
    const res = await attempt(api.PUT('/clinic/profile', { params: { header: { 'If-Match': loaded.version } }, body }))
    if (!res.ok && res.code === 'version_conflict') return t('clinicProfile.conflict')
    if (!res.ok && res.code === 'invalid_profile') return t('clinicProfile.invalid')
    if (!res.ok) return t('clinicProfile.saveFailed')
    onSaved({ profile: res.data, version: res.response.headers.get('ETag') ?? '' })
    setSaved(true)
    return undefined
  })
  const toggle = (language: string, on: boolean) => setLanguages((ls) => (on ? [...ls, language] : ls.filter((l) => l !== language)))
  return (
    <form onSubmit={onSubmit} noValidate>
      <fieldset>
        <legend>{t('clinicProfile.languages')}</legend>
        {contentLanguages.map((language) => (
          <label key={language} className="check">
            <input type="checkbox" checked={languages.includes(language)} onChange={(e) => toggle(language, e.target.checked)} />
            <span lang={language}>{contentLanguageNames[language]}</span>
          </label>
        ))}
      </fieldset>
      <label className="check">
        <input type="checkbox" checked={welcomes} onChange={(e) => setWelcomes(e.target.checked)} />
        <span>{t('clinicProfile.welcomes')}</span>
      </label>
      <div className="field">
        <label htmlFor="clinic-profile-description">{t('clinicProfile.description')}</label>
        <textarea
          id="clinic-profile-description"
          aria-describedby="clinic-profile-description-hint"
          maxLength={2000}
          rows={4}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <p className="hint" id="clinic-profile-description-hint">
          {t('clinicProfile.descriptionHint')}
        </p>
      </div>
      <details>
        <summary>{t('clinicProfile.translations')}</summary>
        {contentLanguages.map((language) => (
          <label key={language} className="field">
            {t('clinicProfile.descriptionIn', { language: contentLanguageNames[language] })}
            <textarea
              lang={language}
              dir={language === 'ar' ? 'rtl' : undefined}
              maxLength={2000}
              rows={3}
              value={translations[language] ?? ''}
              onChange={(e) => setTranslations((ts) => ({ ...ts, [language]: e.target.value }))}
            />
          </label>
        ))}
      </details>
      <Alert message={error} />
      {saved && <Notice message={t('clinicProfile.saved')} />}
      <Submit busy={busy} label={t('clinicProfile.save')} busyLabel={t('clinicProfile.saving')} />
    </form>
  )
}
