import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { formatDateTime } from './i18n/format'
import { useLanguage, useT } from './i18n/useT'
import { Alert, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Export = components['schemas']['ClinicExport']

// archiveURL is where a ready export's archive is downloaded, on the same origin as every other call, so the session
// cookie goes with it.
function archiveURL(id: string): string {
  return `${globalThis.location.origin}/clinic/export/${encodeURIComponent(id)}/archive`
}

// ClinicExport is the admin's export of the clinic's data (docs/clinic-cms M2): asking for it, waiting while the job
// builds the archive (the status is read every pollMs), and the download until it expires (7 days after asking).
// zone is the clinic's time zone, for the expiry.
export function ClinicExport({ api, zone, pollMs = 5000 }: { api: ApiClient; zone: string; pollMs?: number }) {
  const t = useT()
  const { language } = useLanguage()
  const [current, setCurrent] = useState<Export | undefined>()
  const { busy, error, onSubmit } = useSubmit(async () => {
    const res = await attempt(api.POST('/clinic/export', {}))
    if (!res.ok) return res.code === 'forbidden' ? t('admin.exportForbidden') : t('admin.exportFailed')
    setCurrent(res.data)
    return undefined
  })

  const queuedID = current?.status === 'queued' ? current.id : undefined
  useEffect(() => {
    if (!queuedID) return
    let live = true
    const timer = setInterval(async () => {
      const res = await attempt(api.GET('/clinic/export/{id}', { params: { path: { id: queuedID } } }))
      if (live && res.ok && res.data.status !== 'queued') setCurrent(res.data)
    }, pollMs)
    return () => {
      live = false
      clearInterval(timer)
    }
  }, [api, queuedID, pollMs])

  return (
    <section className="panel">
      <h2>{t('admin.exportTitle')}</h2>
      <p className="hint">{t('admin.exportHint')}</p>
      {current?.status === 'queued' && <p className="muted" role="status">{t('admin.exportBuilding')}</p>}
      {current?.status === 'failed' && <Alert message={t('admin.exportBuildFailed')} />}
      {current?.status === 'expired' && <Notice message={t('admin.exportExpired')} />}
      {current?.status === 'ready' && (
        <p>
          <a href={archiveURL(current.id)} download>
            {t('admin.exportDownload')}
          </a>{' '}
          {current.expiresAt && (
            <span className="muted">{t('admin.exportUntil', { at: formatDateTime(new Date(current.expiresAt), zone, language) })}</span>
          )}
        </p>
      )}
      <form onSubmit={onSubmit} noValidate>
        <Alert message={error} />
        {current?.status !== 'queued' && current?.status !== 'ready' && (
          <Submit busy={busy} label={t('admin.exportAsk')} busyLabel={t('admin.exportAsking')} />
        )}
      </form>
    </section>
  )
}
