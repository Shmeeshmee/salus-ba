import { useId, useState, type FormEvent } from 'react'
import { useFormat, useT } from '../i18n/useT'
import type { Member } from '../staff'
import { reasons, type ChatApi, type ChatMessage, type Conversation, type Reason } from './chatApi'
import { Dialog } from './Dialog'
import { errorText } from './names'

// AddDoctor adds a colleague to the conversation (SAL-28): a doctor of the clinic who does not take part yet, the
// reason, and what they may read: the whole history or from a message on, chosen among the latest loaded. The web
// app finds only the clinic's own doctors (its PractitionerRole search), so it adds only those.

type Props = {
  chat: ChatApi
  conversation: Conversation
  // organizationId is the clinic of the doctors offered: the signed-in doctor's.
  organizationId: string
  staff: Member[]
  messages: ChatMessage[]
  zone: string
  onAdded: () => void
  onClose: () => void
}

// choices is how many of the latest messages the history can start from.
const choices = 20

export function AddDoctor({ chat, conversation, organizationId, staff, messages, zone, onAdded, onClose }: Props) {
  const t = useT()
  const format = useFormat()
  const id = useId()
  const taking = new Set(conversation.participants?.map((p) => p.practitioner_role_id))
  const candidates = staff.filter((m) => m.active && !taking.has(m.roleId))
  const latest = messages.slice(-choices).reverse()
  const [roleId, setRoleId] = useState(candidates[0]?.roleId ?? '')
  const [reason, setReason] = useState<Reason>('second_opinion')
  const [history, setHistory] = useState<'full' | 'from'>('full')
  const [fromSeq, setFromSeq] = useState(latest[0]?.seq ?? 1)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!roleId || busy) return
    setBusy(true)
    setError(undefined)
    try {
      const base = { practitioner_role_id: roleId, organization_id: organizationId, reason }
      await chat.addParticipant(conversation.id, history === 'full' ? { ...base, history } : { ...base, history, from_seq: fromSeq })
      onAdded()
      onClose()
    } catch (err) {
      setError(errorText(err, t('chat.add.failed'), t))
      setBusy(false)
    }
  }

  const snippet = (m: ChatMessage) => (m.body ? (m.body.length > 48 ? `${m.body.slice(0, 48)}…` : m.body) : t('chat.add.attachmentOnly'))

  return (
    <Dialog label={t('chat.add.title')} onClose={onClose}>
      <h2>{t('chat.add.title')}</h2>
      {candidates.length === 0 ? (
        <p>{t('chat.add.noDoctors')}</p>
      ) : (
        <form onSubmit={submit}>
          <div className="field">
            <label htmlFor={`${id}-doctor`}>{t('chat.add.doctor')}</label>
            <select id={`${id}-doctor`} value={roleId} onChange={(e) => setRoleId(e.target.value)} aria-describedby={`${id}-own`}>
              {candidates.map((m) => (
                <option key={m.roleId} value={m.roleId}>
                  {m.name}
                </option>
              ))}
            </select>
            <p className="hint" id={`${id}-own`}>
              {t('chat.add.ownClinic')}
            </p>
          </div>
          <div className="field">
            <label htmlFor={`${id}-reason`}>{t('chat.add.reason')}</label>
            <select id={`${id}-reason`} value={reason} onChange={(e) => setReason(e.target.value as Reason)}>
              {reasons.map((r) => (
                <option key={r} value={r}>
                  {t(`chat.reason.${r}`)}
                </option>
              ))}
            </select>
          </div>
          <fieldset className="choice">
            <legend>{t('chat.add.history')}</legend>
            <label>
              <input type="radio" name={`${id}-history`} checked={history === 'full'} onChange={() => setHistory('full')} />
              {t('chat.add.full')}
            </label>
            <label>
              <input type="radio" name={`${id}-history`} checked={history === 'from'} disabled={latest.length === 0} onChange={() => setHistory('from')} />
              {t('chat.add.from')}
            </label>
          </fieldset>
          {history === 'from' && (
            <div className="field">
              <label htmlFor={`${id}-from`}>{t('chat.add.fromMessage')}</label>
              <select id={`${id}-from`} value={fromSeq} onChange={(e) => setFromSeq(Number(e.target.value))}>
                {latest.map((m) => (
                  <option key={m.seq} value={m.seq}>
                    {t('chat.add.messageOption', { time: format.dateTime(new Date(m.sent_at), zone), text: snippet(m) })}
                  </option>
                ))}
              </select>
            </div>
          )}
          {error && (
            <p className="alert" role="alert">
              {error}
            </p>
          )}
          <div className="actions">
            <button type="submit" className="primary" disabled={busy}>
              {busy ? t('chat.add.adding') : t('chat.add.submit')}
            </button>
            <button type="button" onClick={onClose}>
              {t('chat.add.cancel')}
            </button>
          </div>
        </form>
      )}
    </Dialog>
  )
}
