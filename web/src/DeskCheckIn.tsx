import { useState } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { Alert } from './ui'

type Appointment = components['schemas']['Appointment']

// DeskCheckIn marks an in-person visit's patient as arrived at the desk (W6 of clinic-cms,
// POST /clinic/appointments/{id}/arrive). The server sets the time and refuses a video visit or one no longer booked.
export function DeskCheckIn({ api, appointmentId, onArrived }: { api: ApiClient; appointmentId: string; onArrived: (appointment: Appointment) => void }) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function arrive() {
    if (busy) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/clinic/appointments/{id}/arrive', { params: { path: { id: appointmentId } } }))
    setBusy(false)
    if (res.ok) onArrived(res.data)
    else setError(message(t, res.code))
  }

  return (
    <>
      <button type="button" className="primary" disabled={busy} onClick={arrive}>
        {t('desk.arrive')}
      </button>
      <Alert message={error} />
    </>
  )
}

function message(t: ReturnType<typeof useT>, code?: ErrorCode): string {
  if (code === 'not_in_person') return t('desk.notInPerson')
  if (code === 'appointment_closed') return t('desk.closed')
  return t('desk.arriveFailed')
}
