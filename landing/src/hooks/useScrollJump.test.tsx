import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { stubReducedMotion } from '../test/matchMedia'
import { useScrollJump } from './useScrollJump'

function Page() {
  const hero = useRef<HTMLElement>(null)
  const jump = useScrollJump(hero, 'page')
  return (
    <>
      <section ref={hero} aria-label="Hero">
        <button type="button" onClick={jump}>
          Jump
        </button>
      </section>
      <main id="page" />
    </>
  )
}

// jsdom has no touch support worth the name; the hook only reads clientY.
function swipe(element: Element, fromY: number, toY: number) {
  const start = new Event('touchstart')
  Object.defineProperty(start, 'touches', { value: [{ clientY: fromY }] })
  const end = new Event('touchend')
  Object.defineProperty(end, 'changedTouches', { value: [{ clientY: toY }] })
  fireEvent(element, start)
  fireEvent(element, end)
}

const scrollIntoView = vi.fn()

beforeEach(() => {
  vi.useFakeTimers()
  scrollIntoView.mockClear()
  Element.prototype.scrollIntoView = scrollIntoView
})
afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

const hero = () => screen.getByRole('region', { name: 'Hero' })
const page = () => screen.getByRole('main')

describe('useScrollJump', () => {
  it('glides to the page once on a downward wheel in the hero', () => {
    render(<Page />)

    const allowed = fireEvent.wheel(hero(), { deltaY: 120 })
    fireEvent.wheel(hero(), { deltaY: 120 })

    expect(allowed).toBe(false)
    expect(scrollIntoView).toHaveBeenCalledTimes(1)
    expect(scrollIntoView.mock.contexts[0]).toBe(page())
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })
  })

  it('can glide again once the first glide is over', () => {
    render(<Page />)
    fireEvent.wheel(hero(), { deltaY: 120 })

    act(() => vi.advanceTimersByTime(1100))
    fireEvent.wheel(hero(), { deltaY: 120 })

    expect(scrollIntoView).toHaveBeenCalledTimes(2)
  })

  it('ignores an upward wheel', () => {
    render(<Page />)

    expect(fireEvent.wheel(hero(), { deltaY: -120 })).toBe(true)
    expect(scrollIntoView).not.toHaveBeenCalled()
  })

  it('glides to the page on a swipe up, but not on a short one', () => {
    render(<Page />)

    swipe(hero(), 500, 480)
    expect(scrollIntoView).not.toHaveBeenCalled()
    swipe(hero(), 500, 300)
    expect(scrollIntoView).toHaveBeenCalledTimes(1)
  })

  it('leaves the wheel alone when the page is already past the hero', () => {
    render(<Page />)
    hero().getBoundingClientRect = () => ({ top: -500 }) as DOMRect

    expect(fireEvent.wheel(hero(), { deltaY: 120 })).toBe(true)
    swipe(hero(), 500, 300)
    expect(scrollIntoView).not.toHaveBeenCalled()
  })

  it('jumps without smooth scrolling when motion is reduced', () => {
    const media = stubReducedMotion(true)
    render(<Page />)

    fireEvent.click(screen.getByRole('button', { name: 'Jump' }))

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto', block: 'start' })
    media.restore()
  })
})
