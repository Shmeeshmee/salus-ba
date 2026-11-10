import type { components } from '../api/schema'

type Entry = components['schemas']['TermReviewEntry']
type Status = components['schemas']['ReviewStatus']

// entryStatus is an entry's status by the server's rule (docs/terminology/SPEC.md §3 "Review"): needs changes when
// its labels or its explanation do, else unreviewed when it has no labels yet or either is unreviewed, else
// approved. An explanation that does not exist does not count.
export function entryStatus(e: Entry): Status {
  const parts = [e.translation?.review.status, e.explanation?.review.status]
  if (parts.includes('needs_changes')) return 'needs_changes'
  if (!e.translation || parts.includes('unreviewed')) return 'unreviewed'
  return 'approved'
}
