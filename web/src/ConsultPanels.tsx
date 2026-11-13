import { useEffect, useId, useLayoutEffect, useRef, useState, type Dispatch, type KeyboardEvent, type ReactNode } from 'react'
import type { ConsultActions, ConsultMessage, ConsultState, End, SuggestionEvent, TranscriptEvent } from './consult'
import { AllergyWarning } from './AllergiesPanel'
import { Documents } from './Documents'
import type { PatientEntries } from './entries'
import { useFormat, useT } from './i18n/useT'
import { TermList, TermText } from './terminology/Explain'
import { explained, useExplain } from './terminology/terms'
import { Alert } from './ui'

// The call page's live panels (SAL-9): the captions on the patient's video, and beside the video the transcript,
// the suggestions and the documents, one tab at a time. Nothing here covers the video but the captions, which
// keep to a band along its lower edge.

// A final caption goes after this long without a new one, so the band does not hold a sentence long said.
const captionQuiet = 6000

// Captions is the patient's latest words in the practitioner's language, revised in place while they are
// recognised. The original is in a tooltip, shown on hover and on focus and read as the caption's description.
export function Captions({ caption }: { caption?: TranscriptEvent }) {
  const t = useT()
  const id = useId()
  const explain = useExplain()
  const [quiet, setQuiet] = useState<string>()
  const key = caption && `${caption.segment_id}|${caption.is_final}|${caption.translation}`
  useEffect(() => {
    if (!caption?.is_final) return
    const timer = setTimeout(() => setQuiet(key), captionQuiet)
    return () => clearTimeout(timer)
  }, [caption?.is_final, key])
  if (!caption || quiet === key) return null
  const translated = caption.translation !== ''
  const text = translated ? caption.translation : caption.original
  // The caption's explained terms (SAL-13) are marked in it and explained in its tooltip: the caption's text is
  // clipped to two lines, so a tooltip of each term's own would be cut off.
  const spans = translated ? caption.terms?.translation : caption.terms?.original
  const tip = translated || explained(explain, spans)
  return (
    <div className="captions" role="group" aria-label={t('consultation.live.captions')}>
      <p className="caption tip" tabIndex={0} aria-describedby={tip ? id : undefined}>
        <span className="caption-text" lang={translated ? undefined : caption.lang}>
          <TermText text={text} spans={spans} markOnly />
        </span>
        {tip && (
          <span role="tooltip" id={id}>
            {translated && (
              <span>
                {t('consultation.live.original')} <span lang={caption.lang}>{caption.original}</span>
              </span>
            )}
            <TermList text={text} spans={spans} />
          </span>
        )}
      </p>
    </div>
  )
}

type Tab = 'transcript' | 'suggestions' | 'documents' | 'entries' | 'history'

type SideProps = {
  appointmentId: string
  state: ConsultState
  dispatch: Dispatch<ConsultMessage>
  actions: ConsultActions
  zone: string
  end?: End
  // entries are the patient's allergies, warned of above the drafting editor (shown, never sent: ADR 0014 "No AI"),
  // and the panel of their allergies and medications, a tab of its own (SAL-41). Without them neither is shown.
  entries?: { state: PatientEntries; panel: ReactNode }
  // history is the patient's history (SAL-40), a tab of its own when given; it is mounted, and so read and
  // audited, only once the practitioner opens it.
  history?: ReactNode
}

