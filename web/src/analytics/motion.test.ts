import { describe, expect, it } from 'vitest'
import css from './analytics.css?raw'

describe('analytics.css', () => {
  it('moves nothing for a reader who asked for reduced motion: every transition is inside the no-preference query', () => {
    const query = '@media (prefers-reduced-motion: no-preference) {'
    const at = css.indexOf(query)
    expect(at).toBeGreaterThan(-1)
    const outside = css.slice(0, at) + css.slice(css.indexOf('\n}', at) + 2)
    expect(outside).not.toMatch(/transition|animation/)
    expect(css.slice(at)).toMatch(/transition/)
  })
})
