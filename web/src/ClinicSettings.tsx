import { useEffect, useState } from 'react'
import { FhirError, settingsOf, withSettings, type ClinicSettings as Settings, type FhirClient, type Organization } from './api/fhir'
import { isStaffLanguage, staffLanguages } from './i18n/languages'
import { useT } from './i18n/useT'
import { languageNames } from './language'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Load = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; org: Organization }
type T = ReturnType<typeof useT>

// ClinicSettings reads the clinic's Organization and saves its name, time zone, default language and whether it
// is listed in the patient app, with If-Match on the version read.
export function ClinicSettings({ fhir, organizationId }: { fhir: FhirClient; organizationId: string }) {
  const t = useT()
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
    <section className="panel">
      <h1>{t('admin.clinicSettings')}</h1>
      {load.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {load.name === 'failed' && <Alert message={t('admin.clinicLoadFailed')} />}
      {load.name === 'loaded' && (
        <SettingsForm
          key={reads}
          org={load.org}
          save={(s) => fhir.put<Organization>(`Organization/${encodeURIComponent(load.org.id)}`, withSettings(load.org, s), load.org.meta.versionId)}
          onSaved={(org) => setLoad({ name: 'loaded', org })}
          onReload={reload}
        />
      )}
    </section>
  )
}

type SettingsFormProps = {
  org: Organization
  save: (s: Settings) => Promise<Organization>
  onSaved: (org: Organization) => void
  onReload: () => void
}

function SettingsForm({ org, save, onSaved, onReload }: SettingsFormProps) {
  const t = useT()
  const [settings, setSettings] = useState(settingsOf(org))
  const [saved, setSaved] = useState(false)
  const [conflict, setConflict] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    setConflict(false)
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

function saveMessage(t: T, err: unknown, setConflict: (c: boolean) => void): string {
  if (!(err instanceof FhirError)) return t('common.somethingWrong')
  if (err.status === 412) {
    setConflict(true)
    return t('admin.clinicConflict')
  }
  if (err.status === 422) return t('admin.invalidClinicName')
  if (err.status === 400 && err.code === 'value') return t('admin.invalidTimeZone')
  if (err.status === 403) return t('admin.clinicForbidden')
  return t('common.somethingWrong')
}

// zones is every IANA time zone the browser knows, with the clinic's own first in line when it is missing.
function zones(current: string): string[] {
  const all = Intl.supportedValuesOf('timeZone')
  return all.includes(current) ? all : [current, ...all]
}
