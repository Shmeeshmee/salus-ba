import { describe, expect, it } from 'vitest'
import type { components } from '../api/schema'
import { entryStatus } from './entryStatus'

type Entry = components['schemas']['TermReviewEntry']
type Part = { review: { status: components['schemas']['ReviewStatus'] } }

// entry is a synthetic queue entry with labels whose review is translation, and an explanation when given.
function entry(translation?: Part, explanation?: Part): Entry {
  const at = '2026-11-10T09:00:00Z'
  return {
    status: 'unreviewed',
    term: { id: '00000000-0000-4000-8000-000000000001', category: 'anatomy', reviewStatus: 'unreviewed', codes: {}, labels: [] },
    translation: translation && { termId: '00000000-0000-4000-8000-000000000001', language: 'hr', versionId: 1, lastUpdated: at, ...translation },
    explanation: explanation && { termId: '00000000-0000-4000-8000-000000000001', language: 'hr', text: 'Placeholder', versionId: 1, lastUpdated: at, ...explanation },
  }
}
const s = (status: components['schemas']['ReviewStatus']): Part => ({ review: { status } })

describe('entryStatus', () => {
  it.each([
    ['needs changes when the labels need changes', entry(s('needs_changes'), s('approved')), 'needs_changes'],
    ['needs changes when the explanation does', entry(s('approved'), s('needs_changes')), 'needs_changes'],
    ['needs changes over unreviewed', entry(s('unreviewed'), s('needs_changes')), 'needs_changes'],
    ['unreviewed without labels', entry(), 'unreviewed'],
    ['unreviewed with an unreviewed explanation', entry(s('approved'), s('unreviewed')), 'unreviewed'],
    ['approved with approved labels and no explanation', entry(s('approved')), 'approved'],
    ['approved with both approved', entry(s('approved'), s('approved')), 'approved'],
  ])('is %s', (_, e, want) => {
    expect(entryStatus(e)).toBe(want)
  })
})
