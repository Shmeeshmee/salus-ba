import { useState, type FormEvent } from 'react'
import type { components } from '../api/schema'
import { useT } from '../i18n/useT'
import { escapeCancels } from './reviewHelpers'

type Status = components['schemas']['ReviewStatus']

// The components the review panel's two lists share: the terms (Review) and the diagnosis groups' names (GroupNames).

export function StatusTag({ status }: { status: Status }) {
  const t = useT()
  return <span className={`tag review-status ${status}`}>{t(`review.status.${status}`)}</span>
}


export function NoteForm({ onSave, onCancel, onMissing }: { onSave: (note: string) => void; onCancel: () => void; onMissing: () => void }) {
  const t = useT()
  const [note, setNote] = useState('')
  function submit(event: FormEvent) {
    event.preventDefault()
    const value = note.trim()
    if (value) onSave(value)
    else onMissing()
  }
  return (
    <form className="review-form" onSubmit={submit} onKeyDown={escapeCancels(onCancel)}>
      <div className="field">
        <label htmlFor="review-note">{t('review.noteLabel')}</label>
        <input id="review-note" autoFocus maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
      </div>
      <FormActions onCancel={onCancel} />
    </form>
  )
}

export function FormActions({ onCancel }: { onCancel: () => void }) {
  const t = useT()
  return (
    <div className="actions">
      <button type="submit" className="primary">
        {t('review.save')}
      </button>
      <button type="button" onClick={onCancel}>
        {t('review.cancel')}
      </button>
    </div>
  )
}
