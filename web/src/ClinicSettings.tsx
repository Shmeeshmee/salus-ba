import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { FhirError, settingsOf, withSettings, type ClinicSettings as Settings, type FhirClient, type Organization } from './api/fhir'
import { attempt } from './api/result'
import { ClinicSettingsReminders } from './ClinicSettingsReminders'
import { isStaffLanguage, staffLanguages } from './i18n/languages'
import { useLanguage, useT } from './i18n/useT'
import { languageNames } from './language'
import { useLoad } from './load'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'
import type { components } from './api/schema'

type Load = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; org: Organization }
type T = ReturnType<typeof useT>
type Place = components['schemas']['Place']

// ClinicSettings reads the clinic's Organization and saves its name, time zone, default language, whether it is
// listed in the patient app, whether its patients get a copy of their transcripts, its location (a region and a
// city of GET /places) and whether it contributes to the All of Salus figures, with If-Match on the version read.
// platform is whether the All of Salus view is on (GET /auth/me features): off, the opt-out and the location hint,
// which speak of that view, are not shown (docs/plans/analytics.md §2a). The reminders section follows (SAL-33).
export function ClinicSettings({
  api,
  fhir,
  organizationId,
  platform,
}: {
  api: ApiClient
  fhir: FhirClient
  organizationId: string
  platform: boolean
}) {
  const t = useT()
  const { state: places } = useLoad(async () => {
    const res = await attempt(api.GET('/places'))
    if (!res.ok) throw new Error('places')
    return res.data.places
  }, [api])
  const [load, setLoad] = useState<Load>({ name: 'loading' })
  const [reads, setReads] = useState(0)

  useEffect(() => {
    let current = true
    fhir.get<Organization>(`Organization/${encodeURIComponent(organizationId)}`).then(
      (org) => current && setLoad({ name: 'loaded', org }),
      () => current && setLoad({ name: 'failed' }),
    )
    return () => {
      current = false
    }
  }, [fhir, organizationId, reads])

  function reload() {
    setLoad({ name: 'loading' })
    setReads((n) => n + 1)
  }

  return (
    <>
      <section className="panel">
        <h1>{t('admin.clinicSettings')}</h1>
        {load.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {load.name === 'failed' && <Alert message={t('admin.clinicLoadFailed')} />}
        {load.name === 'loaded' && places.name === 'failed' && <Alert message={t('admin.placesLoadFailed')} />}
        {load.name === 'loaded' && (
          <SettingsForm
            key={reads}
            org={load.org}
            places={places.name === 'loaded' ? places.value : []}
            platform={platform}
            save={(s) => fhir.put<Organization>(`Organization/${encodeURIComponent(load.org.id)}`, withSettings(load.org, s), load.org.meta.versionId)}
            onSaved={(org) => setLoad({ name: 'loaded', org })}
            onReload={reload}
          />
        )}
      </section>
      <ClinicSettingsReminders api={api} />
    </>
  )
}

type SettingsFormProps = {
  org: Organization
  places: Place[]
  platform: boolean
  save: (s: Settings) => Promise<Organization>
  onSaved: (org: Organization) => void
  onReload: () => void
}

function SettingsForm({ org, places, platform, save, onSaved, onReload }: SettingsFormProps) {
  const t = useT()
  const [settings, setSettings] = useState(settingsOf(org))
  const [saved, setSaved] = useState(false)
  const [conflict, setConflict] = useState(false)
  const located = settingsOf(org).city !== ''
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    setConflict(false)
    if (settings.region !== '' && settings.city === '') return t('admin.invalidLocation')
    try {
      onSaved(await save(settings))
      setSaved(true)
      return undefined
    } catch (err) {
      return saveMessage(t, err, setConflict)
    }
  })

  function change(next: Partial<Settings>) {
    setSettings((s) => ({ ...s, ...next }))
    setSaved(false)
  }

  return (
    <form onSubmit={onSubmit} noValidate>
      <Field label={t('admin.clinicName')} value={settings.name} maxLength={200} onChange={(e) => change({ name: e.target.value })} required />
      <div className="field">
        <label htmlFor="time-zone">{t('admin.timeZone')}</label>
        <select id="time-zone" value={settings.timeZone} onChange={(e) => change({ timeZone: e.target.value })} aria-describedby="time-zone-hint">
          {zones(settings.timeZone).map((z) => (
            <option key={z}>{z}</option>
          ))}
        </select>
        <p className="hint" id="time-zone-hint">
          {t('admin.timeZoneHint')}
        </p>
      </div>
      <div className="field">
        <label htmlFor="default-language">{t('admin.defaultLanguage')}</label>
        <select
          id="default-language"
          value={settings.defaultLanguage}
          onChange={(e) => isStaffLanguage(e.target.value) && change({ defaultLanguage: e.target.value })}
          aria-describedby="default-language-hint"
        >
          {staffLanguages.map((l) => (
            <option key={l} value={l} lang={l}>
              {languageNames[l]}
            </option>
          ))}
        </select>
        <p className="hint" id="default-language-hint">
          {t('admin.defaultLanguageHint')}
        </p>
      </div>
      <div className="check">
        <input id="listed" type="checkbox" checked={settings.listed} onChange={(e) => change({ listed: e.target.checked })} aria-describedby="listed-hint" />
        <div className="stack">
          <label htmlFor="listed">{t('admin.listed')}</label>
          <p className="hint" id="listed-hint">
            {t('admin.listedHint')}
          </p>
        </div>
      </div>
      <div className="check">
        <input
          id="transcript-copy"
          type="checkbox"
          checked={settings.transcriptCopy}
          onChange={(e) => change({ transcriptCopy: e.target.checked })}
          aria-describedby="transcript-copy-hint"
        />
        <div className="stack">
          <label htmlFor="transcript-copy">{t('admin.transcriptCopy')}</label>
          <p className="hint" id="transcript-copy-hint">
            {t('admin.transcriptCopyHint')}
          </p>
        </div>
      </div>
      <Location places={places} settings={settings} change={change} located={located} hint={platform} />
      {platform && (
        <div className="check">
          <input
            id="contributes"
            type="checkbox"
            checked={settings.contributes}
            onChange={(e) => change({ contributes: e.target.checked })}
            aria-describedby="contributes-hint"
          />
          <div className="stack">
            <label htmlFor="contributes">{t('admin.contributes')}</label>
            <p className="hint" id="contributes-hint">
              {t('admin.contributesHint')}
            </p>
          </div>
        </div>
      )}
      <Alert message={error} />
      {saved && <Notice message={t('admin.saved')} />}
      <div className="actions">
        <Submit busy={busy} label={t('common.save')} busyLabel={t('common.saving')} />
        {conflict && (
          <button type="button" onClick={onReload}>
            {t('common.reload')}
          </button>
        )}
      </div>
    </form>
  )
}

