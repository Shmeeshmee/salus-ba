import { useCallback, useEffect, useRef, useState } from 'react'
import { useFormat, useLanguage, useT } from '../i18n/useT'
import type { Member } from '../staff'
import { Link } from '../ui'
import { AddDoctor } from './AddDoctor'
import { Attachments } from './Attachments'
import type { ChatApi, ChatMessage, Conversation, Summary } from './chatApi'
import { Composer } from './Composer'
import { errorText, participantName, senderName, systemText, usePatientName, type PatientName } from './names'
import { SummaryPanel } from './Summary'
import { Threads } from './Threads'
import { useConversation } from './useConversation'
import { VirtualList } from './VirtualList'
import './chat.css'

// ChatView is one conversation for a doctor (/chats/{id}): the history in a virtualized list, newest at the bottom,
// older pages read on scrolling up, live messages from the socket, the read position moved as the newest is seen;
// the composer; the doctors taking part, with Add a doctor; the doctor-only threads; and, for a long conversation,
// the AI summary above the history.

export type ChatProps = {
  chat: ChatApi
  staff: Member[]
  // practitionerId is the signed-in doctor's: their roles in staff say which participant they are.
  practitionerId?: string
  organizationId: string
  zone: string
  patientName: PatientName
}

// summaryMinimum is the API's long conversation: the doctor reads at least 50 messages of it.
const summaryMinimum = 50

export function ChatView({ chat, id, staff, practitionerId, organizationId, zone, patientName }: ChatProps & { id: string }) {
  const t = useT()
  const { conversation, messages, older, failure, ended, loadOlder, add, refresh } = useConversation(chat, id)
  const patient = usePatientName(patientName, conversation?.patient_id)
  const [atBottom, setAtBottom] = useState(true)
  const [panel, setPanel] = useState<'add' | 'threads'>()
  const [summary, setSummary] = useState<SummaryState>()
  const { language } = useLanguage()
  const read = useRef(0)

  // summarize asks for the AI summary in the UI's language; it is shown until dismissed and never stored.
  async function summarize() {
    setSummary({ state: 'loading' })
    try {
      setSummary({ state: 'shown', summary: await chat.summarize(id, language) })
    } catch (err) {
      setSummary({ state: 'failed', error: errorText(err, t('chat.summary.failed'), t) })
    }
  }

  // The newest message is read once the doctor is at the bottom of the history.
  const newest = messages.at(-1)?.seq ?? 0
  useEffect(() => {
    if (!atBottom || newest <= read.current) return
    read.current = newest
    chat.markRead(id, newest).catch(() => undefined) // the next message moves it again
  }, [chat, id, newest, atBottom])

  const onScrolled = useCallback(
    ({ nearTop, atBottom }: { nearTop: boolean; atBottom: boolean }) => {
      setAtBottom(atBottom)
      if (nearTop) void loadOlder()
    },
    [loadOlder],
  )

  if (failure) {
    return (
      <p className="alert" role="alert">
        {errorText(failure.error, t('chat.conversationFailed'), t)}
      </p>
    )
  }
  if (!conversation) return <p className="muted">{t('common.loading')}</p>

  const myRoles = new Set(staff.filter((m) => m.practitionerId === practitionerId).map((m) => m.roleId))
  const me = conversation.participants?.find((p) => p.role === 'practitioner' && myRoles.has(p.practitioner_role_id ?? ''))
  const long = me !== undefined && conversation.last_seq - me.history_from_seq >= summaryMinimum
  const thread = conversation.staff_only
  // Only the treating doctor adds and removes doctors (SAL-28); never in a thread.
  const treating = !thread && me !== undefined && me.practitioner_role_id === conversation.practitioner_role_id

  const row = (m: ChatMessage) => <MessageRow m={m} conversation={conversation} mine={m.sender_participant_id === me?.id} staff={staff} patient={patient} zone={zone} />

  return (
    <div className={panel === 'threads' ? 'chat with-side' : 'chat'}>
      <div className="chat-main">
        <Header conversation={conversation} staff={staff} patient={patient} meId={me?.id} treating={treating} chat={chat} onChanged={refresh} />
        <div className="chat-tools">
          {treating && (
            <button type="button" onClick={() => setPanel('add')} disabled={ended !== undefined}>
              {t('chat.add.open')}
            </button>
          )}
          {!thread && (
            <button type="button" aria-pressed={panel === 'threads'} onClick={() => setPanel(panel === 'threads' ? undefined : 'threads')}>
              {t('chat.threads.open')}
            </button>
          )}
          {long && (
            <button type="button" title={t('chat.summary.hint')} onClick={summarize} disabled={summary?.state === 'loading'}>
              {summary?.state === 'loading' ? t('chat.summary.loading') : t('chat.summary.button')}
            </button>
          )}
        </div>
        {summary?.state === 'shown' && <SummaryPanel summary={summary.summary} zone={zone} onDismiss={() => setSummary(undefined)} />}
        {summary?.state === 'failed' && (
          <p className="alert" role="alert">
            {summary.error}
          </p>
        )}
        <p className="muted small chat-older" aria-live="polite">
          {older.loading ? t('chat.olderLoading') : older.failed ? t('chat.olderFailed') : !older.next && messages.length > 0 ? t('chat.start') : ''}
        </p>
        {messages.length === 0 && <p className="muted">{t('chat.noMessages')}</p>}
        <VirtualList items={messages} keyOf={seqOf} render={row} label={t('chat.history')} onScrolled={onScrolled} />
        {ended && (
          <p className="alert" role="alert">
            {t(`chat.ended.${ended}`)}
          </p>
        )}
        <Composer chat={chat} id={id} disabled={ended !== undefined} onSent={add} />
      </div>
      {panel === 'threads' && <Threads chat={chat} id={id} zone={zone} onClose={() => setPanel(undefined)} />}
      {panel === 'add' && <AddDoctor chat={chat} conversation={conversation} organizationId={organizationId} staff={staff} messages={messages} zone={zone} onAdded={refresh} onClose={() => setPanel(undefined)} />}
    </div>
  )
}

