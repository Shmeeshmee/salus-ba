import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import type { AppointmentStatus } from './api/fhir'
import { attempt } from './api/result'
import { useT } from './i18n/useT'
import type { Flag } from './noShowFlags'
import { Alert, Confirm, TipTag } from './ui'

type T = ReturnType<typeof useT>

// correctionDays is how long after its start a no-show or an attendance is corrected (docs/plans/appointments.md Q6).
const correctionDays = 30
const day = 24 * 60 * 60 * 1000

type CorrectionProps = {
  api: ApiClient
  id: string
  start: Date
  status: AppointmentStatus
  now: Date
  onChanged: (s: AppointmentStatus) => void
}

// NoShowCorrection offers the appointment's practitioner and the clinic's admin to correct a no-show into fulfilled
// and back, within 30 days of the start, after a confirmation (SAL-36). The caller decides who may: it renders it
// for them only. The server checks it all again.
export function NoShowCorrection({ api, id, start, status, now, onChanged }: CorrectionProps) {
  const t = useT()
  const [asking, setAsking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  if ((status !== 'noshow' && status !== 'fulfilled') || now.getTime() - start.getTime() > correctionDays * day) return null
  const to = status === 'noshow' ? 'fulfilled' : 'noshow'

  async function correct() {
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.POST('/appointments/{id}/status', { params: { path: { id } }, body: { status: to } }))
    setBusy(false)
    setAsking(false)
    if (res.ok) onChanged(res.data.status)
    else setError(correctionMessage(t, res.code))
  }

  return (
    <div className="appt-actions">
      <Alert message={error} />
      {asking ? (
        <Confirm
          question={t(to === 'fulfilled' ? 'calendar.noshow.questionToFulfilled' : 'calendar.noshow.questionToNoShow')}
          lines={[t('calendar.noshow.correctionLine')]}
          confirm={t('calendar.noshow.correct')}
          cancel={t('calendar.noshow.keep')}
          busy={busy}
          onConfirm={correct}
          onCancel={() => setAsking(false)}
        />
      ) : (
        <div className="actions">
          <button type="button" className="quiet" onClick={() => setAsking(true)}>
            {t(to === 'fulfilled' ? 'calendar.noshow.correctToFulfilled' : 'calendar.noshow.correctToNoShow')}
          </button>
        </div>
      )}
    </div>
  )
}

function correctionMessage(t: T, code?: string): string {
  if (code === 'correction_window_closed') return t('calendar.noshow.windowClosed')
  if (code === 'appointment_closed') return t('calendar.noshow.changed')
  if (code === 'forbidden' || code === 'not_found') return t('calendar.noshow.forbidden')
  return t('common.somethingWrong')
}



// NoShowTag is a flag of useNoShowFlags as a tag inside a calendar card, which is a link: the detail in its title,
// not a focusable tooltip, as nothing focusable may sit inside a link.
export function NoShowTag({ flag }: { flag?: Flag }) {
  const t = useT()
  if (!flag) return null
  return (
    <span className="tag noshow-flag" title={t('calendar.noshow.flagTip', { count: flag.noshows, days: flag.window_days })}>
      {t('calendar.noshow.flag', { count: flag.noshows })}
    </span>
  )
}

// NoShowFlag shows staff how often the appointment's patient missed appointments at the clinic lately, when the
// clinic's rule is on and they reached its threshold (GET /clinic/noshow-flags, SAL-36, Q7): a tag, with the detail
// in its tooltip. Nothing while the rule is off, under the threshold, or when the flags cannot be read. Never in the
// patient's app.
export function NoShowFlag({ api, id, start }: { api: ApiClient; id: string; start: Date }) {
  const t = useT()
  const [flag, setFlag] = useState<Flag>()
  const at = start.getTime()
  useEffect(() => {
    let current = true
    const from = new Date(at).toISOString()
    const to = new Date(at + 1000).toISOString()
    attempt(api.GET('/clinic/noshow-flags', { params: { query: { from, to } } })).then((res) => {
      if (current && res.ok) setFlag(res.data.flags.find((f) => f.appointment_id === id))
    })
    return () => {
      current = false
    }
  }, [api, id, at])
  if (!flag) return null
  return (
    <TipTag
      className="noshow-flag"
      label={t('calendar.noshow.flag', { count: flag.noshows })}
      tip={t('calendar.noshow.flagTip', { count: flag.noshows, days: flag.window_days })}
    />
  )
}
