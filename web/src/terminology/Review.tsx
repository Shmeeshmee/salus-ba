import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import type { ApiClient } from '../api/client'
import type { components } from '../api/schema'
import { attempt, type Outcome } from '../api/result'
import { useLanguage, useT } from '../i18n/useT'
import { Alert, TipTag } from '../ui'
import { entryStatus } from './entryStatus'
import './review.css'

type Entry = components['schemas']['TermReviewEntry']
type TermLanguage = components['schemas']['TermLanguage']
type Status = components['schemas']['ReviewStatus']
type Progress = components['schemas']['TermReviewProgress']['languages'][number]
type Translation = components['schemas']['TermTranslation']
type Explanation = components['schemas']['TermExplanation']

// The term languages in the enum's order (api/openapi.yaml TermLanguage).
const termLanguages: TermLanguage[] = ['la', 'bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr']
const statuses: (Status | 'all')[] = ['all', 'needs_changes', 'unreviewed', 'approved']
const pageSize = 25

// Part is what a review action is on: the term's labels in the language, or its explanation there.
type Part = 'labels' | 'explanation'
// Open is the form open on an entry: an edit of a part, or the note of a request for changes.
type Open = { index: number; part: Part; mode: 'edit' | 'note' }

type Queue = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; entries: Entry[]; next?: string }

// useLanguageName names a term language in the UI language.
function useLanguageName() {
  const t = useT()
  return (l: TermLanguage) => {
    if (l === 'de' || l === 'ar' || l === 'tr' || l === 'it' || l === 'fr') return t(`review.language.${l}`)
    return t(`terminology.language.${l.replace('-', '')}`)
  }
}

