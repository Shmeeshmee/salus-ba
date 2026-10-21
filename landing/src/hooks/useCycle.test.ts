import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { stubReducedMotion } from '../test/matchMedia'
import { useCycle } from './useCycle'

function setHidden(hidden: boolean) {
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden })
}

const advance = (ms: number) => act(() => vi.advanceTimersByTime(ms))

beforeEach(() => vi.useFakeTimers())
afterEach(() => {
  cleanup()
  vi.useRealTimers()
  setHidden(false)
})

describe('useCycle', () => {
  it('advances on the interval and wraps around', () => {
    const { result } = renderHook(() => useCycle(3, { interval: 1000 }))
    expect(result.current.index).toBe(0)

    advance(1000)
    expect(result.current.index).toBe(1)
    advance(1000)
    expect(result.current.index).toBe(2)
    advance(1000)
    expect(result.current.index).toBe(0)
  })

  it('waits for the initial delay before the interval starts', () => {
    const { result } = renderHook(() => useCycle(3, { interval: 1000, initialDelay: 500 }))

    advance(1000)
    expect(result.current.index).toBe(0)
    advance(500)
    expect(result.current.index).toBe(1)
  })

  it('restarts the interval after a manual pick', () => {
    const { result } = renderHook(() => useCycle(3, { interval: 1000 }))
    advance(800)

    act(() => result.current.select(2))
    expect(result.current.index).toBe(2)
    advance(800)
    expect(result.current.index).toBe(2)
    advance(200)
    expect(result.current.index).toBe(0)
  })

  it('counts every change, picked or automatic, so a repeated pick still counts', () => {
    const { result } = renderHook(() => useCycle(3, { interval: 1000 }))
    expect(result.current.step).toBe(1)

    act(() => result.current.select(0))
    expect(result.current.step).toBe(2)
    advance(1000)
    expect(result.current.step).toBe(3)
  })

  it('pauses while the tab is hidden', () => {
    const { result } = renderHook(() => useCycle(3, { interval: 1000 }))

    setHidden(true)
    advance(3000)
    expect(result.current.index).toBe(0)

    setHidden(false)
    advance(1000)
    expect(result.current.index).toBe(1)
  })

  it('does not advance when motion is reduced, but a pick still works', () => {
    const media = stubReducedMotion(true)
    const { result } = renderHook(() => useCycle(3, { interval: 1000 }))

    advance(5000)
    expect(result.current.index).toBe(0)

    act(() => result.current.select(1))
    advance(5000)
    expect(result.current.index).toBe(1)
    media.restore()
  })
})
