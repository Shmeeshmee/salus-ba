import { useId, useState, type FormEvent } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { useT } from './i18n/useT'
import { Alert, Submit } from './ui'

// DeskConsent records the care consent a patient the clinic registered gives in person at the desk, when they have
// none active with the clinic: after an import, or after a withdrawal (POST /clinic/patients/{id}/consent). The same
// legal line as a walk-in's registration; the server records who at the desk recorded it.
export function DeskConsent({ api, patientId, onRecorded }: { api: ApiClient; patientId: string; onRecorded: () => void }) {
  const t = useT()
  const id = useId()
  const [given, setGiven] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function record(e: FormEvent) {
    e.preventDefault()
    if (busy || !given) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/clinic/patients/{id}/consent', { params: { path: { id: patientId } } }))
    setBusy(false)
    // One already active (recorded meanwhile at another desk) is what was asked for.
    if (res.ok || res.code === 'consent_active') onRecorded()
    else setError(t('desk.consent.failed'))
  }

  return (
    <form onSubmit={record}>
      <div className="check">
        <input id={`${id}-consent`} type="checkbox" checked={given} onChange={(e) => setGiven(e.target.checked)} />
        <label htmlFor={`${id}-consent`}>{t('desk.register.consent')}</label>
      </div>
      <Alert message={error} />
      <Submit busy={busy} disabled={!given} label={t('desk.consent.record')} busyLabel={t('desk.consent.recording')} />
    </form>
  )
}
