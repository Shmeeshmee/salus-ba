import { useEffect, useState } from 'react'
import { useFormat, useT } from '../i18n/useT'
import { Link } from '../ui'
import type { ChatApi, Conversation } from './chatApi'
import { usePatientName, type PatientName } from './names'
import './chat.css'

// Conversations is the doctor's list of conversations (/chats), newest activity first: the patient, a doctor-only
// thread marked as such, when it last moved, and how many messages the doctor has not read.

type State = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; conversations: Conversation[]; next?: string; more: boolean }

export function Conversations({ chat, zone, patientName }: { chat: ChatApi; zone: string; patientName: PatientName }) {
  const t = useT()
  const [state, setState] = useState<State>({ name: 'loading' })

  useEffect(() => {
    let current = true
    chat.list().then(
      (page) => current && setState({ name: 'loaded', ...page, more: false }),
      () => current && setState({ name: 'failed' }),
    )
    return () => {
      current = false
    }
  }, [chat])

  async function more() {
    if (state.name !== 'loaded' || !state.next) return
    setState({ ...state, more: true })
    try {
      const page = await chat.list(state.next)
      setState({ name: 'loaded', conversations: [...state.conversations, ...page.conversations], next: page.next, more: false })
    } catch {
      setState({ ...state, more: false })
    }
  }

  return (
    <section>
      <h1>{t('chat.title')}</h1>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && (
        <p className="alert" role="alert">
          {t('chat.loadFailed')}
        </p>
      )}
      {state.name === 'loaded' && state.conversations.length === 0 && <p className="panel">{t('chat.empty')}</p>}
      {state.name === 'loaded' && state.conversations.length > 0 && (
        <ul className="rows chat-list">
          {state.conversations.map((c) => (
            <li key={c.id}>
              <Row c={c} zone={zone} patientName={patientName} />
            </li>
          ))}
        </ul>
      )}
      {state.name === 'loaded' && state.next && (
        <div className="actions">
          <button type="button" onClick={more} disabled={state.more}>
            {t('chat.more')}
          </button>
        </div>
      )}
    </section>
  )
}

function Row({ c, zone, patientName }: { c: Conversation; zone: string; patientName: PatientName }) {
  const t = useT()
  const format = useFormat()
  const name = usePatientName(patientName, c.patient_id)
  return (
    <Link to={`/chats/${encodeURIComponent(c.id)}`} className="chat-row">
      <span className="chat-row-name">
        {name ?? t('chat.patient')}
        {c.staff_only && <span className="tag staff-only">{t('chat.staffOnly')}</span>}
      </span>
      <span className="muted small">{format.dateTime(new Date(c.last_sent_at ?? c.created_at), zone)}</span>
      {c.unread > 0 && <span className="chat-unread">{t('chat.unread', { n: c.unread })}</span>}
    </Link>
  )
}