// ConsultSide is the panel beside the video. A suggestion's or a field's sources open the transcript at
// those lines, highlighted. Every tab stays mounted, so the transcript keeps its scroll while hidden.
export function ConsultSide({ appointmentId, state, dispatch, actions, zone, end, entries, history }: SideProps) {
  const t = useT()
  const { number } = useFormat()
  const id = useId()
  const [tab, setTab] = useState<Tab>('transcript')
  const [cited, setCited] = useState<{ ids: string[]; at: number }>({ ids: [], at: 0 })
  const active = state.suggestions.filter((s) => s.status !== 'dismissed').length
  const tabs: Tab[] = ['transcript', 'suggestions', 'documents', ...(entries ? (['entries'] as const) : []), ...(history ? (['history'] as const) : [])]
  const label = {
    transcript: 'consultation.live.tabTranscript',
    suggestions: 'consultation.live.tabSuggestions',
    documents: 'consultation.live.tabDocuments',
    entries: 'allergies.tab',
    history: 'history.tab',
  } as const
  const [historyOpened, setHistoryOpened] = useState(false)
  if (tab === 'history' && !historyOpened) setHistoryOpened(true)

  function show(ids: string[]) {
    setTab('transcript')
    setCited((c) => ({ ids, at: c.at + 1 }))
  }

  function key(e: KeyboardEvent) {
    const step = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
    if (!step) return
    e.preventDefault()
    const next = tabs[(tabs.indexOf(tab) + step + tabs.length) % tabs.length]
    setTab(next)
    document.getElementById(`${id}-${next}-tab`)?.focus()
  }

  if (end) {
    return (
      <aside className="call-side" aria-label={t('consultation.live.panels')}>
        <p className="muted side-ended" role="status">
          {t(end === 'consent_required' ? 'consultation.peerWithdrewConsent' : 'consultation.notAvailable')}
        </p>
      </aside>
    )
  }

  return (
    <aside className="call-side" aria-label={t('consultation.live.panels')}>
      <div className="tabs" role="tablist" onKeyDown={key}>
        {tabs.map((name) => (
          <button
            key={name}
            type="button"
            role="tab"
            id={`${id}-${name}-tab`}
            aria-selected={tab === name}
            aria-controls={`${id}-${name}`}
            tabIndex={tab === name ? 0 : -1}
            onClick={() => setTab(name)}
          >
            {t(label[name])}
            {name === 'suggestions' && active > 0 && (
              <>
                {' '}
                <span className="count">{number(active)}</span>
              </>
            )}
          </button>
        ))}
      </div>
      <div className="side-panel transcript-panel" role="tabpanel" id={`${id}-transcript`} aria-labelledby={`${id}-transcript-tab`} hidden={tab !== 'transcript'}>
        <Transcript segments={state.segments} cited={cited} zone={zone} />
      </div>
      <div className="side-panel" role="tabpanel" id={`${id}-suggestions`} aria-labelledby={`${id}-suggestions-tab`} hidden={tab !== 'suggestions'}>
        <Suggestions suggestions={state.suggestions} dispatch={dispatch} actions={actions} onSources={show} />
      </div>
      <div className="side-panel" role="tabpanel" id={`${id}-documents`} aria-labelledby={`${id}-documents-tab`} hidden={tab !== 'documents'}>
        {entries && <AllergyWarning entries={entries.state} />}
        <Documents appointmentId={appointmentId} drafts={state.drafts} dispatch={dispatch} actions={actions} onSources={show} />
      </div>
      {entries && (
        <div className="side-panel" role="tabpanel" id={`${id}-entries`} aria-labelledby={`${id}-entries-tab`} hidden={tab !== 'entries'}>
          {entries.panel}
        </div>
      )}
      {history && (
        <div className="side-panel" role="tabpanel" id={`${id}-history`} aria-labelledby={`${id}-history-tab`} hidden={tab !== 'history'}>
          {historyOpened && history}
        </div>
      )}
    </aside>
  )
}

// A practitioner within this many pixels of the bottom is following the transcript.
const bottom = 24

