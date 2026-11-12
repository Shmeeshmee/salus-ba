import { useId, useState, type ChangeEvent, type FormEvent, type KeyboardEvent } from 'react'
import { CrossIcon, PaperclipIcon } from '../icons'
import { useT } from '../i18n/useT'
import type { ChatApi, ChatAttachment, ChatMessage } from './chatApi'
import { errorText } from './names'

// Composer writes a message: its text and the files uploaded for it. A send carries a fresh client id
// (crypto.randomUUID); a failed one is tried again with the same id and the same content, so the API keeps one
// message however often it is retried. The text is locked while a message is on its way or waiting for a retry.

type Pending = { clientId: string; body: string; attachmentIds: string[] }
type Upload = { key: string; name: string; attachment?: ChatAttachment; error?: string }

type Props = { chat: ChatApi; id: string; disabled: boolean; onSent: (m: ChatMessage) => void }

export function Composer({ chat, id, disabled, onSent }: Props) {
  const t = useT()
  const boxId = useId()
  const [text, setText] = useState('')
  const [uploads, setUploads] = useState<Upload[]>([])
  const [pending, setPending] = useState<Pending>()
  const [state, setState] = useState<'idle' | 'sending' | 'failed'>('idle')
  const [error, setError] = useState<string>()

  const ready = uploads.flatMap((u) => (u.attachment ? [u.attachment.id] : []))
  const uploading = uploads.some((u) => !u.attachment && !u.error)

  async function send(p: Pending) {
    setPending(p)
    setState('sending')
    setError(undefined)
    try {
      const m = await chat.send(id, p.clientId, p.body, p.attachmentIds)
      onSent(m)
      setText('')
      setUploads([])
      setPending(undefined)
      setState('idle')
    } catch (err) {
      setState('failed')
      setError(errorText(err, t('chat.composer.failed'), t))
    }
  }

  function submit(e?: FormEvent) {
    e?.preventDefault()
    if (disabled || state === 'sending' || uploading) return
    if (pending) return void send(pending)
    const body = text.trim()
    if (body === '' && ready.length === 0) return
    void send({ clientId: crypto.randomUUID(), body, attachmentIds: ready })
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key !== 'Enter' || e.shiftKey || e.nativeEvent.isComposing) return
    e.preventDefault()
    submit()
  }

  async function attach(e: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    for (const file of files) {
      const key = crypto.randomUUID()
      setUploads((us) => [...us, { key, name: file.name }])
      try {
        const attachment = await chat.upload(id, file)
        setUploads((us) => us.map((u) => (u.key === key ? { ...u, attachment } : u)))
      } catch (err) {
        const message = errorText(err, t('chat.composer.uploadFailed', { name: file.name }), t)
        setUploads((us) => us.map((u) => (u.key === key ? { ...u, error: message } : u)))
      }
    }
  }

  const locked = disabled || pending !== undefined
  return (
    <form className="chat-composer" onSubmit={submit}>
      {uploads.length > 0 && (
        <ul className="chat-uploads">
          {uploads.map((u) => (
            <li key={u.key} className={u.error ? 'alert' : undefined}>
              <span>{u.error ?? (u.attachment ? t('chat.attachment.scanning', { name: u.name }) : u.name)}</span>
              {!pending && (
                <button type="button" className="icon" aria-label={t('chat.composer.removeFile', { name: u.name })} title={t('chat.composer.removeFile', { name: u.name })} onClick={() => setUploads((us) => us.filter((x) => x.key !== u.key))}>
                  <CrossIcon />
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {state === 'failed' && error && (
        <p className="alert" role="alert">
          {error}
        </p>
      )}
      <div className="chat-compose-row">
        <label className="button icon chat-attach" title={t('chat.composer.attach')}>
          <PaperclipIcon />
          <span className="sr-only">{t('chat.composer.attach')}</span>
          <input type="file" className="sr-only" multiple disabled={locked} onChange={attach} />
        </label>
        <label htmlFor={boxId} className="sr-only">
          {t('chat.composer.label')}
        </label>
        <textarea id={boxId} rows={2} value={text} disabled={disabled} readOnly={pending !== undefined} maxLength={4000} onChange={(e) => setText(e.target.value)} onKeyDown={onKeyDown} />
        <button type="submit" className="primary" disabled={disabled || state === 'sending' || uploading}>
          {state === 'sending' ? t('chat.composer.sending') : state === 'failed' ? t('chat.composer.retry') : t('chat.composer.send')}
        </button>
      </div>
    </form>
  )
}
