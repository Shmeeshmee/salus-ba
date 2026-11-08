import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react'
import { binaryContentURL, type ApiClient } from './api/client'
import { attempt } from './api/result'
import { sizeText, type Attached } from './attachment'
import { ChevronIcon, CrossIcon } from './icons'
import { useLanguage, useT } from './i18n/useT'

type Props = { api: ApiClient; files: Attached[] }

type T = ReturnType<typeof useT>

// nameOf is the file's title, or its kind while it has none.
const nameOf = (f: Attached, t: T) => f.title || t(`attachments.kinds.${f.kind}`)
// labelOf is W11, the name a screen reader gives the photo, document or video.
const labelOf = (f: Attached, t: T) => (f.title ? t(`attachments.label.${f.kind}`, { title: f.title }) : t(`attachments.kinds.${f.kind}`))

// Files is the answer of a file upload item on a submitted response (W7 to W9): each file with its kind and
// size, opened in the viewer or downloaded. The bytes are read from the content endpoint built from the binary
// id, on the app's own origin, so the session cookie goes with every request and the browser makes the Range
// requests of a video itself.
export function Files({ api, files }: Props) {
  const t = useT()
  const { language } = useLanguage()
  const shown = files.filter((f) => f.id !== undefined)
  const [open, setOpen] = useState<number>()
  return (
    <>
      <ul className="files">
        {files.map((f, i) => (
          // A file has no other identity on the answer: the same file may be attached twice.
          <li key={`${f.id ?? 'none'}-${i}`}>
            <span className="file-name">
              <span className="file-title">{nameOf(f, t)}</span>
              <span className="file-details">{t('attachments.details', { kind: t(`attachments.kinds.${f.kind}`), size: sizeText(f.size, t, language) })}</span>
            </span>
            {f.id !== undefined && (
              <span className="file-actions">
                <button type="button" className="quiet" onClick={() => setOpen(shown.indexOf(f))}>
                  {t('attachments.open')}
                </button>
                <DownloadLink file={f} className="quiet" />
              </span>
            )}
          </li>
        ))}
      </ul>
      {open !== undefined && <Viewer api={api} files={shown} start={open} onClose={() => setOpen(undefined)} />}
    </>
  )
}

function DownloadLink({ file, className }: { file: Attached; className: string }) {
  const t = useT()
  return (
    <a className={className} href={binaryContentURL(file.id ?? '')} download={file.title || ''}>
      {t('attachments.download')}
    </a>
  )
}

type Problem = 'gone' | 'failed' | 'cannot'
type ViewerProps = { api: ApiClient; files: Attached[]; start: number; onClose: () => void }

// Viewer shows one file at a time over the page (W10 to W13): a photo as an image, a PDF in the browser's own
// viewer, a video in the browser's player. It takes the focus, keeps it inside while open, closes on escape and
// gives the focus back to what had it.
function Viewer({ api, files, start, onClose }: ViewerProps) {
  const t = useT()
  const id = useId()
  const [at, setAt] = useState(start)
  const [tries, setTries] = useState(0) // a new try mounts the media again
  const [problem, setProblem] = useState<Problem>()
  const dialog = useRef<HTMLDivElement>(null)
  const closeButton = useRef<HTMLButtonElement>(null)
  const current = useRef(files[start]) // the file shown, for answers that come back after a step
  const f = files[at]

  useEffect(() => {
    const before = document.activeElement
    closeButton.current?.focus()
    return () => {
      if (before instanceof HTMLElement && before.isConnected) before.focus()
    }
  }, [])

  function go(to: number) {
    if (to < 0 || to >= files.length) return
    current.current = files[to]
    setAt(to)
    setProblem(undefined)
  }

  // failed asks the API why the browser could not load the file: gone (404, or rejected by the scan), there
  // but not shown (W12), or anything else (W13, with Try again). An answer about a file no longer shown is
  // dropped.
  async function failed() {
    const asked = f
    const res = await attempt(api.GET('/binaries/{id}', { params: { path: { id: asked.id ?? '' } } }))
    if (current.current !== asked) return
    if (res.ok) setProblem(res.data.status === 'clean' ? 'cannot' : res.data.status === 'rejected' ? 'gone' : 'failed')
    else setProblem(res.response?.status === 404 ? 'gone' : 'failed')
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
    } else if (e.key === 'Tab') trap(e, dialog.current)
  }

  return (
    <div className="viewer-backdrop">
      <div ref={dialog} className="viewer" role="dialog" aria-modal="true" aria-labelledby={`${id}-n ${id}-t`} onKeyDown={onKeyDown}>
        <div className="viewer-bar">
          <p className="viewer-title">
            <span id={`${id}-t`} className="file-title">
              {nameOf(f, t)}
            </span>
            <span id={`${id}-n`} className="muted">
              {t('attachments.position', { n: at + 1, total: files.length })}
            </span>
          </p>
          {files.length > 1 && (
            <>
              <button type="button" className="icon" aria-label={t('attachments.previous')} title={t('attachments.previous')} aria-disabled={at === 0} onClick={() => go(at - 1)}>
                <ChevronIcon direction="left" />
              </button>
              <button type="button" className="icon" aria-label={t('attachments.next')} title={t('attachments.next')} aria-disabled={at === files.length - 1} onClick={() => go(at + 1)}>
                <ChevronIcon direction="right" />
              </button>
            </>
          )}
          <button type="button" ref={closeButton} className="icon" aria-label={t('attachments.close')} title={t('attachments.close')} onClick={onClose}>
            <CrossIcon />
          </button>
        </div>
        <div className="viewer-body">
          {problem === undefined ? (
            <Media key={`${f.id}-${tries}`} file={f} onError={failed} />
          ) : (
            <ProblemView
              problem={problem}
              file={f}
              onRetry={() => {
                setProblem(undefined)
                setTries((n) => n + 1)
              }}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function Media({ file, onError }: { file: Attached; onError: () => void }) {
  const t = useT()
  const url = binaryContentURL(file.id ?? '')
  if (file.kind === 'photo') return <img src={url} alt={labelOf(file, t)} onError={onError} />
  if (file.kind === 'video') return <video src={url} controls preload="metadata" aria-label={labelOf(file, t)} onError={onError} />
  // The browser's own PDF viewer; what is inside shows where it has none, and an error (a 404) is asked about.
  return (
    <object data={url} type="application/pdf" aria-label={labelOf(file, t)} onError={onError}>
      <CannotShow file={file} />
    </object>
  )
}

function CannotShow({ file }: { file: Attached }) {
  const t = useT()
  return (
    <div className="viewer-message">
      <p>{t('attachments.cannotShow')}</p>
      <div className="actions">
        <DownloadLink file={file} className="button" />
      </div>
    </div>
  )
}

function ProblemView({ problem, file, onRetry }: { problem: Problem; file: Attached; onRetry: () => void }) {
  const t = useT()
  if (problem === 'cannot') return <CannotShow file={file} />
  return (
    <div className="viewer-message">
      <p className="alert" role="alert">
        {t(problem === 'gone' ? 'attachments.gone' : 'attachments.failed')}
      </p>
      {problem === 'failed' && (
        <div className="actions">
          <button type="button" onClick={onRetry}>
            {t('attachments.tryAgain')}
          </button>
        </div>
      )}
    </div>
  )
}

// trap keeps Tab and Shift+Tab inside the dialog: from the last control to the first, and back.
function trap(e: KeyboardEvent, root: HTMLElement | null) {
  const focusable = Array.from(root?.querySelectorAll<HTMLElement>('button, a[href], video[controls], object') ?? [])
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}