// Review is the review panel of the medical term list (docs/terminology/SPEC.md §3 "Review"): the queue of one
// language, needs changes first (every status by default), with the progress; a reviewer edits, approves or
// requests changes, keyboard first.
// Every staff member reads it; without the reviewer permission it has no controls.
export function Review({ api, reviewer }: { api: ApiClient; reviewer: boolean }) {
  const t = useT()
  const name = useLanguageName()
  const { language: uiLanguage } = useLanguage()
  const [language, setLanguage] = useState<TermLanguage>(uiLanguage)
  const [status, setStatus] = useState<Status | 'all'>('all')
  // The queue as read for the filters and reload count in key; anything else is still loading.
  const [read, setRead] = useState<{ key: string; queue: Queue }>()
  const [progress, setProgress] = useState<Progress[]>()
  const [progressReads, setProgressReads] = useState(0)
  const [open, setOpen] = useState<Open>()
  const [error, setError] = useState<string>()
  const [reload, setReload] = useState(0)
  const list = useRef<HTMLOListElement>(null)
  const focusTo = useRef<number>(undefined)
  const key = `${language}|${status}|${reload}`
  const queue: Queue = read?.key === key ? read.queue : { name: 'loading' }

  function ask(after?: string) {
    const query = { language, limit: pageSize, ...(status === 'all' ? {} : { status }), ...(after ? { after } : {}) }
    return attempt(api.GET('/terminology/review', { params: { query } }))
  }

  useEffect(() => {
    let current = true
    ask().then((res) => {
      if (current) setRead({ key, queue: res.ok ? { name: 'loaded', entries: res.data.entries, next: res.data.next } : { name: 'failed' } })
    })
    return () => {
      current = false
    }
    // ask reads the filters that make up key.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [api, key])

  async function more(after: string) {
    const res = await ask(after)
    setRead((r) => {
      if (r?.key !== key || r.queue.name !== 'loaded') return r
      if (!res.ok) return { key, queue: { name: 'failed' } }
      // An entry decided since the last page moved to a later status and can come again: it stays where it is.
      const shown = new Set(r.queue.entries.map((e) => e.term.id))
      const added = res.data.entries.filter((e) => !shown.has(e.term.id))
      return { key, queue: { name: 'loaded', entries: [...r.queue.entries, ...added], next: res.data.next } }
    })
  }

  useEffect(() => {
    let current = true
    attempt(api.GET('/terminology/review/progress')).then((res) => {
      if (current && res.ok) setProgress(res.data.languages)
    })
    return () => {
      current = false
    }
  }, [api, progressReads])

  // After an action the focus goes to the entry in focusTo, once it is rendered.
  useEffect(() => {
    if (focusTo.current === undefined) return
    list.current?.querySelectorAll<HTMLElement>('[data-entry]')[focusTo.current]?.focus()
    focusTo.current = undefined
  })

  function focusEntry(index: number) {
    focusTo.current = index
    list.current?.querySelectorAll<HTMLElement>('[data-entry]')[index]?.focus()
  }

  function filter(change: () => void) {
    setOpen(undefined)
    setError(undefined)
    change()
  }

  const entries = queue.name === 'loaded' ? queue.entries : []

  // done applies a write's answer to entry index, or says why it failed (a conflict reloads the queue).
  function done<T>(index: number, res: Outcome<T>, apply: (e: Entry, v: T) => Entry): boolean {
    if (!res.ok) {
      if (res.code === 'version_conflict') {
        setError(t('review.conflict'))
        setReload((n) => n + 1)
      } else if (res.code === 'forbidden') setError(t('review.readOnly'))
      else if (res.code === 'note_required') setError(t('review.noteRequired'))
      else if (res.code === 'invalid_request') setError(t('review.invalid'))
      else setError(t('common.somethingWrong'))
      return false
    }
    setError(undefined)
    setRead((r) =>
      r && r.queue.name === 'loaded' ? { ...r, queue: { ...r.queue, entries: r.queue.entries.map((e, i) => (i === index ? withStatus(apply(e, res.data)) : e)) } } : r,
    )
    setProgressReads((n) => n + 1)
    return true
  }

  function path(e: Entry) {
    return { id: e.term.id, language }
  }

  function ifMatch(version?: number) {
    return { 'If-Match': `W/"${version ?? 0}"` }
  }

  async function decide(index: number, part: Part, decision: { status: 'approved' | 'needs_changes'; note?: string }) {
    const e = entries[index]
    let ok: boolean
    if (part === 'labels') {
      if (!e.translation) return
      const res = await attempt(
        api.POST('/terminology/terms/{id}/translations/{language}/review', { params: { path: path(e), header: ifMatch(e.translation.versionId) }, body: decision }),
      )
      ok = done(index, res, (en, translation: Translation) => ({ ...en, translation }))
    } else {
      if (!e.explanation || language === 'la') return
      const res = await attempt(
        api.POST('/terminology/terms/{id}/explanations/{language}/review', {
          params: { path: { id: e.term.id, language }, header: ifMatch(e.explanation.versionId) },
          body: decision,
        }),
      )
      ok = done(index, res, (en, explanation: Explanation) => ({ ...en, explanation }))
    }
    if (!ok) return
    setOpen(undefined)
    focusTo.current = decision.status === 'approved' ? Math.min(index + 1, entries.length - 1) : index
  }

  async function saveLabels(index: number, preferred: string, synonyms: string[]) {
    const e = entries[index]
    const res = await attempt(
      api.PUT('/terminology/terms/{id}/translations/{language}', {
        params: { path: path(e), header: ifMatch(e.translation?.versionId) },
        body: { preferred, synonyms },
      }),
    )
    const ok = done(index, res, (en, translation: Translation) => ({ ...en, translation, term: { ...en.term, labels: withLabels(en, language, preferred, synonyms) } }))
    if (ok) {
      setOpen(undefined)
      focusTo.current = index
    }
  }

  async function saveExplanation(index: number, text: string) {
    if (language === 'la') return
    const e = entries[index]
    const res = await attempt(
      api.PUT('/terminology/terms/{id}/explanations/{language}', {
        params: { path: { id: e.term.id, language }, header: ifMatch(e.explanation?.versionId) },
        body: { text },
      }),
    )
    if (done(index, res, (en, explanation: Explanation) => ({ ...en, explanation }))) {
      setOpen(undefined)
      focusTo.current = index
    }
  }

  // onEntryKey: arrows or j and k move between entries; a reviewer's a, e and n approve, edit or request changes on
  // the labels, with Shift on the explanation. Nothing fires while typing in a field.
  function onEntryKey(index: number, event: KeyboardEvent<HTMLElement>) {
    const target = event.target as HTMLElement
    if (target.closest('input, textarea, select') || event.ctrlKey || event.metaKey || event.altKey) return
    const pressed = event.key.toLowerCase()
    const part: Part = event.shiftKey ? 'explanation' : 'labels'
    if (event.key === 'ArrowDown' || pressed === 'j') focusEntry(Math.min(index + 1, entries.length - 1))
    else if (event.key === 'ArrowUp' || pressed === 'k') focusEntry(Math.max(index - 1, 0))
    else if (!reviewer || (part === 'explanation' && language === 'la')) return
    else if (pressed === 'a') void decide(index, part, { status: 'approved' })
    else if (pressed === 'e') setOpen({ index, part, mode: 'edit' })
    else if (pressed === 'n' && (part === 'labels' ? entries[index].translation : entries[index].explanation)) setOpen({ index, part, mode: 'note' })
    else return
    event.preventDefault()
  }

  function cancel(index: number) {
    setOpen(undefined)
    setError(undefined)
    focusTo.current = index
  }

  return (
    <section className="review">
      <h1>{t('review.title')}</h1>
      <p className="hint">{t('review.intro')}</p>
      {!reviewer && <p className="notice">{t('review.readOnly')}</p>}
      <div className="review-filters">
        <div className="field">
          <label htmlFor="review-language">{t('review.languageFilter')}</label>
          <select id="review-language" value={language} onChange={(e) => filter(() => setLanguage(e.target.value as TermLanguage))}>
            {termLanguages.map((l) => (
              <option key={l} value={l}>
                {name(l)}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="review-status">{t('review.statusFilter')}</label>
          <select id="review-status" value={status} onChange={(e) => filter(() => setStatus(e.target.value as Status | 'all'))}>
            {statuses.map((s) => (
              <option key={s} value={s}>
                {t(`review.status.${s}`)}
              </option>
            ))}
          </select>
        </div>
      </div>
      {progress && <ProgressLine progress={progress} language={language} />}
      {reviewer && <Shortcuts />}
      <Alert message={error} />
      {queue.name === 'failed' && <Alert message={t('review.loadFailed')} />}
      {queue.name === 'loaded' && entries.length === 0 && <p className="panel">{t('review.empty')}</p>}
      <ol className="review-entries" ref={list}>
        {entries.map((e, i) => (
          <EntryCard
            key={e.term.id}
            entry={e}
            language={language}
            reviewer={reviewer}
            open={open?.index === i ? open : undefined}
            onKeyDown={(event) => onEntryKey(i, event)}
            onOpen={(part, mode) => setOpen({ index: i, part, mode })}
            onCancel={() => cancel(i)}
            onDecide={(part, decision) => decide(i, part, decision)}
            onSaveLabels={(preferred, synonyms) => saveLabels(i, preferred, synonyms)}
            onSaveExplanation={(text) => saveExplanation(i, text)}
            onNoteMissing={() => setError(t('review.noteRequired'))}
          />
        ))}
      </ol>
      {queue.name === 'loaded' && queue.next && (
        <div className="actions">
          <button type="button" onClick={() => more(queue.next!)}>
            {t('review.more')}
          </button>
        </div>
      )}
    </section>
  )
}

// withStatus sets an entry's status from its parts.
function withStatus(e: Entry): Entry {
  return { ...e, status: entryStatus(e) }
}

// withLabels is the term's labels with those of the queue's language replaced, first.
function withLabels(e: Entry, language: TermLanguage, preferred: string, synonyms: string[]) {
  return [{ language, preferred, synonyms }, ...e.term.labels.filter((l) => l.language !== language)]
}

function ProgressLine({ progress, language }: { progress: Progress[]; language: TermLanguage }) {
  const t = useT()
  const name = useLanguageName()
  const here = progress.find((p) => p.language === language)
  return (
    <div className="review-progress">
      {here && (
        <>
          <p>
            {t('review.progress', { approved: here.approved, terms: here.terms, language: name(language) })}
            {here.needsChanges > 0 && <span className="review-progress-changes">{t('review.progressNeedsChanges', { count: here.needsChanges })}</span>}
          </p>
          <progress value={here.approved} max={Math.max(here.terms, 1)} aria-label={t('review.progressBar', { language: name(language) })} />
        </>
      )}
      <details>
        <summary>{t('review.progressAll')}</summary>
        <table>
          <thead>
            <tr>
              <th scope="col">{t('review.languageFilter')}</th>
              <th scope="col">{t('review.status.approved')}</th>
              <th scope="col">{t('review.status.needs_changes')}</th>
              <th scope="col">{t('review.terms')}</th>
            </tr>
          </thead>
          <tbody>
            {progress.map((p) => (
              <tr key={p.language}>
                <td>{name(p.language)}</td>
                <td>{p.approved}</td>
                <td>{p.needsChanges}</td>
                <td>{p.terms}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </div>
  )
}

function Shortcuts() {
  const t = useT()
  return (
    <details className="review-shortcuts">
      <summary>{t('review.shortcuts')}</summary>
      <ul>
        <li>{t('review.shortcut.move')}</li>
        <li>{t('review.shortcut.labels')}</li>
        <li>{t('review.shortcut.explanation')}</li>
        <li>{t('review.shortcut.save')}</li>
      </ul>
    </details>
  )
}

type Decision = { status: 'approved' | 'needs_changes'; note?: string }

type EntryProps = {
  entry: Entry
  language: TermLanguage
  reviewer: boolean
  open?: Open
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void
  onOpen: (part: Part, mode: 'edit' | 'note') => void
  onCancel: () => void
  onDecide: (part: Part, decision: Decision) => void
  onSaveLabels: (preferred: string, synonyms: string[]) => void
  onSaveExplanation: (text: string) => void
  onNoteMissing: () => void
}

function EntryCard({ entry, language, reviewer, open, onKeyDown, onOpen, onCancel, onDecide, onSaveLabels, onSaveExplanation, onNoteMissing }: EntryProps) {
  const t = useT()
  const name = useLanguageName()
  const term = entry.term
  const labels = term.labels.find((l) => l.language === language)
  const context = term.labels.filter((l) => l.language !== language)
  const codes = (['icd10', 'ta98', 'atc', 'pcs'] as const).filter((c) => term.codes[c]).map((c) => t(`terminology.code.${c}`, { code: term.codes[c] ?? '' }))
  const headingID = `review-${term.id}`
  const category = t(`terminology.category.${term.category}`)
  const actions = (part: Part, exists: boolean, editLabel: string) =>
    reviewer &&
    open?.part !== part && (
      <div className="actions review-actions">
        {exists && (
          <button type="button" className="primary" aria-keyshortcuts={part === 'labels' ? 'A' : 'Shift+A'} onClick={() => onDecide(part, { status: 'approved' })}>
            {t('review.approve')}
          </button>
        )}
        <button type="button" aria-keyshortcuts={part === 'labels' ? 'E' : 'Shift+E'} onClick={() => onOpen(part, 'edit')}>
          {editLabel}
        </button>
        {exists && (
          <button type="button" aria-keyshortcuts={part === 'labels' ? 'N' : 'Shift+N'} onClick={() => onOpen(part, 'note')}>
            {t('review.requestChanges')}
          </button>
        )}
      </div>
    )
  const noteForm = (part: Part) =>
    open?.part === part && open.mode === 'note' && <NoteForm onSave={(note) => onDecide(part, { status: 'needs_changes', note })} onCancel={onCancel} onMissing={onNoteMissing} />
  return (
    <li>
      <article tabIndex={0} data-entry="" aria-labelledby={headingID} className={`review-entry ${entry.status}`} onKeyDown={onKeyDown}>
        <header className="review-entry-head">
          <h2 id={headingID}>{labels?.preferred ?? t('review.noLabel', { language: name(language) })}</h2>
          {codes.length > 0 ? <TipTag className="category" label={category} tip={codes.join(', ')} /> : <span className="tag">{category}</span>}
        </header>
        {context.length > 0 && (
          <p className="review-context">
            {context.map((l) => (
              <span key={l.language}>
                <span className="review-context-language">{name(l.language)}</span> {l.preferred}
              </span>
            ))}
          </p>
        )}
        <section className="review-part" aria-label={t('review.labels')}>
          <div className="review-part-head">
            <h3>{t('review.labels')}</h3>
            {entry.translation && <StatusTag status={entry.translation.review.status} />}
          </div>
          {open?.part === 'labels' && open.mode === 'edit' ? (
            <LabelsForm preferred={labels?.preferred ?? ''} synonyms={labels?.synonyms ?? []} onSave={onSaveLabels} onCancel={onCancel} />
          ) : (
            labels && (
              <p className="review-labels">
                <strong>{labels.preferred}</strong>
                {labels.synonyms.length > 0 && <span className="review-synonyms">{labels.synonyms.join(', ')}</span>}
              </p>
            )
          )}
          {entry.translation?.review.note && <p className="review-note">{t('review.note', { note: entry.translation.review.note })}</p>}
          {noteForm('labels')}
          {actions('labels', !!entry.translation, labels ? t('review.edit') : t('review.add'))}
        </section>
        {language !== 'la' && (
          <section className="review-part" aria-label={t('review.explanation')}>
            <div className="review-part-head">
              <h3>{t('review.explanation')}</h3>
              {entry.explanation && <StatusTag status={entry.explanation.review.status} />}
            </div>
            {open?.part === 'explanation' && open.mode === 'edit' ? (
              <ExplanationForm text={entry.explanation?.text ?? ''} onSave={onSaveExplanation} onCancel={onCancel} />
            ) : (
              <p className={entry.explanation ? 'review-explanation' : 'review-missing'}>{entry.explanation?.text ?? t('review.noExplanation')}</p>
            )}
            {entry.explanation?.review.note && <p className="review-note">{t('review.note', { note: entry.explanation.review.note })}</p>}
            {noteForm('explanation')}
            {actions('explanation', !!entry.explanation, entry.explanation ? t('review.edit') : t('review.write'))}
          </section>
        )}
      </article>
    </li>
  )
}

function StatusTag({ status }: { status: Status }) {
  const t = useT()
  return <span className={`tag review-status ${status}`}>{t(`review.status.${status}`)}</span>
}

// escapeCancels makes Escape in a form cancel it, without reaching the entry's shortcuts.
function escapeCancels(onCancel: () => void) {
  return (event: KeyboardEvent<HTMLFormElement>) => {
    if (event.key !== 'Escape') return
    event.preventDefault()
    event.stopPropagation()
    onCancel()
  }
}

function LabelsForm({ preferred, synonyms, onSave, onCancel }: { preferred: string; synonyms: string[]; onSave: (p: string, s: string[]) => void; onCancel: () => void }) {
  const t = useT()
  const [p, setP] = useState(preferred)
  const [s, setS] = useState(synonyms.join(' | '))
  function submit(event: FormEvent) {
    event.preventDefault()
    const value = p.trim()
    if (!value) return
    onSave(
      value,
      s
        .split('|')
        .map((x) => x.trim())
        .filter(Boolean),
    )
  }
  return (
    <form className="review-form" onSubmit={submit} onKeyDown={escapeCancels(onCancel)}>
      <div className="field">
        <label htmlFor="review-preferred">{t('review.preferred')}</label>
        <input id="review-preferred" autoFocus required maxLength={300} value={p} onChange={(e) => setP(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="review-synonyms">{t('review.synonyms')}</label>
        <input id="review-synonyms" value={s} onChange={(e) => setS(e.target.value)} />
      </div>
      <FormActions onCancel={onCancel} />
    </form>
  )
}

function ExplanationForm({ text, onSave, onCancel }: { text: string; onSave: (text: string) => void; onCancel: () => void }) {
  const t = useT()
  const [value, setValue] = useState(text)
  function save() {
    const trimmed = value.trim()
    if (trimmed) onSave(trimmed)
  }
  return (
    <form
      className="review-form"
      onSubmit={(e) => {
        e.preventDefault()
        save()
      }}
      onKeyDown={escapeCancels(onCancel)}
    >
      <div className="field">
        <label htmlFor="review-explanation">{t('review.explanationText')}</label>
        <textarea
          id="review-explanation"
          autoFocus
          required
          rows={3}
          maxLength={500}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              save()
            }
          }}
        />
        <p className="hint">{t('review.explanationHint')}</p>
      </div>
      <FormActions onCancel={onCancel} />
    </form>
  )
}

function NoteForm({ onSave, onCancel, onMissing }: { onSave: (note: string) => void; onCancel: () => void; onMissing: () => void }) {
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

function FormActions({ onCancel }: { onCancel: () => void }) {
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
