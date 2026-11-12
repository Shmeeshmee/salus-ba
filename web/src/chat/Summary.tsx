import { useId } from 'react'
import { CrossIcon } from '../icons'
import { useFormat, useT } from '../i18n/useT'
import type { Summary } from './chatApi'

// SummaryPanel shows the AI summary above the history, labelled AI generated, with when it was written and up to
// which message; the messages below stay as they are. It is the reader's to dismiss; nothing of it is stored.
export function SummaryPanel({ summary, zone, onDismiss }: { summary: Summary; zone: string; onDismiss: () => void }) {
  const t = useT()
  const format = useFormat()
  const heading = useId()
  return (
    <section className="chat-summary" aria-labelledby={heading}>
      <div className="chat-summary-head">
        <span className="tag ai">{t('chat.summary.label')}</span>
        <h2 id={heading}>{t('chat.summary.heading')}</h2>
        <button type="button" className="icon" aria-label={t('chat.summary.dismiss')} title={t('chat.summary.dismiss')} onClick={onDismiss}>
          <CrossIcon />
        </button>
      </div>
      <p className="chat-summary-text">{summary.summary}</p>
      <p className="muted small">{t('chat.summary.meta', { time: format.time(new Date(summary.generated_at), zone), n: summary.up_to_seq })}</p>
    </section>
  )
}
