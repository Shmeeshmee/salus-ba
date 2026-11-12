import { useEffect, useRef, useState, type FormEvent } from 'react'
import type { ApiClient } from '../api/client'
import type { components } from '../api/schema'
import { attempt } from '../api/result'
import { useLanguage, useT } from '../i18n/useT'
import { Alert } from '../ui'
import { escapeCancels, useLanguageName } from './reviewHelpers'
import { FormActions, NoteForm, StatusTag } from './reviewParts'

type GroupName = components['schemas']['DiagnosisGroupName']
type Language = components['schemas']['Language']

// The app languages a group is named in (api/openapi.yaml Language).
const languages: Language[] = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr']

type List = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; groups: GroupName[] }
type Open = { code: string; mode: 'edit' | 'note' }

// GroupNames is the review of the diagnosis groups' names (docs/plans/analytics.md §3): every group of one
// language, a subgroup with its group, its status and note. A reviewer approves a name, edits it (which sets it
// back to unreviewed) or requests changes with a note. Every staff member reads it.
export function GroupNames({ api, reviewer }: { api: ApiClient; reviewer: boolean }) {
  const t = useT()
  const name = useLanguageName()
  const { language: ui } = useLanguage()
  const [language, setLanguage] = useState<Language>(languages.includes(ui as Language) ? (ui as Language) : 'en')
  const [list, setList] = useState<List>({ name: 'loading' })
  const [open, setOpen] = useState<Open>()
  const [error, setError] = useState<string>()
  const [reload, setReload] = useState(0)
  // shown is the language on screen: an answer to a write in another one is dropped.
  const shown = useRef(language)

  useEffect(() => {
    let live = true
    attempt(api.GET('/terminology/review/diagnosis-groups', { params: { query: { language } } })).then((res) => {
      if (live) setList(res.ok ? { name: 'loaded', groups: res.data.groups } : { name: 'failed' })
    })
    return () => {
      live = false
    }
  }, [api, language, reload])

  const groups = list.name === 'loaded' ? list.groups : []
  const text = new Map(groups.map((g) => [g.code, g.text]))

  async function write(g: GroupName, request: Promise<{ data?: GroupName; error?: unknown; response: Response }>) {
    const asked = shown.current
    const res = await attempt(request)
    if (shown.current !== asked) return
    if (!res.ok) {
      if (res.code === 'version_conflict') {
        setError(t('review.conflict'))
        setReload((n) => n + 1)
      } else if (res.code === 'forbidden') setError(t('review.readOnly'))
      else if (res.code === 'note_required') setError(t('review.noteRequired'))
      else if (res.code === 'invalid_request') setError(t('review.groups.invalid'))
      else setError(t('common.somethingWrong'))
      return
    }
    setError(undefined)
    setOpen(undefined)
    setList((l) => (l.name === 'loaded' ? { ...l, groups: l.groups.map((x) => (x.code === g.code ? { ...res.data, parent: g.parent } : x)) } : l))
  }

  const params = (g: GroupName) => ({ path: { code: g.code, language }, header: { 'If-Match': `W/"${g.versionId}"` } })
  const decide = (g: GroupName, decision: { status: 'approved' | 'needs_changes'; note?: string }) =>
    write(g, api.POST('/terminology/diagnosis-groups/{code}/names/{language}/review', { params: params(g), body: decision }))
  const rename = (g: GroupName, value: string) =>
    write(g, api.PUT('/terminology/diagnosis-groups/{code}/names/{language}', { params: params(g), body: { text: value } }))

  return (
    <>
      <p className="hint">{t('review.groups.intro')}</p>
      {!reviewer && <p className="notice">{t('review.readOnly')}</p>}
      <div className="review-filters">
        <div className="field">
          <label htmlFor="group-language">{t('review.languageFilter')}</label>
          <select
            id="group-language"
            value={language}
            onChange={(e) => {
              shown.current = e.target.value as Language
              setOpen(undefined)
              setError(undefined)
              setLanguage(shown.current)
            }}
          >
            {languages.map((l) => (
              <option key={l} value={l}>
                {name(l)}
              </option>
            ))}
          </select>
        </div>
      </div>
      <Alert message={error ?? (list.name === 'failed' ? t('review.groups.loadFailed') : undefined)} />
      <ol className="review-entries">
        {groups.map((g) => {
          const heading = `group-${g.code}`
          const isOpen = open?.code === g.code
          return (
            <li key={g.code}>
              <article aria-labelledby={heading} className={`review-entry ${g.review.status}`}>
                <header className="review-entry-head">
                  <h2 id={heading}>{g.text}</h2>
                  <StatusTag status={g.review.status} />
                </header>
                {g.parent && <p className="review-context">{t('review.groups.subgroupOf', { group: text.get(g.parent) ?? g.parent })}</p>}
                {g.review.note && <p className="review-note">{t('review.note', { note: g.review.note })}</p>}
                {isOpen && open.mode === 'edit' && <NameForm text={g.text} onSave={(v) => void rename(g, v)} onCancel={() => setOpen(undefined)} />}
                {isOpen && open.mode === 'note' && (
                  <NoteForm onSave={(note) => void decide(g, { status: 'needs_changes', note })} onCancel={() => setOpen(undefined)} onMissing={() => setError(t('review.noteRequired'))} />
                )}
                {reviewer && !isOpen && (
                  <div className="actions review-actions">
                    <button type="button" className="primary" onClick={() => void decide(g, { status: 'approved' })}>
                      {t('review.approve')}
                    </button>
                    <button type="button" onClick={() => setOpen({ code: g.code, mode: 'edit' })}>
                      {t('review.edit')}
                    </button>
                    <button type="button" onClick={() => setOpen({ code: g.code, mode: 'note' })}>
                      {t('review.requestChanges')}
                    </button>
                  </div>
                )}
              </article>
            </li>
          )
        })}
      </ol>
    </>
  )
}

function NameForm({ text, onSave, onCancel }: { text: string; onSave: (text: string) => void; onCancel: () => void }) {
  const t = useT()
  const [value, setValue] = useState(text)
  function submit(event: FormEvent) {
    event.preventDefault()
    const trimmed = value.trim()
    if (trimmed) onSave(trimmed)
  }
  return (
    <form className="review-form" onSubmit={submit} onKeyDown={escapeCancels(onCancel)}>
      <div className="field">
        <label htmlFor="group-name">{t('review.groups.nameLabel')}</label>
        <input id="group-name" autoFocus required maxLength={100} value={value} onChange={(e) => setValue(e.target.value)} />
      </div>
      <FormActions onCancel={onCancel} />
    </form>
  )
}
