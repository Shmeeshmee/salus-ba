import { describe, expect, it } from 'vitest'
import { cx } from './cx'

describe('cx', () => {
  it('joins the class names that are set', () => {
    expect(cx('a', false, undefined, 'b')).toBe('a b')
  })

  it('returns an empty string when none is set', () => {
    expect(cx(false, undefined)).toBe('')
  })
})
