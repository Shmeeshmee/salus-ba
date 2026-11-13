import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert } from './ui'

type Change = components['schemas']['AppointmentChange']
type T = ReturnType<typeof useT>
type F = ReturnType<typeof useFormat>

// AppointmentChanges is an appointment's history, oldest first (GET /appointments/{id}/changes): its confirmation,
// its reschedules and its status changes, each with how and when it was made, in the clinic's zone. Who made a change
// is never in it. A kind this screen does not describe yet is left out.
export function AppointmentChanges({ api, id, zone }: { api: ApiClient; id: string; zone: string }) {
  const t = useT()
  const f = useFormat()
  const { state } = useLoad(async () => {
    const res = await attempt(api.GET('/appointments/{id}/changes', { params: { path: { id } } }))
    if (!res.ok) throw new Error('changes')
    return res.data.changes
  }, [api, id])

  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('appointments.changesLoadFailed')} />
  const rows = state.value.flatMap((c) => {
    const what = describe(t, f, c, zone)
    return what ? [{ c, what }] : []
  })
  return (
    <section className="changes" aria-label={t('appointments.changes')}>
      <h2>{t('appointments.changes')}</h2>
      {rows.length === 0 ? (
        <p className="muted">{t('appointments.noChanges')}</p>
      ) : (
        <ul>
          {rows.map(({ c, what }, i) => (
            <li key={`${c.at}-${i}`}>
              <span>{what}</span>
              <span className="muted small">
                {f.dateTime(new Date(c.at), zone)} · {t(`appointments.via.${c.via}`)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

// describe is what the change c did, or nothing for a kind not described here.
function describe(t: T, f: F, c: Change, zone: string): string | undefined {
  switch (c.kind) {
    case 'confirmed':
      return t('appointments.change.confirmed')
    case 'rescheduled':
      if (!c.from_start || !c.to_start) return undefined
      return t('appointments.change.rescheduled', { from: f.dateTime(new Date(c.from_start), zone), to: f.dateTime(new Date(c.to_start), zone) })
    case 'status_set':
      if (c.to_status !== 'cancelled' && c.to_status !== 'fulfilled' && c.to_status !== 'noshow') return undefined
      return t(`appointments.change.status.${c.to_status}`)
    case 'status_corrected':
      if (c.to_status !== 'fulfilled' && c.to_status !== 'noshow') return undefined
      return t(`appointments.change.corrected.${c.to_status}`)
    default:
      return undefined
  }
}
