import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { stubReducedMotion } from '../test/matchMedia'
import { usePrefersReducedMotion } from './usePrefersReducedMotion'

afterEach(cleanup)

describe('usePrefersReducedMotion', () => {
  it('is false when the browser cannot tell', () => {
    const { result } = renderHook(() => usePrefersReducedMotion())

    expect(result.current).toBe(false)
  })

  it('follows the media query as it changes', () => {
    const media = stubReducedMotion(true)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(true)

    act(() => media.set(false))

    expect(result.current).toBe(false)
    media.restore()
  })
})
