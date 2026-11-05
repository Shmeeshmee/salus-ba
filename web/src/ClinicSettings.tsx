import { useEffect, useState } from 'react'
import { FhirError, settingsOf, withSettings, type ClinicSettings as Settings, type FhirClient, type Organization } from './api/fhir'
import { somethingWrong } from './messages'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Load = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; org: Organization }

// ClinicSettings reads the clinic's Organization and saves its name, time zone and whether it is listed in the
// patient app, with If-Match on the version read.
export function ClinicSettings({ fhir, organizationId }: { fhir: FhirClient; organizationId: string }) {
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
      <h1>Clinic settings</h1>
      {load.name === 'loading' && <p className="muted">Loading</p>}
      {load.name === 'failed' && <Alert message="The clinic could not be loaded. Reload the page to try again." />}
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
      return saveMessage(err, setConflict)
    }
  })

  function change(next: Partial<Settings>) {
    setSettings((s) => ({ ...s, ...next }))
    setSaved(false)
  }

  return (
    <form onSubmit={onSubmit} noValidate>
      <Field label="Clinic name" value={settings.name} maxLength={200} onChange={(e) => change({ name: e.target.value })} required />
      <div className="field">
        <label htmlFor="time-zone">Time zone</label>
        <select id="time-zone" value={settings.timeZone} onChange={(e) => change({ timeZone: e.target.value })} aria-describedby="time-zone-hint">
          {zones(settings.timeZone).map((z) => (
            <option key={z}>{z}</option>
          ))}
        </select>
        <p className="hint" id="time-zone-hint">
          Appointments and working hours are shown in this time zone.
        </p>
      </div>
      <div className="check">
        <input id="listed" type="checkbox" checked={settings.listed} onChange={(e) => change({ listed: e.target.checked })} aria-describedby="listed-hint" />
        <div className="stack">
          <label htmlFor="listed">List the clinic in the patient app</label>
          <p className="hint" id="listed-hint">
            Patients can then find the clinic, its practitioners and their free times, and book.
          </p>
        </div>
      </div>
      <Alert message={error} />
      {saved && <Notice message="Saved." />}
      <div className="actions">
        <Submit busy={busy} label="Save" busyLabel="Saving" />
        {conflict && (
          <button type="button" onClick={onReload}>
            Reload
          </button>
        )}
      </div>
    </form>
  )
}

function saveMessage(err: unknown, setConflict: (c: boolean) => void): string {
  if (!(err instanceof FhirError)) return somethingWrong
  if (err.status === 412) {
    setConflict(true)
    return 'Someone else changed the clinic settings. Reload to see their changes.'
  }
  if (err.status === 422) return 'Enter a name of 1 to 200 characters.'
  if (err.status === 400 && err.code === 'value') return 'Choose a time zone from the list.'
  if (err.status === 403) return 'Only a clinic admin can change these settings.'
  return somethingWrong
}

// zones is every IANA time zone the browser knows, with the clinic's own first in line when it is missing.
function zones(current: string): string[] {
  const all = Intl.supportedValuesOf('timeZone')
  return all.includes(current) ? all : [current, ...all]
}