// Transcript is every line so far, with its speaker and time, the text in the practitioner's language first.
// It follows the latest line unless the practitioner has scrolled up; then it offers to show the latest.
function Transcript({ segments, cited, zone }: { segments: TranscriptEvent[]; cited: { ids: string[]; at: number }; zone: string }) {
  const t = useT()
  const { time } = useFormat()
  const list = useRef<HTMLOListElement>(null)
  const following = useRef(true)
  const [behind, setBehind] = useState(false)

  useLayoutEffect(() => {
    const el = list.current
    if (!el || segments.length === 0) return
    if (following.current) el.scrollTop = el.scrollHeight
    else setBehind(true)
  }, [segments])

  useEffect(() => {
    if (cited.at === 0) return
    following.current = false
    const first = list.current?.querySelector('[aria-current="true"]')
    first?.scrollIntoView({ block: 'center', behavior: reducedMotion() ? 'auto' : 'smooth' })
  }, [cited])

  function scrolled() {
    const el = list.current!
    following.current = el.scrollHeight - el.scrollTop - el.clientHeight <= bottom
    if (following.current) setBehind(false)
  }

  function latest() {
    const el = list.current!
    el.scrollTop = el.scrollHeight
    following.current = true
    setBehind(false)
  }

  return (
    <div className="transcript">
      {segments.length === 0 && <p className="muted empty">{t('consultation.live.transcriptEmpty')}</p>}
      <ol ref={list} className="transcript-lines" aria-label={t('consultation.live.tabTranscript')} onScroll={scrolled}>
        {segments.map((s) => {
          const flipped = s.speaker === 'patient' && s.translation !== ''
          const [first, second] = flipped ? [s.translation, s.original] : [s.original, s.translation]
          const [firstTerms, secondTerms] = flipped ? [s.terms?.translation, s.terms?.original] : [s.terms?.original, s.terms?.translation]
          const firstLang = first === s.original ? s.lang : undefined
          return (
            <li key={s.segment_id} className={s.is_final ? undefined : 'partial'} aria-current={cited.ids.includes(s.segment_id) || undefined}>
              <p className="line-meta">
                <span className="speaker">{t(`consultation.live.speaker.${s.speaker}`)}</span>
                <time dateTime={s.ts_start}>{time(new Date(s.ts_start), zone)}</time>
                {!s.is_final && <span className="sr-only">{t('consultation.live.inProgress')}</span>}
              </p>
              <p lang={firstLang}>
                <TermText text={first} spans={firstTerms} />
              </p>
              {second && (
                <p className="muted" lang={second === s.original ? s.lang : undefined}>
                  <TermText text={second} spans={secondTerms} />
                </p>
              )}
            </li>
          )
        })}
      </ol>
      {behind && (
        <button type="button" className="latest" onClick={latest}>
          {t('consultation.live.latest')}
        </button>
      )}
    </div>
  )
}

function reducedMotion(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

type SuggestionsProps = {
  suggestions: SuggestionEvent[]
  dispatch: Dispatch<ConsultMessage>
  actions: ConsultActions
  onSources: (ids: string[]) => void
}

// Suggestions lists what is not dismissed, pinned first, then the newest first. Each is labelled as a
// suggestion of its kind; a pin or a dismissal shows at once and is put back if it cannot be saved.
function Suggestions({ suggestions, dispatch, actions, onSources }: SuggestionsProps) {
  const t = useT()
  const [error, setError] = useState<string>()
  const shown = suggestions
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => s.status !== 'dismissed')
    .sort((a, b) => Number(b.s.status === 'pinned') - Number(a.s.status === 'pinned') || b.i - a.i)
    .map(({ s }) => s)

  async function change(s: SuggestionEvent, status: SuggestionEvent['status']) {
    setError(undefined)
    dispatch({ type: 'suggestion', version: 1, payload: { ...s, status } })
    try {
      await actions.setSuggestionStatus(s.id, status)
    } catch {
      dispatch({ type: 'suggestion', version: 1, payload: s })
      setError(t('consultation.live.statusFailed'))
    }
  }

  return (
    <div className="suggestions">
      <p className="note">{t('consultation.live.suggestionsNote')}</p>
      <Alert message={error} />
      {shown.length === 0 && <p className="muted">{t('consultation.live.noSuggestions')}</p>}
      {shown.map((s) => (
        <SuggestionCard key={s.id} s={s} onChange={(status) => void change(s, status)} onSources={() => onSources(s.source_segment_ids)} />
      ))}
    </div>
  )
}

function SuggestionCard({ s, onChange, onSources }: { s: SuggestionEvent; onChange: (status: SuggestionEvent['status']) => void; onSources: () => void }) {
  const t = useT()
  const id = useId()
  const pinned = s.status === 'pinned'
  return (
    <article className={`suggestion ${s.type}${pinned ? ' pinned' : ''}`} aria-labelledby={id}>
      <p className="kind" id={id}>
        {t(`consultation.live.kind.${s.type}`)}
      </p>
      <p className="suggestion-text">{s.text}</p>
      <div className="suggestion-actions">
        {s.source_segment_ids.length > 0 && (
          <button type="button" className="quiet" onClick={onSources}>
            {t('consultation.live.sources', { count: s.source_segment_ids.length })}
          </button>
        )}
        <button type="button" className="quiet" onClick={() => onChange(pinned ? 'shown' : 'pinned')}>
          {t(pinned ? 'consultation.live.unpin' : 'consultation.live.pin')}
        </button>
        <button type="button" className="quiet" onClick={() => onChange('dismissed')}>
          {t('consultation.live.dismiss')}
        </button>
      </div>
    </article>
  )
}
