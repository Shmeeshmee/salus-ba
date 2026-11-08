import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react'
import type { ApiClient } from '../api/client'
import { attempt } from '../api/result'
import type { components } from '../api/schema'
import { staffLanguages, type Language } from '../i18n/languages'
import { useLanguage, useT } from '../i18n/useT'
import { TipTag } from '../ui'
import './cheatsheet.css'

type Term = components['schemas']['Term']
type Names = components['schemas']['TermLabels']
// Shown is a language the cheatsheet shows: Latin and the staff languages.
type Shown = 'la' | Language

// The search waits this long after the last change before it asks (SPEC §3 "The cheatsheet").
const debounceMs = 150

type Answer = { name: 'none' } | { name: 'found'; terms: Term[] } | { name: 'failed' }
type Copied = { term: string; language: Shown }

// Cheatsheet is the header button and the palette it opens (also on Ctrl+K, ⌘K on macOS): a search of the medical
// terms that shows each term in Latin, the UI language and English, the other staff languages on demand, and
// copies a label with its button or, for the chosen term's UI-language label, with Enter.
export function Cheatsheet({ api }: { api: ApiClient }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const label = t('terminology.open', { shortcut: isMac() ? '⌘K' : 'Ctrl+K' })

  useEffect(() => {
    function onKeyDown(e: globalThis.KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  return (
    <>
      <button type="button" className="quiet cheatsheet-open" aria-label={label} title={label} onClick={() => setOpen(true)}>
        <SearchIcon />
      </button>
      {open && <Palette api={api} onClose={() => setOpen(false)} />}
    </>
  )
}

function isMac(): boolean {
  return /Mac|iPhone|iPad/.test(navigator.platform)
}

// columns is the languages shown first, Latin, the UI language and English, then the other staff languages.
function columns(ui: Language): { first: Shown[]; others: Shown[] } {
  const first = [...new Set<Shown>(['la', ui, 'en'])]
  return { first, others: staffLanguages.filter((l) => !first.includes(l)) }
}

function Palette({ api, onClose }: { api: ApiClient; onClose: () => void }) {
  const t = useT()
  const id = useId()
  const { language } = useLanguage()
  const [text, setText] = useState('')
  const [answer, setAnswer] = useState<Answer>({ name: 'none' })
  const [active, setActive] = useState(0)
  const [more, setMore] = useState(false)
  const [copied, setCopied] = useState<Copied>()
  const dialog = useRef<HTMLDivElement>(null)
  const input = useRef<HTMLInputElement>(null)
  const { first, others } = columns(language)

  useEffect(() => {
    const before = document.activeElement
    input.current?.focus()
    return () => {
      if (before instanceof HTMLElement && before.isConnected) before.focus()
    }
  }, [])

  // A change waits out the debounce and cancels the search before it, so only the last text's answer is shown.
  useEffect(() => {
    const q = text.trim()
    if (q === '') {
      setAnswer({ name: 'none' })
      return
    }
    const abort = new AbortController()
    const timer = setTimeout(async () => {
      const res = await attempt(
        api.GET('/terminology/search', {
          params: { query: { q, languages: [...first, ...others] } },
          querySerializer: { array: { style: 'form', explode: false } },
          signal: abort.signal,
        }),
      )
      if (abort.signal.aborted) return
      setAnswer(res.ok ? { name: 'found', terms: res.data.terms } : { name: 'failed' })
      setActive(0)
      setCopied(undefined)
    }, debounceMs)
    return () => {
      clearTimeout(timer)
      abort.abort()
    }
    // first and others follow the UI language, which does not change while the palette is open.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [api, text])

  const terms = answer.name === 'found' ? answer.terms : []
  const shown = more ? [...first, ...others] : first

  async function copy(term: Term, l: Shown) {
    const names = term.labels.find((n) => n.language === l)
    if (!names) return
    try {
      await navigator.clipboard.writeText(names.preferred)
      setCopied({ term: term.id, language: l })
    } catch {
      setCopied(undefined)
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
    } else if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && e.target === input.current) {
      e.preventDefault()
      setActive((a) => Math.max(0, Math.min(terms.length - 1, a + (e.key === 'ArrowDown' ? 1 : -1))))
    } else if (e.key === 'Enter' && e.target === input.current && terms[active]) {
      e.preventDefault()
      void copy(terms[active], language)
    } else if (e.key === 'Tab') trap(e, dialog.current)
  }

  const optionId = (i: number) => `${id}-term-${i}`
  return (
    <div className="cheatsheet-backdrop" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div ref={dialog} className="cheatsheet" role="dialog" aria-modal="true" aria-label={t('terminology.title')} onKeyDown={onKeyDown}>
        <div className="cheatsheet-search">
          <SearchIcon />
          <input
            ref={input}
            type="text"
            role="combobox"
            aria-expanded={terms.length > 0}
            aria-controls={`${id}-terms`}
            aria-autocomplete="list"
            aria-activedescendant={terms.length > 0 ? optionId(active) : undefined}
            aria-label={t('terminology.placeholder')}
            placeholder={t('terminology.placeholder')}
            maxLength={100}
            autoComplete="off"
            spellCheck={false}
            value={text}
            onChange={(e) => setText(e.target.value)}
          />
          <button type="button" className="quiet" onClick={onClose}>
            {t('terminology.close')}
          </button>
        </div>
        {terms.length > 0 && (
          <button type="button" className="quiet cheatsheet-more" aria-expanded={more} onClick={() => setMore((m) => !m)}>
            {t(more ? 'terminology.fewer' : 'terminology.more')}
          </button>
        )}
        {answer.name === 'failed' && (
          <p className="alert" role="alert">
            {t('terminology.failed')}
          </p>
        )}
        {answer.name === 'found' && terms.length === 0 && <p className="cheatsheet-empty muted">{t('terminology.none')}</p>}
        <ul id={`${id}-terms`} role="listbox" aria-label={t('terminology.title')} className="cheatsheet-terms">
          {terms.map((term, i) => (
            <li key={term.id} id={optionId(i)} role="option" aria-selected={i === active} className="term" onMouseEnter={() => setActive(i)}>
              <TermHead term={term} />
              <div className="term-labels">
                {shown.map((l) => (
                  <Cell key={l} language={l} names={term.labels.find((n) => n.language === l)} copied={copied?.term === term.id && copied.language === l} onCopy={() => copy(term, l)} />
                ))}
              </div>
            </li>
          ))}
        </ul>
        <p className="sr-only" role="status">
          {copied ? t('terminology.copied') : ''}
        </p>
        <p className="cheatsheet-source muted">{t('terminology.source')}</p>
      </div>
    </div>
  )
}

const codeKinds = ['icd10', 'ta98', 'atc', 'pcs'] as const

// TermHead is the term's category, with its codes behind a tooltip, and the mark of a term not reviewed yet.
function TermHead({ term }: { term: Term }) {
  const t = useT()
  const category = t(`terminology.category.${term.category}`)
  const codes = codeKinds.flatMap((k) => (term.codes[k] ? [t(`terminology.code.${k}`, { code: term.codes[k] })] : []))
  return (
    <div className="term-head">
      {codes.length > 0 ? <TipTag className="category" label={category} tip={codes.join(', ')} /> : <span className="tag category">{category}</span>}
      {term.reviewStatus === 'draft' && <span className="tag draft">{t('terminology.draft')}</span>}
    </div>
  )
}

// The catalog key of each shown language's name.
const languageKeys: Record<Shown, string> = { la: 'la', bs: 'bs', hr: 'hr', 'sr-Latn': 'srLatn', 'sr-Cyrl': 'srCyrl', en: 'en' }

type CellProps = { language: Shown; names?: Names; copied: boolean; onCopy: () => void }

// Cell is one language of a term: its preferred label with a copy button and its synonyms, or a note that the
// language has no label yet.
function Cell({ language, names, copied, onCopy }: CellProps) {
  const t = useT()
  const id = useId()
  const name = t(`terminology.language.${languageKeys[language]}`)
  return (
    <div className="term-label">
      <span className="term-language">{name}</span>
      {names ? (
        <>
          <span className="term-preferred" id={id} lang={language}>
            {names.preferred}
          </span>
          {names.synonyms.length > 0 && (
            <span className="term-synonyms" lang={language}>
              {names.synonyms.join(', ')}
            </span>
          )}
          <button type="button" className="quiet term-copy" aria-describedby={id} onClick={onCopy}>
            {t(copied ? 'terminology.copied' : 'terminology.copy')}
          </button>
        </>
      ) : (
        <span className="term-missing">{t('terminology.noLabel', { language: name })}</span>
      )}
    </div>
  )
}

// trap keeps Tab and Shift+Tab inside the palette: from the last control to the first, and back.
function trap(e: KeyboardEvent, root: HTMLElement | null) {
  const focusable = Array.from(root?.querySelectorAll<HTMLElement>('input, button, [tabindex="0"]') ?? [])
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

function SearchIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="8.75" cy="8.75" r="5.25" />
      <path d="m12.5 12.5 4 4" />
    </svg>
  )
}
