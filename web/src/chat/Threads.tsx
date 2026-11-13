import { useEffect, useId, useState } from 'react'
import { CrossIcon } from '../icons'
import { useFormat, useT } from '../i18n/useT'
import { navigate } from '../route'
import { Link } from '../ui'
import type { ChatApi, Conversation } from './chatApi'
import { errorText } from './names'

// Threads is the side panel of a conversation's doctor-only threads (SAL-28): each opens as its own chat, marked
// as not visible to the patient; a new one starts with the conversation's doctors.
export function Threads({ chat, id, zone, onClose }: { chat: ChatApi; id: string; zone: string; onClose: () => void }) {
  const t = useT()
  const format = useFormat()
  const heading = useId()
  const [threads, setThreads] = useState<Conversation[]>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let current = true
    chat.threads(id).then(
      (ts) => current && setThreads(ts),
      (err: unknown) => current && setError(errorText(err, t('chat.threads.failed'), t)),
    )
    return () => {
      current = false
    }
    // t is new with every render; the threads are read once per conversation.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [chat, id])

  async function start() {
    setBusy(true)
    setError(undefined)
    try {
      const thread = await chat.openThread(id)
      navigate(`/chats/${encodeURIComponent(thread.id)}`)
    } catch (err) {
      setError(errorText(err, t('chat.threads.startFailed'), t))
      setBusy(false)
    }
  }

  return (
    <aside className="chat-threads" aria-labelledby={heading}>
      <div className="chat-threads-head">
        <h2 id={heading}>{t('chat.threads.title')}</h2>
        <button type="button" className="icon" aria-label={t('chat.threads.close')} title={t('chat.threads.close')} onClick={onClose}>
          <CrossIcon />
        </button>
      </div>
      <p className="hint">{t('chat.threads.note')}</p>
      {threads?.length === 0 && <p className="muted">{t('chat.threads.none')}</p>}
      {threads && threads.length > 0 && (
        <ul className="chat-thread-list">
          {threads.map((th) => (
            <li key={th.id}>
              <Link to={`/chats/${encodeURIComponent(th.id)}`}>
                {t('chat.threads.item', { time: format.dateTime(new Date(th.created_at), zone) })}
                {th.unread > 0 && <span className="chat-unread">{t('chat.unread', { n: th.unread })}</span>}
              </Link>
            </li>
          ))}
        </ul>
      )}
      {error && (
        <p className="alert" role="alert">
          {error}
        </p>
      )}
      <div className="actions">
        <button type="button" onClick={start} disabled={busy}>
          {t('chat.threads.start')}
        </button>
      </div>
    </aside>
  )
}
