import { type RefObject, useCallback, useEffect, useRef } from 'react'
import { usePrefersReducedMotion } from './usePrefersReducedMotion'

// Long enough for the smooth scroll to land, so the wheel events it leaves behind do not jump again.
const GLIDE_MS = 1100
// The hero still counts as in view until it has scrolled this far up.
const AT_HERO_TOP_PX = -40
const MIN_WHEEL_DELTA = 4
const MIN_SWIPE_PX = 40

// The first wheel or swipe down in the hero glides to the element with targetId. Returns the jump itself,
// for buttons that do the same.
export function useScrollJump(heroRef: RefObject<HTMLElement | null>, targetId: string): () => void {
  const reduced = usePrefersReducedMotion()
  const gliding = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const jump = useCallback(() => {
    if (gliding.current !== undefined) return
    document.getElementById(targetId)?.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'start' })
    gliding.current = setTimeout(() => {
      gliding.current = undefined
    }, GLIDE_MS)
  }, [targetId, reduced])

  useEffect(() => () => clearTimeout(gliding.current), [])

  useEffect(() => {
    const hero = heroRef.current
    if (!hero) return
    const atHero = () => hero.getBoundingClientRect().top > AT_HERO_TOP_PX
    let touchY: number | null = null

    const onWheel = (e: WheelEvent) => {
      if (e.deltaY > MIN_WHEEL_DELTA && atHero()) {
        e.preventDefault()
        jump()
      }
    }
    const onTouchStart = (e: TouchEvent) => {
      touchY = e.touches[0]?.clientY ?? null
    }
    const onTouchEnd = (e: TouchEvent) => {
      const end = e.changedTouches[0]
      if (touchY === null || !end) return
      const travelled = touchY - end.clientY
      touchY = null
      if (travelled > MIN_SWIPE_PX && atHero()) jump()
    }

    hero.addEventListener('wheel', onWheel, { passive: false })
    hero.addEventListener('touchstart', onTouchStart, { passive: true })
    hero.addEventListener('touchend', onTouchEnd, { passive: true })
    return () => {
      hero.removeEventListener('wheel', onWheel)
      hero.removeEventListener('touchstart', onTouchStart)
      hero.removeEventListener('touchend', onTouchEnd)
    }
  }, [heroRef, jump])

  return jump
}