type LocationProps = {
  places: Place[]
  settings: Settings
  change: (next: Partial<Settings>) => void
  located: boolean
  hint: boolean
}

// Location is the clinic's region and a city of it, chosen from the places; a new region clears the city. A location
// once saved can be changed, not cleared; hint shows what the All of Salus view does with it.
function Location({ places, settings, change, located, hint }: LocationProps) {
  const t = useT()
  const name = useLanguage().language === 'sr-Cyrl' ? (p: Place) => p.nameCyrl : (p: Place) => p.nameLatn
  const byName = (a: Place, b: Place) => name(a).localeCompare(name(b))
  const regions = places.filter((p) => p.kind === 'region').sort(byName)
  const cities = places.filter((p) => p.kind === 'city' && p.parent === settings.region).sort(byName)
  return (
    <>
      <div className="field">
        <label htmlFor="region">{t('admin.region')}</label>
        <select
          id="region"
          value={settings.region}
          onChange={(e) => change({ region: e.target.value, city: '', cityName: '' })}
          aria-describedby={hint ? 'location-hint' : undefined}
        >
          <option value="" disabled={located}>
            {t('admin.chooseRegion')}
          </option>
          {regions.map((p) => (
            <option key={p.id} value={p.id}>
              {name(p)}
            </option>
          ))}
        </select>
        {hint && (
          <p className="hint" id="location-hint">
            {t('admin.locationHint')}
          </p>
        )}
      </div>
      <div className="field">
        <label htmlFor="city">{t('admin.city')}</label>
        <select
          id="city"
          value={settings.city}
          disabled={settings.region === ''}
          onChange={(e) => change({ city: e.target.value, cityName: places.find((p) => p.id === e.target.value)?.nameLatn ?? '' })}
        >
          <option value="">{t('admin.chooseCity')}</option>
          {cities.map((p) => (
            <option key={p.id} value={p.id}>
              {name(p)}
            </option>
          ))}
        </select>
      </div>
    </>
  )
}

function saveMessage(t: T, err: unknown, setConflict: (c: boolean) => void): string {
  if (!(err instanceof FhirError)) return t('common.somethingWrong')
  if (err.status === 412) {
    setConflict(true)
    return t('admin.clinicConflict')
  }
  if (err.status === 422) return t('admin.invalidClinicName')
  if (err.status === 400 && err.code === 'value') return t('admin.invalidTimeZone')
  if (err.status === 400 && err.code === 'code-invalid') return t('admin.invalidLocation')
  if (err.status === 403) return t('admin.clinicForbidden')
  return t('common.somethingWrong')
}

// zones is every IANA time zone the browser knows, with the clinic's own first in line when it is missing.
function zones(current: string): string[] {
  const all = Intl.supportedValuesOf('timeZone')
  return all.includes(current) ? all : [current, ...all]
}
