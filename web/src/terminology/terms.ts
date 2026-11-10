import { createContext, useContext } from 'react'
import type { components } from '../api/schema'

// terms.ts holds what Explain.tsx's components share (SAL-13): the explanations' context and the cutting of a text
// at its spans.

export type Explanation = components['schemas']['ExplainedTerm']
export type Span = components['schemas']['TermSpan']

export type Explain = (termId: string) => Explanation | undefined

export const explainContext = createContext<Explain>(() => undefined)

// useExplain gives a term's explanation in the UI language (else English), undefined while there is none.
export function useExplain(): Explain {
  return useContext(explainContext)
}

// explained reports whether any span of spans has an explanation.
export function explained(explain: Explain, spans?: Span[]): boolean {
  return (spans ?? []).some((s) => explain(s.term_id) !== undefined)
}

export type Piece = { text: string; termId?: string }

// pieces cuts text at its spans (UTF-16 offsets, as JavaScript indexes strings): the parts between them, and each
// span's words with its term. A span outside the text or overlapping the one before is left out; joined, the pieces
// are the text.
export function pieces(text: string, spans: Span[] = []): Piece[] {
  const out: Piece[] = []
  let at = 0
  for (const s of [...spans].sort((a, b) => a.start - b.start)) {
    if (s.start < at || s.end > text.length || s.end <= s.start) continue
    if (s.start > at) out.push({ text: text.slice(at, s.start) })
    out.push({ text: text.slice(s.start, s.end), termId: s.term_id })
    at = s.end
  }
  if (at < text.length || out.length === 0) out.push({ text: text.slice(at) })
  return out
}