const seqOf = (m: ChatMessage) => m.seq

type SummaryState = { state: 'loading' } | { state: 'shown'; summary: Summary } | { state: 'failed'; error: string }

type HeaderProps = { conversation: Conversation; staff: Member[]; patient?: string; meId?: string; treating: boolean; chat: ChatApi; onChanged: () => void }

// Header names the conversation (the patient, or a doctor-only thread) and lists its doctors behind a dropdown,
// each added one with Remove for the treating doctor.
function Header({ conversation, staff, patient, meId, treating, chat, onChanged }: HeaderProps) {
  const t = useT()
  const [error, setError] = useState<string>()
  const doctors = conversation.participants?.filter((p) => p.role === 'practitioner') ?? []

  async function remove(participantId: string) {
    setError(undefined)
    try {
      await chat.removeParticipant(conversation.id, participantId)
      onChanged()
    } catch (err) {
      setError(errorText(err, t('chat.removeFailed'), t))
    }
  }

  return (
    <div className="chat-head">
      <Link to={conversation.parent_id ? `/chats/${encodeURIComponent(conversation.parent_id)}` : '/chats'} className="back">
        {conversation.parent_id ? t('chat.toParent') : t('chat.back')}
      </Link>
      <h1>
        {patient ?? t('chat.patient')}
        {conversation.staff_only && <span className="tag staff-only">{t('chat.staffOnly')}</span>}
      </h1>
      {conversation.staff_only && <p className="hint">{t('chat.staffOnlyNote')}</p>}
      <details className="more chat-doctors">
        <summary>{t('chat.doctors', { n: doctors.length })}</summary>
        <ul>
          {doctors.map((p) => (
            <li key={p.id}>
              <span>{participantName(p, staff, patient, t)}</span>
              {p.status === 'pending_consent' && <span className="tag">{t('chat.pending')}</span>}
              {treating && p.id !== meId && (
                <button type="button" className="quiet" onClick={() => remove(p.id)}>
                  {t('chat.remove', { name: participantName(p, staff, patient, t) })}
                </button>
              )}
            </li>
          ))}
        </ul>
      </details>
      {error && (
        <p className="alert" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

type RowProps = { m: ChatMessage; conversation: Conversation; mine: boolean; staff: Member[]; patient?: string; zone: string }

// MessageRow is one message: a system message from the catalog, or a text with its sender, time and files.
function MessageRow({ m, conversation, mine, staff, patient, zone }: RowProps) {
  const t = useT()
  const format = useFormat()
  if (m.kind === 'system') {
    const { text, reason } = systemText(m, staff, t)
    return (
      <p className="chat-system">
        <span>{text}</span>
        {reason && <span> {reason}</span>}
      </p>
    )
  }
  const sent = new Date(m.sent_at)
  return (
    <div className={mine ? 'chat-message mine' : 'chat-message'}>
      <p className="chat-meta">
        <span className="chat-sender">{mine ? t('chat.you') : senderName(m, conversation, staff, patient, t)}</span>
        <time dateTime={m.sent_at} title={format.dateTime(sent, zone)}>
          {format.time(sent, zone)}
        </time>
        {m.edited_at && <span className="muted">{t('chat.edited')}</span>}
      </p>
      {m.body && <p className="chat-body">{m.body}</p>}
      {m.attachments && m.attachments.length > 0 && <Attachments conversationId={conversation.id} files={m.attachments} />}
    </div>
  )
}
