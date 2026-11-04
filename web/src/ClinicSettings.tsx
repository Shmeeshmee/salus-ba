import { useEffect, useState } from 'react'
import { FhirError, type FhirClient, type Organization } from './api/fhir'
import { somethingWrong } from './messages'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Load = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; org: Organization }

// ClinicSettings reads the clinic's Organization and renames it with If-Match on the version read.
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
        <NameForm
          key={reads}
          org={load.org}
          save={(name) => fhir.put<Organization>(`Organization/${encodeURIComponent(load.org.id)}`, { ...load.org, name }, load.org.meta.versionId)}
          onSaved={(org) => setLoad({ name: 'loaded', org })}
          onReload={reload}
        />
      )}
    </section>
  )
}

type NameFormProps = {
  org: Organization
  save: (name: string) => Promise<Organization>
  onSaved: (org: Organization) => void
  onReload: () => void
}

function NameForm({ org, save, onSaved, onReload }: NameFormProps) {
  const [name, setName] = useState(org.name)
  const [saved, setSaved] = useState(false)
  const [conflict, setConflict] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSaved(false)
    setConflict(false)
    try {
      onSaved(await save(name))
      setSaved(true)
      return undefined
    } catch (err) {
      return saveMessage(err, setConflict)
    }
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label="Clinic name"
        value={name}
        maxLength={200}
        onChange={(e) => {
          setName(e.target.value)
          setSaved(false)
        }}
        required
      />
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
  if (err.status === 403) return 'Only a clinic admin can change these settings.'
  return somethingWrong
}
