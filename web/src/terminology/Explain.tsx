import { useContext, useEffect, useId, useState, type ReactNode } from 'react'
import type { ApiClient } from '../api/client'
import { attempt } from '../api/result'
import { useLanguage, useT } from '../i18n/useT'
import { explainContext, pieces, type Explanation, type Span } from './terms'
import './explain.css'

// Explain.tsx is the plain-language explanation of the medical terms a transcript line or a document marks (SAL-13,
// docs/terminology/SPEC.md §3 "Explanations"): the server tags the text (TermSpan) where it is read, and the page
// looks the terms up in the explanations of the UI language, English where there is none. An overlay: the text is
// shown as it is, a tagged term only gets a dotted underline and a tooltip. The explanations are reference data;
// the texts never leave the page for them.

// ExplanationsProvider reads the explanations in the UI language once, and again when it changes, for everything
// inside it. Until they are read, or when they cannot be, the terms are shown as plain text.
export function ExplanationsProvider({ api, children }: { api: ApiClient; children: ReactNode }) {
  const { language } = useLanguage()
  const [byTerm, setByTerm] = useState<Map<string, Explanation>>(new Map())
  useEffect(() => {
    let open = true
    void attempt(api.GET('/terminology/explanations', { params: { query: { language } } })).then((res) => {
      if (open && res.ok) setByTerm(new Map(res.data.explanations.map((e) => [e.termId, e])))
    })
    return () => {
      open = false
    }
  }, [api, language])
  return <explainContext.Provider value={(id) => byTerm.get(id)}>{children}</explainContext.Provider>
}

// TermText is text with its explained terms marked, each explained in a tooltip on hover and on focus; with
// markOnly, marked without a tooltip of its own (where the text's tooltip lists them, TermList). A term without an
// explanation in the UI language or English stays plain.
export function TermText({ text, spans, markOnly }: { text: string; spans?: Span[]; markOnly?: boolean }) {
  const explain = useContext(explainContext)
  return (
    <>
      {pieces(text, spans).map((p, i) => {
        const e = p.termId ? explain(p.termId) : undefined
        if (!e) return p.text
        return markOnly ? (
          <span key={i} className="term-mark">
            {p.text}
          </span>
        ) : (
          <Term key={i} text={p.text} explanation={e} />
        )
      })}
    </>
  )
}

function Term({ text, explanation }: { text: string; explanation: Explanation }) {
  const id = useId()
  return (
    <span className="term" tabIndex={0} aria-describedby={id}>
      {text}
      <span role="tooltip" id={id}>
        <ExplanationText explanation={explanation} />
      </span>
    </span>
  )
}

// ExplanationText is an explanation, in its own language when it is the English fallback, then the label that it
// is informational, never a diagnosis, and "Not reviewed yet" until a reviewer approved it.
function ExplanationText({ explanation: e }: { explanation: Explanation }) {
  const t = useT()
  const { language } = useLanguage()
  return (
    <>
      <span lang={e.language === language ? undefined : e.language}>{e.text}</span>
      <span className="term-note">
        {t('terminology.explainNote')}
        {e.reviewStatus !== 'approved' && ` · ${t('terminology.draft')}`}
      </span>
    </>
  )
}

// TermList is the explained terms of text as a list, each term's words and its explanation, for a tooltip that
// already holds the text (the captions'); nothing when no term is explained.
export function TermList({ text, spans }: { text: string; spans?: Span[] }) {
  const t = useT()
  const { language } = useLanguage()
  const explain = useContext(explainContext)
  const found = pieces(text, spans).flatMap((p) => {
    const e = p.termId ? explain(p.termId) : undefined
    return e ? [{ words: p.text, e }] : []
  })
  if (found.length === 0) return null
  return (
    <span className="term-list">
      {found.map(({ words, e }, i) => (
        <span key={i} className="term-item">
          <span className="term-words">{words}</span> <span lang={e.language === language ? undefined : e.language}>{e.text}</span>
        </span>
      ))}
      <span className="term-note">
        {t('terminology.explainNote')}
        {found.some(({ e }) => e.reviewStatus !== 'approved') && ` · ${t('terminology.draft')}`}
      </span>
    </span>
  )
}

// TermTextLines is a multi-line text with each line's spans (lines[i] for its i-th line), as a document's field
// gives them; lines without spans stay plain.
export function TermTextLines({ text, lines }: { text: string; lines?: Span[][] }) {
  return (
    <>
      {text.split('\n').map((line, i) => (
        <span key={i}>
          {i > 0 && '\n'}
          <TermText text={line} spans={lines?.[i]} />
        </span>
      ))}
    </>
  )
}
